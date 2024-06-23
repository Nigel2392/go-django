package auditlogs_sqlite

import (
	"database/sql"
	"fmt"

	auditlogs "github.com/Nigel2392/django/contrib/reports/audit_logs"
	"github.com/Nigel2392/django/core/logger"
	"github.com/google/uuid"
)

const createTableSQLITE = `CREATE TABLE IF NOT EXISTS audit_logs (
	id BLOB(16) PRIMARY KEY NOT NULL,
	type TEXT NOT NULL,
	level NUMBER NOT NULL,
	timestamp TEXT NOT NULL,
	user_id BLOB,
	object_id BLOB,
	content_type TEXT,
	data TEXT
);`

const insertSQLITE = `INSERT INTO audit_logs (id, type, level, timestamp, user_id, object_id, content_type, data) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8);`
const selectSQLITE = `SELECT id, type, level, timestamp, user_id, object_id, content_type, data FROM audit_logs WHERE id = ?1;`
const selectManySQLITE = `SELECT id, type, level, timestamp, user_id, object_id, content_type, data FROM audit_logs ORDER BY timestamp DESC LIMIT ?1 OFFSET ?2;`
const selectTypedSQLITE = `SELECT id, type, level, timestamp, user_id, object_id, content_type, data FROM audit_logs WHERE type = ? ORDER BY timestamp DESC LIMIT ?1 OFFSET ?2;`

type sqliteStorageBackend struct {
	db *sql.DB
}

func NewSQLiteStorageBackend(db *sql.DB) auditlogs.StorageBackend {
	return &sqliteStorageBackend{db: db}
}

func (s *sqliteStorageBackend) Setup() error {
	_, err := s.db.Exec(createTableSQLITE)
	return err
}

func (s *sqliteStorageBackend) Store(logEntry auditlogs.LogEntry) (uuid.UUID, error) {
	var log = logger.NameSpace(logEntry.Type())
	log.Log(logEntry.Level(), fmt.Sprint(logEntry))

	var id, typeStr, level, timestamp, userID, objectID, contentType, data = auditlogs.SerializeRow(logEntry)
	if id == uuid.Nil {
		id = uuid.New()
	}

	_, err := s.db.Exec(insertSQLITE, id, typeStr, level, timestamp, userID, objectID, contentType, string(data))
	return id, err
}

func (s *sqliteStorageBackend) StoreMany(logEntries []auditlogs.LogEntry) ([]uuid.UUID, error) {
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

func (s *sqliteStorageBackend) Retrieve(id uuid.UUID) (auditlogs.LogEntry, error) {

	row := s.db.QueryRow(selectSQLITE, id)
	if row.Err() != nil {
		return nil, row.Err()
	}

	return auditlogs.ScanRow(row)
}

func (s *sqliteStorageBackend) RetrieveMany(amount, offset int) ([]auditlogs.LogEntry, error) {
	rows, err := s.db.Query(selectManySQLITE, amount, offset)
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

func (s *sqliteStorageBackend) RetrieveTyped(logType string, amount, offset int) ([]auditlogs.LogEntry, error) {
	rows, err := s.db.Query(selectTypedSQLITE, logType, amount, offset)
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
