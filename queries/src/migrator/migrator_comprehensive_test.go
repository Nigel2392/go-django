package migrator_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	_ "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	derrors "github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/drivers/securedb"
	"github.com/Nigel2392/go-django/queries/src/expr/constraints"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/queries/src/migrator/sql/sqlite"
	testsql "github.com/Nigel2392/go-django/queries/src/migrator/sql/test_sql"
	"github.com/Nigel2392/go-django/queries/src/models"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var migCompDBSeq atomic.Int64

func openTestSQLite(t *testing.T) (drivers.Database, *sqlite.SQLiteSchemaEditor) {
	t.Helper()
	dsn := fmt.Sprintf("file:mig_comp_%d?mode=memory&cache=shared", migCompDBSeq.Add(1))
	db, err := drivers.Open(context.Background(), "sqlite3", dsn)
	if err != nil {
		t.Fatalf("failed to open sqlite db %q: %v", dsn, err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db, sqlite.NewSQLiteSchemaEditor(db)
}

func registerModelsContentTypes(models ...attrs.Definer) {
	for _, mdl := range models {
		cType := contenttypes.NewContentType(mdl)
		if contenttypes.DefinitionForObject(mdl) != nil {
			continue
		}
		typName := cType.ShortTypeName()
		_ = typName
		target := mdl
		contenttypes.Register(&contenttypes.ContentTypeDefinition{
			ContentObject: target,
			GetObject: func() any {
				return attrs.NewObject[attrs.Definer](context.Background(), target)
			},
			GetLabel: func(context.Context) string {
				return contenttypes.NewContentType(target).Model()
			},
		})
	}
}

func initMigTestApp(t *testing.T, db drivers.Database, appConfigs ...func() django.AppConfig) {
	t.Helper()
	django.Global = nil
	dbtype.TYPES.Unlock()
	_ = attrs.ResetDefinitions.Send(context.Background(), nil)

	settings := map[string]any{}
	if db != nil {
		settings[django.APPVAR_DATABASE] = db
	}

	appArgs := make([]any, 0, len(appConfigs))
	for _, fn := range appConfigs {
		appArgs = append(appArgs, fn)
	}

	app := django.App(
		django.Configure(settings),
		django.Apps(appArgs...),
		django.Flag(
			django.FlagSkipCmds,
			django.FlagSkipDepsCheck,
			django.FlagSkipChecks,
		),
	)
	if err := app.Initialize(); err != nil {
		t.Fatalf("failed to initialize django app: %v", err)
	}
	for head := app.Apps.Front(); head != nil; head = head.Next() {
		registerModelsContentTypes(head.Value.Models()...)
	}
}

type sqliteColInfo struct {
	CID       int
	Name      string
	ColType   string
	NotNull   bool
	DfltValue sql.NullString
	PK        int
}

func sqliteTables(t *testing.T, db drivers.DB) map[string]bool {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), "SELECT name FROM sqlite_master WHERE type = 'table'")
	if err != nil {
		t.Fatalf("query sqlite_master tables: %v", err)
	}
	defer rows.Close()

	res := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table name: %v", err)
		}
		res[name] = true
	}
	return res
}

func sqliteColumns(t *testing.T, db drivers.DB, table string) map[string]sqliteColInfo {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), fmt.Sprintf("PRAGMA table_info(`%s`)", table))
	if err != nil {
		t.Fatalf("PRAGMA table_info(%s): %v", table, err)
	}
	defer rows.Close()

	cols := make(map[string]sqliteColInfo)
	for rows.Next() {
		var c sqliteColInfo
		if err := rows.Scan(&c.CID, &c.Name, &c.ColType, &c.NotNull, &c.DfltValue, &c.PK); err != nil {
			t.Fatalf("scan pragma table_info(%s): %v", table, err)
		}
		cols[c.Name] = c
	}
	return cols
}

func sqliteIndexes(t *testing.T, db drivers.DB, table string) map[string]bool {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), fmt.Sprintf("PRAGMA index_list(`%s`)", table))
	if err != nil {
		t.Fatalf("PRAGMA index_list(%s): %v", table, err)
	}
	defer rows.Close()

	idxs := make(map[string]bool)
	for rows.Next() {
		var seq int
		var name, origin string
		var unique, partial int
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			t.Fatalf("scan pragma index_list(%s): %v", table, err)
		}
		idxs[name] = unique != 0
	}
	return idxs
}

type sqliteFKInfo struct {
	ID       int
	Seq      int
	Table    string
	From     string
	To       string
	OnUpdate string
	OnDelete string
	Match    string
}

func sqliteForeignKeys(t *testing.T, db drivers.DB, table string) []sqliteFKInfo {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), fmt.Sprintf("PRAGMA foreign_key_list(`%s`)", table))
	if err != nil {
		t.Fatalf("PRAGMA foreign_key_list(%s): %v", table, err)
	}
	defer rows.Close()

	var fks []sqliteFKInfo
	for rows.Next() {
		var fk sqliteFKInfo
		if err := rows.Scan(&fk.ID, &fk.Seq, &fk.Table, &fk.From, &fk.To, &fk.OnUpdate, &fk.OnDelete, &fk.Match); err != nil {
			t.Fatalf("scan pragma foreign_key_list(%s): %v", table, err)
		}
		fks = append(fks, fk)
	}
	return fks
}

func readMigrationFileJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration file %q: %v", path, err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal migration file %q: %v", path, err)
	}
	return m
}

func countMigFiles(t *testing.T, root string) int {
	t.Helper()
	count := 0
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == migrator.MIGRATION_FILE_SUFFIX {
			count++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %q: %v", root, err)
	}
	return count
}

type noopMigLog struct{}

func (noopMigLog) Log(migrator.ActionType, *migrator.MigrationFile, *migrator.Changed[*migrator.ModelTable], *migrator.Changed[*migrator.Column], *migrator.Changed[*migrator.Index], *migrator.Changed[constraints.Constraint]) {
}

func assertNoPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()
	fn()
}

// --- Models for TestCheckCanMigrate ---

type cantMigrateEmbeddedModel struct {
	migrator.CantMigrate
	ID int64 `attrs:"primary"`
}

func (m *cantMigrateEmbeddedModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions(ctx, m)
}

type customCanMigrateModel struct {
	allow bool
	ID    int64 `attrs:"primary"`
}

func (m *customCanMigrateModel) CanMigrate() bool { return m.allow }
func (m *customCanMigrateModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions(ctx, m)
}

type metaBoolFalseModel struct{ ID int64 `attrs:"primary"` }

func (m *metaBoolFalseModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions(ctx, m)
}

type metaBoolTrueModel struct{ ID int64 `attrs:"primary"` }

func (m *metaBoolTrueModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions(ctx, m)
}

type metaIntZeroModel struct{ ID int64 `attrs:"primary"` }

func (m *metaIntZeroModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions(ctx, m)
}

type metaFloatNonZeroModel struct{ ID int64 `attrs:"primary"` }

func (m *metaFloatNonZeroModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions(ctx, m)
}

type metaInvalidTypeModel struct{ ID int64 `attrs:"primary"` }

func (m *metaInvalidTypeModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions(ctx, m)
}

type plainMigModel struct {
	ID   int64 `attrs:"primary"`
	Name string
}

func (m *plainMigModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions(ctx, m)
}

func TestCheckCanMigrate(t *testing.T) {
	if migrator.CheckCanMigrate(&cantMigrateEmbeddedModel{}) {
		t.Errorf("expected CantMigrate model to return false")
	}
	if !migrator.CheckCanMigrate(&customCanMigrateModel{allow: true}) {
		t.Errorf("expected CanMigrate()=true to return true")
	}
	if migrator.CheckCanMigrate(&customCanMigrateModel{allow: false}) {
		t.Errorf("expected CanMigrate()=false to return false")
	}
	if !migrator.CheckCanMigrate("not a definer") {
		t.Errorf("expected non-definer value to return true")
	}
	attrs.RegisterModel(&plainMigModel{})
	if !migrator.CheckCanMigrate(&plainMigModel{}) {
		t.Errorf("expected plain model without meta override to return true")
	}

	attrs.RegisterModel(&metaBoolFalseModel{})
	attrs.StoreOnMeta(&metaBoolFalseModel{}, migrator.MetaAllowMigrateKey, false)
	if migrator.CheckCanMigrate(&metaBoolFalseModel{}) {
		t.Errorf("expected MetaAllowMigrateKey=false to return false")
	}

	attrs.RegisterModel(&metaBoolTrueModel{})
	attrs.StoreOnMeta(&metaBoolTrueModel{}, migrator.MetaAllowMigrateKey, true)
	if !migrator.CheckCanMigrate(&metaBoolTrueModel{}) {
		t.Errorf("expected MetaAllowMigrateKey=true to return true")
	}

	attrs.RegisterModel(&metaIntZeroModel{})
	attrs.StoreOnMeta(&metaIntZeroModel{}, migrator.MetaAllowMigrateKey, 0)
	if migrator.CheckCanMigrate(&metaIntZeroModel{}) {
		t.Errorf("expected MetaAllowMigrateKey=0 to return false")
	}

	attrs.RegisterModel(&metaFloatNonZeroModel{})
	attrs.StoreOnMeta(&metaFloatNonZeroModel{}, migrator.MetaAllowMigrateKey, float64(1.5))
	if !migrator.CheckCanMigrate(&metaFloatNonZeroModel{}) {
		t.Errorf("expected MetaAllowMigrateKey=1.5 to return true")
	}

	attrs.RegisterModel(&metaInvalidTypeModel{})
	attrs.StoreOnMeta(&metaInvalidTypeModel{}, migrator.MetaAllowMigrateKey, "invalid")
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic for invalid MetaAllowMigrateKey type")
			}
		}()
		migrator.CheckCanMigrate(&metaInvalidTypeModel{})
	}()

	// Verify MakeMigrations skips CantMigrate model while creating migration for sibling model.
	db, editor := openTestSQLite(t)
	initMigTestApp(t, db, func() django.AppConfig {
		cfg := apps.NewAppConfig("migcan")
		cfg.ModelObjects = []attrs.Definer{
			&cantMigrateEmbeddedModel{},
			&plainMigModel{},
		}
		return cfg
	})
	tmpDir := t.TempDir()
	engine := migrator.NewMigrationEngine(tmpDir, editor)
	engine.MigrationLog = noopMigLog{}

	if err := engine.MakeMigrations(context.Background()); err != nil {
		t.Fatalf("MakeMigrations failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "migcan", "cantMigrateEmbeddedModel")); !os.IsNotExist(err) {
		t.Errorf("expected no migration directory for CantMigrate model, stat err=%v", err)
	}
	if countMigFiles(t, filepath.Join(tmpDir, "migcan", "plainMigModel")) != 1 {
		t.Errorf("expected 1 migration file for sibling plainMigModel")
	}
}

