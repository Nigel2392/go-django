package mail

import (
	"crypto/tls"
	"errors"
	"net/smtp"
	"time"
)

type Config struct {
	Host        string        // Mail server host.
	Port        int           // Mail server port.
	Username    string        // Mail server username.
	Password    string        // Mail server password.
	UseTLS      bool          // USE_TLS
	UseSSL      bool          // USE_SSL
	MailFrom    string        // Mail server from address.
	Timeout     time.Duration // Timeout duration for sending email.
	TLSConfig   *tls.Config   // TLS Config
	DefaultAuth smtp.Auth     // Default SMTP Auth
}

func (m *Config) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte(m.Username), nil
}

func (m *Config) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(m.Username), nil
		case "Password:":
			return []byte(m.Password), nil
		default:
			return nil, errors.New(
				"unknown credential requested from server: %s",
			)
		}
	}
	return nil, nil
}
