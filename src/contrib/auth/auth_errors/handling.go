package autherrors

import (
	"net/http"
	"net/url"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/goldcrest"
)

func RegisterHook() {
	goldcrest.Register(
		django.HOOK_SERVER_ERROR, 0,
		isAuthErrorHook,
	)
}

var _ except.ServerError = (*AuthenticationError)(nil)

type AuthenticationError struct {
	Message string
	NextURL string
	Status  int
}

// isAuthErrorHook is a hook that will redirect the user to the login page if an authentication error occurs.
//
// Authentication errors can be raised using auth.Fail(...)
//
// Under the hood this induced a panic; which is then caught by django and allows for more advanced error handling.
//
// This makes sure boilerplate code for failing auth is not repeated.
//
// It also allows for a more consistent way to handle auth errors.
func isAuthErrorHook(w http.ResponseWriter, r *http.Request, app *django.Application, serverError except.ServerError) {
	var (
		authError *AuthenticationError
		ok        bool
	)

	if authError, ok = serverError.(*AuthenticationError); !ok {
		return
	}

	var hooks = goldcrest.Get[AuthErrorHook](AUTH_ERROR_HOOK)
	for _, hook := range hooks {
		if hook(w, r, app, authError) {
			return
		}
	}

	var redirectURL, err = app.Mux.Reverse("auth:login")
	if err != nil {
		return
	}

	if authError.NextURL != "" {
		var u, err = url.Parse(redirectURL)
		if err != nil {
			goto respond
		}

		q := u.Query()
		q.Set("next", authError.NextURL)
		u.RawQuery = q.Encode()
		redirectURL = u.String()
	}

respond:
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// Authentication errors can be raised using autherrors.Fail(...)
//
// This makes sure boilerplate code for failing auth is not repeated.
//
// It also allows for a more consistent way to handle auth errors.
//
// We have a hook setup to catch any authentication errors and redirect to the login page (see hooks.go)
func Fail(code int, msg string, next ...string) {

	assert.True(
		code == 0 || code >= 400 && code < 600,
		"Invalid status code %d, must be between 400 and 599", code,
	)

	assert.True(
		msg != "",
		"Message must not be empty",
	)

	if code == 0 {
		code = 401
	}

	var nextURL string
	if len(next) > 0 {
		nextURL = next[0]
	}

	// Hidden control flow is resumed in isAuthErrorHook
	panic(&AuthenticationError{
		Message: msg,
		Status:  code,
		NextURL: nextURL,
	})
}

func (e *AuthenticationError) Error() string {
	return e.Message
}

func (e *AuthenticationError) StatusCode() int {
	return e.Status
}

func (e *AuthenticationError) UserMessage() string {
	return e.Message
}
