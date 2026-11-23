# Migrations

The `github.com/Nigel2392/go-django/queries/src/migrator` package tracks schema snapshots in `.mig` files, computes diffs between model definitions and applied state, and executes DDL changes against SQLite, MySQL, MariaDB, and PostgreSQL databases.

---

## Auto-Registration and Setup

Importing `github.com/Nigel2392/go-django/queries/src` automatically registers:

- Schema editors for SQLite (`sql/sqlite`), MySQL and MariaDB (`sql/mysql`), and PostgreSQL (`sql/postgres`)
- `migrator.CompilerForModel`, used to resolve model constraints during migrations

Wire `migrator.NewAppConfig` into your application alongside your database setting and apps:

```go
package main

import (
	"context"

	_ "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
)

func main() {
	db, err := drivers.Open(context.Background(), "sqlite3", "app.sqlite3")
	if err != nil {
		panic(err)
	}

	app := django.App(
		django.Configure(map[string]any{
			django.APPVAR_DATABASE:         db,
			migrator.APPVAR_MIGRATION_DIR: "./migrations", // default if omitted
		}),
		django.Apps(
			migrator.NewAppConfig,
			// your apps...
		),
	)

	if err := app.Initialize(); err != nil {
		panic(err)
	}
}
```

When `migrator.NewAppConfig` is active:

- It registers two CLI commands on your application (`makemigrations` and `migrate`), documented further in [Commands](../../docs/commands.md).
- Its `Check` hook inspects registered apps and emits `checks.Critical` messages on startup if migrations still need to be generated (`makemigrations`) or applied (`migrate`).

---

## CLI Commands

### Built-In Application Commands

When running your compiled Go-Django binary:

```bash
# Generate .mig files for detected model changes
go run . makemigrations

# Apply pending migrations to the configured database
go run . migrate
```

Both commands accept:

- `--apps` / `-a`: comma-separated or repeatable app names to target (defaults to all apps)
- `--fake` / `-f`: dry-run mode; prints or simulates without writing `.mig` files (`makemigrations`) or executing SQL (`migrate`)

Note that `migrate` refuses to run and returns an error if `NeedsToMakeMigrations` detects uncaptured model changes — run `makemigrations` first.

### Standalone `cmd/migrator` Tool

For repositories using embedded per-app migration directories, `github.com/Nigel2392/go-django/cmd/migrator` reads a `migrations.yml` file (default `-conf ./migrations.yml`, with optional repeatable `-app` flags):

```yaml
migrate:
  db:
    engine: sqlite3
    dsn: ./.private/db.sqlite3
  src:
    - ./migrations
  apps:
    myapp:
      dst: ./myapp/migrations
      setup: ./myapp
```

`cmd/migrator` exposes:

- `makemigrations`: runs `MigrationEngine.MakeMigrations`, moves generated `.mig` files into each configured `dst/<app>` directory, and writes a `setup.migrator.go` embed helper (`migrator.RegisterFileSystem`) inside `setup` if one does not already exist.
- `clearmigrations` (flags `-src` and `-dst`, both `true` by default): removes generated migration files and `setup.migrator.go`.

