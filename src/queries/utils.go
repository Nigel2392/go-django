package queries

import (
	"database/sql"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/go-sql-driver/mysql"
	pg_stdlib "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/mattn/go-sqlite3"
)

func sqlxDriverName(db *sql.DB) string {
	switch db.Driver().(type) {
	case *mysql.MySQLDriver:
		return "mysql"
	case *sqlite3.SQLiteDriver:
		return "sqlite3"
	case *pg_stdlib.Driver:
		return "postgres"
	default:
		return ""
	}
}

type queryInfo struct {
	db          *sql.DB
	dbx         *sqlx.DB
	sqlxDriver  string
	tableName   string
	definitions attrs.Definitions
	fields      []attrs.Field
	// fields_map  map[string]attrs.Field
}

func getQueryInfo[T attrs.Definer](obj T) (*queryInfo, error) {
	var db = django.ConfigGet[*sql.DB](
		django.Global.Settings,
		django.APPVAR_DATABASE,
	)
	if db == nil {
		return nil, ErrNoDatabase
	}

	var sqlxDriver = sqlxDriverName(db)
	if sqlxDriver == "" {
		return nil, ErrUnknownDriver
	}

	var fieldDefs = obj.FieldDefs()
	var tableName = fieldDefs.TableName()
	if tableName == "" {
		return nil, ErrNoTableName
	}

	// var fields_map = make(map[string]attrs.Field)
	var fields = fieldDefs.Fields()
	// for _, field := range fieldDefs.Fields() {
	// fields_map[field.ColumnName()] = field
	// }

	var dbx = sqlx.NewDb(db, sqlxDriver)
	return &queryInfo{
		db:          db,
		dbx:         dbx,
		sqlxDriver:  sqlxDriver,
		definitions: fieldDefs,
		tableName:   tableName,
		fields:      fields,
		// fields_map:  fields_map,
	}, nil
}
