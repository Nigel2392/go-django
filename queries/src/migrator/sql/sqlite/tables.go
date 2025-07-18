package sqlite

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
	"github.com/elliotchance/orderedmap/v2"
	"github.com/mattn/go-sqlite3"
)

var _ migrator.SchemaEditor = &SQLiteSchemaEditor{}

func init() {
	migrator.RegisterSchemaEditor(&sqlite3.SQLiteDriver{}, func() (migrator.SchemaEditor, error) {
		var db, ok = django.ConfigGetOK[drivers.Database](
			django.Global.Settings,
			django.APPVAR_DATABASE,
		)
		if !ok {
			return nil, fmt.Errorf("migrator: mysql: no database connection found")
		}
		return NewSQLiteSchemaEditor(db), nil
	})
}

const (
	createTableMigrations = `CREATE TABLE IF NOT EXISTS migrations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		app_name TEXT NOT NULL,
		model_name TEXT NOT NULL,
		migration_name TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(app_name, model_name, migration_name)
	);`
	insertTableMigrations = `INSERT INTO migrations (app_name, model_name, migration_name) VALUES (?, ?, ?);`
	deleteTableMigrations = `DELETE FROM migrations WHERE app_name = ? AND model_name = ? AND migration_name = ?;`
	selectTableMigrations = `SELECT COUNT(*) FROM migrations WHERE app_name = ? AND model_name = ? AND migration_name = ? LIMIT 1;`
)

type SQLiteSchemaEditor struct {
	db            drivers.Database
	tablesCreated bool
}

func NewSQLiteSchemaEditor(db drivers.Database) *SQLiteSchemaEditor {
	return &SQLiteSchemaEditor{db: db}
}

func (m *SQLiteSchemaEditor) query(ctx context.Context, query string, args ...any) (drivers.SQLRows, error) {
	// logger.Debugf("SQLiteSchemaEditor.QueryContext:\n%s", query)
	rows, err := m.db.QueryContext(ctx, query, args...)
	return rows, err
}

func (m *SQLiteSchemaEditor) queryRow(ctx context.Context, query string, args ...any) drivers.SQLRow {
	// logger.Debugf("SQLiteSchemaEditor.QueryRowContext:\n%s", query)
	return m.db.QueryRowContext(ctx, query, args...)
}