*(Note: `cmd/migrator` generates and cleans up `.mig` files via `makemigrations` and `clearmigrations`; applying migrations at runtime is done with your app's `migrate` command or `migrator.Migrate` / `migrator.AutoMigrate`.)*

---

## Embedding Migrations in Apps

To bundle `.mig` files inside reusable Go packages rather than reading from `./migrations` at runtime, either:

1. Implement `migrator.MigrationAppConfig` (`GetMigrationFS() fs.FS`) or wrap your app with `*migrator.MigratorAppConfig{AppConfig: ..., MigrationFS: ...}`, or
2. Register an `fs.FS` via `migrator.RegisterFileSystem(appName, fSys)`:

```go
package myapp

import (
	"embed"

	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/src/core/filesystem"
)

//go:embed migrations/*
var migrationFS embed.FS

var _ = migrator.RegisterFileSystem("myapp", filesystem.Sub(migrationFS, "migrations/myapp"))
```

When `MigrationEngine` initializes, `EngineOptionApps` checks `MigrationAppConfig` first and then falls back to filesystems registered with `RegisterFileSystem`.

---

## Programmatic API (`AutoMigrate` & `MigrationEngine`)

Package `migrator` offers top-level helpers and the lower-level `MigrationEngine`:

```go
// Top-level helpers using options
func NeedsToMakeMigrations(ctx context.Context, opts ...AutoMigrateOption) ([]*contenttypes.BaseContentType[attrs.Definer], error)
func MakeMigrations(ctx context.Context, opts ...AutoMigrateOption) error
func NeedsToMigrate(ctx context.Context, opts ...AutoMigrateOption) ([]NeedsToMigrateInfo, error)
func Migrate(ctx context.Context, opts ...AutoMigrateOption) error
func AutoMigrate(ctx context.Context, opts ...AutoMigrateOption) (madeMigrations, migrated bool, err error)

// AutoMigrate options
func AutoMigrateApps(applist ...string) AutoMigrateOption
func AutoMigrateMigrationDir(s string) AutoMigrateOption
func AutoMigrateDatabase(db drivers.Database) AutoMigrateOption
func AutoMigrateLog(log MigrationLog) AutoMigrateOption
```

For example, running automatic migrations during test setup or small single-binary deployments:

```go
made, applied, err := migrator.AutoMigrate(
	ctx,
	migrator.AutoMigrateMigrationDir("./migrations"),
	migrator.AutoMigrateApps("todos"),
)
```

For manual control, construct `migrator.NewMigrationEngine(basePath, schemaEditor, opts...)` with `EngineOptionApps(apps...)` and `EngineOptionDirs(dirs...)`.

---

## Model Customization Interfaces and Attributes

Models and fields customize migration behavior through interfaces and attribute keys (see [Models](./models/models.md) and [Constraints](./expressions/constraints.md)):

| Symbol | Purpose |
| :--- | :--- |
| `migrator.IndexDefiner` | `DatabaseIndexes(obj attrs.Definer) []Index` defines table indexes (`Identifier`, `Type`, `Fields`, `Unique`, `Comment`). |
| `migrator.UniqueTogetherDefiner` | `UniqueTogether() [][]string` creates composite unique indexes (`<cols>_unique_together_idx`). |
| `migrator.ConstraintDefiner` | `Constraints() []constraints.Constraint` attaches `CheckConstraint`, `UniqueConstraint`, and `ForeignKeyConstraint` definitions. |
| `migrator.CanMigrate` / `migrator.CantMigrate` | Embed `migrator.CantMigrate` or return `false` from `CanMigrate() bool` to skip migrating a model or field; `MetaAllowMigrateKey` (`"migrator.allow_migrate"`) in model storage is also checked. |
| `migrator.CanColumnDBType` | Custom `DBType(*Column) dbtype.Type` override on a field. |
| `migrator.RegisterColumnType` | Maps `(driver.Driver, dbtype.Type)` to a SQL column type string (`func(c *Column) string`). |
| `migrator.AttrDBTypeKey` | Explicit SQL type string override stored on `attrs.Field` attributes. |
| `migrator.AttrUseInDBKey` | Boolean controlling whether a field maps to a physical column (`false` for reverse/M2M/embedded fields). |
| `migrator.AttrOnDeleteKey`, `migrator.AttrOnUpdateKey` | `constraints.MutationAction` (`A_RESTRICT`, `A_CASCADE`, `A_SET_NULL`, `A_SET_DEFAULT`) stored on relation fields. |

---

## Migration Files, Actions, and Custom Go Hooks

Generated `.mig` files live under `<migration_dir>/<app_name>/<model_name>/<NNNN>_<summary>.mig` and store a JSON snapshot of the `ModelTable` alongside `Dependencies`, `LazyDependencies`, and `Actions`:

- `ActionCreateTable`, `ActionDropTable`, `ActionRenameTable`
- `ActionAddIndex`, `ActionDropIndex`, `ActionRenameIndex`
- `ActionAddConstraint`, `ActionDropConstraint`
- `ActionAddField`, `ActionAlterField`, `ActionRemoveField`
- `ActionExecGoCode` (`"exec"`)

To execute custom Go logic during a migration step whose `.mig` action uses `"action": "exec"`, register a callback matching the model type and migration filename (`migration.FileName()`):

```go
migrator.RegisterMigrateFunc(&User{}, "0003_backfill_slugs.mig", func(
	ctx context.Context,
	engine *migrator.MigrationEngine,
	table *migrator.ModelTable,
) error {
	// Access the migration transaction via migrator.DbFromContext(ctx, defaultDb)
	return nil
})
```

---

## SchemaEditor Summary

`migrator.SchemaEditor` abstracts DDL across drivers (`SQLiteSchemaEditor`, `MySQLSchemaEditor`, `PostgresSchemaEditor`):

- Stores applied migration records in a `migrations` table (`Setup`, `StoreMigration`, `HasMigration`, `RemoveMigration`).
- Wraps `MigrationEngine.Migrate` inside `SchemaEditor.StartTransaction(ctx)` and injects the active transaction into `ctx` via `migrator.ContextWithDb` (retrievable via `migrator.DbFromContext`).
- SQLite rebuilds tables under the hood for `AlterField`, `RemoveField`, `AddConstraint`, and `DropConstraint`.

---

## Caveats

- **`django.APPVAR_DATABASE` must be set in `django.Global.Settings`:** The registered `SchemaEditor` factories for SQLite, MySQL/MariaDB, and PostgreSQL read `django.APPVAR_DATABASE` from `django.Global.Settings` inside `migrator.GetSchemaEditor`. Even when calling `AutoMigrateDatabase(db)`, ensure `django.APPVAR_DATABASE` is configured (or construct your `SchemaEditor` explicitly via `NewSQLiteSchemaEditor` / `NewMigrationEngine`).
- **DDL parameter restriction on constraints:** `CheckConstraint` expressions used in migrations cannot produce `?` bound placeholders — use `expr.V(val, true)` (`unsafe = true`) as noted in [Constraints](./expressions/constraints.md#caveats).
- **Empty app list / initialization errors:** `AutoMigrate` requires `django.Global` to be initialized with at least one app and returns an error (`errs.ErrInvalidValue`) if not.
- **Lazy relations:** Foreign keys defined with `attrs.RelateLazy(...)` record `LazyDependencies` (`"app.Model"` or content type key) so dependency topological sorting delays applying dependent migrations until the target model's table migration runs.
