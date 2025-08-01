package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
)

var _ migrator.SchemaEditor = &PostgresSchemaEditor{}

func init() {
	migrator.RegisterSchemaEditor(&drivers.DriverPostgres{}, func() (migrator.SchemaEditor, error) {
		var db, ok = django.ConfigGetOK[drivers.Database](
			django.Global.Settings,
			django.APPVAR_DATABASE,
		)
		if !ok {
			return nil, fmt.Errorf("migrator: mysql: no database connection found")
		}
		return NewPostgresSchemaEditor(db), nil
	})
}

const (
	createTableMigrations = `CREATE TABLE IF NOT EXISTS migrations (
		id SERIAL PRIMARY KEY,
		app_name TEXT NOT NULL,
		model_name TEXT NOT NULL,
		migration_name TEXT NOT NULL,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(app_name, model_name, migration_name)
	);`
	insertTableMigrations = `INSERT INTO migrations (app_name, model_name, migration_name) VALUES ($1, $2, $3);`
	deleteTableMigrations = `DELETE FROM migrations WHERE app_name = $1 AND model_name = $2 AND migration_name = $3;`
	selectTableMigrations = `SELECT COUNT(*) FROM migrations WHERE app_name = $1 AND model_name = $2 AND migration_name = $3;`
)

type PostgresSchemaEditor struct {
	db drivers.Database
}

func NewPostgresSchemaEditor(db drivers.Database) *PostgresSchemaEditor {
	return &PostgresSchemaEditor{db: db}
}

func (m *PostgresSchemaEditor) Setup(ctx context.Context) error {
	_, err := m.Execute(ctx, createTableMigrations)
	return err
}

func (m *PostgresSchemaEditor) StartTransaction(ctx context.Context) (drivers.Transaction, error) {
	return m.db.Begin(ctx)
}

func (m *PostgresSchemaEditor) StoreMigration(ctx context.Context, appName string, modelName string, migrationName string) error {
	_, err := m.Execute(ctx, insertTableMigrations, appName, modelName, migrationName)
	return err
}

func (m *PostgresSchemaEditor) HasMigration(ctx context.Context, appName string, modelName string, migrationName string) (bool, error) {
	var count int
	err := m.QueryRow(ctx, selectTableMigrations, appName, modelName, migrationName).Scan(&count)
	return count > 0, err
}

func (m *PostgresSchemaEditor) RemoveMigration(ctx context.Context, appName string, modelName string, migrationName string) error {
	_, err := m.Execute(ctx, deleteTableMigrations, appName, modelName, migrationName)
	return err
}

func (m *PostgresSchemaEditor) QueryRow(ctx context.Context, query string, args ...any) drivers.SQLRow {
	return m.db.QueryRowContext(ctx, query, args...)
}

func (m *PostgresSchemaEditor) Execute(ctx context.Context, query string, args ...any) (sql.Result, error) {
	// logger.Debugf("PostgresSchemaEditor.ExecContext:\n%s", query)
	return m.db.ExecContext(ctx, query, args...)
}

func (m *PostgresSchemaEditor) CreateTable(ctx context.Context, table migrator.Table, ifNotExists bool) error {
	var w strings.Builder
	w.WriteString(`CREATE TABLE `)
	if ifNotExists {
		w.WriteString(`IF NOT EXISTS `)
	}
	w.WriteString(`"`)
	w.WriteString(table.TableName())
	w.WriteString(`" (`)

	var written bool
	var cols = table.Columns()
	for _, col := range cols {
		if !col.UseInDB {
			continue
		}
		if written {
			w.WriteString(", ")
		}
		m.WriteColumn(&w, *col)
		written = true
	}

	w.WriteString(");")

	_, err := m.Execute(ctx, w.String())
	return err
}

func (m *PostgresSchemaEditor) DropTable(ctx context.Context, table migrator.Table, ifExists bool) error {
	var w strings.Builder
	w.WriteString(`DROP TABLE `)
	if ifExists {
		w.WriteString(`IF EXISTS `)
	}
	w.WriteString(`"`)
	w.WriteString(table.TableName())
	w.WriteString(`" CASCADE;`)
	_, err := m.Execute(ctx, w.String())
	return err
}

