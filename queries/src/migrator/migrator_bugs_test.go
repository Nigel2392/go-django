package migrator_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/queries/src/expr/constraints"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/shopspring/decimal"
)

// 1. TestBug_MakeMigrations_DropIndexPanic

var bugDropIdxEnabled bool

type bugDropIdxModel struct {
	ID    int64 `attrs:"primary"`
	Title string
}

func (m *bugDropIdxModel) DatabaseIndexes(attrs.Definer) []migrator.Index {
	if bugDropIdxEnabled {
		return []migrator.Index{
			{Identifier: "idx_bug_drop_title", Fields: []string{"Title"}, Unique: false},
		}
	}
	return nil
}

func (m *bugDropIdxModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(m, "Title", &attrs.Config{MaxLength: 100}),
	).WithTableName("bug_drop_idx_model")
}

func TestBug_MakeMigrations_DropIndexPanic(t *testing.T) {
	bugDropIdxEnabled = true
	t.Cleanup(func() { bugDropIdxEnabled = false })

	ctx := context.Background()
	db, editor := openTestSQLite(t)
	initMigTestApp(t, db, func() django.AppConfig {
		cfg := apps.NewAppConfig("bugdropidx")
		cfg.ModelObjects = []attrs.Definer{&bugDropIdxModel{}}
		return cfg
	})

	tmpDir := t.TempDir()
	engine := migrator.NewMigrationEngine(tmpDir, editor)
	engine.MigrationLog = noopMigLog{}

	if err := engine.MakeMigrations(ctx); err != nil {
		t.Fatalf("Step 1 MakeMigrations: %v", err)
	}
	if err := engine.Migrate(ctx); err != nil {
		t.Fatalf("Step 1 Migrate: %v", err)
	}
	if _, ok := sqliteIndexes(t, db, "bug_drop_idx_model")["idx_bug_drop_title"]; !ok {
		t.Fatalf("expected idx_bug_drop_title created in Step 1")
	}

	bugDropIdxEnabled = false
	_ = attrs.ResetDefinitions.Send(ctx, nil)

	assertNoPanic(t, func() {
		if err := engine.MakeMigrations(ctx); err != nil {
			t.Fatalf("Step 2 MakeMigrations (DropIndex): %v", err)
		}
		if err := engine.Migrate(ctx); err != nil {
			t.Fatalf("Step 2 Migrate (DropIndex): %v", err)
		}
	})

	if _, ok := sqliteIndexes(t, db, "bug_drop_idx_model")["idx_bug_drop_title"]; ok {
		t.Errorf("expected idx_bug_drop_title to be dropped")
	}
}

// 2. TestBug_MakeMigrations_ModifyIndex

var bugModIdxUnique bool

type bugModIdxModel struct {
	ID    int64 `attrs:"primary"`
	Title string
}

func (m *bugModIdxModel) DatabaseIndexes(attrs.Definer) []migrator.Index {
	return []migrator.Index{
		{Identifier: "idx_bug_mod_title", Fields: []string{"Title"}, Unique: bugModIdxUnique},
	}
}

func (m *bugModIdxModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(m, "Title", &attrs.Config{MaxLength: 100}),
	).WithTableName("bug_mod_idx_model")
}

func TestBug_MakeMigrations_ModifyIndex(t *testing.T) {
	bugModIdxUnique = false
	t.Cleanup(func() { bugModIdxUnique = false })

	ctx := context.Background()
	db, editor := openTestSQLite(t)
	initMigTestApp(t, db, func() django.AppConfig {
		cfg := apps.NewAppConfig("bugmodidx")
		cfg.ModelObjects = []attrs.Definer{&bugModIdxModel{}}
		return cfg
	})

	tmpDir := t.TempDir()
	engine := migrator.NewMigrationEngine(tmpDir, editor)
	engine.MigrationLog = noopMigLog{}

	if err := engine.MakeMigrations(ctx); err != nil {
		t.Fatalf("Step 1 MakeMigrations: %v", err)
	}
	if err := engine.Migrate(ctx); err != nil {
		t.Fatalf("Step 1 Migrate: %v", err)
	}
	if uniq, ok := sqliteIndexes(t, db, "bug_mod_idx_model")["idx_bug_mod_title"]; !ok || uniq {
		t.Fatalf("expected non-unique idx_bug_mod_title in Step 1, ok=%v uniq=%v", ok, uniq)
	}

	bugModIdxUnique = true
	_ = attrs.ResetDefinitions.Send(ctx, nil)

	assertNoPanic(t, func() {
		if err := engine.MakeMigrations(ctx); err != nil {
			t.Fatalf("Step 2 MakeMigrations (ModifyIndex): %v", err)
		}
		if err := engine.Migrate(ctx); err != nil {
			t.Fatalf("Step 2 Migrate (ModifyIndex): %v", err)
		}
	})

	if uniq, ok := sqliteIndexes(t, db, "bug_mod_idx_model")["idx_bug_mod_title"]; !ok || !uniq {
		t.Errorf("expected unique idx_bug_mod_title after Step 2, ok=%v uniq=%v", ok, uniq)
	}
}

// 3. TestBug_MakeMigrations_RenameIndex

var bugRenameIdxName = "bug_idx_old"

type bugRenameIdxModel struct {
	ID    int64 `attrs:"primary"`
	Title string
}

func (m *bugRenameIdxModel) DatabaseIndexes(attrs.Definer) []migrator.Index {
	return []migrator.Index{
		{Identifier: bugRenameIdxName, Fields: []string{"Title"}, Unique: false},
	}
}

func (m *bugRenameIdxModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(m, "Title", &attrs.Config{MaxLength: 100}),
	).WithTableName("bug_rename_idx_model")
}

