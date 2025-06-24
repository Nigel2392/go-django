// Package mysql provides a MySQL implementation of the migrator.SchemaEditor interface.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/go-sql-driver/mysql"
)

var _ migrator.SchemaEditor = &MySQLSchemaEditor{}

func init() {
	migrator.RegisterSchemaEditor(&drivers.DriverMySQL{}, func() (migrator.SchemaEditor, error) {
		var db, ok = django.ConfigGetOK[drivers.Database](
			django.Global.Settings,
			django.APPVAR_DATABASE,
		)
		if !ok {
			return nil, fmt.Errorf("migrator: mysql: no database connection found")
		}
		return NewMySQLSchemaEditor(db), nil
	})

	migrator.RegisterSchemaEditor(&drivers.DriverMariaDB{}, func() (migrator.SchemaEditor, error) {
		var db, ok = django.ConfigGetOK[drivers.Database](
			django.Global.Settings,
			django.APPVAR_DATABASE,
		)
		if !ok {
			return nil, fmt.Errorf("migrator: mariadb: no database connection found")
		}
		return NewMySQLSchemaEditor(db), nil
	})
}

const (
	createTableMigrations = `CREATE TABLE IF NOT EXISTS migrations (
		id INT AUTO_INCREMENT PRIMARY KEY,
		app_name VARCHAR(255) NOT NULL,
		model_name VARCHAR(255) NOT NULL,
		migration_name VARCHAR(255) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE KEY unique_migration (app_name, model_name, migration_name)
	);`
	insertTableMigrations = `INSERT INTO migrations (app_name, model_name, migration_name) VALUES (?, ?, ?);`
	deleteTableMigrations = `DELETE FROM migrations WHERE app_name = ? AND model_name = ? AND migration_name = ?;`
	selectTableMigrations = `SELECT COUNT(*) FROM migrations WHERE app_name = ? AND model_name = ? AND migration_name = ? LIMIT 1;`
)

type MySQLSchemaEditor struct {
	db            drivers.Database
	tablesCreated bool
}

func NewMySQLSchemaEditor(db drivers.Database) *MySQLSchemaEditor {
	return &MySQLSchemaEditor{db: db}
}

func (m *MySQLSchemaEditor) Setup() error {
	if m.tablesCreated {
		return nil
	}
	_, err := m.db.ExecContext(context.Background(), createTableMigrations)
	if err != nil {
		return err
	}
	m.tablesCreated = true
	return nil
}

func (m *MySQLSchemaEditor) StoreMigration(appName, modelName, migrationName string) error {
	_, err := m.Execute(context.Background(), insertTableMigrations, appName, modelName, migrationName)
	return err
}

