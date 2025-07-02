package command

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

var _ Manager = (*CommandManager)(nil)

type CommandManager struct {
	OUT io.Writer
	ERR io.Writer
	IN  *os.File
	Cmd Command
	Reg Registry
}

func (m *CommandManager) Log(message string) {
	fmt.Fprintln(m.OUT, message)
}

func (m *CommandManager) Logf(format string, args ...interface{}) {
	fmt.Fprintf(m.OUT, format, args...)
}

func (m *CommandManager) Stdout() io.Writer {
	return m.OUT
}

func (m *CommandManager) Stderr() io.Writer {
	return m.ERR
}

func (m *CommandManager) Stdin() io.Reader {
	return m.IN
}

func (m *CommandManager) Registry() Registry {
	return m.Reg
}

func (m *CommandManager) Input(question string) (string, error) {
	fmt.Fprint(m.OUT, question)
	var reader = bufio.NewReader(m.IN)
	var input, _, err = reader.ReadLine()
	return string(input), err
}

func (m *CommandManager) ProtectedInput(question string) (string, error) {
	fmt.Fprint(m.OUT, question)
	var bytesPass, err = term.ReadPassword(
		int(m.IN.Fd()),
	)
	if err != nil {
		return "", err
	}
	fmt.Println()
	return string(bytesPass), nil
}

func (m *CommandManager) Command() Command {
	return m.Cmd
}
