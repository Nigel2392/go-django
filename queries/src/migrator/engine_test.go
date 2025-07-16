package migrator_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	_ "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	testsql "github.com/Nigel2392/go-django/queries/src/migrator/sql/test_sql"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/auth/users"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/pkg/errors"
)

func TestMigrator(t *testing.T) {

	contenttypes.Register(&contenttypes.ContentTypeDefinition{
		ContentObject: &users.Base{},
	})

	django.Global = nil
	dbtype.TYPES.Unlock()
	var app = django.App(
		django.Configure(make(map[string]interface{})),
		django.Apps(
			testsql.NewAuthAppConfig,
			testsql.NewTodoAppConfig,
			testsql.NewBlogAppConfig,
		),
		django.Flag(
			django.FlagSkipCmds,
			django.FlagSkipDepsCheck,
			django.FlagSkipChecks,
		),
	)

	if err := app.Initialize(); err != nil {
		t.Fatalf("failed to initialize app: %v", err)
	}

	// db, _ = drivers.Open(context.Background(),"sqlite3", "file:./migrator_test.db")
	// tmpDir = t.TempDir()
	var tmpDir = "./migrations"
	var editor = testsql.NewTestMigrationEngine(t)
	var engine = migrator.NewMigrationEngine(
		tmpDir,
		editor,
	)
	// editor = sqlite.NewSQLiteSchemaEditor(db)
	engine.SchemaEditor = editor
	engine.MigrationLog = &migrator.MigrationEngineConsoleLog{}

	os.RemoveAll(tmpDir)

	// MakeMigrations
	if err := engine.MakeMigrations(); err != nil {
		t.Fatalf("MakeMigrations failed: %v", err)
	}

	t.Logf("Migrations created in %q", tmpDir)

	// Ensure migration files exist
	var files = 0
	var err = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) == migrator.MIGRATION_FILE_SUFFIX {
			files++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}
	if files == 0 {
		t.Fatalf("expected migration files, got none")
	}

	// Migrate
	if err := engine.Migrate(); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	// Verify stored migrations
	if len(engine.Migrations["auth"]) == 0 {
		t.Fatalf("expected engine to track stored migrations for app 'auth' %v", engine.Migrations)
	}
	if len(engine.Migrations["todo"]) == 0 {
		t.Fatalf("expected engine to track stored migrations for app 'todo' %v", engine.Migrations)
	}
	if len(engine.Migrations["blog"]) == 0 {
		t.Fatalf("expected engine to track stored migrations for app 'blog' %v", engine.Migrations)
	}

	for model, migs := range engine.Migrations["auth"] {
		if len(migs) == 0 {
			t.Errorf("expected at least one migration stored for model %q", model)
		}
	}
	for model, migs := range engine.Migrations["todo"] {
		if len(migs) == 0 {
			t.Errorf("expected at least one migration stored for model %q", model)
		}
	}
	for model, migs := range engine.Migrations["blog"] {
		if len(migs) == 0 {
			t.Errorf("expected at least one migration stored for model %q", model)
		}
	}

	if len(editor.Actions) == 0 {
		t.Fatalf("expected actions, got none")
	}

	// Verify actions were logged (at least CreateTable)
	found := false
	for _, a := range editor.Actions {
		if a.Type == migrator.ActionCreateTable {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected CreateTable action")
	}

	t.Run("TestMigrationAddField", func(t *testing.T) {
		testsql.ExtendedDefinitions = true

		needsToMigrate, err := engine.NeedsToMakeMigrations()
		if err != nil {
			t.Fatalf("NeedsToMakeMigrations failed: %v", err)
		}

		if len(needsToMigrate) != 5 {
			t.Fatalf("expected 5 migrations, got %d", len(needsToMigrate))
		}

		if err := engine.MakeMigrations(); err != nil {
			t.Fatalf("MakeMigrations failed: %v", err)
		}

		t.Logf("Migrations created in %q", tmpDir)

		// Ensure migration files exist
		files = 0
		err = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
			if filepath.Ext(path) == migrator.MIGRATION_FILE_SUFFIX {
				files++
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Walk failed: %v", err)
		}

		if files == 0 {
			t.Fatalf("expected migration files, got none")
		}

		// Migrate
		if err := engine.Migrate(); err != nil {
			t.Fatalf("Migrate failed: %v", err)
		}

		// Verify stored migrations
		var (
			latestMigrationProfile = engine.Migrations["auth"]["Profile"][len(engine.Migrations["auth"]["Profile"])-1]
			latestMigrationTodo    = engine.Migrations["todo"]["Todo"][len(engine.Migrations["todo"]["Todo"])-1]
			latestMigrationUser    = engine.Migrations["auth"]["User"][len(engine.Migrations["auth"]["User"])-1]
		)

		// Verify stored migrations
		if len(engine.Migrations["auth"]) != 2 {
			t.Fatalf("expected 2 migrations, got %d", len(engine.Migrations["auth"]))
		}
		if len(engine.Migrations["todo"]) != 1 {
			t.Fatalf("expected 1 migrations, got %d", len(engine.Migrations["todo"]))
		}
		if len(engine.Migrations["blog"]) != 2 {
			t.Fatalf("expected 2 migrations, got %d", len(engine.Migrations["blog"]))
		}

		if len(engine.Migrations["auth"]["Profile"]) != 4 {
			t.Fatalf("expected 6 migrations for Profile, got %d", len(engine.Migrations["auth"]["Profile"]))
		}

		if len(engine.Migrations["todo"]["Todo"]) != 2 {
			t.Fatalf("expected 2 migration for Todo, got %d", len(engine.Migrations["todo"]["Todo"]))
		}

		if len(engine.Migrations["auth"]["User"]) != 4 {
			t.Fatalf("expected 4 migration for User, got %d", len(engine.Migrations["auth"]["User"]))
		}

		if len(latestMigrationProfile.LazyDependencies) != 1 {
			t.Fatalf("expected 1 dependency for Profile, got %d", len(latestMigrationProfile.LazyDependencies))
		}

		if latestMigrationProfile.Actions[len(latestMigrationProfile.Actions)-1].ActionType != migrator.ActionAddField {
			t.Fatalf("expected last action to be AddField, got %s", latestMigrationProfile.Actions[len(latestMigrationProfile.Actions)-1].ActionType)
		}

		if len(latestMigrationTodo.LazyDependencies) != 1 {
			t.Fatalf("expected 1 dependency for Todo, got %d", len(latestMigrationTodo.LazyDependencies))
		}

		if latestMigrationTodo.Actions[len(latestMigrationTodo.Actions)-1].ActionType != migrator.ActionAddField {
			t.Fatalf("expected last action to be AddField, got %s", latestMigrationTodo.Actions[len(latestMigrationTodo.Actions)-1].ActionType)
		}

		if len(latestMigrationUser.Dependencies) != 0 {
			t.Fatalf("expected 0 dependencies for User, got %d", len(latestMigrationUser.Dependencies))
		}

		if latestMigrationUser.Actions[len(latestMigrationUser.Actions)-1].ActionType != migrator.ActionAddField {
			t.Fatalf("expected last action to be AddField, got %s", latestMigrationUser.Actions[len(latestMigrationUser.Actions)-1].ActionType)
		}
	})

	t.Run("TestMigrationRemoveField", func(t *testing.T) {
		testsql.ExtendedDefinitions = false

		var needsToMigrate, err = engine.NeedsToMakeMigrations()
		if err != nil {
			t.Fatalf("NeedsToMakeMigrations failed: %v", err)
		}

		if len(needsToMigrate) != 5 {
			t.Fatalf("expected 5 migrations, got %d", len(needsToMigrate))
		}

		if err := engine.MakeMigrations(); err != nil {
			t.Fatalf("MakeMigrations failed: %v", err)
		}

		t.Logf("Migrations created in %q", tmpDir)

		// Ensure migration files exist
		files = 0
		err = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
			if filepath.Ext(path) == migrator.MIGRATION_FILE_SUFFIX {
				files++
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Walk failed: %v", err)
		}

		if files == 0 {
			t.Fatalf("expected migration files, got none")
		}

		// Migrate
		if err := engine.Migrate(); err != nil {
			t.Fatalf("Migrate failed: %v", err)
		}

		var (
			latestMigrationProfile = engine.Migrations["auth"]["Profile"][len(engine.Migrations["auth"]["Profile"])-1]
			latestMigrationTodo    = engine.Migrations["todo"]["Todo"][len(engine.Migrations["todo"]["Todo"])-1]
			latestMigrationUser    = engine.Migrations["auth"]["User"][len(engine.Migrations["auth"]["User"])-1]
		)

		// Verify stored migrations
		if len(engine.Migrations["auth"]) != 2 {
			t.Fatalf("expected 2 migrations for 'auth', got %d", len(engine.Migrations["auth"]))
		}
		if len(engine.Migrations["todo"]) != 1 {
			t.Fatalf("expected 1 migrations for 'todo', got %d", len(engine.Migrations["todo"]))
		}
		if len(engine.Migrations["blog"]) != 2 {
			t.Fatalf("expected 2 migrations for 'blog', got %d", len(engine.Migrations["blog"]))
		}

		if len(engine.Migrations["auth"]["Profile"]) != 5 {
			t.Fatalf("expected 5 migrations for Profile, got %d", len(engine.Migrations["auth"]["Profile"]))
		}

		if len(engine.Migrations["todo"]["Todo"]) != 3 {
			t.Fatalf("expected 3 migration for Todo, got %d", len(engine.Migrations["todo"]["Todo"]))
		}

		if len(engine.Migrations["auth"]["User"]) != 5 {
			t.Fatalf("expected 5 migration for User, got %d", len(engine.Migrations["auth"]["User"]))
		}

		if len(latestMigrationProfile.LazyDependencies) != 1 {
			t.Fatalf("expected 1 dependency for Profile, got %d", len(latestMigrationProfile.LazyDependencies))
		}

		if latestMigrationProfile.Actions[len(latestMigrationProfile.Actions)-1].ActionType != migrator.ActionRemoveField {
			t.Fatalf("expected last action to be RemoveField, got %s", latestMigrationProfile.Actions[len(latestMigrationProfile.Actions)-1].ActionType)
		}

		if len(latestMigrationTodo.LazyDependencies) != 1 {
			t.Fatalf("expected 1 dependency for Todo, got %d", len(latestMigrationTodo.LazyDependencies))
		}

		if latestMigrationTodo.Actions[len(latestMigrationTodo.Actions)-1].ActionType != migrator.ActionRemoveField {
			t.Fatalf("expected last action to be RemoveField, got %s", latestMigrationTodo.Actions[len(latestMigrationTodo.Actions)-1].ActionType)
		}

		if len(latestMigrationUser.Dependencies) != 0 {
			t.Fatalf("expected 0 dependencies for User, got %d", len(latestMigrationUser.Dependencies))
		}

		if latestMigrationUser.Actions[len(latestMigrationUser.Actions)-1].ActionType != migrator.ActionRemoveField {
			t.Fatalf("expected last action to be RemoveField, got %s", latestMigrationUser.Actions[len(latestMigrationUser.Actions)-1].ActionType)
		}
	})

	t.Run("TestMigrationAddFieldNoDeps", func(t *testing.T) {
		testsql.ExtendedDefinitions = false
		testsql.ExtendedDefinitionsProfile = true
		testsql.ExtendedDefinitionsTodo = true

		var needsToMigrate, err = engine.NeedsToMakeMigrations()
		if err != nil {
			t.Fatalf("NeedsToMakeMigrations failed: %v", err)
		}

		if len(needsToMigrate) != 2 {
			t.Fatalf("expected 2 migrations, got %d", len(needsToMigrate))
		}

		if err := engine.MakeMigrations(); err != nil {
			t.Fatalf("MakeMigrations failed: %v", err)
		}

		t.Logf("Migrations created in %q", tmpDir)

		// Ensure migration files exist
		files = 0
		err = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
			if filepath.Ext(path) == migrator.MIGRATION_FILE_SUFFIX {
				files++
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Walk failed: %v", err)
		}

		if files == 0 {
			t.Fatalf("expected migration files, got none")
		}

		// Migrate
		if err := engine.Migrate(); err != nil {
			t.Fatalf("Migrate failed: %v", err)
		}

		var (
			latestMigrationProfile = engine.Migrations["auth"]["Profile"][len(engine.Migrations["auth"]["Profile"])-1]
			latestMigrationTodo    = engine.Migrations["todo"]["Todo"][len(engine.Migrations["todo"]["Todo"])-1]
			latestMigrationUser    = engine.Migrations["auth"]["User"][len(engine.Migrations["auth"]["User"])-1]
		)

		// Verify stored migrations
		if len(engine.Migrations["auth"]) != 2 {
			t.Fatalf("expected 2 migrations for 'auth', got %d", len(engine.Migrations["auth"]))
		}
		if len(engine.Migrations["todo"]) != 1 {
			t.Fatalf("expected 1 migrations for 'todo', got %d", len(engine.Migrations["todo"]))
		}
		if len(engine.Migrations["blog"]) != 2 {
			t.Fatalf("expected 2 migrations for 'blog', got %d", len(engine.Migrations["blog"]))
		}

		if len(engine.Migrations["auth"]["Profile"]) != 6 {
			t.Fatalf("expected 6 migrations for Profile, got %d", len(engine.Migrations["auth"]["Profile"]))
		}

		if len(engine.Migrations["todo"]["Todo"]) != 4 {
			t.Fatalf("expected 4 migration for Todo, got %d", len(engine.Migrations["todo"]["Todo"]))
		}

		if len(engine.Migrations["auth"]["User"]) != 5 {
			t.Fatalf("expected 5 migration for User, got %d", len(engine.Migrations["auth"]["User"]))
		}

		if len(latestMigrationProfile.LazyDependencies) != 1 {
			t.Fatalf("expected 0 dependencies for Profile, got %d", len(latestMigrationProfile.LazyDependencies))
		}

		if latestMigrationProfile.Actions[len(latestMigrationProfile.Actions)-1].ActionType != migrator.ActionAddField {
			t.Fatalf("expected last action to be AddField, got %s", latestMigrationProfile.Actions[len(latestMigrationProfile.Actions)-1].ActionType)
		}

		if len(latestMigrationTodo.LazyDependencies) != 1 {
			t.Fatalf("expected 0 dependencies for Todo, got %d", len(latestMigrationTodo.LazyDependencies))
		}

		if latestMigrationTodo.Actions[len(latestMigrationTodo.Actions)-1].ActionType != migrator.ActionAddField {
			t.Fatalf("expected last action to be AddField, got %s", latestMigrationTodo.Actions[len(latestMigrationTodo.Actions)-1].ActionType)
		}

		if len(latestMigrationUser.Dependencies) != 0 {
			t.Fatalf("expected 0 dependencies for User, got %d", len(latestMigrationUser.Dependencies))
		}

		if latestMigrationUser.Actions[len(latestMigrationUser.Actions)-1].ActionType != migrator.ActionRemoveField {
			t.Fatalf("expected last action to be RemoveField, got %s", latestMigrationUser.Actions[len(latestMigrationUser.Actions)-1].ActionType)
		}
	})
}

