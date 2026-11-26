package auth_test

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/contrib/auth"
	autherrors "github.com/Nigel2392/go-django/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/contrib/auth/users"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/securedb"
	"github.com/Nigel2392/go-django/queries/src/models"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/utils/djangoapps"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-signals"
	"github.com/Nigel2392/goldcrest"
	"github.com/Nigel2392/mux/v2"
	"github.com/Nigel2392/mux/v2/middleware/authentication"
	"github.com/Nigel2392/mux/v2/middleware/sessions"
)

func createTestUser(t *testing.T, ctx context.Context, email, username, rawPassword string, active, admin bool) *auth.User {
	t.Helper()
	u := models.Setup(ctx, &auth.User{
		Email:     drivers.MustParseEmail(email),
		Username:  username,
		FirstName: "First",
		LastName:  "Last",
		Password:  auth.NewPassword(rawPassword),
		Base: users.Base{
			IsAdministrator: admin,
			IsActive:        active,
		},
	})
	created, err := auth.GetUserQuerySet().WithContext(ctx).Create(u)
	if err != nil {
		t.Fatalf("createTestUser(%q): %v", username, err)
	}
	t.Cleanup(func() {
		_, _ = queries.GetQuerySet(&users.UserPermission{}).Filter("UserID", created.ID).Delete()
		_, _ = queries.GetQuerySet(&users.UserGroup{}).Filter("UserID", created.ID).Delete()
		_, _ = auth.GetUserQuerySet().Filter("ID", created.ID).Delete()
	})
	return created
}

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

func runWithSessionMiddleware(handler http.HandlerFunc) (*httptest.ResponseRecorder, *http.Request) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mw := sessions.SessionMiddleware(auth.Auth.Session)
	mw(handler).ServeHTTP(rec, req)
	return rec, req
}

func newSessionRequest(method, target string) *http.Request {
	rawReq := httptest.NewRequest(method, target, nil)
	out := rawReq
	mw := sessions.SessionMiddleware(auth.Auth.Session)
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out = r
	})).ServeHTTP(httptest.NewRecorder(), rawReq)
	return out
}

func TestPasswordAndField_EdgeCases(t *testing.T) {
	ctx := t.Context()

	t.Run("ValidatorEdgeCases", func(t *testing.T) {
		vAll := &auth.PasswordCharValidator{Flags: auth.ChrFlagAll}
		if err := vAll.Validate(""); !stderrors.Is(err, errs.ErrFieldRequired) {
			t.Errorf("Validate(\"\") = %v, want ErrFieldRequired", err)
		}

		// All-uppercase triggers upp_ct == len(password) check
		vUpper := &auth.PasswordCharValidator{Flags: auth.ChrFlagUpper}
		if err := vUpper.Validate("ALLUPPER"); !stderrors.Is(err, autherrors.ErrPwdCasingUpper) {
			t.Errorf("Validate(\"ALLUPPER\") = %v, want ErrPwdCasingUpper", err)
		}

		// All-lowercase triggers low_ct == len(password) check
		vLower := &auth.PasswordCharValidator{Flags: auth.ChrFlagLower}
		if err := vLower.Validate("alllower"); !stderrors.Is(err, autherrors.ErrPwdCasingLower) {
			t.Errorf("Validate(\"alllower\") = %v, want ErrPwdCasingLower", err)
		}

		// All-digits triggers dig_ct == len(password) check
		vDigit := &auth.PasswordCharValidator{Flags: auth.ChrFlagDigit}
		if err := vDigit.Validate("12345678"); !stderrors.Is(err, autherrors.ErrPwdDigits) {
			t.Errorf("Validate(\"12345678\") = %v, want ErrPwdDigits", err)
		}

		// GenericError masks granular errors (used when isRegister == false)
		vGeneric := &auth.PasswordCharValidator{
			Flags:        auth.ChrFlagAll,
			GenericError: autherrors.ErrGenericAuthFail,
		}
		if err := vGeneric.Validate("weak"); !stderrors.Is(err, autherrors.ErrGenericAuthFail) {
			t.Errorf("Validate with GenericError = %v, want ErrGenericAuthFail", err)
		}
	})

	t.Run("ValidateCharactersFieldAdapter", func(t *testing.T) {
		fLogin := fields.CharField(auth.ValidateCharacters(false, auth.ChrFlagAll))
		if errsList := fLogin.Validate(ctx, nil); len(errsList) != 0 {
			t.Errorf("nil value should pass ValidateCharacters, got %v", errsList)
		}
		if errsList := fLogin.Validate(ctx, ""); len(errsList) != 0 {
			t.Errorf("empty string should pass ValidateCharacters, got %v", errsList)
		}
		if errsList := fLogin.Validate(ctx, "weak"); !stderrors.Is(stderrors.Join(errsList...), autherrors.ErrGenericAuthFail) {
			t.Errorf("weak string with isRegister=false got %v, want ErrGenericAuthFail", errsList)
		}
		if errsList := fLogin.Validate(ctx, auth.NewPassword("ValidPass1!")); len(errsList) != 0 {
			t.Errorf("valid *Password got unexpected error: %v", errsList)
		}
		if errsList := fLogin.Validate(ctx, 12345); !stderrors.Is(stderrors.Join(errsList...), errs.ErrInvalidType) {
			t.Errorf("invalid type got %v, want ErrInvalidType", errsList)
		}
	})

	t.Run("PasswordField_ValueToFormAndClean", func(t *testing.T) {
		customErr := stderrors.New("custom password error")
		pfRequired := auth.NewPasswordField(
			auth.PasswordFieldOptions{
				Flags:             auth.ChrFlagAll,
				IsRegistering:     true,
				UseDefaultOptions: true,
			},
			fields.Required(true),
		)

		if got := pfRequired.ValueToForm("not-a-password"); got != nil {
			t.Errorf("ValueToForm(non-password) = %#v, want nil", got)
		}
		if got := pfRequired.ValueToForm(auth.NewPassword("")); got != "" {
			t.Errorf("ValueToForm(zero) = %#v, want \"\"", got)
		}
		if got := pfRequired.ValueToForm(auth.NewHashedPassword("$2a$10$hash")); got != "$2a$10$hash" {
			t.Errorf("ValueToForm(hashed) = %#v, want hash", got)
		}
		if got := pfRequired.ValueToForm(auth.NewPassword("RawPass1!")); got != "RawPass1!" {
			t.Errorf("ValueToForm(raw) = %#v, want RawPass1!", got)
		}

		if _, err := pfRequired.Clean(ctx, 999); !stderrors.Is(err, errs.ErrInvalidType) {
			t.Errorf("Clean(int) = %v, want ErrInvalidType", err)
		}
		if _, err := pfRequired.Clean(ctx, ""); !stderrors.Is(err, errs.ErrFieldRequired) {
			t.Errorf("Clean(\"\") when required = %v, want ErrFieldRequired", err)
		}

		pfOptional := auth.NewPasswordField(
			auth.PasswordFieldOptions{
				Flags:         auth.ChrFlagAll,
				IsRegistering: true,
			},
			fields.Required(false),
		)
		auth.PasswordValidators(func(p *auth.Password) error {
			if p.Raw == "Forbidden1!" {
				return customErr
			}
			return nil
		})(pfOptional)

		if val, err := pfOptional.Clean(ctx, ""); err != nil || val != nil {
			t.Errorf("Clean(\"\") when optional = (%v, %v), want (nil, nil)", val, err)
		}
		if _, err := pfOptional.Clean(ctx, "Forbidden1!"); !stderrors.Is(err, customErr) {
			t.Errorf("Clean(custom-rejected) = %v, want customErr", err)
		}
		val, err := pfOptional.Clean(ctx, "ValidPass1!")
		if err != nil {
			t.Fatalf("Clean(valid) unexpected error: %v", err)
		}
		pw, ok := val.(*auth.Password)
		if !ok || pw.Raw != "ValidPass1!" {
			t.Errorf("Clean(valid) = %#v, want *Password with Raw='ValidPass1!'", val)
		}

		// UseDefaultOptions enforces MinLength(8) and MaxLength(64)
		if errsList := pfRequired.Validate(ctx, "Aa1!"); !stderrors.Is(stderrors.Join(errsList...), errs.ErrLengthMin) {
			t.Errorf("Validate short with UseDefaultOptions = %v, want ErrLengthMin", errsList)
		}
		tooLong := strings.Repeat("Aa1!", 17) // 68 chars
		if errsList := pfRequired.Validate(ctx, tooLong); !stderrors.Is(stderrors.Join(errsList...), errs.ErrLengthMax) {
			t.Errorf("Validate long with UseDefaultOptions = %v, want ErrLengthMax", errsList)
		}
	})
}

