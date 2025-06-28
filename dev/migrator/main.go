package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/openauth2"
	"github.com/Nigel2392/go-django/src/contrib/pages"
	"github.com/Nigel2392/go-django/src/contrib/revisions"
	"github.com/Nigel2392/go-django/src/contrib/session"
	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/Nigel2392/go-django/src/core/logger"
)

// utility application to easily generate migrations during development
//
// migrations should be placed in <app>/migrations/<app_name>/<model_name>/<migration_name>.mig
//
// the app where the migrations are placed should be wrapped by [migrator.MigratorAppConfig], or implement [migrator.MigrationAppConfig],
// which is used to provide the migration filesystem to the migrator engine

const ROOT_MIGRATION_DIR = "./migrations"

var appMapping = [][2]string{
	{"pages", "src/contrib/pages/migrations"},
	{"revisions", "src/contrib/revisions/migrations"},
	{"session", "src/contrib/session/migrations"},
	{"openauth2", "src/contrib/openauth2/migrations"},
}

func main() {
	var db, err = drivers.Open(context.Background(), "sqlite3", "./.private/db.sqlite3")
	if err != nil {
		panic(err)
	}

	var app = django.App(
		django.Configure(map[string]interface{}{
			django.APPVAR_ALLOWED_HOSTS: []string{"*"},
			django.APPVAR_DEBUG:         false,
			django.APPVAR_HOST:          "127.0.0.1",
			django.APPVAR_PORT:          "8080",
			django.APPVAR_DATABASE:      db,
		}),
		django.AppLogger(&logger.Logger{
			Level:      logger.INF,
			OutputInfo: os.Stdout,
			WrapPrefix: logger.ColoredLogWrapper,
		}),
		django.Flag(
			django.FlagSkipCmds,
		),
		django.Apps(
			session.NewAppConfig,
			pages.NewAppConfig,
			revisions.NewAppConfig,
			openauth2.NewAppConfig(openauth2.Config{}),
			migrator.NewAppConfig,
		),
	)

	if err = app.Initialize(); err != nil {
		panic(err)
	}

	schemaEditor, err := migrator.GetSchemaEditor(db.Driver())
	if err != nil {
		panic(err)
	}

	var engine = migrator.NewMigrationEngine(
		ROOT_MIGRATION_DIR,
		schemaEditor,
	)

	var reg = command.NewRegistry(
		"django",
		flag.ContinueOnError,
	)

	var cmds = make(map[string]command.Command)
	for _, c := range app.Commands.Commands() {
		cmds[c.Name()] = c
	}

	reg.Register(cmds["migrate"])

	reg.Register(&command.Cmd[any]{
		ID:   "makemigrations",
		Desc: "Create new database migrations to be applied with `migrate`",
		FlagFunc: func(m command.Manager, stored *any, f *flag.FlagSet) error {
			return nil
		},
		Execute: func(m command.Manager, stored any, args []string) error {
			var err = engine.MakeMigrations()
			if err != nil {
				return err
			}

			for _, mapping := range appMapping {
				err := copyDir(
					filepath.Join(ROOT_MIGRATION_DIR, mapping[0]),
					filepath.Join(mapping[1], mapping[0]),
				)
				if err != nil {
					return err
				}

				logger.Infof("Copied migrations from %s to %s", mapping[0], mapping[1])
			}

			return nil
		},
	})

	reg.Register(&command.Cmd[any]{
		ID:   "clearmigrations",
		Desc: "Clear all migration files for all apps",
		FlagFunc: func(m command.Manager, stored *any, f *flag.FlagSet) error {
			return nil
		},
		Execute: func(m command.Manager, stored any, args []string) error {
			logger.Info("Clearing all migration files...")

			for _, mapping := range appMapping {
				migrationDir := filepath.Join(ROOT_MIGRATION_DIR, mapping[0])
				if err := os.RemoveAll(migrationDir); err != nil {
					return err
				}

				logger.Infof("Removed migration directory: %s", migrationDir)
			}

			for _, mapping := range appMapping {
				appMigrationDir := filepath.Join(mapping[1], mapping[0])
				if err := os.RemoveAll(appMigrationDir); err != nil {
					return err
				}

				logger.Infof("Removed app migration directory: %s", appMigrationDir)
			}

			return nil
		},
	})

	if err := reg.ExecCommand(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		panic(err)
	}
}

func copyDir(src, dst string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	if _, err := os.Stat(src); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return nil
	}

	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			logger.Infof("Copying directory: %s to %s", srcPath, dstPath)
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			logger.Infof("Copying file: %s to %s", srcPath, dstPath)
			if err := os.Rename(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}
