package mail

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"time"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/jordan-wright/email"
)

type pooledManager struct {
	cnf  *Config
	pool *email.Pool
}

func NewPooledEmailBackend(poolCount int, cnf *Config) (EmailBackend, error) {
	if cnf == nil {
		return nil, ErrConfigNil
	}

	var msg = "%s cannot be an empty string"
	assert.Truthy(cnf.Host, msg, "EmailHost")
	assert.Truthy(cnf.Username, msg, "EmailUsername")
	assert.Truthy(cnf.Password, msg, "Password")
	assert.Truthy(cnf.MailFrom, msg, "MailFrom")

	if cnf.Port == 0 {
		cnf.Port = 25
	}

	if cnf.Timeout == 0 {
		cnf.Timeout = time.Second * 10
	}

	var tlsConfig = make([]*tls.Config, 0)
	if cnf.UseTLS && cnf.TLSConfig != nil {
		tlsConfig = append(tlsConfig, cnf.TLSConfig)
	} else if cnf.UseTLS {
		tlsConfig = append(tlsConfig, &tls.Config{
			ServerName: cnf.Host,
		})
	}

	var auth smtp.Auth = nil
	if cnf.DefaultAuth != nil {
		auth = cnf.DefaultAuth
	} else {
		auth = cnf
	}

	var pool, err = email.NewPool(
		fmt.Sprintf("%s:%d", cnf.Host, cnf.Port),
		poolCount,
		auth,
		tlsConfig...,
	)
	if err != nil {
		return nil, err
	}

	return &pooledManager{
		cnf:  cnf,
		pool: pool,
	}, nil
}

func (m *pooledManager) Open() error {
	return nil
}

func (m *pooledManager) Send(e *email.Email) error {
	return m.pool.Send(e, m.cnf.Timeout)
}

func (m *pooledManager) Close() error {
	m.pool.Close()
	return nil
}
