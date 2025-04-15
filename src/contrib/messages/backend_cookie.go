package messages

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"net/http"
	"slices"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/logger"
)

var (
	_ MessageBackend = (*CookieBackend)(nil)

	// The base coookie from which cookies will be created.
	// This can be used to set default values for the cookies.
	// For example, the domain, path, max-age, etc.
	cookieBackendBaseCoookie = http.Cookie{
		Name: "messages.cookieBackendKey",
	}
)

const (
	APPVAR_COOKIE_KEY = "messages.cookieKey"
)

type CookieBackend struct {
	Request        *http.Request
	BaseCookie     *http.Cookie
	QueuedMessages []Message
	MinTagLevel    uint
	used           bool
	cleared        bool
}

func encodeMessages(messages []Message) (string, error) {
	var b = new(bytes.Buffer)
	enc := gob.NewEncoder(b)
	err := enc.Encode(messages)
	if err != nil {
		return "", err
	}
	var encoded = base64.StdEncoding.EncodeToString(
		b.Bytes(),
	)
	return encoded, nil
}

func decodeMessages(encoded string) ([]Message, error) {
	var decoded, err = base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	var b = new(bytes.Buffer)
	b.Write(decoded)
	dec := gob.NewDecoder(b)

	var messages []Message
	err = dec.Decode(&messages)
	return messages, err
}

func NewCookieBackend(r *http.Request) (MessageBackend, error) {
	var key = django.ConfigGet(
		django.Global.Settings,
		APPVAR_COOKIE_KEY,
		&cookieBackendBaseCoookie,
	)

	var cookie, err = r.Cookie(key.Name)
	if err != nil {
		if err != http.ErrNoCookie {
			logger.Errorf("Error retrieving cookie: %v", err)
		}
	}

	var messages = make([]Message, 0)
	if cookie != nil {
		messages, err = decodeMessages(cookie.Value)
		if err != nil {
			logger.Errorf("Error decoding cookie: %v", err)
			messages = make([]Message, 0)
		}
	}

	return &CookieBackend{
		Request:        r,
		MinTagLevel:    TagLevels[DefaultTags.Debug],
		BaseCookie:     key,
		QueuedMessages: messages,
	}, nil
}

func (d *CookieBackend) Finalize(w http.ResponseWriter, r *http.Request) error {
	if d.cleared {
		http.SetCookie(w, &http.Cookie{
			Name:   d.BaseCookie.Name,
			Value:  "",
			MaxAge: -1,
		})
		return nil
	}

	if !d.used {
		return nil
	}

	var encoded, err = encodeMessages(d.QueuedMessages)
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     d.BaseCookie.Name,
		Value:    encoded,
		Path:     d.BaseCookie.Path,
		Domain:   d.BaseCookie.Domain,
		Expires:  d.BaseCookie.Expires,
		MaxAge:   d.BaseCookie.MaxAge,
		HttpOnly: d.BaseCookie.HttpOnly,
		SameSite: d.BaseCookie.SameSite,
	})

	return nil
}

func (d *CookieBackend) Get() (messages []Message, AllRetrieved bool) {
	if len(d.QueuedMessages) == 0 {
		return nil, false
	}
	return slices.Clone(d.QueuedMessages), true
}

func (d *CookieBackend) Store(message Message) error {
	if message.Message() == "" {
		return nil
	}

	if TagLevels[message.Tag()] < d.MinTagLevel {
		return nil
	}

	d.QueuedMessages = append(d.QueuedMessages, message)
	d.used = true
	d.cleared = false

	return nil
}

func (d *CookieBackend) Clear() error {
	if len(d.QueuedMessages) == 0 {
		return nil
	}
	d.QueuedMessages = make([]Message, 0)
	d.used = false
	d.cleared = true
	return nil
}

func (d *CookieBackend) Level() MessageTag {
	return LevelTags[d.MinTagLevel]
}

func (d *CookieBackend) SetLevel(level MessageTag) error {
	d.MinTagLevel = TagLevels[level]
	return nil
}
