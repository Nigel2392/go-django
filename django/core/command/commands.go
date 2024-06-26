package command

import (
	"flag"
	"io"
)

type Manager interface {
	Log(message string)
	Stdout() io.Writer
	Stderr() io.Writer
	Stdin() io.Reader
	Input(question string) (string, error)
	ProtectedInput(question string) (string, error)
	Command() Command
}

type CommandDescriptor interface {
	Command
	Description() string
}

type Command interface {
	// How the command should be called from the command line
	Name() string

	// Add optional flags to the flagset
	AddFlags(m Manager, f *flag.FlagSet) error

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