func TestBug_MakeMigrations_RenameIndex(t *testing.T) {
	bugRenameIdxName = "bug_idx_old"
	t.Cleanup(func() { bugRenameIdxName = "bug_idx_old" })

	ctx := context.Background()
	db, editor := openTestSQLite(t)
	initMigTestApp(t, db, func() django.AppConfig {
		cfg := apps.NewAppConfig("bugrenidx")
		cfg.ModelObjects = []attrs.Definer{&bugRenameIdxModel{}}
		return cfg
	})

	tmpDir := t.TempDir()
	engine := migrator.NewMigrationEngine(tmpDir, editor)
	engine.MigrationLog = noopMigLog{}

	if err := engine.MakeMigrations(ctx); err != nil {
		t.Fatalf("Step 1 MakeMigrations: %v", err)
	}
	if err := engine.Migrate(ctx); err != nil {
		t.Fatalf("Step 1 Migrate: %v", err)
	}

	bugRenameIdxName = "bug_idx_new"
	_ = attrs.ResetDefinitions.Send(ctx, nil)

	assertNoPanic(t, func() {
		if err := engine.MakeMigrations(ctx); err != nil {
			t.Fatalf("Step 2 MakeMigrations (RenameIndex): %v", err)
		}
		if err := engine.Migrate(ctx); err != nil {
			t.Fatalf("Step 2 Migrate (RenameIndex): %v", err)
		}
	})

	entries, err := os.ReadDir(filepath.Join(tmpDir, "bugrenidx", "bugRenameIdxModel"))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	foundRenameAction := false
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "0002_") {
			m := readMigrationFileJSON(t, filepath.Join(tmpDir, "bugrenidx", "bugRenameIdxModel", entry.Name()))
			actions, _ := m["actions"].([]any)
			for _, raw := range actions {
				act, _ := raw.(map[string]any)
				if act["action"] == "rename_index" {
					foundRenameAction = true
				}
			}
		}
	}
	if !foundRenameAction {
		t.Errorf("expected migration 0002 to contain rename_index action")
	}

	idxs := sqliteIndexes(t, db, "bug_rename_idx_model")
	if _, ok := idxs["bug_idx_new"]; !ok {
		t.Errorf("expected bug_idx_new in sqliteIndexes, got %v", idxs)
	}
	if _, ok := idxs["bug_idx_old"]; ok {
		t.Errorf("expected bug_idx_old removed from sqliteIndexes, got %v", idxs)
	}
}

// 4. TestBug_SQLite_MultiColumnAlterField_NotNullAndRenameOrder

var bugMultiNotNullStep2 bool

type bugMultiAlterNotNullModel struct {
	ID       int64 `attrs:"primary"`
	Title    string
	Priority int64
}

func (m *bugMultiAlterNotNullModel) FieldDefs(ctx context.Context) attrs.Definitions {
	titleLen := int64(50)
	prioNull := true
	if bugMultiNotNullStep2 {
		titleLen = 100
		prioNull = false
	}
	return attrs.Make(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(m, "Title", &attrs.Config{MaxLength: titleLen}),
		attrs.NewField(m, "Priority", &attrs.Config{Null: prioNull}),
	).WithTableName("bug_multi_alter_notnull")
}

var bugMultiRenameStep2 bool

type bugMultiAlterRenameModel struct {
	ID      int64 `attrs:"primary"`
	Summary string
	Score   int64
}

func (m *bugMultiAlterRenameModel) FieldDefs(ctx context.Context) attrs.Definitions {
	sumLen := int64(50)
	scoreCol := "score"
	if bugMultiRenameStep2 {
		sumLen = 200
		scoreCol = "points"
	}
	return attrs.Make(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(m, "Summary", &attrs.Config{MaxLength: sumLen}),
		attrs.NewField(m, "Score", &attrs.Config{Column: scoreCol}),
	).WithTableName("bug_multi_alter_rename")
}

func TestBug_SQLite_MultiColumnAlterField_NotNullAndRenameOrder(t *testing.T) {
	ctx := context.Background()

	t.Run("NotNullOnSecondAlteredColumn", func(t *testing.T) {
		bugMultiNotNullStep2 = false
		t.Cleanup(func() { bugMultiNotNullStep2 = false })

		db, editor := openTestSQLite(t)
		initMigTestApp(t, db, func() django.AppConfig {
			cfg := apps.NewAppConfig("bugmultinn")
			cfg.ModelObjects = []attrs.Definer{&bugMultiAlterNotNullModel{}}
			return cfg
		})

		tmpDir := t.TempDir()
		engine := migrator.NewMigrationEngine(tmpDir, editor)
		engine.MigrationLog = noopMigLog{}

		if err := engine.MakeMigrations(ctx); err != nil {
			t.Fatalf("Step 1 MakeMigrations: %v", err)
		}
		if err := engine.Migrate(ctx); err != nil {
			t.Fatalf("Step 1 Migrate: %v", err)
		}

		if _, err := db.ExecContext(ctx, "INSERT INTO bug_multi_alter_notnull (title, priority) VALUES ('t1', NULL)"); err != nil {
			t.Fatalf("seed row: %v", err)
		}

		bugMultiNotNullStep2 = true
		_ = attrs.ResetDefinitions.Send(ctx, nil)
		if err := engine.MakeMigrations(ctx); err != nil {
			t.Fatalf("Step 2 MakeMigrations: %v", err)
		}
		if err := engine.Migrate(ctx); err != nil {
			t.Fatalf("Step 2 Migrate: %v", err)
		}

		cols := sqliteColumns(t, db, "bug_multi_alter_notnull")
		if !cols["priority"].NotNull {
			t.Errorf("expected priority NOT NULL after Step 2")
		}
		var prio int64
		if err := db.QueryRowContext(ctx, "SELECT priority FROM bug_multi_alter_notnull WHERE id = 1").Scan(&prio); err != nil {
			t.Fatalf("query priority: %v", err)
		}
		if prio != 0 {
			t.Errorf("expected coerced NULL priority = 0, got %d", prio)
		}
	})

	t.Run("RenameOnSecondAlteredColumn", func(t *testing.T) {
		bugMultiRenameStep2 = false
		t.Cleanup(func() { bugMultiRenameStep2 = false })

		db, editor := openTestSQLite(t)
		initMigTestApp(t, db, func() django.AppConfig {
			cfg := apps.NewAppConfig("bugmultiren")
			cfg.ModelObjects = []attrs.Definer{&bugMultiAlterRenameModel{}}
			return cfg
		})

		tmpDir := t.TempDir()
		engine := migrator.NewMigrationEngine(tmpDir, editor)
		engine.MigrationLog = noopMigLog{}

		if err := engine.MakeMigrations(ctx); err != nil {
			t.Fatalf("Step 1 MakeMigrations: %v", err)
		}
		if err := engine.Migrate(ctx); err != nil {
			t.Fatalf("Step 1 Migrate: %v", err)
		}

		if _, err := db.ExecContext(ctx, "INSERT INTO bug_multi_alter_rename (summary, score) VALUES ('hi', 10)"); err != nil {
			t.Fatalf("seed row: %v", err)
		}

		bugMultiRenameStep2 = true
		_ = attrs.ResetDefinitions.Send(ctx, nil)
		if err := engine.MakeMigrations(ctx); err != nil {
			t.Fatalf("Step 2 MakeMigrations: %v", err)
		}
		if err := engine.Migrate(ctx); err != nil {
			t.Fatalf("Step 2 Migrate: %v", err)
		}

		cols := sqliteColumns(t, db, "bug_multi_alter_rename")
		if _, ok := cols["points"]; !ok {
			t.Errorf("expected renamed column points to exist")
		}
		if _, ok := cols["score"]; ok {
			t.Errorf("expected old column score to be removed")
		}
		var pts int64
		if err := db.QueryRowContext(ctx, "SELECT points FROM bug_multi_alter_rename WHERE id = 1").Scan(&pts); err != nil {
			t.Fatalf("query points: %v", err)
		}
		if pts != 10 {
			t.Errorf("expected points = 10, got %d", pts)
		}
	})
}

