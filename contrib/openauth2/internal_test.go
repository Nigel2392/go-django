package openauth2

import (
	"bytes"
	stderrors "errors"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	autherrors "github.com/Nigel2392/go-django/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/contrib/auth/users"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/mux/middleware/authentication"
	"github.com/Nigel2392/mux/middleware/sessions"
	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"
	"golang.org/x/oauth2"
)

type mockCmdMgr struct {
	inputs       []string
	idx          int
	inputErrOnce bool
	inputErrAt   int
	logs         bytes.Buffer
	cmd          command.Command
	reg          command.Registry
}

func (m *mockCmdMgr) Log(msg string)              { m.logs.WriteString(msg + "\n") }
func (m *mockCmdMgr) Logf(f string, args ...any)  {}
func (m *mockCmdMgr) Stdout() io.Writer           { return &m.logs }
func (m *mockCmdMgr) Stderr() io.Writer           { return &m.logs }
func (m *mockCmdMgr) Stdin() io.Reader            { return strings.NewReader("") }
func (m *mockCmdMgr) Registry() command.Registry  { return m.reg }
func (m *mockCmdMgr) Command() command.Command    { return m.cmd }
func (m *mockCmdMgr) ProtectedInput(string) (string, error) {
	return "", nil
}
func (m *mockCmdMgr) Input(q string) (string, error) {
	if m.inputErrOnce {
		m.inputErrOnce = false
		return "", stderrors.New("transient input error")
	}
	if m.inputErrAt > 0 && m.idx == m.inputErrAt {
		return "", stderrors.New("indexed input error")
	}
	if m.idx >= len(m.inputs) {
		panic("mockCmdMgr.Input exhausted")
	}
	v := m.inputs[m.idx]
	m.idx++
	return v, nil
}

type failingTokenSource struct {
	err error
}

func (f failingTokenSource) Token() (*oauth2.Token, error) {
	return nil, f.err
}

func newTestSessionManager() *scs.SessionManager {
	sm := scs.New()
	sm.Store = memstore.New()
	return sm
}

func TestBoundProviderInfo_AndProviderHelpers(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	// Full metadata
	full := &boundProviderInfo{
		r: req,
		cnf: ConfigInfo{
			Provider:         "github",
			ProviderLabel:    trans.S("GitHub"),
			ProviderLogoURL:  func(*http.Request) string { return "/static/github.svg" },
			DocumentationURL: func(*http.Request) string { return "https://docs.example.com" },
			PrivacyPolicyURL: func(*http.Request) string { return "https://privacy.example.com" },
		},
	}
	if full.Provider() != "github" {
		t.Errorf("Provider() = %q, want github", full.Provider())
	}
	if full.ProviderLabel() != "GitHub" {
		t.Errorf("ProviderLabel() = %q, want GitHub", full.ProviderLabel())
	}
	if full.ProviderLogoURL() != "/static/github.svg" {
		t.Errorf("ProviderLogoURL() = %q", full.ProviderLogoURL())
	}
	if full.DocumentationURL() != "https://docs.example.com" {
		t.Errorf("DocumentationURL() = %q", full.DocumentationURL())
	}
	if full.PrivacyPolicyURL() != "https://privacy.example.com" {
		t.Errorf("PrivacyPolicyURL() = %q", full.PrivacyPolicyURL())
	}

	// Nil optional metadata falls back safely
	minimal := &boundProviderInfo{
		r: req,
		cnf: ConfigInfo{
			Provider: "custom",
		},
	}
	if minimal.ProviderLabel() != "custom" {
		t.Errorf("ProviderLabel() fallback = %q, want custom", minimal.ProviderLabel())
	}
	if minimal.ProviderLogoURL() != "" || minimal.DocumentationURL() != "" || minimal.PrivacyPolicyURL() != "" {
		t.Errorf("expected empty URLs for nil callbacks")
	}

	// App provider helpers
	if _, err := App.Provider("does-not-exist"); !stderrors.Is(err, ErrUnknownProvider) {
		t.Errorf("App.Provider(unknown) = %v, want ErrUnknownProvider", err)
	}
	if len(App.Providers()) == 0 {
		t.Errorf("expected at least 1 configured provider")
	}

	App._cnfs["meta_test"] = AuthConfig{
		ProviderInfo: ConfigInfo{Provider: "meta_test"},
		DataConfig: DataConfig{
			EditDisplayOrder:  []string{"name", "email"},
			EditDisplayLabels: map[string]any{"name": "Full Name"},
			EditDisplayWidgets: map[string]widgets.Widget{
				"name": widgets.NewTextInput(nil),
			},
		},
	}
	t.Cleanup(func() { delete(App._cnfs, "meta_test") })

	if ord := App.getProviderDataFieldOrder("meta_test"); len(ord) != 2 || ord[0] != "name" {
		t.Errorf("unexpected field order: %v", ord)
	}
	if lbls := App.getProviderDataLabels("meta_test"); lbls["name"] != "Full Name" {
		t.Errorf("unexpected labels: %v", lbls)
	}
	if wdgs := App.getProviderDataWidgets("meta_test"); wdgs["name"] == nil {
		t.Errorf("expected widget for name")
	}

	if ord := App.getProviderDataFieldOrder("unknown"); len(ord) != 0 {
		t.Errorf("expected empty order for unknown provider, got %v", ord)
	}
	if lbls := App.getProviderDataLabels("unknown"); len(lbls) != 0 {
		t.Errorf("expected empty labels for unknown provider, got %v", lbls)
	}
	if wdgs := App.getProviderDataWidgets("unknown"); len(wdgs) != 0 {
		t.Errorf("expected empty widgets for unknown provider, got %v", wdgs)
	}

	// Default AccessType & RedirectURL populated on Ready
	prov, err := App.Provider("test")
	if err != nil {
		t.Fatalf("App.Provider(test): %v", err)
	}
	if prov.Oauth2.RedirectURL != "https://example.com/auth2/test/callback/" {
		t.Errorf("RedirectURL = %q, want https://example.com/auth2/test/callback/", prov.Oauth2.RedirectURL)
	}
}

