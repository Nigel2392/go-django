package postgres_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	derrors "github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/expr/constraints"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/queries/src/migrator/sql/postgres"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/queries/src/resolver"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/pkg/errors"
)

type recordedPGSQL struct {
	query string
	args  []any
}

type mockPGRows struct{}

func (r *mockPGRows) Close() error             { return nil }
func (r *mockPGRows) Columns() ([]string, error) { return nil, nil }
func (r *mockPGRows) Err() error               { return nil }
func (r *mockPGRows) Next() bool               { return false }
func (r *mockPGRows) NextResultSet() bool      { return false }
func (r *mockPGRows) Scan(...any) error        { return nil }

type mockPGRow struct {
	countVal int64
}

func (r mockPGRow) Err() error { return nil }
func (r mockPGRow) Scan(dest ...any) error {
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

type mockPostgresDB struct {
	queries    []recordedPGSQL
	rowCount   int64
	txFinished bool
}

func (m *mockPostgresDB) record(q string, args ...any) {
	m.queries = append(m.queries, recordedPGSQL{query: q, args: args})
}

func (m *mockPostgresDB) Unwrap() any             { return m }
func (m *mockPostgresDB) Driver() driver.Driver   { return &drivers.DriverPostgres{} }
func (m *mockPostgresDB) Ping(context.Context) error { return nil }
func (m *mockPostgresDB) Close() error            { return nil }
func (m *mockPostgresDB) Finished() bool          { return m.txFinished }
func (m *mockPostgresDB) Commit(context.Context) error {
	m.txFinished = true
	return nil
}
func (m *mockPostgresDB) Rollback(context.Context) error {
	m.txFinished = true
	return nil
}
func (m *mockPostgresDB) Begin(context.Context) (drivers.Transaction, error) {
	m.txFinished = false
	return m, nil
}
func (m *mockPostgresDB) QueryContext(_ context.Context, query string, args ...any) (drivers.SQLRows, error) {
	m.record(query, args...)
	return &mockPGRows{}, nil
}
func (m *mockPostgresDB) QueryRowContext(_ context.Context, query string, args ...any) drivers.SQLRow {
	m.record(query, args...)
	return mockPGRow{countVal: m.rowCount}
}
func (m *mockPostgresDB) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	m.record(query, args...)
	return driver.RowsAffected(1), nil
}

func (m *mockPostgresDB) lastQuery() recordedPGSQL {
	if len(m.queries) == 0 {
		return recordedPGSQL{}
	}
	return m.queries[len(m.queries)-1]
}

type pgTypeHolder struct {
	ID  int64 `attrs:"primary"`
	Val string
}

func (m *pgTypeHolder) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions(ctx, m)
}

type overridePGDBTypeField struct {
	attrs.Field
	dbType dbtype.Type
}

func (f *overridePGDBTypeField) DBType(*migrator.Column) dbtype.Type {
	return f.dbType
}

func TestPostgres_AllDBTypes(t *testing.T) {
	attrs.RegisterModel(&pgTypeHolder{})
	holder := &pgTypeHolder{}
	baseField := attrs.NewField(holder, "Val", &attrs.Config{Column: "val"})

	drv := &drivers.DriverPostgres{}
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
		{"String_MaxLen", dbtype.String, 80, 0, 0, false},
		{"Char", dbtype.Char, 12, 0, 0, false},
		{"Int", dbtype.Int, 0, 0, 0, false},
		{"Int_Auto", dbtype.Int, 0, 0, 0, true},
		{"Uint", dbtype.Uint, 0, 0, 0, false},
		{"Float", dbtype.Float, 0, 0, 0, false},
		{"Decimal_Default", dbtype.Decimal, 0, 0, 0, false},
		{"Decimal_PrecScale", dbtype.Decimal, 0, 15, 3, false},
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

	for _, tc := range allTypes {
		t.Run(tc.name, func(t *testing.T) {
			col := migrator.Column{
				Field:     &overridePGDBTypeField{Field: baseField, dbType: tc.dbType},
				Name:      tc.name,
				Column:    strings.ToLower(tc.name),
				UseInDB:   true,
				MaxLength: tc.maxLen,
				Precision: tc.precision,
				Scale:     tc.scale,
				Auto:      tc.auto,
			}
			sqlType := migrator.GetFieldType(drv, &col)
			if sqlType == "" {
				t.Fatalf("empty Postgres SQL type for %s", tc.name)
			}
			if tc.dbType == dbtype.Int && sqlType != "BIGINT" {
				t.Errorf("expected BIGINT for Int in Postgres, got %q", sqlType)
			}
			if tc.maxLen > 0 && tc.dbType == dbtype.String && !strings.Contains(sqlType, fmt.Sprintf("%d", tc.maxLen)) {
				t.Errorf("expected %d in Postgres VARCHAR type, got %q", tc.maxLen, sqlType)
			}
			if tc.precision > 0 && tc.dbType == dbtype.Decimal && (!strings.Contains(sqlType, "15") || !strings.Contains(sqlType, "3")) {
				t.Errorf("expected 15,3 in Postgres DECIMAL/NUMERIC type, got %q", sqlType)
			}
			t.Logf("Postgres %s -> %s", tc.name, sqlType)
		})
	}
}