// 5. TestBug_ReplayMigrationHistory_AfterFieldRemoved

var bugReplayIncludeLegacy = true

type bugReplayRemovedModel struct {
	ID     int64 `attrs:"primary"`
	Title  string
	Legacy string
}

func (m *bugReplayRemovedModel) FieldDefs(ctx context.Context) attrs.Definitions {
	flds := []attrs.Field{
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(m, "Title"),
	}
	if bugReplayIncludeLegacy {
		flds = append(flds, attrs.NewField(m, "Legacy", &attrs.Config{Default: "old"}))
	}
	return attrs.Make(ctx, m, flds...).WithTableName("bug_replay_removed")
}

func TestBug_ReplayMigrationHistory_AfterFieldRemoved(t *testing.T) {
	bugReplayIncludeLegacy = true
	t.Cleanup(func() { bugReplayIncludeLegacy = true })

	ctx := context.Background()
	tmpDir := t.TempDir()

	db1, editor1 := openTestSQLite(t)
	initMigTestApp(t, db1, func() django.AppConfig {
		cfg := apps.NewAppConfig("bugreplay")
		cfg.ModelObjects = []attrs.Definer{&bugReplayRemovedModel{}}
		return cfg
	})
	engine1 := migrator.NewMigrationEngine(tmpDir, editor1)
	engine1.MigrationLog = noopMigLog{}

	if err := engine1.MakeMigrations(ctx); err != nil {
		t.Fatalf("Step 1 MakeMigrations: %v", err)
	}
	if err := engine1.Migrate(ctx); err != nil {
		t.Fatalf("Step 1 Migrate: %v", err)
	}

	bugReplayIncludeLegacy = false
	_ = attrs.ResetDefinitions.Send(ctx, nil)
	if err := engine1.MakeMigrations(ctx); err != nil {
		t.Fatalf("Step 2 MakeMigrations (remove Legacy): %v", err)
	}
	if err := engine1.Migrate(ctx); err != nil {
		t.Fatalf("Step 2 Migrate (remove Legacy): %v", err)
	}

	// Replay full migration history (0001 + 0002) onto a fresh DB #2 while Go model has no Legacy field.
	db2, editor2 := openTestSQLite(t)
	initMigTestApp(t, db2, func() django.AppConfig {
		cfg := apps.NewAppConfig("bugreplay")
		cfg.ModelObjects = []attrs.Definer{&bugReplayRemovedModel{}}
		return cfg
	})
	engine2 := migrator.NewMigrationEngine(tmpDir, editor2)
	engine2.MigrationLog = noopMigLog{}

	assertNoPanic(t, func() {
		if err := engine2.Migrate(ctx); err != nil {
			t.Fatalf("Replay Migrate on fresh DB failed: %v", err)
		}
	})

	cols := sqliteColumns(t, db2, "bug_replay_removed")
	if _, ok := cols["id"]; !ok {
		t.Errorf("expected id column in replayed table")
	}
	if _, ok := cols["title"]; !ok {
		t.Errorf("expected title column in replayed table")
	}
	if _, ok := cols["legacy"]; ok {
		t.Errorf("expected legacy column to not exist after replaying 0001+0002")
	}
}

// 6. TestBug_FattrsLazyRelationPreserved

type bugFattrsTarget struct {
	ID   int64 `attrs:"primary"`
	Name string
}

func (m *bugFattrsTarget) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make[*bugFattrsTarget, attrs.Field](ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*bugFattrsTarget, int64] {
			return fattrs.PtrFieldConfig[*bugFattrsTarget, int64]{Config: attrs.Config{Primary: true}}
		}),
		attrs.NewField(m, "Name"),
	).WithTableName("bug_fattrs_target")
}

type bugFattrsSource struct {
	ID     int64 `attrs:"primary"`
	Target *bugFattrsTarget
}

