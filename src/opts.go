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
	// Immediately set up the logger, just in case
	// someone needs to log anything before calling [App]
	logger.Setup(w)

	return func(a *Application) error {
		a.Log = w
		return nil
	}
}

func Configure(m map[string]interface{}) func(*Application) error {
	return func(a *Application) error {
		var s = Config(m)
		a.Settings = s
		return s.Bind(a)
	}
}
