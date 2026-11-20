package sqlite_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	derrors "github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr/constraints"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/queries/src/migrator/sql/sqlite"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

var sqliteTestSeq atomic.Int64

func openSQLiteSchemaTestDB(t *testing.T) (drivers.Database, *sqlite.SQLiteSchemaEditor) {
	t.Helper()
	dsn := fmt.Sprintf("file:sqlite_schema_test_%d?mode=memory&cache=shared", sqliteTestSeq.Add(1))
	db, err := drivers.Open(context.Background(), "sqlite3", dsn)
	if err != nil {
		t.Fatalf("failed to open sqlite db %q: %v", dsn, err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	ed := sqlite.NewSQLiteSchemaEditor(db)
	if err := ed.Setup(context.Background()); err != nil {
		t.Fatalf("editor.Setup failed: %v", err)
	}
	return db, ed
}

type sqliteTypeHolder struct {
	ID  int64 `attrs:"primary"`
	Val string
}

func (m *sqliteTypeHolder) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions(ctx, m)
}

type overrideDBTypeField struct {
	attrs.Field
	dbType dbtype.Type
}

func (f *overrideDBTypeField) DBType(*migrator.Column) dbtype.Type {
	return f.dbType
}

func TestSQLite_AllDBTypes(t *testing.T) {
	attrs.RegisterModel(&sqliteTypeHolder{})
	holder := &sqliteTypeHolder{}
	baseField := attrs.NewField(holder, "Val", &attrs.Config{Column: "val"})

	drv := &drivers.DriverSQLite{}
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
		{"String_MaxLen", dbtype.String, 128, 0, 0, false},
		{"Char", dbtype.Char, 10, 0, 0, false},
		{"Int", dbtype.Int, 0, 0, 0, false},
		{"Int_Auto", dbtype.Int, 0, 0, 0, true},
		{"Uint", dbtype.Uint, 0, 0, 0, false},
		{"Float", dbtype.Float, 0, 0, 0, false},
		{"Decimal_Default", dbtype.Decimal, 0, 0, 0, false},
		{"Decimal_PrecScale", dbtype.Decimal, 0, 12, 4, false},
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
				Field:     &overrideDBTypeField{Field: baseField, dbType: tc.dbType},
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
				t.Fatalf("empty SQLite type for %s (%v)", tc.name, tc.dbType)
			}
			if tc.dbType == dbtype.Decimal && tc.precision == 12 && tc.scale == 4 {
				if !strings.Contains(strings.ToUpper(sqlType), "12") || !strings.Contains(sqlType, "4") {
					t.Errorf("expected Decimal(12,4) in SQLite type, got %q", sqlType)
				}
			}
			t.Logf("%s -> %s", tc.name, sqlType)
		})
	}
}

type sqliteValuerDefault string

func (v sqliteValuerDefault) Value() (driver.Value, error) {
	return "valuer_" + string(v), nil
}

type sqliteParentModel struct {
	ID   int64 `attrs:"primary"`
	Name string
}

func (m *sqliteParentModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(m, "Name"),
	).WithTableName("ddl_test_parent")
}

type sqliteItemModel struct {
	ID        int64 `attrs:"primary"`
	Title     string
	Code      string
	Count     int64
	Ratio     float64
	Active    bool
	ZeroTime  time.Time
	FixedTime time.Time
	CustomDef sqliteValuerDefault
	Parent    *sqliteParentModel
	Note      sql.NullString
}

var sqliteItemAlterMode atomic.Int32

func (m *sqliteItemModel) DatabaseIndexes(attrs.Definer) []migrator.Index {
	return []migrator.Index{
		{Identifier: "ddl_test_item_title_idx", Fields: []string{"Title"}, Unique: false},
		{Identifier: "ddl_test_item_code_uniq", Fields: []string{"Code"}, Unique: true},
	}
}

