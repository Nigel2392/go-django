package migrator

import (
	"io/fs"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/command"
)

const (
	DEFAULT_MIGRATION_DIR = "./migrations"
)

type MigrationAppConfig interface {
	django.AppConfig
	GetMigrationFS() fs.FS
}

type MigratorAppConfig struct {
	django.AppConfig
	MigrationFS fs.FS
}

func (m *MigratorAppConfig) GetMigrationFS() fs.FS {
	return m.MigrationFS
}

type migratorAppConfig struct {
	*apps.DBRequiredAppConfig
	MigrationDir string
	engine       *MigrationEngine
}

var app = &migratorAppConfig{
	DBRequiredAppConfig: apps.NewDBAppConfig("migrator"),
}

func NewAppConfig() *migratorAppConfig {

	app.MigrationDir, _ = django.ConfigGetOK(
		django.Global.Settings,
		APPVAR_MIGRATION_DIR,
		DEFAULT_MIGRATION_DIR,
	)

	if app.MigrationDir == "" {
		app.MigrationDir = DEFAULT_MIGRATION_DIR
	}

	app.Init = func(settings django.Settings, db drivers.Database) error {

		var schemaEditor, err = GetSchemaEditor(db.Driver())
		if err != nil {
			return err
		}

		app.engine = NewMigrationEngine(
			app.MigrationDir,
			schemaEditor,
		)
		return nil
	}

	app.Cmd = []command.Command{
		commandMakeMigrations,
		commandMigrate,
	}

	return app
}
