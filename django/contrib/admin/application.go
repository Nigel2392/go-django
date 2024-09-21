package admin

import (
	"net/http"
	"regexp"
	"sync/atomic"

	"github.com/Nigel2392/django/apps"
	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/logger"
	"github.com/Nigel2392/django/forms"
	"github.com/Nigel2392/mux"
	"github.com/elliotchance/orderedmap/v2"
)

var nameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

type AdminApplication struct {
	*apps.AppConfig
	ready atomic.Bool

	// Ordering is the order in which the apps are displayed
	// in the admin interface.
	Ordering []string

	Route *mux.Route

	// Apps is a map of all the apps that are registered
	// with the admin site.
	Apps *orderedmap.OrderedMap[
		string, *AppDefinition,
	]

	getAdminLoginForm func(r *http.Request, formOpts ...func(forms.Form)) LoginForm
	logoutFunc        func(r *http.Request) error
}

type AuthConfig struct {
	GetLoginForm func(r *http.Request, formOpts ...func(forms.Form)) LoginForm
	Logout       func(r *http.Request) error
}

func (a *AdminApplication) IsReady() bool {
	return a.ready.Load()
}

func (a *AdminApplication) AuthLogout(r *http.Request) error {
	return a.logoutFunc(r)
}

func (a *AdminApplication) AuthLoginForm(r *http.Request, formOpts ...func(forms.Form)) LoginForm {
	return a.getAdminLoginForm(r, formOpts...)
}

func (a *AdminApplication) configureAuth(config AuthConfig) {
	if a.getAdminLoginForm != nil {
		logger.Warn(
			"AdminApplication.configureAuth: getAdminLoginForm was already set",
		)
	}

	if a.logoutFunc != nil {
		logger.Warn(
			"AdminApplication.configureAuth: logoutFunc was already set",
		)
	}

	a.getAdminLoginForm = config.GetLoginForm
	a.logoutFunc = config.Logout
}

func (a *AdminApplication) RegisterApp(name string, appOptions AppOptions, opts ...ModelOptions) *AppDefinition {

	assert.False(
		a.IsReady(),
		"AdminApplication is already initialized",
	)

	assert.True(
		nameRegex.MatchString(name),
		"App name must match regex %v",
		nameRegex,
	)

	var app = &AppDefinition{
		Name:    name,
		Options: appOptions,
		Models: orderedmap.NewOrderedMap[
			string, *ModelDefinition,
		](),
	}

	for _, opt := range opts {
		app.Register(opt)
	}

	a.Apps.Set(name, app)

	return app
}
