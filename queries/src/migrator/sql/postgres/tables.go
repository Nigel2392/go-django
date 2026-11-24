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
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr/builder"
	"github.com/Nigel2392/go-django/queries/src/expr/constraints"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/utils/djangoapps"
)

var _ migrator.SchemaEditor = &PostgresSchemaEditor{}

func init() {
	migrator.RegisterSchemaEditor(&drivers.DriverPostgres{}, func() (migrator.SchemaEditor, error) {
		var db, ok = djangoapps.CONFIG.GetOK[drivers.Database](
			django.APPVAR_DATABASE,
		)
		if !ok {
			return nil, fmt.Errorf("migrator: postgres: no database connection found")
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
	return migrator.DbFromContext(ctx, m.db).QueryRowContext(ctx, query, args...)
}

func (m *PostgresSchemaEditor) Execute(ctx context.Context, query string, args ...any) (sql.Result, error) {
	// logger.Debugf("PostgresSchemaEditor.ExecContext:\n%s", query)
	return migrator.DbFromContext(ctx, m.db).ExecContext(ctx, query, args...)
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
		includeRel := true
		for _, c := range table.Constraints() {
			if c != nil && c.Type() == constraints.C_FOREIGN_KEY {
				if fk, ok := c.(interface{ ForeignKeyField() string }); ok {
					if name := fk.ForeignKeyField(); name != "" && (name == col.Name || name == col.Column) {
						includeRel = false
						break
					}
				}
			}
		}
		m.writeColumn(&w, *col, includeRel)
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
	var colName = col.Column
	if colName == "" && col.Field != nil {
		colName = col.Field.ColumnName()
	}
	var w strings.Builder
	w.WriteString(`ALTER TABLE "`)
	w.WriteString(table.TableName())
	w.WriteString(`" DROP COLUMN IF EXISTS "`)
	w.WriteString(colName)
	w.WriteString(`" CASCADE;`)
	_, err := m.Execute(ctx, w.String())
	return err
}

func pgDefaultSQL(v any) (string, error) {
	if valuer, ok := v.(driver.Valuer); ok {
		val, err := valuer.Value()
		if err != nil {
			return "", err
		}
		v = val
	}
	if v == nil {
		return "NULL", nil
	}
	var rv = reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return "NULL", nil
		}
		rv = rv.Elem()
	}
	if t, ok := rv.Interface().(time.Time); ok {
		if t.IsZero() {
			return "CURRENT_TIMESTAMP", nil
		}
		return "'" + t.Format("2006-01-02 15:04:05") + "'", nil
	}
	switch rv.Kind() {
	case reflect.String:
		return "'" + strings.ReplaceAll(rv.String(), "'", "''") + "'", nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", rv.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", rv.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%f", rv.Float()), nil
	case reflect.Bool:
		if rv.Bool() {
			return "TRUE", nil
		}
		return "FALSE", nil
	case reflect.Slice, reflect.Array:
		if rv.Kind() == reflect.Slice && rv.IsNil() {
			return "NULL", nil
		}
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			return "'" + strings.ReplaceAll(string(rv.Bytes()), "'", "''") + "'", nil
		}
	}
	return "", fmt.Errorf("unsupported default type %T", v)
}

func (m *PostgresSchemaEditor) AlterField(ctx context.Context, table migrator.Table, oldCol migrator.Column, newCol migrator.Column) error {
	var (
		tableName = table.TableName()
		colName   = oldCol.Column
		clauses   []string
	)

	if colName == "" && oldCol.Field != nil {
		colName = oldCol.Field.ColumnName()
	}

	if newCol.Column != "" && newCol.Column != colName {
		var rename = fmt.Sprintf(`ALTER TABLE "%s" RENAME COLUMN "%s" TO "%s";`, tableName, colName, newCol.Column)
		if _, err := m.Execute(ctx, rename); err != nil {
			return fmt.Errorf("alter field rename failed: %w\nquery: %s", err, rename)
		}
		colName = newCol.Column
	}

	var (
		aTyp = migrator.GetFieldType(&drivers.DriverPostgres{}, &oldCol)
		bTyp = migrator.GetFieldType(&drivers.DriverPostgres{}, &newCol)
	)

	if aTyp != bTyp {
		clauses = append(clauses, fmt.Sprintf(`ALTER COLUMN "%s" TYPE %s`, colName, bTyp))
	}

	if oldCol.Nullable != newCol.Nullable {
		if newCol.Nullable {
			clauses = append(clauses, fmt.Sprintf(`ALTER COLUMN "%s" DROP NOT NULL`, colName))
		} else {
			clauses = append(clauses, fmt.Sprintf(`ALTER COLUMN "%s" SET NOT NULL`, colName))
		}
	}

	if !migrator.EqualDefaultValue(oldCol.Default.V, newCol.Default.V) {
		if newCol.Default.V == nil {
			clauses = append(clauses, fmt.Sprintf(`ALTER COLUMN "%s" DROP DEFAULT`, colName))
		} else {
			var dflt, err = pgDefaultSQL(newCol.Default.V)
			if err != nil {
				return err
			}
			clauses = append(clauses, fmt.Sprintf(`ALTER COLUMN "%s" SET DEFAULT %s`, colName, dflt))
		}
	}

	if len(clauses) == 0 {
		return nil
	}

	var sql = fmt.Sprintf(`ALTER TABLE "%s" %s`, tableName, strings.Join(clauses, ", "))
	if _, err := m.Execute(ctx, sql); err != nil {
		return fmt.Errorf("alter field failed: %w\nquery: %s", err, sql)
	}

	return nil
}

