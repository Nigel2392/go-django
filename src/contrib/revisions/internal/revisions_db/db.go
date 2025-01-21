package revisions_db

import (
	"context"
	"database/sql"
	"time"

	"github.com/Nigel2392/go-django/src/models"
)

type Querier interface {
	Close() error
	WithTx(tx *sql.Tx) Querier
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

var (
	registry   = models.NewBackendRegistry[Querier]()
	Register   = registry.RegisterForDriver
	GetBackend = registry.BackendForDB
)