func TestNewSavingTokenSource_EdgeCases(t *testing.T) {
	t.Run("NilUserPanics", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec == nil {
				t.Errorf("expected panic when creating savingTokenSource with nil user")
			}
		}()
		_ = newSavingTokenSource(failingTokenSource{}, nil)
	})

	t.Run("ValidTokenDoesNotCallUnderlyingSource", func(t *testing.T) {
		u := &User{
			UniqueIdentifier: "ts_valid",
			ProviderName:     "test",
			AccessToken:      "still-valid",
			ExpiresAt:        drivers.Timestamp(time.Now().Add(time.Hour)),
		}
		ts := newSavingTokenSource(failingTokenSource{err: stderrors.New("should not be called")}, u)
		tok, err := ts.Token()
		if err != nil {
			t.Fatalf("unexpected error when token valid: %v", err)
		}
		if tok.AccessToken != "still-valid" {
			t.Errorf("AccessToken = %q, want still-valid", tok.AccessToken)
		}
	})

	t.Run("RefreshErrorPropagatesWithoutSaving", func(t *testing.T) {
		refreshErr := stderrors.New("refresh failed")
		u := &User{
			UniqueIdentifier: "ts_fail",
			ProviderName:     "test",
			AccessToken:      "expired-tok",
			ExpiresAt:        drivers.Timestamp(time.Now().Add(-time.Hour)),
		}
		ts := newSavingTokenSource(failingTokenSource{err: refreshErr}, u)
		if _, err := ts.Token(); !stderrors.Is(err, refreshErr) {
			t.Errorf("Token() error = %v, want refreshErr", err)
		}
		if string(u.AccessToken) != "expired-tok" {
			t.Errorf("expected AccessToken not to be modified on refresh error")
		}
	})
}

