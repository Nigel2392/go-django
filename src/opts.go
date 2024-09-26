package django

import (
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/logger"
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

func Flag(flags ...AppFlag) func(*Application) error {
	return func(a *Application) error {
		for _, flag := range flags {
			a.flags |= flag
		}
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