func TestForms_LoginRegisterAndBaseUserForm_EdgeCases(t *testing.T) {
	ctx := t.Context()
	origEmailLogin := auth.Auth.LoginWithEmail
	t.Cleanup(func() { auth.Auth.LoginWithEmail = origEmailLogin })

	activeUser := createTestUser(t, ctx, "form_active@example.com", "form_active", "ValidPass1!", true, false)
	inactiveUser := createTestUser(t, ctx, "form_inactive@example.com", "form_inactive", "ValidPass1!", false, false)

	t.Run("LoginForm_UsernameMode", func(t *testing.T) {
		auth.Auth.LoginWithEmail = false

		// Valid login
		req := newSessionRequest(http.MethodPost, "/auth/login")
		form := auth.UserLoginForm(req)
		form.WithData(url.Values{
			"username": {activeUser.Username},
			"password": {"ValidPass1!"},
		}, nil, req)

		if !forms.IsValid(form.Context(), form) {
			t.Fatalf("expected login form valid, errs=%v", form.Errors)
		}
		if err := form.Login(); err != nil {
			t.Fatalf("Login() failed: %v", err)
		}
		if form.Instance == nil || form.Instance.ID != activeUser.ID || !form.Instance.IsLoggedIn {
			t.Errorf("expected logged-in instance ID=%d, got %+v", activeUser.ID, form.Instance)
		}

		// Wrong password
		reqWrong := newSessionRequest(http.MethodPost, "/auth/login")
		fWrong := auth.UserLoginForm(reqWrong)
		fWrong.WithData(url.Values{
			"username": {activeUser.Username},
			"password": {"WrongPass1!"},
		}, nil, reqWrong)
		if !forms.IsValid(fWrong.Context(), fWrong) {
			t.Fatalf("expected form valid before auth check")
		}
		if err := fWrong.Login(); !stderrors.Is(err, autherrors.ErrGenericAuthFail) {
			t.Errorf("wrong password Login() = %v, want ErrGenericAuthFail", err)
		}

		// Non-existent user
		reqMissing := newSessionRequest(http.MethodPost, "/auth/login")
		fMissing := auth.UserLoginForm(reqMissing)
		fMissing.WithData(url.Values{
			"username": {"no_such_user"},
			"password": {"ValidPass1!"},
		}, nil, reqMissing)
		if !forms.IsValid(fMissing.Context(), fMissing) {
			t.Fatalf("expected form valid before auth check")
		}
		if err := fMissing.Login(); !stderrors.Is(err, autherrors.ErrGenericAuthFail) {
			t.Errorf("missing user Login() = %v, want ErrGenericAuthFail", err)
		}

		// Inactive user
		reqInactive := newSessionRequest(http.MethodPost, "/auth/login")
		fInactive := auth.UserLoginForm(reqInactive)
		fInactive.WithData(url.Values{
			"username": {inactiveUser.Username},
			"password": {"ValidPass1!"},
		}, nil, reqInactive)
		if !forms.IsValid(fInactive.Context(), fInactive) {
			t.Fatalf("expected form valid before auth check")
		}
		if err := fInactive.Login(); !stderrors.Is(err, autherrors.ErrIsActive) {
			t.Errorf("inactive user Login() = %v, want ErrIsActive", err)
		}

		// Invalid field triggers Protected ErrInvalidLogin and SIGNAL_LOGIN_FAILED
		var loginFailedFired bool
		recv, err := core.SIGNAL_LOGIN_FAILED.Listen(ctx, func(_ context.Context, _ signals.Signal[*http.Request], r *http.Request) error {
			loginFailedFired = true
			return nil
		})
		if err != nil {
			t.Fatalf("listen SIGNAL_LOGIN_FAILED: %v", err)
		}
		defer func() { _ = core.SIGNAL_LOGIN_FAILED.Disconnect(ctx, recv) }()

		reqBad := newSessionRequest(http.MethodPost, "/auth/login")
		fBad := auth.UserLoginForm(reqBad)
		fBad.WithData(url.Values{
			"username": {"1bad"},
			"password": {"weak"},
		}, nil, reqBad)
		if forms.IsValid(fBad.Context(), fBad) {
			t.Fatalf("expected invalid login form for bad regex & weak password")
		}
		if !loginFailedFired {
			t.Errorf("expected SIGNAL_LOGIN_FAILED to fire on invalid login form")
		}
		if uErrs, ok := fBad.Errors.Get("username"); !ok || !stderrors.Is(stderrors.Join(uErrs...), autherrors.ErrInvalidLogin) {
			t.Errorf("expected username error to be protected as ErrInvalidLogin, got %v", uErrs)
		}
	})

	t.Run("LoginForm_EmailMode", func(t *testing.T) {
		auth.Auth.LoginWithEmail = true

		req := newSessionRequest(http.MethodPost, "/auth/login")
		form := auth.UserLoginForm(req)
		form.WithData(url.Values{
			"email":    {activeUser.Email.Address},
			"password": {"ValidPass1!"},
		}, nil, req)
		if !forms.IsValid(form.Context(), form) {
			t.Fatalf("expected email login form valid, errs=%v", form.Errors)
		}
		if err := form.Login(); err != nil {
			t.Fatalf("email Login() failed: %v", err)
		}
		if form.Instance == nil || form.Instance.ID != activeUser.ID {
			t.Errorf("expected logged-in user ID=%d, got %+v", activeUser.ID, form.Instance)
		}

		reqMissing := newSessionRequest(http.MethodPost, "/auth/login")
		fMissing := auth.UserLoginForm(reqMissing)
		fMissing.WithData(url.Values{
			"email":    {"nobody@example.com"},
			"password": {"ValidPass1!"},
		}, nil, reqMissing)
		if !forms.IsValid(fMissing.Context(), fMissing) {
			t.Fatalf("expected email login form valid before lookup")
		}
		if err := fMissing.Login(); !stderrors.Is(err, autherrors.ErrGenericAuthFail) {
			t.Errorf("missing email Login() = %v, want ErrGenericAuthFail", err)
		}
	})

	t.Run("RegisterForm_DuplicatesSignalsAndInvalidDefaults", func(t *testing.T) {
		auth.Auth.LoginWithEmail = false

		// Duplicate email
		reqDupEmail := httptest.NewRequest(http.MethodPost, "/auth/register", nil)
		fDupEmail := auth.UserRegisterForm(reqDupEmail, auth.RegisterFormConfig{AlwaysAllLoginFields: true})
		fDupEmail.WithData(url.Values{
			"email":           {activeUser.Email.Address},
			"username":        {"brand_new_user"},
			"password":        {"ValidPass1!"},
			"passwordConfirm": {"ValidPass1!"},
		}, nil, reqDupEmail)
		if forms.IsValid(fDupEmail.Context(), fDupEmail) {
			t.Fatalf("expected duplicate email registration to fail")
		}
		if !stderrors.Is(stderrors.Join(fDupEmail.ErrorList_...), autherrors.ErrUserExists) {
			t.Errorf("expected ErrUserExists for duplicate email, got %v", fDupEmail.ErrorList_)
		}
		if _, ok := fDupEmail.InvalidDefaults["password"]; ok {
			t.Errorf("expected InvalidDefaults['password'] to be deleted on invalid form")
		}
		if _, ok := fDupEmail.InvalidDefaults["passwordConfirm"]; ok {
			t.Errorf("expected InvalidDefaults['passwordConfirm'] to be deleted on invalid form")
		}

		// Duplicate username
		reqDupUser := httptest.NewRequest(http.MethodPost, "/auth/register", nil)
		fDupUser := auth.UserRegisterForm(reqDupUser, auth.RegisterFormConfig{AlwaysAllLoginFields: true})
		fDupUser.WithData(url.Values{
			"email":           {"brand_new@example.com"},
			"username":        {activeUser.Username},
			"password":        {"ValidPass1!"},
			"passwordConfirm": {"ValidPass1!"},
		}, nil, reqDupUser)
		if forms.IsValid(fDupUser.Context(), fDupUser) {
			t.Fatalf("expected duplicate username registration to fail")
		}
		if !stderrors.Is(stderrors.Join(fDupUser.ErrorList_...), autherrors.ErrUserExists) {
			t.Errorf("expected ErrUserExists for duplicate username, got %v", fDupUser.ErrorList_)
		}

		// SignalPreValidateLogonField aborts validation
		preErr := stderrors.New("prevalidate blocked")
		preRecv, err := auth.SignalPreValidateLogonField.Listen(ctx, func(_ context.Context, _ signals.Signal[*auth.FormSignal], fs *auth.FormSignal) error {
			return preErr
		})
		if err != nil {
			t.Fatalf("listen prevalidate: %v", err)
		}
		reqPre := httptest.NewRequest(http.MethodPost, "/auth/register", nil)
		fPre := auth.UserRegisterForm(reqPre, auth.RegisterFormConfig{AlwaysAllLoginFields: true})
		fPre.WithData(url.Values{
			"email":           {"pre_ok@example.com"},
			"username":        {"pre_ok_user"},
			"password":        {"ValidPass1!"},
			"passwordConfirm": {"ValidPass1!"},
		}, nil, reqPre)
		if forms.IsValid(fPre.Context(), fPre) {
			t.Errorf("expected SignalPreValidateLogonField error to invalidate form")
		}
		if len(fPre.ErrorList_) == 0 {
			t.Errorf("expected SignalPreValidateLogonField error in ErrorList_, got empty")
		} else if _, ok := signals.SignalError(fPre.ErrorList_[0]); !ok && !stderrors.Is(fPre.ErrorList_[0], preErr) {
			t.Errorf("expected SignalError wrapping preErr, got %v", fPre.ErrorList_)
		}
		_ = auth.SignalPreValidateLogonField.Disconnect(ctx, preRecv)

		// SignalPostValidateLogonField aborts validation
		postErr := stderrors.New("postvalidate blocked")
		postRecv, err := auth.SignalPostValidateLogonField.Listen(ctx, func(_ context.Context, _ signals.Signal[*auth.FormSignal], fs *auth.FormSignal) error {
			return postErr
		})
		if err != nil {
			t.Fatalf("listen postvalidate: %v", err)
		}
		reqPost := httptest.NewRequest(http.MethodPost, "/auth/register", nil)
		fPost := auth.UserRegisterForm(reqPost, auth.RegisterFormConfig{AlwaysAllLoginFields: true})
		fPost.WithData(url.Values{
			"email":           {"post_ok@example.com"},
			"username":        {"post_ok_user"},
			"password":        {"ValidPass1!"},
			"passwordConfirm": {"ValidPass1!"},
		}, nil, reqPost)
		if forms.IsValid(fPost.Context(), fPost) {
			t.Errorf("expected SignalPostValidateLogonField error to invalidate form")
		}
		if len(fPost.ErrorList_) == 0 {
			t.Errorf("expected SignalPostValidateLogonField error in ErrorList_, got empty")
		} else if _, ok := signals.SignalError(fPost.ErrorList_[0]); !ok && !stderrors.Is(fPost.ErrorList_[0], postErr) {
			t.Errorf("expected SignalError wrapping postErr, got %v", fPost.ErrorList_)
		}
		_ = auth.SignalPostValidateLogonField.Disconnect(ctx, postRecv)
	})

	t.Run("BaseUserForm_BasicChecksAndCannotSave", func(t *testing.T) {
		auth.Auth.LoginWithEmail = false

		// Uncleaned form
		req := newSessionRequest(http.MethodPost, "/auth/login")
		f := auth.UserLoginForm(req)
		if _, err := f.Save(); err == nil || !strings.Contains(err.Error(), "Form is not cleaned") {
			t.Errorf("expected 'Form is not cleaned', got %v", err)
		}
		if err := f.Login(); err == nil || !strings.Contains(err.Error(), "Form is not cleaned") {
			t.Errorf("expected 'Form is not cleaned' from Login(), got %v", err)
		}

		// SetRequest updates Request
		req2 := newSessionRequest(http.MethodPost, "/auth/login-new")
		f.SetRequest(req2)
		if f.Request != req2 {
			t.Errorf("SetRequest did not update Request")
		}

		// Validated login form cannot be saved (canSave == false)
		f.WithData(url.Values{
			"username": {activeUser.Username},
			"password": {"ValidPass1!"},
		}, nil, req2)
		if !forms.IsValid(f.Context(), f) {
			t.Fatalf("expected valid login form")
		}
		if _, err := f.Save(); err == nil || !strings.Contains(err.Error(), "Form cannot be saved") {
			t.Errorf("expected 'Form cannot be saved', got %v", err)
		}

		// Form with ErrorList_ fails basicChecks
		f.AddFormError(stderrors.New("manual form error"))
		if err := f.Login(); err == nil || !strings.Contains(err.Error(), "manual form error") {
			t.Errorf("expected manual form error from basicChecks, got %v", err)
		}
	})
}