func (m *bugFattrsSource) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make[*bugFattrsSource, attrs.Field](ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*bugFattrsSource, int64] {
			return fattrs.PtrFieldConfig[*bugFattrsSource, int64]{Config: attrs.Config{Primary: true}}
		}),
		fattrs.Field(m, "Target", &m.Target, func() fattrs.PtrFieldConfig[*bugFattrsSource, *bugFattrsTarget] {
			return fattrs.PtrFieldConfig[*bugFattrsSource, *bugFattrsTarget]{
				Config: attrs.Config{
					Column:        "target_id",
					Null:          true,
					RelForeignKey: attrs.RelatedDeferred(attrs.RelManyToOne, "bugfattrs.Target", "", nil),
				},
			}
		}),
	).WithTableName("bug_fattrs_source")
}

func TestBug_FattrsLazyRelationPreserved(t *testing.T) {
	ctx := context.Background()
	_ = contenttypes.NewLazyRegistry("bugfattrs.Target", func(def *contenttypes.ContentTypeDefinition) bool {
		_, ok := def.ContentObject.(*bugFattrsTarget)
		return ok
	})
	registerModelsContentTypes(&bugFattrsTarget{}, &bugFattrsSource{})

	db, editor := openTestSQLite(t)
	initMigTestApp(t, db,
		func() django.AppConfig {
			cfg := apps.NewAppConfig("bugfattrstgt")
			cfg.ModelObjects = []attrs.Definer{&bugFattrsTarget{}}
			return cfg
		},
		func() django.AppConfig {
			cfg := apps.NewAppConfig("bugfattrssrc")
			cfg.ModelObjects = []attrs.Definer{&bugFattrsSource{}}
			return cfg
		},
	)

	tbl := migrator.NewModelTable(&bugFattrsSource{})
	targetCol, ok := tbl.Fields.Get("Target")
	if !ok || targetCol.Rel == nil {
		t.Fatalf("expected Target column with Rel")
	}
	if !targetCol.Rel.IsLazy() {
		t.Errorf("expected fattrs.Field deferred FK column to preserve IsLazy() == true")
	}

	tmpDir := t.TempDir()
	engine := migrator.NewMigrationEngine(tmpDir, editor)
	engine.MigrationLog = noopMigLog{}

	if err := engine.MakeMigrations(ctx); err != nil {
		t.Fatalf("MakeMigrations: %v", err)
	}

	srcMig := readMigrationFileJSON(t, filepath.Join(tmpDir, "bugfattrssrc", "bugFattrsSource", "0001_create_table.mig"))
	ldeps, _ := srcMig["lazy_dependencies"].([]any)
	if len(ldeps) != 1 || ldeps[0] != "bugfattrs.Target" {
		t.Errorf("expected lazy_dependencies=[\"bugfattrs.Target\"], got %v", srcMig["lazy_dependencies"])
	}
	if deps, hasDeps := srcMig["dependencies"].([]any); hasDeps && len(deps) > 0 {
		t.Errorf("expected no concrete dependencies when using RelatedDeferred, got %v", deps)
	}

	if err := engine.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
}

// 7. TestBug_NeedsToMakeMigrations_And_AutoMigrate_SkipCantMigrate

type bugSkipCantMigModel struct {
	migrator.CantMigrate
	ID int64 `attrs:"primary"`
}

func (m *bugSkipCantMigModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions(ctx, m)
}

type bugNormalMigModel struct {
	ID   int64 `attrs:"primary"`
	Name string
}

func (m *bugNormalMigModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(m, "Name"),
	).WithTableName("bug_normal_mig_model")
}

func TestBug_NeedsToMakeMigrations_And_AutoMigrate_SkipCantMigrate(t *testing.T) {
	ctx := context.Background()

	db1, editor1 := openTestSQLite(t)
	initMigTestApp(t, db1, func() django.AppConfig {
		cfg := apps.NewAppConfig("bugcantmig")
		cfg.ModelObjects = []attrs.Definer{&bugSkipCantMigModel{}, &bugNormalMigModel{}}
		return cfg
	})

	tmpDir1 := t.TempDir()
	engine1 := migrator.NewMigrationEngine(tmpDir1, editor1)
	engine1.MigrationLog = noopMigLog{}

	if err := engine1.MakeMigrations(ctx); err != nil {
		t.Fatalf("MakeMigrations: %v", err)
	}
	if err := engine1.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	need, err := engine1.NeedsToMakeMigrations(ctx)
	if err != nil {
		t.Fatalf("NeedsToMakeMigrations: %v", err)
	}
	if len(need) != 0 {
		t.Errorf("expected NeedsToMakeMigrations to return 0 after MakeMigrations (CantMigrate skipped), got %d", len(need))
	}

	// AutoMigrate on fresh DB + dir with the same app.
	db2, _ := openTestSQLite(t)
	initMigTestApp(t, db2, func() django.AppConfig {
		cfg := apps.NewAppConfig("bugcantmig")
		cfg.ModelObjects = []attrs.Definer{&bugSkipCantMigModel{}, &bugNormalMigModel{}}
		return cfg
	})
	tmpDir2 := t.TempDir()
	opts := []migrator.AutoMigrateOption{
		migrator.AutoMigrateDatabase(db2),
		migrator.AutoMigrateMigrationDir(tmpDir2),
		migrator.AutoMigrateApps("bugcantmig"),
		migrator.AutoMigrateLog(noopMigLog{}),
	}

	made, migrated, err := migrator.AutoMigrate(ctx, opts...)
	if err != nil || !made || !migrated {
		t.Fatalf("AutoMigrate #1 = (made=%v, migrated=%v, err=%v), want (true, true, nil)", made, migrated, err)
	}
	made, migrated, err = migrator.AutoMigrate(ctx, opts...)
	if err != nil || made || migrated {
		t.Fatalf("AutoMigrate #2 = (made=%v, migrated=%v, err=%v), want (false, false, nil)", made, migrated, err)
	}
}

// 8. TestBug_AutoMigrate_ReturnsInitConfigErrorInsteadOfPanicking

