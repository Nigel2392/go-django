package autherrors_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	autherrors "github.com/Nigel2392/go-django/contrib/auth/auth_errors"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/goldcrest"
	"github.com/Nigel2392/mux"
)

func TestAuthenticationError_AndFail(t *testing.T) {
	t.Run("DefaultStatusCode401", func(t *testing.T) {
		defer func() {
			rec := recover()
			if rec == nil {
				t.Fatalf("expected panic from Fail")
			}
			authErr, ok := rec.(*autherrors.AuthenticationError)
			if !ok {
				t.Fatalf("expected *AuthenticationError, got %T", rec)
			}
			if authErr.StatusCode() != http.StatusUnauthorized {
				t.Errorf("StatusCode() = %d, want %d", authErr.StatusCode(), http.StatusUnauthorized)
			}
			if authErr.Error() != "login required" || authErr.UserMessage() != "login required" {
				t.Errorf("unexpected error/message: %q / %q", authErr.Error(), authErr.UserMessage())
			}
			if authErr.NextURL != "/next" {
				t.Errorf("NextURL = %q, want /next", authErr.NextURL)
			}
		}()
		autherrors.Fail(0, "login required", "/next")
	})

	t.Run("ExplicitStatusCodeAndNoNext", func(t *testing.T) {
		defer func() {
			rec := recover()
			authErr := rec.(*autherrors.AuthenticationError)
			if authErr.StatusCode() != http.StatusForbidden {
				t.Errorf("StatusCode() = %d, want %d", authErr.StatusCode(), http.StatusForbidden)
			}
			if authErr.NextURL != "" {
				t.Errorf("NextURL = %q, want empty", authErr.NextURL)
			}
		}()
		autherrors.Fail(http.StatusForbidden, "forbidden")
	})

	t.Run("InvalidStatusCodePanics", func(t *testing.T) {
		for _, code := range []int{200, 302, 600, -1} {
			func() {
				defer func() {
					if rec := recover(); rec == nil {
						t.Errorf("expected assertion panic for invalid code %d", code)
					}
				}()
				autherrors.Fail(code, "bad status")
			}()
		}
	})

	t.Run("EmptyMessagePanics", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec == nil {
				t.Errorf("expected assertion panic for empty message")
			}
		}()
		autherrors.Fail(http.StatusUnauthorized, "")
	})
}

func TestRegisterHook_AndOnAuthenticationError(t *testing.T) {
	t.Cleanup(func() {
		goldcrest.Unregister(django.HOOK_SERVER_ERROR)
		goldcrest.Unregister(autherrors.AUTH_ERROR_HOOK)
	})

	m := mux.New()
	g := m.Any("/auth", nil, "test")
	g.Handle(mux.GET, "/login", mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {}), "login")
	app := &django.Application{Mux: m}

	autherrors.RegisterHook("test:login")
	hooks := goldcrest.Get[django.ServerErrorHook](django.HOOK_SERVER_ERROR)
	if len(hooks) == 0 {
		t.Fatalf("expected server error hook to be registered")
	}

	t.Run("IgnoresNonAuthServerError", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/any", nil)
		hooks[0](w, r, app, except.NewServerError(http.StatusNotFound, "not found"))
		if w.Code != http.StatusOK || w.Header().Get("Location") != "" {
			t.Errorf("expected non-auth server error to be ignored, got code=%d loc=%q", w.Code, w.Header().Get("Location"))
		}
	})

	t.Run("RedirectsWithNextURL", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/secret", nil)
		hooks[0](w, r, app, &autherrors.AuthenticationError{
			Status:  http.StatusUnauthorized,
			Message: "unauth",
			NextURL: "/secret?tab=1",
		})
		if w.Code != http.StatusFound {
			t.Fatalf("expected 302 Found, got %d", w.Code)
		}
		if loc := w.Header().Get("Location"); loc != "/auth/login/?next=%2Fsecret%3Ftab%3D1" {
			t.Errorf("Location = %q, want /auth/login/?next=%%2Fsecret%%3Ftab%%3D1", loc)
		}
	})

	t.Run("CustomAuthErrorHookShortCircuitsWhenWrittenTrue", func(t *testing.T) {
		goldcrest.Unregister(autherrors.AUTH_ERROR_HOOK)
		t.Cleanup(func() { goldcrest.Unregister(autherrors.AUTH_ERROR_HOOK) })

		autherrors.OnAuthenticationError(func(w http.ResponseWriter, r *http.Request, app *django.Application, err *autherrors.AuthenticationError) bool {
			return false
		})
		autherrors.OnAuthenticationError(func(w http.ResponseWriter, r *http.Request, app *django.Application, err *autherrors.AuthenticationError) bool {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("custom-unauth"))
			return true
		})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/secret", nil)
		hooks[0](w, r, app, &autherrors.AuthenticationError{
			Status:  http.StatusUnauthorized,
			Message: "unauth",
			NextURL: "/api/secret",
		})
		if w.Code != http.StatusUnauthorized || w.Body.String() != "custom-unauth" {
			t.Errorf("expected custom 401 response, got code=%d body=%q", w.Code, w.Body.String())
		}
	})
}