func (m *sqliteItemModel) FieldDefs(ctx context.Context) attrs.Definitions {
	ratioCfg := &attrs.Config{Null: true}
	noteCol := "note"
	if sqliteItemAlterMode.Load() == 1 {
		ratioCfg = &attrs.Config{Null: false}
	} else if sqliteItemAlterMode.Load() == 2 {
		ratioCfg = &attrs.Config{Null: false}
		noteCol = "note_renamed"
	}

	return attrs.Make(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(m, "Title", &attrs.Config{Default: "untitled"}),
		attrs.NewField(m, "Code", &attrs.Config{Attributes: map[string]any{attrs.AttrUniqueKey: true}}),
		attrs.NewField(m, "Count", &attrs.Config{Default: int64(7)}),
		attrs.NewField(m, "Ratio", ratioCfg),
		attrs.NewField(m, "Active", &attrs.Config{Default: true}),
		attrs.NewField(m, "ZeroTime", &attrs.Config{}),
		attrs.NewField(m, "FixedTime", &attrs.Config{Default: time.Date(2025, 2, 3, 4, 5, 6, 0, time.UTC)}),
		attrs.NewField(m, "CustomDef", &attrs.Config{Default: sqliteValuerDefault("ok")}),
		attrs.NewField(m, "Parent", &attrs.Config{
			Column:        "parent_id",
			Null:          true,
			RelForeignKey: attrs.Relate(&sqliteParentModel{}, "", nil),
			Attributes: map[string]any{
				migrator.AttrOnDeleteKey: constraints.A_CASCADE,
				migrator.AttrOnUpdateKey: constraints.A_SET_NULL,
			},
		}),
		attrs.NewField(m, "Note", &attrs.Config{Column: noteCol, Null: true}),
	).WithTableName("ddl_test_item")
}