func TestBug_AutoMigrate_ReturnsInitConfigErrorInsteadOfPanicking(t *testing.T) {
	ctx := context.Background()

	t.Run("UninitializedDjango", func(t *testing.T) {
		oldGlobal := django.Global
		t.Cleanup(func() { django.Global = oldGlobal })
		django.Global = nil

		assertNoPanic(t, func() {
			_, _, err := migrator.AutoMigrate(ctx)
			if err == nil {
				t.Errorf("expected error when django.Global is nil, got nil")
			}
		})
	})

	t.Run("OptionReturnsError", func(t *testing.T) {
		db, _ := openTestSQLite(t)
		initMigTestApp(t, db, func() django.AppConfig {
			cfg := apps.NewAppConfig("bugautoerr")
			cfg.ModelObjects = []attrs.Definer{&bugNormalMigModel{}}
			return cfg
		})

		sentinel := errors.New("custom auto-migrate option error")
		badOpt := func(context.Context, *migrator.AutoMigrateConfig) error {
			return sentinel
		}

		assertNoPanic(t, func() {
			_, _, err := migrator.AutoMigrate(ctx, badOpt)
			if !errors.Is(err, sentinel) {
				t.Errorf("expected errors.Is(err, sentinel), got %v", err)
			}
		})
	})
}

// 9. TestBug_ReadMigrations_AppsFilterOnDisk

type bugAppOneModel struct {
	ID int64 `attrs:"primary"`
}

func (m *bugAppOneModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m, attrs.NewField(m, "ID", &attrs.Config{Primary: true})).WithTableName("bug_app_one_tbl")
}

type bugAppTwoModel struct {
	ID int64 `attrs:"primary"`
}

func (m *bugAppTwoModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m, attrs.NewField(m, "ID", &attrs.Config{Primary: true})).WithTableName("bug_app_two_tbl")
}

func TestBug_ReadMigrations_AppsFilterOnDisk(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	db1, editor1 := openTestSQLite(t)
	initMigTestApp(t, db1,
		func() django.AppConfig {
			cfg := apps.NewAppConfig("bugapp_one")
			cfg.ModelObjects = []attrs.Definer{&bugAppOneModel{}}
			return cfg
		},
		func() django.AppConfig {
			cfg := apps.NewAppConfig("bugapp_two")
			cfg.ModelObjects = []attrs.Definer{&bugAppTwoModel{}}
			return cfg
		},
	)
	engine1 := migrator.NewMigrationEngine(tmpDir, editor1)
	engine1.MigrationLog = noopMigLog{}

	if err := engine1.MakeMigrations(ctx); err != nil {
		t.Fatalf("MakeMigrations: %v", err)
	}

	db2, editor2 := openTestSQLite(t)
	initMigTestApp(t, db2,
		func() django.AppConfig {
			cfg := apps.NewAppConfig("bugapp_one")
			cfg.ModelObjects = []attrs.Definer{&bugAppOneModel{}}
			return cfg
		},
		func() django.AppConfig {
			cfg := apps.NewAppConfig("bugapp_two")
			cfg.ModelObjects = []attrs.Definer{&bugAppTwoModel{}}
			return cfg
		},
	)
	engine2 := migrator.NewMigrationEngine(tmpDir, editor2)
	engine2.MigrationLog = noopMigLog{}

	migs, err := engine2.ReadMigrations("bugapp_one")
	if err != nil {
		t.Fatalf("ReadMigrations(\"bugapp_one\"): %v", err)
	}
	if len(migs) == 0 {
		t.Fatalf("expected at least 1 migration for bugapp_one")
	}
	for _, m := range migs {
		if m.AppName != "bugapp_one" {
			t.Errorf("ReadMigrations(\"bugapp_one\") leaked migration from app %q (%s)", m.AppName, m.ModelName)
		}
	}

	if err := engine2.Migrate(ctx, "bugapp_one"); err != nil {
		t.Fatalf("Migrate(\"bugapp_one\"): %v", err)
	}
	tbls := sqliteTables(t, db2)
	if !tbls["bug_app_one_tbl"] {
		t.Errorf("expected bug_app_one_tbl to be created")
	}
	if tbls["bug_app_two_tbl"] {
		t.Errorf("bug_app_two_tbl should NOT have been created when running Migrate(\"bugapp_one\")")
	}
}

// 10. TestBug_ColumnChangeList_PrecisionScaleOnDeleteOnUpdate

var bugDecPrec = 10
var bugDecScale = 2

type bugDecModel struct {
	ID     int64 `attrs:"primary"`
	Amount decimal.Decimal
}

func (m *bugDecModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(m, "Amount", &attrs.Config{Precision: bugDecPrec, Scale: bugDecScale}),
	).WithTableName("bug_dec_model")
}

type bugFKActionParent struct {
	ID int64 `attrs:"primary"`
}

func (m *bugFKActionParent) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m, attrs.NewField(m, "ID", &attrs.Config{Primary: true})).WithTableName("bug_fk_action_parent")
}

var bugFKOnDelete = constraints.A_RESTRICT
var bugFKOnUpdate = constraints.A_RESTRICT

type bugFKActionChild struct {
	ID     int64 `attrs:"primary"`
	Parent *bugFKActionParent
}

func (m *bugFKActionChild) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(m, "Parent", &attrs.Config{
			Column:        "parent_id",
			Null:          true,
			RelForeignKey: attrs.Relate(&bugFKActionParent{}, "", nil),
			Attributes: map[string]any{
				migrator.AttrOnDeleteKey: bugFKOnDelete,
				migrator.AttrOnUpdateKey: bugFKOnUpdate,
			},
		}),
	).WithTableName("bug_fk_action_child")
}

