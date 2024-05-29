package django

import (
	"github.com/Nigel2392/django/core/errs"
	"github.com/Nigel2392/django/core/http_"
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

func AppMiddleware(middleware ...http_.Middleware) func(*Application) error {
	return func(a *Application) error {
		a.Middleware = append(a.Middleware, middleware...)
		return nil
	}
}

func AppURLs(u ...http_.URL) func(*Application) error {
	return func(a *Application) error {
		if a.URLs == nil {
			a.URLs = make([]http_.URL, 0)
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
