package checks

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/internal/django_reflect"
)

/*
	Checks are used to verify the state of the application, settings, database, etc.
	They are registered with the `Register` function and can be run with `RunChecks`.

	A check is a function that takes a context and returns a list of messages.
	Messages are used to report the state of the application, settings, database, etc.

	Objects, fields and other types might also implement the [Checker] interface,
	which allows them to be checked for their state and return messages.
*/

type (
	Type = logger.LogLevel

	USELESS = struct{} // helps with registering at global variable scope
)

const (
	CRITICAL Type = logger.CRIT
	ERROR    Type = logger.ERR
	WARNING  Type = logger.WRN
	INFO     Type = logger.INF
	DEBUG    Type = logger.DBG

	SERIOUS_TYPE = ERROR
)

type Checker interface {
	Check(ctx context.Context) []Message
}

type Message struct {
	ID     string
	Type   Type
	Object any
	Hint   string
	Text   string
}

// New creates a new message with the same ID and type as the original message,
//
// but with a new object and an optional hint.
//
// It is used to create a new message with the same ID and type as an existing message,
// but with a different object and/or hint.
//
// This is useful when you want to create global messages that can be reused
// with different objects or hints, while still keeping the same ID and type.
//
// Any Message created with this method will still compare as equal when calling [Message.Is].
func (m Message) New(object any, hint ...string) Message {
	var hintText = m.Hint
	if len(hint) > 0 {
		hintText = hint[0]
	}
	return Message{
		ID:     m.ID,
		Type:   m.Type,
		Object: object,
		Hint:   hintText,
		Text:   m.Text,
	}
}

// Is checks if the message is of the same type and ID as another message.
//
// It is used to compare messages for equality, ignoring the text and object fields.
//
// It should still compare as true when it is compared to a message called with [Message.New]().
func (m Message) Is(other Message) bool {
	return m.ID == other.ID &&
		m.Type == other.Type
}

// IsSerious checks if the message is of a serious type (ERROR or higher).
// It is used to determine if the message should be treated as a serious issue
//
// If, for any reason, the seriousness comparison needs to be changed,
// it can be done by changing the value of [SERIOUS_TYPE].
func (m Message) IsSerious() bool {
	return m.Type >= SERIOUS_TYPE
}

// Silenced checks if the message is silenced or not.
//
// A message is silenced if its ID is in the registry's silenced list,
// or if its type (analogous to logger's log level) is lower than the current logger level.
func (m Message) Silenced(ctx context.Context) bool {
	return registry.Silenced(m) || m.Type.LT(logger.GetLevel())
}

type TypeStringer interface {
	TypeString() string
}

// String returns a string representation of the message.
// It includes the ID, type, object type (if any), text, and hint (if any).
func (m Message) String(ctx context.Context) string {
	var sb strings.Builder

	sb.WriteString(m.ID)

	if m.Object != nil {
		sb.WriteString(" ")
		sb.WriteString("(")
		var typeStr string
		switch typ := m.Object.(type) {
		case string:
			typeStr = typ
		case TypeStringer:
			typeStr = typ.TypeString()
		case int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64,
			float32, float64:
			typeStr = fmt.Sprintf("%v", typ)
		default:
			var ts = fmt.Sprintf("%T", m.Object)
			if len(ts) > 30 {
				// If the type string is too long, truncate it to 30 characters
				ts = fmt.Sprintf("%s...", ts[:27])
			}
			typeStr = ts
		}
		sb.WriteString(logger.Colorize(
			ctx, logger.CMD_Red,
			typeStr,
		))
		sb.WriteString(")")
	}

	sb.WriteString(fmt.Sprintf(": %s", m.Text))

	if m.Hint != "" {
		//sb.WriteString(fmt.Sprintf(
		//	"\n\t(%s%sHint:%s %s)\n",
		//	logger.CMD_Bold, logger.CMD_Green,
		//	logger.CMD_Reset,
		//	m.Hint,
		//))
		sb.WriteString("\n\t(")
		sb.WriteString(logger.Colorize(
			ctx, []string{logger.CMD_Bold, logger.CMD_Green},
			"Hint:",
		))
		sb.WriteString(" ")
		sb.WriteString(m.Hint)
		sb.WriteString(")")
	}

	return sb.String()
}

func typed(typ Type, ID, text string, object any, hint ...string) Message {
	var hintText string
	if len(hint) > 0 {
		hintText = hint[0]
	}
	return Message{
		ID:     ID,
		Type:   typ,
		Object: object,
		Hint:   hintText,
		Text:   text,
	}
}