func (m *SQLiteSchemaEditor) Execute(ctx context.Context, query string, args ...any) (sql.Result, error) {
	// logger.Debugf("SQLiteSchemaEditor.ExecContext:\n%s", query)
	result, err := m.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (m *SQLiteSchemaEditor) Setup(ctx context.Context) error {
	if m.tablesCreated {
		return nil
	}
	_, err := m.Execute(
		ctx,
		createTableMigrations,
	)
	if err != nil {
		return err
	}
	m.tablesCreated = true
	return nil
}

func (m *SQLiteSchemaEditor) StoreMigration(ctx context.Context, appName string, modelName string, migrationName string) error {
	_, err := m.Execute(ctx, insertTableMigrations, appName, modelName, migrationName)
	if err != nil {
		return err
	}
	return nil
}

func (m *SQLiteSchemaEditor) HasMigration(ctx context.Context, appName string, modelName string, migrationName string) (bool, error) {
	var count int
	var err = m.queryRow(ctx, selectTableMigrations, appName, modelName, migrationName).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (m *SQLiteSchemaEditor) RemoveMigration(ctx context.Context, appName string, modelName string, migrationName string) error {
	_, err := m.Execute(ctx, deleteTableMigrations, appName, modelName, migrationName)
	if err != nil {
		return err
	}
	return nil
}

func (m *SQLiteSchemaEditor) CreateTable(ctx context.Context, table migrator.Table, ifNotExists bool) error {
	var w strings.Builder
	w.WriteString("CREATE TABLE ")
	if ifNotExists {
		w.WriteString("IF NOT EXISTS ")
	}
	w.WriteString("`")
	w.WriteString(table.TableName())
	w.WriteString("` (")
	w.WriteString("\n")

	var written bool
	var cols = table.Columns()
	for _, col := range cols {
		if !col.UseInDB {
			continue
		}

		if written {
			w.WriteString(", ")
			w.WriteString("\n")
		}

		w.WriteString("  ")
		m.WriteColumn(&w, *col)

		written = true
	}
	w.WriteString("\n")
	w.WriteString(");")
	w.WriteString("\n")

	// Execute the query
	_, err := m.Execute(ctx, w.String())
	return err
}

func (m *SQLiteSchemaEditor) DropTable(ctx context.Context, table migrator.Table, ifExists bool) error {
	var w strings.Builder
	w.WriteString("DROP TABLE ")
	if ifExists {
		w.WriteString("IF EXISTS ")
	}
	w.WriteString("`")
	w.WriteString(table.TableName())
	w.WriteString("`;")
	w.WriteString("\n")

	// Execute the query
	_, err := m.Execute(ctx, w.String())
	return err
}

func (m *SQLiteSchemaEditor) RenameTable(ctx context.Context, table migrator.Table, newName string) error {
	var w strings.Builder
	w.WriteString("ALTER TABLE `")
	w.WriteString(table.TableName())
	w.WriteString("` RENAME TO `")
	w.WriteString(newName)
	w.WriteString("`;")
	w.WriteString("\n")

	// Execute the query
	_, err := m.Execute(ctx, w.String())
	return err
}

func (m *SQLiteSchemaEditor) AddIndex(ctx context.Context, table migrator.Table, index migrator.Index, ifNotExists bool) error {
	var w strings.Builder
	if index.Unique {
		w.WriteString("CREATE UNIQUE INDEX ")
	} else {
		w.WriteString("CREATE INDEX ")
	}
	if ifNotExists {
		w.WriteString("IF NOT EXISTS ")
	}
	w.WriteString("`")
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
	}
	w.WriteString(")")
	if index.Comment != "" {
		w.WriteString(" COMMENT '")
		w.WriteString(index.Comment)
		w.WriteString("'")
	}
	w.WriteString("\n")
	w.WriteString(";")
	w.WriteString("\n")

	// Execute the query
	_, err := m.Execute(ctx, w.String())
	return err
}

func (m *SQLiteSchemaEditor) DropIndex(ctx context.Context, table migrator.Table, index migrator.Index, ifExists bool) error {
	var w strings.Builder
	w.WriteString("DROP INDEX ")
	if ifExists {
		w.WriteString("IF EXISTS ")
	}
	w.WriteString("`")
	w.WriteString(index.Name())
	w.WriteString("`;")
	w.WriteString("\n")

	// Execute the query
	_, err := m.Execute(ctx, w.String())
	return err
}

func (m *SQLiteSchemaEditor) RenameIndex(ctx context.Context, table migrator.Table, oldName string, newName string) error {
	var w strings.Builder
	w.WriteString("ALTER INDEX `")
	w.WriteString(oldName)
	w.WriteString("` RENAME TO `")
	w.WriteString(newName)
	w.WriteString("`;")
	w.WriteString("\n")

	// Execute the query
	_, err := m.Execute(ctx, w.String())
	return err
}

//	func (m *SQLiteSchemaEditor) AlterUniqueTogether(table migrator.Table, unique bool) error {
//		var w strings.Builder
//		w.WriteString("ALTER TABLE `")
//		w.WriteString(table.TableName())
//		w.WriteString("` SET UNIQUE (")
//		for i, col := range table.Columns() {
//			if i > 0 {
//				w.WriteString(", ")
//			}
//			w.WriteString("`")
//			w.WriteString(col.Field.ColumnName())
//			w.WriteString("`")
//		}
//		w.WriteString(")")
//		if unique {
//			w.WriteString(" UNIQUE")
//		} else {
//			w.WriteString(" NOT UNIQUE")
//		}
//		w.WriteString("\n")
//		w.WriteString(";")
//		w.WriteString("\n")
//
//		// Execute the query
//		_, err := m.db.Exec(w.String())
//		return err
//	}
//
//	func (m *SQLiteSchemaEditor) AlterIndexTogether(table migrator.Table, unique bool) error {
//		var w strings.Builder
//		w.WriteString("ALTER TABLE `")
//		w.WriteString(table.TableName())
//		w.WriteString("` SET INDEX (")
//		for i, col := range table.Columns() {
//			if i > 0 {
//				w.WriteString(", ")
//			}
//			w.WriteString("`")
//			w.WriteString(col.Field.ColumnName())
//			w.WriteString("`")
//		}
//		w.WriteString(")")
//		if unique {
//			w.WriteString(" UNIQUE")
//		} else {
//			w.WriteString(" NOT UNIQUE")
//		}
//		w.WriteString("\n")
//		w.WriteString(";")
//		w.WriteString("\n")
//
//		// Execute the query
//		_, err := m.db.Exec(w.String())
//		return err
//	}

func (m *SQLiteSchemaEditor) AddField(ctx context.Context, table migrator.Table, col migrator.Column) error {
	var w strings.Builder
	w.WriteString("ALTER TABLE `")
	w.WriteString(table.TableName())
	w.WriteString("` ADD COLUMN ")
	m.WriteColumn(&w, col)
	w.WriteString("\n")
	w.WriteString(";")
	w.WriteString("\n")

	// Execute the query
	_, err := m.Execute(ctx, w.String())
	return err
}

func (m *SQLiteSchemaEditor) RemoveField(ctx context.Context, table migrator.Table, col migrator.Column) error {
	var w strings.Builder
	w.WriteString("ALTER TABLE `")
	w.WriteString(table.TableName())
	w.WriteString("` DROP COLUMN `")
	w.WriteString(col.Column)
	w.WriteString("`;")
	w.WriteString("\n")

	// Execute the query
	_, err := m.Execute(ctx, w.String())
	return err
}
func (m *SQLiteSchemaEditor) AlterField(
	ctx context.Context,
	table migrator.Table,
	oldCol migrator.Column,
	newCol migrator.Column,
) error {
	var (
		tableName     = table.TableName()
		tempTableName = tableName + "__tmp"
		columns       = table.Columns()
	)

	// Step 1: Prepare new table structure with updated column
	var newTable = &migrator.ModelTable{
		Table:  tempTableName,
		Object: table.Model(),
		Fields: orderedmap.NewOrderedMap[string, migrator.Column](),
	}

	var (
		columnNamesDst []string
		columnNamesSrc []string
	)

	for _, c := range columns {
		if !c.UseInDB {
			continue
		}

		if c.Name == oldCol.Name {
			newTable.Fields.Set(newCol.Name, newCol)
			columnNamesDst = append(columnNamesDst, fmt.Sprintf("`%s`", newCol.Column))
			if oldCol.Nullable {
				columnNamesSrc = append(columnNamesSrc, fmt.Sprintf("NULL AS `%s`", oldCol.Column))
			} else {
				switch oldCol.Field.Type().Kind() {
				case reflect.String:
					columnNamesSrc = append(columnNamesSrc, fmt.Sprintf("'' AS `%s`", oldCol.Column))
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					columnNamesSrc = append(columnNamesSrc, fmt.Sprintf("0 AS `%s`", oldCol.Column))
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					columnNamesSrc = append(columnNamesSrc, fmt.Sprintf("0 AS `%s`", oldCol.Column))
				case reflect.Float32, reflect.Float64:
					columnNamesSrc = append(columnNamesSrc, fmt.Sprintf("0.0 AS `%s`", oldCol.Column))
				case reflect.Bool:
					columnNamesSrc = append(columnNamesSrc, fmt.Sprintf("0 AS `%s`", oldCol.Column))
				default:
					columnNamesSrc = append(columnNamesSrc, fmt.Sprintf("NULL AS `%s`", oldCol.Column))
				}
			}
		} else {
			newTable.Fields.Set(c.Name, *c)
			columnNamesDst = append(columnNamesDst, fmt.Sprintf("`%s`", c.Column))
			columnNamesSrc = append(columnNamesSrc, fmt.Sprintf("`%s`", c.Column))
		}
	}

	// Step 2: Fetch related schema (indexes, triggers)
	var rows, err = m.query(ctx, `
		SELECT type, name, sql FROM sqlite_schema
		WHERE tbl_name = ? AND type IN ('index', 'trigger') AND sql IS NOT NULL;
	`, tableName)
	if err != nil {
		return fmt.Errorf("fetch schema items: %w", err)
	}
	defer rows.Close()

	_, err = m.Execute(ctx, fmt.Sprintf(
		"DROP TABLE IF EXISTS `%s`;", tempTableName,
	))
	if err != nil {
		return fmt.Errorf("drop temp table: %w", err)
	}

	var schemaItems []struct {
		Type string
		Name string
		SQL  string
	}

	for rows.Next() {
		var typ, name, sqlStmt string
		if err := rows.Scan(&typ, &name, &sqlStmt); err != nil {
			return fmt.Errorf("scan schema item: %w", err)
		}
		schemaItems = append(schemaItems, struct {
			Type string
			Name string
			SQL  string
		}{typ, name, sqlStmt})
	}

	// Step 3: Create temp table
	if err := m.CreateTable(ctx, newTable, false); err != nil {
		return fmt.Errorf("create temp table: %w", err)
	}

	// Step 4: Copy data
	var copyStmt = fmt.Sprintf(
		"INSERT INTO `%s` (%s) SELECT %s FROM `%s`;",
		tempTableName,
		strings.Join(columnNamesDst, ", "),
		strings.Join(columnNamesSrc, ", "),
		tableName,
	)
	if _, err := m.Execute(ctx, copyStmt); err != nil {
		return fmt.Errorf("copy data to temp table: %w", err)
	}

	// Step 5: Drop original table
	if err := m.DropTable(ctx, table, false); err != nil {
		return fmt.Errorf("drop original table: %w", err)
	}

	// Check if the original table was dropped
	var count int
	err = m.queryRow(ctx, `
		SELECT COUNT(*) FROM sqlite_schema
		WHERE name = ? AND type = 'table';
	`, tableName).Scan(&count)
	if err != nil {
		return fmt.Errorf("check original table: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("original table still exists")
	}

	// Step 6: Rename temp table back
	var renameStmt = fmt.Sprintf(
		"ALTER TABLE `%s` RENAME TO `%s`;",
		tempTableName, tableName,
	)
	if _, err := m.Execute(ctx, renameStmt); err != nil {
		return fmt.Errorf("rename temp table: %w", err)
	}

	// Step 7: Recreate triggers and indexes
	for _, item := range schemaItems {
		sql := strings.ReplaceAll(item.SQL, tempTableName, tableName)
		if _, err := m.Execute(ctx, sql); err != nil {
			return fmt.Errorf("recreate %s %q: %w", item.Type, item.Name, err)
		}
	}

	return nil
}

func (m *SQLiteSchemaEditor) WriteColumn(w *strings.Builder, col migrator.Column) {

	if col.Field == nil {
		panic(fmt.Errorf("field is nil for column %q %#v", col.Column, col))
	}

	w.WriteString("`")
	w.WriteString(col.Column)
	w.WriteString("` ")
	w.WriteString(migrator.GetFieldType(
		&sqlite3.SQLiteDriver{}, &col,
	))

	if col.Primary {
		w.WriteString(" PRIMARY KEY")
	}

	if col.Auto {
		w.WriteString(" AUTOINCREMENT")
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
		case nil:
			w.WriteString("NULL")
		}
	}

checkRels:
	if col.Rel != nil {
		var (
			relDefs  = col.Rel.Model().FieldDefs()
			relField = col.Rel.Field()
		)

		if col.Rel.Through != nil {
			// do nothing
			return
		}

		// field is automatically inferred
		// to be primary field of target model
		if relField == nil {
			relField = relDefs.Primary()
		}

		if relField == nil {
			panic(fmt.Errorf(
				"related field %q not found in target model %T",
				relField, col.Rel.Model(),
			))
		}

		if relField.ColumnName() == "" {
			panic(fmt.Errorf(
				"related field %T.%s has no column name",
				col.Rel.Model(), relField.Name(),
			))
		}

		w.WriteString(" REFERENCES `")
		w.WriteString(relDefs.TableName())
		w.WriteString("` (`")
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
