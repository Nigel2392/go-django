package email

import (
	"bytes"
	"crypto/tls"
	"errors"
	"html/template"
	"net/smtp"
	"strconv"
	"time"

	"github.com/Nigel2392/go-django/core/httputils"

	"github.com/jordan-wright/email"
)

type Manager struct {
	EMAIL_HOST     string // Mail server host.
	EMAIL_PORT     int    // Mail server port.
	EMAIL_USERNAME string // Mail server username.
	EMAIL_PASSWORD string // Mail server password.
	EMAIL_USE_TLS  bool   // USE_TLS
	EMAIL_USE_SSL  bool   // USE_SSL
	EMAIL_FROM     string // Mail server from address.

	TIMEOUT      time.Duration
	TLS_Config   *tls.Config
	DEFAULT_AUTH smtp.Auth

	OnSend func(e *email.Email)

	pool *email.Pool
}

func (m *Manager) Init() {
	var pool *email.Pool
	var err error

	var auth smtp.Auth
	if m.DEFAULT_AUTH != nil {
		auth = m.DEFAULT_AUTH
	} else {
		auth = m
	}

	if m.EMAIL_USE_TLS {
		pool, err = email.NewPool(m.Addr(), 10, auth, m.TLS_Config)
	} else {
		pool, err = email.NewPool(m.Addr(), 10, auth)
	}
	if err != nil {
		panic(err)
	}
	if m.TIMEOUT == 0 {
		m.TIMEOUT = 50 * time.Second
	}
	m.pool = pool

}

func (m *Manager) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte(m.EMAIL_USERNAME), nil
}

func (m *Manager) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(m.EMAIL_USERNAME), nil
		case "Password:":
			return []byte(m.EMAIL_PASSWORD), nil
		default:
			return nil, errors.New("unknown from server")
		}
	}
	return nil, nil
}

func (m *Manager) Addr() string {
	return m.EMAIL_HOST + ":" + strconv.Itoa(m.EMAIL_PORT)
}

func (m *Manager) send(e *email.Email) error {

	if m.OnSend != nil {
		m.OnSend(e)
	}

	return m.pool.Send(e, m.TIMEOUT)
}

func (m *Manager) Send(to string, subject string, body string) error {
	// Set up authentication information.
	e := email.NewEmail()
	e.From = m.EMAIL_FROM
	e.To = []string{to}
	e.Subject = subject
	e.Text = []byte(body)
	// return pool.Send(e, 50*time.Second)
	return m.send(e)
}

func (m *Manager) SendWithTemplate(to string, subject string, t *template.Template, data any) error {
	if m.EMAIL_HOST == "" || m.EMAIL_PORT == 0 {
		//lint:ignore ST1005 This is a log message
		return errors.New("Mail server not configured")
	}
	e := email.NewEmail()
	e.From = m.EMAIL_FROM
	e.To = []string{to}
	e.Subject = subject

	var buf bytes.Buffer
	err := t.ExecuteTemplate(&buf, httputils.FilenameFromPath(t.Name()), data)
	if err != nil {
		return err
	}
	e.HTML = buf.Bytes()

	return m.send(e)
}