func TestAuthApp_Check(t *testing.T) {
	ctx := t.Context()

	s := mapSettings{}
	msgs := auth.Auth.Check(ctx, s)
	var gotRedirectWarn, gotReverseWarn bool
	for _, m := range msgs {
		if m.ID == "auth.login_redirect_url_not_set" {
			gotRedirectWarn = true
		}
		if m.ID == "auth.login_reverse_url_not_set" {
			gotReverseWarn = true
		}
	}
	if !gotRedirectWarn || !gotReverseWarn {
		t.Errorf("expected both auth check warnings, got %+v", msgs)
	}

	s[auth.APPVAR_LOGIN_REDIRECT_URL] = "/home"
	s[auth.APPVAR_LOGIN_VIEW_REVERSE_URL] = "auth:login"
	msgs2 := auth.Auth.Check(ctx, s)
	for _, m := range msgs2 {
		if m.ID == "auth.login_redirect_url_not_set" || m.ID == "auth.login_reverse_url_not_set" {
			t.Errorf("did not expect auth warning %s when settings provided", m.ID)
		}
	}
}

type mapSettings map[string]any

func (m mapSettings) App() *django.Application           { return django.Global }
func (m mapSettings) Bind(app *django.Application) error { return nil }
func (m mapSettings) Set(key string, val interface{})    { m[key] = val }
func (m mapSettings) Get(key string) (interface{}, bool) { v, ok := m[key]; return v, ok }
func (m mapSettings) Delete(key string)                  { delete(m, key) }

