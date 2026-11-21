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
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr/builder"
	"github.com/Nigel2392/go-django/queries/src/expr/constraints"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
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
			return nil, fmt.Errorf("migrator: sqlite: no database connection found")
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
	rows, err := migrator.DbFromContext(ctx, m.db).QueryContext(ctx, query, args...)
	return rows, err
}

func (m *SQLiteSchemaEditor) queryRow(ctx context.Context, query string, args ...any) drivers.SQLRow {
	// logger.Debugf("SQLiteSchemaEditor.QueryRowContext:\n%s", query)
	return migrator.DbFromContext(ctx, m.db).QueryRowContext(ctx, query, args...)
}

func (m *SQLiteSchemaEditor) Execute(ctx context.Context, query string, args ...any) (sql.Result, error) {
	// logger.Debugf("SQLiteSchemaEditor.ExecContext:\n%s", query)
	result, err := migrator.DbFromContext(ctx, m.db).ExecContext(ctx, query, args...)
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

type sqliteMigrationTx struct {
	drivers.Transaction
	db          drivers.Database
	restoreFKOn bool
}

func (t *sqliteMigrationTx) restoreFK(ctx context.Context) {
	if !t.restoreFKOn {
		return
	}
	t.restoreFKOn = false
	_, _ = t.db.ExecContext(ctx, "PRAGMA foreign_keys = ON;")
}

func (t *sqliteMigrationTx) Commit(ctx context.Context) error {
	err := t.Transaction.Commit(ctx)
	t.restoreFK(ctx)
	return err
}

func (t *sqliteMigrationTx) Rollback(ctx context.Context) error {
	err := t.Transaction.Rollback(ctx)
	t.restoreFK(ctx)
	return err
}

func (m *SQLiteSchemaEditor) StartTransaction(ctx context.Context) (drivers.Transaction, error) {
	var fkOn int
	_ = m.db.QueryRowContext(ctx, "PRAGMA foreign_keys;").Scan(&fkOn)
	if fkOn != 0 {
		if _, err := m.db.ExecContext(ctx, "PRAGMA foreign_keys = OFF;"); err != nil {
			return nil, err
		}
	}
	tx, err := m.db.Begin(ctx)
	if err != nil {
		if fkOn != 0 {
			_, _ = m.db.ExecContext(ctx, "PRAGMA foreign_keys = ON;")
		}
		return nil, err
	}
	if fkOn != 0 {
		return &sqliteMigrationTx{
			Transaction: tx,
			db:          m.db,
			restoreFKOn: true,
		}, nil
	}
	return tx, nil
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
		w.WriteString(" /* ")
		w.WriteString(strings.ReplaceAll(index.Comment, "*/", ""))
		w.WriteString(" */")
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
	// SQLite has no ALTER INDEX: recreate the index under the new name.
	var stmt string
	var err = m.queryRow(ctx,
		"SELECT sql FROM sqlite_schema WHERE type = 'index' AND name = ?", oldName,
	).Scan(&stmt)
	if err != nil {
		return fmt.Errorf("fetch index %q: %w", oldName, err)
	}
	if _, err = m.Execute(ctx, fmt.Sprintf("DROP INDEX `%s`;", oldName)); err != nil {
		return err
	}
	_, err = m.Execute(ctx, strings.Replace(stmt, "`"+oldName+"`", "`"+newName+"`", 1))
	return err
}

func (m *SQLiteSchemaEditor) AddConstraint(ctx context.Context, table migrator.Table, constraint constraints.Constraint) error {
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
	constraintSQL := strings.TrimSpace(sb.String())
	if constraintSQL == "" {
		return fmt.Errorf("constraint %q generated empty SQL", constraint.ID())
	}

	items, suffix, err := m.readSQLiteTableItems(ctx, table.TableName())
	if err != nil {
		return err
	}
	if constraint.ID() != "" {
		for _, item := range items {
			if isC, name := parseSQLiteTableConstraint(item); isC && strings.EqualFold(name, constraint.ID()) {
				return fmt.Errorf("constraint %q already exists on table %q", constraint.ID(), table.TableName())
			}
		}
	}
	items = append(items, constraintSQL)

	cols, err := m.sqliteColumnNames(ctx, table.TableName())
	if err != nil {
		return err
	}
	return m.rebuildSQLiteTable(ctx, table.TableName(), items, suffix, cols, cols, "", "")
}

func (m *SQLiteSchemaEditor) DropConstraint(ctx context.Context, table migrator.Table, constraint constraints.Constraint) error {
	if constraint == nil {
		return errors.NilPointer.Wrap("constraint is nil")
	}
	if constraint.ID() == "" {
		return errors.ValueError.Wrap("constraint ID cannot be empty")
	}

	items, suffix, err := m.readSQLiteTableItems(ctx, table.TableName())
	if err != nil {
		return err
	}

	filtered := make([]string, 0, len(items))
	found := false
	for _, item := range items {
		if isC, name := parseSQLiteTableConstraint(item); isC && strings.EqualFold(name, constraint.ID()) {
			found = true
			continue
		}
		filtered = append(filtered, item)
	}
	if !found {
		return fmt.Errorf("constraint %q not found on table %q", constraint.ID(), table.TableName())
	}

	cols, err := m.sqliteColumnNames(ctx, table.TableName())
	if err != nil {
		return err
	}
	return m.rebuildSQLiteTable(ctx, table.TableName(), filtered, suffix, cols, cols, "", "")
}

func (m *SQLiteSchemaEditor) readSQLiteTableItems(ctx context.Context, tableName string) (items []string, suffix string, err error) {
	var createSQL string
	if err := m.queryRow(ctx,
		"SELECT sql FROM sqlite_schema WHERE type = 'table' AND name = ?", tableName,
	).Scan(&createSQL); err != nil {
		return nil, "", fmt.Errorf("fetch create table SQL for %q: %w", tableName, err)
	}

	openIdx := strings.IndexByte(createSQL, '(')
	if openIdx == -1 {
		return nil, "", fmt.Errorf("invalid CREATE TABLE SQL for %q: missing '('", tableName)
	}

	var (
		depth     = 0
		itemStart = openIdx + 1
		closeIdx  = -1
		inQuote   byte
	)
	for i := openIdx; i < len(createSQL); i++ {
		ch := createSQL[i]
		if inQuote != 0 {
			if inQuote == '[' {
				if ch == ']' {
					inQuote = 0
				}
				continue
			}
			if ch == inQuote {
				if i+1 < len(createSQL) && createSQL[i+1] == inQuote {
					i++
					continue
				}
				inQuote = 0
			}
			continue
		}
		switch ch {
		case '\'', '"', '`', '[':
			inQuote = ch
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				if item := strings.TrimSpace(createSQL[itemStart:i]); item != "" {
					items = append(items, item)
				}
				closeIdx = i
				goto done
			}
		case ',':
			if depth == 1 {
				if item := strings.TrimSpace(createSQL[itemStart:i]); item != "" {
					items = append(items, item)
				}
				itemStart = i + 1
			}
		}
	}
done:
	if closeIdx == -1 {
		return nil, "", fmt.Errorf("invalid CREATE TABLE SQL for %q: missing matching ')'", tableName)
	}
	return items, strings.TrimSpace(createSQL[closeIdx:]), nil
}

