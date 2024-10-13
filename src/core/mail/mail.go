package mail

import (
	"errors"
	"net/smtp"
	"os"

	"github.com/jordan-wright/email"
)

var (
	_                  smtp.Auth = (*Config)(nil)
	ErrConfigNil                 = errors.New("config is nil")
	ErrBackendNotFound           = errors.New("backend not found")
)

const (
	DefaultBackend = "default"
)

func init() {
	Register(DefaultBackend, NewConsoleBackend(
		os.Stdout,
	))
}

type OpenableEmailBackend interface {
	Open() error
	IsOpen() bool
	Close() error
}

type EmailBackend interface {
	Send(e *email.Email) error
}

type backendRegistry struct {
	backends map[string]EmailBackend
}

var registry = &backendRegistry{
	backends: make(map[string]EmailBackend),
}

func Register(name string, backend EmailBackend) {
	if name != DefaultBackend && Default() == nil {
		SetDefault(backend)
	}
	registry.backends[name] = backend
}

func Unregister(name string) {
	delete(registry.backends, name)
}

func Get(name string) EmailBackend {
	var backend = registry.backends[name]
	openable, ok := backend.(OpenableEmailBackend)
	if ok && !openable.IsOpen() {
		openable.Open()
	}
	return backend
}

func Default() EmailBackend {
	return Get(DefaultBackend)
}

func SetDefault(backend EmailBackend) {
	Register(DefaultBackend, backend)
}

func backendByNameOrInstance(backendOrName ...interface{}) EmailBackend {
	for _, b := range backendOrName {
		switch b := b.(type) {
		case string:
			return Get(b)
		case EmailBackend:
			return b
		}
	}
	return Default()
}

func Send(e *email.Email, backendOrName ...interface{}) error {
	var backend = backendByNameOrInstance(
		backendOrName...,
	)
	if backend == nil {
		return ErrBackendNotFound
	}
	if openable, ok := backend.(OpenableEmailBackend); ok && !openable.IsOpen() {
		if err := openable.Open(); err != nil {
			return err
		}
	}

	return backend.Send(e)
}

func Close[T string | interface{}](backendOrName T) error {
	var backend = backendByNameOrInstance(
		backendOrName,
	)
	if openable, ok := backend.(OpenableEmailBackend); ok {
		return openable.Close()
	}
	return nil
}
