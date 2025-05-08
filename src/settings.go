package django

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/Nigel2392/go-django/src/core/assert"
)

type Settings interface {
	Set(key string, value interface{})
	Get(key string) (any, bool)
	Bind(app *Application) error
	App() *Application
}

type settings struct {
	Data map[string]any
	a    *Application
}

func Config(m map[string]interface{}) Settings {
	if m == nil {
		m = make(map[string]interface{})
	}
	return &settings{
		Data: m,
	}
}

func Configure(m map[string]interface{}) func(*Application) error {
	return func(a *Application) error {
		var s = Config(m)
		a.Settings = s
		return s.Bind(a)
	}
}

func (s *settings) Bind(app *Application) error {
	s.a = app
	app.Settings = s
	return nil
}

func (s *settings) App() *Application {
	return s.a
}

func (s *settings) Set(key string, value interface{}) {
	s.Data[key] = value
}

func (s *settings) Get(key string) (any, bool) {
	var value, ok = s.Data[key]
	return value, ok
}

func ConfigGet[T any](s Settings, key string, default_ ...T) T {
	var value, _ = ConfigGetOK[T](s, key, default_...)
	return value
}

func ConfigGetOK[T any](s Settings, key string, default_ ...T) (T, bool) {
	if s == nil && len(default_) == 0 {
		return *(new(T)), false
	}

	assert.Lt(default_, 2, "Too many arguments")
	assert.Gte(default_, 0, "Too few arguments")

	if s == nil {
		return default_[0], false
	}

	var value, ok = s.Get(key)
	if !ok && len(default_) > 0 {
		return default_[0], false
	} else if !ok || value == nil {
		return *(new(T)), false
	}

	var t T
	var rTyp = reflect.TypeOf(t)
	if s, ok := value.(string); ok {
		if s == "" {
			return default_[0], true
		}

		if rTyp.Kind() == reflect.String {
			v, ok := value.(T)
			assert.True(ok, "Invalid type for key %s", key)
			return v, true
		}

		switch rTyp.Kind() {
		case reflect.String:
			return value.(T), true
		case reflect.Bool:
			var v, err = strconv.ParseBool(s)
			assert.ErrNil(fmt.Errorf("invalid value for key %s: %v (%v)", key, v, err))
			return any(v).(T), true
		case reflect.Int:
			var v, err = strconv.Atoi(s)
			assert.ErrNil(fmt.Errorf("invalid value for key %s: %v (%v)", key, v, err))
			return any(v).(T), true
		case reflect.Int64:
			var v, err = strconv.ParseInt(s, 10, 64)
			assert.ErrNil(fmt.Errorf("invalid value for key %s: %v (%v)", key, v, err))
			return any(v).(T), true
		case reflect.Float64:
			var v, err = strconv.ParseFloat(s, 64)
			assert.ErrNil(fmt.Errorf("invalid value for key %s: %v (%v)", key, v, err))
			return any(v).(T), true
		default:
			assert.Fail("Invalid type for key %s", key)
		}
	}

	v, ok := value.(T)
	assert.True(ok, "Invalid type for key %s", key)
	return v, true
}
