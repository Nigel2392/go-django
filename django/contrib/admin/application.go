package admin

import (
	"net/http"
	"regexp"
	"sync/atomic"

	"github.com/Nigel2392/django/apps"
	"github.com/Nigel2392/django/core/assert"
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

	GetLoginForm func(req *http.Request) LoginForm
	LogoutFunc   func(req *http.Request) error
}

func (a *AdminApplication) IsReady() bool {
	return a.ready.Load()
}

func (a *AdminApplication) LoginForm(req *http.Request) LoginForm {
	assert.True(
		a.GetLoginForm != nil,
		"GetLoginForm is not set on the global AdminSite struct variable",
	)

	return a.GetLoginForm(req)
}

func (a *AdminApplication) Logout(req *http.Request) error {
	assert.True(
		a.LogoutFunc != nil,
		"LogoutFunc is not set on the global AdminSite struct variable",
	)

	return a.LogoutFunc(req)
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
