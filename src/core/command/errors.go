package command

import "errors"

var (
	ErrUnknownCommand   = errors.New("unknown command")
	ErrTooManyArguments = errors.New("too many arguments")
	ErrShouldExit       = errors.New("should exit")
	ErrNoCommand        = errors.New("no command provided")
)
