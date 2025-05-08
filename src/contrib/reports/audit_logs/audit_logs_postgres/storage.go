package auditlogs_postgres

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Nigel2392/go-django/src/contrib/reports/audit_logs/backend"
	"github.com/Nigel2392/go-django/src/models"
	"github.com/google/uuid"
	pg_stdlib "github.com/jackc/pgx/v5/stdlib"
)

func init() {
	backend.Register(&pg_stdlib.Driver{}, &models.BaseBackend[backend.StorageBackend]{
		CreateTableFn: func(d *sql.DB) error {
			_, err := d.Exec(createTablePostgres)
			return err
		},
		NewQuerier: func(d *sql.DB) (backend.StorageBackend, error) {
			return NewPostgresStorageBackend(d), nil
		},
	})
}

const (
	createTablePostgres = `CREATE TABLE IF NOT EXISTS audit_logs (
		id UUID PRIMARY KEY NOT NULL,
		type TEXT NOT NULL,
		level INTEGER NOT NULL,
		timestamp TIMESTAMP NOT NULL,
		user_id TEXT,
		object_id TEXT,
		content_type TEXT,
		data TEXT
	);`

	insertPostgres          = `INSERT INTO audit_logs (id, type, level, timestamp, user_id, object_id, content_type, data) VALUES ($1, $2, $3, $4, $5, $6, $7, $8);`
	selectPostgres          = `SELECT id, type, level, timestamp, user_id, object_id, content_type, data FROM audit_logs WHERE id = $1;`
	selectManyPostgres      = `SELECT id, type, level, timestamp, user_id, object_id, content_type, data FROM audit_logs ORDER BY timestamp DESC LIMIT $1 OFFSET $2;`
	selectTypedPostgres     = `SELECT id, type, level, timestamp, user_id, object_id, content_type, data FROM audit_logs WHERE type = $1 ORDER BY timestamp DESC LIMIT $2 OFFSET $3;`
	selectForUserPostgres   = `SELECT id, type, level, timestamp, user_id, object_id, content_type, data FROM audit_logs WHERE user_id = $1 ORDER BY timestamp DESC LIMIT $2 OFFSET $3;`
	selectForObjectPostgres = `SELECT id, type, level, timestamp, user_id, object_id, content_type, data FROM audit_logs WHERE object_id = $1 ORDER BY timestamp DESC LIMIT $2 OFFSET $3;`
	selectCountPostgres     = `SELECT COUNT(id) FROM audit_logs;`
)

type postgresStorageBackend struct {
	db *sql.DB
}

func NewPostgresStorageBackend(db *sql.DB) backend.StorageBackend {
	return &postgresStorageBackend{db: db}
}

func (s *postgresStorageBackend) WithTx(tx *sql.Tx) backend.StorageBackend {
	return s
}

func (s *postgresStorageBackend) Close() error {
	return nil
}

func (s *postgresStorageBackend) Store(logEntry backend.LogEntry) (uuid.UUID, error) {
	id, typeStr, level, timestamp, userID, objectID, contentType, data := backend.SerializeRow(logEntry)
	_, err := s.db.Exec(insertPostgres, id, typeStr, level, timestamp, string(userID), string(objectID), contentType, string(data))
	return id, err
}