func TestMiddlewareAndUserFromRequest_EdgeCases(t *testing.T) {
	ctx := t.Context()

	t.Run("UnAuthenticatedUser", func(t *testing.T) {
		u := auth.UnAuthenticatedUser()
		if u.Username != "AnonymousUser" || u.Email.Address != "<unknown>" || u.IsLoggedIn || u.IsAuthenticated() {
			t.Errorf("unexpected UnAuthenticatedUser: %+v", u)
		}
	})

	t.Run("UserFromRequest_NoSessionPanics", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec == nil {
				t.Errorf("expected panic when no session exists in request")
			}
		}()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		_ = auth.UserFromRequest(req)
	})

	t.Run("SessionValues_EmptyWrongTypeMissingUserAndValidPreload", func(t *testing.T) {
		user := createTestUser(t, ctx, "mw_user@example.com", "mw_user", "ValidPass1!", true, false)
		perm := models.Setup(ctx, &users.Permission{Name: "mw:perm:direct", Description: "Direct"})
		if _, err := queries.GetQuerySet(&users.Permission{}).Create(perm); err != nil {
			t.Fatalf("create perm: %v", err)
		}
		gPerm := models.Setup(ctx, &users.Permission{Name: "mw:perm:group", Description: "Group"})
		if _, err := queries.GetQuerySet(&users.Permission{}).Create(gPerm); err != nil {
			t.Fatalf("create group perm: %v", err)
		}
		grp := models.Setup(ctx, &users.Group{Name: "mw_group"})
		if _, err := queries.GetQuerySet(&users.Group{}).Create(grp); err != nil {
			t.Fatalf("create group: %v", err)
		}
		t.Cleanup(func() {
			_, _ = queries.GetQuerySet(&users.GroupPermission{}).Filter("GroupID", grp.ID).Delete()
			_, _ = queries.GetQuerySet(&users.Group{}).Filter("ID", grp.ID).Delete()
			_, _ = queries.GetQuerySet(&users.Permission{}).Filter("ID__in", perm.ID, gPerm.ID).Delete()
		})

		if _, err := user.Permissions.Objects().WithContext(ctx).AddTarget(perm); err != nil {
			t.Fatalf("add direct perm: %v", err)
		}
		if _, err := grp.Permissions.Objects().WithContext(ctx).AddTarget(gPerm); err != nil {
			t.Fatalf("add group perm: %v", err)
		}
		if _, err := user.Groups.Objects().WithContext(ctx).AddTarget(grp); err != nil {
			t.Fatalf("add group to user: %v", err)
		}

		// Empty session -> UnAuthenticatedUser
		runWithSessionMiddleware(func(w http.ResponseWriter, r *http.Request) {
			u := auth.UserFromRequest(r)
			if u.IsAuthenticated() || u.Username != "AnonymousUser" {
				t.Errorf("empty session got %+v, want AnonymousUser", u)
			}
		})

		// Wrong type in session -> UnAuthenticatedUser
		runWithSessionMiddleware(func(w http.ResponseWriter, r *http.Request) {
			sessions.Retrieve(r).Set(auth.SESSION_COOKIE_NAME, "not-a-uint64")
			u := auth.UserFromRequest(r)
			if u.IsAuthenticated() || u.Username != "AnonymousUser" {
				t.Errorf("wrong type in session got %+v, want AnonymousUser", u)
			}
		})

		// Non-existent user ID -> UnAuthenticatedUser
		runWithSessionMiddleware(func(w http.ResponseWriter, r *http.Request) {
			sessions.Retrieve(r).Set(auth.SESSION_COOKIE_NAME, uint64(99999999))
			u := auth.UserFromRequest(r)
			if u.IsAuthenticated() || u.Username != "AnonymousUser" {
				t.Errorf("missing user ID got %+v, want AnonymousUser", u)
			}
		})

		// Valid user ID -> loads user + preloaded permissions & groups + caches via AddUserMiddleware
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		mw := sessions.SessionMiddleware(auth.Auth.Session)(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				sessions.Retrieve(r).Set(auth.SESSION_COOKIE_NAME, user.ID)
				auth.AddUserMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					fromCtx := authentication.Retrieve(r)
					if fromCtx == nil || !fromCtx.IsAuthenticated() {
						t.Fatalf("expected authenticated user from AddUserMiddleware")
					}
					u := auth.UserFromRequest(r)
					if u.ID != user.ID || !u.IsLoggedIn {
						t.Errorf("UserFromRequest() = %+v, want user ID %d", u, user.ID)
					}
					if !u.HasObjectPermission(r.Context(), nil, "mw:perm:direct", "mw:perm:group") {
						t.Errorf("expected preloaded direct & group permissions on loaded user")
					}
				})).ServeHTTP(w, r)
			}),
		)
		mw.ServeHTTP(rec, req)
	})
}

