//go:build !testing_auth
// +build !testing_auth

package auth_test

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/contrib/auth"
	autherrors "github.com/Nigel2392/go-django/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/contrib/auth/users"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/models"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core"
	"github.com/Nigel2392/go-signals"
	"github.com/Nigel2392/mux/v2"
	"github.com/Nigel2392/mux/v2/middleware/sessions"
)

func TestAuthSessions_NilSessionGuards(t *testing.T) {
	t.Run("Login_NilSessionPanics", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec == nil {
				t.Errorf("expected panic when session is nil in Login")
			}
		}()
		_, _ = auth.Login(httptest.NewRequest(http.MethodPost, "/login", nil), &auth.User{ID: 1})
	})

	t.Run("Logout_NilSessionReturnsErrNoSession", func(t *testing.T) {
		err := auth.Logout(httptest.NewRequest(http.MethodPost, "/logout", nil))
		if !stderrors.Is(err, autherrors.ErrNoSession) {
			t.Errorf("Logout(nil session) = %v, want ErrNoSession", err)
		}
	})
}

func TestAuthSessions_FullSessionLifecycleAndFinalizer(t *testing.T) {
	ctx := t.Context()

	u := models.Setup(ctx, &auth.User{
		Email:     drivers.MustParseEmail("real_sess@example.com"),
		Username:  "real_sess",
		FirstName: "Real",
		LastName:  "Session",
		Password:  auth.NewPassword("ValidPass1!"),
		Base: users.Base{
			IsActive: true,
		},
	})
	created, err := auth.GetUserQuerySet().WithContext(ctx).Create(u)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = auth.GetUserQuerySet().Filter("ID", created.ID).Delete()
	})

	django.Global.Mux.Handle(mux.GET, "/e2e-sessions/whoami", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		curr := auth.UserFromRequest(r)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":            curr.ID,
			"username":      curr.Username,
			"authenticated": curr.IsAuthenticated(),
		})
	}))

	django.Global.Mux.Handle(mux.POST, "/e2e-sessions/save-before-end", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		row, err := auth.GetUserQuerySet().WithContext(r.Context()).Filter("ID", created.ID).First()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		loggedInUser, err := auth.Login(r, row.Object)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Saving explicitly clears state changed flag so session finalizer skips duplicate UPDATE
		if _, err := queries.GetQuerySet(&auth.User{}).
			WithContext(r.Context()).
			ExplicitSave().
			Select("LastLogin").
			Filter("ID", loggedInUser.ID).
			Update(loggedInUser); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	srv, err := django.Global.TestServe(false)
	if err != nil {
		t.Fatalf("TestServe: %v", err)
	}
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:       jar,
		Transport: handlerTransport{handler: srv.Config.Handler},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	var loggedInCount, loggedOutCount int
	rIn, _ := core.SIGNAL_USER_LOGGED_IN.Listen(ctx, func(_ context.Context, _ signals.Signal[core.UserWithRequest], _ core.UserWithRequest) error {
		loggedInCount++
		return nil
	})
	rOut, _ := core.SIGNAL_USER_LOGGED_OUT.Listen(ctx, func(_ context.Context, _ signals.Signal[core.UserWithRequest], _ core.UserWithRequest) error {
		loggedOutCount++
		return nil
	})
	t.Cleanup(func() {
		_ = core.SIGNAL_USER_LOGGED_IN.Disconnect(ctx, rIn)
		_ = core.SIGNAL_USER_LOGGED_OUT.Disconnect(ctx, rOut)
	})

	// 1. Real E2E Login via POST /auth/login persists session token & LastLogin via finalizer
	respLogin, err := client.PostForm("http://test.local/auth/login", map[string][]string{
		"username": {created.Username},
		"password": {"ValidPass1!"},
	})
	if err != nil {
		t.Fatalf("POST /auth/login: %v", err)
	}
	respLogin.Body.Close()
	if respLogin.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST /auth/login status = %d, want 303", respLogin.StatusCode)
	}
	if loggedInCount != 1 {
		t.Errorf("expected loggedInCount=1, got %d", loggedInCount)
	}

	// Verify LastLogin was persisted by session finalizer
	reloaded, err := auth.GetUserQuerySet().WithContext(ctx).Filter("ID", created.ID).First()
	if err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if reloaded.Object.LastLogin.IsZero() || time.Since(reloaded.Object.LastLogin) > time.Minute {
		t.Errorf("expected LastLogin to be persisted recently, got %v", reloaded.Object.LastLogin)
	}

	// 2. Subsequent GET /e2e-sessions/whoami uses session cookie to authenticate user automatically
	respWho, err := client.Get("http://test.local/e2e-sessions/whoami")
	if err != nil {
		t.Fatalf("GET whoami: %v", err)
	}
	var who map[string]any
	_ = json.NewDecoder(respWho.Body).Decode(&who)
	respWho.Body.Close()
	if who["username"] != created.Username || who["authenticated"] != true {
		t.Errorf("expected authenticated session user %q, got %#v", created.Username, who)
	}

	// 3. Logout destroys session and unauthenticates subsequent requests
	respLogout, err := client.Post("http://test.local/auth/logout?next=/out", "", nil)
	if err != nil {
		t.Fatalf("POST /auth/logout: %v", err)
	}
	respLogout.Body.Close()
	if respLogout.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST /auth/logout status = %d, want 303", respLogout.StatusCode)
	}
	if loggedOutCount != 1 {
		t.Errorf("expected loggedOutCount=1, got %d", loggedOutCount)
	}

	respAfterLogout, err := client.Get("http://test.local/e2e-sessions/whoami")
	if err != nil {
		t.Fatalf("GET whoami after logout: %v", err)
	}
	var whoAfter map[string]any
	_ = json.NewDecoder(respAfterLogout.Body).Decode(&whoAfter)
	respAfterLogout.Body.Close()
	if whoAfter["authenticated"] != false || whoAfter["username"] != "AnonymousUser" {
		t.Errorf("expected AnonymousUser after logout, got %#v", whoAfter)
	}

	// 4. Finalizer skips update when user already saved explicitly
	respSaveBefore, err := client.Post("http://test.local/e2e-sessions/save-before-end", "", nil)
	if err != nil {
		t.Fatalf("POST save-before-end: %v", err)
	}
	respSaveBefore.Body.Close()
	if respSaveBefore.StatusCode != http.StatusOK {
		t.Errorf("save-before-end status = %d, want 200", respSaveBefore.StatusCode)
	}

	// Verify session key was stored in session
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	sessions.SessionMiddleware(auth.Auth.Session)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := sessions.Retrieve(r)
		if _, err := auth.Login(r, reloaded.Object); err != nil {
			t.Fatalf("direct auth.Login: %v", err)
		}
		if got := sess.Get(auth.SESSION_COOKIE_NAME); got != created.ID {
			t.Errorf("session key = %#v, want %d", got, created.ID)
		}
	})).ServeHTTP(rec, req)
}
