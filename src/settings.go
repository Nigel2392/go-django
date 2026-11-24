package django

import (
	"github.com/Nigel2392/go-django/src/utils/djangoapps"
)

type Settings interface {
	djangoapps.BaseSettings
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
