package pages

import (
	"context"
	"flag"

	"github.com/Nigel2392/go-django/src/core/command"
)

var commandFixTree command.Command = &command.Cmd[any]{
	ID:   "fix-tree",
	Desc: "Fix the tree structure of pages",
	FlagFunc: func(m command.Manager, stored *any, f *flag.FlagSet) error {
		return nil
	},
	Execute: func(m command.Manager, stored any, args []string) error {

		// Fix the tree structure of the pages
		if err := FixTree(context.Background()); err != nil {
			return err
		}

		m.Log("Tree structure fixed successfully.")

		return nil
	},
}