func TestUsers_CreatePermissionsAndPermissionsBackend_Integration(t *testing.T) {
	ctx := t.Context()

	t.Cleanup(func() {
		_, _ = queries.GetQuerySet(&users.Permission{}).Filter("Name__in",
			"integ:perm:one",
			"integ:perm:two",
			"integ:perm:three",
		).Delete()
	})

	p1 := models.Setup(ctx, &users.Permission{Name: "integ:perm:one", Description: "One"})
	p2 := models.Setup(ctx, &users.Permission{Name: "integ:perm:two", Description: "Two"})
	p3 := models.Setup(ctx, &users.Permission{Name: "integ:perm:three", Description: "Three"})

	createdAny, err := users.CreatePermissions(ctx, p1, p2)
	if err != nil {
		t.Fatalf("CreatePermissions first call failed: %v", err)
	}
	if !createdAny {
		t.Errorf("expected createdAny=true on initial CreatePermissions")
	}

	// Idempotent second call
	createdAny, err = users.CreatePermissions(ctx, p1, p2)
	if err != nil {
		t.Fatalf("CreatePermissions second call failed: %v", err)
	}
	if createdAny {
		t.Errorf("expected createdAny=false when all permissions already exist")
	}

	// Partial new permission
	createdAny, err = users.CreatePermissions(ctx, p1, p3)
	if err != nil {
		t.Fatalf("CreatePermissions partial call failed: %v", err)
	}
	if !createdAny {
		t.Errorf("expected createdAny=true when at least one permission is new")
	}

	// Lazy user model resolution
	userModelDef := users.GetUserModel()
	if userModelDef == nil {
		t.Fatalf("users.GetUserModel() returned nil")
	}
	if modelStr := users.GetUserModelString(); modelStr == "" {
		t.Errorf("users.GetUserModelString() returned empty string")
	}

	// PermissionsBackend integration with real DB users
	pb := users.NewPermissionsBackend()
	normalUser := createTestUser(t, ctx, "pb_normal@example.com", "pb_normal", "ValidPass1!", true, false)
	adminUser := createTestUser(t, ctx, "pb_admin@example.com", "pb_admin", "ValidPass1!", true, true)
	inactiveAdmin := createTestUser(t, ctx, "pb_inactadmin@example.com", "pb_inactadmin", "ValidPass1!", false, true)

	row, err := queries.GetQuerySet(&users.Permission{}).Filter("Name", "integ:perm:one").First()
	if err != nil {
		t.Fatalf("fetch integ:perm:one: %v", err)
	}
	if _, err := normalUser.Permissions.Objects().WithContext(ctx).AddTarget(row.Object); err != nil {
		t.Fatalf("assign perm: %v", err)
	}

	normalUser.IsLoggedIn = true
	adminUser.IsLoggedIn = true
	inactiveAdmin.IsLoggedIn = true

	reqNormal := httptest.NewRequest(http.MethodGet, "/", nil)
	reqNormal = reqNormal.WithContext(authentication.ContextWithUser(reqNormal.Context(), normalUser))
	if !pb.HasPermission(reqNormal, "integ:perm:one") {
		t.Errorf("expected normalUser to have integ:perm:one via DB check")
	}
	if pb.HasPermission(reqNormal, "integ:perm:two") {
		t.Errorf("expected normalUser NOT to have integ:perm:two")
	}

	reqAdmin := httptest.NewRequest(http.MethodGet, "/", nil)
	reqAdmin = reqAdmin.WithContext(authentication.ContextWithUser(reqAdmin.Context(), adminUser))
	if !pb.HasPermission(reqAdmin, "anything:at:all") {
		t.Errorf("expected active admin to have all permissions")
	}

	reqInactiveAdmin := httptest.NewRequest(http.MethodGet, "/", nil)
	reqInactiveAdmin = reqInactiveAdmin.WithContext(authentication.ContextWithUser(reqInactiveAdmin.Context(), inactiveAdmin))
	if pb.HasPermission(reqInactiveAdmin, "integ:perm:one") {
		t.Errorf("expected inactive admin to be denied")
	}
}

