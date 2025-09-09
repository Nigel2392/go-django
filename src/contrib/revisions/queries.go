package revisions

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"reflect"
	"slices"
	"time"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/trans"
)

var (
	_ queries.ActsBeforeCreate = (*Revision)(nil)
	_ queries.OrderByDefiner   = (*Revision)(nil)
	_ attrs.Definer            = (*Revision)(nil)
)

type Revision struct {
	models.Model `table:"revisions_revision"`
	ID           int64     `json:"id"`
	ObjectID     string    `json:"object_id"`
	ContentType  string    `json:"content_type"`
	Data         string    `json:"data"`
	CreatedAt    time.Time `json:"created_at"`
}

type TypedRevision[T attrs.Definer] Revision

func (r *TypedRevision[T]) typedObject(obj any, err error) (T, error) {
	if err != nil {
		if errors.Is(err, errors.NoRows) {
			return *new(T), nil
		}
		return *new(T), errors.Wrap(err, "converting to typed revision")
	}

	if obj == nil {
		return *new(T), nil
	}

	typed, ok := obj.(T)
	if !ok {
		return *new(T), errors.TypeMismatch.Wrapf(
			"expected revision to be of type %T, got %T",
			*new(T), obj,
		)
	}

	return typed, nil
}

func (r *TypedRevision[T]) AsObject(ctx context.Context) (T, error) {
	var base = (*Revision)(r)
	return r.typedObject(base.AsObject(ctx))
}

func (r *TypedRevision[T]) ObjectFromDB(ctx context.Context) (T, error) {
	var base = (*Revision)(r)
	return r.typedObject(base.ObjectFromDB(ctx))
}

func (r *TypedRevision[T]) BeforeCreate(context.Context) error {
	return (*Revision)(r).BeforeCreate(context.Background())
}

func (r *TypedRevision[T]) OrderBy() []string {
	return (*Revision)(r).OrderBy()
}

func (r *TypedRevision[T]) FieldDefs() attrs.Definitions {
	return (*Revision)(r).FieldDefs()
}

func NewRevision(forObj attrs.Definer, getRevInfo ...QueryInfoFunc) (*Revision, error) {
	var data, err = MarshalRevisionData(forObj)
	if err != nil {
		return nil, errors.Wrap(err, "NewRevision")
	}

	objKey, cType, err := getIdAndContentType(context.Background(), forObj, getRevInfo...)
	if err != nil {
		return nil, errors.Wrap(err, "NewRevision")
	}

	var rev = &Revision{
		ObjectID:    objKey,
		ContentType: cType,
		Data:        string(data),
	}

	return rev, nil
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
	return r.Model.Define(r,
		attrs.Unbound("ID", &attrs.FieldConfig{
			Primary:  true,
			Column:   "id",
			Label:    trans.S("ID"),
			HelpText: trans.S("The unique identifier for the revision."),
		}),
		attrs.Unbound("ObjectID", &attrs.FieldConfig{
			Column:   "object_id",
			Label:    trans.S("Object ID"),
			HelpText: trans.S("The unique identifier for the object this revision belongs to."),
		}),
		attrs.Unbound("ContentType", &attrs.FieldConfig{
			Column:   "content_type",
			Label:    trans.S("Content Type"),
			HelpText: trans.S("The content type of the object this revision belongs to."),
		}),
		attrs.Unbound("Data", &attrs.FieldConfig{
			Column:   "data",
			Label:    trans.S("Data"),
			HelpText: trans.S("The data of the revision, this is a snapshot of the object at the time of the revision."),
		}),
		attrs.Unbound("CreatedAt", &attrs.FieldConfig{
			Column:   "created_at",
			Label:    trans.S("Created At"),
			HelpText: trans.S("The date and time when the revision was created."),
		}),
	)
}

// AsObject will return the object that this revision is for.
//
// It will populate the object with the data from the revision.
//
// If the context does not return true when passed to [queries.IsCommitContext]
// the object will not be fetched from the database.
func (r *Revision) AsObject(ctx context.Context) (definer attrs.Definer, err error) {
	var cTypeDef = contenttypes.DefinitionForType(r.ContentType)
	if cTypeDef == nil {
		return nil, errors.TypeMismatch.Wrapf(
			"content type %q not found", r.ContentType,
		)
	}

	var newInstance = cTypeDef.Object()
	if newInstance == nil {
		return nil, errors.TypeMismatch.Wrapf(
			"content type %q cannot provide a valid new object", r.ContentType,
		)
	}

	if queries.IsCommitContext(ctx) {
		definer, err = r.ObjectFromDB(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "getting object from db")
		}
	} else {
		definer = attrs.NewObject[attrs.Definer](newInstance)
	}

	err = UnmarshalRevisionData(definer, []byte(r.Data))
	if err != nil {
		return nil, errors.Wrap(err, "unmarshaling revision data into object")
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

func ListRevisions(ctx context.Context, limit, offset int) ([]*Revision, error) {
	var rows, err = queries.GetQuerySet(&Revision{}).
		WithContext(ctx).
		Limit(limit).
		Offset(offset).
		OrderBy("-CreatedAt").
		All()
	if err != nil {
		return nil, err
	}
	return slices.Collect(rows.Objects()), nil
}

func GetRevisionByID(ctx context.Context, id int64) (*Revision, error) {
	var row, err = queries.GetQuerySet(&Revision{}).Filter("ID", id).Get()
	if err != nil {
		return nil, err
	}
	return row.Object, nil
}

func LatestRevision(ctx context.Context, obj attrs.Definer, getRevInfo ...QueryInfoFunc) (*Revision, error) {
	var res, err = GetRevisionsByObject(ctx, obj, 1, 0, getRevInfo...)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, sql.ErrNoRows
	}
	return res[0], nil
}

