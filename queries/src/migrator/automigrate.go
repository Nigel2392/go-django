package migrator

import (
	"context"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/pkg/errors"
)

/*

	AUTO MIGRATE OPTIONS

*/

type AutoMigrateOption = func(context.Context, *AutoMigrateConfig) error

func AutoMigrateMigrationDir(s string) AutoMigrateOption {
	if s == "" {
		panic("No migration directory specified for AutoMigrate()")
	}
	return func(_ context.Context, c *AutoMigrateConfig) error {
		c.MigrationDir = s
		return nil
	}
}

func AutoMigrateDatabase(db drivers.Database) AutoMigrateOption {
	if db == nil {
		panic("No database specified for AutoMigrate()")
	}
	return func(_ context.Context, c *AutoMigrateConfig) error {
		c.Database = db
		return nil
	}
}

func AutoMigrateLog(log MigrationLog) AutoMigrateOption {
	if log == nil {
		panic("No log specified for AutoMigrate()")
	}
	return func(_ context.Context, c *AutoMigrateConfig) error {
		c.Log = log
		return nil
	}
}

func AutoMigrateApps(applist ...string) AutoMigrateOption {
	return func(_ context.Context, c *AutoMigrateConfig) error {
		c.Apps = applist
		return nil
	}
}

/*

	AUTO MIGRATE CONFIG

*/
// Easily provide a custom configuration for your automigrations.
type AutoMigrateConfig struct {
	Apps         []string
	MigrationDir string
	Database     drivers.Database
	Log          MigrationLog
}

func (a *AutoMigrateConfig) changed() bool {
	return a.Database != nil || a.MigrationDir != ""
}

func (a *AutoMigrateConfig) apps() []string {
	if len(a.Apps) > 0 {
		return a.Apps
	}
	var apps = make([]string, 0, django.Global.Apps.Len())
	for head := django.Global.Apps.Front(); head != nil; head = head.Next() {
		apps = append(apps, head.Key)
	}
	a.Apps = apps
	return apps
}

func (a *AutoMigrateConfig) migrationDir() string {
	if a.MigrationDir == "" {
		a.MigrationDir = django.ConfigGet(
			django.Global.Settings,
			APPVAR_MIGRATION_DIR,
			DEFAULT_MIGRATION_DIR,
		)
	}
	return a.MigrationDir
}

func (a *AutoMigrateConfig) db() drivers.Database {
	if a.Database == nil {
		var dbInt, ok = django.Global.Settings.Get(django.APPVAR_DATABASE)
		assert.True(ok, "DATABASE setting is required for AutoMigrate()")

		db, ok := dbInt.(drivers.Database)
		assert.True(ok, "DATABASE setting must be of type drivers.Database, got %T", dbInt)
		a.Database = db
	}
	return a.Database
}

func (a *AutoMigrateConfig) log() MigrationLog {
	if a.Log == nil {
		a.Log = &MigrationEngineConsoleLog{}
	}
	return a.Log
}

func NeedsToMakeMigrations(ctx context.Context, opts ...AutoMigrateOption) ([]*contenttypes.BaseContentType[attrs.Definer], error) {
	cnf, engine, err := initAutoMigrateConfig(ctx, opts)
	if err != nil {
		return nil, err
	}
	return engine.NeedsToMakeMigrations(ctx, cnf.apps()...)
}

func MakeMigrations(ctx context.Context, opts ...AutoMigrateOption) error {
	cnf, engine, err := initAutoMigrateConfig(ctx, opts)
	if err != nil {
		return err
	}

	return engine.MakeMigrations(ctx, cnf.apps()...)
}

func NeedsToMigrate(ctx context.Context, opts ...AutoMigrateOption) ([]NeedsToMigrateInfo, error) {
	cnf, engine, err := initAutoMigrateConfig(ctx, opts)
	if err != nil {
		return nil, err
	}
	return engine.NeedsToMigrate(ctx, cnf.apps()...)
}

func Migrate(ctx context.Context, opts ...AutoMigrateOption) error {
	cnf, engine, err := initAutoMigrateConfig(ctx, opts)
	if err != nil {
		return err
	}
	return engine.Migrate(ctx, cnf.apps()...)
}

func AutoMigrate(ctx context.Context, opts ...AutoMigrateOption) (madeMigrations, migrated bool, err error) {
	cnf, engine, err := initAutoMigrateConfig(ctx, opts)
	types, err := engine.NeedsToMakeMigrations(ctx, cnf.apps()...)
	if err != nil {
		return false, false, err
	}

	if len(types) > 0 {
		err = engine.MakeMigrations(ctx, cnf.apps()...)
		if err != nil {
			return false, false, err
		}
	}

	which, err := engine.NeedsToMigrate(ctx, cnf.apps()...)
	if err != nil {
		return len(types) > 0, false, err
	}
	if len(which) > 0 {
		err = engine.Migrate(ctx, cnf.apps()...)
		if err != nil {
			return len(types) > 0, false, err
		}
	}

	return len(types) > 0, len(which) > 0, nil
}

func initAutoMigrateConfig(ctx context.Context, opts []AutoMigrateOption) (c *AutoMigrateConfig, engine *MigrationEngine, err error) {
	if django.Global == nil || django.Global.Apps == nil || django.Global.Apps.Len() == 0 {
		return nil, nil, errors.Wrap(
			errs.ErrInvalidValue,
			"Django application has not been initialized, or does not contain any apps.",
		)
	}

	var cnf = &AutoMigrateConfig{}
	for _, fn := range opts {
		if err = fn(ctx, cnf); err != nil {
			return nil, nil, err
		}
	}

	switch {
	case app == nil || app.engine == nil || cnf.changed():
		schemaEditor, err := GetSchemaEditor(cnf.db().Driver())
		if err != nil {
			return nil, nil, err
		}

		engine = NewMigrationEngine(cnf.migrationDir(), schemaEditor)
	case app != nil && app.engine != nil:
		if cnf.Log == nil {
			cnf.Log = app.engine.MigrationLog
		}

		engine = app.engine
	}

	// initialize engine or create a new one
	engine.MigrationLog = cnf.log()
	return cnf, engine, nil
}
