package command

import "errors"

var (
	ErrUnknownCommand = errors.New("unknown command")
	ErrNoCommand      = errors.New("no command provided")
)
