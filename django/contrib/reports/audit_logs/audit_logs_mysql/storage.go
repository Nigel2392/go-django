package auditlogs_mysql

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	auditlogs "github.com/Nigel2392/django/contrib/reports/audit_logs"
	"github.com/google/uuid"
)

const (
	createTableMySQL = `CREATE TABLE IF NOT EXISTS audit_logs (
	id VARCHAR(36) PRIMARY KEY NOT NULL,
	type VARCHAR(255) NOT NULL,
	level INT NOT NULL,
	timestamp DATETIME NOT NULL,
	user_id TEXT,
	object_id TEXT,
	content_type VARCHAR(255),
	data TEXT
);`
	insertMySQL          = `INSERT INTO audit_logs (id, type, level, timestamp, user_id, object_id, content_type, data) VALUES (?, ?, ?, ?, ?, ?, ?, ?);`
	selectMySQL          = `SELECT id, type, level, timestamp, user_id, object_id, content_type, data FROM audit_logs WHERE id = ?;`
	selectManyMySQL      = `SELECT id, type, level, timestamp, user_id, object_id, content_type, data FROM audit_logs ORDER BY timestamp DESC LIMIT ? OFFSET ?;`
	selectTypedMySQL     = `SELECT id, type, level, timestamp, user_id, object_id, content_type, data FROM audit_logs WHERE type = ? ORDER BY timestamp DESC LIMIT ? OFFSET ?;`
	selectForUserMySQL   = `SELECT id, type, level, timestamp, user_id, object_id, content_type, data FROM audit_logs WHERE user_id = ? ORDER BY timestamp DESC LIMIT ? OFFSET ?;`
	selectForObjectMySQL = `SELECT id, type, level, timestamp, user_id, object_id, content_type, data FROM audit_logs WHERE object_id = ? ORDER BY timestamp DESC LIMIT ? OFFSET ?;`
)

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
	// var log = logger.NameSpace(logEntry.Type())
	// log.Log(logEntry.Level(), fmt.Sprint(logEntry))

	var id, typeStr, level, timestamp, userID, objectID, contentType, data = auditlogs.SerializeRow(logEntry)

	_, err := s.db.Exec(insertMySQL, id, typeStr, level, timestamp, string(userID), string(objectID), contentType, string(data))
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

func (s *MySQLStorageBackend) RetrieveForUser(userID interface{}, amount, offset int) ([]auditlogs.LogEntry, error) {
	var idBuf = new(bytes.Buffer)
	enc := json.NewEncoder(idBuf)
	err := enc.Encode(userID)
	if err != nil {
		return nil, err
	}
	var id = idBuf.Bytes()
	rows, err := s.db.Query(selectForUserMySQL, string(id), amount, offset)
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

func (s *MySQLStorageBackend) RetrieveForObject(objectID interface{}, amount, offset int) ([]auditlogs.LogEntry, error) {
	var idBuf = new(bytes.Buffer)
	enc := json.NewEncoder(idBuf)
	err := enc.Encode(objectID)
	if err != nil {
		return nil, err
	}
	var id = idBuf.Bytes()
	rows, err := s.db.Query(selectForObjectMySQL, string(id), amount, offset)
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

func (s *MySQLStorageBackend) EntryFilter(filters []auditlogs.AuditLogFilter, amount, offset int) ([]auditlogs.LogEntry, error) {
	var query = new(strings.Builder)
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
				fmt.Fprintf(query, "id IN (%s)", strings.Join(inQ, ","))
			} else {
				fmt.Fprint(query, "id = ?")
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
				fmt.Fprintf(query, "type IN (%s)", strings.Join(inQ, ","))
			} else {
				fmt.Fprint(query, "type = ?")
				args = append(args, value[0])
			}
		case auditlogs.AuditLogFilterLevel_EQ:
			fmt.Fprint(query, "level = ?")
			args = append(args, filter.Value()[0])
		case auditlogs.AuditLogFilterLevel_GT:
			fmt.Fprint(query, "level > ?")
			args = append(args, filter.Value()[0])
		case auditlogs.AuditLogFilterLevel_LT:
			fmt.Fprint(query, "level < ?")
			args = append(args, filter.Value()[0])
		case auditlogs.AuditLogFilterTimestamp_EQ:
			fmt.Fprint(query, "timestamp = ?")
			args = append(args, filter.Value()[0])
		case auditlogs.AuditLogFilterTimestamp_GT:
			fmt.Fprint(query, "timestamp > ?")
			args = append(args, filter.Value()[0])
		case auditlogs.AuditLogFilterTimestamp_LT:
			fmt.Fprint(query, "timestamp < ?")
			args = append(args, filter.Value()[0])
		case auditlogs.AuditLogFilterUserID:
			if len(filter.Value()) > 1 {
				var inQ = make([]string, len(filter.Value()))
				for i, v := range filter.Value() {
					inQ[i] = "?"
					args = append(args, v)
				}
				fmt.Fprintf(query, "user_id IN (%s)", strings.Join(inQ, ","))
			} else {
				fmt.Fprint(query, "user_id = ?")
				args = append(args, filter.Value()[0])
			}
		case auditlogs.AuditLogFilterObjectID:
			if len(filter.Value()) > 1 {
				var inQ = make([]string, len(filter.Value()))
				for i, v := range filter.Value() {
					inQ[i] = "?"
					args = append(args, v)
				}
				fmt.Fprintf(query, "object_id IN (%s)", strings.Join(inQ, ","))
			} else {
				fmt.Fprint(query, "object_id = ?")
				args = append(args, filter.Value()[0])
			}
		case auditlogs.AuditLogFilterContentType:
			fmt.Fprint(query, "content_type = ?")
			args = append(args, filter.Value()[0])
		case auditlogs.AuditLogFilterData:
			var value = filter.Value()
			switch value[0].(type) {
			case string:
				fmt.Fprint(query, "data = ?")
				args = append(args, value[0])
			default:
				fmt.Fprint(query, "data = ?")
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
			fmt.Fprint(query, " AND ")
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