func TestSQLiteSchemaEditor_FullDDL(t *testing.T) {
	ctx := context.Background()
	db, ed := openSQLiteSchemaTestDB(t)

	sqliteItemAlterMode.Store(0)
	t.Cleanup(func() { sqliteItemAlterMode.Store(0) })

	attrs.RegisterModel(&sqliteParentModel{})
	attrs.RegisterModel(&sqliteItemModel{})

	// 1. Migrations tracking table lifecycle (migrations table created by ed.Setup)
	has, err := ed.HasMigration(ctx, "sqlapp", "Model", "0001_init.mig")
	if err != nil || has {
		t.Fatalf("expected HasMigration=false before store, got has=%v err=%v", has, err)
	}
	if err := ed.StoreMigration(ctx, "sqlapp", "Model", "0001_init.mig"); err != nil {
		t.Fatalf("StoreMigration: %v", err)
	}
	has, err = ed.HasMigration(ctx, "sqlapp", "Model", "0001_init.mig")
	if err != nil || !has {
		t.Fatalf("expected HasMigration=true after store, got has=%v err=%v", has, err)
	}
	if err := ed.RemoveMigration(ctx, "sqlapp", "Model", "0001_init.mig"); err != nil {
		t.Fatalf("RemoveMigration: %v", err)
	}
	has, err = ed.HasMigration(ctx, "sqlapp", "Model", "0001_init.mig")
	if err != nil || has {
		t.Fatalf("expected HasMigration=false after remove, got has=%v err=%v", has, err)
	}

	// 2. WriteColumn checks for defaults, constraints, and FKs
	itemTbl := migrator.NewModelTable(&sqliteItemModel{})
	colsByName := make(map[string]migrator.Column)
	for _, c := range itemTbl.Columns() {
		colsByName[c.Name] = *c
	}

	var colBuf strings.Builder
	ed.WriteColumn(&colBuf, colsByName["Title"])
	if !strings.Contains(colBuf.String(), "DEFAULT 'untitled'") {
		t.Errorf("expected string default on Title, got %q", colBuf.String())
	}

	colBuf.Reset()
	ed.WriteColumn(&colBuf, colsByName["FixedTime"])
	if !strings.Contains(colBuf.String(), "DEFAULT '2025-02-03 04:05:06'") {
		t.Errorf("expected formatted timestamp default on FixedTime, got %q", colBuf.String())
	}

	colBuf.Reset()
	zeroCol := colsByName["ZeroTime"]
	zeroCol.Default = drivers.Value[any]{V: time.Time{}}
	// Ensure zero time does not panic and emits clean SQL even when HasDefault() is false.
	ed.WriteColumn(&colBuf, zeroCol)
	if !strings.Contains(colBuf.String(), "`zero_time`") {
		t.Errorf("expected zero_time column DDL, got %q", colBuf.String())
	}

	colBuf.Reset()
	ed.WriteColumn(&colBuf, colsByName["CustomDef"])
	if !strings.Contains(colBuf.String(), "DEFAULT 'valuer_ok'") {
		t.Errorf("expected driver.Valuer default on CustomDef, got %q", colBuf.String())
	}

	colBuf.Reset()
	ed.WriteColumn(&colBuf, colsByName["Parent"])
	if !strings.Contains(colBuf.String(), "REFERENCES `ddl_test_parent` (`id`) ON DELETE CASCADE ON UPDATE SET NULL") {
		t.Errorf("expected FK constraint clauses on Parent, got %q", colBuf.String())
	}

	// 3. CreateTable (with and without ifNotExists)
	parentTbl := migrator.NewModelTable(&sqliteParentModel{})
	if err := ed.CreateTable(ctx, parentTbl, false); err != nil {
		t.Fatalf("CreateTable parent: %v", err)
	}
	if err := ed.CreateTable(ctx, parentTbl, true); err != nil {
		t.Fatalf("CreateTable parent ifNotExists=true: %v", err)
	}
	if err := ed.CreateTable(ctx, parentTbl, false); err == nil {
		t.Errorf("expected CreateTable parent ifNotExists=false on existing table to fail")
	}

	if err := ed.CreateTable(ctx, itemTbl, false); err != nil {
		t.Fatalf("CreateTable item: %v", err)
	}

	// 4. AddIndex (unique + non-unique, ifNotExists) & DropIndex
	idx := itemTbl.Indexes()[0]
	if err := ed.AddIndex(ctx, itemTbl, idx, false); err != nil {
		t.Fatalf("AddIndex: %v", err)
	}
	if err := ed.AddIndex(ctx, itemTbl, idx, true); err != nil {
		t.Fatalf("AddIndex ifNotExists=true: %v", err)
	}
	uniqIdx := itemTbl.Indexes()[1]
	if err := ed.AddIndex(ctx, itemTbl, uniqIdx, false); err != nil {
		t.Fatalf("AddIndex unique: %v", err)
	}

	// Insert a parent & an item row with ratio=NULL and note=NULL to test AlterField COALESCE
	if _, err := db.ExecContext(ctx, "INSERT INTO ddl_test_parent (name) VALUES ('p1')"); err != nil {
		t.Fatalf("insert parent: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO ddl_test_item (title, code, count, ratio, active, zero_time, fixed_time, custom_def, parent_id, note) VALUES ('t1', 'c1', 3, NULL, 1, '2025-01-01', '2025-01-01', 'x', 1, 'hello')"); err != nil {
		t.Fatalf("insert item: %v", err)
	}

	// 5. AlterField: nullable Ratio -> NOT NULL (COALESCE) and rename Note -> note_renamed, preserving indexes
	oldRatio := colsByName["Ratio"]
	oldNote := colsByName["Note"]

	sqliteItemAlterMode.Store(1)
	_ = attrs.ResetDefinitions.Send(ctx, nil)
	itemTblAfterRatio := migrator.NewModelTable(&sqliteItemModel{})
	var newRatio migrator.Column
	for _, c := range itemTblAfterRatio.Columns() {
		if c.Name == "Ratio" {
			newRatio = *c
		}
	}
	if err := ed.AlterField(ctx, itemTblAfterRatio, oldRatio, newRatio); err != nil {
		t.Fatalf("AlterField Ratio (NULL -> NOT NULL): %v", err)
	}

	var coercedRatio float64
	if err := db.QueryRowContext(ctx, "SELECT ratio FROM ddl_test_item WHERE id = 1").Scan(&coercedRatio); err != nil {
		t.Fatalf("select ratio after AlterField: %v", err)
	}
	if coercedRatio != 0.0 {
		t.Errorf("expected NULL ratio coerced to 0.0, got %v", coercedRatio)
	}

	sqliteItemAlterMode.Store(2)
	_ = attrs.ResetDefinitions.Send(ctx, nil)
	itemTblRenamed := migrator.NewModelTable(&sqliteItemModel{})
	var newNote migrator.Column
	for _, c := range itemTblRenamed.Columns() {
		if c.Name == "Note" {
			newNote = *c
		}
	}
	if err := ed.AlterField(ctx, itemTblRenamed, oldNote, newNote); err != nil {
		t.Fatalf("AlterField Note rename: %v", err)
	}

	var noteVal string
	if err := db.QueryRowContext(ctx, "SELECT note_renamed FROM ddl_test_item WHERE id = 1").Scan(&noteVal); err != nil || noteVal != "hello" {
		t.Errorf("expected note_renamed='hello', got %q err=%v", noteVal, err)
	}

	// Verify indexes survived both AlterField rebuilds
	rows, err := db.QueryContext(ctx, "PRAGMA index_list(`ddl_test_item`)")
	if err != nil {
		t.Fatalf("PRAGMA index_list: %v", err)
	}
	foundIdxs := map[string]bool{}
	for rows.Next() {
		var seq int
		var name string
		var uniq int
		var origin string
		var partial int
		if err := rows.Scan(&seq, &name, &uniq, &origin, &partial); err != nil {
			t.Fatal(err)
		}
		foundIdxs[name] = true
	}
	_ = rows.Close()
	if !foundIdxs["ddl_test_item_title_idx"] || !foundIdxs["ddl_test_item_code_uniq"] {
		t.Errorf("expected indexes preserved after AlterField, got %v", foundIdxs)
	}

	// 6. AddField & RemoveField
	extraCol := newNote
	extraCol.Name = "ExtraCol"
	extraCol.Column = "extra_col"
	extraCol.Nullable = true
	if err := ed.AddField(ctx, itemTblRenamed, extraCol); err != nil {
		t.Fatalf("AddField: %v", err)
	}
	if err := ed.RemoveField(ctx, itemTblRenamed, extraCol); err != nil {
		t.Fatalf("RemoveField: %v", err)
	}

	// 7. DropIndex (ifExists false & true)
	if err := ed.DropIndex(ctx, itemTblRenamed, uniqIdx, false); err != nil {
		t.Fatalf("DropIndex: %v", err)
	}
	if err := ed.DropIndex(ctx, itemTblRenamed, uniqIdx, true); err != nil {
		t.Fatalf("DropIndex ifExists=true: %v", err)
	}

	// 8. RenameTable & DropTable
	if err := ed.RenameTable(ctx, itemTblRenamed, "ddl_test_item_v2"); err != nil {
		t.Fatalf("RenameTable: %v", err)
	}
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM ddl_test_item_v2").Scan(&count); err != nil || count != 1 {
		t.Errorf("expected 1 row in renamed table, got count=%d err=%v", count, err)
	}
	renamedTbl := &migrator.ModelTable{Table: "ddl_test_item_v2"}
	if err := ed.DropTable(ctx, renamedTbl, false); err != nil {
		t.Fatalf("DropTable: %v", err)
	}
	if err := ed.DropTable(ctx, renamedTbl, true); err != nil {
		t.Fatalf("DropTable ifExists=true: %v", err)
	}

	// 9. AddConstraint / DropConstraint return NotImplemented
	if err := ed.AddConstraint(ctx, parentTbl, nil); !errors.Is(err, derrors.NotImplemented) {
		t.Errorf("expected NotImplemented from AddConstraint, got %v", err)
	}
	if err := ed.DropConstraint(ctx, parentTbl, nil); !errors.Is(err, derrors.NotImplemented) {
		t.Errorf("expected NotImplemented from DropConstraint, got %v", err)
	}
}