func Criticalf(ID, text string, object any, hint string, args ...any) Message {
	return typed(
		CRITICAL, ID, fmt.Sprintf(text, args...), object, hint,
	)
}

func Errorf(ID, text string, object any, hint string, args ...any) Message {
	return typed(
		ERROR, ID, fmt.Sprintf(text, args...), object, hint,
	)
}

func Warningf(ID, text string, object any, hint string, args ...any) Message {
	return typed(
		WARNING, ID, fmt.Sprintf(text, args...), object, hint,
	)
}

func Infof(ID, text string, object any, hint string, args ...any) Message {
	return typed(
		INFO, ID, fmt.Sprintf(text, args...), object, hint,
	)
}

func Debugf(ID, text string, object any, hint string, args ...any) Message {
	return typed(
		DEBUG, ID, fmt.Sprintf(text, args...), object, hint,
	)
}

func Critical(ID, text string, object any, hint ...string) Message {
	return typed(CRITICAL, ID, text, object, hint...)
}

func Error(ID, text string, object any, hint ...string) Message {
	return typed(ERROR, ID, text, object, hint...)
}

func Warning(ID, text string, object any, hint ...string) Message {
	return typed(WARNING, ID, text, object, hint...)
}

func Info(ID, text string, object any, hint ...string) Message {
	return typed(INFO, ID, text, object, hint...)
}

func Debug(ID, text string, object any, hint ...string) Message {
	return typed(DEBUG, ID, text, object, hint...)
}

type Tag string

const (
	TagSettings Tag = "settings"
	TagSecurity Tag = "security"
	TagCommands Tag = "commands"
	TagModels   Tag = "models"
)

var (
	registry = NewCheckRegistry()
	Register = registry.Register
	RunCheck = registry.RunCheck
	Shutup   = registry.Shutup
	HasTag   = registry.HasTag
)

type checkRegistry struct {
	checks   map[Tag][]*django_reflect.Func
	silenced map[string]func(m Message) bool
}

func NewCheckRegistry() *checkRegistry {
	return &checkRegistry{
		checks:   make(map[Tag][]*django_reflect.Func),
		silenced: make(map[string]func(m Message) bool),
	}
}

func (r *checkRegistry) Silenced(m Message) bool {
	if fn, ok := r.silenced[m.ID]; ok {
		return fn(m)
	}

	return m.Type.LT(logger.GetLevel())
}

func (r *checkRegistry) Shutup(id string, yes any) {
	switch yes := yes.(type) {
	case bool:
		if yes {
			r.silenced[id] = func(m Message) bool {
				return true
			}
		} else {
			delete(r.silenced, id)
		}
	case func(m Message) bool:
		r.silenced[id] = yes
	default:
		panic(fmt.Sprintf(
			"shutup expects a bool or a function, got %T", yes,
		))
	}
}

func (r *checkRegistry) Register(tag Tag, check any) USELESS {
	var checkType = reflect.TypeOf(check)
	if checkType.Kind() != reflect.Func {
		panic(fmt.Sprintf(
			"check must be a function, got %s", checkType.Kind(),
		))
	}

	var checks, ok = r.checks[tag]
	if !ok {
		checks = make([]*django_reflect.Func, 0, 1)
	}

	var fn = django_reflect.NewFunc(check, reflect.TypeOf([]Message{}))
	fn.BeforeExec = func(in []reflect.Value) error {
		if len(in) == 0 {
			return errors.New("check function must accept at least one argument (context.Context)")
		}
		if in[0].Type() != reflect.TypeOf((*context.Context)(nil)).Elem() {
			return fmt.Errorf(
				"check function first argument must be context.Context, got %s",
				in[0].Type(),
			)
		}
		return nil
	}

	r.checks[tag] = append(
		checks, fn,
	)

	return USELESS{}
}

func (r *checkRegistry) RunCheck(ctx context.Context, tag Tag, args ...any) []Message {
	if !r.HasTag(tag) {
		return nil
	}

	var checks = r.checks[tag]
	var results = make([]Message, 0, len(checks))
	for _, check := range checks {
		var res = check.Call(append([]any{ctx}, args...)...)
		var messages = res[0].([]Message)
		results = append(results, messages...)
	}

	return results
}

func (r *checkRegistry) HasTag(tag Tag) bool {
	l, ok := r.checks[tag]
	return ok && len(l) > 0
}
