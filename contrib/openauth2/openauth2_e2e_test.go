package openauth2_test

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/contrib/auth/users"
	"github.com/Nigel2392/go-django/contrib/openauth2"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/models"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-signals"
	"github.com/Nigel2392/mux/v2"
	"github.com/Nigel2392/mux/v2/middleware/authentication"
	"golang.org/x/oauth2"
)

type handlerTransport struct {
	handler http.Handler
}

func (t handlerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	if clone.URL.Path == "" {
		clone.URL.Path = "/"
	}
	clone.RequestURI = clone.URL.RequestURI()
	if clone.Host == "" {
		clone.Host = clone.URL.Host
	}
	if clone.RemoteAddr == "" {
		clone.RemoteAddr = "127.0.0.1:12345"
	}
	rec := httptest.NewRecorder()
	t.handler.ServeHTTP(rec, clone)
	resp := rec.Result()
	resp.Request = req
	return resp, nil
}

type mockProviderProfile struct {
	Sub         string `json:"sub"`
	Display     string `json:"display"`
	FailBind    bool   `json:"fail_bind,omitempty"`
	BoundToUser *openauth2.User
}

func (p *mockProviderProfile) BindUser(u *openauth2.User) error {
	if p.FailBind {
		return stderrors.New("bind failed")
	}
	p.BoundToUser = u
	return nil
}

type formDefinerProfile struct {
	Sub     string `json:"sub"`
	Display string `json:"display"`
}

func (p *formDefinerProfile) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, p, attrs.NewField(p, "Display", &attrs.Config{Label: "Display Name"}))
}

func (p *formDefinerProfile) UserFormFields(u *openauth2.User) []attrs.Field {
	return []attrs.Field{
		attrs.NewField(p, "Display", &attrs.Config{
			Label: "Display Name",
		}),
	}
}

type mockOAuthServerState struct {
	failToken       bool
	failUserInfo    bool
	malformedJSON   bool
	tokenCalls      int
	userInfoCalls   int
	lastAuthHeader  string
	issuedAccessTok string
	issuedRefresh   string
	sub             string
	display         string
}