func TestE2E_HTTP_AuthFlows(t *testing.T) {
	ctx := t.Context()

	origEmailLogin := auth.Auth.LoginWithEmail
	origRedirect, hadRedirect := django.Global.Settings.Get(auth.APPVAR_LOGIN_REDIRECT_URL)
	t.Cleanup(func() {
		auth.Auth.LoginWithEmail = origEmailLogin
		if hadRedirect {
			django.Global.Settings.Set(auth.APPVAR_LOGIN_REDIRECT_URL, origRedirect)
		} else {
			django.Global.Settings.Set(auth.APPVAR_LOGIN_REDIRECT_URL, func(*http.Request) string {
				return auth.DEFAULT_LOGIN_REDIRECT_URL
			})
		}
		goldcrest.Unregister(autherrors.AUTH_ERROR_HOOK)
	})
	auth.Auth.LoginWithEmail = false

	// Register test helper routes on Global.Mux once for this test
	django.Global.Mux.Handle(mux.GET, "/e2e-auth/whoami", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := auth.UserFromRequest(r)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"username":      u.Username,
			"authenticated": u.IsAuthenticated(),
			"admin":         u.IsAdmin(),
		})
	}))
	django.Global.Mux.Handle(mux.POST, "/e2e-auth/session-login", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		uid, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		sess := sessions.Retrieve(r)
		_ = sess.RenewToken()
		sess.Set(auth.SESSION_COOKIE_NAME, uid)
		w.WriteHeader(http.StatusOK)
	}))
	django.Global.Mux.Handle(mux.GET, "/e2e-auth/fail", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		autherrors.Fail(http.StatusUnauthorized, "login required", r.URL.RequestURI())
	}))
	django.Global.Mux.Handle(mux.GET, "/e2e-auth/custom-view", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v := auth.LoginView(&views.BaseView{
			BaseTemplateKey: "auth",
			TemplateName:    "auth/login.tmpl",
		})
		v.OnSuccess = func(w http.ResponseWriter, req *http.Request, form *auth.BaseUserForm) error {
			if req.URL.Query().Get("fail_success") == "1" {
				return stderrors.New("onSuccess error")
			}
			w.Header().Set("X-On-Success", "called")
			return nil
		}
		views.Invoke(v, w, r)
	}))
	django.Global.Mux.Handle(mux.POST, "/e2e-auth/custom-view", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v := auth.LoginView(&views.BaseView{
			BaseTemplateKey: "auth",
			TemplateName:    "auth/login.tmpl",
		})
		v.OnSuccess = func(w http.ResponseWriter, req *http.Request, form *auth.BaseUserForm) error {
			if req.URL.Query().Get("fail_success") == "1" {
				return stderrors.New("onSuccess error")
			}
			w.Header().Set("X-On-Success", "called")
			return nil
		}
		views.Invoke(v, w, r)
	}))
	django.Global.Mux.Handle(mux.GET, "/e2e-auth/target", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("target:" + r.URL.Path))
	}))

	srv, err := django.Global.TestServe(false)
	if err != nil {
		t.Fatalf("TestServe failed: %v", err)
	}
	defer srv.Close()

	baseURL := "http://test.local"
	newNoRedirectClient := func() *http.Client {
		jar, _ := cookiejar.New(nil)
		return &http.Client{
			Jar:       jar,
			Transport: handlerTransport{handler: srv.Config.Handler},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}

	t.Run("GetLoginAndRegisterPages", func(t *testing.T) {
		client := newNoRedirectClient()

		resp, err := client.Get(baseURL + "/auth/login?next=/dashboard")
		if err != nil {
			t.Fatalf("GET /auth/login: %v", err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET /auth/login status = %d, want 200 (body=%s)", resp.StatusCode, string(body))
		}
		if !strings.Contains(string(body), `action="/auth/login/"`) || !strings.Contains(string(body), `value="/dashboard"`) {
			t.Errorf("unexpected login HTML: %s", string(body))
		}

		respReg, err := client.Get(baseURL + "/auth/register")
		if err != nil {
			t.Fatalf("GET /auth/register: %v", err)
		}
		regBody, _ := io.ReadAll(respReg.Body)
		respReg.Body.Close()
		if respReg.StatusCode != http.StatusOK {
			t.Fatalf("GET /auth/register status = %d, want 200", respReg.StatusCode)
		}
		if !strings.Contains(string(regBody), "<form") {
			t.Errorf("expected register HTML form, got %s", string(regBody))
		}
	})

	t.Run("RegisterFlow_InvalidAndValid", func(t *testing.T) {
		client := newNoRedirectClient()
		t.Cleanup(func() {
			_, _ = auth.GetUserQuerySet().Filter("Username", "e2e_reg_user").Delete()
		})

		// Invalid register (mismatched passwords) re-renders 200 with error
		respBad, err := client.PostForm(baseURL+"/auth/register", url.Values{
			"email":           {"e2e_reg@example.com"},
			"username":        {"e2e_reg_user"},
			"password":        {"ValidPass1!"},
			"passwordConfirm": {"DifferentPass1!"},
		})
		if err != nil {
			t.Fatalf("POST invalid register: %v", err)
		}
		badBody, _ := io.ReadAll(respBad.Body)
		respBad.Body.Close()
		if respBad.StatusCode != http.StatusOK || !strings.Contains(string(badBody), "form-error") {
			t.Errorf("expected 200 with form-error on invalid register, got %d body=%s", respBad.StatusCode, string(badBody))
		}

		// Valid register redirects 303 to ?next=/e2e-auth/target and persists user in DB
		respGood, err := client.PostForm(baseURL+"/auth/register?next=/e2e-auth/target", url.Values{
			"email":           {"e2e_reg@example.com"},
			"username":        {"e2e_reg_user"},
			"password":        {"ValidPass1!"},
			"passwordConfirm": {"ValidPass1!"},
		})
		if err != nil {
			t.Fatalf("POST valid register: %v", err)
		}
		respGood.Body.Close()
		if respGood.StatusCode != http.StatusSeeOther {
			t.Fatalf("expected 303 See Other after valid register, got %d", respGood.StatusCode)
		}
		if loc := respGood.Header.Get("Location"); loc != "/e2e-auth/target" {
			t.Errorf("Location = %q, want /e2e-auth/target", loc)
		}

		row, err := auth.GetUserQuerySet().WithContext(ctx).Filter("Username", "e2e_reg_user").Get()
		if err != nil {
			t.Fatalf("registered user not found in DB: %v", err)
		}
		if err := auth.CheckPassword(row.Object, "ValidPass1!"); err != nil {
			t.Errorf("registered user password check failed: %v", err)
		}
	})

	t.Run("LoginFlow_InvalidValidAndDynamicRedirect", func(t *testing.T) {
		u := createTestUser(t, ctx, "e2e_login@example.com", "e2e_login", "ValidPass1!", true, false)
		client := newNoRedirectClient()

		var loginSignalUser auth.User
		var loggedInCount, loginFailedCount int
		rIn, _ := core.SIGNAL_USER_LOGGED_IN.Listen(ctx, func(_ context.Context, _ signals.Signal[core.UserWithRequest], uwr core.UserWithRequest) error {
			loggedInCount++
			if usr, ok := uwr.User.(*auth.User); ok {
				loginSignalUser = *usr
			}
			return nil
		})
		rFail, _ := core.SIGNAL_LOGIN_FAILED.Listen(ctx, func(_ context.Context, _ signals.Signal[*http.Request], _ *http.Request) error {
			loginFailedCount++
			return nil
		})
		t.Cleanup(func() {
			_ = core.SIGNAL_USER_LOGGED_IN.Disconnect(ctx, rIn)
			_ = core.SIGNAL_LOGIN_FAILED.Disconnect(ctx, rFail)
		})

		// Wrong password re-renders 200 with form-error
		respBad, err := client.PostForm(baseURL+"/auth/login", url.Values{
			"username": {u.Username},
			"password": {"WrongPass1!"},
		})
		if err != nil {
			t.Fatalf("POST bad login: %v", err)
		}
		badBody, _ := io.ReadAll(respBad.Body)
		respBad.Body.Close()
		if respBad.StatusCode != http.StatusOK || !strings.Contains(string(badBody), "form-error") {
			t.Errorf("expected 200 with form-error on wrong password, got %d body=%s", respBad.StatusCode, string(badBody))
		}

		// Invalid form field fires SIGNAL_LOGIN_FAILED
		respInvalid, err := client.PostForm(baseURL+"/auth/login", url.Values{
			"username": {"1"},
			"password": {"weak"},
		})
		if err != nil {
			t.Fatalf("POST invalid login: %v", err)
		}
		respInvalid.Body.Close()
		if loginFailedCount == 0 {
			t.Errorf("expected SIGNAL_LOGIN_FAILED on invalid form submission")
		}

		// Valid login with ?next
		respOk, err := client.PostForm(baseURL+"/auth/login?next=/custom-next", url.Values{
			"username": {u.Username},
			"password": {"ValidPass1!"},
		})
		if err != nil {
			t.Fatalf("POST valid login: %v", err)
		}
		respOk.Body.Close()
		if respOk.StatusCode != http.StatusSeeOther || respOk.Header.Get("Location") != "/custom-next" {
			t.Errorf("expected 303 -> /custom-next, got %d -> %q", respOk.StatusCode, respOk.Header.Get("Location"))
		}
		if loggedInCount != 1 || loginSignalUser.ID != u.ID {
			t.Errorf("expected SIGNAL_USER_LOGGED_IN for user ID %d (count=%d)", u.ID, loggedInCount)
		}

		// Dynamic func(*http.Request) string for APPVAR_LOGIN_REDIRECT_URL
		django.Global.Settings.Set(auth.APPVAR_LOGIN_REDIRECT_URL, func(r *http.Request) string {
			return "/dynamic-dest?from=" + r.URL.Path
		})
		dynClient := newNoRedirectClient()
		respDyn, err := dynClient.PostForm(baseURL+"/auth/login", url.Values{
			"username": {u.Username},
			"password": {"ValidPass1!"},
		})
		if err != nil {
			t.Fatalf("POST login dynamic redirect: %v", err)
		}
		respDyn.Body.Close()
		if respDyn.StatusCode != http.StatusSeeOther || respDyn.Header.Get("Location") != "/dynamic-dest?from=/auth/login" {
			t.Errorf("expected dynamic redirect '/dynamic-dest?from=/auth/login', got %d -> %q", respDyn.StatusCode, respDyn.Header.Get("Location"))
		}
	})

	t.Run("AuthenticatedRedirect_And_LogoutFlows", func(t *testing.T) {
		u := createTestUser(t, ctx, "e2e_sess@example.com", "e2e_sess", "ValidPass1!", true, true)
		client := newNoRedirectClient()

		django.Global.Settings.Set(auth.APPVAR_LOGIN_REDIRECT_URL, func(*http.Request) string {
			return "/home-redirect"
		})

		// GET /auth/logout -> 404 from router (only POST is routed), while direct GET to LogoutView fails with 405
		respGetLogout, err := client.Get(baseURL + "/auth/logout")
		if err != nil {
			t.Fatalf("GET /auth/logout: %v", err)
		}
		respGetLogout.Body.Close()
		if respGetLogout.StatusCode != http.StatusNotFound {
			t.Errorf("GET /auth/logout status = %d, want 404", respGetLogout.StatusCode)
		}
		func() {
			defer func() {
				rec := recover()
				if rec == nil {
					t.Errorf("expected panic from LogoutView on GET")
				}
			}()
			auth.LogoutView(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/auth/logout", nil))
		}()

		// POST /auth/logout while unauthenticated -> 303 to login
		respUnauthLogout, err := client.Post(baseURL+"/auth/logout", "", nil)
		if err != nil {
			t.Fatalf("POST unauth logout: %v", err)
		}
		respUnauthLogout.Body.Close()
		if respUnauthLogout.StatusCode != http.StatusSeeOther || respUnauthLogout.Header.Get("Location") != "/auth/login/" {
			t.Errorf("unauth logout expected 303 -> /auth/login/, got %d -> %q", respUnauthLogout.StatusCode, respUnauthLogout.Header.Get("Location"))
		}

		// Establish authenticated session cookie
		respSess, err := client.Post(baseURL+"/e2e-auth/session-login?id="+strconv.FormatUint(u.ID, 10), "", nil)
		if err != nil {
			t.Fatalf("POST session-login: %v", err)
		}
		respSess.Body.Close()

		// Confirm whoami sees authenticated admin user
		respWho, err := client.Get(baseURL + "/e2e-auth/whoami")
		if err != nil {
			t.Fatalf("GET whoami: %v", err)
		}
		var who map[string]any
		_ = json.NewDecoder(respWho.Body).Decode(&who)
		respWho.Body.Close()
		if who["username"] != u.Username || who["authenticated"] != true || who["admin"] != true {
			t.Errorf("unexpected whoami payload: %#v", who)
		}

		// GET /auth/login when already authenticated -> 302 redirect to LOGIN_REDIRECT_URL
		respAlready, err := client.Get(baseURL + "/auth/login")
		if err != nil {
			t.Fatalf("GET /auth/login authenticated: %v", err)
		}
		respAlready.Body.Close()
		if respAlready.StatusCode != http.StatusFound || respAlready.Header.Get("Location") != "/home-redirect" {
			t.Errorf("authenticated GET /auth/login expected 302 -> /home-redirect, got %d -> %q", respAlready.StatusCode, respAlready.Header.Get("Location"))
		}

		// POST /auth/logout?next=/goodbye fires SIGNAL_USER_LOGGED_OUT and redirects
		var loggedOutFired bool
		rOut, _ := core.SIGNAL_USER_LOGGED_OUT.Listen(ctx, func(_ context.Context, _ signals.Signal[core.UserWithRequest], _ core.UserWithRequest) error {
			loggedOutFired = true
			return nil
		})
		t.Cleanup(func() { _ = core.SIGNAL_USER_LOGGED_OUT.Disconnect(ctx, rOut) })

		respLogout, err := client.Post(baseURL+"/auth/logout?next=/goodbye", "", nil)
		if err != nil {
			t.Fatalf("POST /auth/logout authenticated: %v", err)
		}
		respLogout.Body.Close()
		if respLogout.StatusCode != http.StatusSeeOther || respLogout.Header.Get("Location") != "/goodbye" {
			t.Errorf("authenticated logout expected 303 -> /goodbye, got %d -> %q", respLogout.StatusCode, respLogout.Header.Get("Location"))
		}
		if !loggedOutFired {
			t.Errorf("expected SIGNAL_USER_LOGGED_OUT to fire")
		}
	})

	t.Run("CustomAuthView_OnSuccessAndErrorHandling", func(t *testing.T) {
		u := createTestUser(t, ctx, "e2e_custom@example.com", "e2e_custom", "ValidPass1!", true, false)
		client := newNoRedirectClient()

		respOk, err := client.PostForm(baseURL+"/e2e-auth/custom-view?next=/done", url.Values{
			"username": {u.Username},
			"password": {"ValidPass1!"},
		})
		if err != nil {
			t.Fatalf("POST custom-view: %v", err)
		}
		respOk.Body.Close()
		if respOk.StatusCode != http.StatusSeeOther || respOk.Header.Get("X-On-Success") != "called" {
			t.Errorf("expected 303 with X-On-Success=called, got %d header=%q", respOk.StatusCode, respOk.Header.Get("X-On-Success"))
		}

		respErr, err := client.PostForm(baseURL+"/e2e-auth/custom-view?fail_success=1", url.Values{
			"username": {u.Username},
			"password": {"ValidPass1!"},
		})
		if err != nil {
			t.Fatalf("POST custom-view fail_success: %v", err)
		}
		respErr.Body.Close()
		if respErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("expected 500 when OnSuccess returns error, got %d", respErr.StatusCode)
		}
	})

	t.Run("AuthErrors_HookInE2EServer", func(t *testing.T) {
		client := newNoRedirectClient()

		// Default hook redirects 302 to /auth/login/?next=...
		resp, err := client.Get(baseURL + "/e2e-auth/fail?tab=2")
		if err != nil {
			t.Fatalf("GET /e2e-auth/fail: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusFound {
			t.Fatalf("expected 302 Found from autherrors.Fail, got %d", resp.StatusCode)
		}
		if loc := resp.Header.Get("Location"); loc != "/auth/login/?next=%2Fe2e-auth%2Ffail%3Ftab%3D2" {
			t.Errorf("Location = %q, want /auth/login/?next=%%2Fe2e-auth%%2Ffail%%3Ftab%%3D2", loc)
		}

		// Custom OnAuthenticationError hook overrides redirect
		autherrors.OnAuthenticationError(func(w http.ResponseWriter, r *http.Request, app *django.Application, err *autherrors.AuthenticationError) bool {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("api-unauthorized:" + err.Message))
			return true
		})
		respCustom, err := client.Get(baseURL + "/e2e-auth/fail")
		if err != nil {
			t.Fatalf("GET /e2e-auth/fail with custom hook: %v", err)
		}
		b, _ := io.ReadAll(respCustom.Body)
		respCustom.Body.Close()
		if respCustom.StatusCode != http.StatusUnauthorized || string(b) != "api-unauthorized:login required" {
			t.Errorf("expected 401 api-unauthorized:login required, got %d %q", respCustom.StatusCode, string(b))
		}
	})

	t.Run("AuthView_MissingTemplatePanics", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec == nil {
				t.Errorf("expected panic when AuthView.TemplateName is empty")
			}
		}()
		v := auth.LoginView(&views.BaseView{TemplateName: ""})
		_, _ = v.GetContext(httptest.NewRequest(http.MethodGet, "/auth/login", nil))
	})
}