func TestMigratorBroad(t *testing.T) {
	django.Global = nil
	dbtype.TYPES.Unlock()
	var app = django.App(
		django.Configure(make(map[string]interface{})),
		django.Apps(
			testsql.NewBroadAppConfig,
		),
		django.Flag(
			django.FlagSkipCmds,
			django.FlagSkipDepsCheck,
			django.FlagSkipChecks,
		),
	)

	if err := app.Initialize(); err != nil {
		t.Fatalf("App initialization failed: %v", err)
	}

	var tmpDir = "./migrations"
	var editor = testsql.NewTestMigrationEngine(t)
	var engine = migrator.NewMigrationEngine(
		tmpDir,
		editor,
	)

	engine.SchemaEditor = editor
	engine.MigrationLog = &migrator.MigrationEngineConsoleLog{
		Prefix: "TestMigratorBroad",
	}

	os.RemoveAll(tmpDir)

	t.Logf("Migration directory: %s, proceeding to \"makemigrations\"", tmpDir)

	if err := engine.MakeMigrations(); err == nil || !errors.Is(err, migrator.ErrNoChanges) {
		t.Fatalf("expected MakeMigrations to return ErrNoChanges, got: %v", err)
	}

	if len(engine.Migrations["broad"]) == 0 {
		t.Fatalf("expected migrations exist, got none")
	}

	// The following tests default values and their types after
	// the migration has been JSON encoded and decoded.
	//
	// This is to ensure that the default values always
	// match the expected types, even after serialization.
	//
	// TODO:
	//   I see an issue here where when: the default values are generic types,
	//   the JSON encoding/decoding may not be able to retrieve the proper
	//   type information due to generic types not getting registered
	//   the same way as concrete types (see [drivers/dbtype/types.Add]).
	t.Run("TestBroadTableDefaultValues", func(t *testing.T) {
		var mig = engine.GetLastMigration("broad", "Broad")
		if mig == nil {
			t.Fatalf("expected a migration for 'Broad', got nil")
		}

		var table = mig.Table
		var m = testsql.BroadDefaultValues()

		for _, col := range table.Columns() {
			var m = m[col.Name]

			if !reflect.DeepEqual(col.Default, m) {
				t.Errorf("expected column %s default value to be \"%v\" (%T), got \"%v\" (%T)", col.Name, m, m, col.Default, col.Default)
			}
		}
	})
}

func TestEqualDefaultTime(t *testing.T) {
	var a, b = time.Time{}, time.Time{}
	var aPtr, bPtr = &a, &b

	if !migrator.EqualDefaultValue(a, b) {
		t.Errorf("expected %v == %v", a, b)
	}

	if !migrator.EqualDefaultValue(aPtr, bPtr) {
		t.Errorf("expected %v == %v", aPtr, bPtr)
	}
}
