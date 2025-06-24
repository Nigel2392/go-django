package revisions_db

import (
	"context"
	"time"

	"github.com/Nigel2392/go-django-queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/models"
)

type Querier interface {
	Close() error
	WithTx(tx drivers.Transaction) Querier
	DeleteRevision(ctx context.Context, id int64) error
	GetRevisionByID(ctx context.Context, id int64) (Revision, error)
	GetRevisionsByObjectID(ctx context.Context, objectID string, contentType string, limit int32, offset int32) ([]Revision, error)
	InsertRevision(ctx context.Context, objectID string, contentType string, data string) (int64, error)
	ListRevisions(ctx context.Context, limit int32, offset int32) ([]Revision, error)
	UpdateRevision(ctx context.Context, objectID string, contentType string, data string, iD int64) error
}

type Revision struct {
	ID          int64     `json:"id"`
	ObjectID    string    `json:"object_id"`
	ContentType string    `json:"content_type"`
	Data        string    `json:"data"`
	CreatedAt   time.Time `json:"created_at"`
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

var (
	registry   = models.NewBackendRegistry[Querier]()
	Register   = registry.RegisterForDriver
	GetBackend = registry.BackendForDB
)