type sqliteIdxCommentModel struct {
	ID   int64 `attrs:"primary"`
	Name string
}

func (m *sqliteIdxCommentModel) DatabaseIndexes(attrs.Definer) []migrator.Index {
	return []migrator.Index{
		{Identifier: "idx_commented", Fields: []string{"Name"}, Comment: "index comment"},
	}
}

func (m *sqliteIdxCommentModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(m, "Name"),
	).WithTableName("sqlite_idx_comment_tbl")
}

func TestBug_SQLite_AddIndexWithComment(t *testing.T) {
	ctx := context.Background()
	_, ed := openSQLiteSchemaTestDB(t)
	attrs.RegisterModel(&sqliteIdxCommentModel{})

	tbl := migrator.NewModelTable(&sqliteIdxCommentModel{})
	if err := ed.CreateTable(ctx, tbl, false); err != nil {
		t.Fatalf("CreateTable: %v", err)
	}
	if err := ed.AddIndex(ctx, tbl, tbl.Indexes()[0], false); err != nil {
		t.Fatalf("AddIndex with comment failed on SQLite: %v", err)
	}
}

type sqliteRenIdxModel struct {
	ID   int64 `attrs:"primary"`
	Name string
}

func (m *sqliteRenIdxModel) DatabaseIndexes(attrs.Definer) []migrator.Index {
	return []migrator.Index{
		{Identifier: "idx_before", Fields: []string{"Name"}},
	}
}

