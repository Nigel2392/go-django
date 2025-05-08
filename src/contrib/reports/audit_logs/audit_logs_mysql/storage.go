package auditlogs_mysql

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Nigel2392/go-django/src/contrib/reports/audit_logs/backend"
	"github.com/Nigel2392/go-django/src/models"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
)

func init() {
	backend.Register(&mysql.MySQLDriver{}, &models.BaseBackend[backend.StorageBackend]{
		CreateTableFn: func(d *sql.DB) error {
			_, err := d.Exec(createTableMySQL)
			return err
		},
		NewQuerier: func(d *sql.DB) (backend.StorageBackend, error) {
			return NewMySQLStorageBackend(d), nil
		},
	})
}

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
	selectCountMySQL     = `SELECT COUNT(id) FROM audit_logs;`
)

type MySQLStorageBackend struct {
	db *sql.DB
}

func NewMySQLStorageBackend(db *sql.DB) backend.StorageBackend {
	return &MySQLStorageBackend{db: db}
}

func (i *MySQLStorageBackend) WithTx(tx *sql.Tx) backend.StorageBackend {
	return i
}

func (i *MySQLStorageBackend) Close() error {
	return nil
}

func (s *MySQLStorageBackend) Store(logEntry backend.LogEntry) (uuid.UUID, error) {
	// var log = logger.NameSpace(logEntry.Type())
	// log.Log(logEntry.Level(), fmt.Sprint(logEntry))

	var id, typeStr, level, timestamp, userID, objectID, contentType, data = backend.SerializeRow(logEntry)

	_, err := s.db.Exec(insertMySQL, id, typeStr, level, timestamp, string(userID), string(objectID), contentType, string(data))
	return id, err
}

func (s *MySQLStorageBackend) StoreMany(logEntries []backend.LogEntry) ([]uuid.UUID, error) {
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

func (s *MySQLStorageBackend) Retrieve(id uuid.UUID) (backend.LogEntry, error) {

	row := s.db.QueryRow(selectMySQL, id)
	if row.Err() != nil {
		return nil, row.Err()
	}

	return backend.ScanRow(row)
}

func (s *MySQLStorageBackend) RetrieveForUser(userID interface{}, amount, offset int) ([]backend.LogEntry, error) {
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

	var entries = make([]backend.LogEntry, 0)
	for rows.Next() {
		entry, err := backend.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func (s *MySQLStorageBackend) RetrieveForObject(objectID interface{}, amount, offset int) ([]backend.LogEntry, error) {
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

	var entries = make([]backend.LogEntry, 0)
	for rows.Next() {
		entry, err := backend.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func (s *MySQLStorageBackend) RetrieveMany(amount, offset int) ([]backend.LogEntry, error) {
	rows, err := s.db.Query(selectManyMySQL, amount, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries = make([]backend.LogEntry, 0)
	for rows.Next() {
		entry, err := backend.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func (s *MySQLStorageBackend) RetrieveTyped(logType string, amount, offset int) ([]backend.LogEntry, error) {
	rows, err := s.db.Query(selectTypedMySQL, logType, amount, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries = make([]backend.LogEntry, 0)
	for rows.Next() {
		entry, err := backend.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func (s *MySQLStorageBackend) EntryFilter(filters []backend.AuditLogFilter, amount, offset int) ([]backend.LogEntry, error) {
	var query = new(strings.Builder)
	var args, err = makeWhereQuery(query, filters)
	if err != nil {
		return nil, err
	}
	var queryString = fmt.Sprintf(
		`SELECT 
			id,
			type,
			level,
			timestamp,
			user_id,
			object_id,
			content_type,
			data
			FROM audit_logs
			WHERE %s
			ORDER BY timestamp DESC
			LIMIT ? OFFSET ?;`,
		query,
	)
	rows, err := s.db.Query(queryString, append(args, amount, offset)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries = make([]backend.LogEntry, 0)
	for rows.Next() {
		entry, err := backend.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func (s *MySQLStorageBackend) CountFilter(filters []backend.AuditLogFilter) (int, error) {
	var query = new(strings.Builder)
	var args, err = makeWhereQuery(query, filters)
	if err != nil {
		return 0, err
	}
	row := s.db.QueryRow(fmt.Sprintf("SELECT COUNT(id) FROM audit_logs WHERE %s;", query), args...)
	if row.Err() != nil {
		return 0, row.Err()
	}

	var count int
	err = row.Scan(&count)
	return count, err
}

func (s *MySQLStorageBackend) Count() (int, error) {
	row := s.db.QueryRow(selectCountMySQL)
	if row.Err() != nil {
		return 0, row.Err()
	}

	var count int
	err := row.Scan(&count)
	return count, err
}

func makeWhereQuery(query *strings.Builder, filters []backend.AuditLogFilter) ([]interface{}, error) {
	var args []interface{}
	for i, filter := range filters {
		switch filter.Name() {
		case backend.AuditLogFilterID:
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
		case backend.AuditLogFilterType:
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
		case backend.AuditLogFilterLevel_EQ:
			fmt.Fprint(query, "level = ?")
			args = append(args, filter.Value()[0])
		case backend.AuditLogFilterLevel_GT:
			fmt.Fprint(query, "level > ?")
			args = append(args, filter.Value()[0])
		case backend.AuditLogFilterLevel_LT:
			fmt.Fprint(query, "level < ?")
			args = append(args, filter.Value()[0])
		case backend.AuditLogFilterTimestamp_EQ:
			fmt.Fprint(query, "timestamp = ?")
			args = append(args, filter.Value()[0])
		case backend.AuditLogFilterTimestamp_GT:
			fmt.Fprint(query, "timestamp > ?")
			args = append(args, filter.Value()[0])
		case backend.AuditLogFilterTimestamp_LT:
			fmt.Fprint(query, "timestamp < ?")
			args = append(args, filter.Value()[0])
		case backend.AuditLogFilterUserID:
			if len(filter.Value()) > 1 {
				var inQ = make([]string, len(filter.Value()))
				for i, v := range filter.Value() {
					inQ[i] = "?"
					var b = new(bytes.Buffer)
					enc := json.NewEncoder(b)
					err := enc.Encode(v)
					if err != nil {
						return nil, err
					}
					args = append(args, b.String())
				}
				fmt.Fprintf(query, "user_id IN (%s)", strings.Join(inQ, ","))
			} else {
				fmt.Fprint(query, "user_id = ?")
				var b = new(bytes.Buffer)
				enc := json.NewEncoder(b)
				err := enc.Encode(filter.Value()[0])
				if err != nil {
					return nil, err
				}
				args = append(args, b.String())
			}
		case backend.AuditLogFilterObjectID:
			if len(filter.Value()) > 1 {
				var inQ = make([]string, len(filter.Value()))
				for i, v := range filter.Value() {
					inQ[i] = "?"
					var b = new(bytes.Buffer)
					enc := json.NewEncoder(b)
					err := enc.Encode(v)
					if err != nil {
						return nil, err
					}
					args = append(args, b.String())
				}
				fmt.Fprintf(query, "object_id IN (%s)", strings.Join(inQ, ","))
			} else {
				fmt.Fprint(query, "object_id = ?")
				var b = new(bytes.Buffer)
				enc := json.NewEncoder(b)
				err := enc.Encode(filter.Value()[0])
				if err != nil {
					return nil, err
				}
				args = append(args, b.String())
			}
		case backend.AuditLogFilterContentType:
			fmt.Fprint(query, "content_type = ?")
			args = append(args, filter.Value()[0])
		case backend.AuditLogFilterData:
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
	return args, nil
}