func TestBug_ColumnChangeList_PrecisionScaleOnDeleteOnUpdate(t *testing.T) {
	ctx := context.Background()
	attrs.RegisterModel(&bugDecModel{})
	attrs.RegisterModel(&bugFKActionParent{})
	attrs.RegisterModel(&bugFKActionChild{})

	// Decimal Precision/Scale
	bugDecPrec, bugDecScale = 10, 2
	t.Cleanup(func() { bugDecPrec, bugDecScale = 10, 2 })
	_ = attrs.ResetDefinitions.Send(ctx, nil)
	oldDecTbl := migrator.NewModelTable(&bugDecModel{})
	oldDecCol, _ := oldDecTbl.Fields.Get("Amount")

	bugDecPrec, bugDecScale = 18, 4
	_ = attrs.ResetDefinitions.Send(ctx, nil)
	newDecTbl := migrator.NewModelTable(&bugDecModel{})
	newDecCol, _ := newDecTbl.Fields.Get("Amount")

	if cl := newDecCol.ChangeList(&oldDecCol); len(cl) == 0 {
		t.Errorf("expected ChangeList to detect Precision/Scale change, got empty")
	}
	if _, _, diffs := newDecTbl.Diff(oldDecTbl); len(diffs) != 1 {
		t.Errorf("expected Diff to report 1 changed Decimal column, got %d", len(diffs))
	}

	// FK OnDelete/OnUpdate
	bugFKOnDelete, bugFKOnUpdate = constraints.A_RESTRICT, constraints.A_RESTRICT
	t.Cleanup(func() { bugFKOnDelete, bugFKOnUpdate = constraints.A_RESTRICT, constraints.A_RESTRICT })
	_ = attrs.ResetDefinitions.Send(ctx, nil)
	oldFKTbl := migrator.NewModelTable(&bugFKActionChild{})
	oldFKCol, _ := oldFKTbl.Fields.Get("Parent")

	bugFKOnDelete, bugFKOnUpdate = constraints.A_CASCADE, constraints.A_SET_NULL
	_ = attrs.ResetDefinitions.Send(ctx, nil)
	newFKTbl := migrator.NewModelTable(&bugFKActionChild{})
	newFKCol, _ := newFKTbl.Fields.Get("Parent")

	if cl := newFKCol.ChangeList(&oldFKCol); len(cl) == 0 {
		t.Errorf("expected ChangeList to detect OnDelete/OnUpdate change, got empty")
	}
	if _, _, diffs := newFKTbl.Diff(oldFKTbl); len(diffs) != 1 {
		t.Errorf("expected Diff to report 1 changed FK column, got %d", len(diffs))
	}
}

// 11. TestBug_ModelTableDiff_ToggleUseInDB

var bugUseInDBFlag = true

type bugUseInDBToggleModel struct {
	ID   int64 `attrs:"primary"`
	Flag string
}

func (m *bugUseInDBToggleModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(m, "Flag", &attrs.Config{
			Column:     "flag",
			Attributes: map[string]any{migrator.AttrUseInDBKey: bugUseInDBFlag},
		}),
	).WithTableName("bug_use_in_db_toggle")
}

func TestBug_ModelTableDiff_ToggleUseInDB(t *testing.T) {
	ctx := context.Background()
	bugUseInDBFlag = true
	t.Cleanup(func() { bugUseInDBFlag = true })

	db, editor := openTestSQLite(t)
	initMigTestApp(t, db, func() django.AppConfig {
		cfg := apps.NewAppConfig("buguseindb")
		cfg.ModelObjects = []attrs.Definer{&bugUseInDBToggleModel{}}
		return cfg
	})

	tmpDir := t.TempDir()
	engine := migrator.NewMigrationEngine(tmpDir, editor)
	engine.MigrationLog = noopMigLog{}

	oldTbl := migrator.NewModelTable(&bugUseInDBToggleModel{})
	if err := engine.MakeMigrations(ctx); err != nil {
		t.Fatalf("Step 1 MakeMigrations: %v", err)
	}
	if err := engine.Migrate(ctx); err != nil {
		t.Fatalf("Step 1 Migrate: %v", err)
	}

	// Toggle true -> false: should report removed in Diff and drop column in DB.
	bugUseInDBFlag = false
	_ = attrs.ResetDefinitions.Send(ctx, nil)
	disabledTbl := migrator.NewModelTable(&bugUseInDBToggleModel{})
	_, removed, _ := disabledTbl.Diff(oldTbl)
	if len(removed) != 1 || removed[0].Name != "Flag" {
		t.Errorf("expected Flag in Diff removed slice when UseInDB=false, got %+v", removed)
	}
	if err := engine.MakeMigrations(ctx); err != nil {
		t.Errorf("Step 2 MakeMigrations (UseInDB true->false): %v", err)
	} else if err := engine.Migrate(ctx); err != nil {
		t.Errorf("Step 2 Migrate (UseInDB true->false): %v", err)
	} else if _, ok := sqliteColumns(t, db, "bug_use_in_db_toggle")["flag"]; ok {
		t.Errorf("expected flag column dropped after UseInDB=false migration")
	}

	// Toggle false -> true: should report added in Diff and add column in DB.
	bugUseInDBFlag = true
	_ = attrs.ResetDefinitions.Send(ctx, nil)
	reenabledTbl := migrator.NewModelTable(&bugUseInDBToggleModel{})
	added, _, _ := reenabledTbl.Diff(disabledTbl)
	if len(added) != 1 || added[0].Name != "Flag" {
		t.Errorf("expected Flag in Diff added slice when UseInDB=true, got %+v", added)
	}
}

// 12. TestBug_BuildDependencyGraph_PrevOrderingAndMissingLazyTarget

type bugPrevOrderModel struct {
	ID    int64 `attrs:"primary"`
	Name  string
	Extra string
}

func (m *bugPrevOrderModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(m, "Name"),
		attrs.NewField(m, "Extra"),
	).WithTableName("bug_prev_order")
}

type bugLazyCantMigTarget struct {
	migrator.CantMigrate
	ID int64 `attrs:"primary"`
}

func (m *bugLazyCantMigTarget) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions(ctx, m)
}

type bugLazyMissingSrcModel struct {
	ID     int64 `attrs:"primary"`
	Target attrs.Definer
}

func (m *bugLazyMissingSrcModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(m, "Target", &attrs.Config{
			Column:        "target_id",
			Null:          true,
			RelForeignKey: attrs.RelatedDeferred(attrs.RelManyToOne, "buglazymiss.Target", "", nil),
		}),
	).WithTableName("bug_lazy_missing_src")
}

