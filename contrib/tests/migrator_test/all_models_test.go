//go:build test

package migrator_test

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Nigel2392/go-django/contrib/auth"
	"github.com/Nigel2392/go-django/contrib/auth/users"
	"github.com/Nigel2392/go-django/contrib/blocks"
	"github.com/Nigel2392/go-django/contrib/documents"
	"github.com/Nigel2392/go-django/contrib/editor"
	"github.com/Nigel2392/go-django/contrib/images"
	"github.com/Nigel2392/go-django/contrib/openauth2"
	"github.com/Nigel2392/go-django/contrib/pages"
	auditlogs "github.com/Nigel2392/go-django/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/contrib/revisions"
	"github.com/Nigel2392/go-django/contrib/session"
	"github.com/Nigel2392/go-django/contrib/settings"
	"github.com/Nigel2392/go-django/contrib/shop/carts"
	"github.com/Nigel2392/go-django/contrib/shop/coupons"
	"github.com/Nigel2392/go-django/contrib/shop/customers"
	shopmodels "github.com/Nigel2392/go-django/contrib/shop/models"
	"github.com/Nigel2392/go-django/contrib/shop/orders"
	"github.com/Nigel2392/go-django/contrib/shop/payments"
	"github.com/Nigel2392/go-django/contrib/shop/products"
	"github.com/Nigel2392/go-django/contrib/shop/products/supplier"
	_ "github.com/Nigel2392/go-django/contrib/shop/shared/relations/modelkeys"
	_ "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/Nigel2392/go-django/queries/src/expr/constraints"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	_ "github.com/Nigel2392/go-django/queries/src/migrator/sql/mysql"
	_ "github.com/Nigel2392/go-django/queries/src/migrator/sql/postgres"
	"github.com/Nigel2392/go-django/queries/src/migrator/sql/sqlite"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles/memory"
)

var contribDBSeq atomic.Int64

type noopMigLog struct{}

func (noopMigLog) Log(migrator.ActionType, *migrator.MigrationFile, *migrator.Changed[*migrator.ModelTable], *migrator.Changed[*migrator.Column], *migrator.Changed[*migrator.Index], *migrator.Changed[constraints.Constraint]) {
}

// contribRichContentModel exercises contrib/blocks and contrib/editor DB types alongside contrib models.
type contribRichContentModel struct {
	ID      int64 `attrs:"primary"`
	Blocks  *blocks.ListBlockValue
	Stream  *blocks.StreamBlockValue
	Content *editor.EditorJSBlockData
}

func (m *contribRichContentModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make[*contribRichContentModel, attrs.Field](ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(m, "Blocks", &attrs.Config{Null: true}),
		attrs.NewField(m, "Stream", &attrs.Config{Null: true}),
		attrs.NewField(m, "Content", &attrs.Config{Null: true}),
	).WithTableName("contrib_rich_content_model")
}

