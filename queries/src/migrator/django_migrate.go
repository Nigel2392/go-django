package migrator

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/Nigel2392/go-django/src/core/logger"
)

var commandMigrate = &command.Cmd[migrationFlags]{
	ID:   "migrate",
	Desc: "Apply database migrations created with `makemigrations`",
	FlagFunc: func(m command.Manager, flags *migrationFlags, f *flag.FlagSet) error {
		f.BoolVar(&flags.Fake, "fake", false, "Do not create the migration files, just print what would be done")
		f.BoolVar(&flags.Fake, "f", false, "Alias for --fake")
		f.Var(&flags.Apps, "apps", "List of apps to create migrations for (default: all apps)")
		f.Var(&flags.Apps, "a", "Alias for --apps")
		return nil
	},
	Execute: func(m command.Manager, stored migrationFlags, args []string) error {
		var engine = app.engine
		if engine == nil {
			panic("migrate: engine is nil, please call django.Initialize() first")
		}

		engine.Fake = stored.Fake

		var appsList = stored.Apps.List()
		var ctx = context.Background()

		var types, err = engine.NeedsToMakeMigrations(ctx, appsList...)
		if err != nil {
			return err
		}
		if len(types) > 0 {
			// Handle the types that need migrations
			return fmt.Errorf(
				"there are %d apps with unapplied migrations, please run `makemigrations` to create the migration files: %w",
				len(types), command.ErrShouldExit,
			)
		}

		err = engine.Migrate(ctx, appsList...)
		if errors.Is(err, ErrNoChanges) {
			logger.Info(err)
			return errors.Join(
				err, command.ErrShouldExit,
			)
		}
		if err != nil {
			return err
		}

		return command.ErrShouldExit
	},
}
