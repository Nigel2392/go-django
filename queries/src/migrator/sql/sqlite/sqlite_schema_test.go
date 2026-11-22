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

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	derrors "github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/expr/constraints"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/queries/src/migrator/sql/sqlite"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/queries/src/resolver"
	django "github.com/Nigel2392/go-django/src"
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

	// 9. AddConstraint / DropConstraint reject nil constraints cleanly
	if err := ed.AddConstraint(ctx, parentTbl, nil); !errors.Is(err, derrors.NilPointer) {
		t.Errorf("expected NilPointer from AddConstraint(nil), got %v", err)
	}
	if err := ed.DropConstraint(ctx, parentTbl, nil); !errors.Is(err, derrors.NilPointer) {
		t.Errorf("expected NilPointer from DropConstraint(nil), got %v", err)
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
	).WithTableName("ddl_idx_comment_tbl")
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
	).WithTableName("ddl_ren_idx_tbl")
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

	rows, err := db.QueryContext(ctx, "PRAGMA index_list(`ddl_ren_idx_tbl`)")
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
	).WithTableName("ddl_quote_dflt_tbl")
}

func TestBug_SQLite_DefaultSingleQuoteEscaping(t *testing.T) {
	ctx := context.Background()
	db, ed := openSQLiteSchemaTestDB(t)
	attrs.RegisterModel(&sqliteQuoteDefaultModel{})

	tbl := migrator.NewModelTable(&sqliteQuoteDefaultModel{})
	if err := ed.CreateTable(ctx, tbl, false); err != nil {
		t.Fatalf("CreateTable with single-quote default failed: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO ddl_quote_dflt_tbl (id) VALUES (1)"); err != nil {
		t.Fatalf("INSERT using default failed: %v", err)
	}
	var got string
	if err := db.QueryRowContext(ctx, "SELECT note FROM ddl_quote_dflt_tbl WHERE id = 1").Scan(&got); err != nil {
		t.Fatalf("SELECT note: %v", err)
	}
	if got != "it's ok" {
		t.Errorf("expected \"it's ok\", got %q", got)
	}
}

type sqliteConstraintParent struct {
	models.Model `table:"ddl_c_parent"`
	ID           int64
	Label        string
}

func (m *sqliteConstraintParent) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true, Column: "id"}),
		attrs.NewField(m, "Label", &attrs.Config{Column: "label"}),
	)
}

type sqliteConstraintChild struct {
	models.Model `table:"ddl_c_child"`
	ID           int64
	Age          int64
	Code         string
	Slug         string
	Note         string
	Parent       *sqliteConstraintParent
}

func (m *sqliteConstraintChild) Constraints() []constraints.Constraint {
	return []constraints.Constraint{
		constraints.CheckConstraint("chk_age_min", "too young", expr.Q("Age__gte", expr.V(18, true))),
		constraints.CheckConstraint("chk_note_comma", "reserved note", expr.Q("Note__not", expr.V("bad, (val)", true))),
		constraints.UniqueConstraint("uq_code_slug", "duplicate pair", "Code", "Slug"),
		constraints.ForeignKeyConstraint("fk_parent_id", "invalid parent", "Parent", constraints.A_CASCADE, constraints.A_CASCADE),
	}
}

func (m *sqliteConstraintChild) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true, Column: "id"}),
		attrs.NewField(m, "Age", &attrs.Config{Column: "age"}),
		attrs.NewField(m, "Code", &attrs.Config{Column: "code", MaxLength: 32}),
		attrs.NewField(m, "Slug", &attrs.Config{Column: "slug", MaxLength: 32}),
		attrs.NewField(m, "Note", &attrs.Config{Column: "note", Default: "ok"}),
		attrs.NewField(m, "Parent", &attrs.Config{
			Column:        "parent_id",
			Null:          true,
			RelForeignKey: attrs.Relate(&sqliteConstraintParent{}, "", nil),
		}),
	)
}

type sqliteConstraintRestrict struct {
	models.Model `table:"ddl_c_restrict"`
	ID           int64
	Parent       *sqliteConstraintParent
}

func (m *sqliteConstraintRestrict) Constraints() []constraints.Constraint {
	return []constraints.Constraint{
		constraints.ForeignKeyConstraint("fk_res_parent", "", "Parent", constraints.A_RESTRICT, constraints.A_RESTRICT),
	}
}