func TestPassword_StringHashEdgeCases(t *testing.T) {
	var nilPwd *auth.Password
	if !nilPwd.IsZero() {
		t.Errorf("expected nil *Password to be IsZero")
	}
	if nilPwd.String() != "<nil>" {
		t.Errorf("nilPwd.String() = %q, want <nil>", nilPwd.String())
	}
	if err := nilPwd.Check(""); err != nil {
		t.Errorf("nilPwd.Check(\"\") = %v, want nil", err)
	}
	if err := nilPwd.Check("nonempty"); !stderrors.Is(err, securedb.ErrSecureValue) {
		t.Errorf("nilPwd.Check(\"nonempty\") = %v, want ErrSecureValue", err)
	}

	rawPwd := auth.NewPassword("Secret123!")
	if rawPwd.String() != "***********" {
		t.Errorf("rawPwd.String() = %q, want ***********", rawPwd.String())
	}
	hashedPwd := auth.NewHashedPassword("$2a$10$examplehashvalue")
	if hashedPwd.String() != "$2a$10$examplehashvalue" {
		t.Errorf("hashedPwd.String() = %q", hashedPwd.String())
	}

	zeroPwd := auth.NewPassword("")
	if !zeroPwd.IsZero() {
		t.Errorf("expected empty NewPassword to be IsZero")
	}
	if v, err := zeroPwd.Value(); err != nil || v != nil {
		t.Errorf("zeroPwd.Value() = (%v, %v), want (nil, nil)", v, err)
	}
	if err := zeroPwd.SetValue("ValidPass1!"); err != nil {
		t.Fatalf("SetValue failed: %v", err)
	}
	if zeroPwd.Raw != "ValidPass1!" || zeroPwd.Hash == "" {
		t.Errorf("expected Raw and Hash set after SetValue, got %+v", zeroPwd)
	}

	// Scan variants
	var scanned auth.Password
	if err := scanned.Scan(nil); err != nil {
		t.Errorf("Scan(nil) error: %v", err)
	}
	if err := scanned.Scan("hash-str"); err != nil || scanned.Hash != "hash-str" {
		t.Errorf("Scan(string) = (%q, %v)", scanned.Hash, err)
	}
	if err := scanned.Scan([]byte("hash-bytes")); err != nil || scanned.Hash != "hash-bytes" {
		t.Errorf("Scan([]byte) = (%q, %v)", scanned.Hash, err)
	}
	if err := scanned.Scan(securedb.StringHash[auth.BcryptHasher]{Hash: "hash-struct"}); err != nil || scanned.Hash != "hash-struct" {
		t.Errorf("Scan(StringHash) = (%q, %v)", scanned.Hash, err)
	}
	if err := scanned.Scan(12345); !stderrors.Is(err, securedb.ErrSecureValue) {
		t.Errorf("Scan(int) = %v, want ErrSecureValue", err)
	}
}