type pgParentModel struct {
	ID   int64 `attrs:"primary"`
	Name string
}

func (m *pgParentModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(m, "Name", &attrs.Config{MaxLength: 100}),
	).WithTableName("pg_ddl_parent")
}

type pgChildModel struct {
	ID        int64 `attrs:"primary"`
	Title     string
	Code      string
	Count     int64
	Ratio     float64
	Active    bool
	FixedTime time.Time
	Parent    *pgParentModel
}

func (m *pgChildModel) DatabaseIndexes(attrs.Definer) []migrator.Index {
	return []migrator.Index{
		{Identifier: "pg_ddl_child_title_idx", Fields: []string{"Title"}, Unique: false},
		{Identifier: "pg_ddl_child_code_uniq", Fields: []string{"Code"}, Unique: true},
	}
}

func (m *pgChildModel) FieldDefs(ctx context.Context) attrs.Definitions {
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
			RelForeignKey: attrs.Relate(&pgParentModel{}, "", nil),
			Attributes: map[string]any{
				migrator.AttrOnDeleteKey: constraints.A_CASCADE,
				migrator.AttrOnUpdateKey: constraints.A_SET_NULL,
			},
		}),
	).WithTableName("pg_ddl_child")
}

func TestPostgresSchemaEditor_DDL(t *testing.T) {
	ctx := context.Background()
	db := &mockPostgresDB{}
	ed := postgres.NewPostgresSchemaEditor(db)

	attrs.RegisterModel(&pgParentModel{})
	attrs.RegisterModel(&pgChildModel{})

	// 1. Setup / migrations table
	if err := ed.Setup(ctx); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if len(db.queries) == 0 || !strings.Contains(strings.ToUpper(db.lastQuery().query), "CREATE TABLE") || !strings.Contains(db.lastQuery().query, "migrations") {
		t.Errorf("expected Setup to create migrations table, got %q", db.lastQuery().query)
	}

	// 2. StoreMigration / HasMigration / RemoveMigration
	if err := ed.StoreMigration(ctx, "pgapp", "PGModel", "0001_init.mig"); err != nil {
		t.Fatalf("StoreMigration: %v", err)
	}
	if !strings.Contains(strings.ToUpper(db.lastQuery().query), "INSERT INTO") || len(db.lastQuery().args) != 3 {
		t.Errorf("unexpected StoreMigration query: %+v", db.lastQuery())
	}

	db.rowCount = 1
	has, err := ed.HasMigration(ctx, "pgapp", "PGModel", "0001_init.mig")
	if err != nil || !has {
		t.Fatalf("expected HasMigration=true, got has=%v err=%v", has, err)
	}
	db.rowCount = 0
	has, err = ed.HasMigration(ctx, "pgapp", "PGModel", "0001_init.mig")
	if err != nil || has {
		t.Fatalf("expected HasMigration=false, got has=%v err=%v", has, err)
	}

	if err := ed.RemoveMigration(ctx, "pgapp", "PGModel", "0001_init.mig"); err != nil {
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
	childTbl := migrator.NewModelTable(&pgChildModel{})
	cols := make(map[string]migrator.Column)
	for _, c := range childTbl.Columns() {
		cols[c.Name] = *c
	}

	var colBuf strings.Builder
	ed.WriteColumn(&colBuf, cols["ID"])
	if !strings.Contains(strings.ToUpper(colBuf.String()), "PRIMARY KEY") || !strings.Contains(strings.ToUpper(colBuf.String()), "SERIAL") {
		t.Errorf("expected SERIAL PRIMARY KEY on ID, got %q", colBuf.String())
	}

	colBuf.Reset()
	ed.WriteColumn(&colBuf, cols["Title"])
	if !strings.Contains(colBuf.String(), "DEFAULT 'untitled'") {
		t.Errorf("expected string default on Title, got %q", colBuf.String())
	}

	colBuf.Reset()
	ed.WriteColumn(&colBuf, cols["Code"])
	if !strings.Contains(strings.ToUpper(colBuf.String()), "UNIQUE") {
		t.Errorf("expected UNIQUE on Code, got %q", colBuf.String())
	}

	colBuf.Reset()
	ed.WriteColumn(&colBuf, cols["Parent"])
	if !strings.Contains(colBuf.String(), "REFERENCES") || !strings.Contains(strings.ToUpper(colBuf.String()), "ON DELETE CASCADE") {
		t.Errorf("expected FK constraint on Parent, got %q", colBuf.String())
	}

	// 5. CreateTable / DropTable / RenameTable
	if err := ed.CreateTable(ctx, childTbl, true); err != nil {
		t.Fatalf("CreateTable ifNotExists=true: %v", err)
	}
	if !strings.Contains(strings.ToUpper(db.lastQuery().query), "CREATE TABLE IF NOT EXISTS") || !strings.Contains(db.lastQuery().query, "pg_ddl_child") {
		t.Errorf("unexpected CreateTable SQL: %q", db.lastQuery().query)
	}

	if err := ed.RenameTable(ctx, childTbl, "pg_ddl_child_v2"); err != nil {
		t.Fatalf("RenameTable: %v", err)
	}
	if !strings.Contains(db.lastQuery().query, "pg_ddl_child") || !strings.Contains(db.lastQuery().query, "pg_ddl_child_v2") {
		t.Errorf("unexpected RenameTable SQL: %q", db.lastQuery().query)
	}

	if err := ed.DropTable(ctx, childTbl, true); err != nil {
		t.Fatalf("DropTable: %v", err)
	}
	if !strings.Contains(strings.ToUpper(db.lastQuery().query), "DROP TABLE IF EXISTS") {
		t.Errorf("unexpected DropTable SQL: %q", db.lastQuery().query)
	}

	// 6. AddField / AlterField / RemoveField (cols constructed via NewModelTable have Field != nil)
	if err := ed.AddField(ctx, childTbl, cols["Title"]); err != nil {
		t.Fatalf("AddField: %v", err)
	}
	if !strings.Contains(strings.ToUpper(db.lastQuery().query), "ALTER TABLE") || !strings.Contains(strings.ToUpper(db.lastQuery().query), "ADD COLUMN") {
		t.Errorf("unexpected AddField SQL: %q", db.lastQuery().query)
	}

	newTitle := cols["Title"]
	newTitle.MaxLength = 250
	newTitle.Nullable = false
	if err := ed.AlterField(ctx, childTbl, cols["Title"], newTitle); err != nil {
		t.Fatalf("AlterField: %v", err)
	}
	t.Logf("Postgres AlterField SQL: %s", db.lastQuery().query)
	if !strings.Contains(strings.ToUpper(db.lastQuery().query), "ALTER TABLE") {
		t.Errorf("unexpected AlterField SQL: %q", db.lastQuery().query)
	}

	if err := ed.RemoveField(ctx, childTbl, cols["Title"]); err != nil {
		t.Fatalf("RemoveField: %v", err)
	}
	if !strings.Contains(strings.ToUpper(db.lastQuery().query), "DROP COLUMN") {
		t.Errorf("unexpected RemoveField SQL: %q", db.lastQuery().query)
	}

	// 7. AddIndex / DropIndex
	for _, idx := range childTbl.Indexes() {
		if err := ed.AddIndex(ctx, childTbl, idx, true); err != nil {
			t.Fatalf("AddIndex %s: %v", idx.Name(), err)
		}
		if !strings.Contains(db.lastQuery().query, idx.Name()) {
			t.Errorf("expected index name %q in AddIndex SQL %q", idx.Name(), db.lastQuery().query)
		}
	}
	if err := ed.DropIndex(ctx, childTbl, childTbl.Indexes()[0], true); err != nil {
		t.Fatalf("DropIndex: %v", err)
	}
	if !strings.Contains(strings.ToUpper(db.lastQuery().query), "DROP INDEX") {
		t.Errorf("unexpected DropIndex SQL: %q", db.lastQuery().query)
	}
}

func assertNoPanicPG(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()
	fn()
}

func TestBug_Postgres_RemoveField_NilField(t *testing.T) {
	ctx := context.Background()
	db := &mockPostgresDB{}
	ed := postgres.NewPostgresSchemaEditor(db)
	attrs.RegisterModel(&pgChildModel{})
	childTbl := migrator.NewModelTable(&pgChildModel{})

	col := migrator.Column{Name: "Removed", Column: "removed_col", UseInDB: true} // Field == nil as in MigrationEngine.Migrate ActionRemoveField
	assertNoPanicPG(t, func() {
		if err := ed.RemoveField(ctx, childTbl, col); err != nil {
			t.Fatalf("RemoveField with nil Field failed: %v", err)
		}
	})
	q := db.lastQuery().query
	if !strings.Contains(strings.ToUpper(q), "DROP COLUMN") || !strings.Contains(q, `"removed_col"`) {
		t.Errorf("expected DROP COLUMN \"removed_col\", got %q", q)
	}
}

func TestBug_Postgres_AlterField_NilPtrDefaultExecutes(t *testing.T) {
	ctx := context.Background()
	db := &mockPostgresDB{}
	ed := postgres.NewPostgresSchemaEditor(db)
	attrs.RegisterModel(&pgChildModel{})
	childTbl := migrator.NewModelTable(&pgChildModel{})

	var titleCol migrator.Column
	for _, c := range childTbl.Columns() {
		if c.Name == "Title" {
			titleCol = *c
		}
	}

	oldCol := titleCol
	oldCol.Default = drivers.Value[any]{V: "abc"}
	oldCol.Nullable = false

	newCol := titleCol
	newCol.Default = drivers.Value[any]{V: (*string)(nil)}
	newCol.Nullable = true

	before := len(db.queries)
	if err := ed.AlterField(ctx, childTbl, oldCol, newCol); err != nil {
		t.Fatalf("AlterField failed: %v", err)
	}
	if len(db.queries) != before+1 {
		t.Fatalf("expected AlterField to execute 1 SQL statement, executed %d", len(db.queries)-before)
	}
	q := db.lastQuery().query
	if !strings.Contains(q, "DROP NOT NULL") || (!strings.Contains(q, "DROP DEFAULT") && !strings.Contains(q, "SET DEFAULT NULL")) {
		t.Errorf("expected DROP NOT NULL and default drop/null in SQL, got %q", q)
	}
}

func TestBug_Postgres_AlterField_NoSQLChangeSkipsBareAlterTable(t *testing.T) {
	ctx := context.Background()
	db := &mockPostgresDB{}
	ed := postgres.NewPostgresSchemaEditor(db)
	attrs.RegisterModel(&pgChildModel{})
	childTbl := migrator.NewModelTable(&pgChildModel{})

	var titleCol migrator.Column
	for _, c := range childTbl.Columns() {
		if c.Name == "Title" {
			titleCol = *c
		}
	}

	before := len(db.queries)
	if err := ed.AlterField(ctx, childTbl, titleCol, titleCol); err != nil {
		t.Fatalf("AlterField no-op failed: %v", err)
	}
	if len(db.queries) > before {
		q := strings.TrimSpace(db.lastQuery().query)
		if !strings.Contains(q, "ALTER COLUMN") && !strings.Contains(q, "RENAME COLUMN") {
			t.Errorf("AlterField executed invalid bare SQL with no action clauses: %q", q)
		}
	}
}

func TestBug_Postgres_AlterField_ColumnRename(t *testing.T) {
	ctx := context.Background()
	db := &mockPostgresDB{}
	ed := postgres.NewPostgresSchemaEditor(db)
	attrs.RegisterModel(&pgChildModel{})
	childTbl := migrator.NewModelTable(&pgChildModel{})

	var titleCol migrator.Column
	for _, c := range childTbl.Columns() {
		if c.Name == "Title" {
			titleCol = *c
		}
	}
	oldCol := titleCol
	oldCol.Column = "title"
	newCol := titleCol
	newCol.Column = "headline"

	if err := ed.AlterField(ctx, childTbl, oldCol, newCol); err != nil {
		t.Fatalf("AlterField rename failed: %v", err)
	}
	foundRename := false
	for _, rec := range db.queries {
		if strings.Contains(rec.query, `RENAME COLUMN "title" TO "headline"`) {
			foundRename = true
			break
		}
	}
	if !foundRename {
		t.Errorf("expected Postgres AlterField to emit RENAME COLUMN \"title\" TO \"headline\", queries=%+v", db.queries)
	}
}

func TestBug_Postgres_AlterField_UintAndTimeDefaults(t *testing.T) {
	ctx := context.Background()
	db := &mockPostgresDB{}
	ed := postgres.NewPostgresSchemaEditor(db)
	attrs.RegisterModel(&pgChildModel{})
	childTbl := migrator.NewModelTable(&pgChildModel{})

	var countCol, timeCol migrator.Column
	for _, c := range childTbl.Columns() {
		switch c.Name {
		case "Count":
			countCol = *c
		case "FixedTime":
			timeCol = *c
		}
	}

	newCount := countCol
	newCount.Default = drivers.Value[any]{V: uint64(7)}
	if err := ed.AlterField(ctx, childTbl, countCol, newCount); err != nil {
		t.Errorf("AlterField uint64 default failed: %v", err)
	} else if !strings.Contains(db.lastQuery().query, "SET DEFAULT 7") {
		t.Errorf("expected SET DEFAULT 7, got %q", db.lastQuery().query)
	}

	oldTime := timeCol
	oldTime.Default = drivers.Value[any]{V: time.Time{}}
	newTime := timeCol
	newTime.Default = drivers.Value[any]{V: time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)}
	if err := ed.AlterField(ctx, childTbl, oldTime, newTime); err != nil {
		t.Errorf("AlterField time.Time default failed: %v", err)
	} else if !strings.Contains(db.lastQuery().query, "SET DEFAULT '2025-01-02 03:04:05'") {
		t.Errorf("expected formatted timestamp default in SET DEFAULT, got %q", db.lastQuery().query)
	}
}