func TestBug_BuildDependencyGraph_PrevOrderingAndMissingLazyTarget(t *testing.T) {
	ctx := context.Background()

	t.Run("SameModelPrevOrdering", func(t *testing.T) {
		migDir := t.TempDir()
		modelDir := filepath.Join(migDir, "bugprev", "bugPrevOrderModel")
		if err := os.MkdirAll(modelDir, 0755); err != nil {
			t.Fatal(err)
		}

		mig0001 := `{
  "table": {
    "table": "bug_prev_order",
    "model": "github.com/Nigel2392/go-django/queries/src/migrator_test.bugPrevOrderModel",
    "fields": [
      {"name":"ID","column":"id","use_in_db":true,"primary":true,"auto":true,"db_type":"INT"},
      {"name":"Name","column":"name","use_in_db":true,"nullable":true,"default":{"go_type":"string","value":""},"db_type":"STRING"}
    ],
    "indexes": null
  },
  "actions": [{"action":"create_table"}]
}`
		mig0002 := `{
  "table": {
    "table": "bug_prev_order",
    "model": "github.com/Nigel2392/go-django/queries/src/migrator_test.bugPrevOrderModel",
    "fields": [
      {"name":"ID","column":"id","use_in_db":true,"primary":true,"auto":true,"db_type":"INT"},
      {"name":"Name","column":"name","use_in_db":true,"nullable":true,"default":{"go_type":"string","value":""},"db_type":"STRING"},
      {"name":"Extra","column":"extra","use_in_db":true,"nullable":true,"default":{"go_type":"string","value":""},"db_type":"STRING"}
    ],
    "indexes": null
  },
  "actions": [{
    "action":"add_field",
    "field":{"new":{"name":"Extra","column":"extra","use_in_db":true,"nullable":true,"default":{"go_type":"string","value":""}}}
  }]
}`
		if err := os.WriteFile(filepath.Join(modelDir, "0001_create_table.mig"), []byte(mig0001), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(modelDir, "0002_add_field_extra.mig"), []byte(mig0002), 0644); err != nil {
			t.Fatal(err)
		}

		// Go map iteration over nodeMap is non-deterministic; across 20 runs without Prev links some run 0002 before 0001.
		for i := 0; i < 20; i++ {
			db, editor := openTestSQLite(t)
			initMigTestApp(t, db, func() django.AppConfig {
				cfg := apps.NewAppConfig("bugprev")
				cfg.ModelObjects = []attrs.Definer{&bugPrevOrderModel{}}
				return cfg
			})
			engine := migrator.NewMigrationEngine(migDir, editor)
			engine.MigrationLog = noopMigLog{}
			if err := engine.Migrate(ctx); err != nil {
				t.Fatalf("iteration %d Migrate failed (expected 0001 before 0002 via Prev ordering): %v", i, err)
			}
		}
	})

	t.Run("LazyTargetWithoutMigrationsReturnsError", func(t *testing.T) {
		_ = contenttypes.NewLazyRegistry("buglazymiss.Target", func(def *contenttypes.ContentTypeDefinition) bool {
			_, ok := def.ContentObject.(*bugLazyCantMigTarget)
			return ok
		})
		registerModelsContentTypes(&bugLazyCantMigTarget{}, &bugLazyMissingSrcModel{})

		db, editor := openTestSQLite(t)
		initMigTestApp(t, db,
			func() django.AppConfig {
				cfg := apps.NewAppConfig("buglazymisstgt")
				cfg.ModelObjects = []attrs.Definer{&bugLazyCantMigTarget{}}
				return cfg
			},
			func() django.AppConfig {
				cfg := apps.NewAppConfig("buglazymisssrc")
				cfg.ModelObjects = []attrs.Definer{&bugLazyMissingSrcModel{}}
				return cfg
			},
		)

		tmpDir := t.TempDir()
		srcDir := filepath.Join(tmpDir, "buglazymisssrc", "bugLazyMissingSrcModel")
		if err := os.MkdirAll(srcDir, 0755); err != nil {
			t.Fatal(err)
		}
		srcMig := `{
  "lazy_dependencies": ["buglazymiss.Target"],
  "table": {
    "table": "bug_lazy_missing_src",
    "model": "github.com/Nigel2392/go-django/queries/src/migrator_test.bugLazyMissingSrcModel",
    "fields": [
      {"name":"ID","column":"id","use_in_db":true,"primary":true,"auto":true,"db_type":"INT"}
    ],
    "indexes": null
  },
  "actions": [{"action":"create_table"}]
}`
		if err := os.WriteFile(filepath.Join(srcDir, "0001_create_table.mig"), []byte(srcMig), 0644); err != nil {
			t.Fatal(err)
		}

		engine := migrator.NewMigrationEngine(tmpDir, editor)
		engine.MigrationLog = noopMigLog{}

		assertNoPanic(t, func() {
			err := engine.Migrate(ctx)
			if err == nil {
				t.Errorf("expected error when lazy dependency target has no migrations, got nil")
			}
		})
	})
}

// 13. TestBug_ActionType_UnmarshalUnknownReturnsError

func TestBug_ActionType_UnmarshalUnknownReturnsError(t *testing.T) {
	var act migrator.ActionType
	if err := json.Unmarshal([]byte(`"not_a_real_action"`), &act); err == nil {
		t.Errorf("expected ActionType.UnmarshalJSON to return error for unknown action string")
	}
}

// 14. TestBug_MigrationRelation_UnmarshalMissingTargetFieldReturnsError

type bugRelTargetModel struct {
	ID   int64 `attrs:"primary"`
	Name string
}

func (m *bugRelTargetModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions(ctx, m)
}

func TestBug_MigrationRelation_UnmarshalMissingTargetFieldReturnsError(t *testing.T) {
	registerModelsContentTypes(&bugRelTargetModel{})
	cType := contenttypes.NewContentType(&bugRelTargetModel{})

	raw := fmt.Sprintf(`{"type":1,"model":{"content_type":%q},"field":"NonExistentField"}`, cType.TypeName())
	var rel migrator.MigrationRelation
	if err := json.Unmarshal([]byte(raw), &rel); err == nil {
		t.Errorf("expected MigrationRelation.UnmarshalJSON to return error when TargetField is missing")
	}
}