func parseSQLiteTableConstraint(item string) (isConstraint bool, name string) {
	s := strings.TrimSpace(item)
	upper := strings.ToUpper(s)
	if strings.HasPrefix(upper, "CONSTRAINT") && len(s) > len("CONSTRAINT") && (s[10] == ' ' || s[10] == '\t' || s[10] == '\n' || s[10] == '\r') {
		rest := strings.TrimSpace(s[len("CONSTRAINT"):])
		if rest == "" {
			return true, ""
		}
		switch rest[0] {
		case '`', '"':
			q := rest[0]
			var out strings.Builder
			for i := 1; i < len(rest); i++ {
				if rest[i] == q {
					if i+1 < len(rest) && rest[i+1] == q {
						out.WriteByte(q)
						i++
						continue
					}
					return true, out.String()
				}
				out.WriteByte(rest[i])
			}
		case '[':
			if end := strings.IndexByte(rest[1:], ']'); end != -1 {
				return true, rest[1 : 1+end]
			}
		default:
			end := strings.IndexAny(rest, " \t\n\r(")
			if end == -1 {
				return true, rest
			}
			return true, rest[:end]
		}
		return true, ""
	}
	for _, kw := range []string{"CHECK", "UNIQUE", "FOREIGN KEY", "PRIMARY KEY"} {
		if strings.HasPrefix(upper, kw) && (len(s) == len(kw) || s[len(kw)] == ' ' || s[len(kw)] == '\t' || s[len(kw)] == '\n' || s[len(kw)] == '(') {
			return true, ""
		}
	}
	return false, ""
}

