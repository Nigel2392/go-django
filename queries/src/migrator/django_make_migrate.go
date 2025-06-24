package migrator

import (
	"flag"

	"github.com/Nigel2392/go-django/src/core/command"
)

var commandMakeMigrations = &command.Cmd[any]{
	ID:   "makemigrations",
	Desc: "Create new database migrations to be applied with `migrate`",
	FlagFunc: func(m command.Manager, stored *any, f *flag.FlagSet) error {
		return nil
	},
	Execute: func(m command.Manager, stored any, args []string) error {
		var engine = app.engine
		if engine == nil {
			panic("migrate: engine is nil, please call django.Initialize() first")
		}

		var err = engine.MakeMigrations()
		if err != nil {
			return err
		}

		return nil
	},
}