func (s *postgresStorageBackend) StoreMany(logEntries []backend.LogEntry) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(logEntries))
	for _, entry := range logEntries {
		id, err := s.Store(entry)
		if err != nil {
			return ids, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *postgresStorageBackend) Retrieve(id uuid.UUID) (backend.LogEntry, error) {
	row := s.db.QueryRow(selectPostgres, id)
	return backend.ScanRow(row)
}

func (s *postgresStorageBackend) RetrieveForUser(userID interface{}, amount, offset int) ([]backend.LogEntry, error) {
	b := new(bytes.Buffer)
	if err := json.NewEncoder(b).Encode(userID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(selectForUserPostgres, b.String(), amount, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRows(rows)
}

func (s *postgresStorageBackend) RetrieveForObject(objectID interface{}, amount, offset int) ([]backend.LogEntry, error) {
	b := new(bytes.Buffer)
	if err := json.NewEncoder(b).Encode(objectID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(selectForObjectPostgres, b.String(), amount, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRows(rows)
}

func (s *postgresStorageBackend) RetrieveMany(amount, offset int) ([]backend.LogEntry, error) {
	rows, err := s.db.Query(selectManyPostgres, amount, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows)
}

func (s *postgresStorageBackend) RetrieveTyped(logType string, amount, offset int) ([]backend.LogEntry, error) {
	rows, err := s.db.Query(selectTypedPostgres, logType, amount, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows)
}

func (s *postgresStorageBackend) EntryFilter(filters []backend.AuditLogFilter, amount, offset int) ([]backend.LogEntry, error) {
	query := new(strings.Builder)
	args, err := makeWhereQueryPostgres(query, filters)
	if err != nil {
		return nil, err
	}
	args = append(args, amount, offset)
	queryStr := fmt.Sprintf(`SELECT id, type, level, timestamp, user_id, object_id, content_type, data FROM audit_logs WHERE %s ORDER BY timestamp DESC LIMIT $%d OFFSET $%d;`, query.String(), len(args)-1, len(args))
	rows, err := s.db.Query(queryStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows)
}

func (s *postgresStorageBackend) CountFilter(filters []backend.AuditLogFilter) (int, error) {
	query := new(strings.Builder)
	args, err := makeWhereQueryPostgres(query, filters)
	if err != nil {
		return 0, err
	}
	row := s.db.QueryRow(fmt.Sprintf("SELECT COUNT(id) FROM audit_logs WHERE %s;", query), args...)
	var count int
	err = row.Scan(&count)
	return count, err
}

func (s *postgresStorageBackend) Count() (int, error) {
	row := s.db.QueryRow(selectCountPostgres)
	var count int
	err := row.Scan(&count)
	return count, err
}

func scanRows(rows *sql.Rows) ([]backend.LogEntry, error) {
	var entries []backend.LogEntry
	for rows.Next() {
		entry, err := backend.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func makeWhereQueryPostgres(query *strings.Builder, filters []backend.AuditLogFilter) ([]interface{}, error) {
	var args []interface{}
	paramIndex := 1

	for i, filter := range filters {
		switch filter.Name() {
		case backend.AuditLogFilterID:
			value := filter.Value()
			if len(value) > 1 {
				inQ := make([]string, len(value))
				for j, v := range value {
					inQ[j] = fmt.Sprintf("$%d", paramIndex)
					args = append(args, v)
					paramIndex++
				}
				fmt.Fprintf(query, "id IN (%s)", strings.Join(inQ, ","))
			} else {
				fmt.Fprintf(query, "id = $%d", paramIndex)
				args = append(args, value[0])
				paramIndex++
			}

		case backend.AuditLogFilterType:
			value := filter.Value()
			if len(value) > 1 {
				inQ := make([]string, len(value))
				for j, v := range value {
					inQ[j] = fmt.Sprintf("$%d", paramIndex)
					args = append(args, v)
					paramIndex++
				}
				fmt.Fprintf(query, "type IN (%s)", strings.Join(inQ, ","))
			} else {
				fmt.Fprintf(query, "type = $%d", paramIndex)
				args = append(args, value[0])
				paramIndex++
			}

		case backend.AuditLogFilterLevel_EQ:
			fmt.Fprintf(query, "level = $%d", paramIndex)
			args = append(args, filter.Value()[0])
			paramIndex++

		case backend.AuditLogFilterLevel_GT:
			fmt.Fprintf(query, "level > $%d", paramIndex)
			args = append(args, filter.Value()[0])
			paramIndex++

		case backend.AuditLogFilterLevel_LT:
			fmt.Fprintf(query, "level < $%d", paramIndex)
			args = append(args, filter.Value()[0])
			paramIndex++

		case backend.AuditLogFilterTimestamp_EQ:
			fmt.Fprintf(query, "timestamp = $%d", paramIndex)
			args = append(args, filter.Value()[0])
			paramIndex++

		case backend.AuditLogFilterTimestamp_GT:
			fmt.Fprintf(query, "timestamp > $%d", paramIndex)
			args = append(args, filter.Value()[0])
			paramIndex++

		case backend.AuditLogFilterTimestamp_LT:
			fmt.Fprintf(query, "timestamp < $%d", paramIndex)
			args = append(args, filter.Value()[0])
			paramIndex++

		case backend.AuditLogFilterUserID:
			value := filter.Value()
			if len(value) > 1 {
				inQ := make([]string, len(value))
				for j, v := range value {
					b := new(bytes.Buffer)
					if err := json.NewEncoder(b).Encode(v); err != nil {
						return nil, err
					}
					inQ[j] = fmt.Sprintf("$%d", paramIndex)
					args = append(args, b.String())
					paramIndex++
				}
				fmt.Fprintf(query, "user_id IN (%s)", strings.Join(inQ, ","))
			} else {
				b := new(bytes.Buffer)
				if err := json.NewEncoder(b).Encode(value[0]); err != nil {
					return nil, err
				}
				fmt.Fprintf(query, "user_id = $%d", paramIndex)
				args = append(args, b.String())
				paramIndex++
			}

		case backend.AuditLogFilterObjectID:
			value := filter.Value()
			if len(value) > 1 {
				inQ := make([]string, len(value))
				for j, v := range value {
					b := new(bytes.Buffer)
					if err := json.NewEncoder(b).Encode(v); err != nil {
						return nil, err
					}
					inQ[j] = fmt.Sprintf("$%d", paramIndex)
					args = append(args, b.String())
					paramIndex++
				}
				fmt.Fprintf(query, "object_id IN (%s)", strings.Join(inQ, ","))
			} else {
				b := new(bytes.Buffer)
				if err := json.NewEncoder(b).Encode(value[0]); err != nil {
					return nil, err
				}
				fmt.Fprintf(query, "object_id = $%d", paramIndex)
				args = append(args, b.String())
				paramIndex++
			}

		case backend.AuditLogFilterContentType:
			fmt.Fprintf(query, "content_type = $%d", paramIndex)
			args = append(args, filter.Value()[0])
			paramIndex++

		case backend.AuditLogFilterData:
			v := filter.Value()[0]
			fmt.Fprintf(query, "data = $%d", paramIndex)
			switch v.(type) {
			case string:
				args = append(args, v)
			default:
				var b bytes.Buffer
				if err := json.NewEncoder(&b).Encode(v); err != nil {
					return nil, err
				}
				args = append(args, b.Bytes())
			}
			paramIndex++
		}

		if i < len(filters)-1 {
			query.WriteString(" AND ")
		}
	}
	return args, nil
}