// --- Models & helpers for TestNewModelTable_ColumnsIndexesAndDiff ---

type customMigField struct {
	attrs.Field
	canMigrate bool
	dbType     dbtype.Type
}

func (c *customMigField) CanMigrate() bool               { return c.canMigrate }
func (c *customMigField) DBType(*migrator.Column) dbtype.Type { return c.dbType }

type customDBTypeField struct {
	attrs.Field
	dbType dbtype.Type
}

func (c *customDBTypeField) DBType(*migrator.Column) dbtype.Type { return c.dbType }

type unitTarget struct {
	ID   int64 `attrs:"primary"`
	Code string
}

func (m *unitTarget) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions(ctx, m)
}

type unitThrough struct {
	Source *unitSource
	Target *unitTarget
}

func (m *unitThrough) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make[*unitThrough, attrs.Field](ctx, m,
		attrs.NewField(m, "Source", &attrs.Config{Column: "source_id", ReverseAlias: "ThroughSources", RelForeignKey: attrs.Relate(&unitSource{}, "", nil)}),
		attrs.NewField(m, "Target", &attrs.Config{Column: "target_id", ReverseAlias: "ThroughTargets", RelForeignKey: attrs.Relate(&unitTarget{}, "", nil)}),
	)
}

type unitSource struct {
	Code       string
	ID         int64
	Email      string
	Tag        string
	Skipped    string
	NonMig     string
	Embedded   string
	CustomAttr string
	CustomType string
	TargetFK   *unitTarget
	TargetO2O  *unitTarget
	TargetM2M  any
	TargetO2OT any
	RevItems   any
}

func (m *unitSource) DatabaseIndexes(obj attrs.Definer) []migrator.Index {
	return []migrator.Index{
		{Identifier: "explicit_code_idx", Fields: []string{"Code"}, Unique: true},
		{Fields: []string{"Email", "Tag"}, Unique: true},
	}
}

func (m *unitSource) UniqueTogether() [][]string {
	return [][]string{
		{"Tag", "Email"}, // duplicate of Email+Tag DatabaseIndex -> deduplicated
		{"Code", "Tag"},
	}
}

func (m *unitSource) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make[*unitSource, attrs.Field](ctx, m,
		attrs.NewField(m, "Code", &attrs.Config{
			Column:     "code",
			Attributes: map[string]any{attrs.AttrUniqueKey: true},
		}),
		attrs.NewField(m, "Email", &attrs.Config{Column: "email"}),
		attrs.NewField(m, "Tag", &attrs.Config{Column: "tag"}),
		attrs.NewField(m, "ID", &attrs.Config{Primary: true, Column: "id"}), // declared after non-PK fields
		attrs.NewField(m, "Skipped", &attrs.Config{
			Column: "skipped",
			Attributes: map[string]any{
				migrator.AttrUseInDBKey: false,
			},
		}),
		&customMigField{
			Field:      attrs.NewField(m, "NonMig", &attrs.Config{Column: "non_mig"}),
			canMigrate: false,
			dbType:     dbtype.String,
		},
		attrs.NewField(m, "Embedded", &attrs.Config{Column: "embedded", IsEmbedded: true}),
		attrs.NewField(m, "CustomAttr", &attrs.Config{
			Column: "custom_attr",
			Attributes: map[string]any{
				migrator.AttrDBTypeKey: "MY_CUSTOM_SQL_TYPE",
			},
		}),
		&customDBTypeField{
			Field:  attrs.NewField(m, "CustomType", &attrs.Config{Column: "custom_type"}),
			dbType: dbtype.UUID,
		},
		attrs.NewField(m, "TargetFK", &attrs.Config{
			Column:        "target_fk_id",
			ReverseAlias:  "FKSourceSet",
			RelForeignKey: attrs.Relate(&unitTarget{}, "", nil),
			Null:          false,
			Attributes: map[string]any{
				migrator.AttrOnDeleteKey: constraints.A_CASCADE,
				migrator.AttrOnUpdateKey: constraints.A_SET_NULL,
			},
		}),
		attrs.NewField(m, "TargetO2O", &attrs.Config{
			Column:       "target_o2o_id",
			ReverseAlias: "O2OSource",
			RelOneToOne:  attrs.Relate(&unitTarget{}, "", nil),
		}),
		attrs.NewField(m, "TargetM2M", &attrs.Config{
			Column:       "id",
			ReverseAlias: "M2MSourceSet",
			RelManyToMany: attrs.Relate(&unitTarget{}, "", &attrs.ThroughModel{
				This: &unitThrough{}, Source: "Source", Target: "Target",
			}),
		}),
		attrs.NewField(m, "TargetO2OT", &attrs.Config{
			Column:       "id",
			ReverseAlias: "O2OTSource",
			RelOneToOne: attrs.Relate(&unitTarget{}, "", &attrs.ThroughModel{
				This: &unitThrough{}, Source: "Source", Target: "Target",
			}),
		}),
		attrs.NewField(m, "RevItems", &attrs.Config{
			Column:       "id",
			ReverseAlias: "Source",
		}),
	).WithTableName("unit_source")
}

type stringPKModel struct {
	ID drivers.ULID `attrs:"primary"`
}

func (m *stringPKModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions(ctx, m)
}

type dupIndexModel struct {
	ID int64 `attrs:"primary"`
	A  string
	B  string
}

func (m *dupIndexModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions(ctx, m)
}

func (m *dupIndexModel) DatabaseIndexes(attrs.Definer) []migrator.Index {
	return []migrator.Index{
		{Fields: []string{"A", "B"}},
		{Fields: []string{"B", "A"}},
	}
}

func TestNewModelTable_ColumnsIndexesAndDiff(t *testing.T) {
	attrs.RegisterModel(&unitTarget{})
	attrs.RegisterModel(&unitThrough{})
	attrs.RegisterModel(&unitSource{})
	registerModelsContentTypes(&unitTarget{}, &unitThrough{}, &unitSource{})

	tbl := migrator.NewModelTable(&unitSource{})
	cols := tbl.Columns()
	if len(cols) == 0 || cols[0].Name != "ID" || !cols[0].Primary {
		t.Fatalf("expected primary key ID sorted to front, got cols[0]=%+v", cols)
	}
	if !cols[0].Auto {
		t.Errorf("expected int64 non-null PK Auto=true")
	}

	colByName := make(map[string]*migrator.Column)
	for _, c := range cols {
		colByName[c.Name] = c
	}

	if colByName["Skipped"].UseInDB {
		t.Errorf("AttrUseInDBKey=false should set UseInDB=false")
	}
	if colByName["NonMig"].UseInDB {
		t.Errorf("CanMigrate()=false field should set UseInDB=false")
	}
	if colByName["Embedded"].UseInDB {
		t.Errorf("IsEmbedded=true should set UseInDB=false")
	}
	if colByName["TargetM2M"].UseInDB || colByName["TargetM2M"].Rel == nil || colByName["TargetM2M"].Rel.Through == nil {
		t.Errorf("M2M column should have UseInDB=false and Through populated: %+v", colByName["TargetM2M"])
	}
	if colByName["TargetO2OT"].UseInDB || colByName["TargetO2OT"].Rel == nil || colByName["TargetO2OT"].Rel.Through == nil {
		t.Errorf("O2O through column should have UseInDB=false and Through populated: %+v", colByName["TargetO2OT"])
	}
	if !colByName["TargetFK"].UseInDB || colByName["TargetFK"].Rel.OnDelete != constraints.A_CASCADE || colByName["TargetFK"].Rel.OnUpdate != constraints.A_SET_NULL {
		t.Errorf("FK column should keep UseInDB=true and OnDelete/OnUpdate: %+v", colByName["TargetFK"])
	}
	if !colByName["TargetO2O"].UseInDB {
		t.Errorf("direct O2O without through should keep UseInDB=true")
	}
	if colByName["Code"].Nullable {
		t.Errorf("unique string field Code should have Nullable=false")
	}
	if !colByName["Email"].Nullable || colByName["Email"].Default.V != "" {
		t.Errorf("non-unique string Email expected Nullable=true Default=\"\", got Nullable=%v Default=%#v", colByName["Email"].Nullable, colByName["Email"].Default.V)
	}
	if got := colByName["CustomType"].DBType(); got != dbtype.UUID {
		t.Errorf("CanColumnDBType override got %v, want UUID", got)
	}
	if got := migrator.GetFieldType(&drivers.DriverSQLite{}, colByName["CustomAttr"]); got != "MY_CUSTOM_SQL_TYPE" {
		t.Errorf("AttrDBTypeKey override got %q, want MY_CUSTOM_SQL_TYPE", got)
	}

	// String/ULID primary key should not auto-increment.
	attrs.RegisterModel(&stringPKModel{})
	strPKTbl := migrator.NewModelTable(&stringPKModel{})
	strPKCols := strPKTbl.Columns()
	if strPKCols[0].Auto {
		t.Errorf("expected string/ULID primary key Auto=false")
	}

	// Indexes + UniqueTogether deduplication.
	idxs := tbl.Indexes()
	if len(idxs) != 3 {
		t.Fatalf("expected 3 deduplicated indexes, got %d: %+v", len(idxs), idxs)
	}
	if idxs[0].Name() != "explicit_code_idx" {
		t.Errorf("expected explicit index name, got %q", idxs[0].Name())
	}
	if idxs[1].Name() != "unit_source_idx_Email_Tag" {
		t.Errorf("expected generated index name unit_source_idx_Email_Tag, got %q", idxs[1].Name())
	}
	if idxs[2].Name() != "Code_Tag_unique_together_idx" || !idxs[2].Unique {
		t.Errorf("expected unique_together index, got %+v", idxs[2])
	}
	if len(idxs[1].Columns()) != 2 {
		t.Errorf("expected 2 columns resolved for composite index")
	}

	// Unknown index field panics.
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic for missing column in Index.Columns()")
			}
		}()
		bad := migrator.Index{Fields: []string{"DoesNotExist"}}
		_ = bad
		// attach via table by copying slice item behavior:
		_ = (&migrator.Index{Identifier: "bad", Fields: []string{"DoesNotExist"}}).Columns()
	}()

	// Duplicate DatabaseIndexes panics.
	attrs.RegisterModel(&dupIndexModel{})
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic on duplicate DatabaseIndexes")
			}
		}()
		migrator.NewModelTable(&dupIndexModel{})
	}()

	// HasDefault edge cases.
	var nilPtr *int
	var nonNilVal = 5
	var emptySlice []byte
	nonEmptySlice := []byte{1}
	if (&migrator.Column{Default: drivers.Value[any]{V: nil}}).HasDefault() {
		t.Errorf("nil default should return false")
	}
	if (&migrator.Column{Default: drivers.Value[any]{V: nilPtr}}).HasDefault() {
		t.Errorf("nil ptr default should return false")
	}
	if !(&migrator.Column{Default: drivers.Value[any]{V: &nonNilVal}}).HasDefault() {
		t.Errorf("non-nil ptr default should return true")
	}
	if (&migrator.Column{Default: drivers.Value[any]{V: ""}}).HasDefault() {
		t.Errorf("empty string default should return false")
	}
	if !(&migrator.Column{Default: drivers.Value[any]{V: "abc"}}).HasDefault() {
		t.Errorf("non-empty string default should return true")
	}
	if !(&migrator.Column{Default: drivers.Value[any]{V: int64(0)}}).HasDefault() {
		t.Errorf("zero int on non-PK column should return HasDefault=true")
	}
	if (&migrator.Column{Primary: true, Default: drivers.Value[any]{V: int64(0)}}).HasDefault() {
		t.Errorf("zero int on PK column should return HasDefault=false")
	}
	if (&migrator.Column{Default: drivers.Value[any]{V: time.Time{}}}).HasDefault() {
		t.Errorf("zero time.Time should return HasDefault=false")
	}
	if !(&migrator.Column{Default: drivers.Value[any]{V: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)}}).HasDefault() {
		t.Errorf("non-zero time.Time should return HasDefault=true")
	}
	if (&migrator.Column{Default: drivers.Value[any]{V: emptySlice}}).HasDefault() {
		t.Errorf("nil slice should return HasDefault=false")
	}
	if !(&migrator.Column{Default: drivers.Value[any]{V: nonEmptySlice}}).HasDefault() {
		t.Errorf("non-nil slice should return HasDefault=true")
	}

	// EqualDefaultValue edge cases.
	if !migrator.EqualDefaultValue(nil, "") {
		t.Errorf("expected nil == zero string")
	}
	if migrator.EqualDefaultValue(nil, "x") {
		t.Errorf("expected nil != \"x\"")
	}
	if !migrator.EqualDefaultValue(int64(42), float64(42)) {
		t.Errorf("expected convertible numeric 42 == 42.0")
	}
	tUTC := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	tPlus2 := tUTC.In(time.FixedZone("UTC+2", 2*3600))
	if !migrator.EqualDefaultValue(tUTC, tPlus2) {
		t.Errorf("expected identical instants in different zones to be equal")
	}

	// Column.ChangeList & Equals.
	if cl := (*migrator.Column)(nil).ChangeList(colByName["Email"]); len(cl) != 1 || cl[0] != "*" {
		t.Errorf("nil vs non-nil ChangeList = %v, want [*]", cl)
	}
	c1 := *colByName["Email"]
	c2 := *colByName["Email"]
	if !c1.Equals(&c2) {
		t.Errorf("expected identical column copies to be equal")
	}
	c2.Column = "email_v2"
	c2.MinLength = 3
	c2.MaxLength = 120
	c2.Unique = true
	c2.Nullable = false
	c2.Default = drivers.Value[any]{V: "a@b.c"}
	changes := c2.ChangeList(&c1)
	for _, want := range []string{"Column", "MinLength", "MaxLength", "Unique", "Nullable", "Default"} {
		found := false
		for _, ch := range changes {
			if ch == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %q in ChangeList, got %v", want, changes)
		}
	}

	fkCopy1 := *colByName["TargetFK"]
	fkCopy2 := *colByName["TargetFK"]
	fkRelCopy := *fkCopy2.Rel
	fkRelCopy.Type = attrs.RelOneToOne
	fkCopy2.Rel = &fkRelCopy
	if cl := fkCopy1.ChangeList(&fkCopy2); len(cl) == 0 || cl[0] != "Rel:Type" {
		t.Errorf("expected Rel:Type change, got %v", cl)
	}
}

