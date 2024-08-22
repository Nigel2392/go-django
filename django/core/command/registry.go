package command

import (
	"flag"
	"os"

	"github.com/elliotchance/orderedmap/v2"
	"github.com/pkg/errors"
)

type commandRegistry struct {
	commands *orderedmap.OrderedMap[string, Command]
}

func NewRegistry(flagsetName string, errorHandling flag.ErrorHandling) *commandRegistry {
	return &commandRegistry{
		commands: orderedmap.NewOrderedMap[string, Command](),
	}
}

func (r *commandRegistry) Register(cmd Command) {
	r.commands.Set(cmd.Name(), cmd)
}

func (r *commandRegistry) Commands() []Command {
	var cmds = make([]Command, r.commands.Len())
	var i = 0
	for front := r.commands.Front(); front != nil; front = front.Next() {
		cmds[i] = front.Value
		i++
	}
	return cmds
}

func (r *commandRegistry) ExecCommand(args []string) error {
	var cmdName, arguments = parseCommand(args)
	if cmdName == "" {
		return errors.Wrap(ErrNoCommand, "no command provided")
	}

	var cmd, ok = r.commands.Get(cmdName)
	if !ok {
		return errors.Wrapf(
			ErrUnknownCommand,
			"could not run command %q",
			cmdName,
		)
	}

	var m Manager = &manager{
		stdout: os.Stdout,
		stderr: os.Stderr,
		stdin:  os.Stdin,
		cmd:    cmd,
	}

	var flagSet = flag.NewFlagSet(cmdName, flag.ContinueOnError)
	var err = cmd.AddFlags(m, flagSet)
	if err != nil {
		return errors.Wrapf(
			err,
			"could not add flags for command %q",
			cmdName,
		)
	}

	err = flagSet.Parse(arguments)
	if err != nil {
		return errors.Wrapf(
			err,
			"could not parse flags for command %q",
			cmdName,
		)
	}

	var remaining = flagSet.Args()

	//  if len(remaining) > 0 && !cmd.AcceptsRemaining() {
	//  	return fmt.Errorf(
	//  		"too many arguments for command %q: %v",
	//  		cmdName, remaining,
	//  	)
	//  }

	return cmd.Exec(m, remaining)
}

func parseCommand(args []string) (cmdName string, arguments []string) {
	if len(args) == 0 {
		return
	}

	cmdName = args[0]
	arguments = args[1:]

	return
}