func (m *PostgresSchemaEditor) AddConstraint(ctx context.Context, table migrator.Table, constraint constraints.Constraint) error {
	if constraint == nil {
		return errors.NilPointer.Wrap("constraint is nil")
	}
	var sb builder.BaseBuilder
	constraint.SQL(&sb)
	if err := sb.GetError(); err != nil {
		return err
	}
	if len(sb.Vars) > 0 {
		return fmt.Errorf("constraint %q cannot use bound parameters in DDL", constraint.ID())
	}
	sqlStr := strings.TrimSpace(sb.String())
	if sqlStr == "" {
		return fmt.Errorf("constraint %q generated empty SQL", constraint.ID())
	}
	_, err := m.Execute(ctx, fmt.Sprintf(`ALTER TABLE "%s" ADD %s;`, table.TableName(), sqlStr))
	return err
}

func (m *PostgresSchemaEditor) DropConstraint(ctx context.Context, table migrator.Table, constraint constraints.Constraint) error {
	if constraint == nil {
		return errors.NilPointer.Wrap("constraint is nil")
	}
	if constraint.ID() == "" {
		return errors.ValueError.Wrap("constraint ID cannot be empty")
	}
	_, err := m.Execute(ctx, fmt.Sprintf(`ALTER TABLE "%s" DROP CONSTRAINT IF EXISTS "%s";`, table.TableName(), constraint.ID()))
	return err
}

func (m *PostgresSchemaEditor) WriteColumn(w *strings.Builder, col migrator.Column) {
	m.writeColumn(w, col, true)
}

func (m *PostgresSchemaEditor) writeColumn(w *strings.Builder, col migrator.Column, includeRel bool) {
	var colName = col.Column
	if colName == "" && col.Field != nil {
		colName = col.Field.ColumnName()
	}
	w.WriteString(`"`)
	w.WriteString(colName)
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

		if valuer, ok := col.Default.V.(driver.Valuer); ok {
			// If the default value is a driver.Valuer, we need to call it to get the actual value.
			val, err := valuer.Value()
			if err != nil {
				panic(fmt.Errorf("failed to get value from driver.Valuer: %w", err))
			}
			col.Default.V = val
		}

		if col.Default.V == nil && !col.Nullable {
			goto checkRels
		}

		w.WriteString(" DEFAULT ")
		switch v := col.Default.V.(type) {
		case string:
			w.WriteString(fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''")))
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
	if includeRel && col.Rel != nil {
		// handle foreign keys
		var relField = col.Rel.Field()
		if relField == nil {
			relField = attrs.GetModelMeta(col.Rel.Model()).Definitions().Primary()
		}
		if relField == nil {
			panic(fmt.Errorf("missing foreign key target for column %q", col.Name))
		}
		w.WriteString(fmt.Sprintf(` REFERENCES "%s"("%s")`,
			attrs.GetModelMeta(col.Rel.Model()).Definitions().TableName(),
			relField.ColumnName(),
		))

		var (
			onDelete = col.Rel.OnDelete
			onUpdate = col.Rel.OnUpdate
		)

		if onDelete == 0 {
			if col.Nullable {
				onDelete = constraints.A_SET_NULL
			}
		}

		if onUpdate == 0 {
			if col.Nullable {
				onUpdate = constraints.A_SET_NULL
			}
		}

		w.WriteString(" ON DELETE ")
		w.WriteString(col.Rel.OnDelete.String())

		w.WriteString(" ON UPDATE ")
		w.WriteString(col.Rel.OnUpdate.String())
	}
}
