package migrator

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/checks"
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

		app.engine.MigrationLog = &MigrationEngineConsoleLog{}

		return nil
	}

	app.Cmd = []command.Command{
		commandMakeMigrations,
		commandMigrate,
	}

	return app
}

func (a *migratorAppConfig) Check(ctx context.Context, settings django.Settings) []checks.Message {
	var messages = a.AppConfig.Check(ctx, settings)
	var cTypes, err = a.engine.NeedsToMakeMigrations()
	if err != nil {
		messages = append(messages, checks.Critical(
			"migrator.engine.error",
			fmt.Sprintf("Failed to check if migrations are needed: %s", err.Error()),
			nil,
		))
		return messages
	}

	for _, cType := range cTypes {
		messages = append(messages, checks.Error(
			"migrator.engine.needs_makemigrations",
			"Migrations need to be made",
			cType.New(), "create new migrations by running `<your.executable> makemigrations`",
		))
	}

	needsToMigrate, err := a.engine.NeedsToMigrate()
	if err != nil {
		messages = append(messages, checks.Critical(
			"migrator.engine.error",
			fmt.Sprintf("Failed to check if migrations are needed: %s", err.Error()),
			nil,
		))
		return messages
	}

	for _, info := range needsToMigrate {
		messages = append(messages, checks.Critical(
			"migrator.engine.needs_migrate",
			fmt.Sprintf(
				"Migration \"%s/%s/%s\" needs to be applied",
				info.mig.AppName,
				info.mig.ModelName,
				info.mig.FileName(),
			),
			info.model.New(),
			"run `<your.executable> migrate` to apply the migration's to the database",
		))
	}

	return messages
}