// --- Model for TestModelTable_JSONRoundTrip_AllFieldTypes ---

type migAllTypesTarget struct {
	ID   int64 `attrs:"primary"`
	Name string
}

func (m *migAllTypesTarget) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions(ctx, m)
}

type migAllTypesThrough struct {
	Src *migAllTypesModel
	Tgt *migAllTypesTarget
}

func (m *migAllTypesThrough) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make[*migAllTypesThrough, attrs.Field](ctx, m,
		attrs.NewField(m, "Src", &attrs.Config{Column: "src_id", ReverseAlias: "AllTypesThroughSrc", RelForeignKey: attrs.Relate(&migAllTypesModel{}, "", nil)}),
		attrs.NewField(m, "Tgt", &attrs.Config{Column: "tgt_id", ReverseAlias: "AllTypesThroughTgt", RelForeignKey: attrs.Relate(&migAllTypesTarget{}, "", nil)}),
	)
}

type migAllTypesModel struct {
	ID            int64 `attrs:"primary"`
	FInt          int
	FInt8         int8
	FInt16        int16
	FInt32        int32
	FInt64        int64
	FUint         uint
	FUint64       uint64
	FFloat32      float32
	FFloat64      float64
	FBool         bool
	FString       string
	FBytes        []byte
	FTime         time.Time
	FDecimal      decimal.Decimal
	FULID         drivers.ULID
	FUUID         drivers.UUID
	FEmail        drivers.Email
	FChar         drivers.Char
	FText         drivers.Text
	FDrvString    drivers.String
	FDrvBytes     drivers.Bytes
	FDrvBLOB      drivers.BLOB
	FTimestamp    drivers.Timestamp
	FLocalTime    drivers.LocalTime
	FDateTime     drivers.DateTime
	FJSON         drivers.JSON[map[string]any]
	FNullString   sql.NullString
	FNullInt64    sql.NullInt64
	FNullBool     sql.NullBool
	FNullFloat64  sql.NullFloat64
	FNullTime     sql.NullTime
	FNullDecimal  decimal.NullDecimal
	FEncrypted    *securedb.EncryptedString
	FFK           *migAllTypesTarget
	FLazyFK       *migAllTypesTarget
	FM2M          any
}

func (m *migAllTypesModel) DatabaseIndexes(attrs.Definer) []migrator.Index {
	return []migrator.Index{
		{Identifier: "mig_all_types_str_idx", Fields: []string{"FString"}, Unique: false},
	}
}

func (m *migAllTypesModel) UniqueTogether() [][]string {
	return [][]string{{"FEmail", "FULID"}}
}

func (m *migAllTypesModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make[*migAllTypesModel, attrs.Field](ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*migAllTypesModel, int64] {
			return fattrs.PtrFieldConfig[*migAllTypesModel, int64]{Config: attrs.Config{Primary: true}}
		}),
		attrs.NewField(m, "FInt"),
		attrs.NewField(m, "FInt8"),
		attrs.NewField(m, "FInt16"),
		attrs.NewField(m, "FInt32"),
		fattrs.Field(m, "FInt64", &m.FInt64, func() fattrs.PtrFieldConfig[*migAllTypesModel, int64] {
			return fattrs.PtrFieldConfig[*migAllTypesModel, int64]{Config: attrs.Config{Default: int64(42)}}
		}),
		attrs.NewField(m, "FUint"),
		attrs.NewField(m, "FUint64"),
		attrs.NewField(m, "FFloat32"),
		fattrs.Field(m, "FFloat64", &m.FFloat64, func() fattrs.PtrFieldConfig[*migAllTypesModel, float64] {
			return fattrs.PtrFieldConfig[*migAllTypesModel, float64]{Config: attrs.Config{Default: 2.5}}
		}),
		fattrs.Field(m, "FBool", &m.FBool, func() fattrs.PtrFieldConfig[*migAllTypesModel, bool] {
			return fattrs.PtrFieldConfig[*migAllTypesModel, bool]{Config: attrs.Config{Default: true}}
		}),
		fattrs.Field(m, "FString", &m.FString, func() fattrs.PtrFieldConfig[*migAllTypesModel, string] {
			return fattrs.PtrFieldConfig[*migAllTypesModel, string]{Config: attrs.Config{MaxLength: 100, Default: "hello"}}
		}),
		attrs.NewField(m, "FBytes"),
		fattrs.Field(m, "FTime", &m.FTime, func() fattrs.PtrFieldConfig[*migAllTypesModel, time.Time] {
			return fattrs.PtrFieldConfig[*migAllTypesModel, time.Time]{Config: attrs.Config{Default: testsql.DEFAULT_TIME}}
		}),
		fattrs.Field(m, "FDecimal", &m.FDecimal, func() fattrs.PtrFieldConfig[*migAllTypesModel, decimal.Decimal] {
			return fattrs.PtrFieldConfig[*migAllTypesModel, decimal.Decimal]{Config: attrs.Config{Precision: 13, Scale: 4}}
		}),
		attrs.NewField(m, "FULID"),
		attrs.NewField(m, "FUUID"),
		attrs.NewField(m, "FEmail"),
		fattrs.Field(m, "FChar", &m.FChar, func() fattrs.PtrFieldConfig[*migAllTypesModel, drivers.Char] {
			return fattrs.PtrFieldConfig[*migAllTypesModel, drivers.Char]{Config: attrs.Config{MaxLength: 12}}
		}),
		attrs.NewField(m, "FText"),
		attrs.NewField(m, "FDrvString"),
		attrs.NewField(m, "FDrvBytes"),
		attrs.NewField(m, "FDrvBLOB"),
		attrs.NewField(m, "FTimestamp"),
		attrs.NewField(m, "FLocalTime"),
		attrs.NewField(m, "FDateTime"),
		attrs.NewField(m, "FJSON"),
		attrs.NewField(m, "FNullString"),
		attrs.NewField(m, "FNullInt64"),
		attrs.NewField(m, "FNullBool"),
		attrs.NewField(m, "FNullFloat64"),
		attrs.NewField(m, "FNullTime"),
		attrs.NewField(m, "FNullDecimal"),
		attrs.NewField(m, "FEncrypted"),
		fattrs.Field(m, "FFK", &m.FFK, func() fattrs.PtrFieldConfig[*migAllTypesModel, *migAllTypesTarget] {
			return fattrs.PtrFieldConfig[*migAllTypesModel, *migAllTypesTarget]{
				Config: attrs.Config{
					Column:        "f_fk_id",
					Null:          true,
					ReverseAlias:  "AllTypesFKSet",
					RelForeignKey: attrs.Relate(&migAllTypesTarget{}, "", nil),
					Attributes: map[string]any{
						migrator.AttrOnDeleteKey: constraints.A_CASCADE,
						migrator.AttrOnUpdateKey: constraints.A_RESTRICT,
					},
				},
			}
		}),
		attrs.NewField(m, "FLazyFK", &attrs.Config{
			Column:        "f_lazy_fk_id",
			Null:          true,
			ReverseAlias:  "AllTypesLazyFKSet",
			RelForeignKey: attrs.RelatedDeferred(attrs.RelManyToOne, "migtypes.Target", "", nil),
		}),
		attrs.NewField(m, "FM2M", &attrs.Config{
			Column:       "id",
			ReverseAlias: "AllTypesM2MSet",
			RelManyToMany: attrs.Relate(&migAllTypesTarget{}, "", &attrs.ThroughModel{
				This: &migAllTypesThrough{}, Source: "Src", Target: "Tgt",
			}),
		}),
	).WithTableName("mig_all_types_model")
}