func TestSettings_ConfigGetInterfaceEdgeCases(t *testing.T) {
	s := djangoapps.NewConfig(mapSettings{})

	// Unset key falls back to default
	if v, ok := s.GetOK[interface{}]("LOGIN_REDIRECT_URL", "/default"); ok || v != "/default" {
		t.Errorf("unset ConfigGetOK[any] = (%v, %v), want (/default, false)", v, ok)
	}

	// Explicit nil value returns (nil, false)
	s.ApplicationSettings.(mapSettings)["LOGIN_REDIRECT_URL"] = nil
	if v, ok := s.GetOK[interface{}]("LOGIN_REDIRECT_URL", "/default"); ok || v != nil {
		t.Errorf("nil ConfigGetOK[any] = (%v, %v), want (nil, false)", v, ok)
	}

	// Empty string returns default with ok=true
	s.ApplicationSettings.(mapSettings)["LOGIN_REDIRECT_URL"] = ""
	if v, ok := s.GetOK[interface{}]("LOGIN_REDIRECT_URL", "/default"); !ok || v != "/default" {
		t.Errorf("empty string ConfigGetOK[any] = (%v, %v), want (/default, true)", v, ok)
	}

	// func(*http.Request) string succeeds
	fn := func(*http.Request) string { return "/fn-target" }
	s.ApplicationSettings.(mapSettings)["LOGIN_REDIRECT_URL"] = fn
	if v, ok := s.GetOK[interface{}]("LOGIN_REDIRECT_URL", "/default"); !ok || v == nil {
		t.Errorf("func ConfigGetOK[any] = (%v, %v), want non-nil func, true", v, ok)
	}
}
