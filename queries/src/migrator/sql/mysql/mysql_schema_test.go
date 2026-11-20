package mysql_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"
	"testing"
	"time"

	_ "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/Nigel2392/go-django/queries/src/expr/constraints"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/queries/src/migrator/sql/mysql"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

type recordedSQL struct {
	query string
	args  []any
}

type mockSQLRows struct{}

func (r *mockSQLRows) Close() error             { return nil }
func (r *mockSQLRows) Columns() ([]string, error) { return nil, nil }
func (r *mockSQLRows) Err() error               { return nil }
func (r *mockSQLRows) Next() bool               { return false }
func (r *mockSQLRows) NextResultSet() bool      { return false }
func (r *mockSQLRows) Scan(...any) error        { return nil }

type mockSQLRow struct {
	countVal int64
}

func (r mockSQLRow) Err() error { return nil }
func (r mockSQLRow) Scan(dest ...any) error {
	if len(dest) > 0 {
		switch p := dest[0].(type) {
		case *int:
			*p = int(r.countVal)
		case *int64:
			*p = r.countVal
		case *bool:
			*p = r.countVal > 0
		}
	}
	return nil
}

type mockMySQLDB struct {
	drv        driver.Driver
	queries    []recordedSQL
	queryRows  []recordedSQL
	rowCount   int64
	txFinished bool
}

func (m *mockMySQLDB) record(q string, args ...any) {
	m.queries = append(m.queries, recordedSQL{query: q, args: args})
}

func (m *mockMySQLDB) Unwrap() any             { return m }
func (m *mockMySQLDB) Driver() driver.Driver   { return m.drv }
func (m *mockMySQLDB) Ping(context.Context) error { return nil }
func (m *mockMySQLDB) Close() error            { return nil }
func (m *mockMySQLDB) Finished() bool          { return m.txFinished }
func (m *mockMySQLDB) Commit(context.Context) error {
	m.txFinished = true
	return nil
}
func (m *mockMySQLDB) Rollback(context.Context) error {
	m.txFinished = true
	return nil
}
func (m *mockMySQLDB) Begin(context.Context) (drivers.Transaction, error) {
	m.txFinished = false
	return m, nil
}
func (m *mockMySQLDB) QueryContext(_ context.Context, query string, args ...any) (drivers.SQLRows, error) {
	m.record(query, args...)
	return &mockSQLRows{}, nil
}
func (m *mockMySQLDB) QueryRowContext(_ context.Context, query string, args ...any) drivers.SQLRow {
	m.queryRows = append(m.queryRows, recordedSQL{query: query, args: args})
	m.record(query, args...)
	return mockSQLRow{countVal: m.rowCount}
}
func (m *mockMySQLDB) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	m.record(query, args...)
	return driver.RowsAffected(1), nil
}

func (m *mockMySQLDB) lastQuery() recordedSQL {
	if len(m.queries) == 0 {
		return recordedSQL{}
	}
	return m.queries[len(m.queries)-1]
}

type mysqlTypeHolder struct {
	ID  int64 `attrs:"primary"`
	Val string
}

func (m *mysqlTypeHolder) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions(ctx, m)
}

type overrideMySQLDBTypeField struct {
	attrs.Field
	dbType dbtype.Type
}

func (f *overrideMySQLDBTypeField) DBType(*migrator.Column) dbtype.Type {
	return f.dbType
}

