package auditlogs_sqlite

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	auditlogs "github.com/Nigel2392/django/contrib/reports/audit_logs"
	"github.com/google/uuid"
)

const createTableSQLITE = `CREATE TABLE IF NOT EXISTS audit_logs (
	id BLOB(16) PRIMARY KEY NOT NULL,
	type TEXT NOT NULL,
	level NUMBER NOT NULL,
	timestamp TIMESTAMP NOT NULL,
	user_id TEXT,
	object_id TEXT,
	content_type TEXT,
	data TEXT
);`

const insertSQLITE = `INSERT INTO audit_logs (id, type, level, timestamp, user_id, object_id, content_type, data) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8);`
const selectSQLITE = `SELECT id, type, level, timestamp, user_id, object_id, content_type, data FROM audit_logs WHERE id = ?1;`
const selectManySQLITE = `SELECT id, type, level, timestamp, user_id, object_id, content_type, data FROM audit_logs ORDER BY timestamp DESC LIMIT ?1 OFFSET ?2;`
const selectTypedSQLITE = `SELECT id, type, level, timestamp, user_id, object_id, content_type, data FROM audit_logs WHERE type = ?1 ORDER BY timestamp DESC LIMIT ?2 OFFSET ?3;`
const selectForUserSQLITE = `SELECT id, type, level, timestamp, user_id, object_id, content_type, data FROM audit_logs WHERE user_id = ?1 ORDER BY timestamp DESC LIMIT ?2 OFFSET ?3;`
const selectForObjectSQLITE = `SELECT id, type, level, timestamp, user_id, object_id, content_type, data FROM audit_logs WHERE object_id = ?1 ORDER BY timestamp DESC LIMIT ?2 OFFSET ?3;`

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
	// var log = logger.NameSpace(logEntry.Type())
	// log.Log(logEntry.Level(), fmt.Sprint(logEntry))

	var id, typeStr, level, timestamp, userID, objectID, contentType, data = auditlogs.SerializeRow(logEntry)
	if id == uuid.Nil {
		id = uuid.New()
	}

	_, err := s.db.Exec(insertSQLITE, id, typeStr, level, timestamp, string(userID), string(objectID), contentType, string(data))
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

func (s *sqliteStorageBackend) RetrieveForUser(userID interface{}, amount, offset int) ([]auditlogs.LogEntry, error) {
	var idBuf = new(bytes.Buffer)
	enc := json.NewEncoder(idBuf)
	err := enc.Encode(userID)
	if err != nil {
		return nil, err
	}
	var id = idBuf.Bytes()
	rows, err := s.db.Query(selectForUserSQLITE, string(id), amount, offset)
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

func (s *sqliteStorageBackend) RetrieveForObject(objectID interface{}, amount, offset int) ([]auditlogs.LogEntry, error) {
	var idBuf = new(bytes.Buffer)
	enc := json.NewEncoder(idBuf)
	err := enc.Encode(objectID)
	if err != nil {
		return nil, err
	}
	var id = idBuf.Bytes()
	rows, err := s.db.Query(selectForObjectSQLITE, string(id), amount, offset)
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

func (s *sqliteStorageBackend) Filter(filters []auditlogs.AuditLogFilter, amount, offset int) ([]auditlogs.LogEntry, error) {
	var query string
	var args []interface{}
	for i, filter := range filters {
		switch filter.Name() {
		case auditlogs.AuditLogFilterID:
			var value = filter.Value()
			if len(value) > 1 {
				var inQ = make([]string, len(value))
				for i, v := range value {
					inQ[i] = "?"
					args = append(args, v)
				}
				query += fmt.Sprintf("id IN (%s)", strings.Join(inQ, ","))
			} else {
				query += "id = ?"
				args = append(args, value[0])
			}
		case auditlogs.AuditLogFilterType:
			var value = filter.Value()
			if len(value) > 1 {
				var inQ = make([]string, len(value))
				for i, v := range value {
					inQ[i] = "?"
					args = append(args, v)
				}
				query += fmt.Sprintf("type IN (%s)", strings.Join(inQ, ","))
			} else {
				query += "type = ?"
				args = append(args, value[0])
			}
		case auditlogs.AuditLogFilterLevel_EQ:
			query += "level = ?"
			args = append(args, filter.Value()[0])
		case auditlogs.AuditLogFilterLevel_GT:
			query += "level > ?"
			args = append(args, filter.Value()[0])
		case auditlogs.AuditLogFilterLevel_LT:
			query += "level < ?"
			args = append(args, filter.Value()[0])
		case auditlogs.AuditLogFilterTimestamp_EQ:
			query += "timestamp = ?"
			args = append(args, filter.Value()[0])
		case auditlogs.AuditLogFilterTimestamp_GT:
			query += "timestamp > ?"
			args = append(args, filter.Value()[0])
		case auditlogs.AuditLogFilterTimestamp_LT:
			query += "timestamp < ?"
			args = append(args, filter.Value()[0])
		case auditlogs.AuditLogFilterUserID:
			query += "user_id = ?"
			args = append(args, filter.Value()[0])
		case auditlogs.AuditLogFilterObjectID:
			query += "object_id = ?"
			args = append(args, filter.Value()[0])
		case auditlogs.AuditLogFilterContentType:
			query += "content_type = ?"
			args = append(args, filter.Value()[0])
		case auditlogs.AuditLogFilterData:
			var value = filter.Value()
			switch value[0].(type) {
			case string:
				query += "data = ?"
				args = append(args, value[0])
			default:
				query += "data = ?"
				var dataBuf = new(bytes.Buffer)
				enc := json.NewEncoder(dataBuf)
				err := enc.Encode(value[0])
				if err != nil {
					return nil, err
				}
				args = append(args, dataBuf.Bytes())
			}
		}
		if i < len(filters)-1 {
			query += " AND "
		}
	}

	rows, err := s.db.Query(fmt.Sprintf("SELECT id, type, level, timestamp, user_id, object_id, content_type, data FROM audit_logs WHERE %s ORDER BY timestamp DESC LIMIT ? OFFSET ?;", query), append(args, amount, offset)...)
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