func openFreshSQLite(t *testing.T) (drivers.Database, *sqlite.SQLiteSchemaEditor) {
	t.Helper()
	dsn := fmt.Sprintf("file:contrib_mig_%d?mode=memory&cache=shared", contribDBSeq.Add(1))
	db, err := drivers.Open(context.Background(), "sqlite3", dsn)
	if err != nil {
		t.Fatalf("failed to open sqlite db %q: %v", dsn, err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db, sqlite.NewSQLiteSchemaEditor(db)
}

func ensureMediaMemoryBackend() {
	if _, ok := mediafiles.RetrieveBackend("memory"); !ok {
		mediafiles.RegisterBackend("memory", memory.NewBackend(memory.DEFAULT_RETRIES))
	}
	if mediafiles.GetDefault() == nil {
		mediafiles.SetDefault("memory")
	}
}

type appModelGroup struct {
	name   string
	models []attrs.Definer
	fs     fs.FS
}

func allContribAppGroups() []appModelGroup {
	return []appModelGroup{
		{
			name:   "session",
			models: []attrs.Definer{&session.Session{}},
		},
		{
			name:   "settings",
			models: []attrs.Definer{&settings.RegisteredSetting{}},
		},
		{
			name: "auth",
			models: []attrs.Definer{
				&auth.User{},
				&users.Group{},
				&users.Permission{},
				&users.UserGroup{},
				&users.GroupPermission{},
				&users.UserPermission{},
			},
		},
		{
			name:   "documents",
			models: []attrs.Definer{&documents.Document{}},
		},
		{
			name:   "images",
			models: []attrs.Definer{&images.Image{}},
		},
		{
			name:   "pages",
			models: []attrs.Definer{&pages.Site{}, &pages.PageNode{}},
		},
		{
			name:   "revisions",
			models: []attrs.Definer{&revisions.Revision{}},
		},
		{
			name:   "auditlogs",
			models: []attrs.Definer{&auditlogs.Entry{}},
		},
		{
			name: "shop",
			models: []attrs.Definer{
				&shopmodels.Category{},
				&supplier.SupplierModel{},
				&products.Product{},
				&carts.Cart{},
				&carts.CartItem{},
				&customers.Customer{},
				&payments.Payment{},
				&orders.OrderState{},
				&orders.Order{},
				&orders.OrderItem{},
				&coupons.Coupon{},
				&coupons.CouponProductThrough{},
				&coupons.Redemption{},
			},
		},
		{
			name:   "openauth2",
			models: []attrs.Definer{&openauth2.User{}},
		},
		{
			name:   "richfields",
			models: []attrs.Definer{&contribRichContentModel{}},
		},
	}
}

func flattenGroupModels(groups []appModelGroup) []attrs.Definer {
	var out []attrs.Definer
	for _, g := range groups {
		out = append(out, g.models...)
	}
	return out
}

func initContribTestApp(t *testing.T, db drivers.Database, groups []appModelGroup) {
	t.Helper()
	ensureMediaMemoryBackend()
	django.Global = nil
	dbtype.TYPES.Unlock()
	_ = attrs.ResetDefinitions.Send(context.Background(), nil)

	appArgs := make([]any, 0, len(groups))
	for _, g := range groups {
		grp := g
		appArgs = append(appArgs, func() django.AppConfig {
			cfg := apps.NewAppConfig(grp.name)
			cfg.ModelObjects = grp.models
			if grp.fs != nil {
				return &migrator.MigratorAppConfig{
					AppConfig:   cfg,
					MigrationFS: grp.fs,
				}
			}
			return cfg
		})
	}

	settingsMap := map[string]any{}
	if db != nil {
		settingsMap[django.APPVAR_DATABASE] = db
	}

	app := django.App(
		django.Configure(settingsMap),
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
}

func TestAllContribModels_TableAndDialectTypeMapping(t *testing.T) {
	groups := allContribAppGroups()
	initContribTestApp(t, nil, groups)

	dialects := []struct {
		name string
		drv  driver.Driver
	}{
		{"SQLite", &drivers.DriverSQLite{}},
		{"MySQL", &drivers.DriverMySQL{}},
		{"MariaDB", &drivers.DriverMariaDB{}},
		{"Postgres", &drivers.DriverPostgres{}},
	}

	allModels := flattenGroupModels(groups)
	if len(allModels) != 29 {
		t.Fatalf("expected 28 contrib models + 1 richfields model = 29, got %d", len(allModels))
	}

	for _, mdl := range allModels {
		modelType := fmt.Sprintf("%T", mdl)
		t.Run(modelType, func(t *testing.T) {
			tbl := migrator.NewModelTable(mdl)
			if tbl.TableName() == "" {
				t.Fatalf("%s has empty TableName()", modelType)
			}
			cols := tbl.Columns()
			if len(cols) == 0 {
				t.Fatalf("%s has zero columns", modelType)
			}
			for _, col := range cols {
				if !col.UseInDB {
					continue
				}
				for _, d := range dialects {
					sqlType := migrator.GetFieldType(d.drv, col)
					if sqlType == "" {
						t.Errorf("%s.%s (%s) mapped to empty %s SQL type", modelType, col.Name, col.DBType().String(), d.name)
					}
				}
			}
		})
	}
}

func TestAllContribModels_JSONRoundTripZeroDiff(t *testing.T) {
	groups := allContribAppGroups()
	initContribTestApp(t, nil, groups)

	if !dbtype.IsLocked() {
		dbtype.Lock()
	}
	_ = attrs.ResetDefinitions.Send(context.Background(), nil)

	for _, mdl := range flattenGroupModels(groups) {
		modelType := fmt.Sprintf("%T", mdl)
		t.Run(modelType, func(t *testing.T) {
			orig := migrator.NewModelTable(mdl)
			raw, err := json.Marshal(orig)
			if err != nil {
				t.Fatalf("json.Marshal(%s): %v", modelType, err)
			}
			var decoded migrator.ModelTable
			if err := json.Unmarshal(raw, &decoded); err != nil {
				t.Fatalf("json.Unmarshal(%s): %v", modelType, err)
			}
			added, removed, diffs := orig.Diff(&decoded)
			if len(added) != 0 || len(removed) != 0 || len(diffs) != 0 {
				for _, d := range diffs {
					t.Errorf("%s field %q diff: %v", modelType, d.New.Name, d.New.ChangeList(&d.Old))
				}
				t.Fatalf("%s JSON round-trip produced non-zero diff: added=%d removed=%d diffs=%d", modelType, len(added), len(removed), len(diffs))
			}
		})
	}
}

func sqliteTableSet(t *testing.T, db drivers.Database) map[string]bool {
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
			t.Fatal(err)
		}
		res[name] = true
	}
	return res
}

func sqliteColumnSet(t *testing.T, db drivers.Database, table string) map[string]bool {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), fmt.Sprintf("PRAGMA table_info(`%s`)", table))
	if err != nil {
		t.Fatalf("PRAGMA table_info(%s): %v", table, err)
	}
	defer rows.Close()

	res := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt any
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			t.Fatal(err)
		}
		res[name] = true
	}
	return res
}

