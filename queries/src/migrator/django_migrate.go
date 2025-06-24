package migrator

import (
	"flag"

	"github.com/Nigel2392/go-django/src/core/command"
)

var commandMigrate = &command.Cmd[any]{
	ID:   "migrate",
	Desc: "Apply database migrations created with `makemigrations`",
	FlagFunc: func(m command.Manager, stored *any, f *flag.FlagSet) error {
		return nil
	},
	Execute: func(m command.Manager, stored any, args []string) error {
		var engine = app.engine
		if engine == nil {
			panic("migrate: engine is nil, please call django.Initialize() first")
		}

		var err = engine.Migrate()
		if err != nil {
			return err
		}

		return nil
	},
}