func TestBug_Postgres_DefaultSingleQuoteEscaping(t *testing.T) {
	ctx := context.Background()
	db := &mockPostgresDB{}
	ed := postgres.NewPostgresSchemaEditor(db)
	attrs.RegisterModel(&pgChildModel{})
	childTbl := migrator.NewModelTable(&pgChildModel{})

	var titleCol migrator.Column
	for _, c := range childTbl.Columns() {
		if c.Name == "Title" {
			titleCol = *c
		}
	}
	quotedCol := titleCol
	quotedCol.Default = drivers.Value[any]{V: "it's"}

	var buf strings.Builder
	ed.WriteColumn(&buf, quotedCol)
	if !strings.Contains(buf.String(), "DEFAULT 'it''s'") {
		t.Errorf("WriteColumn did not escape single quote in default: %q", buf.String())
	}

	if err := ed.AlterField(ctx, childTbl, titleCol, quotedCol); err != nil {
		t.Fatalf("AlterField failed: %v", err)
	}
	if !strings.Contains(db.lastQuery().query, "SET DEFAULT 'it''s'") {
		t.Errorf("AlterField did not escape single quote in default: %q", db.lastQuery().query)
	}
}

type pgConstraintModel struct {
	models.Model `table:"pg_ddl_constrained"`
	ID           int64
	Age          int64
	Email        string
	Parent       *pgParentModel
}