func sqliteIndexSet(t *testing.T, db drivers.Database, table string) map[string]bool {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), fmt.Sprintf("PRAGMA index_list(`%s`)", table))
	if err != nil {
		t.Fatalf("PRAGMA index_list(%s): %v", table, err)
	}
	defer rows.Close()

	res := make(map[string]bool)
	for rows.Next() {
		var seq int
		var name string
		var unique int
		var origin string
		var partial int
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			t.Fatal(err)
		}
		res[name] = true
	}
	return res
}

func TestAllContribModels_MakeMigrationsAndMigrateSQLite(t *testing.T) {
	ctx := context.Background()
	db, editor := openFreshSQLite(t)
	groups := allContribAppGroups()
	initContribTestApp(t, db, groups)

	migDir := t.TempDir()
	engine := migrator.NewMigrationEngine(migDir, editor)
	engine.MigrationLog = noopMigLog{}

	if err := engine.MakeMigrations(ctx); err != nil {
		t.Fatalf("MakeMigrations failed: %v", err)
	}

	if err := engine.Migrate(ctx); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	tbls := sqliteTableSet(t, db)
	for _, mdl := range flattenGroupModels(groups) {
		mt := migrator.NewModelTable(mdl)
		if !tbls[mt.TableName()] {
			t.Errorf("missing migrated table %q for %T", mt.TableName(), mdl)
			continue
		}

		dbCols := sqliteColumnSet(t, db, mt.TableName())
		for _, col := range mt.Columns() {
			if col.UseInDB && !dbCols[col.Column] {
				t.Errorf("table %q (%T) missing DB column %q", mt.TableName(), mdl, col.Column)
			}
			if !col.UseInDB && dbCols[col.Column] && !strings.EqualFold(col.Column, "id") {
				t.Logf("note: non-UseInDB col %q on %T happens to share name with DB col", col.Column, mdl)
			}
		}

		dbIdxs := sqliteIndexSet(t, db, mt.TableName())
		for _, idx := range mt.Indexes() {
			if !dbIdxs[idx.Name()] {
				t.Errorf("table %q (%T) missing index %q", mt.TableName(), mdl, idx.Name())
			}
		}
	}

	// Idempotency checks
	if err := engine.MakeMigrations(ctx); !errors.Is(err, migrator.ErrNoChanges) {
		t.Errorf("expected ErrNoChanges after full migrate, got %v", err)
	}
	pending, err := engine.NeedsToMigrate(ctx)
	if err != nil {
		t.Fatalf("NeedsToMigrate failed: %v", err)
	}
	if len(pending) != 0 {
		t.Errorf("expected 0 pending migrations after full migrate, got %d", len(pending))
	}
}

