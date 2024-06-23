package auditlogs

import (
	"errors"

	"github.com/google/uuid"
)

var ErrLogEntryNotFound = errors.New("log entry not found")

type StorageBackend interface {
	Setup() error
	Store(logEntry LogEntry) (uuid.UUID, error)
	StoreMany(logEntries []LogEntry) ([]uuid.UUID, error)
	Retrieve(id uuid.UUID) (LogEntry, error)
	RetrieveForObject(objectID interface{}, amount, offset int) ([]LogEntry, error)
	RetrieveForUser(userID interface{}, amount, offset int) ([]LogEntry, error)
	RetrieveMany(amount, offset int) ([]LogEntry, error)
	RetrieveTyped(logType string, amount, offset int) ([]LogEntry, error)
	Filter(filters []AuditLogFilter, amount, offset int) ([]LogEntry, error)
}
