package command

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

var _ Manager = (*manager)(nil)

type manager struct {
	stdout *os.File
	stderr *os.File
	stdin  *os.File
	cmd    Command
}

func (m *manager) Log(message string) {
	fmt.Fprintln(m.stdout, message)
}

func (m *manager) Stdout() io.Writer {
	return m.stdout
}

func (m *manager) Stderr() io.Writer {
	return m.stderr
}

func (m *manager) Stdin() io.Reader {
	return m.stdin
}

func (m *manager) Input(question string) (string, error) {
	fmt.Fprint(m.stdout, question)
	var reader = bufio.NewReader(m.stdin)
	var input, _, err = reader.ReadLine()
	return string(input), err
}

func (m *manager) ProtectedInput(question string) (string, error) {
	fmt.Fprint(m.stdout, question)
	var bytesPass, err = term.ReadPassword(
		int(m.stdin.Fd()),
	)
	if err != nil {
		return "", err
	}
	return string(bytesPass), nil
}

func (m *manager) Command() Command {
	return m.cmd
}