func TestCommands_OpenAuth2CreateAndChangeUser(t *testing.T) {
	ctx := t.Context()
	t.Cleanup(func() {
		_, _ = queries.GetQuerySet(&User{}).Filter("UniqueIdentifier", "oa2_cmd_user").Delete()
	})

	t.Run("CreateUser_RetriesAndFlags", func(t *testing.T) {
		cmd := &command.Cmd[createUserStorage]{
			ID:       command_create_user.ID,
			Desc:     command_create_user.Desc,
			FlagFunc: command_create_user.FlagFunc,
			Execute:  command_create_user.Execute,
		}
		fs := flag.NewFlagSet("createuser", flag.ContinueOnError)
		m := &mockCmdMgr{
			inputErrOnce: true,
			// 1: transient err -> retries identifier
			// 2: identifier "oa2_cmd_user"
			// 3: unknown provider "unknown_prov" -> loops back (identifier already set!)
			// 4: valid provider "test"
			inputs: []string{"oa2_cmd_user", "unknown_prov", "test"},
			cmd:    cmd,
		}
		if err := cmd.AddFlags(m, fs); err != nil {
			t.Fatalf("AddFlags: %v", err)
		}
		if err := fs.Parse([]string{"-s", "-i"}); err != nil {
			t.Fatalf("fs.Parse: %v", err)
		}
		if err := cmd.Exec(m, fs.Args()); err != nil {
			t.Fatalf("createuser Exec: %v", err)
		}

		u, err := GetUserByIdentifier(ctx, "test", "oa2_cmd_user")
		if err != nil {
			t.Fatalf("GetUserByIdentifier: %v", err)
		}
		if !u.IsAdministrator || u.IsActive {
			t.Errorf("expected IsAdministrator=true, IsActive=false; got %+v", u.Base)
		}
	})

	t.Run("ChangeUser_SuccessAndErrors", func(t *testing.T) {
		// Error on first Input
		mErr1 := &mockCmdMgr{inputErrOnce: true}
		if err := command_change_user.Exec(mErr1, nil); err == nil {
			t.Errorf("expected error when first Input fails")
		}

		// Error on second Input
		mErr2 := &mockCmdMgr{inputs: []string{"oa2_cmd_user"}, inputErrAt: 1}
		if err := command_change_user.Exec(mErr2, nil); err == nil {
			t.Errorf("expected error when second Input fails")
		}

		// Missing user in DB
		mMissing := &mockCmdMgr{inputs: []string{"nonexistent_id", "test"}}
		if err := command_change_user.Exec(mMissing, nil); err == nil {
			t.Errorf("expected error when user does not exist")
		}

		// Valid update: super=false, inactive=false (IsActive=true)
		origChangeCmd := command_change_user.(*command.Cmd[changeUserStorage])
		cmd := &command.Cmd[changeUserStorage]{
			ID:       origChangeCmd.ID,
			Desc:     origChangeCmd.Desc,
			FlagFunc: origChangeCmd.FlagFunc,
			Execute:  origChangeCmd.Execute,
		}
		fs := flag.NewFlagSet("changeuser", flag.ContinueOnError)
		mValid := &mockCmdMgr{
			inputs: []string{"oa2_cmd_user", "test"},
			cmd:    cmd,
		}
		_ = cmd.AddFlags(mValid, fs)
		_ = fs.Parse([]string{})
		if err := cmd.Exec(mValid, fs.Args()); err != nil {
			t.Fatalf("changeuser Exec: %v", err)
		}

		u, err := GetUserByIdentifier(ctx, "test", "oa2_cmd_user")
		if err != nil {
			t.Fatalf("GetUserByIdentifier after change: %v", err)
		}
		if u.IsAdministrator || !u.IsActive {
			t.Errorf("expected IsAdministrator=false, IsActive=true; got %+v", u.Base)
		}
	})
}

func TestMiddleware_AndSessionsEdgeCases(t *testing.T) {
	ctx := t.Context()
	sm := newTestSessionManager()

	t.Run("NilSessionGuards", func(t *testing.T) {
		func() {
			defer func() {
				if rec := recover(); rec == nil {
					t.Errorf("expected panic from UserFromRequest without session")
				}
			}()
			_ = UserFromRequest(httptest.NewRequest(http.MethodGet, "/", nil))
		}()

		func() {
			defer func() {
				if rec := recover(); rec == nil {
					t.Errorf("expected panic from Login without session")
				}
			}()
			_, _ = Login(httptest.NewRequest(http.MethodPost, "/", nil), &User{ID: 1})
		}()

		if err := Logout(httptest.NewRequest(http.MethodPost, "/", nil)); !stderrors.Is(err, autherrors.ErrNoSession) {
			t.Errorf("Logout(no session) = %v, want ErrNoSession", err)
		}
	})

	activeUser, err := CreateUser(ctx, &User{
		UniqueIdentifier: "mw_active_oa2",
		ProviderName:     "test",
		Data:             []byte(`{"name":"Active"}`),
		Base:             users.Base{IsActive: true},
	})
	if err != nil {
		t.Fatalf("create activeUser: %v", err)
	}
	inactiveUser, err := CreateUser(ctx, &User{
		UniqueIdentifier: "mw_inactive_oa2",
		ProviderName:     "test",
		Data:             []byte(`{"name":"Inactive"}`),
		Base:             users.Base{IsActive: false},
	})
	if err != nil {
		t.Fatalf("create inactiveUser: %v", err)
	}
	t.Cleanup(func() {
		_, _ = queries.GetQuerySet(&User{}).Filter("ID__in", activeUser.ID, inactiveUser.ID).Delete()
	})

	t.Run("SessionStates_EmptyMissingInactiveActiveAndCached", func(t *testing.T) {
		runSess := func(fn func(r *http.Request)) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			sessions.SessionMiddleware(sm)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fn(r)
			})).ServeHTTP(rec, req)
		}

		// Empty session
		runSess(func(r *http.Request) {
			u := UserFromRequest(r)
			if u.IsAuthenticated() || u.IsLoggedIn {
				t.Errorf("expected unauthenticated user for empty session")
			}
		})

		// Missing user ID in DB
		runSess(func(r *http.Request) {
			sessions.Retrieve(r).Set(USER_ID_SESSION_KEY, uint64(88888888))
			u := UserFromRequest(r)
			if u.IsAuthenticated() {
				t.Errorf("expected unauthenticated user for non-existent user ID")
			}
		})

		// Inactive user in DB is filtered out by Filter("IsActive", true)
		runSess(func(r *http.Request) {
			sessions.Retrieve(r).Set(USER_ID_SESSION_KEY, inactiveUser.ID)
			u := UserFromRequest(r)
			if u.IsAuthenticated() {
				t.Errorf("expected inactive user in session to resolve as unauthenticated")
			}
		})

		// Active user in DB loads with IsLoggedIn=true
		runSess(func(r *http.Request) {
			sessions.Retrieve(r).Set(USER_ID_SESSION_KEY, activeUser.ID)
			u := UserFromRequest(r)
			if !u.IsAuthenticated() || !u.IsLoggedIn || u.ID != activeUser.ID {
				t.Errorf("expected active user ID %d, got %+v", activeUser.ID, u)
			}
		})

		// User already in authentication context is returned directly
		directReq := httptest.NewRequest(http.MethodGet, "/", nil)
		directReq = directReq.WithContext(authentication.ContextWithUser(directReq.Context(), activeUser))
		if got := UserFromRequest(directReq); got != activeUser {
			t.Errorf("expected UserFromRequest to return context-cached user")
		}
	})
}