func (m *sqliteConstraintRestrict) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true, Column: "id"}),
		attrs.NewField(m, "Parent", &attrs.Config{
			Column:        "parent_id",
			ReverseAlias:  "RestrictChildren",
			RelForeignKey: attrs.Relate(&sqliteConstraintParent{}, "", nil),
		}),
	)
}

type sqliteConstraintSetNull struct {
	models.Model `table:"ddl_c_setnull"`
	ID           int64
	Parent       *sqliteConstraintParent
}

func (m *sqliteConstraintSetNull) Constraints() []constraints.Constraint {
	return []constraints.Constraint{
		constraints.ForeignKeyConstraint("fk_sn_parent", "", "Parent", constraints.A_CASCADE, constraints.A_SET_NULL),
	}
}

func (m *sqliteConstraintSetNull) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true, Column: "id"}),
		attrs.NewField(m, "Parent", &attrs.Config{
			Column:        "parent_id",
			Null:          true,
			ReverseAlias:  "SetNullChildren",
			RelForeignKey: attrs.Relate(&sqliteConstraintParent{}, "", nil),
		}),
	)
}

func TestSQLiteSchemaEditor_Constraints_LiveEnforcement(t *testing.T) {
	ctx := context.Background()
	testDB, ed := openSQLiteSchemaTestDB(t)
	if _, err := testDB.ExecContext(ctx, "PRAGMA foreign_keys = ON;"); err != nil {
		t.Fatalf("enable foreign_keys: %v", err)
	}

	prevGlobal := django.Global
	t.Cleanup(func() { django.Global = prevGlobal })
	django.Global = nil
	django.App(
		django.Configure(map[string]any{django.APPVAR_DATABASE: testDB}),
		django.Flag(django.FlagSkipCmds, django.FlagSkipDepsCheck, django.FlagSkipChecks),
	)

	attrs.RegisterModel(&sqliteConstraintParent{})
	attrs.RegisterModel(&sqliteConstraintChild{})
	attrs.RegisterModel(&sqliteConstraintRestrict{})
	attrs.RegisterModel(&sqliteConstraintSetNull{})

	parentTbl := migrator.NewModelTable(&sqliteConstraintParent{})
	childTbl := migrator.NewModelTable(&sqliteConstraintChild{})
	restrictTbl := migrator.NewModelTable(&sqliteConstraintRestrict{})
	setNullTbl := migrator.NewModelTable(&sqliteConstraintSetNull{})

	for _, tbl := range []*migrator.ModelTable{parentTbl, childTbl, restrictTbl, setNullTbl} {
		if err := ed.CreateTable(ctx, tbl, false); err != nil {
			t.Fatalf("CreateTable(%s): %v", tbl.TableName(), err)
		}
	}

	comp := queries.Compiler(django.APPVAR_DATABASE)

	// Verify CreateTable skipped unnamed inline REFERENCES on ddl_c_child.parent_id
	var createSQLBefore string
	if err := testDB.QueryRowContext(ctx, "SELECT sql FROM sqlite_schema WHERE type='table' AND name='ddl_c_child'").Scan(&createSQLBefore); err != nil {
		t.Fatalf("fetch create sql: %v", err)
	}
	if strings.Contains(createSQLBefore, "REFERENCES") {
		t.Errorf("expected CreateTable to skip inline REFERENCES when explicit ForeignKeyConstraint exists, got:\n%s", createSQLBefore)
	}

	childRes := resolver.New(ctx, &sqliteConstraintChild{}, comp)
	for _, c := range childTbl.Constraints() {
		resolved := childRes.ResolveExpression(c).(constraints.Constraint)
		if err := ed.AddConstraint(ctx, childTbl, resolved); err != nil {
			t.Fatalf("AddConstraint(%s): %v", c.ID(), err)
		}
	}

	restrictRes := resolver.New(ctx, &sqliteConstraintRestrict{}, comp)
	for _, c := range restrictTbl.Constraints() {
		resolved := restrictRes.ResolveExpression(c).(constraints.Constraint)
		if err := ed.AddConstraint(ctx, restrictTbl, resolved); err != nil {
			t.Fatalf("AddConstraint(%s): %v", c.ID(), err)
		}
	}

	setNullRes := resolver.New(ctx, &sqliteConstraintSetNull{}, comp)
	for _, c := range setNullTbl.Constraints() {
		resolved := setNullRes.ResolveExpression(c).(constraints.Constraint)
		if err := ed.AddConstraint(ctx, setNullTbl, resolved); err != nil {
			t.Fatalf("AddConstraint(%s): %v", c.ID(), err)
		}
	}

	// Duplicate ID must error
	dupChk := childRes.ResolveExpression(constraints.CheckConstraint("chk_age_min", "", expr.Q("Age__gte", expr.V(0, true)))).(constraints.Constraint)
	if err := ed.AddConstraint(ctx, childTbl, dupChk); err == nil {
		t.Errorf("expected duplicate constraint error for chk_age_min")
	}

	// Unresolved & bound-parameter checks must error
	if err := ed.AddConstraint(ctx, childTbl, constraints.CheckConstraint("chk_unres", "", expr.Q("Age__gt", expr.V(0, true)))); err == nil {
		t.Errorf("expected error for unresolved constraint")
	}
	boundChk := childRes.ResolveExpression(constraints.CheckConstraint("chk_bound", "", expr.Q("Age__gt", 0))).(constraints.Constraint)
	if err := ed.AddConstraint(ctx, childTbl, boundChk); err == nil || !strings.Contains(err.Error(), "bound parameters") {
		t.Errorf("expected bound parameters error, got %v", err)
	}

	// Live enforcement
	if _, err := testDB.ExecContext(ctx, "INSERT INTO ddl_c_parent (id, label) VALUES (1, 'p1'), (2, 'p2'), (3, 'p3')"); err != nil {
		t.Fatalf("insert parents: %v", err)
	}
	if _, err := testDB.ExecContext(ctx, "INSERT INTO ddl_c_child (id, age, code, slug, note, parent_id) VALUES (1, 17, 'C1', 's1', 'ok', 1)"); err == nil {
		t.Errorf("expected CHECK chk_age_min to reject age=17")
	}
	if _, err := testDB.ExecContext(ctx, "INSERT INTO ddl_c_child (id, age, code, slug, note, parent_id) VALUES (1, 20, 'C1', 's1', 'bad, (val)', 1)"); err == nil {
		t.Errorf("expected CHECK chk_note_comma to reject note='bad, (val)'")
	}
	if _, err := testDB.ExecContext(ctx, "INSERT INTO ddl_c_child (id, age, code, slug, note, parent_id) VALUES (1, 20, 'C1', 's1', 'ok', 999)"); err == nil {
		t.Errorf("expected FK fk_parent_id to reject missing parent_id=999")
	}
	if _, err := testDB.ExecContext(ctx, "INSERT INTO ddl_c_child (id, age, code, slug, note, parent_id) VALUES (1, 20, 'C1', 's1', 'ok', 1)"); err != nil {
		t.Fatalf("valid insert failed: %v", err)
	}
	if _, err := testDB.ExecContext(ctx, "INSERT INTO ddl_c_child (id, age, code, slug, note, parent_id) VALUES (2, 22, 'C1', 's1', 'ok', 1)"); err == nil {
		t.Errorf("expected UNIQUE uq_code_slug to reject duplicate ('C1', 's1')")
	}
	if _, err := testDB.ExecContext(ctx, "INSERT INTO ddl_c_child (id, age, code, slug, note, parent_id) VALUES (2, 22, 'C1', 's2', 'ok', 2)"); err != nil {
		t.Fatalf("valid non-dup composite insert failed: %v", err)
	}

	// Verify FK RESTRICT and SET NULL and CASCADE behavior
	if _, err := testDB.ExecContext(ctx, "INSERT INTO ddl_c_restrict (id, parent_id) VALUES (1, 2)"); err != nil {
		t.Fatalf("insert restrict row: %v", err)
	}
	if _, err := testDB.ExecContext(ctx, "DELETE FROM ddl_c_parent WHERE id = 2"); err == nil {
		t.Errorf("expected RESTRICT FK to prevent deleting parent 2")
	}
	if _, err := testDB.ExecContext(ctx, "DELETE FROM ddl_c_restrict WHERE id = 1"); err != nil {
		t.Fatalf("delete restrict row: %v", err)
	}

	if _, err := testDB.ExecContext(ctx, "INSERT INTO ddl_c_setnull (id, parent_id) VALUES (1, 3)"); err != nil {
		t.Fatalf("insert setnull row: %v", err)
	}
	if _, err := testDB.ExecContext(ctx, "DELETE FROM ddl_c_parent WHERE id = 3"); err != nil {
		t.Fatalf("delete parent 3 with SET NULL: %v", err)
	}
	var nullParent sql.NullInt64
	if err := testDB.QueryRowContext(ctx, "SELECT parent_id FROM ddl_c_setnull WHERE id = 1").Scan(&nullParent); err != nil || nullParent.Valid {
		t.Errorf("expected parent_id to be NULL after SET NULL cascade, got %+v err=%v", nullParent, err)
	}

	// Delete parent 2 now -> CASCADE removes ddl_c_child id=2
	if _, err := testDB.ExecContext(ctx, "DELETE FROM ddl_c_parent WHERE id = 2"); err != nil {
		t.Fatalf("delete parent 2 with CASCADE: %v", err)
	}
	var remaining int
	if err := testDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM ddl_c_child").Scan(&remaining); err != nil || remaining != 1 {
		t.Errorf("expected 1 child row after CASCADE delete, got %d err=%v", remaining, err)
	}

	// Create an index and trigger on ddl_c_child, then AlterField renaming `age` -> `years`
	if _, err := testDB.ExecContext(ctx, "CREATE INDEX `idx_child_code` ON `ddl_c_child` (`code`, `age`);"); err != nil {
		t.Fatalf("create index: %v", err)
	}
	if _, err := testDB.ExecContext(ctx, "CREATE TABLE `ddl_c_audit` (`msg` TEXT);"); err != nil {
		t.Fatalf("create audit table: %v", err)
	}
	if _, err := testDB.ExecContext(ctx, "CREATE TRIGGER `trg_child_audit` AFTER INSERT ON `ddl_c_child` BEGIN INSERT INTO `ddl_c_audit` (`msg`) VALUES ('inserted'); END;"); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	var ageCol migrator.Column
	for _, c := range childTbl.Columns() {
		if c.Name == "Age" {
			ageCol = *c
		}
	}
	yearsCol := ageCol
	yearsCol.Column = "years"

	tx, err := ed.StartTransaction(ctx)
	if err != nil {
		t.Fatalf("StartTransaction: %v", err)
	}
	txCtx := migrator.ContextWithDb(ctx, tx)
	if err := ed.AlterField(txCtx, childTbl, ageCol, yearsCol); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("AlterField rename with constraints: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Verify FK pragmas were restored ON after migration transaction
	var fkState int
	if err := testDB.QueryRowContext(ctx, "PRAGMA foreign_keys;").Scan(&fkState); err != nil || fkState != 1 {
		t.Errorf("expected PRAGMA foreign_keys = 1 restored, got %d err=%v", fkState, err)
	}

	// Verify index, trigger, and all 4 constraints survived AlterField and age -> years was rewritten
	var schemaAfterAlter string
	if err := testDB.QueryRowContext(ctx, "SELECT sql FROM sqlite_schema WHERE type='table' AND name='ddl_c_child'").Scan(&schemaAfterAlter); err != nil {
		t.Fatalf("fetch alter sql: %v", err)
	}
	for _, want := range []string{"chk_age_min", "chk_note_comma", "uq_code_slug", "fk_parent_id", "`years` >="} {
		if !strings.Contains(schemaAfterAlter, want) {
			t.Errorf("expected %q preserved after AlterField, got:\n%s", want, schemaAfterAlter)
		}
	}

	var trigCount, idxCount int
	_ = testDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_schema WHERE tbl_name='ddl_c_child' AND type='trigger' AND name='trg_child_audit'").Scan(&trigCount)
	_ = testDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_schema WHERE tbl_name='ddl_c_child' AND type='index' AND name='idx_child_code'").Scan(&idxCount)
	if trigCount != 1 || idxCount != 1 {
		t.Errorf("expected trigger=1 & index=1 preserved across AlterField, got trigger=%d index=%d", trigCount, idxCount)
	}

	// Rewritten CHECK still rejects years=15; valid insert fires trigger
	if _, err := testDB.ExecContext(ctx, "INSERT INTO ddl_c_child (id, years, code, slug, note, parent_id) VALUES (3, 15, 'C3', 's3', 'ok', 1)"); err == nil {
		t.Errorf("expected rewritten CHECK on years to reject 15")
	}
	if _, err := testDB.ExecContext(ctx, "INSERT INTO ddl_c_child (id, years, code, slug, note, parent_id) VALUES (3, 25, 'C3', 's3', 'ok', 1)"); err != nil {
		t.Fatalf("insert valid row after rename failed: %v", err)
	}
	var auditRows int
	if err := testDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM ddl_c_audit").Scan(&auditRows); err != nil || auditRows != 1 {
		t.Errorf("expected preserved trigger to insert 1 audit row, got %d err=%v", auditRows, err)
	}

	// AddConstraint must fail when existing rows violate the new CHECK
	violatingChk := constraints.CheckConstraint("chk_years_gt_50", "", expr.Raw("![Age] >= ?", expr.V(50, true)))
	// Resolve via column years since field Age now maps to age; use raw SQL matching live column `years`
	_ = violatingChk
	liveCheck := constraints.CheckConstraint("chk_years_gt_50", "", expr.String("`years` >= 50")).Resolve(childRes.ExpressionInfo()).(constraints.Constraint)
	if err := ed.AddConstraint(ctx, childTbl, liveCheck); err == nil {
		t.Errorf("expected AddConstraint to fail when existing rows violate CHECK")
	}

	// DropConstraint error cases & valid drop
	if err := ed.DropConstraint(ctx, childTbl, constraints.UniqueConstraint("", "", "Code")); !errors.Is(err, derrors.ValueError) {
		t.Errorf("expected ValueError for empty constraint ID, got %v", err)
	}
	if err := ed.DropConstraint(ctx, childTbl, constraints.CheckConstraint("does_not_exist", "", nil)); err == nil {
		t.Errorf("expected error dropping missing constraint")
	}
	if err := ed.DropConstraint(ctx, childTbl, constraints.CheckConstraint("chk_age_min", "", nil)); err != nil {
		t.Fatalf("DropConstraint(chk_age_min): %v", err)
	}
	// Dropping chk_age_min allows years=10, while chk_note_comma and uq_code_slug remain enforced
	if _, err := testDB.ExecContext(ctx, "INSERT INTO ddl_c_child (id, years, code, slug, note, parent_id) VALUES (4, 10, 'C4', 's4', 'ok', 1)"); err != nil {
		t.Fatalf("insert years=10 after dropping chk_age_min failed: %v", err)
	}
	if _, err := testDB.ExecContext(ctx, "INSERT INTO ddl_c_child (id, years, code, slug, note, parent_id) VALUES (5, 10, 'C5', 's5', 'bad, (val)', 1)"); err == nil {
		t.Errorf("expected remaining chk_note_comma to still enforce after dropping chk_age_min")
	}
	if _, err := testDB.ExecContext(ctx, "INSERT INTO ddl_c_child (id, years, code, slug, note, parent_id) VALUES (5, 10, 'C4', 's4', 'ok', 1)"); err == nil {
		t.Errorf("expected remaining uq_code_slug to still enforce after dropping chk_age_min")
	}

	// WITHOUT ROWID parser edge case
	if _, err := testDB.ExecContext(ctx, "CREATE TABLE `ddl_c_wor` (`k` TEXT PRIMARY KEY, `v` INTEGER) WITHOUT ROWID;"); err != nil {
		t.Fatalf("create WITHOUT ROWID table: %v", err)
	}
	worTbl := &migrator.ModelTable{Table: "ddl_c_wor"}
	worCheck := constraints.CheckConstraint("chk_wor_v", "", expr.String("`v` > 0")).Resolve(childRes.ExpressionInfo()).(constraints.Constraint)
	if err := ed.AddConstraint(ctx, worTbl, worCheck); err != nil {
		t.Fatalf("AddConstraint on WITHOUT ROWID failed: %v", err)
	}
	if _, err := testDB.ExecContext(ctx, "INSERT INTO `ddl_c_wor` (`k`, `v`) VALUES ('a', -1)"); err == nil {
		t.Errorf("expected CHECK on WITHOUT ROWID table to fail for v=-1")
	}
	if _, err := testDB.ExecContext(ctx, "INSERT INTO `ddl_c_wor` (`k`, `v`) VALUES ('a', 5)"); err != nil {
		t.Fatalf("valid insert into WITHOUT ROWID failed: %v", err)
	}
	if err := ed.DropConstraint(ctx, worTbl, worCheck); err != nil {
		t.Fatalf("DropConstraint on WITHOUT ROWID failed: %v", err)
	}
	if _, err := testDB.ExecContext(ctx, "INSERT INTO `ddl_c_wor` (`k`, `v`) VALUES ('b', -1)"); err != nil {
		t.Fatalf("expected insert v=-1 after DropConstraint on WITHOUT ROWID to succeed, got %v", err)
	}
}
