package autherrors

import (
	"net/http"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/goldcrest"
)

type AuthErrorHook func(w http.ResponseWriter, r *http.Request, app *django.Application, err *AuthenticationError) (written bool)

const AUTH_ERROR_HOOK = "auth.errors.AuthenticationError"

func OnAuthenticationError(h AuthErrorHook) {
	goldcrest.Register(AUTH_ERROR_HOOK, 0, h)
}