func (m *SQLiteSchemaEditor) sqliteColumnNames(ctx context.Context, tableName string) ([]string, error) {
	rows, err := m.query(ctx, fmt.Sprintf("PRAGMA table_info(`%s`);", tableName))
	if err != nil {
		return nil, fmt.Errorf("fetch columns for %q: %w", tableName, err)
	}
	defer rows.Close()

	var cols []string
	for rows.Next() {
		var (
			cid     int
			name    string
			typ     string
			notnull int
			dflt    sql.NullString
			pk      int
		)
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return nil, fmt.Errorf("scan table_info for %q: %w", tableName, err)
		}
		cols = append(cols, fmt.Sprintf("`%s`", name))
	}
	if len(cols) == 0 {
		return nil, fmt.Errorf("no columns found for table %q", tableName)
	}
	return cols, nil
}

func (m *SQLiteSchemaEditor) rebuildSQLiteTable(
	ctx context.Context,
	tableName string,
	items []string,
	suffix string,
	dstCols []string,
	srcCols []string,
	renameOldCol string,
	renameNewCol string,
) error {
	rows, err := m.query(ctx, `
		SELECT type, name, sql FROM sqlite_schema
		WHERE tbl_name = ? AND type IN ('index', 'trigger') AND sql IS NOT NULL;
	`, tableName)
	if err != nil {
		return fmt.Errorf("fetch schema items: %w", err)
	}

	var schemaItems []struct {
		Type string
		Name string
		SQL  string
	}
	for rows.Next() {
		var typ, name, sqlStmt string
		if err := rows.Scan(&typ, &name, &sqlStmt); err != nil {
			rows.Close()
			return fmt.Errorf("scan schema item: %w", err)
		}
		schemaItems = append(schemaItems, struct {
			Type string
			Name string
			SQL  string
		}{typ, name, sqlStmt})
	}
	rows.Close()

	tempTableName := tableName + "__tmp"
	if _, err := m.Execute(ctx, fmt.Sprintf("DROP TABLE IF EXISTS `%s`;", tempTableName)); err != nil {
		return fmt.Errorf("drop temp table: %w", err)
	}

	if suffix == "" {
		suffix = ")"
	}
	if !strings.HasSuffix(suffix, ";") {
		suffix += ";"
	}
	createTemp := fmt.Sprintf("CREATE TABLE `%s` (\n  %s\n%s\n", tempTableName, strings.Join(items, ",\n  "), suffix)
	if _, err := m.Execute(ctx, createTemp); err != nil {
		return fmt.Errorf("create temp table: %w", err)
	}

	copyStmt := fmt.Sprintf(
		"INSERT INTO `%s` (%s) SELECT %s FROM `%s`;",
		tempTableName,
		strings.Join(dstCols, ", "),
		strings.Join(srcCols, ", "),
		tableName,
	)
	if _, err := m.Execute(ctx, copyStmt); err != nil {
		return fmt.Errorf("copy data to temp table: %w", err)
	}

	if _, err := m.Execute(ctx, fmt.Sprintf("DROP TABLE `%s`;", tableName)); err != nil {
		return fmt.Errorf("drop original table: %w", err)
	}

	var count int
	if err := m.queryRow(ctx, `SELECT COUNT(*) FROM sqlite_schema WHERE name = ? AND type = 'table';`, tableName).Scan(&count); err != nil {
		return fmt.Errorf("check original table: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("original table still exists")
	}

	if _, err := m.Execute(ctx, fmt.Sprintf("ALTER TABLE `%s` RENAME TO `%s`;", tempTableName, tableName)); err != nil {
		return fmt.Errorf("rename temp table: %w", err)
	}

	for _, item := range schemaItems {
		stmt := strings.ReplaceAll(item.SQL, tempTableName, tableName)
		if renameOldCol != "" && renameOldCol != renameNewCol {
			stmt = strings.ReplaceAll(stmt, "`"+renameOldCol+"`", "`"+renameNewCol+"`")
		}
		if _, err := m.Execute(ctx, stmt); err != nil {
			return fmt.Errorf("recreate %s %q: %w", item.Type, item.Name, err)
		}
	}

	return nil
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
		tableName = table.TableName()
		columns   = table.Columns()
	)

	var (
		columnNamesDst []string
		columnNamesSrc []string
		items          []string
	)

	for _, c := range columns {
		if !c.UseInDB {
			continue
		}

		var colBuf strings.Builder
		includeRel := true
		targetCol := *c
		if c.Name == oldCol.Name {
			targetCol = newCol
			columnNamesDst = append(columnNamesDst, fmt.Sprintf("`%s`", newCol.Column))

			if oldCol.Nullable && !newCol.Nullable {
				switch newCol.Field.Type().Kind() {
				case reflect.String:
					columnNamesSrc = append(columnNamesSrc, fmt.Sprintf("COALESCE(`%s`, '') AS `%s`", oldCol.Column, oldCol.Column))
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
					reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
					reflect.Bool:
					columnNamesSrc = append(columnNamesSrc, fmt.Sprintf("COALESCE(`%s`, 0) AS `%s`", oldCol.Column, oldCol.Column))
				case reflect.Float32, reflect.Float64:
					columnNamesSrc = append(columnNamesSrc, fmt.Sprintf("COALESCE(`%s`, 0.0) AS `%s`", oldCol.Column, oldCol.Column))
				default:
					columnNamesSrc = append(columnNamesSrc, fmt.Sprintf("`%s`", oldCol.Column))
				}
			} else {
				columnNamesSrc = append(columnNamesSrc, fmt.Sprintf("`%s`", oldCol.Column))
			}
		} else {
			columnNamesDst = append(columnNamesDst, fmt.Sprintf("`%s`", c.Column))
			columnNamesSrc = append(columnNamesSrc, fmt.Sprintf("`%s`", c.Column))
		}

		for _, tc := range table.Constraints() {
			if tc != nil && tc.Type() == constraints.C_FOREIGN_KEY {
				if fk, ok := tc.(interface{ ForeignKeyField() string }); ok {
					if name := fk.ForeignKeyField(); name != "" && (name == targetCol.Name || name == targetCol.Column) {
						includeRel = false
						break
					}
				}
			}
		}

		m.writeColumn(&colBuf, targetCol, includeRel)
		items = append(items, colBuf.String())
	}

	var suffix string
	if origItems, origSuffix, err := m.readSQLiteTableItems(ctx, tableName); err == nil {
		suffix = origSuffix
		for _, item := range origItems {
			if isC, _ := parseSQLiteTableConstraint(item); isC {
				if oldCol.Column != newCol.Column {
					item = strings.ReplaceAll(item, "`"+oldCol.Column+"`", "`"+newCol.Column+"`")
					item = strings.ReplaceAll(item, `"`+oldCol.Column+`"`, `"`+newCol.Column+`"`)
				}
				items = append(items, item)
			}
		}
	}

	return m.rebuildSQLiteTable(ctx, tableName, items, suffix, columnNamesDst, columnNamesSrc, oldCol.Column, newCol.Column)
}

func (m *SQLiteSchemaEditor) WriteColumn(w *strings.Builder, col migrator.Column) {
	m.writeColumn(w, col, true)
}

func (m *SQLiteSchemaEditor) writeColumn(w *strings.Builder, col migrator.Column, includeRel bool) {

	if col.Field == nil && col.DBType() == 0 {
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

		defaultVal := col.Default.V
		if valuer, ok := defaultVal.(driver.Valuer); ok {
			// If the default value is a driver.Valuer, we need to call it to get the actual value.
			val, err := valuer.Value()
			if err != nil {
				panic(fmt.Errorf("failed to get value from driver.Valuer: %w", err))
			}
			defaultVal = val
		}

		if defaultVal == nil && !col.Nullable {
			goto checkRels
		}

		w.WriteString(" DEFAULT ")

		switch v := defaultVal.(type) {
		case string:
			w.WriteString("'")
			w.WriteString(strings.ReplaceAll(v, "'", "''"))
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
		default:
			logger.Errorf("Unknown default type for column %q: %T: %v", col.Name, col.Default.V, col.Default.V)
		}
	}

checkRels:
	if includeRel && col.Rel != nil {
		var (
			relDefs  = attrs.GetModelMeta(col.Rel.Model()).Definitions()
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
