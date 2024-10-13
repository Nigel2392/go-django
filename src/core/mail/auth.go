package mail

import (
	"fmt"
	"net/smtp"
)

type plainAuth struct {
	identity, username, password string
	host                         string
}

func (a *plainAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if server.Name != a.host {
		return "", nil, fmt.Errorf("wrong host name")
	}
	var response = fmt.Sprintf(
		"\x00%s\x00%s\x00%s",
		a.identity, a.username, a.password,
	)
	return "PLAIN", []byte(response), nil
}

func (a *plainAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	return nil, nil // No further steps in PLAIN auth
}

func PlainAuth(identity, username, password, host string) smtp.Auth {
	return &plainAuth{identity, username, password, host}
}

type xoauth2Auth struct {
	username, token string
}

func (a *xoauth2Auth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	var authString = fmt.Sprintf(
		"user=%s\x01auth=Bearer %s\x01\x01",
		a.username, a.token,
	)
	return "XOAUTH2", []byte(authString), nil
}

func (a *xoauth2Auth) Next(fromServer []byte, more bool) ([]byte, error) {
	return nil, nil // No further steps in XOAUTH2
}

func XOAuth2Auth(username, token string) smtp.Auth {
	return &xoauth2Auth{username, token}
}