func TestMySQL_And_MariaDB_AllDBTypes(t *testing.T) {
	attrs.RegisterModel(&mysqlTypeHolder{})
	holder := &mysqlTypeHolder{}
	baseField := attrs.NewField(holder, "Val", &attrs.Config{Column: "val"})

	driversList := []struct {
		name string
		drv  driver.Driver
	}{
		{"MySQL", &drivers.DriverMySQL{}},
		{"MariaDB", &drivers.DriverMariaDB{}},
	}

	allTypes := []struct {
		name      string
		dbType    dbtype.Type
		maxLen    int64
		precision int64
		scale     int64
		auto      bool
	}{
		{"Text", dbtype.Text, 0, 0, 0, false},
		{"String_Default", dbtype.String, 0, 0, 0, false},
		{"String_MaxLen", dbtype.String, 100, 0, 0, false},
		{"Char", dbtype.Char, 16, 0, 0, false},
		{"Int", dbtype.Int, 0, 0, 0, false},
		{"Int_Auto", dbtype.Int, 0, 0, 0, true},
		{"Uint", dbtype.Uint, 0, 0, 0, false},
		{"Float", dbtype.Float, 0, 0, 0, false},
		{"Decimal_Default", dbtype.Decimal, 0, 0, 0, false},
		{"Decimal_PrecScale", dbtype.Decimal, 0, 14, 4, false},
		{"Bool", dbtype.Bool, 0, 0, 0, false},
		{"UUID", dbtype.UUID, 0, 0, 0, false},
		{"ULID", dbtype.ULID, 0, 0, 0, false},
		{"Bytes", dbtype.Bytes, 0, 0, 0, false},
		{"JSON", dbtype.JSON, 0, 0, 0, false},
		{"BLOB", dbtype.BLOB, 0, 0, 0, false},
		{"Timestamp", dbtype.Timestamp, 0, 0, 0, false},
		{"LocalTime", dbtype.LocalTime, 0, 0, 0, false},
		{"DateTime", dbtype.DateTime, 0, 0, 0, false},
	}

	for _, d := range driversList {
		t.Run(d.name, func(t *testing.T) {
			for _, tc := range allTypes {
				t.Run(tc.name, func(t *testing.T) {
					col := migrator.Column{
						Field:     &overrideMySQLDBTypeField{Field: baseField, dbType: tc.dbType},
						Name:      tc.name,
						Column:    strings.ToLower(tc.name),
						UseInDB:   true,
						MaxLength: tc.maxLen,
						Precision: tc.precision,
						Scale:     tc.scale,
						Auto:      tc.auto,
					}
					sqlType := migrator.GetFieldType(d.drv, &col)
					if sqlType == "" {
						t.Fatalf("empty %s SQL type for %s", d.name, tc.name)
					}
					if tc.maxLen > 0 && tc.dbType == dbtype.String {
						if !strings.Contains(sqlType, fmt.Sprintf("%d", tc.maxLen)) {
							t.Errorf("expected length %d in %s type, got %q", tc.maxLen, d.name, sqlType)
						}
					}
					if tc.precision > 0 && tc.dbType == dbtype.Decimal {
						if !strings.Contains(sqlType, "14") || !strings.Contains(sqlType, "4") {
							t.Errorf("expected DECIMAL(14, 4) in %s type, got %q", d.name, sqlType)
						}
					}
					t.Logf("%s %s -> %s", d.name, tc.name, sqlType)
				})
			}
		})
	}
}

type mysqlParentModel struct {
	ID   int64 `attrs:"primary"`
	Name string
}

func (m *mysqlParentModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(m, "Name", &attrs.Config{MaxLength: 100}),
	).WithTableName("mysql_ddl_parent")
}

type mysqlChildModel struct {
	ID        int64 `attrs:"primary"`
	Title     string
	Code      string
	Count     int64
	Ratio     float64
	Active    bool
	FixedTime time.Time
	Parent    *mysqlParentModel
}

func (m *mysqlChildModel) DatabaseIndexes(attrs.Definer) []migrator.Index {
	return []migrator.Index{
		{Identifier: "mysql_ddl_child_title_idx", Fields: []string{"Title"}, Unique: false},
		{Identifier: "mysql_ddl_child_code_uniq", Fields: []string{"Code"}, Unique: true},
	}
}