func (m *PostgresSchemaEditor) RenameTable(ctx context.Context, table migrator.Table, newName string) error {
	var w strings.Builder
	w.WriteString(`ALTER TABLE "`)
	w.WriteString(table.TableName())
	w.WriteString(`" RENAME TO "`)
	w.WriteString(newName)
	w.WriteString(`";`)
	_, err := m.Execute(ctx, w.String())
	return err
}

func (m *PostgresSchemaEditor) AddIndex(ctx context.Context, table migrator.Table, index migrator.Index, ifNotExists bool) error {
	var w strings.Builder
	if index.Unique {
		w.WriteString(`CREATE UNIQUE INDEX `)
	} else {
		w.WriteString(`CREATE INDEX `)
	}
	if ifNotExists {
		w.WriteString(`IF NOT EXISTS `)
	}
	w.WriteString(`"`)
	w.WriteString(index.Name())
	w.WriteString(`" ON "`)
	w.WriteString(table.TableName())
	w.WriteString(`" (`)
	for i, col := range index.Columns() {
		if i > 0 {
			w.WriteString(", ")
		}
		w.WriteString(`"`)
		w.WriteString(col.Column)
		w.WriteString(`"`)
	}
	w.WriteString(");")

	_, err := m.Execute(ctx, w.String())
	return err
}

func (m *PostgresSchemaEditor) DropIndex(ctx context.Context, table migrator.Table, index migrator.Index, ifExists bool) error {
	var w strings.Builder
	w.WriteString(`DROP INDEX `)
	if ifExists {
		w.WriteString(`IF EXISTS `)
	}
	w.WriteString(`"`)
	w.WriteString(index.Name())
	w.WriteString(`";`)
	_, err := m.Execute(ctx, w.String())
	return err
}

func (m *PostgresSchemaEditor) RenameIndex(ctx context.Context, table migrator.Table, oldName string, newName string) error {
	var w strings.Builder
	w.WriteString(`ALTER INDEX "`)
	w.WriteString(oldName)
	w.WriteString(`" RENAME TO "`)
	w.WriteString(newName)
	w.WriteString(`";`)
	_, err := m.Execute(ctx, w.String())
	return err
}

func (m *PostgresSchemaEditor) AddField(ctx context.Context, table migrator.Table, col migrator.Column) error {
	var w strings.Builder
	w.WriteString(`ALTER TABLE "`)
	w.WriteString(table.TableName())
	w.WriteString(`" ADD COLUMN `)
	m.WriteColumn(&w, col)
	w.WriteString(";")
	_, err := m.Execute(ctx, w.String())
	return err
}

func (m *PostgresSchemaEditor) RemoveField(ctx context.Context, table migrator.Table, col migrator.Column) error {
	var w strings.Builder
	w.WriteString(`ALTER TABLE "`)
	w.WriteString(table.TableName())
	w.WriteString(`" DROP COLUMN IF EXISTS "`)
	w.WriteString(col.Field.ColumnName())
	w.WriteString(`" CASCADE;`)
	_, err := m.Execute(ctx, w.String())
	return err
}

