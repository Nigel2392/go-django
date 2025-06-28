package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/yaml.v3"

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

// go run ./dev/migrator makemigrations
// go run ./dev/migrator clearmigrations
// go run ./dev/migrator migrate

const ROOT_MIGRATION_DIR = "./migrations"
const MIGRATION_MAP_FILE = "./migrations.yml"

func main() {
	var __cnf = new(struct {
		Migrate MigrationConfig `yaml:"migrate"`
	})

	var file, err = os.Open(MIGRATION_MAP_FILE)
	if err != nil {
		logger.Errorf("Failed to open migration map file: %v", err)
		return
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)
	if err = decoder.Decode(__cnf); err != nil {
		logger.Errorf("Failed to decode migration map file: %v", err)
		return
	}

	var config = __cnf.Migrate
	var dbConf = &DatabaseConfig{
		Engine: "sqlite3",
		DSN:    "./.private/db.sqlite3",
	}
	if config.Database != nil {
		dbConf = config.Database
	}

	db, err := drivers.Open(
		context.Background(),
		dbConf.Engine,
		dbConf.DSN,
	)
	if err != nil {
		panic(err)
	}

	app := django.App(
		django.Configure(map[string]interface{}{
			django.APPVAR_DATABASE: db,
		}),
		django.AppLogger(&logger.Logger{
			Level:       logger.INF,
			OutputDebug: os.Stdout,
			Prefix:      "migrator",
			WrapPrefix:  logger.ColoredLogWrapper,
		}),
		django.Flag(
			django.FlagSkipCmds,
		),
		django.Apps(
			session.NewAppConfig,
			pages.NewAppConfig,
			revisions.NewAppConfig,
			migrator.NewAppConfig,
			openauth2.NewAppConfig(openauth2.Config{}),
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
		"migrator",
		flag.ContinueOnError,
	)

	reg.Register(&command.Cmd[any]{
		ID:   "makemigrations",
		Desc: "Create new database migrations to be applied with `migrate`",
		Execute: func(m command.Manager, stored any, args []string) error {
			var err = engine.MakeMigrations()
			if err != nil {
				return err
			}

			for app, conf := range config.Targets.Iter() {
				var n, err = copyDir(
					filepath.Join(ROOT_MIGRATION_DIR, app),
					filepath.Join(conf.Destination, app),
				)
				if err != nil {
					return err
				}

				if n == 0 {
					logger.Debugf("No migrations were copied for app %q", app)
					continue
				}

				logger.Infof("Copied migrations from %q to %q", app, conf.Destination)
			}

			return nil
		},
	})

	reg.Register(&command.Cmd[any]{
		ID:   "clearmigrations",
		Desc: "Clear all migration files for all apps",
		Execute: func(m command.Manager, stored any, args []string) error {
			logger.Warn("Clearing all migration files...")

			var anyRemoved bool
			for app, conf := range config.Targets.Iter() {
				var (
					migrationDir    = filepath.Join(ROOT_MIGRATION_DIR, app)
					dirRemoved      = false
					appMigrationDir = filepath.Join(conf.Destination, app)
					appDirRemoved   = false
				)

				if _, err := os.Stat(migrationDir); err == nil || !os.IsNotExist(err) {
					if err := os.RemoveAll(migrationDir); err != nil {
						return err
					}
					dirRemoved = true
				}

				if _, err := os.Stat(appMigrationDir); err == nil || !os.IsNotExist(err) {
					if err := os.RemoveAll(appMigrationDir); err != nil {
						return err
					}
					appDirRemoved = true
				}

				switch {
				case dirRemoved && appDirRemoved:
					logger.Infof("Removed migration directories for app %q", app)
				case dirRemoved:
					logger.Infof("Removed migration directory %q for app %q", migrationDir, app)
				case appDirRemoved:
					logger.Infof("Removed app migration directory %q for app %q", appMigrationDir, app)
				default:
					logger.Debugf("No migration directories found for app %q", app)
				}

				anyRemoved = anyRemoved || dirRemoved || appDirRemoved
			}

			if !anyRemoved {
				logger.Info("No migration directories were removed")
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

func copyDir(src, dst string) (copied int, err error) {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	if _, err = os.Stat(src); err != nil {
		if !os.IsNotExist(err) {
			return copied, err
		}
		return copied, nil
	}

	if err = os.MkdirAll(dst, 0755); err != nil {
		return copied, err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return copied, err
	}

	for _, entry := range entries {
		var (
			srcPath = filepath.Join(src, entry.Name())
			dstPath = filepath.Join(dst, entry.Name())
		)

		if entry.IsDir() {
			var n, err = copyDir(srcPath, dstPath)
			copied += n
			if err != nil {
				return copied, err
			}
			continue
		}

		if err := os.Rename(srcPath, dstPath); err != nil {
			return copied, err
		}
		copied++
	}

	return copied, nil
}