func TestModelTable_JSONRoundTrip_AllFieldTypes(t *testing.T) {
	_ = contenttypes.NewLazyRegistry("migtypes.Target", func(def *contenttypes.ContentTypeDefinition) bool {
		_, ok := def.ContentObject.(*migAllTypesTarget)
		return ok
	})
	registerModelsContentTypes(&migAllTypesTarget{}, &migAllTypesThrough{}, &migAllTypesModel{})
	attrs.RegisterModel(&migAllTypesTarget{})
	attrs.RegisterModel(&migAllTypesThrough{})
	attrs.RegisterModel(&migAllTypesModel{})
	if !dbtype.IsLocked() {
		dbtype.Lock()
	}
	_ = attrs.ResetDefinitions.Send(context.Background(), nil)

	orig := migrator.NewModelTable(&migAllTypesModel{})
	raw, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("json.Marshal(ModelTable) failed: %v", err)
	}

	var decoded migrator.ModelTable
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(ModelTable) failed: %v", err)
	}

	added, removed, diffs := orig.Diff(&decoded)
	if len(added) != 0 || len(removed) != 0 || len(diffs) != 0 {
		for _, d := range diffs {
			t.Errorf("diff on %s: changes=%v", d.New.Name, d.New.ChangeList(&d.Old))
		}
		t.Fatalf("expected 0 round-trip diff, got added=%d removed=%d diffs=%d", len(added), len(removed), len(diffs))
	}

	if len(orig.Indexes()) != len(decoded.Indexes()) {
		t.Fatalf("index count mismatch: orig=%d decoded=%d", len(orig.Indexes()), len(decoded.Indexes()))
	}
	if _, ok := decoded.Indexes()[0].Columns()[0].Field.Attrs()["unused"]; ok {
		t.Fatalf("unexpected attr")
	}
	for i := range orig.Indexes() {
		a, b := orig.Indexes()[i], decoded.Indexes()[i]
		if a.Name() != b.Name() || a.Unique != b.Unique || !reflect.DeepEqual(a.Fields, b.Fields) {
			t.Errorf("index[%d] mismatch: orig=%+v decoded=%+v", i, a, b)
		}
	}
}

// --- Dynamic Schema Lifecycle Models for TestMigrator_RealSQLite_FullSchemaLifecycle ---

var migEvoState struct {
	addArticleFields     bool
	alterPriorityNotNull bool
	alterScoreAndSummary bool
	articleCompositeIdx  int // 0=none, 1=non-unique, 2=unique
	removeArticleFields  bool
	renameTagTable       bool
}

func resetMigEvoState() {
	migEvoState.addArticleFields = false
	migEvoState.alterPriorityNotNull = false
	migEvoState.alterScoreAndSummary = false
	migEvoState.articleCompositeIdx = 0
	migEvoState.removeArticleFields = false
	migEvoState.renameTagTable = false
}

type MigEvoAuthor struct {
	ID   int64 `attrs:"primary"`
	Name string
}

func (m *MigEvoAuthor) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*MigEvoAuthor, int64] {
			return fattrs.PtrFieldConfig[*MigEvoAuthor, int64]{Config: attrs.Config{Primary: true}}
		}),
		fattrs.Field(m, "Name", &m.Name, func() fattrs.PtrFieldConfig[*MigEvoAuthor, string] {
			return fattrs.PtrFieldConfig[*MigEvoAuthor, string]{Config: attrs.Config{MaxLength: 100}}
		}),
	).WithTableName("mig_evo_author")
}

type MigEvoTag struct {
	ID    int64 `attrs:"primary"`
	Label string
}

func (m *MigEvoTag) FieldDefs(ctx context.Context) attrs.Definitions {
	tbl := "mig_evo_tag"
	if migEvoState.renameTagTable {
		tbl = "mig_evo_tag_v2"
	}
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*MigEvoTag, int64] {
			return fattrs.PtrFieldConfig[*MigEvoTag, int64]{Config: attrs.Config{Primary: true}}
		}),
		fattrs.Field(m, "Label", &m.Label, func() fattrs.PtrFieldConfig[*MigEvoTag, string] {
			return fattrs.PtrFieldConfig[*MigEvoTag, string]{Config: attrs.Config{MaxLength: 50}}
		}),
	).WithTableName(tbl)
}

type MigEvoArticleTag struct {
	Article *MigEvoArticle
	Tag     *MigEvoTag
}

func (m *MigEvoArticleTag) UniqueTogether() [][]string {
	return [][]string{{"Article", "Tag"}}
}

func (m *MigEvoArticleTag) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "Article", &m.Article, func() fattrs.PtrFieldConfig[*MigEvoArticleTag, *MigEvoArticle] {
			return fattrs.PtrFieldConfig[*MigEvoArticleTag, *MigEvoArticle]{
				Config: attrs.Config{Column: "article_id", RelForeignKey: attrs.Relate(&MigEvoArticle{}, "", nil)},
			}
		}),
		fattrs.Field(m, "Tag", &m.Tag, func() fattrs.PtrFieldConfig[*MigEvoArticleTag, *MigEvoTag] {
			return fattrs.PtrFieldConfig[*MigEvoArticleTag, *MigEvoTag]{
				Config: attrs.Config{Column: "tag_id", RelForeignKey: attrs.Relate(&MigEvoTag{}, "", nil)},
			}
		}),
	).WithTableName("mig_evo_article_tag")
}

type MigEvoArticle struct {
	ID       int64 `attrs:"primary"`
	Title    string
	Author   *MigEvoAuthor
	Bio      string
	Score    int64
	Priority int64
	Rating   decimal.Decimal
	Summary  string
	Computed string
	Tags     any
}

func (m *MigEvoArticle) DatabaseIndexes(attrs.Definer) []migrator.Index {
	idxs := []migrator.Index{
		{Identifier: "mig_evo_article_title_idx", Fields: []string{"Title"}, Unique: false},
	}
	if migEvoState.articleCompositeIdx == 1 {
		idxs = append(idxs, migrator.Index{
			Identifier: "mig_evo_article_title_summary_idx",
			Fields:     []string{"Title", "Summary"},
			Unique:     false,
		})
	} else if migEvoState.articleCompositeIdx == 2 {
		idxs = append(idxs, migrator.Index{
			Identifier: "mig_evo_article_title_summary_idx",
			Fields:     []string{"Title", "Summary"},
			Unique:     true,
		})
	}
	return idxs
}

func (m *MigEvoArticle) FieldDefs(ctx context.Context) attrs.Definitions {
	flds := []attrs.Field{
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*MigEvoArticle, int64] {
			return fattrs.PtrFieldConfig[*MigEvoArticle, int64]{Config: attrs.Config{Primary: true}}
		}),
		fattrs.Field(m, "Title", &m.Title, func() fattrs.PtrFieldConfig[*MigEvoArticle, string] {
			return fattrs.PtrFieldConfig[*MigEvoArticle, string]{Config: attrs.Config{MaxLength: 120}}
		}),
		fattrs.Field(m, "Author", &m.Author, func() fattrs.PtrFieldConfig[*MigEvoArticle, *MigEvoAuthor] {
			return fattrs.PtrFieldConfig[*MigEvoArticle, *MigEvoAuthor]{
				Config: attrs.Config{
					Column:        "author_id",
					RelForeignKey: attrs.Relate(&MigEvoAuthor{}, "", nil),
				},
			}
		}),
		attrs.NewField(m, "Tags", &attrs.Config{
			Column: "id",
			RelManyToMany: attrs.Relate(&MigEvoTag{}, "", &attrs.ThroughModel{
				This: &MigEvoArticleTag{}, Source: "Article", Target: "Tag",
			}),
		}),
	}

	if migEvoState.addArticleFields {
		flds = append(flds, attrs.NewField(m, "Bio"))
		if migEvoState.alterScoreAndSummary {
			flds = append(flds, attrs.NewField(m, "Score", &attrs.Config{Column: "points", Default: int64(99)}))
		} else {
			flds = append(flds, attrs.NewField(m, "Score", &attrs.Config{Column: "score", Default: int64(10)}))
		}
		if migEvoState.alterPriorityNotNull {
			flds = append(flds, attrs.NewField(m, "Priority", &attrs.Config{Null: false}))
		} else {
			flds = append(flds, attrs.NewField(m, "Priority", &attrs.Config{Null: true}))
		}

		if !migEvoState.removeArticleFields {
			sumLen := int64(200)
			if migEvoState.alterScoreAndSummary {
				sumLen = 500
			}
			flds = append(flds,
				attrs.NewField(m, "Rating", &attrs.Config{Null: true, Precision: 10, Scale: 2}),
				attrs.NewField(m, "Summary", &attrs.Config{MaxLength: sumLen}),
			)
		}

		flds = append(flds, attrs.NewField(m, "Computed", &attrs.Config{
			Attributes: map[string]any{migrator.AttrUseInDBKey: false},
		}))
	}

	return attrs.Make(ctx, m, flds...).WithTableName("mig_evo_article")
}