func (m *mysqlChildModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(m, "Title", &attrs.Config{MaxLength: 120, Default: "untitled"}),
		attrs.NewField(m, "Code", &attrs.Config{MaxLength: 32, Attributes: map[string]any{attrs.AttrUniqueKey: true}}),
		attrs.NewField(m, "Count", &attrs.Config{Default: int64(5)}),
		attrs.NewField(m, "Ratio", &attrs.Config{Null: true}),
		attrs.NewField(m, "Active", &attrs.Config{Default: true}),
		attrs.NewField(m, "FixedTime", &attrs.Config{Default: time.Date(2025, 3, 4, 5, 6, 7, 0, time.UTC)}),
		attrs.NewField(m, "Parent", &attrs.Config{
			Column:        "parent_id",
			Null:          true,
			RelForeignKey: attrs.Relate(&mysqlParentModel{}, "", nil),
			Attributes: map[string]any{
				migrator.AttrOnDeleteKey: constraints.A_CASCADE,
				migrator.AttrOnUpdateKey: constraints.A_SET_NULL,
			},
		}),
	).WithTableName("mysql_ddl_child")
}

func TestMySQLSchemaEditor_DDL(t *testing.T) {
	ctx := context.Background()
	db := &mockMySQLDB{drv: &drivers.DriverMySQL{}}
	ed := mysql.NewMySQLSchemaEditor(db)

	attrs.RegisterModel(&mysqlParentModel{})
	attrs.RegisterModel(&mysqlChildModel{})

	// 1. Setup / migrations table
	if err := ed.Setup(ctx); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if len(db.queries) == 0 || !strings.Contains(strings.ToUpper(db.lastQuery().query), "CREATE TABLE") || !strings.Contains(db.lastQuery().query, "migrations") {
		t.Errorf("expected Setup to create migrations table, got %q", db.lastQuery().query)
	}

	// 2. StoreMigration / HasMigration / RemoveMigration
	if err := ed.StoreMigration(ctx, "myapp", "MyModel", "0001_init.mig"); err != nil {
		t.Fatalf("StoreMigration: %v", err)
	}
	if !strings.Contains(strings.ToUpper(db.lastQuery().query), "INSERT INTO") || len(db.lastQuery().args) != 3 {
		t.Errorf("unexpected StoreMigration query: %+v", db.lastQuery())
	}

	db.rowCount = 1
	has, err := ed.HasMigration(ctx, "myapp", "MyModel", "0001_init.mig")
	if err != nil || !has {
		t.Fatalf("expected HasMigration=true, got has=%v err=%v", has, err)
	}
	db.rowCount = 0
	has, err = ed.HasMigration(ctx, "myapp", "MyModel", "0001_init.mig")
	if err != nil || has {
		t.Fatalf("expected HasMigration=false, got has=%v err=%v", has, err)
	}

	if err := ed.RemoveMigration(ctx, "myapp", "MyModel", "0001_init.mig"); err != nil {
		t.Fatalf("RemoveMigration: %v", err)
	}
	if !strings.Contains(strings.ToUpper(db.lastQuery().query), "DELETE FROM") {
		t.Errorf("unexpected RemoveMigration query: %q", db.lastQuery().query)
	}

	// 3. Transaction helper
	tx, err := ed.StartTransaction(ctx)
	if err != nil {
		t.Fatalf("StartTransaction: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("tx.Commit: %v", err)
	}

	// 4. WriteColumn checks
	childTbl := migrator.NewModelTable(&mysqlChildModel{})
	cols := make(map[string]migrator.Column)
	for _, c := range childTbl.Columns() {
		cols[c.Name] = *c
	}

	var colBuf strings.Builder
	mysql.WriteColumn(&colBuf, cols["ID"])
	if !strings.Contains(strings.ToUpper(colBuf.String()), "PRIMARY KEY") || !strings.Contains(strings.ToUpper(colBuf.String()), "AUTO_INCREMENT") {
		t.Errorf("expected PRIMARY KEY AUTO_INCREMENT on ID, got %q", colBuf.String())
	}

	colBuf.Reset()
	mysql.WriteColumn(&colBuf, cols["Title"])
	if !strings.Contains(colBuf.String(), "DEFAULT 'untitled'") {
		t.Errorf("expected string default on Title, got %q", colBuf.String())
	}

	colBuf.Reset()
	mysql.WriteColumn(&colBuf, cols["Code"])
	if !strings.Contains(strings.ToUpper(colBuf.String()), "UNIQUE") {
		t.Errorf("expected UNIQUE on Code, got %q", colBuf.String())
	}

	colBuf.Reset()
	mysql.WriteColumn(&colBuf, cols["Parent"])
	t.Logf("MySQL FK column: %s", colBuf.String())

	// 5. CreateTable / DropTable / RenameTable
	if err := ed.CreateTable(ctx, childTbl, true); err != nil {
		t.Fatalf("CreateTable ifNotExists=true: %v", err)
	}
	if !strings.Contains(db.lastQuery().query, "CREATE TABLE IF NOT EXISTS `mysql_ddl_child`") {
		t.Errorf("unexpected CreateTable SQL: %q", db.lastQuery().query)
	}

	if err := ed.RenameTable(ctx, childTbl, "mysql_ddl_child_v2"); err != nil {
		t.Fatalf("RenameTable: %v", err)
	}
	if !strings.Contains(db.lastQuery().query, "mysql_ddl_child") || !strings.Contains(db.lastQuery().query, "mysql_ddl_child_v2") {
		t.Errorf("unexpected RenameTable SQL: %q", db.lastQuery().query)
	}

	if err := ed.DropTable(ctx, childTbl, true); err != nil {
		t.Fatalf("DropTable: %v", err)
	}
	if !strings.Contains(db.lastQuery().query, "DROP TABLE IF EXISTS `mysql_ddl_child`") {
		t.Errorf("unexpected DropTable SQL: %q", db.lastQuery().query)
	}

	// 6. AddField / AlterField / RemoveField
	if err := ed.AddField(ctx, childTbl, cols["Title"]); err != nil {
		t.Fatalf("AddField: %v", err)
	}
	if !strings.Contains(strings.ToUpper(db.lastQuery().query), "ALTER TABLE") || !strings.Contains(strings.ToUpper(db.lastQuery().query), "ADD COLUMN") {
		t.Errorf("unexpected AddField SQL: %q", db.lastQuery().query)
	}

	newTitle := cols["Title"]
	newTitle.Column = "title_v2"
	newTitle.MaxLength = 250
	if err := ed.AlterField(ctx, childTbl, cols["Title"], newTitle); err != nil {
		t.Fatalf("AlterField: %v", err)
	}
	t.Logf("MySQL AlterField SQL: %s", db.lastQuery().query)
	if !strings.Contains(strings.ToUpper(db.lastQuery().query), "CHANGE COLUMN") {
		t.Errorf("expected CHANGE COLUMN for MySQL rename in AlterField SQL: %q", db.lastQuery().query)
	}

	if err := ed.RemoveField(ctx, childTbl, cols["Title"]); err != nil {
		t.Fatalf("RemoveField: %v", err)
	}
	if !strings.Contains(strings.ToUpper(db.lastQuery().query), "DROP COLUMN") {
		t.Errorf("unexpected RemoveField SQL: %q", db.lastQuery().query)
	}

	// 7. AddIndex / DropIndex
	for _, idx := range childTbl.Indexes() {
		if err := ed.AddIndex(ctx, childTbl, idx, false); err != nil {
			t.Fatalf("AddIndex %s: %v", idx.Name(), err)
		}
		if !strings.Contains(db.lastQuery().query, idx.Name()) {
			t.Errorf("expected index name %q in AddIndex SQL %q", idx.Name(), db.lastQuery().query)
		}
	}
	if err := ed.DropIndex(ctx, childTbl, childTbl.Indexes()[0], false); err != nil {
		t.Fatalf("DropIndex: %v", err)
	}
	if !strings.Contains(strings.ToUpper(db.lastQuery().query), "DROP INDEX") {
		t.Errorf("unexpected DropIndex SQL: %q", db.lastQuery().query)
	}
}

func TestBug_MySQL_AlterField_ColumnRenameUsesChangeColumn(t *testing.T) {
	ctx := context.Background()
	db := &mockMySQLDB{drv: &drivers.DriverMySQL{}}
	ed := mysql.NewMySQLSchemaEditor(db)
	attrs.RegisterModel(&mysqlParentModel{})
	attrs.RegisterModel(&mysqlChildModel{})

	childTbl := migrator.NewModelTable(&mysqlChildModel{})
	var oldTitle migrator.Column
	for _, c := range childTbl.Columns() {
		if c.Name == "Title" {
			oldTitle = *c
		}
	}
	newTitle := oldTitle
	newTitle.Column = "title_v2"
	newTitle.MaxLength = 250

	if err := ed.AlterField(ctx, childTbl, oldTitle, newTitle); err != nil {
		t.Fatalf("AlterField: %v", err)
	}
	q := db.lastQuery().query
	if !strings.Contains(q, "CHANGE COLUMN `title` `title_v2`") {
		t.Errorf("expected CHANGE COLUMN `title` `title_v2` in MySQL rename SQL, got %q", q)
	}
}

func TestBug_MySQL_CreateTable_ForeignKeyUsesTableLevelConstraint(t *testing.T) {
	ctx := context.Background()
	db := &mockMySQLDB{drv: &drivers.DriverMySQL{}}
	ed := mysql.NewMySQLSchemaEditor(db)
	attrs.RegisterModel(&mysqlParentModel{})
	attrs.RegisterModel(&mysqlChildModel{})

	childTbl := migrator.NewModelTable(&mysqlChildModel{})
	if err := ed.CreateTable(ctx, childTbl, false); err != nil {
		t.Fatalf("CreateTable: %v", err)
	}
	q := db.lastQuery().query
	if !strings.Contains(q, "FOREIGN KEY (`parent_id`) REFERENCES `mysql_ddl_parent`(`id`)") {
		t.Errorf("expected table-level FOREIGN KEY constraint in MySQL CREATE TABLE (InnoDB ignores inline REFERENCES), got:\n%s", q)
	}
}

func TestBug_MySQL_DefaultSingleQuoteEscaping(t *testing.T) {
	attrs.RegisterModel(&mysqlChildModel{})
	childTbl := migrator.NewModelTable(&mysqlChildModel{})
	var titleCol migrator.Column
	for _, c := range childTbl.Columns() {
		if c.Name == "Title" {
			titleCol = *c
		}
	}
	titleCol.Default = drivers.Value[any]{V: "it's"}

	var buf strings.Builder
	mysql.WriteColumn(&buf, titleCol)
	s := buf.String()
	if !strings.Contains(s, "DEFAULT 'it''s'") && !strings.Contains(s, `DEFAULT 'it\'s'`) {
		t.Errorf("MySQL WriteColumn did not escape single quote in default: %q", s)
	}
}

func TestBug_MySQL_DropIndex_IfExistsChecksInformationSchema(t *testing.T) {
	ctx := context.Background()
	db := &mockMySQLDB{drv: &drivers.DriverMySQL{}, rowCount: 0}
	ed := mysql.NewMySQLSchemaEditor(db)
	attrs.RegisterModel(&mysqlParentModel{})
	attrs.RegisterModel(&mysqlChildModel{})

	childTbl := migrator.NewModelTable(&mysqlChildModel{})
	idx := childTbl.Indexes()[0]

	if err := ed.DropIndex(ctx, childTbl, idx, true); err != nil {
		t.Fatalf("DropIndex(ifExists=true): %v", err)
	}
	checkedInfoSchema := false
	for _, rec := range db.queryRows {
		if strings.Contains(strings.ToLower(rec.query), "information_schema.statistics") {
			checkedInfoSchema = true
		}
	}
	if !checkedInfoSchema {
		t.Errorf("expected DropIndex(ifExists=true) to check information_schema.statistics before dropping")
	}
	for _, rec := range db.queries {
		if strings.Contains(strings.ToUpper(rec.query), "DROP INDEX") {
			t.Errorf("expected DROP INDEX to be skipped when index does not exist (rowCount=0), executed %q", rec.query)
		}
	}
}
