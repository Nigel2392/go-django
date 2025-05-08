package backend

import (
	"database/sql"
	"time"

	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/logger"
	models "github.com/Nigel2392/go-django/src/models"
	"github.com/google/uuid"
)

var (
	registry   = models.NewBackendRegistry[StorageBackend]()
	Register   = registry.RegisterForDriver
	GetBackend = registry.BackendForDB
)

type StorageBackend interface {
	WithTx(tx *sql.Tx) StorageBackend
	Close() error

	Count() (int, error)
	Store(logEntry LogEntry) (uuid.UUID, error)
	RetrieveMany(amount, offset int) ([]LogEntry, error)
	StoreMany(logEntries []LogEntry) ([]uuid.UUID, error)
	Retrieve(id uuid.UUID) (LogEntry, error)
	EntryFilter(filters []AuditLogFilter, amount, offset int) ([]LogEntry, error)
	CountFilter(filters []AuditLogFilter) (int, error)
}

type AuditLogFilter interface {
	Is(string) bool
	Name() string
	Value() []interface{}
}

const (
	AuditLogFilterID           = "id"
	AuditLogFilterType         = "type"
	AuditLogFilterLevel_EQ     = "level_eq"
	AuditLogFilterLevel_GT     = "level_gt"
	AuditLogFilterLevel_LT     = "level_lt"
	AuditLogFilterTimestamp_EQ = "timestamp_eq"
	AuditLogFilterTimestamp_GT = "timestamp_gt"
	AuditLogFilterTimestamp_LT = "timestamp_lt"
	AuditLogFilterUserID       = "user_id"
	AuditLogFilterObjectID     = "object_id"
	AuditLogFilterContentType  = "content_type"
	AuditLogFilterData         = "data"
)

type LogEntry interface {
	ID() uuid.UUID
	Type() string
	Level() logger.LogLevel
	Timestamp() time.Time
	UserID() interface{}
	ObjectID() interface{}
	ContentType() contenttypes.ContentType
	Data() map[string]interface{}
}
