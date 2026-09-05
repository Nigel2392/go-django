package messages

import (
	"context"
	"fmt"
	"net/http"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/logger"
)

type backendContextKey struct{}

const (
	ErrRequestRequired      = errs.Error("messages: request required")
	ErrBackendNotConfigured = errs.Error("messages: backend not configured")
	ErrMessagesNotInstalled = errs.Error("messages: app not installed")
)

var (
	backendKey = backendContextKey{}
)

func Tags() *MessageTags {
	return app.Tags
}

func setBackend(r *http.Request, backend MessageBackend) *http.Request {
	if backend == nil {
		return r
	}

	var ctx = r.Context()
	ctx = context.WithValue(ctx, backendKey, backend)
	return r.WithContext(ctx)
}

func BackendFromContext(r *http.Request) MessageBackend {
	var b = r.Context().Value(backendKey)
	if b != nil {
		return b.(MessageBackend)
	}

	return nil
}

func Backend(r *http.Request) MessageBackend {
	if r == nil {
		return nil
	}

	var b = BackendFromContext(r)
	if b != nil {
		return b
	}

	except.AssertNotNil(
		app.initBackend,
		500, ErrBackendNotConfigured,
	)

	backend, err := app.initBackend(r)
	if err != nil {
		return nil
	}

	return backend
}

func ConfigureBackend(initBackend func(r *http.Request) (MessageBackend, error)) {
	app.ConfigureBackend(initBackend)
}

func AddMessage(r *http.Request, tag MessageTag, message string, extraTags ...MessageTag) error {
	if !django.AppInstalled(MESSAGES_NAMESPACE) {
		logger.NameSpace(MESSAGES_NAMESPACE).Warn("Messages app not installed, not sending any messages")
		return ErrMessagesNotInstalled
	}

	var backend = Backend(r)
	if backend == nil {
		return ErrBackendNotConfigured
	}

	app.Logger().Debugf(
		"Adding message: %s, level: %d, extraTags: %v",
		message, TagLevels[tag], extraTags,
	)

	return backend.Store(&BaseMessage{
		Level:       tag,
		Text:        message,
		ExtraLevels: extraTags,
	})
}

func Messages(r *http.Request) (messages []Message, AllRetrieved bool) {
	var backend = Backend(r)
	if backend == nil {
		return nil, false
	}
	return backend.Get()
}

func GetLevel(r *http.Request) (MessageTag, error) {
	var backend = Backend(r)
	if backend == nil {
		return DEBUG, ErrBackendNotConfigured
	}
	return backend.Level(), nil
}

func SetLevel(r *http.Request, level MessageTag) error {
	var backend = Backend(r)
	if backend == nil {
		return ErrBackendNotConfigured
	}
	return backend.SetLevel(level)
}

func Debug(r *http.Request, message string, extraTags ...MessageTag) error {
	return AddMessage(r, DEBUG, message, extraTags...)
}

func Info(r *http.Request, message string, extraTags ...MessageTag) error {
	return AddMessage(r, INFO, message, extraTags...)
}

func Success(r *http.Request, message string, extraTags ...MessageTag) error {
	return AddMessage(r, SUCCESS, message, extraTags...)
}

func Warning(r *http.Request, message string, extraTags ...MessageTag) error {
	return AddMessage(r, WARNING, message, extraTags...)
}

func Error(r *http.Request, message string, extraTags ...MessageTag) error {
	return AddMessage(r, ERROR, message, extraTags...)
}

func tagsFromArgs(args []any) ([]any, []MessageTag) {
	var (
		t       = make([]MessageTag, 0)
		lastIdx int
	)

loop:
	for i, arg := range args {
		switch a := arg.(type) {
		case MessageTag:
			t = append(t, a)
		case []MessageTag:
			t = append(t, a...)
		default:
			lastIdx = i
			break loop
		}
	}

	return args[lastIdx:], t
}

func Debugf(r *http.Request, message string, argsAndTags ...any) error {
	args, tags := tagsFromArgs(argsAndTags)
	message = fmt.Sprintf(message, args...)
	return AddMessage(r, DEBUG, message, tags...)
}

func Infof(r *http.Request, message string, argsAndTags ...any) error {
	args, tags := tagsFromArgs(argsAndTags)
	message = fmt.Sprintf(message, args...)
	return AddMessage(r, INFO, message, tags...)
}

func Successf(r *http.Request, message string, argsAndTags ...any) error {
	args, tags := tagsFromArgs(argsAndTags)
	message = fmt.Sprintf(message, args...)
	return AddMessage(r, SUCCESS, message, tags...)
}

func Warningf(r *http.Request, message string, argsAndTags ...any) error {
	args, tags := tagsFromArgs(argsAndTags)
	message = fmt.Sprintf(message, args...)
	return AddMessage(r, WARNING, message, tags...)
}

func Errorf(r *http.Request, message string, argsAndTags ...any) error {
	args, tags := tagsFromArgs(argsAndTags)
	message = fmt.Sprintf(message, args...)
	return AddMessage(r, ERROR, message, tags...)
}
