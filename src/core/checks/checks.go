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

//  type (
//  	TagDatabaseFunc  func(ctx context.Context, args ...any) []Message // *django.Application, drivers.Database
//  	TagSettingsFunc  func(ctx context.Context, args ...any) []Message // *django.Application, django.Settings
//  	TagSecurityFunc  func(ctx context.Context, args ...any) []Message // *django.Application, django.Settings
//  	TagCommandsFunc  func(ctx context.Context, args ...any) []Message // *django.Application, django.AppConfig, []command.Command
//  	TagModelsFunc    func(ctx context.Context, args ...any) []Message // *django.Application, attrs.Definer
//  )

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

func (m Message) Silenced() bool {
	return registry.Silenced(m.ID) || m.Type.LT(logger.GetLevel())
}

func (m Message) String() string {
	var sb strings.Builder
	sb.WriteString(m.ID)
	if m.Object != nil {
		sb.WriteString(fmt.Sprintf(" (%T)", m.Object))
	}

	sb.WriteString(fmt.Sprintf(": %s", m.Text))

	if m.Hint != "" {
		sb.WriteString(fmt.Sprintf("\n(Hint: %s)\n", m.Hint))
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

var allTags = [...]Tag{
	TagSettings,
	TagSecurity,
	TagCommands,
	TagModels,
}

var (
	registry  = NewCheckRegistry()
	Register  = registry.Register
	RunChecks = registry.RunChecks
	RunCheck  = registry.RunCheck
	Silence   = registry.Silence
	Silenced  = registry.Silenced
	HasTag    = registry.HasTag
)

type checkRegistry struct {
	checks   map[Tag][]*django_reflect.Func
	silenced map[string]struct{}
}

func NewCheckRegistry() *checkRegistry {
	return &checkRegistry{
		checks:   make(map[Tag][]*django_reflect.Func),
		silenced: make(map[string]struct{}),
	}
}

func (r *checkRegistry) Silenced(id string) bool {
	_, ok := r.silenced[id]
	return ok
}

func (r *checkRegistry) Silence(id string, silenced bool) {
	if silenced {
		r.silenced[id] = struct{}{}
	} else {
		delete(r.silenced, id)
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

func (r *checkRegistry) RunChecks(ctx context.Context, tags []Tag, args ...any) CheckResult {
	if len(tags) == 0 {
		tags = allTags[:]
	}

	var results = &checksResult{}
	for _, tag := range tags {
		if !r.HasTag(tag) {
			continue
		}

		var res = r.RunCheck(ctx, tag, args...)
		if len(res) == 0 {
			continue
		}

		results.m = append(results.m, res...)
	}

	return results
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