func TestMigrator_RealSQLite_FullSchemaLifecycle(t *testing.T) {
	resetMigEvoState()
	t.Cleanup(resetMigEvoState)

	db, editor := openTestSQLite(t)
	initMigTestApp(t, db, func() django.AppConfig {
		cfg := apps.NewAppConfig("migevo")
		cfg.ModelObjects = []attrs.Definer{
			&MigEvoAuthor{},
			&MigEvoTag{},
			&MigEvoArticle{},
			&MigEvoArticleTag{},
		}
		return cfg
	})

	ctx := context.Background()
	tmpDir := t.TempDir()
	engine := migrator.NewMigrationEngine(tmpDir, editor)
	engine.MigrationLog = noopMigLog{}

	// Step 1: CreateTable + FK + Index + UniqueTogether + M2M Through
	pendingMake, err := engine.NeedsToMakeMigrations(ctx)
	if err != nil {
		t.Fatalf("Step 1 NeedsToMakeMigrations: %v", err)
	}
	if len(pendingMake) != 4 {
		t.Fatalf("Step 1 expected 4 models needing migrations, got %d", len(pendingMake))
	}
	if err := engine.MakeMigrations(ctx); err != nil {
		t.Fatalf("Step 1 MakeMigrations: %v", err)
	}
	articleMig := readMigrationFileJSON(t, filepath.Join(tmpDir, "migevo", "MigEvoArticle", "0001_create_table_and_1_more.mig"))
	depsRaw, _ := articleMig["dependencies"].([]any)
	if len(depsRaw) == 0 || depsRaw[0] != "migevo:MigEvoAuthor:0001_create_table.mig" {
		t.Fatalf("expected MigEvoArticle dependency on MigEvoAuthor, got %v", articleMig["dependencies"])
	}

	pendingApply, err := engine.NeedsToMigrate(ctx)
	if err != nil {
		t.Fatalf("Step 1 NeedsToMigrate: %v", err)
	}
	if len(pendingApply) != 4 {
		t.Fatalf("Step 1 expected 4 pending migrations, got %d", len(pendingApply))
	}
	if err := engine.Migrate(ctx); err != nil {
		t.Fatalf("Step 1 Migrate: %v", err)
	}

	tbls := sqliteTables(t, db)
	for _, wantTbl := range []string{"mig_evo_author", "mig_evo_tag", "mig_evo_article", "mig_evo_article_tag"} {
		if !tbls[wantTbl] {
			t.Fatalf("Step 1 missing table %q in sqlite_master", wantTbl)
		}
	}
	artIdxs := sqliteIndexes(t, db, "mig_evo_article")
	if _, ok := artIdxs["mig_evo_article_title_idx"]; !ok {
		t.Errorf("Step 1 expected mig_evo_article_title_idx, got %v", artIdxs)
	}
	thruIdxs := sqliteIndexes(t, db, "mig_evo_article_tag")
	if isUnique, ok := thruIdxs["Article_Tag_unique_together_idx"]; !ok || !isUnique {
		t.Errorf("Step 1 expected unique through index, got %v", thruIdxs)
	}
	fks := sqliteForeignKeys(t, db, "mig_evo_article")
	if len(fks) != 1 || fks[0].Table != "mig_evo_author" || fks[0].From != "author_id" {
		t.Errorf("Step 1 expected FK to mig_evo_author(id), got %+v", fks)
	}

	// Idempotency checks
	if err := engine.MakeMigrations(ctx); !errors.Is(err, migrator.ErrNoChanges) {
		t.Errorf("expected ErrNoChanges on repeated MakeMigrations, got %v", err)
	}
	if err := engine.Migrate(ctx); !errors.Is(err, migrator.ErrNoChanges) {
		t.Errorf("expected ErrNoChanges on repeated Migrate, got %v", err)
	}

	// Seed rows before adding columns
	if _, err := db.ExecContext(ctx, "INSERT INTO mig_evo_author (name) VALUES ('Alice')"); err != nil {
		t.Fatalf("insert author: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO mig_evo_article (title, author_id) VALUES ('First Post', 1)"); err != nil {
		t.Fatalf("insert article: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO mig_evo_tag (label) VALUES ('golang')"); err != nil {
		t.Fatalf("insert tag: %v", err)
	}

	// Step 2: AddField (including UseInDB=false field)
	migEvoState.addArticleFields = true
	_ = attrs.ResetDefinitions.Send(ctx, nil)
	if err := engine.MakeMigrations(ctx); err != nil {
		t.Fatalf("Step 2 MakeMigrations: %v", err)
	}
	if err := engine.Migrate(ctx); err != nil {
		t.Fatalf("Step 2 Migrate: %v", err)
	}

	cols := sqliteColumns(t, db, "mig_evo_article")
	for _, wantCol := range []string{"bio", "score", "priority", "rating", "summary"} {
		if _, ok := cols[wantCol]; !ok {
			t.Errorf("Step 2 missing added column %q", wantCol)
		}
	}
	if _, ok := cols["computed"]; ok {
		t.Errorf("Step 2 AttrUseInDBKey=false column 'computed' must not be created")
	}

	var scoreVal int64
	var priorityVal sql.NullInt64
	if err := db.QueryRowContext(ctx, "SELECT score, priority FROM mig_evo_article WHERE id = 1").Scan(&scoreVal, &priorityVal); err != nil {
		t.Fatalf("query article after Step 2: %v", err)
	}
	if scoreVal != 10 || priorityVal.Valid {
		t.Errorf("Step 2 existing row score=%d priority.Valid=%v, want 10 / false", scoreVal, priorityVal.Valid)
	}

	// Step 3: AlterField Priority (NULL -> NOT NULL via COALESCE), Score (rename score->points, change default), & Summary (max_length) in one migration
	migEvoState.alterPriorityNotNull = true
	migEvoState.alterScoreAndSummary = true
	_ = attrs.ResetDefinitions.Send(ctx, nil)
	if err := engine.MakeMigrations(ctx); err != nil {
		t.Fatalf("Step 3 MakeMigrations: %v", err)
	}
	if err := engine.Migrate(ctx); err != nil {
		t.Fatalf("Step 3 Migrate: %v", err)
	}

	cols = sqliteColumns(t, db, "mig_evo_article")
	if _, ok := cols["points"]; !ok {
		t.Errorf("Step 3 expected renamed column 'points'")
	}
	if _, ok := cols["score"]; ok {
		t.Errorf("Step 3 old column 'score' should no longer exist after rename")
	}
	if !cols["priority"].NotNull {
		t.Errorf("Step 3 expected 'priority' NOT NULL after AlterField")
	}
	if cols["points"].DfltValue.String != "99" {
		t.Errorf("Step 3 expected points default '99', got %+v", cols["points"].DfltValue)
	}

	var titleAfter string
	var pointsAfter, priorityAfter int64
	if err := db.QueryRowContext(ctx, "SELECT title, points, priority FROM mig_evo_article WHERE id = 1").Scan(&titleAfter, &pointsAfter, &priorityAfter); err != nil {
		t.Fatalf("query article after Step 3: %v", err)
	}
	if titleAfter != "First Post" || pointsAfter != 10 || priorityAfter != 0 {
		t.Errorf("Step 3 row data after AlterField = (%q, %d, %d), want (\"First Post\", 10, 0)", titleAfter, pointsAfter, priorityAfter)
	}
	artIdxs = sqliteIndexes(t, db, "mig_evo_article")
	if _, ok := artIdxs["mig_evo_article_title_idx"]; !ok {
		t.Errorf("Step 3 expected mig_evo_article_title_idx preserved across AlterField rebuild, got %v", artIdxs)
	}

	// Step 4: Add, Modify (unique change), and Drop Indexes
	migEvoState.articleCompositeIdx = 1
	_ = attrs.ResetDefinitions.Send(ctx, nil)
	if err := engine.MakeMigrations(ctx); err != nil {
		t.Fatalf("Step 4a MakeMigrations (AddIndex): %v", err)
	}
	if err := engine.Migrate(ctx); err != nil {
		t.Fatalf("Step 4a Migrate (AddIndex): %v", err)
	}
	artIdxs = sqliteIndexes(t, db, "mig_evo_article")
	if uniq, ok := artIdxs["mig_evo_article_title_summary_idx"]; !ok || uniq {
		t.Errorf("Step 4a expected non-unique mig_evo_article_title_summary_idx, got %v", artIdxs)
	}

	// Step 4b: Modify composite index (non-unique -> unique)
	migEvoState.articleCompositeIdx = 2
	_ = attrs.ResetDefinitions.Send(ctx, nil)
	assertNoPanic(t, func() {
		if err := engine.MakeMigrations(ctx); err != nil {
			t.Fatalf("Step 4b MakeMigrations (ModifyIndex): %v", err)
		}
		if err := engine.Migrate(ctx); err != nil {
			t.Fatalf("Step 4b Migrate (ModifyIndex): %v", err)
		}
	})
	artIdxs = sqliteIndexes(t, db, "mig_evo_article")
	if uniq, ok := artIdxs["mig_evo_article_title_summary_idx"]; !ok || !uniq {
		t.Errorf("Step 4b expected unique mig_evo_article_title_summary_idx, got %v", artIdxs)
	}

	// Step 4c: Drop composite index via MakeMigrations + Migrate
	migEvoState.articleCompositeIdx = 0
	_ = attrs.ResetDefinitions.Send(ctx, nil)
	assertNoPanic(t, func() {
		if err := engine.MakeMigrations(ctx); err != nil {
			t.Fatalf("Step 4c MakeMigrations (DropIndex): %v", err)
		}
		if err := engine.Migrate(ctx); err != nil {
			t.Fatalf("Step 4c Migrate (DropIndex): %v", err)
		}
	})
	artIdxs = sqliteIndexes(t, db, "mig_evo_article")
	if _, ok := artIdxs["mig_evo_article_title_summary_idx"]; ok {
		t.Errorf("Step 4c expected mig_evo_article_title_summary_idx dropped")
	}
	if _, ok := artIdxs["mig_evo_article_title_idx"]; !ok {
		t.Errorf("Step 4c expected mig_evo_article_title_idx to remain")
	}

	// Step 5: RemoveField
	migEvoState.removeArticleFields = true
	_ = attrs.ResetDefinitions.Send(ctx, nil)
	if err := engine.MakeMigrations(ctx); err != nil {
		t.Fatalf("Step 5 MakeMigrations (RemoveField): %v", err)
	}
	if err := engine.Migrate(ctx); err != nil {
		t.Fatalf("Step 5 Migrate (RemoveField): %v", err)
	}
	cols = sqliteColumns(t, db, "mig_evo_article")
	if _, ok := cols["summary"]; ok {
		t.Errorf("Step 5 column 'summary' should be removed")
	}
	if _, ok := cols["rating"]; ok {
		t.Errorf("Step 5 column 'rating' should be removed")
	}
	if err := engine.MakeMigrations(ctx); !errors.Is(err, migrator.ErrNoChanges) {
		t.Errorf("Step 5 expected ErrNoChanges after RemoveField, got %v", err)
	}

	// Step 6: RenameTable
	migEvoState.renameTagTable = true
	_ = attrs.ResetDefinitions.Send(ctx, nil)
	if err := engine.MakeMigrations(ctx); err != nil {
		t.Fatalf("Step 6 MakeMigrations (RenameTable): %v", err)
	}
	if err := engine.Migrate(ctx); err != nil {
		t.Fatalf("Step 6 Migrate (RenameTable): %v", err)
	}
	tbls = sqliteTables(t, db)
	if tbls["mig_evo_tag"] || !tbls["mig_evo_tag_v2"] {
		t.Fatalf("Step 6 expected mig_evo_tag renamed to mig_evo_tag_v2, got %v", tbls)
	}
	var tagLabel string
	if err := db.QueryRowContext(ctx, "SELECT label FROM mig_evo_tag_v2 WHERE id = 1").Scan(&tagLabel); err != nil || tagLabel != "golang" {
		t.Errorf("Step 6 row survival check failed: label=%q err=%v", tagLabel, err)
	}

	// Step 7: DropTable (explicit ActionDropTable migration file)
	dropJSON := `{
  "table": {
    "table": "mig_evo_tag_v2",
    "model": "github.com/Nigel2392/go-django/queries/src/migrator_test.MigEvoTag",
    "fields": [],
    "indexes": null,
    "comment": ""
  },
  "actions": [
    {
      "action": "drop_table",
      "table": {
        "old": {
          "table": "mig_evo_tag_v2",
          "model": "github.com/Nigel2392/go-django/queries/src/migrator_test.MigEvoTag",
          "fields": [],
          "indexes": null,
          "comment": ""
        }
      }
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "migevo", "MigEvoTag", "0003_drop_table.mig"), []byte(dropJSON), 0644); err != nil {
		t.Fatalf("write drop_table mig: %v", err)
	}
	if err := engine.Migrate(ctx); err != nil {
		t.Fatalf("Step 7 Migrate (DropTable): %v", err)
	}
	tbls = sqliteTables(t, db)
	if tbls["mig_evo_tag_v2"] {
		t.Errorf("Step 7 expected mig_evo_tag_v2 dropped")
	}
}

// --- Models for TestMigrator_Dependencies_LazyDeps_Cycles_Missing_And_FS ---

type migDepTarget struct {
	ID   int64 `attrs:"primary"`
	Name string
}

func (m *migDepTarget) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make[*migDepTarget, attrs.Field](ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*migDepTarget, int64] {
			return fattrs.PtrFieldConfig[*migDepTarget, int64]{Config: attrs.Config{Primary: true}}
		}),
		attrs.NewField(m, "Name"),
	).WithTableName("mig_dep_target")
}

type migDepChild struct {
	ID     int64 `attrs:"primary"`
	Target *migDepTarget
}

func (m *migDepChild) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*migDepChild, int64] {
			return fattrs.PtrFieldConfig[*migDepChild, int64]{Config: attrs.Config{Primary: true}}
		}),
		fattrs.Field(m, "Target", &m.Target, func() fattrs.PtrFieldConfig[*migDepChild, *migDepTarget] {
			return fattrs.PtrFieldConfig[*migDepChild, *migDepTarget]{
				Config: attrs.Config{Column: "target_id", RelForeignKey: attrs.Relate(&migDepTarget{}, "", nil)},
			}
		}),
	).WithTableName("mig_dep_child")
}

type migSelfCategory struct {
	ID     int64 `attrs:"primary"`
	Title  string
	Parent *migSelfCategory
}

func (m *migSelfCategory) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make[*migSelfCategory, attrs.Field](ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*migSelfCategory, int64] {
			return fattrs.PtrFieldConfig[*migSelfCategory, int64]{Config: attrs.Config{Primary: true}}
		}),
		attrs.NewField(m, "Title"),
		fattrs.Field(m, "Parent", &m.Parent, func() fattrs.PtrFieldConfig[*migSelfCategory, *migSelfCategory] {
			return fattrs.PtrFieldConfig[*migSelfCategory, *migSelfCategory]{
				Config: attrs.Config{Column: "parent_id", Null: true, RelForeignKey: attrs.Relate(&migSelfCategory{}, "", nil)},
			}
		}),
	).WithTableName("mig_self_category")
}

type migLazyTarget struct {
	ID   int64 `attrs:"primary"`
	Name string
}

func (m *migLazyTarget) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make[*migLazyTarget, attrs.Field](ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*migLazyTarget, int64] {
			return fattrs.PtrFieldConfig[*migLazyTarget, int64]{Config: attrs.Config{Primary: true}}
		}),
		attrs.NewField(m, "Name"),
	).WithTableName("mig_lazy_target")
}

type migLazySource struct {
	ID     int64 `attrs:"primary"`
	Target attrs.Definer
}

func (m *migLazySource) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make[*migLazySource, attrs.Field](ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*migLazySource, int64] {
			return fattrs.PtrFieldConfig[*migLazySource, int64]{Config: attrs.Config{Primary: true}}
		}),
		attrs.NewField(m, "Target", &attrs.Config{
			Column:        "target_id",
			Null:          true,
			RelForeignKey: attrs.RelatedDeferred(attrs.RelManyToOne, "miglazy.Target", "", nil),
		}),
	).WithTableName("mig_lazy_source")
}

type migCycleA struct{ ID int64 `attrs:"primary"` }

func (m *migCycleA) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions(ctx, m)
}

type migCycleB struct{ ID int64 `attrs:"primary"` }

func (m *migCycleB) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions(ctx, m)
}

type migFSModel struct {
	ID   int64 `attrs:"primary"`
	Name string
}

func (m *migFSModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make[*migFSModel, attrs.Field](ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*migFSModel, int64] {
			return fattrs.PtrFieldConfig[*migFSModel, int64]{Config: attrs.Config{Primary: true}}
		}),
		attrs.NewField(m, "Name"),
	).WithTableName("mig_fs_model")
}

type migHookFSModel struct {
	ID    int64 `attrs:"primary"`
	Title string
}

func (m *migHookFSModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make[*migHookFSModel, attrs.Field](ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*migHookFSModel, int64] {
			return fattrs.PtrFieldConfig[*migHookFSModel, int64]{Config: attrs.Config{Primary: true}}
		}),
		attrs.NewField(m, "Title"),
	).WithTableName("mig_hook_fs_model")
}

func TestMigrator_Dependencies_LazyDeps_Cycles_Missing_And_FS(t *testing.T) {
	ctx := context.Background()

	t.Run("CrossApp_SelfRef_And_LazyDeps", func(t *testing.T) {
		db, editor := openTestSQLite(t)
		if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON;"); err != nil {
			t.Fatalf("enable foreign_keys: %v", err)
		}

		_ = contenttypes.NewLazyRegistry("miglazy.Target", func(def *contenttypes.ContentTypeDefinition) bool {
			_, ok := def.ContentObject.(*migLazyTarget)
			return ok
		})
		registerModelsContentTypes(&migLazyTarget{}, &migDepTarget{}, &migDepChild{}, &migSelfCategory{}, &migLazySource{})

		initMigTestApp(t, db,
			func() django.AppConfig {
				cfg := apps.NewAppConfig("migdep_b") // listed BEFORE target app intentionally
				cfg.ModelObjects = []attrs.Definer{&migDepChild{}, &migSelfCategory{}, &migLazySource{}}
				return cfg
			},
			func() django.AppConfig {
				cfg := apps.NewAppConfig("migdep_a")
				cfg.ModelObjects = []attrs.Definer{&migDepTarget{}, &migLazyTarget{}}
				return cfg
			},
		)

		tmpDir := t.TempDir()
		engine := migrator.NewMigrationEngine(tmpDir, editor)
		engine.MigrationLog = noopMigLog{}

		if err := engine.MakeMigrations(ctx); err != nil {
			t.Fatalf("MakeMigrations: %v", err)
		}

		// Self-ref must NOT add self-dependency.
		selfMig := readMigrationFileJSON(t, filepath.Join(tmpDir, "migdep_b", "migSelfCategory", "0001_create_table.mig"))
		if deps, ok := selfMig["dependencies"]; ok && len(deps.([]any)) > 0 {
			t.Errorf("self-referential FK should not emit self-dependency, got %v", deps)
		}

		// Lazy FK must emit lazy_dependencies.
		lazyMig := readMigrationFileJSON(t, filepath.Join(tmpDir, "migdep_b", "migLazySource", "0001_create_table.mig"))
		if ldeps, ok := lazyMig["lazy_dependencies"].([]any); !ok || len(ldeps) != 1 || ldeps[0] != "miglazy.Target" {
			t.Errorf("expected lazy_dependencies=[miglazy.Target], got %v", lazyMig["lazy_dependencies"])
		}

		if err := engine.Migrate(ctx); err != nil {
			t.Fatalf("Migrate: %v", err)
		}
		tbls := sqliteTables(t, db)
		for _, want := range []string{"mig_dep_target", "mig_dep_child", "mig_self_category", "mig_lazy_target", "mig_lazy_source"} {
			if !tbls[want] {
				t.Errorf("missing migrated table %q", want)
			}
		}
	})

	t.Run("CyclicDependencyError", func(t *testing.T) {
		db, editor := openTestSQLite(t)
		initMigTestApp(t, db, func() django.AppConfig {
			cfg := apps.NewAppConfig("migcyc")
			cfg.ModelObjects = []attrs.Definer{&migCycleA{}, &migCycleB{}}
			return cfg
		})
		tmpDir := t.TempDir()
		engine := migrator.NewMigrationEngine(tmpDir, editor)
		engine.MigrationLog = noopMigLog{}

		aMig := `{
  "dependencies": ["migcyc:migCycleB:0001_create_table.mig"],
  "table": {"table":"mig_cyc_a","model":"github.com/Nigel2392/go-django/queries/src/migrator_test.migCycleA","fields":[{"name":"ID","column":"id","use_in_db":true,"primary":true,"auto":true,"db_type":"INT"}],"indexes":null},
  "actions": [{"action":"create_table"}]
}`
		bMig := `{
  "dependencies": ["migcyc:migCycleA:0001_create_table.mig"],
  "table": {"table":"mig_cyc_b","model":"github.com/Nigel2392/go-django/queries/src/migrator_test.migCycleB","fields":[{"name":"ID","column":"id","use_in_db":true,"primary":true,"auto":true,"db_type":"INT"}],"indexes":null},
  "actions": [{"action":"create_table"}]
}`
		if err := os.MkdirAll(filepath.Join(tmpDir, "migcyc", "migCycleA"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(tmpDir, "migcyc", "migCycleB"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "migcyc", "migCycleA", "0001_create_table.mig"), []byte(aMig), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "migcyc", "migCycleB", "0001_create_table.mig"), []byte(bMig), 0644); err != nil {
			t.Fatal(err)
		}

		err := engine.Migrate(ctx)
		if err == nil || !strings.Contains(err.Error(), "cyclic dependency") {
			t.Fatalf("expected cyclic dependency error, got %v", err)
		}
	})

	t.Run("MissingDependencyError", func(t *testing.T) {
		db, editor := openTestSQLite(t)
		initMigTestApp(t, db, func() django.AppConfig {
			cfg := apps.NewAppConfig("migmiss")
			cfg.ModelObjects = []attrs.Definer{&migCycleA{}}
			return cfg
		})
		tmpDir := t.TempDir()
		engine := migrator.NewMigrationEngine(tmpDir, editor)
		engine.MigrationLog = noopMigLog{}

		badMig := `{
  "dependencies": ["migmiss:NonExistentModel:0001_create_table.mig"],
  "table": {"table":"mig_cyc_a","model":"github.com/Nigel2392/go-django/queries/src/migrator_test.migCycleA","fields":[{"name":"ID","column":"id","use_in_db":true,"primary":true,"auto":true,"db_type":"INT"}],"indexes":null},
  "actions": [{"action":"create_table"}]
}`
		if err := os.MkdirAll(filepath.Join(tmpDir, "migmiss", "migCycleA"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "migmiss", "migCycleA", "0001_create_table.mig"), []byte(badMig), 0644); err != nil {
			t.Fatal(err)
		}

		err := engine.Migrate(ctx)
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected missing dependency error, got %v", err)
		}
	})

	t.Run("EmbeddedFS_And_RegisterFileSystemHook", func(t *testing.T) {
		// Panic validation
		for _, tc := range []struct {
			app string
			fs  fstest.MapFS
		}{
			{"", fstest.MapFS{}},
			{"app", nil},
		} {
			func() {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("expected RegisterFileSystem(%q, %v) to panic", tc.app, tc.fs)
					}
				}()
				if tc.fs == nil {
					migrator.RegisterFileSystem(tc.app, nil)
				} else {
					migrator.RegisterFileSystem(tc.app, tc.fs)
				}
			}()
		}

		fsAppMig := fstest.MapFS{
			"migFSModel/0001_create_table.mig": {
				Data: []byte(`{"table":{"table":"mig_fs_model","model":"github.com/Nigel2392/go-django/queries/src/migrator_test.migFSModel","fields":[{"name":"ID","column":"id","use_in_db":true,"primary":true,"auto":true,"db_type":"INT"},{"name":"Name","column":"name","use_in_db":true,"nullable":true,"default":{"go_type":"string","value":""},"db_type":"STRING"}],"indexes":null},"actions":[{"action":"create_table"}]}`),
			},
		}
		hookFS := fstest.MapFS{
			"migHookFSModel/0001_create_table.mig": {
				Data: []byte(`{"table":{"table":"mig_hook_fs_model","model":"github.com/Nigel2392/go-django/queries/src/migrator_test.migHookFSModel","fields":[{"name":"ID","column":"id","use_in_db":true,"primary":true,"auto":true,"db_type":"INT"},{"name":"Title","column":"title","use_in_db":true,"nullable":true,"default":{"go_type":"string","value":""},"db_type":"STRING"}],"indexes":null},"actions":[{"action":"create_table"}]}`),
			},
		}
		migrator.RegisterFileSystem("migfs_hook", hookFS)

		db, editor := openTestSQLite(t)
		initMigTestApp(t, db,
			func() django.AppConfig {
				base := apps.NewAppConfig("migfs_app")
				base.ModelObjects = []attrs.Definer{&migFSModel{}}
				return &migrator.MigratorAppConfig{
					AppConfig:   base,
					MigrationFS: fsAppMig,
				}
			},
			func() django.AppConfig {
				cfg := apps.NewAppConfig("migfs_hook")
				cfg.ModelObjects = []attrs.Definer{&migHookFSModel{}}
				return cfg
			},
		)

		engine := migrator.NewMigrationEngine(t.TempDir(), editor)
		engine.MigrationLog = noopMigLog{}

		if err := engine.Migrate(ctx); err != nil {
			t.Fatalf("Migrate from FS failed: %v", err)
		}
		tbls := sqliteTables(t, db)
		if !tbls["mig_fs_model"] || !tbls["mig_hook_fs_model"] {
			t.Fatalf("expected tables created from MigrationFS and hook FS, got %v", tbls)
		}
		if err := engine.MakeMigrations(ctx); !errors.Is(err, migrator.ErrNoChanges) {
			t.Errorf("expected ErrNoChanges when FS matches model defs, got %v", err)
		}
	})
}

// --- Models & Tests for Fake Mode and AutoMigrate ---

type migFakeModel struct {
	ID   int64 `attrs:"primary"`
	Name string
}

func (m *migFakeModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make[*migFakeModel, attrs.Field](ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*migFakeModel, int64] {
			return fattrs.PtrFieldConfig[*migFakeModel, int64]{Config: attrs.Config{Primary: true}}
		}),
		attrs.NewField(m, "Name"),
	).WithTableName("mig_fake_model")
}

var migAutoExtraCol bool

type migAutoModel struct {
	ID    int64 `attrs:"primary"`
	Title string
	Extra string
}

func (m *migAutoModel) FieldDefs(ctx context.Context) attrs.Definitions {
	flds := []attrs.Field{
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*migAutoModel, int64] {
			return fattrs.PtrFieldConfig[*migAutoModel, int64]{Config: attrs.Config{Primary: true}}
		}),
		attrs.NewField(m, "Title"),
	}
	if migAutoExtraCol {
		flds = append(flds, attrs.NewField(m, "Extra"))
	}
	return attrs.Make(ctx, m, flds...).WithTableName("mig_auto_model")
}

func TestMigrator_FakeMode_And_AutoMigrate(t *testing.T) {
	ctx := context.Background()

	t.Run("FakeMode", func(t *testing.T) {
		db, editor := openTestSQLite(t)
		initMigTestApp(t, db, func() django.AppConfig {
			cfg := apps.NewAppConfig("migfake")
			cfg.ModelObjects = []attrs.Definer{&migFakeModel{}}
			return cfg
		})

		tmpDir := t.TempDir()
		fakeEngine := migrator.NewMigrationEngine(tmpDir, editor)
		fakeEngine.Fake = true
		fakeEngine.MigrationLog = noopMigLog{}

		// MakeMigrations in Fake mode returns nil but writes 0 files.
		if err := fakeEngine.MakeMigrations(ctx); err != nil {
			t.Fatalf("Fake MakeMigrations failed: %v", err)
		}
		if cnt := countMigFiles(t, tmpDir); cnt != 0 {
			t.Fatalf("expected 0 files written in Fake MakeMigrations, got %d", cnt)
		}

		// Now write real migration files, then run Migrate in Fake mode:
		// in Engine.Migrate, Fake skips all actions AND skips StoreMigration?
		// Wait: engine.go line 336-342 has `if m.Fake { continue }` before StoreMigration.
		realEngine := migrator.NewMigrationEngine(tmpDir, editor)
		realEngine.MigrationLog = noopMigLog{}
		if err := realEngine.MakeMigrations(ctx); err != nil {
			t.Fatalf("real MakeMigrations failed: %v", err)
		}
		if err := fakeEngine.Migrate(ctx); err != nil {
			t.Fatalf("Fake Migrate failed: %v", err)
		}
		tbls := sqliteTables(t, db)
		if tbls["mig_fake_model"] {
			t.Errorf("expected mig_fake_model NOT created in Fake Migrate mode")
		}
		has, err := editor.HasMigration(ctx, "migfake", "migFakeModel", "0001_create_table.mig")
		if err != nil {
			t.Fatalf("HasMigration: %v", err)
		}
		if has {
			t.Errorf("expected Fake Migrate not to store migration row")
		}
	})

	t.Run("AutoMigrateOptionsPanics", func(t *testing.T) {
		for name, call := range map[string]func(){
			"EmptyDir": func() { migrator.AutoMigrateMigrationDir("") },
			"NilDB":    func() { migrator.AutoMigrateDatabase(nil) },
			"NilLog":   func() { migrator.AutoMigrateLog(nil) },
		} {
			func() {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("expected %s option to panic", name)
					}
				}()
				call()
			}()
		}
	})

	t.Run("AutoMigrateLifecycle", func(t *testing.T) {
		migAutoExtraCol = false
		t.Cleanup(func() { migAutoExtraCol = false })

		db, _ := openTestSQLite(t)
		initMigTestApp(t, db, func() django.AppConfig {
			cfg := apps.NewAppConfig("migauto")
			cfg.ModelObjects = []attrs.Definer{&migAutoModel{}}
			return cfg
		})

		tmpDir := t.TempDir()
		opts := []migrator.AutoMigrateOption{
			migrator.AutoMigrateDatabase(db),
			migrator.AutoMigrateMigrationDir(tmpDir),
			migrator.AutoMigrateApps("migauto"),
			migrator.AutoMigrateLog(noopMigLog{}),
		}

		// First AutoMigrate: creates migration & applies it
		made, migrated, err := migrator.AutoMigrate(ctx, opts...)
		if err != nil {
			t.Fatalf("AutoMigrate #1 failed: %v", err)
		}
		if !made || !migrated {
			t.Errorf("AutoMigrate #1 = (made=%v, migrated=%v), want (true, true)", made, migrated)
		}
		if !sqliteTables(t, db)["mig_auto_model"] {
			t.Fatalf("expected mig_auto_model created")
		}

		// Second AutoMigrate: no-op
		made, migrated, err = migrator.AutoMigrate(ctx, opts...)
		if err != nil {
			t.Fatalf("AutoMigrate #2 failed: %v", err)
		}
		if made || migrated {
			t.Errorf("AutoMigrate #2 = (made=%v, migrated=%v), want (false, false)", made, migrated)
		}

		// Third AutoMigrate after schema addition
		migAutoExtraCol = true
		_ = attrs.ResetDefinitions.Send(ctx, nil)
		made, migrated, err = migrator.AutoMigrate(ctx, opts...)
		if err != nil {
			t.Fatalf("AutoMigrate #3 failed: %v", err)
		}
		if !made || !migrated {
			t.Errorf("AutoMigrate #3 = (made=%v, migrated=%v), want (true, true)", made, migrated)
		}
		if _, ok := sqliteColumns(t, db, "mig_auto_model")["extra"]; !ok {
			t.Errorf("expected added column 'extra' in mig_auto_model")
		}

		// Wrapper functions NeedsToMakeMigrations / NeedsToMigrate
		needMake, err := migrator.NeedsToMakeMigrations(ctx, opts...)
		if err != nil || len(needMake) != 0 {
			t.Errorf("NeedsToMakeMigrations = (%d, %v), want (0, nil)", len(needMake), err)
		}
		needApply, err := migrator.NeedsToMigrate(ctx, opts...)
		if err != nil || len(needApply) != 0 {
			t.Errorf("NeedsToMigrate = (%d, %v), want (0, nil)", len(needApply), err)
		}
	})
}

// --- Models & Tests for RegisterMigrateFunc / ExecMigrateFunc / ActionExecGoCode ---

type MigExecItem struct {
	models.Model
	ID   int64 `attrs:"primary"`
	Name string
	Slug string
}

func (m *MigExecItem) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*MigExecItem, int64] {
			return fattrs.PtrFieldConfig[*MigExecItem, int64]{Config: attrs.Config{Primary: true}}
		}),
		attrs.NewField(m, "Name"),
		attrs.NewField(m, "Slug"),
	).WithTableName("mig_exec_item")
}

type migExecUnregisteredModel struct {
	ID int64 `attrs:"primary"`
}

func (m *migExecUnregisteredModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*migExecUnregisteredModel, int64] {
			return fattrs.PtrFieldConfig[*migExecUnregisteredModel, int64]{Config: attrs.Config{Primary: true}}
		}),
	).WithTableName("mig_exec_unreg")
}

func TestRegisterMigrateFunc_And_ActionExecGoCode(t *testing.T) {
	ctx := context.Background()

	// 1. Validation & panic checks on RegisterMigrateFunc
	noopFn := func(context.Context, *migrator.MigrationEngine, *migrator.ModelTable) error { return nil }
	for name, fn := range map[string]func(){
		"NilModel":    func() { migrator.RegisterMigrateFunc(nil, "0001_x.mig", noopFn) },
		"NilFunc":     func() { migrator.RegisterMigrateFunc(&MigExecItem{}, "0001_x.mig", nil) },
		"EmptyFile":   func() { migrator.RegisterMigrateFunc(&MigExecItem{}, "", noopFn) },
	} {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected RegisterMigrateFunc (%s) to panic", name)
				}
			}()
			fn()
		}()
	}

	// Duplicate registration panic check (unique filename not reused elsewhere)
	migrator.RegisterMigrateFunc(&MigExecItem{}, "0099_dup_check.mig", noopFn)
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected duplicate RegisterMigrateFunc to panic")
			}
		}()
		migrator.RegisterMigrateFunc(&MigExecItem{}, "0099_dup_check.mig", noopFn)
	}()

	// 2. Direct error checks on ExecMigrateFunc
	if err := migrator.ExecMigrateFunc(ctx, nil, "x.mig", &migrator.ModelTable{}); !errors.Is(err, derrors.NilPointer) {
		t.Errorf("expected NilPointer for nil Object, got %v", err)
	}
	if err := migrator.ExecMigrateFunc(ctx, nil, "", &migrator.ModelTable{Object: &MigExecItem{}}); !errors.Is(err, derrors.ValueError) {
		t.Errorf("expected ValueError for empty filename, got %v", err)
	}
	if err := migrator.ExecMigrateFunc(ctx, nil, "x.mig", &migrator.ModelTable{Object: &migExecUnregisteredModel{}}); !errors.Is(err, derrors.NotImplemented) {
		t.Errorf("expected NotImplemented for unregistered model, got %v", err)
	}
	if err := migrator.ExecMigrateFunc(ctx, nil, "nonexistent.mig", &migrator.ModelTable{Object: &MigExecItem{}}); !errors.Is(err, derrors.NotImplemented) {
		t.Errorf("expected NotImplemented for unregistered filename, got %v", err)
	}

	// 3. End-to-end execution + transaction rollback + unregistered migration in Migrate()
	var execCalls atomic.Int32
	migrator.RegisterMigrateFunc(&MigExecItem{}, "0002_backfill_slugs.mig", func(ctx context.Context, eng *migrator.MigrationEngine, tbl *migrator.ModelTable) error {
		execCalls.Add(1)
		if eng == nil {
			return fmt.Errorf("expected non-nil MigrationEngine")
		}
		if tbl == nil || tbl.TableName() != "mig_exec_item" {
			return fmt.Errorf("unexpected table: %+v", tbl)
		}
		if migrator.DbFromContext(ctx, nil) == nil {
			return fmt.Errorf("expected transaction DB in context")
		}
		_, err := eng.SchemaEditor.Execute(ctx, `UPDATE mig_exec_item SET slug = LOWER(REPLACE(name, ' ', '-')) WHERE slug = ''`)
		return err
	})

	errBoom := errors.New("intentional migrate func failure")
	migrator.RegisterMigrateFunc(&MigExecItem{}, "0003_failing_exec.mig", func(ctx context.Context, eng *migrator.MigrationEngine, tbl *migrator.ModelTable) error {
		if _, err := eng.SchemaEditor.Execute(ctx, `UPDATE mig_exec_item SET slug = 'dirty'`); err != nil {
			return err
		}
		return errBoom
	})

	db, editor := openTestSQLite(t)
	initMigTestApp(t, db, func() django.AppConfig {
		cfg := apps.NewAppConfig("migexec")
		cfg.ModelObjects = []attrs.Definer{&MigExecItem{}, &migExecUnregisteredModel{}}
		return cfg
	})

	tmpDir := t.TempDir()
	engine := migrator.NewMigrationEngine(tmpDir, editor)
	engine.MigrationLog = noopMigLog{}

	if err := engine.MakeMigrations(ctx, "migexec"); err != nil {
		t.Fatalf("MakeMigrations: %v", err)
	}
	if err := engine.Migrate(ctx, "migexec"); err != nil {
		t.Fatalf("Migrate 0001: %v", err)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO mig_exec_item (name, slug) VALUES ('Hello World', ''), ('Go Django', '')`); err != nil {
		t.Fatalf("insert test rows: %v", err)
	}

	execMigJSON := `{
  "dependencies": ["migexec:MigExecItem:0001_create_table.mig"],
  "table": null,
  "actions": [{"action": "exec"}]
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "migexec", "MigExecItem", "0002_backfill_slugs.mig"), []byte(execMigJSON), 0644); err != nil {
		t.Fatalf("write 0002 exec migration: %v", err)
	}

	if err := engine.Migrate(ctx, "migexec"); err != nil {
		t.Fatalf("Migrate 0002 exec: %v", err)
	}
	if execCalls.Load() != 1 {
		t.Errorf("expected registered func called once, got %d", execCalls.Load())
	}

	var slug1, slug2 string
	if err := db.QueryRowContext(ctx, "SELECT slug FROM mig_exec_item WHERE name = 'Hello World'").Scan(&slug1); err != nil {
		t.Fatalf("query slug1: %v", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT slug FROM mig_exec_item WHERE name = 'Go Django'").Scan(&slug2); err != nil {
		t.Fatalf("query slug2: %v", err)
	}
	if slug1 != "hello-world" || slug2 != "go-django" {
		t.Errorf("backfilled slugs = (%q, %q), want (\"hello-world\", \"go-django\")", slug1, slug2)
	}

	// Re-run Migrate & MakeMigrations: idempotent, func not called again
	if err := engine.Migrate(ctx, "migexec"); !errors.Is(err, migrator.ErrNoChanges) {
		t.Errorf("expected ErrNoChanges on re-run Migrate, got %v", err)
	}
	if execCalls.Load() != 1 {
		t.Errorf("expected callback not re-executed on idempotent Migrate, count=%d", execCalls.Load())
	}
	if err := engine.MakeMigrations(ctx, "migexec"); !errors.Is(err, migrator.ErrNoChanges) {
		t.Errorf("expected ErrNoChanges on MakeMigrations after exec migration with table:null, got %v", err)
	}

	// Verify rollback when registered func returns an error
	failMigJSON := `{
  "dependencies": ["migexec:MigExecItem:0002_backfill_slugs.mig"],
  "table": null,
  "actions": [{"action": "exec"}]
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "migexec", "MigExecItem", "0003_failing_exec.mig"), []byte(failMigJSON), 0644); err != nil {
		t.Fatalf("write 0003 failing exec migration: %v", err)
	}
	if err := engine.Migrate(ctx, "migexec"); !errors.Is(err, errBoom) {
		t.Fatalf("expected errBoom from failing exec migration, got %v", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT slug FROM mig_exec_item WHERE name = 'Hello World'").Scan(&slug1); err != nil {
		t.Fatalf("query slug1 after rollback: %v", err)
	}
	if slug1 != "hello-world" {
		t.Errorf("expected transaction rollback preserving slug1=\"hello-world\", got %q", slug1)
	}
	hasFailMig, err := editor.HasMigration(ctx, "migexec", "MigExecItem", "0003_failing_exec.mig")
	if err != nil {
		t.Fatalf("HasMigration 0003: %v", err)
	}
	if hasFailMig {
		t.Errorf("expected 0003_failing_exec.mig NOT recorded after rollback")
	}

	// Unregistered exec migration error during Migrate
	_, editorUnreg := openTestSQLite(t)
	tmpUnreg := t.TempDir()
	unregEngine := migrator.NewMigrationEngine(tmpUnreg, editorUnreg)
	unregEngine.MigrationLog = noopMigLog{}
	unregExecJSON := `{
  "dependencies": ["migexec:migExecUnregisteredModel:0001_create_table.mig"],
  "table": null,
  "actions": [{"action": "exec"}]
}`
	if err := os.MkdirAll(filepath.Join(tmpUnreg, "migexec", "migExecUnregisteredModel"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpUnreg, "migexec", "migExecUnregisteredModel", "0001_create_table.mig"), []byte(`{"table":{"table":"mig_exec_unreg","model":"github.com/Nigel2392/go-django/queries/src/migrator_test.migExecUnregisteredModel","fields":[{"name":"ID","column":"id","use_in_db":true,"primary":true,"auto":true,"db_type":"INT"}],"indexes":null},"actions":[{"action":"create_table"}]}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpUnreg, "migexec", "migExecUnregisteredModel", "0002_missing_go_code.mig"), []byte(unregExecJSON), 0644); err != nil {
		t.Fatal(err)
	}
	err = unregEngine.Migrate(ctx, "migexec")
	if !errors.Is(err, derrors.NotImplemented) {
		t.Errorf("expected NotImplemented error on unregistered ActionExecGoCode, got %v", err)
	}
	if has, _ := editorUnreg.HasMigration(ctx, "migexec", "migExecUnregisteredModel", "0002_missing_go_code.mig"); has {
		t.Errorf("expected failed migration not to be stored")
	}

	// Silence unused import checks if any
	_ = uuid.Nil
}
