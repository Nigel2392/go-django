package revisions

import (
	"context"
	"database/sql"
	"encoding/json"
	"reflect"
	"slices"
	"time"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/pkg/errors"
)

var (
	_ queries.ActsBeforeCreate = (*Revision)(nil)
	_ queries.OrderByDefiner   = (*Revision)(nil)
	_ attrs.Definer            = (*Revision)(nil)
)

type Revision struct {
	ID          int64     `json:"id"`
	ObjectID    string    `json:"object_id"`
	ContentType string    `json:"content_type"`
	Data        string    `json:"data"`
	CreatedAt   time.Time `json:"created_at"`
}

func (r *Revision) BeforeCreate(context.Context) error {
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now()
	}
	return nil
}

func (r *Revision) OrderBy() []string {
	return []string{"-CreatedAt"}
}

func (r *Revision) FieldDefs() attrs.Definitions {
	return attrs.Define(r,
		attrs.Unbound("ID", &attrs.FieldConfig{
			Primary: true,
			Column:  "id",
		}),
		attrs.Unbound("ObjectID", &attrs.FieldConfig{
			Column: "object_id",
		}),
		attrs.Unbound("ContentType", &attrs.FieldConfig{
			Column: "content_type",
		}),
		attrs.Unbound("Data", &attrs.FieldConfig{
			Column: "data",
		}),
		attrs.Unbound("CreatedAt", &attrs.FieldConfig{
			Column: "created_at",
		}),
	)
}

// AsObject will return the object that this revision is for.
//
// It will populate the object with the data from the revision.
func (r *Revision) AsObject() (attrs.Definer, error) {
	var cTypeDef = contenttypes.DefinitionForType(r.ContentType)
	if cTypeDef == nil {
		return nil, errors.Errorf(
			"content type %q not found", r.ContentType,
		)
	}

	var newInstance = cTypeDef.Object()
	if newInstance == nil {
		return nil, errors.Errorf(
			"content type %q cannot provide a valid new object", r.ContentType,
		)
	}

	var definer = attrs.NewObject[attrs.Definer](newInstance)
	err := json.Unmarshal([]byte(r.Data), definer)
	if err != nil {
		return nil, err
	}

	if err = attrs.SetPrimaryKey(definer, r.ObjectID); err != nil {
		return nil, err
	}

	return definer, nil
}

// ObjectFromDB will return the object that this revision is for.
//
// It will NOT populate the object with the data from the revision.
func (r *Revision) ObjectFromDB(ctx context.Context) (attrs.Definer, error) {
	var cTypeDef = contenttypes.DefinitionForType(r.ContentType)
	var primaryKey = attrs.PrimaryKey(
		cTypeDef.Object().(attrs.Definer),
	)
	var rPrimaryKeyTyp = reflect.TypeOf(primaryKey)
	var newPrimaryKey = reflect.New(rPrimaryKeyTyp).Interface()
	err := json.Unmarshal([]byte(r.ObjectID), newPrimaryKey)
	if err != nil {
		return nil, err
	}

	objInstance, err := cTypeDef.Instance(ctx, newPrimaryKey)
	if err != nil {
		return nil, err
	}

	return objInstance.(attrs.Definer), nil
}

func ListRevisions(limit, offset int) ([]*Revision, error) {
	var rows, err = queries.GetQuerySet(&Revision{}).
		Limit(limit).
		Offset(offset).
		OrderBy("-CreatedAt").
		All()
	if err != nil {
		return nil, err
	}
	return slices.Collect(rows.Objects()), nil
}

func GetRevisionByID(id int64) (*Revision, error) {
	var row, err = queries.GetQuerySet(&Revision{}).Filter("ID", id).Get()
	if err != nil {
		return nil, err
	}
	return row.Object, nil
}

func LatestRevision(obj attrs.Definer) (*Revision, error) {
	var res, err = GetRevisionsByObject(obj, 1, 0)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, sql.ErrNoRows
	}
	return res[0], nil
}

func GetRevisionsByObject(obj attrs.Definer, limit int, offset int) ([]*Revision, error) {
	var objKey = attrs.PrimaryKey(obj)
	var objectID, err = json.Marshal(objKey)
	if err != nil {
		return nil, err
	}

	var cTypeDef = contenttypes.DefinitionForObject(obj)
	var cType = cTypeDef.ContentType()
	rowCount, rowsIter, err := queries.GetQuerySet(&Revision{}).
		Filter("ObjectID", string(objectID)).
		Filter("ContentType", cType.TypeName()).
		Limit(limit).
		Offset(offset).
		OrderBy("-CreatedAt").
		IterAll()
	if err != nil {
		return nil, errors.Wrap(
			err, "GetRevisionsByObject",
		)
	}
	var idx = 0
	var revisions = make([]*Revision, 0, rowCount)
	for row, err := range rowsIter {
		if err != nil {
			return nil, errors.Wrapf(
				err, "GetRevisionsByObject: row %d", idx,
			)
		}
		revisions = append(revisions, row.Object)
		idx++
	}
	return revisions, nil
}

func CreateRevision(forObj attrs.Definer) (*Revision, error) {
	var objKey = attrs.PrimaryKey(forObj)
	var dataBuf, err = json.Marshal(forObj)
	if err != nil {
		return nil, err
	}

	idSerialized, err := json.Marshal(objKey)
	if err != nil {
		return nil, err
	}

	var cTypeDef = contenttypes.DefinitionForObject(forObj)
	var cType = cTypeDef.ContentType()
	var rev = &Revision{
		ObjectID:    string(idSerialized),
		ContentType: cType.TypeName(),
		Data:        string(dataBuf),
	}

	return queries.GetQuerySet(&Revision{}).Create(rev)
}

func UpdateRevision(rev *Revision) error {
	var _, err = queries.GetQuerySet(&Revision{}).Filter("ID", rev.ID).Update(rev)
	return err
}

func DeleteRevision(rev *Revision) error {
	var _, err = queries.GetQuerySet(&Revision{}).Filter("ID", rev.ID).Delete()
	return err
}