func GetRevisionsByObject(ctx context.Context, obj attrs.Definer, limit int, offset int, getRevInfo ...QueryInfoFunc) ([]*Revision, error) {
	var objKey, cTypeName, err = getIdAndContentType(ctx, obj, getRevInfo...)
	if err != nil {
		return nil, errors.Wrap(err, "GetRevisionsByObject")
	}

	rowCount, rowsIter, err := queries.GetQuerySet(&Revision{}).
		WithContext(ctx).
		Filter("ObjectID", objKey).
		Filter("ContentType", cTypeName).
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

func DeleteRevisionsByObject(ctx context.Context, obj attrs.Definer, getRevInfo ...QueryInfoFunc) (int64, error) {
	var objKey, cType, err = getIdAndContentType(ctx, obj, getRevInfo...)
	if err != nil {
		return 0, errors.Wrap(err, "DeleteRevisionsByObject")
	}

	return queries.GetQuerySet(&Revision{}).
		WithContext(ctx).
		Filter("ObjectID", objKey).
		Filter("ContentType", cType).
		Delete()
}

func CreateRevision(ctx context.Context, forObj attrs.Definer, getRevInfo ...QueryInfoFunc) (*Revision, error) {
	var revision *Revision
	switch obj := forObj.(type) {
	case *Revision:
		revision = obj
	default:
		var rev, err = NewRevision(forObj, getRevInfo...)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create revision")
		}
		revision = rev
	}
	return queries.GetQuerySet(&Revision{}).WithContext(ctx).Create(revision)
}

func UpdateRevision(ctx context.Context, rev *Revision) error {
	var _, err = queries.GetQuerySet(&Revision{}).WithContext(ctx).Filter("ID", rev.ID).Update(rev)
	return err
}

func DeleteRevision(ctx context.Context, rev *Revision) error {
	var _, err = queries.GetQuerySet(&Revision{}).WithContext(ctx).Filter("ID", rev.ID).Delete()
	return err
}

type QueryInfoFunc func(ctx context.Context, obj attrs.Definer) (pk any, contentType string, err error)

type RevisionInfoDefiner interface {
	GetRevisionInfo(ctx context.Context) (pk any, contentType string, err error)
}

type RevisionDataMarshaller interface {
	MarshalRevisionData() ([]byte, error)
}

type RevisionDataUnMarshaller interface {
	UnmarshalRevisionData(data []byte) error
}

func getIdAndContentType(ctx context.Context, obj attrs.Definer, getter ...QueryInfoFunc) (pk string, contentType string, err error) {
	var (
		objKey any
		fn     QueryInfoFunc
	)

	if oi, ok := obj.(RevisionInfoDefiner); ok {
		objKey, contentType, err = oi.GetRevisionInfo(ctx)
		if err != nil {
			return "", "", err
		}
		goto marshalKey
	}

	if len(getter) > 0 && getter[0] != nil {
		fn = getter[0]
	} else {
		fn = func(ctx context.Context, obj attrs.Definer) (pk any, contentType string, err error) {
			objKey := attrs.PrimaryKey(obj)
			cTypeDef := contenttypes.DefinitionForObject(obj)
			cType := cTypeDef.ContentType()
			return objKey, cType.TypeName(), nil
		}
	}

	objKey, contentType, err = fn(ctx, obj)
	if err != nil {
		return "", "", err
	}

marshalKey:
	objectID, err := json.Marshal(objKey)
	if err != nil {
		return "", "", err
	}

	return string(objectID), contentType, nil
}

func MarshalRevisionData(obj attrs.Definer) ([]byte, error) {

	if m, ok := obj.(RevisionDataMarshaller); ok {
		return m.MarshalRevisionData()
	}

	var databuf = new(bytes.Buffer)
	var objMap = make(map[string]any)
	var defs = obj.FieldDefs()
	for _, def := range defs.Fields() {
		var name = def.Name()
		var value, err = def.Value()
		if err != nil {
			return nil, err
		}
		objMap[name] = value
	}

	var enc = json.NewEncoder(databuf)
	enc.SetEscapeHTML(true)
	err := enc.Encode(objMap)
	if err != nil {
		return nil, err
	}
	return databuf.Bytes(), nil
}

func UnmarshalRevisionData(obj attrs.Definer, data []byte) error {
	if m, ok := obj.(RevisionDataUnMarshaller); ok {
		return m.UnmarshalRevisionData(data)
	}

	var dataMap = make(map[string]json.RawMessage)
	var err = json.Unmarshal(data, &dataMap)
	if err != nil {
		return errors.Wrap(err, "unmarshaling revision data")
	}

	var defs = obj.FieldDefs()
	var fields = defs.Fields()
	for _, def := range fields {
		var name = def.Name()
		var value, ok = dataMap[name]
		if !ok {
			continue
		}

		dbTyp, ok := drivers.DBType(def)
		if !ok {
			return errors.TypeMismatch.Wrapf(
				"cannot determine database type for field %q", name,
			)
		}

		var fT = drivers.DBToDefaultGoType(dbTyp)
		if fT.Kind() == reflect.Ptr {
			fT = fT.Elem()
		}

		var rValuePtr = reflect.New(fT)
		var valuePtr = rValuePtr.Interface()
		err = json.Unmarshal(value, valuePtr)
		if err != nil {
			return errors.Wrapf(
				err, "unmarshaling field %q", name,
			)
		}

		if err = def.Scan(rValuePtr.Elem().Interface()); err != nil {
			return errors.Wrapf(
				err, "scanning field %q for %T", name, rValuePtr.Elem().Interface(),
			)
		}
	}

	return nil
}
