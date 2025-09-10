package revisions

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	queries "github.com/Nigel2392/go-django/queries/src"
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

func (r *Revision) SetObject(obj attrs.Definer) error {
	var pk, cType, err = getIdAndContentType(context.Background(), obj)
	if err != nil {
		return errors.Wrap(err, "failed to get object ID and content type")
	}

	if r.ObjectID != pk {
		return errors.CheckViolation.Wrapf(
			"object ID mismatch: revision has %q, object has %q",
			r.ObjectID, pk,
		)
	}

	if r.ContentType != cType {
		return errors.TypeMismatch.Wrapf(
			"content type mismatch: revision %q does not match object %q",
			r.ContentType, cType,
		)
	}

	data, err := MarshalRevisionData(obj)
	if err != nil {
		return errors.Wrap(err, "failed to marshal revision data")
	}

	r.Data = string(data)
	return nil
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
