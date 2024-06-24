package django

import (
	core "github.com/Nigel2392/django/core"
	"github.com/Nigel2392/django/core/errs"
	"github.com/Nigel2392/django/core/logger"
	"github.com/Nigel2392/mux"
)

func AppSettings(settings Settings) func(*Application) error {
	return func(a *Application) error {
		if a.Settings != nil {
			return errs.Error("Settings already set")
		}

		settings.Bind(a)
		a.Settings = settings
		return nil
	}
}

func SkipDependencyChecks() func(*Application) error {
	return func(a *Application) error {
		a.skipDepsCheck = true
		return nil
	}
}

func AppMiddleware(middleware ...mux.Middleware) func(*Application) error {
	var m = make([]core.Middleware, 0, len(middleware))
	for _, mw := range middleware {
		m = append(m, core.NewMiddleware(mw))
	}

	return func(a *Application) error {
		a.Middleware = append(a.Middleware, m...)
		return nil
	}
}

func AppURLs(u ...core.URL) func(*Application) error {
	return func(a *Application) error {
		if a.URLs == nil {
			a.URLs = make([]core.URL, 0)
		}
		a.URLs = append(a.URLs, u...)
		return nil
	}
}

func Apps(apps ...any) func(*Application) error {
	return func(a *Application) error {
		a.Register(apps...)
		return nil
	}
}

func AppLogger(w logger.Log) func(*Application) error {
	return func(a *Application) error {
		logger.Setup(w)
		a.Log = w
		return nil
	}
}
