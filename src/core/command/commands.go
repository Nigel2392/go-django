package command

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/pkg/errors"
)

type Manager interface {
	Log(message string)
	Logf(format string, args ...interface{})
	Stdout() io.Writer
	Stderr() io.Writer
	Stdin() io.Reader
	Input(question string) (string, error)
	ProtectedInput(question string) (string, error)
	Registry() Registry
	Command() Command
}

type CommandDescriptor interface {
	Command
	Description() string
}

type CommandAdder interface {
	// Add optional flags to the flagset
	AddFlags(m Manager, f *flag.FlagSet) error
}

type Command interface {
	// How the command should be called from the command line
	Name() string

	// Execute the command
	// Any arguments not consumed by the flags will be passed here
	Exec(m Manager, args []string) error
}

type Cmd[T any] struct {
	ID       string
	Desc     string
	FlagFunc func(m Manager, stored *T, f *flag.FlagSet) error
	Execute  func(m Manager, stored T, args []string) error
	// Holds the state of the command
	// Helps with keeping track of the flag's values
	stored T
}

func (c *Cmd[T]) Name() string {
	return c.ID
}

func (c *Cmd[T]) Description() string {
	return c.Desc
}

func (c *Cmd[T]) AddFlags(m Manager, f *flag.FlagSet) error {
	if c.FlagFunc == nil {
		return nil
	}
	return c.FlagFunc(m, &c.stored, f)
}

func (c *Cmd[T]) Exec(m Manager, args []string) error {
	return c.Execute(m, c.stored, args)
}

type HelpCommand struct {
	ID   string
	Desc string
}

func (c *HelpCommand) Name() string {
	return c.ID
}

func (c *HelpCommand) Description() string {
	return c.Desc
}

func (c *HelpCommand) Exec(m Manager, args []string) error {
	var buf = new(bytes.Buffer)

	// print help info for a single command
	var reg = m.Registry()
	var flagsetName = reg.Name()

	fmt.Fprintf(buf, "%s%s[%s]%s",
		logger.CMD_Bold, logger.CMD_Cyan,
		strings.ToUpper(flagsetName),
		logger.CMD_Reset,
	)

	if len(args) > 0 {
		if cmd, ok := m.Registry().Which(args[0]); ok {
			var description string
			if d, ok := cmd.(CommandDescriptor); ok {
				description = d.Description()
			}

			if description != "" {
				fmt.Fprintf(buf, " %s<%s>%s %s\n", logger.CMD_Cyan, cmd.Name(), logger.CMD_Reset, description)
			} else {
				fmt.Fprintf(buf, " %s<%s>%s\n", logger.CMD_Cyan, cmd.Name(), logger.CMD_Reset)
			}

			var flagsAdded bool
			if adder, ok := cmd.(CommandAdder); ok {
				var flagger = flag.NewFlagSet(
					cmd.Name(),
					flag.ContinueOnError,
				)

				if err := adder.AddFlags(m, flagger); err != nil {
					return errors.Wrapf(
						err,
						"could not add flags for command %q",
						cmd.Name(),
					)
				}

				if flagger.NFlag() > 0 {
					flagsAdded = true
					flagger.SetOutput(buf)
					flagger.PrintDefaults()
					fmt.Fprintln(buf)
				}
			}

			if !flagsAdded {
				fmt.Fprintf(buf, "  %sThis command takes no arguments.%s\n", logger.CMD_Yellow, logger.CMD_Reset)
			}

			m.Log(buf.String())
			return flag.ErrHelp
		}

		return errors.Wrapf(
			ErrUnknownCommand,
			"could not find command %q", args[0],
		)
	}

	// inform about how to get detailed help
	buf.WriteString("\n")
	buf.WriteString("  To get help for a specific command, run:\n")
	fmt.Fprintf(buf, "    %s%shelp%s <command>\n\n",
		logger.CMD_Bold, logger.CMD_Cyan, logger.CMD_Reset,
	)

	// print non-detailed help info for all commands
	buf.WriteString("  Available commands:\n")

	for i, cmd := range reg.Commands() {

		if i > 0 {
			buf.WriteString("\n")
		}

		var description string
		var commandName = cmd.Name()
		if d, ok := cmd.(CommandDescriptor); ok {
			description = d.Description()
		}

		commandName = fmt.Sprintf("%s%s<%s>%s",
			logger.CMD_Bold, logger.CMD_Cyan,
			commandName,
			logger.CMD_Reset,
		)

		if description != "" {
			fmt.Fprintf(buf, "    %s\n      %s\n", commandName, description)
		} else {
			fmt.Fprintf(buf, "    %s\n", commandName)
		}
	}

	m.Log(buf.String())

	return flag.ErrHelp
}
