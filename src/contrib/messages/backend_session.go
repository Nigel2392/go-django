package messages

import (
	"net/http"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/mux/middleware/sessions"
	"github.com/pkg/errors"
)

var (
	_                 MessageBackend = (*SessionBackend)(nil)
	sessionBackendKey                = "messages.sessionBackendKey"
)

type SessionBackend struct {
	Request     *http.Request
	MinTagLevel uint
}

func NewSessionBackend(r *http.Request) (MessageBackend, error) {
	if !django.AppInstalled("session") {
		return nil, errors.Wrap(
			ErrBackendNotConfigured,
			"session backend requires 'session' app to be installed",
		)
	}

	return &SessionBackend{
		Request:     r,
		MinTagLevel: TagLevels[DefaultTags.Debug],
	}, nil
}

func (b *SessionBackend) Finalize(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (b *SessionBackend) Level() MessageTag {
	return LevelTags[b.MinTagLevel]
}

func (b *SessionBackend) SetLevel(level MessageTag) error {
	b.MinTagLevel = TagLevels[level]
	return nil
}

func (b *SessionBackend) Get() (msgs []Message, AllRetrieved bool) {
	var session = sessions.Retrieve(b.Request)
	if session == nil {
		return nil, false
	}

	var sessionMessages = session.Get(sessionBackendKey)
	if sessionMessages == nil {
		logger.NameSpace(MESSAGES_NAMESPACE).Debug("No messages found in session")
		return nil, false
	}

	var ok bool
	if msgs, ok = sessionMessages.([]Message); ok {
		logger.NameSpace(MESSAGES_NAMESPACE).Debugf("Retrieved %d messages from session", len(msgs))
		return msgs, true
	}

	return nil, false
}

func (b *SessionBackend) Store(message Message) error {
	if message.Message() == "" {
		return nil
	}

	if TagLevels[message.Tag()] < b.MinTagLevel {
		return nil
	}

	var session = sessions.Retrieve(b.Request)
	if session == nil {
		return nil
	}

	sessionMessages := session.Get(sessionBackendKey)
	if sessionMessages == nil {
		sessionMessages = make([]Message, 0)
	}

	logger.NameSpace(MESSAGES_NAMESPACE).Debugf(
		"Storing into backend, message: %s, level: %v, extraTags: %v",
		message.Message(), message.Tag(), message.ExtraTags(),
	)

	if msgs, ok := sessionMessages.([]Message); ok {
		msgs = append(msgs, message)
		session.Set(sessionBackendKey, msgs)
	} else {
		session.Set(sessionBackendKey, []Message{message})
	}

	return nil
}

func (b *SessionBackend) Clear() error {
	var session = sessions.Retrieve(b.Request)
	if session == nil {
		return nil
	}

	sessionMessages := session.Get(sessionBackendKey)
	if sessionMessages != nil {
		session.Delete(sessionBackendKey)
	}

	return nil
}