func (s *mockOAuthServerState) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		s.tokenCalls++
		if s.failToken {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"token_error"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  s.issuedAccessTok,
			"refresh_token": s.issuedRefresh,
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	})
	m.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		s.userInfoCalls++
		s.lastAuthHeader = r.Header.Get("Authorization")
		if s.failUserInfo {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if s.malformedJSON {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{not-valid-json`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sub":     s.sub,
			"display": s.display,
		})
	})
	m.HandleFunc("/api/me", func(w http.ResponseWriter, r *http.Request) {
		s.lastAuthHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok:" + s.lastAuthHeader))
	})
	return m
}

func TestE2E_HTTP_OAuth2FullFlowAndEdgeCases(t *testing.T) {
	ctx := t.Context()

	origConfigs := append([]openauth2.AuthConfig(nil), openauth2.App.Config.AuthConfigurations...)
	origDefaultDisabled := openauth2.App.Config.UserDefaultIsDisabled
	origRedirectAfterLogin := openauth2.App.Config.RedirectAfterLogin
	origRedirectAfterLogout := openauth2.App.Config.RedirectAfterLogout
	t.Cleanup(func() {
		openauth2.App.Config.AuthConfigurations = origConfigs
		openauth2.App.Config.UserDefaultIsDisabled = origDefaultDisabled
		openauth2.App.Config.RedirectAfterLogin = origRedirectAfterLogin
		openauth2.App.Config.RedirectAfterLogout = origRedirectAfterLogout
		_ = openauth2.App.OnReady()
		_, _ = queries.GetQuerySet(&openauth2.User{}).Filter("ProviderName__in",
			"mock", "mock_disabled", "mock_nodetails", "mock_nilobj", "mock_iderr", "mock_emptyid",
		).Delete()
	})

	state := &mockOAuthServerState{
		issuedAccessTok: "mock-access-1",
		issuedRefresh:   "mock-refresh-1",
		sub:             "mock-sub-1",
		display:         "Mock User",
	}
	mockOAuthClient := &http.Client{Transport: handlerTransport{handler: state.Handler()}}

	idErrProvider := openauth2.AuthConfig{
		ProviderInfo: openauth2.ConfigInfo{Provider: "mock_iderr"},
		Oauth2: &oauth2.Config{
			ClientID:     "cid",
			ClientSecret: "csec",
			Endpoint: oauth2.Endpoint{
				AuthURL:  "http://mock-oauth.local/authorize",
				TokenURL: "http://mock-oauth.local/token",
			},
		},
		DataConfig: openauth2.DataConfig{
			DetailsURL: "http://mock-oauth.local/userinfo",
			Object:     &mockProviderProfile{},
			GetUniqueIdentifier: func(token *oauth2.Token, ds any) (string, error) {
				return "", stderrors.New("id extraction failed")
			},
		},
	}

	emptyIDProvider := openauth2.AuthConfig{
		ProviderInfo: openauth2.ConfigInfo{Provider: "mock_emptyid"},
		Oauth2: &oauth2.Config{
			ClientID:     "cid",
			ClientSecret: "csec",
			Endpoint: oauth2.Endpoint{
				AuthURL:  "http://mock-oauth.local/authorize",
				TokenURL: "http://mock-oauth.local/token",
			},
		},
		DataConfig: openauth2.DataConfig{
			DetailsURL: "http://mock-oauth.local/userinfo",
			Object:     &mockProviderProfile{},
			GetUniqueIdentifier: func(token *oauth2.Token, ds any) (string, error) {
				return "", nil
			},
		},
	}

	noDetailsProvider := openauth2.AuthConfig{
		ProviderInfo: openauth2.ConfigInfo{Provider: "mock_nodetails"},
		Oauth2: &oauth2.Config{
			ClientID:     "cid",
			ClientSecret: "csec",
			Endpoint: oauth2.Endpoint{
				AuthURL:  "http://mock-oauth.local/authorize",
				TokenURL: "http://mock-oauth.local/token",
			},
		},
		DataConfig: openauth2.DataConfig{
			DetailsURL: "",
			Object:     &mockProviderProfile{},
			GetUniqueIdentifier: func(token *oauth2.Token, ds any) (string, error) {
				return ds.(*mockProviderProfile).Sub, nil
			},
		},
	}

	nilObjProvider := openauth2.AuthConfig{
		ProviderInfo: openauth2.ConfigInfo{Provider: "mock_nilobj"},
		Oauth2: &oauth2.Config{
			ClientID:     "cid",
			ClientSecret: "csec",
			Endpoint: oauth2.Endpoint{
				AuthURL:  "http://mock-oauth.local/authorize",
				TokenURL: "http://mock-oauth.local/token",
			},
		},
		DataConfig: openauth2.DataConfig{
			DetailsURL: "http://mock-oauth.local/userinfo",
			Object:     nil,
			GetUniqueIdentifier: func(token *oauth2.Token, ds any) (string, error) {
				return "x", nil
			},
		},
	}

	mockProvider := openauth2.AuthConfig{
		ProviderInfo: openauth2.ConfigInfo{
			Provider:      "mock",
			ProviderLabel: "Mock Provider",
		},
		Oauth2: &oauth2.Config{
			ClientID:     "mock-client-id",
			ClientSecret: "mock-client-secret",
			Scopes:       []string{"read", "email"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "http://mock-oauth.local/authorize",
				TokenURL: "http://mock-oauth.local/token",
			},
		},
		AccessType:  oauth2.AccessTypeOnline,
		State:       "custom-state-xyz",
		ExtraParams: map[string]string{"prompt": "consent"},
		DataConfig: openauth2.DataConfig{
			DetailsURL: "http://mock-oauth.local/userinfo",
			Object:     &mockProviderProfile{},
			GetUniqueIdentifier: func(token *oauth2.Token, ds any) (string, error) {
				return ds.(*mockProviderProfile).Sub, nil
			},
		},
		UserToString: func(u *openauth2.User, ds any) string {
			return "MOCK[" + ds.(*mockProviderProfile).Display + "]"
		},
	}

	disabledProvider := mockProvider
	disabledProvider.ProviderInfo.Provider = "mock_disabled"
	disabledProvider.Oauth2 = &oauth2.Config{
		ClientID:     "mock-client-id",
		ClientSecret: "mock-client-secret",
		Endpoint:     mockProvider.Oauth2.Endpoint,
	}

	openauth2.App.Config.AuthConfigurations = append(openauth2.App.Config.AuthConfigurations,
		mockProvider, disabledProvider, noDetailsProvider, nilObjProvider, idErrProvider, emptyIDProvider,
	)
	if err := openauth2.App.OnReady(); err != nil {
		t.Fatalf("OnReady: %v", err)
	}

	django.Global.Mux.Handle(mux.GET, "/e2e-oauth2/whoami", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := openauth2.UserFromRequest(r)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":            u.ID,
			"identifier":    u.UniqueIdentifier,
			"provider":      u.ProviderName,
			"authenticated": u.IsAuthenticated(),
			"active":        u.IsActive,
		})
	}))

	srv, err := django.Global.TestServe(false)
	if err != nil {
		t.Fatalf("TestServe: %v", err)
	}
	defer srv.Close()

	wrappedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rWithHTTP := r.WithContext(context.WithValue(r.Context(), oauth2.HTTPClient, mockOAuthClient))
		srv.Config.Handler.ServeHTTP(w, rWithHTTP)
	})

	newClient := func() *http.Client {
		jar, _ := cookiejar.New(nil)
		return &http.Client{
			Jar:       jar,
			Transport: handlerTransport{handler: wrappedHandler},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}

	baseURL := "https://test.local"

	t.Run("UnknownProviderAndLoginTemplate", func(t *testing.T) {
		c := newClient()

		respUnknown, err := c.Get(baseURL + "/auth2/does_not_exist")
		if err != nil {
			t.Fatalf("GET unknown provider: %v", err)
		}
		respUnknown.Body.Close()
		if respUnknown.StatusCode != http.StatusBadRequest {
			t.Errorf("unknown provider status = %d, want 400", respUnknown.StatusCode)
		}

		respLogin, err := c.Get(baseURL + "/auth2/login")
		if err != nil {
			t.Fatalf("GET /auth2/login: %v", err)
		}
		b, _ := io.ReadAll(respLogin.Body)
		respLogin.Body.Close()
		if respLogin.StatusCode != http.StatusOK || !strings.Contains(string(b), "oauth2-login") {
			t.Errorf("GET /auth2/login = (%d, %q), want 200 with oauth2-login", respLogin.StatusCode, string(b))
		}
	})

	t.Run("CallbackFailureEdgeCases", func(t *testing.T) {
		c := newClient()

		// 1. Missing code
		respNoCode, err := c.Get(baseURL + "/auth2/mock/callback")
		if err != nil {
			t.Fatalf("callback no code: %v", err)
		}
		respNoCode.Body.Close()
		if respNoCode.StatusCode != http.StatusBadRequest {
			t.Errorf("missing code status = %d, want 400", respNoCode.StatusCode)
		}

		// 2. Token exchange failure
		state.failToken = true
		respTokErr, err := c.Get(baseURL + "/auth2/mock/callback?code=c1")
		if err != nil {
			t.Fatalf("callback token err: %v", err)
		}
		respTokErr.Body.Close()
		if respTokErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("token exchange fail status = %d, want 500", respTokErr.StatusCode)
		}
		state.failToken = false

		// 3. Empty DetailsURL
		respNoDetails, err := c.Get(baseURL + "/auth2/mock_nodetails/callback?code=c1")
		if err != nil {
			t.Fatalf("callback no detailsURL: %v", err)
		}
		respNoDetails.Body.Close()
		if respNoDetails.StatusCode != http.StatusInternalServerError {
			t.Errorf("empty DetailsURL status = %d, want 500", respNoDetails.StatusCode)
		}

		// 4. UserInfo non-200
		state.failUserInfo = true
		respUIErr, err := c.Get(baseURL + "/auth2/mock/callback?code=c1")
		if err != nil {
			t.Fatalf("callback userinfo fail: %v", err)
		}
		respUIErr.Body.Close()
		if respUIErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("userinfo 500 status = %d, want 500", respUIErr.StatusCode)
		}
		state.failUserInfo = false

		// 5a. Malformed JSON from userinfo
		state.malformedJSON = true
		respMalformed, err := c.Get(baseURL + "/auth2/mock/callback?code=c1")
		if err != nil {
			t.Fatalf("callback malformed JSON: %v", err)
		}
		respMalformed.Body.Close()
		if respMalformed.StatusCode != http.StatusInternalServerError {
			t.Errorf("malformed JSON status = %d, want 500", respMalformed.StatusCode)
		}
		state.malformedJSON = false

		// 5b. DataConfig.Object == nil
		respNilObj, err := c.Get(baseURL + "/auth2/mock_nilobj/callback?code=c1")
		if err != nil {
			t.Fatalf("callback nil object: %v", err)
		}
		respNilObj.Body.Close()
		if respNilObj.StatusCode != http.StatusInternalServerError {
			t.Errorf("nil DataConfig.Object status = %d, want 500", respNilObj.StatusCode)
		}

		// 6. GetUniqueIdentifier returns error
		respIDErr, err := c.Get(baseURL + "/auth2/mock_iderr/callback?code=c1")
		if err != nil {
			t.Fatalf("callback id error: %v", err)
		}
		respIDErr.Body.Close()
		if respIDErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("GetUniqueIdentifier error status = %d, want 500", respIDErr.StatusCode)
		}

		// 7. GetUniqueIdentifier returns empty string
		respEmptyID, err := c.Get(baseURL + "/auth2/mock_emptyid/callback?code=c1")
		if err != nil {
			t.Fatalf("callback empty id: %v", err)
		}
		respEmptyID.Body.Close()
		if respEmptyID.StatusCode != http.StatusInternalServerError {
			t.Errorf("empty unique identifier status = %d, want 500", respEmptyID.StatusCode)
		}
	})

	t.Run("FullOAuth2Login_CookieRedirect_Relogin_AndLogout", func(t *testing.T) {
		c := newClient()

		var beforeCreate, afterCreate, loggedIn, loggedOut int
		r1, _ := core.SIGNAL_BEFORE_USER_CREATE.Listen(ctx, func(_ context.Context, _ signals.Signal[core.User], _ core.User) error {
			beforeCreate++
			return nil
		})
		r2, _ := core.SIGNAL_AFTER_USER_CREATE.Listen(ctx, func(_ context.Context, _ signals.Signal[core.User], _ core.User) error {
			afterCreate++
			return nil
		})
		r3, _ := core.SIGNAL_USER_LOGGED_IN.Listen(ctx, func(_ context.Context, _ signals.Signal[core.UserWithRequest], _ core.UserWithRequest) error {
			loggedIn++
			return nil
		})
		r4, _ := core.SIGNAL_USER_LOGGED_OUT.Listen(ctx, func(_ context.Context, _ signals.Signal[core.UserWithRequest], _ core.UserWithRequest) error {
			loggedOut++
			return nil
		})
		t.Cleanup(func() {
			_ = core.SIGNAL_BEFORE_USER_CREATE.Disconnect(ctx, r1)
			_ = core.SIGNAL_AFTER_USER_CREATE.Disconnect(ctx, r2)
			_ = core.SIGNAL_USER_LOGGED_IN.Disconnect(ctx, r3)
			_ = core.SIGNAL_USER_LOGGED_OUT.Disconnect(ctx, r4)
		})

		// 1. AuthHandler sets redirect cookie and redirects 302 to mock OAuth authorize URL
		respAuth, err := c.Get(baseURL + "/auth2/mock?next=/welcome/dashboard")
		if err != nil {
			t.Fatalf("GET /auth2/mock: %v", err)
		}
		respAuth.Body.Close()
		if respAuth.StatusCode != http.StatusFound {
			t.Fatalf("GET /auth2/mock status = %d, want 302", respAuth.StatusCode)
		}
		loc, err := url.Parse(respAuth.Header.Get("Location"))
		if err != nil {
			t.Fatalf("parse redirect Location: %v", err)
		}
		q := loc.Query()
		if q.Get("client_id") != "mock-client-id" || q.Get("state") != "custom-state-xyz" ||
			q.Get("access_type") != "online" || q.Get("prompt") != "consent" {
			t.Errorf("unexpected OAuth2 authorize params: %v", q)
		}
		var sawRedirectCookie bool
		for _, ck := range respAuth.Cookies() {
			if ck.Name == "openauth2.redirectURL" && ck.Value == "/welcome/dashboard" && ck.HttpOnly {
				sawRedirectCookie = true
			}
		}
		if !sawRedirectCookie {
			t.Errorf("expected openauth2.redirectURL cookie to be set, got %v", respAuth.Cookies())
		}

		// 2. Callback creates user, logs in, clears redirect cookie, redirects to /welcome/dashboard
		respCB, err := c.Get(baseURL + "/auth2/mock/callback?code=auth-code-123")
		if err != nil {
			t.Fatalf("GET /auth2/mock/callback: %v", err)
		}
		respCB.Body.Close()
		if respCB.StatusCode != http.StatusSeeOther || respCB.Header.Get("Location") != "/welcome/dashboard" {
			t.Fatalf("callback expected 303 -> /welcome/dashboard, got %d -> %q", respCB.StatusCode, respCB.Header.Get("Location"))
		}
		// Edge case: CallbackHandler uses QuerySet.Create without .ExplicitSave(), so Model.Save
		// and QuerySet.Create each invoke AfterCreate once while BeforeCreate runs once.
		if beforeCreate != 1 || afterCreate != 2 || loggedIn != 1 {
			t.Errorf("signals count mismatch: beforeCreate=%d afterCreate=%d loggedIn=%d", beforeCreate, afterCreate, loggedIn)
		}

		// Redirect cookie should be deleted (MaxAge < 0)
		var clearedCookie bool
		for _, ck := range respCB.Cookies() {
			if ck.Name == "openauth2.redirectURL" && ck.MaxAge < 0 {
				clearedCookie = true
			}
		}
		if !clearedCookie {
			t.Errorf("expected openauth2.redirectURL cookie to be cleared, got %v", respCB.Cookies())
		}

		// Verify DB persistence
		dbUser, err := openauth2.GetUserByIdentifier(ctx, "mock", "mock-sub-1")
		if err != nil {
			t.Fatalf("GetUserByIdentifier: %v", err)
		}
		if string(dbUser.AccessToken) != "mock-access-1" || string(dbUser.RefreshToken) != "mock-refresh-1" || !dbUser.IsActive {
			t.Errorf("unexpected persisted user: %+v", dbUser)
		}

		// Verify authenticated request via session cookie & AddUserMiddleware
		respWho, err := c.Get(baseURL + "/e2e-oauth2/whoami")
		if err != nil {
			t.Fatalf("GET whoami: %v", err)
		}
		var who map[string]any
		_ = json.NewDecoder(respWho.Body).Decode(&who)
		respWho.Body.Close()
		if who["authenticated"] != true || who["identifier"] != "mock-sub-1" {
			t.Errorf("expected authenticated mock-sub-1, got %#v", who)
		}

		// 3. AuthHandler when already authenticated immediately redirects 303 to ?next or "/"
		respAlready, err := c.Get(baseURL + "/auth2/mock?next=/already-done")
		if err != nil {
			t.Fatalf("GET /auth2/mock when authenticated: %v", err)
		}
		respAlready.Body.Close()
		if respAlready.StatusCode != http.StatusSeeOther || respAlready.Header.Get("Location") != "/already-done" {
			t.Errorf("authenticated AuthHandler expected 303 -> /already-done, got %d -> %q", respAlready.StatusCode, respAlready.Header.Get("Location"))
		}

		respAlreadyDefault, err := c.Get(baseURL + "/auth2/mock")
		if err != nil {
			t.Fatalf("GET /auth2/mock default redirect when authenticated: %v", err)
		}
		respAlreadyDefault.Body.Close()
		if respAlreadyDefault.StatusCode != http.StatusSeeOther || respAlreadyDefault.Header.Get("Location") != "/" {
			t.Errorf("authenticated AuthHandler default expected 303 -> '/', got %d -> %q", respAlreadyDefault.StatusCode, respAlreadyDefault.Header.Get("Location"))
		}

		// Edge case: setCallbackHandlerRedirect sets cookie without Path (default-path /auth2)
		// and deletes it on /auth2/mock/callback without Path (default-path /auth2/mock), so RFC 6265
		// cookiejar keeps the original /auth2 cookie active across subsequent callbacks in c.
		openauth2.App.Config.RedirectAfterLogin = func(u *openauth2.User, ds any, r *http.Request) string {
			return "/custom-after-login/" + u.UniqueIdentifier
		}
		respCBStale, err := c.Get(baseURL + "/auth2/mock/callback?code=auth-code-stale")
		if err != nil {
			t.Fatalf("stale-cookie callback: %v", err)
		}
		respCBStale.Body.Close()
		if respCBStale.StatusCode != http.StatusSeeOther || respCBStale.Header.Get("Location") != "/welcome/dashboard" {
			t.Errorf("stale-cookie callback expected 303 -> /welcome/dashboard, got %d -> %q", respCBStale.StatusCode, respCBStale.Header.Get("Location"))
		}

		// 4. Fresh client without stale redirect cookie uses RedirectAfterLogin (no duplicate user created)
		cFresh := newClient()
		respCB2, err := cFresh.Get(baseURL + "/auth2/mock/callback?code=auth-code-456")
		if err != nil {
			t.Fatalf("second callback: %v", err)
		}
		respCB2.Body.Close()
		if respCB2.StatusCode != http.StatusSeeOther || respCB2.Header.Get("Location") != "/custom-after-login/mock-sub-1" {
			t.Errorf("re-login callback expected 303 -> /custom-after-login/mock-sub-1, got %d -> %q", respCB2.StatusCode, respCB2.Header.Get("Location"))
		}
		if beforeCreate != 1 {
			t.Errorf("expected no new user creation on re-login, beforeCreate=%d", beforeCreate)
		}

		// When RedirectAfterLogin is nil, defaults to "/"
		openauth2.App.Config.RedirectAfterLogin = nil
		respCB3, err := cFresh.Get(baseURL + "/auth2/mock/callback?code=auth-code-789")
		if err != nil {
			t.Fatalf("third callback: %v", err)
		}
		respCB3.Body.Close()
		if respCB3.StatusCode != http.StatusSeeOther || respCB3.Header.Get("Location") != "/" {
			t.Errorf("callback default expected 303 -> '/', got %d -> %q", respCB3.StatusCode, respCB3.Header.Get("Location"))
		}

		// 5. LogoutHandler destroys session and uses RedirectAfterLogout / ?next
		openauth2.App.Config.RedirectAfterLogout = func(r *http.Request) string {
			return "/logged-out-home"
		}
		respLogout, err := c.Get(baseURL + "/auth2/logout")
		if err != nil {
			t.Fatalf("GET /auth2/logout: %v", err)
		}
		respLogout.Body.Close()
		if respLogout.StatusCode != http.StatusSeeOther || respLogout.Header.Get("Location") != "/logged-out-home" {
			t.Errorf("logout expected 303 -> /logged-out-home, got %d -> %q", respLogout.StatusCode, respLogout.Header.Get("Location"))
		}
		if loggedOut != 1 {
			t.Errorf("expected SIGNAL_USER_LOGGED_OUT count=1, got %d", loggedOut)
		}

		// Whoami after logout is unauthenticated
		respWhoAfter, err := c.Get(baseURL + "/e2e-oauth2/whoami")
		if err != nil {
			t.Fatalf("GET whoami after logout: %v", err)
		}
		var whoAfter map[string]any
		_ = json.NewDecoder(respWhoAfter.Body).Decode(&whoAfter)
		respWhoAfter.Body.Close()
		if whoAfter["authenticated"] != false {
			t.Errorf("expected unauthenticated after logout, got %#v", whoAfter)
		}

		// Logout when unauthenticated with ?next redirects cleanly
		respUnauthLogout, err := c.Get(baseURL + "/auth2/logout?next=/public")
		if err != nil {
			t.Fatalf("GET unauth logout: %v", err)
		}
		respUnauthLogout.Body.Close()
		if respUnauthLogout.StatusCode != http.StatusSeeOther || respUnauthLogout.Header.Get("Location") != "/public" {
			t.Errorf("unauth logout expected 303 -> /public, got %d -> %q", respUnauthLogout.StatusCode, respUnauthLogout.Header.Get("Location"))
		}
	})

	t.Run("InsecureHTTPWithholdsSecureRedirectCookie", func(t *testing.T) {
		openauth2.App.Config.RedirectAfterLogin = nil
		origSub, origDisplay := state.sub, state.display
		state.sub = "insecure-sub-1"
		state.display = "Insecure User"
		t.Cleanup(func() {
			state.sub = origSub
			state.display = origDisplay
		})

		c := newClient()
		insecureBaseURL := "http://test.local"
		respAuth, err := c.Get(insecureBaseURL + "/auth2/mock?next=/ignored")
		if err != nil {
			t.Fatalf("GET /auth2/mock over http: %v", err)
		}
		respAuth.Body.Close()
		var gotSecureCookie bool
		for _, ck := range respAuth.Cookies() {
			if ck.Name == "openauth2.redirectURL" && ck.Value == "/ignored" && ck.Secure {
				gotSecureCookie = true
			}
		}
		if !gotSecureCookie {
			t.Errorf("expected Secure openauth2.redirectURL cookie on /auth2/mock response, got %v", respAuth.Cookies())
		}

		// net/http/cookiejar withholds Secure cookies over plain http://, so callback falls back to "/"
		respCB, err := c.Get(insecureBaseURL + "/auth2/mock/callback?code=insecure-code")
		if err != nil {
			t.Fatalf("GET /auth2/mock/callback over http: %v", err)
		}
		respCB.Body.Close()
		if respCB.StatusCode != http.StatusSeeOther || respCB.Header.Get("Location") != "/" {
			t.Errorf("expected fallback redirect 303 -> '/' when Secure cookie withheld over http, got %d -> %q", respCB.StatusCode, respCB.Header.Get("Location"))
		}
	})

	t.Run("UserDefaultIsDisabled_CreatesInactiveUserFilteredByMiddleware", func(t *testing.T) {
		openauth2.App.Config.UserDefaultIsDisabled = true
		t.Cleanup(func() { openauth2.App.Config.UserDefaultIsDisabled = false })

		state.sub = "disabled-sub-1"
		c := newClient()

		respCB, err := c.Get(baseURL + "/auth2/mock_disabled/callback?code=code-disabled")
		if err != nil {
			t.Fatalf("callback disabled user: %v", err)
		}
		respCB.Body.Close()
		if respCB.StatusCode != http.StatusSeeOther {
			t.Fatalf("expected 303, got %d", respCB.StatusCode)
		}

		dbDisabled, err := openauth2.GetUserByIdentifier(ctx, "mock_disabled", "disabled-sub-1")
		if err != nil {
			t.Fatalf("GetUserByIdentifier disabled: %v", err)
		}
		if dbDisabled.IsActive {
			t.Errorf("expected UserDefaultIsDisabled=true to create inactive user")
		}

		// Middleware Filter("IsActive", true) treats inactive user in session as unauthenticated
		respWho, err := c.Get(baseURL + "/e2e-oauth2/whoami")
		if err != nil {
			t.Fatalf("GET whoami for disabled user: %v", err)
		}
		var who map[string]any
		_ = json.NewDecoder(respWho.Body).Decode(&who)
		respWho.Body.Close()
		if who["authenticated"] != false {
			t.Errorf("expected inactive user to be treated as unauthenticated by middleware, got %#v", who)
		}
	})
}

func TestModels_ContentObjectUserBinderFormDefinerAndSignals(t *testing.T) {
	ctx := t.Context()

	origConfigs := append([]openauth2.AuthConfig(nil), openauth2.App.Config.AuthConfigurations...)
	t.Cleanup(func() {
		openauth2.App.Config.AuthConfigurations = origConfigs
		_ = openauth2.App.OnReady()
		_, _ = queries.GetQuerySet(&openauth2.User{}).Filter("ProviderName", "binder_prov").Delete()
	})

	openauth2.App.Config.AuthConfigurations = append(openauth2.App.Config.AuthConfigurations,
		openauth2.AuthConfig{
			ProviderInfo: openauth2.ConfigInfo{Provider: "binder_prov"},
			Oauth2: &oauth2.Config{
				ClientID:     "cid",
				ClientSecret: "csec",
			},
			DataConfig: openauth2.DataConfig{
				Object: &mockProviderProfile{},
				GetUniqueIdentifier: func(token *oauth2.Token, ds any) (string, error) {
					return ds.(*mockProviderProfile).Sub, nil
				},
			},
		},
		openauth2.AuthConfig{
			ProviderInfo: openauth2.ConfigInfo{Provider: "form_def_prov"},
			Oauth2: &oauth2.Config{
				ClientID:     "cid",
				ClientSecret: "csec",
			},
			DataConfig: openauth2.DataConfig{
				Object: &formDefinerProfile{},
				GetUniqueIdentifier: func(token *oauth2.Token, ds any) (string, error) {
					return ds.(*formDefinerProfile).Sub, nil
				},
			},
		},
	)
	if err := openauth2.App.OnReady(); err != nil {
		t.Fatalf("OnReady: %v", err)
	}

	t.Run("UserBasicsAndContext", func(t *testing.T) {
		u := &openauth2.User{UniqueIdentifier: "id1", ProviderName: "test"}
		if u.String() != "id1 (test)" {
			t.Errorf("String() = %q, want 'id1 (test)'", u.String())
		}
		ut := u.UniqueTogether()
		if len(ut) != 1 || len(ut[0]) != 2 || ut[0][0] != "UniqueIdentifier" || ut[0][1] != "ProviderName" {
			t.Errorf("unexpected UniqueTogether: %#v", ut)
		}
		if u.Context() == nil {
			t.Errorf("expected non-nil default Context()")
		}
		type ctxKey struct{}
		customCtx := context.WithValue(ctx, ctxKey{}, "val")
		u.SetContext(customCtx)
		if u.Context().Value(ctxKey{}) != "val" {
			t.Errorf("SetContext did not persist custom context")
		}
		if _, err := u.Provider(); err != nil {
			t.Errorf("Provider() error: %v", err)
		}
	})

	t.Run("ContentObject_EdgeCasesAndUserBinder", func(t *testing.T) {
		// Empty data -> error
		uNoData := &openauth2.User{UniqueIdentifier: "x", ProviderName: "test"}
		if _, err := uNoData.ContentObject(); err == nil {
			t.Errorf("expected error when User.Data is empty")
		}

		// Unknown provider -> error
		uUnknownProv := &openauth2.User{UniqueIdentifier: "x", ProviderName: "nope", Data: []byte(`{"name":"a"}`)}
		if _, err := uUnknownProv.ContentObject(); !stderrors.Is(err, openauth2.ErrUnknownProvider) {
			t.Errorf("expected ErrUnknownProvider, got %v", err)
		}

		// Malformed JSON -> error
		uBadJSON := &openauth2.User{UniqueIdentifier: "x", ProviderName: "test", Data: []byte(`{bad`)}
		if _, err := uBadJSON.ContentObject(); err == nil {
			t.Errorf("expected scan error for malformed JSON")
		}

		// UserBinder binds user & caches result
		uBinder := &openauth2.User{
			UniqueIdentifier: "b1",
			ProviderName:     "binder_prov",
			Data:             []byte(`{"sub":"b1","display":"Bound User"}`),
		}
		obj1, err := uBinder.ContentObject()
		if err != nil {
			t.Fatalf("ContentObject(binder) error: %v", err)
		}
		prof := obj1.(*mockProviderProfile)
		if prof.BoundToUser != uBinder || prof.Display != "Bound User" {
			t.Errorf("expected BindUser to bind uBinder, got %+v", prof)
		}
		// Second call returns cached pointer
		obj2, err := uBinder.ContentObject()
		if err != nil || obj2 != obj1 {
			t.Errorf("expected cached ContentObject instance on second call")
		}

		// UserBinder returning error propagates
		uBindFail := &openauth2.User{
			UniqueIdentifier: "b2",
			ProviderName:     "binder_prov",
			Data:             []byte(`{"sub":"b2","fail_bind":true}`),
		}
		if _, err := uBindFail.ContentObject(); err == nil || !strings.Contains(err.Error(), "bind failed") {
			t.Errorf("expected bind failed error, got %v", err)
		}
	})

	t.Run("UserFormDefiner_FieldDefsAndPanicGuards", func(t *testing.T) {
		uForm := &openauth2.User{
			UniqueIdentifier: "uf1",
			ProviderName:     "form_def_prov",
			Data:             []byte(`{"sub":"uf1","display":"Custom Form Display"}`),
		}
		defs := attrs.Define(ctx, uForm)
		f, ok := defs.Field("Display")
		if !ok {
			t.Fatalf("expected custom 'Display' field from UserFormDefiner in FieldDefs")
		}
		if f.GetValue() != "Custom Form Display" {
			t.Errorf("Display field value = %v, want 'Custom Form Display'", f.GetValue())
		}

		// Unknown provider on FieldDefs with non-empty Data panics
		func() {
			defer func() {
				if rec := recover(); rec == nil {
					t.Errorf("expected panic when provider is unknown in FieldDefs")
				}
			}()
			uBad := &openauth2.User{UniqueIdentifier: "uf2", ProviderName: "unknown_prov", Data: []byte(`{}`)}
			_ = attrs.Define(ctx, uBad)
		}()

		// Invalid JSON on FieldDefs with UserFormDefiner panics
		func() {
			defer func() {
				if rec := recover(); rec == nil {
					t.Errorf("expected panic when ContentObject fails in FieldDefs")
				}
			}()
			uBadJSON := &openauth2.User{UniqueIdentifier: "uf3", ProviderName: "form_def_prov", Data: []byte(`{bad`)}
			_ = attrs.Define(ctx, uBadJSON)
		}()
	})

	t.Run("LifecycleSignals_AndQueries", func(t *testing.T) {
		var sigBeforeCreate, sigAfterCreate, sigBeforeUpdate, sigAfterUpdate, sigBeforeDelete, sigAfterDelete int
		r1, _ := core.SIGNAL_BEFORE_USER_CREATE.Listen(ctx, func(_ context.Context, _ signals.Signal[core.User], _ core.User) error { sigBeforeCreate++; return nil })
		r2, _ := core.SIGNAL_AFTER_USER_CREATE.Listen(ctx, func(_ context.Context, _ signals.Signal[core.User], _ core.User) error { sigAfterCreate++; return nil })
		r3, _ := core.SIGNAL_BEFORE_USER_UPDATE.Listen(ctx, func(_ context.Context, _ signals.Signal[core.User], _ core.User) error { sigBeforeUpdate++; return nil })
		r4, _ := core.SIGNAL_AFTER_USER_UPDATE.Listen(ctx, func(_ context.Context, _ signals.Signal[core.User], _ core.User) error { sigAfterUpdate++; return nil })
		r5, _ := core.SIGNAL_BEFORE_USER_DELETE.Listen(ctx, func(_ context.Context, _ signals.Signal[any], _ any) error { sigBeforeDelete++; return nil })
		r6, _ := core.SIGNAL_AFTER_USER_DELETE.Listen(ctx, func(_ context.Context, _ signals.Signal[any], _ any) error { sigAfterDelete++; return nil })
		t.Cleanup(func() {
			_ = core.SIGNAL_BEFORE_USER_CREATE.Disconnect(ctx, r1)
			_ = core.SIGNAL_AFTER_USER_CREATE.Disconnect(ctx, r2)
			_ = core.SIGNAL_BEFORE_USER_UPDATE.Disconnect(ctx, r3)
			_ = core.SIGNAL_AFTER_USER_UPDATE.Disconnect(ctx, r4)
			_ = core.SIGNAL_BEFORE_USER_DELETE.Disconnect(ctx, r5)
			_ = core.SIGNAL_AFTER_USER_DELETE.Disconnect(ctx, r6)
		})

		u := models.Setup(ctx, &openauth2.User{
			UniqueIdentifier: "lifecycle_1",
			ProviderName:     "binder_prov",
			Data:             []byte(`{"sub":"lifecycle_1"}`),
			Base:             users.Base{IsActive: true},
		})
		created, err := openauth2.CreateUser(ctx, u)
		if err != nil {
			t.Fatalf("CreateUser: %v", err)
		}
		if time.Time(created.CreatedAt).IsZero() || time.Time(created.UpdatedAt).IsZero() {
			t.Errorf("expected CreatedAt and UpdatedAt to be populated")
		}
		// Edge case: openauth2.CreateUser uses QuerySet.Create without .ExplicitSave(),
		// so SIGNAL_BEFORE_USER_CREATE fires once and SIGNAL_AFTER_USER_CREATE fires twice.
		if sigBeforeCreate != 1 || sigAfterCreate != 2 {
			t.Errorf("create signals = (%d, %d), want (1, 2)", sigBeforeCreate, sigAfterCreate)
		}

		// Update
		created.AccessToken = "updated-tok"
		if err := created.Save(ctx); err != nil {
			t.Fatalf("Save update: %v", err)
		}
		if sigBeforeUpdate != 1 || sigAfterUpdate != 1 {
			t.Errorf("update signals = (%d, %d), want (1, 1)", sigBeforeUpdate, sigAfterUpdate)
		}

		// Queries
		byID, err := openauth2.GetUserByID(ctx, created.ID)
		if err != nil || byID.UniqueIdentifier != "lifecycle_1" {
			t.Errorf("GetUserByID = (%+v, %v)", byID, err)
		}
		if _, err := openauth2.GetUserByID(ctx, 9999999); !stderrors.Is(err, errors.NoRows) {
			t.Errorf("GetUserByID(missing) = %v, want errors.NoRows", err)
		}
		if _, err := openauth2.GetUserByIdentifier(ctx, "binder_prov", "nope"); !stderrors.Is(err, errors.NoRows) {
			t.Errorf("GetUserByIdentifier(missing) = %v, want errors.NoRows", err)
		}

		// Delete
		if _, err := queries.DeleteObject(created); err != nil {
			t.Fatalf("DeleteObject: %v", err)
		}
		if sigBeforeDelete != 1 || sigAfterDelete != 1 {
			t.Errorf("delete signals = (%d, %d), want (1, 1)", sigBeforeDelete, sigAfterDelete)
		}
	})
}

func TestClientAndRefreshTokens_EdgeCases(t *testing.T) {
	ctx := t.Context()

	if _, err := openauth2.RefreshTokens(nil); !stderrors.Is(err, openauth2.ErrUserNil) {
		t.Errorf("RefreshTokens(nil) = %v, want ErrUserNil", err)
	}
	if _, err := openauth2.Client(nil); !stderrors.Is(err, openauth2.ErrUserNil) {
		t.Errorf("Client(nil) = %v, want ErrUserNil", err)
	}

	t.Run("TokenSource_UnknownProviderPanics", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec == nil {
				t.Errorf("expected panic for unknown provider in TokenSource")
			}
		}()
		_ = openauth2.TokenSource(&openauth2.User{ProviderName: "unknown_provider"})
	})

	t.Run("AuthConfig_ScanContentObject_NilObjectErrors", func(t *testing.T) {
		cfg := &openauth2.AuthConfig{}
		if _, err := cfg.ScanContentObject(strings.NewReader(`{}`)); err == nil {
			t.Errorf("expected error when DataConfig.Object is nil")
		}
	})

	t.Run("Client_AutoRefreshesExpiredTokenAndPersists", func(t *testing.T) {
		origConfigs := append([]openauth2.AuthConfig(nil), openauth2.App.Config.AuthConfigurations...)
		t.Cleanup(func() {
			openauth2.App.Config.AuthConfigurations = origConfigs
			_ = openauth2.App.OnReady()
			_, _ = queries.GetQuerySet(&openauth2.User{}).Filter("ProviderName", "client_prov").Delete()
		})

		state := &mockOAuthServerState{
			issuedAccessTok: "refreshed-access-xyz",
			issuedRefresh:   "refreshed-refresh-xyz",
		}
		mockClient := &http.Client{Transport: handlerTransport{handler: state.Handler()}}

		openauth2.App.Config.AuthConfigurations = append(openauth2.App.Config.AuthConfigurations, openauth2.AuthConfig{
			ProviderInfo: openauth2.ConfigInfo{Provider: "client_prov"},
			Oauth2: &oauth2.Config{
				ClientID:     "cid",
				ClientSecret: "csec",
				Endpoint: oauth2.Endpoint{
					AuthURL:  "http://mock-oauth.local/authorize",
					TokenURL: "http://mock-oauth.local/token",
				},
			},
		})
		if err := openauth2.App.OnReady(); err != nil {
			t.Fatalf("OnReady: %v", err)
		}

		userCtx := context.WithValue(ctx, oauth2.HTTPClient, mockClient)
		u := models.Setup(userCtx, &openauth2.User{
			UniqueIdentifier: "client_user_1",
			ProviderName:     "client_prov",
			Data:             []byte(`{}`),
			AccessToken:      "expired-access",
			RefreshToken:     "valid-refresh",
			TokenType:        "Bearer",
			ExpiresAt:        drivers.Timestamp(time.Now().Add(-time.Hour)),
			Base:             users.Base{IsActive: true},
		})
		created, err := openauth2.CreateUser(userCtx, u)
		if err != nil {
			t.Fatalf("CreateUser: %v", err)
		}
		created.SetContext(userCtx)

		httpClient, err := openauth2.Client(created)
		if err != nil {
			t.Fatalf("openauth2.Client: %v", err)
		}

		resp, err := httpClient.Get("http://mock-oauth.local/api/me")
		if err != nil {
			t.Fatalf("httpClient.Get: %v", err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if string(body) != "ok:Bearer refreshed-access-xyz" {
			t.Errorf("API response = %q, want 'ok:Bearer refreshed-access-xyz'", string(body))
		}
		if state.tokenCalls != 1 {
			t.Errorf("expected 1 token refresh call, got %d", state.tokenCalls)
		}

		reloaded, err := openauth2.GetUserByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetUserByID: %v", err)
		}
		if string(reloaded.AccessToken) != "refreshed-access-xyz" {
			t.Errorf("expected refreshed token persisted in DB, got %q", reloaded.AccessToken)
		}
	})
}

var _ = authentication.Retrieve