func (m *MySQLSchemaEditor) HasMigration(appName, modelName, migrationName string) (bool, error) {
	var count int
	err := m.queryRow(context.Background(), selectTableMigrations, appName, modelName, migrationName).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (m *MySQLSchemaEditor) RemoveMigration(appName, modelName, migrationName string) error {
	_, err := m.Execute(context.Background(), deleteTableMigrations, appName, modelName, migrationName)
	return err
}

func (m *MySQLSchemaEditor) queryRow(ctx context.Context, query string, args ...any) drivers.SQLRow {
	// logger.Debugf("MySQLSchemaEditor.QueryRowContext:\n%s", query)
	return m.db.QueryRowContext(ctx, query, args...)
}

func (m *MySQLSchemaEditor) Execute(ctx context.Context, query string, args ...any) (sql.Result, error) {
	logger.Debugf("MySQLSchemaEditor.ExecContext:\n%s", query)
	result, err := m.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (m *MySQLSchemaEditor) CreateTable(table migrator.Table, ifNotExists bool) error {
	var w strings.Builder
	w.WriteString("CREATE TABLE ")
	if ifNotExists {
		w.WriteString("IF NOT EXISTS ")
	}
	w.WriteString("`")
	w.WriteString(table.TableName())
	w.WriteString("` (")

	var written bool
	for _, col := range table.Columns() {
		if !col.UseInDB {
			continue
		}
		if written {
			w.WriteString(",\n")
		}
		w.WriteString("  ")
		WriteColumn(&w, *col)
		written = true
	}
	w.WriteString("\n);")
	_, err := m.Execute(context.Background(), w.String())
	return err
}

func (m *MySQLSchemaEditor) DropTable(table migrator.Table, ifExists bool) error {
	var w strings.Builder
	w.WriteString("DROP TABLE ")
	if ifExists {
		w.WriteString("IF EXISTS ")
	}
	w.WriteString("`")
	w.WriteString(table.TableName())
	w.WriteString("`;")
	_, err := m.Execute(context.Background(), w.String())
	return err
}

func (m *MySQLSchemaEditor) RenameTable(table migrator.Table, newName string) error {
	query := fmt.Sprintf("RENAME TABLE `%s` TO `%s`;", table.TableName(), newName)
	_, err := m.Execute(context.Background(), query)
	return err
}

func (m *MySQLSchemaEditor) AddIndex(table migrator.Table, index migrator.Index, ifNotExists bool) error {

	if ifNotExists {
		// MySQL does not support IF NOT EXISTS for CREATE INDEX, so we need to check manually.
		var exists bool
		err := m.queryRow(context.Background(),
			"SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?",
			table.TableName(), index.Name(),
		).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check if index exists: %w", err)
		}
		if exists {
			// Index already exists, nothing to do
			logger.Debugf("Index `%s` on table `%s` already exists, skipping creation.", index.Name(), table.TableName())
			return nil
		}
	}

	var w strings.Builder
	if index.Unique {
		w.WriteString("CREATE UNIQUE INDEX")
	} else {
		w.WriteString("CREATE INDEX")
	}
	w.WriteString(" `")
	w.WriteString(index.Name())
	w.WriteString("` ON `")
	w.WriteString(table.TableName())
	w.WriteString("` (")
	for i, col := range index.Columns() {
		if i > 0 {
			w.WriteString(", ")
		}
		w.WriteString("`")
		w.WriteString(col.Column)
		w.WriteString("`")
		var fieldType = col.FieldType()
		switch {
		case fieldType.Kind() == reflect.String:

			if col.MaxLength > 0 {
				w.WriteString("(")
				w.WriteString(fmt.Sprintf("%d", col.MaxLength))
				w.WriteString(")")
			} else {
				// MySQL does not support VARCHAR without length, so we assume a default length
				w.WriteString("(255)")
			}
		}
	}
	w.WriteString(");")
	_, err := m.Execute(context.Background(), w.String())
	return err
}

func (m *MySQLSchemaEditor) DropIndex(table migrator.Table, index migrator.Index, ifExists bool) error {
	// MySQL does not support IF EXISTS for DROP INDEX, so we just drop it
	// without checking if it exists.
	// If you want to check before dropping, you would need to query the information_schema.
	// This is a workaround, as MySQL does not support IF EXISTS for DROP INDEX.
	query := fmt.Sprintf("DROP INDEX `%s` ON `%s`;", index.Name(), table.TableName())
	_, err := m.Execute(context.Background(), query)
	return err
}

func (m *MySQLSchemaEditor) RenameIndex(table migrator.Table, oldName, newName string) error {
	// MySQL does not support RENAME INDEX directly, workaround required
	return fmt.Errorf("mysql does not support RENAME INDEX directly, please drop and recreate")
}

func (m *MySQLSchemaEditor) AddField(table migrator.Table, col migrator.Column) error {
	var w strings.Builder
	w.WriteString("ALTER TABLE `")
	w.WriteString(table.TableName())
	w.WriteString("` ADD COLUMN ")
	WriteColumn(&w, col)
	_, err := m.Execute(context.Background(), w.String())
	return err
}

func (m *MySQLSchemaEditor) RemoveField(table migrator.Table, col migrator.Column) error {
	query := fmt.Sprintf("ALTER TABLE `%s` DROP COLUMN `%s`;", table.TableName(), col.Column)
	_, err := m.Execute(context.Background(), query)
	return err
}

func (m *MySQLSchemaEditor) AlterField(table migrator.Table, oldCol, newCol migrator.Column) error {
	query := fmt.Sprintf("ALTER TABLE `%s` MODIFY COLUMN ", table.TableName())
	var w strings.Builder
	w.WriteString(query)
	WriteColumn(&w, newCol)
	_, err := m.Execute(context.Background(), w.String())
	return err
}

func WriteColumn(w *strings.Builder, col migrator.Column) {
	w.WriteString("`")
	w.WriteString(col.Column)
	w.WriteString("` ")
	w.WriteString(migrator.GetFieldType(
		&mysql.MySQLDriver{}, &col,
	))

	if !col.Nullable {
		w.WriteString(" NOT NULL")
	}

	if col.Primary {
		w.WriteString(" PRIMARY KEY")
	}

	if col.Auto {
		w.WriteString(" AUTO_INCREMENT")
	}

	if col.Unique {
		w.WriteString(" UNIQUE")
	}

	if col.HasDefault() {
		w.WriteString(" DEFAULT ")

		switch v := col.Default.(type) {
		case string:
			w.WriteString("'")
			w.WriteString(v)
			w.WriteString("'")
		case int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64:
			w.WriteString(fmt.Sprintf("%d", v))
		case float64, float32:
			w.WriteString(fmt.Sprintf("%f", v))
		case bool:
			if v {
				w.WriteString("1")
			} else {
				w.WriteString("0")
			}
		case time.Time:
			if v.IsZero() {
				w.WriteString("CURRENT_TIMESTAMP")
			} else {
				w.WriteString("'")
				w.WriteString(v.Format("2006-01-02 15:04:05"))
				w.WriteString("'")
			}
		default:
			panic(fmt.Errorf(
				"unsupported default value type %T (%v)", v, v,
			))
		}
	}
	if col.Rel != nil {
		relField := col.Rel.Field()
		if relField == nil {
			relField = col.Rel.Model().FieldDefs().Primary()
		}
		w.WriteString(" REFERENCES `")
		w.WriteString(col.Rel.Model().FieldDefs().TableName())
		w.WriteString("`(`")
		w.WriteString(relField.ColumnName())
		w.WriteString("`)")
		if col.Rel.OnDelete != 0 {
			w.WriteString(" ON DELETE ")
			w.WriteString(col.Rel.OnDelete.String())
		}
		if col.Rel.OnUpdate != 0 {
			w.WriteString(" ON UPDATE ")
			w.WriteString(col.Rel.OnUpdate.String())
		}
	}
}
