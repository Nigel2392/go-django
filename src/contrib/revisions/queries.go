package revisions

import (
	"context"
	"database/sql"
	"encoding/json"
	"reflect"
	"unsafe"

	_ "github.com/Nigel2392/go-django/src/contrib/revisions/internal/revisions-mysql"
	_ "github.com/Nigel2392/go-django/src/contrib/revisions/internal/revisions-sqlite"
	"github.com/Nigel2392/go-django/src/contrib/revisions/internal/revisions_db"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/pkg/errors"
)

func revisionsDbListToRevisions(dbRevs []revisions_db.Revision) []Revision {
	return *(*[]Revision)(unsafe.Pointer(&dbRevs))
}

type RevisionQuerier struct {
	ctx     context.Context
	Querier revisions_db.Querier
}

type Revision revisions_db.Revision

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

	var rVal = reflect.ValueOf(newInstance)
	if rVal.Kind() != reflect.Ptr {
		rVal = reflect.New(
			rVal.Type(),
		)
	}

	err := json.Unmarshal([]byte(r.Data), rVal.Interface())
	if err != nil {
		return nil, err
	}

	var definer = rVal.Interface().(attrs.Definer)
	if err = attrs.SetPrimaryKey(definer, r.ObjectID); err != nil {
		return nil, err
	}

	return definer, nil
}

// ObjectFromDB will return the object that this revision is for.
//
// It will NOT populate the object with the data from the revision.
func (r *Revision) ObjectFromDB() (attrs.Definer, error) {
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

	objInstance, err := cTypeDef.GetInstance(newPrimaryKey)
	if err != nil {
		return nil, err
	}

	return objInstance.(attrs.Definer), nil
}

func (r *RevisionQuerier) ListRevisions(limit, offset int) ([]Revision, error) {
	res, err := r.Querier.ListRevisions(r.ctx, int32(limit), int32(offset))
	if err != nil {
		return nil, err
	}
	return revisionsDbListToRevisions(res), nil
}

func (r *RevisionQuerier) GetRevisionByID(id int64) (*Revision, error) {
	res, err := r.Querier.GetRevisionByID(r.ctx, id)
	if err != nil {
		return nil, err
	}
	return (*Revision)(&res), nil
}

func (r *RevisionQuerier) LatestRevision(obj attrs.Definer) (*Revision, error) {
	var res, err = r.GetRevisionsByObject(obj, 1, 0)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, sql.ErrNoRows
	}
	return &res[0], nil
}

func (r *RevisionQuerier) GetRevisionsByObject(obj attrs.Definer, limit int, offset int) ([]Revision, error) {
	var objKey = attrs.PrimaryKey(obj)
	var objectID, err = json.Marshal(objKey)
	if err != nil {
		return nil, err
	}

	var cTypeDef = contenttypes.DefinitionForObject(obj)
	var cType = cTypeDef.ContentType()
	results, err := r.Querier.GetRevisionsByObjectID(
		r.ctx,
		string(objectID),
		cType.TypeName(),
		int32(limit),
		int32(offset),
	)

	if err != nil {
		return nil, errors.Wrap(
			err, "GetRevisionsByObjectID",
		)
	}
	return revisionsDbListToRevisions(results), nil
}

func (r *RevisionQuerier) CreateRevision(forObj attrs.Definer) (*Revision, error) {
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
	var rev = Revision{
		ObjectID:    string(idSerialized),
		ContentType: cType.TypeName(),
		Data:        string(dataBuf),
	}

	id, err := r.Querier.InsertRevision(
		r.ctx,
		rev.ObjectID,
		rev.ContentType,
		rev.Data,
	)
	if err != nil {
		return &rev, err
	}

	rev.ID = id
	return &rev, nil
}

func (r *RevisionQuerier) UpdateRevision(rev Revision) error {
	return r.Querier.UpdateRevision(
		r.ctx,
		rev.ObjectID,
		rev.ContentType,
		rev.Data,
		rev.ID,
	)
}

func (r *RevisionQuerier) DeleteRevision(rev Revision) error {
	return r.Querier.DeleteRevision(r.ctx, rev.ID)
}