// 15. TestBug_SQLite_AlterField_ForeignKeysOnAndRenamedIndexedColumn

var bugFKParentLabelLen int64 = 50

type bugParentRefModel struct {
	ID    int64 `attrs:"primary"`
	Label string
}

func (m *bugParentRefModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(m, "Label", &attrs.Config{MaxLength: bugFKParentLabelLen}),
	).WithTableName("bug_parent_ref")
}

type bugChildRefModel struct {
	ID     int64 `attrs:"primary"`
	Parent *bugParentRefModel
}

func (m *bugChildRefModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(m, "Parent", &attrs.Config{
			Column:        "parent_id",
			RelForeignKey: attrs.Relate(&bugParentRefModel{}, "", nil),
		}),
	).WithTableName("bug_child_ref")
}

var bugIndexedColName = "code"

type bugIndexedRenameModel struct {
	ID   int64 `attrs:"primary"`
	Code string
}

func (m *bugIndexedRenameModel) DatabaseIndexes(attrs.Definer) []migrator.Index {
	return []migrator.Index{
		{Identifier: "idx_bug_code", Fields: []string{"Code"}, Unique: false},
	}
}

func (m *bugIndexedRenameModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(m, "Code", &attrs.Config{Column: bugIndexedColName}),
	).WithTableName("bug_indexed_rename")
}

func TestBug_SQLite_AlterField_ForeignKeysOnAndRenamedIndexedColumn(t *testing.T) {
	ctx := context.Background()

	t.Run("AlterReferencedTableWithForeignKeysOn", func(t *testing.T) {
		bugFKParentLabelLen = 50
		t.Cleanup(func() { bugFKParentLabelLen = 50 })

		db, editor := openTestSQLite(t)
		if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON;"); err != nil {
			t.Fatalf("PRAGMA foreign_keys = ON: %v", err)
		}

		initMigTestApp(t, db, func() django.AppConfig {
			cfg := apps.NewAppConfig("bugfkref")
			cfg.ModelObjects = []attrs.Definer{&bugParentRefModel{}, &bugChildRefModel{}}
			return cfg
		})

		tmpDir := t.TempDir()
		engine := migrator.NewMigrationEngine(tmpDir, editor)
		engine.MigrationLog = noopMigLog{}

		if err := engine.MakeMigrations(ctx); err != nil {
			t.Fatalf("Step 1 MakeMigrations: %v", err)
		}
		if err := engine.Migrate(ctx); err != nil {
			t.Fatalf("Step 1 Migrate: %v", err)
		}

		if _, err := db.ExecContext(ctx, "INSERT INTO bug_parent_ref (id, label) VALUES (1, 'p1')"); err != nil {
			t.Fatalf("seed parent: %v", err)
		}
		if _, err := db.ExecContext(ctx, "INSERT INTO bug_child_ref (id, parent_id) VALUES (1, 1)"); err != nil {
			t.Fatalf("seed child: %v", err)
		}

		bugFKParentLabelLen = 100
		_ = attrs.ResetDefinitions.Send(ctx, nil)
		if err := engine.MakeMigrations(ctx); err != nil {
			t.Fatalf("Step 2 MakeMigrations: %v", err)
		}
		if err := engine.Migrate(ctx); err != nil {
			t.Fatalf("Step 2 Migrate (AlterField on FK-referenced table): %v", err)
		}

		var parentID int64
		if err := db.QueryRowContext(ctx, "SELECT parent_id FROM bug_child_ref WHERE id = 1").Scan(&parentID); err != nil || parentID != 1 {
			t.Errorf("expected child row parent_id=1 preserved, got parentID=%d err=%v", parentID, err)
		}
	})

	t.Run("AlterIndexedColumnRename", func(t *testing.T) {
		bugIndexedColName = "code"
		t.Cleanup(func() { bugIndexedColName = "code" })

		db, editor := openTestSQLite(t)
		initMigTestApp(t, db, func() django.AppConfig {
			cfg := apps.NewAppConfig("bugidxrencol")
			cfg.ModelObjects = []attrs.Definer{&bugIndexedRenameModel{}}
			return cfg
		})

		tmpDir := t.TempDir()
		engine := migrator.NewMigrationEngine(tmpDir, editor)
		engine.MigrationLog = noopMigLog{}

		if err := engine.MakeMigrations(ctx); err != nil {
			t.Fatalf("Step 1 MakeMigrations: %v", err)
		}
		if err := engine.Migrate(ctx); err != nil {
			t.Fatalf("Step 1 Migrate: %v", err)
		}
		if _, err := db.ExecContext(ctx, "INSERT INTO bug_indexed_rename (code) VALUES ('abc')"); err != nil {
			t.Fatalf("seed row: %v", err)
		}

		bugIndexedColName = "sku"
		_ = attrs.ResetDefinitions.Send(ctx, nil)
		if err := engine.MakeMigrations(ctx); err != nil {
			t.Fatalf("Step 2 MakeMigrations (rename indexed col): %v", err)
		}
		if err := engine.Migrate(ctx); err != nil {
			t.Fatalf("Step 2 Migrate (rename indexed col): %v", err)
		}

		cols := sqliteColumns(t, db, "bug_indexed_rename")
		if _, ok := cols["sku"]; !ok {
			t.Errorf("expected column sku after rename")
		}
		var skuVal string
		if err := db.QueryRowContext(ctx, "SELECT sku FROM bug_indexed_rename WHERE id = 1").Scan(&skuVal); err != nil || skuVal != "abc" {
			t.Errorf("expected sku='abc', got %q err=%v", skuVal, err)
		}
		if _, ok := sqliteIndexes(t, db, "bug_indexed_rename")["idx_bug_code"]; !ok {
			t.Errorf("expected index idx_bug_code preserved on renamed column sku")
		}
	})
}