func (m *PostgresSchemaEditor) AlterField(ctx context.Context, table migrator.Table, oldCol migrator.Column, newCol migrator.Column) error {
	var (
		w         strings.Builder
		tableName = table.TableName()
		colName   = oldCol.Field.ColumnName()
	)

	w.WriteString(`ALTER TABLE "`)
	w.WriteString(tableName)
	w.WriteString(`"`)

	// Alter column type

	var (
		aTyp = migrator.GetFieldType(&drivers.DriverPostgres{}, &oldCol)
		bTyp = migrator.GetFieldType(&drivers.DriverPostgres{}, &newCol)
	)

	if aTyp != bTyp {
		w.WriteString(` ALTER COLUMN "`)
		w.WriteString(colName)
		w.WriteString(`" TYPE `)
		w.WriteString(bTyp)
		w.WriteString(`,`)
	}

	// Alter NULL / NOT NULL
	if oldCol.Nullable != newCol.Nullable {
		w.WriteString(` ALTER COLUMN "`)
		w.WriteString(colName)
		if newCol.Nullable {
			w.WriteString(`" DROP NOT NULL,`)
		} else {
			w.WriteString(`" SET NOT NULL,`)
		}
	}

	// Alter default
	var oldDefault = oldCol.Default
	var newDefault = newCol.Default
	if !migrator.EqualDefaultValue(oldDefault, newDefault) {
		w.WriteString(` ALTER COLUMN "`)
		w.WriteString(colName)
		if newDefault == nil {
			w.WriteString(`" DROP DEFAULT,`)
		} else {
			w.WriteString(`" SET DEFAULT `)
			var rv = reflect.ValueOf(newDefault)
			if rv.Kind() == reflect.Ptr {
				if !rv.IsValid() || rv.IsNil() {
					w.WriteString(`NULL`)
					w.WriteString(`,`)
					return nil
				}
				rv = rv.Elem()
			}

			switch rv.Kind() {
			case reflect.String:
				w.WriteString(fmt.Sprintf("'%s'", rv.String()))
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				w.WriteString(fmt.Sprintf("%d", rv.Int()))
			case reflect.Float32, reflect.Float64:
				w.WriteString(fmt.Sprintf("%f", rv.Float()))
			case reflect.Bool:
				if rv.Bool() {
					w.WriteString("TRUE")
				} else {
					w.WriteString("FALSE")
				}
			case reflect.Slice, reflect.Array:
				if rv.IsNil() {
					w.WriteString(`NULL`)
					w.WriteString(`,`)
					return nil
				}
				if rv.Type().Elem().Kind() == reflect.Uint8 {
					w.WriteString(`'`)
					w.Write(rv.Bytes())
					w.WriteString(`'`)
				} else {
					return fmt.Errorf("unsupported default type %T", rv.Interface())
				}
			default:
				return fmt.Errorf("unsupported default type %T", newDefault)
			}
			w.WriteString(`,`)
		}
	}

	// Trim trailing comma
	sql := strings.TrimSuffix(w.String(), ",")

	// Execute ALTER TABLE ... for collected changes
	if _, err := m.Execute(ctx, sql); err != nil {
		return fmt.Errorf("alter field failed: %w\nquery: %s", err, sql)
	}

	return nil
}

func (m *PostgresSchemaEditor) WriteColumn(w *strings.Builder, col migrator.Column) {
	w.WriteString(`"`)
	w.WriteString(col.Field.ColumnName())
	w.WriteString(`" `)
	w.WriteString(migrator.GetFieldType(&drivers.DriverPostgres{}, &col))

	if col.Primary {
		w.WriteString(" PRIMARY KEY")
	}

	if !col.Nullable {
		w.WriteString(" NOT NULL")
	}

	if col.Unique {
		w.WriteString(" UNIQUE")
	}

	if col.HasDefault() {

		if valuer, ok := col.Default.(driver.Valuer); ok {
			// If the default value is a driver.Valuer, we need to call it to get the actual value.
			val, err := valuer.Value()
			if err != nil {
				panic(fmt.Errorf("failed to get value from driver.Valuer: %w", err))
			}
			col.Default = val
		}

		if col.Default == nil && !col.Nullable {
			goto checkRels
		}

		w.WriteString(" DEFAULT ")
		switch v := col.Default.(type) {
		case string:
			w.WriteString(fmt.Sprintf("'%s'", v))
		case int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64:
			w.WriteString(fmt.Sprintf("%d", v))
		case float32, float64:
			w.WriteString(fmt.Sprintf("%f", v))
		case bool:
			if v {
				w.WriteString("TRUE")
			} else {
				w.WriteString("FALSE")
			}
		case time.Time:
			if v.IsZero() {
				w.WriteString("CURRENT_TIMESTAMP")
			} else {
				w.WriteString("'")
				w.WriteString(v.Format("2006-01-02 15:04:05"))
				w.WriteString("'")
			}
		case nil:
			w.WriteString("NULL")
		default:
			panic(fmt.Errorf("unsupported default type: %T", v))
		}
	}

checkRels:
	if col.Rel != nil {
		// handle foreign keys
		var relField = col.Rel.Field()
		if relField == nil {
			relField = col.Rel.Model().FieldDefs().Primary()
		}
		if relField == nil {
			panic(fmt.Errorf("missing foreign key target for column %q", col.Name))
		}
		w.WriteString(fmt.Sprintf(` REFERENCES "%s"("%s")`,
			col.Rel.Model().FieldDefs().TableName(),
			relField.ColumnName(),
		))
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