func TestAllContribModels_ShippedEmbeddedMigrationsApplyCleanly(t *testing.T) {
	ctx := context.Background()

	t.Run("CoreContribShippedMigrations", func(t *testing.T) {
		db, editor := openFreshSQLite(t)
		groups := []appModelGroup{
			{
				name:   "session",
				models: []attrs.Definer{&session.Session{}},
				fs:     os.DirFS("../session/migrations/session"),
			},
			{
				name:   "settings",
				models: []attrs.Definer{&settings.RegisteredSetting{}},
				fs:     os.DirFS("../settings/migrations/settings"),
			},
			{
				name: "auth",
				models: []attrs.Definer{
					&auth.User{},
					&users.Group{},
					&users.Permission{},
					&users.UserGroup{},
					&users.GroupPermission{},
					&users.UserPermission{},
				},
				fs: os.DirFS("../auth/migrations/auth"),
			},
			{
				name:   "documents",
				models: []attrs.Definer{&documents.Document{}},
				fs:     os.DirFS("../documents/migrations/documents"),
			},
			{
				name:   "images",
				models: []attrs.Definer{&images.Image{}},
				fs:     os.DirFS("../images/migrations/images"),
			},
			{
				name:   "pages",
				models: []attrs.Definer{&pages.Site{}, &pages.PageNode{}},
				fs:     os.DirFS("../pages/migrations/pages"),
			},
			{
				name:   "revisions",
				models: []attrs.Definer{&revisions.Revision{}},
				fs:     os.DirFS("../revisions/migrations/revisions"),
			},
			{
				name:   "auditlogs",
				models: []attrs.Definer{&auditlogs.Entry{}},
				fs:     os.DirFS("../reports/audit_logs/migrations/auditlogs"),
			},
		}

		initContribTestApp(t, db, groups)
		engine := migrator.NewMigrationEngine(t.TempDir(), editor)
		engine.MigrationLog = noopMigLog{}

		if err := engine.Migrate(ctx); err != nil {
			t.Fatalf("CoreContribShippedMigrations Migrate failed: %v", err)
		}

		tbls := sqliteTableSet(t, db)
		for _, mdl := range flattenGroupModels(groups) {
			tblName := migrator.NewModelTable(mdl).TableName()
			if !tbls[tblName] {
				t.Errorf("shipped migrations did not create expected table %q (%T)", tblName, mdl)
			}
		}

		needMake, err := engine.NeedsToMakeMigrations(ctx)
		if err != nil {
			t.Fatalf("CoreContribShippedMigrations NeedsToMakeMigrations failed: %v", err)
		}
		if len(needMake) != 0 {
			var drifted []string
			for _, ct := range needMake {
				drifted = append(drifted, fmt.Sprintf("%s.%s", ct.ShortTypeName(), ct.Model()))
			}
			t.Errorf("shipped migrations have drifted from model definitions (%d models need migrations): %v", len(needMake), drifted)
		}
	})

	t.Run("OpenAuth2ShippedMigrations", func(t *testing.T) {
		db, editor := openFreshSQLite(t)
		groups := []appModelGroup{
			{
				name: "openauth2",
				models: []attrs.Definer{
					&openauth2.User{},
					&users.Group{},
					&users.Permission{},
					&users.UserGroup{},
					&users.GroupPermission{},
					&users.UserPermission{},
				},
				fs: os.DirFS("../openauth2/migrations/openauth2"),
			},
		}

		initContribTestApp(t, db, groups)
		engine := migrator.NewMigrationEngine(t.TempDir(), editor)
		engine.MigrationLog = noopMigLog{}

		if err := engine.Migrate(ctx); err != nil {
			t.Fatalf("OpenAuth2ShippedMigrations Migrate failed: %v", err)
		}

		tbls := sqliteTableSet(t, db)
		for _, mdl := range flattenGroupModels(groups) {
			tblName := migrator.NewModelTable(mdl).TableName()
			if !tbls[tblName] {
				t.Errorf("openauth2 shipped migrations did not create table %q (%T)", tblName, mdl)
			}
		}

		needMake, err := engine.NeedsToMakeMigrations(ctx)
		if err != nil {
			t.Fatalf("OpenAuth2ShippedMigrations NeedsToMakeMigrations failed: %v", err)
		}
		if len(needMake) != 0 {
			var drifted []string
			for _, ct := range needMake {
				drifted = append(drifted, fmt.Sprintf("%s.%s", ct.ShortTypeName(), ct.Model()))
			}
			t.Errorf("openauth2 shipped migrations have drifted from model definitions (%d models need migrations): %v", len(needMake), drifted)
		}
	})
}
