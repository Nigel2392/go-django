package command

import (
	"flag"
	"os"

	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/elliotchance/orderedmap/v2"
	"github.com/pkg/errors"
)

var BUILTINS = make([]Command, 0)

func init() {
	BUILTINS = append(BUILTINS, &HelpCommand{
		ID:   "help",
		Desc: "List all available commands and their usage information",
	})
}

type Registry interface {
	Name() string
	Which(cmdName string) (Command, bool)
	Register(cmd Command)
	Unregister(cmdName string) error
	Commands() []Command
	ExecCommand(args []string) error
}

type commandRegistry struct {
	name          string
	errorHandling flag.ErrorHandling
	commands      *orderedmap.OrderedMap[string, Command]
}

func NewRegistry(flagsetName string, errorHandling flag.ErrorHandling) Registry {
	var reg = &commandRegistry{
		name:          flagsetName,
		errorHandling: errorHandling,
		commands:      orderedmap.NewOrderedMap[string, Command](),
	}

	// Register built-in commands
	for _, cmd := range BUILTINS {
		reg.Register(cmd)
	}

	return reg
}

func (r *commandRegistry) Name() string {
	return r.name
}

func (r *commandRegistry) Which(cmdName string) (Command, bool) {
	return r.commands.Get(cmdName)
}

func (r *commandRegistry) Register(cmd Command) {
	r.commands.Set(cmd.Name(), cmd)
}

func (r *commandRegistry) Unregister(cmdName string) error {
	if r.commands.Delete(cmdName) {
		return nil
	}
	return errors.Wrapf(
		ErrUnknownCommand,
		"could not unregister command %q", cmdName,
	)
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
		stderr: os.Stdout,
		stdin:  os.Stdin,
		cmd:    cmd,
		reg:    r,
	}

	var remaining = arguments
	if cmd, ok := cmd.(CommandAdder); ok {
		var flagSet = flag.NewFlagSet(cmdName, flag.ContinueOnError)
		var err = cmd.AddFlags(m, flagSet)
		if err != nil {
			return errors.Wrapf(
				err,
				"could not add flags for command %q",
				cmdName,
			)
		}

		if hasFlags(flagSet) {
			err = flagSet.Parse(arguments)
			if err != nil {
				return errors.Wrapf(
					err,
					"could not parse flags for command %q",
					cmdName,
				)
			}
		}
	}

	var err = cmd.Exec(m, remaining)
	if err != nil {

		if errors.Is(err, flag.ErrHelp) {
			return nil
		}

		switch r.errorHandling {
		case flag.ExitOnError:
			logger.Debugf("Error executing command %q: %s", cmdName, err.Error())
			os.Exit(1)
		case flag.ContinueOnError:
			logger.Debugf("Error executing command %q: %s", cmdName, err.Error())
		case flag.PanicOnError:
			panic(errors.Wrapf(
				err, "Error executing command %q", cmdName,
			))
		}
	}
	return nil
}

func parseCommand(args []string) (cmdName string, arguments []string) {
	if len(args) == 0 {
		return
	}

	cmdName = args[0]
	arguments = args[1:]

	return
}

func hasFlags(flagset *flag.FlagSet) bool {
	if flagset == nil {
		return false
	}

	var ct = 0
	flagset.VisitAll(func(f *flag.Flag) {
		ct++
	})
	return ct > 0
}
