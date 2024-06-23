package auditlogs_mysql

import (
	"database/sql"
	"fmt"

	auditlogs "github.com/Nigel2392/django/contrib/reports/audit_logs"
	"github.com/Nigel2392/django/core/logger"
	"github.com/google/uuid"
)

const createTableMySQL = `CREATE TABLE IF NOT EXISTS audit_logs (
	id BINARY(16) PRIMARY KEY NOT NULL,
	type VARCHAR(255) NOT NULL,
	level INT NOT NULL,
	timestamp DATETIME NOT NULL,
	object_id BLOB,
	content_type VARCHAR(255),
	data TEXT
);`

const insertMySQL = `INSERT INTO audit_logs (id, type, level, timestamp, object_id, content_type, data) VALUES (?, ?, ?, ?, ?, ?, ?);`
const selectMySQL = `SELECT id, type, level, timestamp, object_id, content_type, data FROM audit_logs WHERE id = ?;`
const selectManyMySQL = `SELECT id, type, level, timestamp, object_id, content_type, data FROM audit_logs ORDER BY timestamp DESC LIMIT ? OFFSET ?;`
const selectTypedMySQL = `SELECT id, type, level, timestamp, object_id, content_type, data FROM audit_logs WHERE type = ? ORDER BY timestamp DESC LIMIT ? OFFSET ?;`

type MySQLStorageBackend struct {
	db *sql.DB
}

func NewMySQLStorageBackend(db *sql.DB) auditlogs.StorageBackend {
	return &MySQLStorageBackend{db: db}
}

func (s *MySQLStorageBackend) Setup() error {
	_, err := s.db.Exec(createTableMySQL)
	return err
}

func (s *MySQLStorageBackend) Store(logEntry auditlogs.LogEntry) (uuid.UUID, error) {
	var log = logger.NameSpace(logEntry.Type())
	log.Log(logEntry.Level(), fmt.Sprint(logEntry))

	var id, typeStr, level, timestamp, objectID, contentType, data = auditlogs.SerializeRow(logEntry)
	if id == uuid.Nil {
		id = uuid.New()
	}

	_, err := s.db.Exec(insertMySQL, id, typeStr, level, timestamp, objectID, contentType, string(data))
	return id, err
}

func (s *MySQLStorageBackend) StoreMany(logEntries []auditlogs.LogEntry) ([]uuid.UUID, error) {
	var ids = make([]uuid.UUID, 0)
	for _, logEntry := range logEntries {
		id, err := s.Store(logEntry)
		if err != nil {
			return ids, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *MySQLStorageBackend) Retrieve(id uuid.UUID) (auditlogs.LogEntry, error) {

	row := s.db.QueryRow(selectMySQL, id)
	if row.Err() != nil {
		return nil, row.Err()
	}

	return auditlogs.ScanRow(row)
}

func (s *MySQLStorageBackend) RetrieveMany(amount, offset int) ([]auditlogs.LogEntry, error) {
	rows, err := s.db.Query(selectManyMySQL, amount, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries = make([]auditlogs.LogEntry, 0)
	for rows.Next() {
		entry, err := auditlogs.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func (s *MySQLStorageBackend) RetrieveTyped(logType string, amount, offset int) ([]auditlogs.LogEntry, error) {
	rows, err := s.db.Query(selectTypedMySQL, logType, amount, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries = make([]auditlogs.LogEntry, 0)
	for rows.Next() {
		entry, err := auditlogs.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}
