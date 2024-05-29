package django

import (
	"fmt"
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
		return s.Bind(a)
	}
}

func (s *settings) Bind(app *Application) error {
	s.a = app
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

	if len(default_) > 1 {
		panic("Too many arguments")
	}

	if s == nil {
		return default_[0], false
	}

	var value, ok = s.Get(key)
	if !ok && len(default_) == 1 {
		return default_[0], false
	} else if !ok {
		return *(new(T)), false

	}

	v, ok := value.(T)
	if !ok {
		panic(fmt.Sprintf("Invalid type for key %s", key))
	}

	return v, true
}
