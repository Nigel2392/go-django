package migrator

import (
	"context"
	"errors"
	"flag"

	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/Nigel2392/go-django/src/core/command/flags"
	"github.com/Nigel2392/go-django/src/core/logger"
)

type migrationFlags struct {
	Fake bool
	Apps flags.List
}

var commandMakeMigrations = &command.Cmd[migrationFlags]{
	ID:   "makemigrations",
	Desc: "Create new database migrations to be applied with `migrate`",
	FlagFunc: func(m command.Manager, flags *migrationFlags, f *flag.FlagSet) error {
		f.BoolVar(&flags.Fake, "fake", false, "Do not create the migration files, just print what would be done")
		f.BoolVar(&flags.Fake, "f", false, "Alias for --fake")
		f.Var(&flags.Apps, "apps", "List of apps to create migrations for (default: all apps)")
		f.Var(&flags.Apps, "a", "Alias for --apps")
		return nil
	},
	Execute: func(m command.Manager, flags migrationFlags, args []string) error {
		var engine = app.engine
		if engine == nil {
			panic("migrate: engine is nil, please call django.Initialize() first")
		}

		engine.Fake = flags.Fake

		var ctx = context.Background()
		var err = engine.MakeMigrations(ctx, flags.Apps.List()...)
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
