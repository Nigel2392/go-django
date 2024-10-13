package mail

import (
	"errors"
	"io"

	"github.com/jordan-wright/email"
)

type consoleManager struct {
	o io.Writer
}

func NewConsoleBackend(f io.Writer) EmailBackend {
	return &consoleManager{
		o: f,
	}
}

func (m *consoleManager) Send(e *email.Email) error {
	var b, err = e.Bytes()
	if err != nil {
		return err
	}

	_, err = m.o.Write(b)
	_, err2 := m.o.Write([]byte("\n"))
	return errors.Join(
		err, err2,
	)
}
