package admin

import (
	"net/http"
	"regexp"
	"sync/atomic"

	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/forms"
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

	auth *AuthConfig
}

type AuthConfig struct {
	// GetLoginForm is a function that returns a LoginForm
	//
	// This function is called when the user tries to login to the admin
	// interface. It should return a LoginForm that is used to render
	// the login form.
	//
	// If GetLoginHandler is set, this function will not be called.
	GetLoginForm func(r *http.Request, formOpts ...func(forms.Form)) LoginForm

	// GetLoginHandler is a function that returns a http.HandlerFunc for
	// logging a user in to the admin interface.
	//
	// If GetLoginHandler is set, this function will be called instead
	// of GetLoginForm. It should return a http.HandlerFunc that is
	// used to render the login form.
	GetLoginHandler func(w http.ResponseWriter, r *http.Request)

	// Logout is a function that is called when the user logs out of
	// the admin interface.
	Logout func(r *http.Request) error
}

func (a *AdminApplication) IsReady() bool {
	return a.ready.Load()
}

func (a *AdminApplication) AuthLogout(r *http.Request) error {
	return a.auth.Logout(r)
}

func (a *AdminApplication) AuthLoginForm(r *http.Request, formOpts ...func(forms.Form)) LoginForm {
	return a.auth.GetLoginForm(r, formOpts...)
}

func (a *AdminApplication) AuthLoginHandler() func(w http.ResponseWriter, r *http.Request) {
	return a.auth.GetLoginHandler
}

func (a *AdminApplication) configureAuth(config AuthConfig) {
	if a.auth != nil && a.auth.GetLoginForm != nil {
		logger.Warn(
			"AdminApplication.configureAuth: GetLoginForm was already set",
		)
	}

	if a.auth != nil && a.auth.GetLoginHandler != nil {
		logger.Warn(
			"AdminApplication.configureAuth: GetLoginHandler was already set",
		)
	}

	if a.auth != nil && a.auth.Logout != nil {
		logger.Warn(
			"AdminApplication.configureAuth: Logout was already set",
		)
	}

	if config.GetLoginForm != nil && config.GetLoginHandler != nil {
		logger.Warn(
			"AdminApplication.configureAuth: GetLoginForm and GetLoginHandler were both set in config, only the handler will be used",
		)
	}

	a.auth = &config
}

func (a *AdminApplication) RegisterApp(name string, appOptions AppOptions, opts ...ModelOptions) *AppDefinition {

	assert.False(
		a.IsReady(),
		"AdminApplication is already initialized",
	)

	var app = &AppDefinition{
		Name:    name,
		Options: appOptions,
		Models: orderedmap.NewOrderedMap[
			string, *ModelDefinition,
		](),
		modelsByName: make(map[string]*ModelDefinition),
	}

	for _, opt := range opts {
		app.Register(opt)
	}

	a.Apps.Set(name, app)

	return app
}