func (m *sqliteRenIdxModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(m, "Name"),
	).WithTableName("sqlite_ren_idx_tbl")
}

func TestBug_SQLite_RenameIndex(t *testing.T) {
	ctx := context.Background()
	db, ed := openSQLiteSchemaTestDB(t)
	attrs.RegisterModel(&sqliteRenIdxModel{})

	tbl := migrator.NewModelTable(&sqliteRenIdxModel{})
	if err := ed.CreateTable(ctx, tbl, false); err != nil {
		t.Fatalf("CreateTable: %v", err)
	}
	if err := ed.AddIndex(ctx, tbl, tbl.Indexes()[0], false); err != nil {
		t.Fatalf("AddIndex: %v", err)
	}

	if err := ed.RenameIndex(ctx, tbl, "idx_before", "idx_after"); err != nil {
		t.Fatalf("RenameIndex failed on SQLite: %v", err)
	}

	rows, err := db.QueryContext(ctx, "PRAGMA index_list(`sqlite_ren_idx_tbl`)")
	if err != nil {
		t.Fatalf("PRAGMA index_list: %v", err)
	}
	defer rows.Close()

	idxs := map[string]bool{}
	for rows.Next() {
		var seq int
		var name, origin string
		var uniq, partial int
		if err := rows.Scan(&seq, &name, &uniq, &origin, &partial); err != nil {
			t.Fatal(err)
		}
		idxs[name] = true
	}
	if !idxs["idx_after"] {
		t.Errorf("expected idx_after in PRAGMA index_list, got %v", idxs)
	}
	if idxs["idx_before"] {
		t.Errorf("expected idx_before removed, got %v", idxs)
	}
}

type sqliteQuoteDefaultModel struct {
	ID   int64 `attrs:"primary"`
	Note string
}

func (m *sqliteQuoteDefaultModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(m, "Note", &attrs.Config{Default: "it's ok"}),
	).WithTableName("sqlite_quote_dflt_tbl")
}

func TestBug_SQLite_DefaultSingleQuoteEscaping(t *testing.T) {
	ctx := context.Background()
	db, ed := openSQLiteSchemaTestDB(t)
	attrs.RegisterModel(&sqliteQuoteDefaultModel{})

	tbl := migrator.NewModelTable(&sqliteQuoteDefaultModel{})
	if err := ed.CreateTable(ctx, tbl, false); err != nil {
		t.Fatalf("CreateTable with single-quote default failed: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO sqlite_quote_dflt_tbl (id) VALUES (1)"); err != nil {
		t.Fatalf("INSERT using default failed: %v", err)
	}
	var got string
	if err := db.QueryRowContext(ctx, "SELECT note FROM sqlite_quote_dflt_tbl WHERE id = 1").Scan(&got); err != nil {
		t.Fatalf("SELECT note: %v", err)
	}
	if got != "it's ok" {
		t.Errorf("expected \"it's ok\", got %q", got)
	}
}
