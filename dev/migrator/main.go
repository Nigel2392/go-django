package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"

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

	_ "embed"
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

//go:embed setupfile.go.txt
var setupFile string

func main() {

	// Setup global tool arguments
	// This allows us to use the same arguments for all commands
	var (
		apps    Apps
		confDir string
	)

	var fSet = flag.NewFlagSet("migrator", flag.ContinueOnError)
	fSet.Var(&apps, "app", "App to include in the migration engine (can be specified multiple times)")
	fSet.StringVar(&confDir, "conf", MIGRATION_MAP_FILE, "Path to the migration configuration file")
	if err := fSet.Parse(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		panic(err)
	}

	// Read the YAML configuration file
	var __cnf = new(struct {
		Migrate MigrationConfig `yaml:"migrate"`
	})

	var file, err = os.Open(confDir)
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

	// Setup the database connection
	// Use the configuration from the YAML file or default to SQLite
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

	// Initialize the Go-Django application with the database and other configurations
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

	// Setup the migration engine's schema editor
	schemaEditor, err := migrator.GetSchemaEditor(db.Driver())
	if err != nil {
		panic(err)
	}

	// If no apps were provided, we use all apps in the above
	// django.Apps() call
	if apps.Len() == 0 {
		apps = AppList(
			django.Global.Apps.Keys()...,
		)
	}

	var opts = make([]migrator.EngineOption, 0, 2)
	opts = append(opts,
		migrator.EngineOptionApps(apps.List()...),
	)

	if len(config.SourceDirs) == 0 {
		config.SourceDirs = []string{ROOT_MIGRATION_DIR}
	}

	var rootDir string
	if len(config.SourceDirs) > 0 {
		rootDir = config.SourceDirs[0]

		if len(config.SourceDirs) > 1 {
			opts = append(opts,
				migrator.EngineOptionDirs(config.SourceDirs[1:]...),
			)
		}
	}

	var engine = migrator.NewMigrationEngine(
		rootDir, schemaEditor, opts...,
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

				if _, ok := apps.Lookup(app); !ok {
					logger.Debugf("Skipping app %q as it is not included")
					continue
				}

				var n, err = copyDir(
					filepath.Join(rootDir, app),
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

				if conf.Setup != "" && !wasSetup(conf.Setup) {

					// the setupfile needs to know where <app_package>/<<migrations_dir(s)>>/<app_name> is located
					var diff, err = filepath.Rel(conf.Setup, conf.Destination)
					if err != nil {
						return err
					}

					var mDir = filepath.ToSlash(filepath.Join(diff, app))
					logger.Warnf("Using migration directory %q for app %q", mDir, app)

					var setupFilePath = filepath.Join(conf.Setup, "setup.migrator.go")
					var replacer = strings.NewReplacer(
						"{{APP_NAME}}", app,
						"{{MIGRATIONS_DIR}}", filepath.Dir(mDir),
					)
					if err := os.WriteFile(setupFilePath, []byte(replacer.Replace(setupFile)), 0644); err != nil {
						return err
					}
					logger.Infof("Created setup file for app %q at %q", app, setupFilePath)
				}

				// Remove the migration directory if it exists
				var migrationDir = filepath.Join(rootDir, app)
				if _, err := os.Stat(migrationDir); err == nil || !os.IsNotExist(err) {
					if err := os.RemoveAll(migrationDir); err != nil {
						return err
					}
					logger.Debugf("Removed migration directory %q for app %q", migrationDir, app)
				}
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

				if _, ok := apps.Lookup(app); !ok {
					logger.Debugf("Skipping app %q as it is not included")
					continue
				}

				var (
					migrationDir    = filepath.Join(rootDir, app)
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

				if conf.Setup != "" && wasSetup(conf.Setup) {
					var setupFilePath = filepath.Join(conf.Setup, "setup.migrator.go")
					if err := os.Remove(setupFilePath); err != nil && !os.IsNotExist(err) {
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

	if err := reg.ExecCommand(fSet.Args()); err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		panic(err)
	}
}

func wasSetup(pathToApp string) bool {
	var setupFilePath = filepath.Join(pathToApp, "setup.migrator.go")
	if _, err := os.Stat(setupFilePath); err != nil {
		if os.IsNotExist(err) {
			return false
		}
		panic(err)
	}

	var file, err = os.ReadFile(setupFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		panic(err)
	}

	return file != nil && len(file) > 0
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
