package django

import (
	"context"
	"crypto/tls"
	"fmt"
	"reflect"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/checks"
	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/alexedwards/scs/v2"
)

func settingMatchesType(rhs reflect.Type, settings Settings, key string) (value any, present, matches bool) {
	value, present = settings.Get(key)
	if !present {
		return nil, present, matches
	}

	if value == nil {
		return nil, present, matches
	}

	var (
		lhs    = reflect.TypeOf(value)
		lhsVal = reflect.ValueOf(value)
	)
	if rhs.Kind() == reflect.Interface && lhs.Implements(rhs) {
		var z = reflect.New(rhs)
		z.Elem().Set(lhsVal)
		return z.Interface(), present, true
	}

	if !lhs.AssignableTo(rhs) && !lhs.ConvertibleTo(rhs) {
		return value, present, matches
	}

	if lhs == rhs {
		// If the types are the same, we can return the value as is
		return value, present, true
	}

	// Convert the value to the required type
	value = lhsVal.Convert(rhs).Interface()
	return value, present, matches
}

type _settingsCheckImpl[T any] struct {
	setting   string
	isPresent func(setting T) []string
}

func (s _settingsCheckImpl[T]) name() string {
	return s.setting
}

func (s _settingsCheckImpl[T]) requiredType() reflect.Type {
	return reflect.TypeOf(new(T)).Elem()
}

func (s _settingsCheckImpl[T]) present() func(setting any) []string {
	if s.isPresent == nil {
		return nil
	}
	return func(setting any) []string {
		return s.isPresent(setting.(T))
	}
}

type _settingsCheck interface {
	name() string
	present() func(setting any) []string
	requiredType() reflect.Type
}

func typeStr(v any) []byte {
	var t reflect.Type
	switch vt := v.(type) {
	case reflect.Type:
		t = vt
	default:
		t = reflect.TypeOf(v)
	}

	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		return append([]byte("[]"), typeStr(t.Elem())...)
	case reflect.Map:
		return append(append([]byte("map["), typeStr(t.Key())...), append([]byte("]"), typeStr(t.Elem())...)...)
	case reflect.Pointer:
		return append([]byte("*"), typeStr(t.Elem())...)
	}
	return []byte(t.Name())
}

var _ = checks.Register(checks.TagSettings, func(ctx context.Context, app *Application, settings Settings) (messages []checks.Message) {

	var settingsChecks = []_settingsCheck{
		_settingsCheckImpl[bool]{setting: APPVAR_DEBUG},
		_settingsCheckImpl[[]string]{setting: APPVAR_ALLOWED_HOSTS, isPresent: func(setting []string) []string {
			if len(setting) > 0 {
				return nil
			}
			return []string{"ALLOWED_HOSTS setting must not be empty"}
		}},
		_settingsCheckImpl[bool]{setting: APPVAR_RECOVERER},
		_settingsCheckImpl[string]{setting: APPVAR_HOST},
		_settingsCheckImpl[string]{setting: APPVAR_PORT},
		_settingsCheckImpl[string]{setting: APPVAR_STATIC_URL},
		_settingsCheckImpl[string]{setting: APPVAR_TLS_PORT},
		_settingsCheckImpl[string]{setting: APPVAR_TLS_CERT},
		_settingsCheckImpl[string]{setting: APPVAR_TLS_KEY},
		_settingsCheckImpl[*tls.Config]{setting: APPVAR_TLS_CONFIG},
		_settingsCheckImpl[drivers.Database]{setting: APPVAR_DATABASE},
		_settingsCheckImpl[bool]{setting: APPVAR_CONTINUE_AFTER_COMMANDS},
		_settingsCheckImpl[bool]{setting: APPVAR_ROUTE_LOGGING_ENABLED},
		_settingsCheckImpl[bool]{setting: APPVAR_REQUESTS_PROXIED},
		_settingsCheckImpl[*scs.SessionManager]{setting: APPVAR_SESSION_MANAGER},
		_settingsCheckImpl[bool]{setting: APPVAR_DISABLE_NOSURF},
	}

	for _, check := range settingsChecks {
		var (
			checkName    = check.name()
			checkID      = fmt.Sprintf("settings.matches.%s", checkName)
			presentFn    = check.present()
			requiredType = check.requiredType()
		)

		var value, present, matches = settingMatchesType(requiredType, settings, checkName)
		if !present && presentFn != nil {
			messages = append(messages, checks.Critical(
				checkID,
				fmt.Sprintf("setting %q was not found in the settings", checkName),
				nil,
				fmt.Sprintf("please set %q in the settings to the correct type %s", checkName, typeStr(requiredType)),
			))
			continue
		}

		// if the setting is not present it can be skipped
		// if we reach here and the setting is not present,
		// it means that the present function is not set.
		if !present {
			continue
		}

		// if the setting is present, but does not match the required type
		// we add the error to the list of messages
		if !matches {
			messages = append(messages, checks.Critical(
				checkID,
				fmt.Sprintf("setting %q (%s) is not of type %s", checkName, typeStr(value), typeStr(requiredType)),
				nil,
				fmt.Sprintf("please set %q in the settings to the correct type %s", checkName, typeStr(requiredType)),
			))
			continue
		}

		if presentFn == nil {
			continue
		}

		for _, msg := range presentFn(value) {
			messages = append(messages, checks.Critical(
				checkID, msg, nil,
			))
		}
	}

	return messages
})

var _ = checks.Register(checks.TagCommands, func(ctx context.Context, app *Application, settings Settings, commands []command.Command) (messages []checks.Message) {
	for _, cmd := range commands {
		if checker, ok := cmd.(checks.Checker); ok {
			messages = append(messages, checker.Check(ctx)...)
		}
	}
	return messages
})