func TestOpenAuth2AppConfig_Check_AndHandlerGuards(t *testing.T) {
	ctx := t.Context()

	t.Run("CheckValidationCases", func(t *testing.T) {
		runCheck := func(cfg Config) []string {
			tmp := &OpenAuth2AppConfig{
				DBRequiredAppConfig: App.DBRequiredAppConfig,
				Config:              &cfg,
			}
			var ids []string
			for _, m := range tmp.Check(ctx, django.Global.Settings) {
				if strings.HasPrefix(m.ID, "openauth2.") {
					ids = append(ids, m.ID)
				}
			}
			return ids
		}

		if ids := runCheck(Config{}); len(ids) != 1 || ids[0] != "openauth2.no_providers" {
			t.Errorf("empty providers got %v, want [openauth2.no_providers]", ids)
		}
		if ids := runCheck(Config{
			AuthConfigurations: []AuthConfig{{}},
		}); len(ids) != 1 || ids[0] != "openauth2.provider_missing" {
			t.Errorf("missing provider got %v, want [openauth2.provider_missing]", ids)
		}
		if ids := runCheck(Config{
			AuthConfigurations: []AuthConfig{{ProviderInfo: ConfigInfo{Provider: "p"}}},
		}); len(ids) != 1 || ids[0] != "openauth2.oauth2_missing" {
			t.Errorf("nil oauth2 got %v, want [openauth2.oauth2_missing]", ids)
		}
		if ids := runCheck(Config{
			AuthConfigurations: []AuthConfig{{
				ProviderInfo: ConfigInfo{Provider: "p"},
				Oauth2:       &oauth2.Config{},
			}},
		}); len(ids) != 1 || ids[0] != "openauth2.client_id_missing" {
			t.Errorf("empty client_id got %v, want [openauth2.client_id_missing]", ids)
		}
		if ids := runCheck(Config{
			AuthConfigurations: []AuthConfig{{
				ProviderInfo: ConfigInfo{Provider: "p"},
				Oauth2:       &oauth2.Config{ClientID: "cid"},
			}},
		}); len(ids) != 1 || ids[0] != "openauth2.client_secret_missing" {
			t.Errorf("empty client_secret got %v, want [openauth2.client_secret_missing]", ids)
		}
		if ids := runCheck(Config{
			AuthConfigurations: []AuthConfig{{
				ProviderInfo: ConfigInfo{Provider: "p"},
				Oauth2:       &oauth2.Config{ClientID: "cid", ClientSecret: "csec"},
			}},
		}); len(ids) != 0 {
			t.Errorf("valid config expected 0 openauth2 check errors, got %v", ids)
		}
	})

	t.Run("ProviderComponentMediaAndOrdering", func(t *testing.T) {
		compFn := newProviderComponent(App)
		comp := compFn(httptest.NewRequest(http.MethodGet, "/", nil), nil, nil)
		if comp.Ordering() != 0 {
			t.Errorf("Ordering() = %d, want 0", comp.Ordering())
		}
		if m := comp.Media(); m == nil {
			t.Errorf("expected non-nil Media")
		}
	})

	t.Run("HandlerMissingProviderPanics", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec == nil {
				t.Errorf("expected panic for missing provider in mux vars")
			}
		}()
		h := App.handler(func(w http.ResponseWriter, r *http.Request, ac *AuthConfig) {})
		h(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/auth2/", nil))
	})
}
