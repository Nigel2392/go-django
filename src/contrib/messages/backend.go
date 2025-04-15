package messages

import (
	"encoding/gob"
	"net/http"
)

func init() {
	gob.Register(&BaseMessage{})
	gob.Register([]Message{})
}

type BaseMessage struct {
	Level       MessageTag   `json:"tag"`
	Text        string       `json:"message"`
	ExtraLevels []MessageTag `json:"extra"`
}

func (m *BaseMessage) Prepare() error {
	return nil
}

func (m *BaseMessage) Tag() MessageTag {
	return m.Level
}

func (m *BaseMessage) Message() string {
	return m.Text
}

func (m *BaseMessage) ExtraTags() []MessageTag {
	return m.ExtraLevels
}

func (m *BaseMessage) String() string {
	return m.Message()
}

type Message interface {
	Tag() MessageTag
	Message() string
	ExtraTags() []MessageTag
	Prepare() error
}

type MessageBackend interface {
	Get() (messages []Message, AllRetrieved bool)
	Store(message Message) error
	Level() MessageTag
	SetLevel(level MessageTag) error
	Finalize(w http.ResponseWriter, r *http.Request) error
	Clear() error
}