func (m *pgConstraintModel) Constraints() []constraints.Constraint {
	return []constraints.Constraint{
		constraints.CheckConstraint("chk_pg_age", "too young", expr.Q("Age__gte", expr.V(18, true))),
		constraints.UniqueConstraint("uq_pg_email", "dup email", "Email"),
		constraints.ForeignKeyConstraint("fk_pg_parent", "bad parent", "Parent", constraints.A_CASCADE, constraints.A_SET_NULL),
	}
}

func (m *pgConstraintModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true, Column: "id"}),
		attrs.NewField(m, "Age", &attrs.Config{Column: "age"}),
		attrs.NewField(m, "Email", &attrs.Config{Column: "email"}),
		attrs.NewField(m, "Parent", &attrs.Config{
			Column:        "parent_id",
			Null:          true,
			ReverseAlias:  "PGConstrainedChildren",
			RelForeignKey: attrs.Relate(&pgParentModel{}, "", nil),
		}),
	)
}

func TestPostgresSchemaEditor_Constraints_DDL(t *testing.T) {
	ctx := context.Background()
	db := &mockPostgresDB{}
	ed := postgres.NewPostgresSchemaEditor(db)

	prevGlobal := django.Global
	t.Cleanup(func() { django.Global = prevGlobal })
	django.Global = nil
	django.App(
		django.Configure(map[string]any{django.APPVAR_DATABASE: db}),
		django.Flag(django.FlagSkipCmds, django.FlagSkipDepsCheck, django.FlagSkipChecks),
	)

	attrs.RegisterModel(&pgParentModel{})
	attrs.RegisterModel(&pgConstraintModel{})
	tbl := migrator.NewModelTable(&pgConstraintModel{})

	if err := ed.CreateTable(ctx, tbl, false); err != nil {
		t.Fatalf("CreateTable: %v", err)
	}
	createQ := db.lastQuery().query
	if strings.Contains(createQ, "REFERENCES") {
		t.Errorf("expected Postgres CreateTable to skip inline REFERENCES when ForeignKeyConstraint exists, got:\n%s", createQ)
	}

	res := resolver.New(ctx, &pgConstraintModel{}, queries.Compiler(django.APPVAR_DATABASE))
	cs := tbl.Constraints()

	if err := ed.AddConstraint(ctx, tbl, res.ResolveExpression(cs[0]).(constraints.Constraint)); err != nil {
		t.Fatalf("AddConstraint check: %v", err)
	}
	if want := `ALTER TABLE "pg_ddl_constrained" ADD CONSTRAINT "chk_pg_age" CHECK ("age" >= 18);`; db.lastQuery().query != want {
		t.Errorf("AddConstraint(Check):\n  got:  %s\n  want: %s", db.lastQuery().query, want)
	}

	if err := ed.AddConstraint(ctx, tbl, res.ResolveExpression(cs[1]).(constraints.Constraint)); err != nil {
		t.Fatalf("AddConstraint unique: %v", err)
	}
	if want := `ALTER TABLE "pg_ddl_constrained" ADD CONSTRAINT "uq_pg_email" UNIQUE ("email");`; db.lastQuery().query != want {
		t.Errorf("AddConstraint(Unique):\n  got:  %s\n  want: %s", db.lastQuery().query, want)
	}

	if err := ed.AddConstraint(ctx, tbl, res.ResolveExpression(cs[2]).(constraints.Constraint)); err != nil {
		t.Fatalf("AddConstraint fk: %v", err)
	}
	if want := `ALTER TABLE "pg_ddl_constrained" ADD CONSTRAINT "fk_pg_parent" FOREIGN KEY ("parent_id") REFERENCES "pg_ddl_parent" ("id") ON DELETE SET NULL ON UPDATE CASCADE;`; db.lastQuery().query != want {
		t.Errorf("AddConstraint(FK):\n  got:  %s\n  want: %s", db.lastQuery().query, want)
	}

	for _, c := range cs {
		if err := ed.DropConstraint(ctx, tbl, c); err != nil {
			t.Fatalf("DropConstraint(%s): %v", c.ID(), err)
		}
		want := fmt.Sprintf(`ALTER TABLE "pg_ddl_constrained" DROP CONSTRAINT IF EXISTS "%s";`, c.ID())
		if db.lastQuery().query != want {
			t.Errorf("DropConstraint(%s):\n  got:  %s\n  want: %s", c.ID(), db.lastQuery().query, want)
		}
	}

	if err := ed.AddConstraint(ctx, tbl, nil); !errors.Is(err, derrors.NilPointer) {
		t.Errorf("expected NilPointer for AddConstraint(nil), got %v", err)
	}
	if err := ed.DropConstraint(ctx, tbl, nil); !errors.Is(err, derrors.NilPointer) {
		t.Errorf("expected NilPointer for DropConstraint(nil), got %v", err)
	}
	if err := ed.DropConstraint(ctx, tbl, constraints.CheckConstraint("", "", nil)); !errors.Is(err, derrors.ValueError) {
		t.Errorf("expected ValueError for DropConstraint empty ID, got %v", err)
	}
	if err := ed.AddConstraint(ctx, tbl, cs[0]); err == nil {
		t.Errorf("expected error adding unresolved constraint")
	}
}
