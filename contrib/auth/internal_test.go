package auth

import (
	"bytes"
	stderrors "errors"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	autherrors "github.com/Nigel2392/go-django/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/contrib/auth/users"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/queries/src/models"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/mux"
)

type mockCommandManager struct {
	inputs          []string
	protectedInputs []string
	inputIdx        int
	protectedIdx    int
	out             bytes.Buffer
	err             bytes.Buffer
	cmd             command.Command
	reg             command.Registry
	inputErrOnce    bool
	protErrOnce     bool
}

func (m *mockCommandManager) Log(message string)               { m.out.WriteString(message + "\n") }
func (m *mockCommandManager) Logf(format string, args ...any)  {}
func (m *mockCommandManager) Stdout() io.Writer                { return &m.out }
func (m *mockCommandManager) Stderr() io.Writer                { return &m.err }
func (m *mockCommandManager) Stdin() io.Reader                 { return strings.NewReader("") }
func (m *mockCommandManager) Registry() command.Registry       { return m.reg }
func (m *mockCommandManager) Command() command.Command         { return m.cmd }
func (m *mockCommandManager) Input(question string) (string, error) {
	if m.inputErrOnce {
		m.inputErrOnce = false
		return "", stderrors.New("transient input error")
	}
	if m.inputIdx >= len(m.inputs) {
		panic("mockCommandManager.Input exhausted")
	}
	v := m.inputs[m.inputIdx]
	m.inputIdx++
	return v, nil
}
func (m *mockCommandManager) ProtectedInput(question string) (string, error) {
	if m.protErrOnce {
		m.protErrOnce = false
		return "", stderrors.New("transient protected input error")
	}
	if m.protectedIdx >= len(m.protectedInputs) {
		panic("mockCommandManager.ProtectedInput exhausted")
	}
	v := m.protectedInputs[m.protectedIdx]
	m.protectedIdx++
	return v, nil
}

func cleanupInternalUsers(t *testing.T, usernames ...string) {
	t.Helper()
	t.Cleanup(func() {
		for _, uname := range usernames {
			_, _ = GetUserQuerySet().Filter("Username", uname).Delete()
		}
	})
}

func TestCommands_CreateUser(t *testing.T) {
	ctx := t.Context()
	cleanupInternalUsers(t, "cmd_u0", "cmd_u1", "cmd_u2", "cmd_u3")

	t.Run("TooManyArgsReturnsValueError", func(t *testing.T) {
		m := &mockCommandManager{}
		err := command_create_user.Exec(m, []string{"a", "b", "c", "d"})
		if !stderrors.Is(err, errors.ValueError) {
			t.Errorf("expected errors.ValueError for >3 args, got %v", err)
		}
	})

	t.Run("ThreeArgsWithFlags", func(t *testing.T) {
		cmd := &command.Cmd[createUserStorage]{
			ID:       command_create_user.ID,
			Desc:     command_create_user.Desc,
			FlagFunc: command_create_user.FlagFunc,
			Execute:  command_create_user.Execute,
		}
		fs := flag.NewFlagSet("createuser", flag.ContinueOnError)
		m := &mockCommandManager{cmd: cmd}
		if err := cmd.AddFlags(m, fs); err != nil {
			t.Fatalf("AddFlags: %v", err)
		}
		if err := fs.Parse([]string{"-s", "-i", "cmd_u3@example.com", "cmd_u3", "ValidPass1!"}); err != nil {
			t.Fatalf("fs.Parse: %v", err)
		}
		if err := cmd.Exec(m, fs.Args()); err != nil {
			t.Fatalf("Exec(3 args): %v", err)
		}

		row, err := GetUserQuerySet().WithContext(ctx).Filter("Username", "cmd_u3").Get()
		if err != nil {
			t.Fatalf("query created user: %v", err)
		}
		if !row.Object.IsAdministrator || row.Object.IsActive {
			t.Errorf("expected IsAdministrator=true, IsActive=false; got %+v", row.Object.Base)
		}
		if err := CheckPassword(row.Object, "ValidPass1!"); err != nil {
			t.Errorf("password mismatch: %v", err)
		}
	})

	t.Run("InteractiveRetriesAndArgModes", func(t *testing.T) {
		// 0 args with transient input errors, invalid email + password mismatch retry, weak password retry, then success
		m0 := &mockCommandManager{
			inputErrOnce:    true,
			protErrOnce:     true,
			inputs:          []string{"not-an-email", "cmd_u0", "cmd_u0@example.com"},
			protectedInputs: []string{"ValidPass1!", "Mismatch1!", "weak", "weak", "ValidPass1!", "ValidPass1!"},
		}
		if err := command_create_user.Execute(m0, createUserStorage{}, nil); err != nil {
			t.Fatalf("Exec(0 args interactive): %v", err)
		}
		row0, err := GetUserQuerySet().WithContext(ctx).Filter("Username", "cmd_u0").Get()
		if err != nil {
			t.Fatalf("query cmd_u0: %v", err)
		}
		if !row0.Object.IsActive || row0.Object.IsAdministrator {
			t.Errorf("unexpected flags for cmd_u0: %+v", row0.Object.Base)
		}

		// 1 arg (email)
		m1 := &mockCommandManager{
			inputs:          []string{"cmd_u1"},
			protectedInputs: []string{"ValidPass1!", "ValidPass1!"},
		}
		if err := command_create_user.Execute(m1, createUserStorage{}, []string{"cmd_u1@example.com"}); err != nil {
			t.Fatalf("Exec(1 arg): %v", err)
		}

		// 2 args (email, username)
		m2 := &mockCommandManager{
			protectedInputs: []string{"ValidPass1!", "ValidPass1!"},
		}
		if err := command_create_user.Execute(m2, createUserStorage{}, []string{"cmd_u2@example.com", "cmd_u2"}); err != nil {
			t.Fatalf("Exec(2 args): %v", err)
		}
	})
}

func TestCommands_ChangeUserAndSetPassword(t *testing.T) {
	ctx := t.Context()
	origEmailLogin := Auth.LoginWithEmail
	t.Cleanup(func() { Auth.LoginWithEmail = origEmailLogin })
	cleanupInternalUsers(t, "cmd_chg")

	u := models.Setup(ctx, &User{
		Email:    drivers.MustParseEmail("cmd_chg@example.com"),
		Username: "cmd_chg",
		Password: NewPassword("InitialPass1!"),
		Base: users.Base{
			IsAdministrator: false,
			IsActive:        true,
		},
	})
	if _, err := GetUserQuerySet().WithContext(ctx).Create(u); err != nil {
		t.Fatalf("create cmd_chg user: %v", err)
	}

	t.Run("ChangeUser_UsernameMode", func(t *testing.T) {
		Auth.LoginWithEmail = false
		cmd := &command.Cmd[changeUserStorage]{
			ID:       command_change_user.ID,
			Desc:     command_change_user.Desc,
			FlagFunc: command_change_user.FlagFunc,
			Execute:  command_change_user.Execute,
		}
		fs := flag.NewFlagSet("changeuser", flag.ContinueOnError)
		m := &mockCommandManager{
			inputErrOnce: true,
			inputs:       []string{"missing_user", "cmd_chg"},
			cmd:          cmd,
		}
		if err := cmd.AddFlags(m, fs); err != nil {
			t.Fatalf("AddFlags: %v", err)
		}
		if err := fs.Parse([]string{"-s", "-i"}); err != nil {
			t.Fatalf("fs.Parse: %v", err)
		}
		if err := cmd.Exec(m, fs.Args()); err != nil {
			t.Fatalf("changeuser username mode: %v", err)
		}

		row, err := GetUserQuerySet().WithContext(ctx).Filter("Username", "cmd_chg").Get()
		if err != nil {
			t.Fatalf("reload cmd_chg: %v", err)
		}
		if !row.Object.IsAdministrator || row.Object.IsActive {
			t.Errorf("expected IsAdministrator=true, IsActive=false; got %+v", row.Object.Base)
		}
	})

	t.Run("ChangeUser_EmailMode", func(t *testing.T) {
		Auth.LoginWithEmail = true
		m := &mockCommandManager{
			inputErrOnce: true,
			inputs:       []string{"bad-email", "missing@example.com", "cmd_chg@example.com"},
		}
		if err := command_change_user.Execute(m, changeUserStorage{super: false, inactive: false}, nil); err != nil {
			t.Fatalf("changeuser email mode: %v", err)
		}
		row, err := GetUserQuerySet().WithContext(ctx).Filter("Username", "cmd_chg").Get()
		if err != nil {
			t.Fatalf("reload cmd_chg: %v", err)
		}
		if row.Object.IsAdministrator || !row.Object.IsActive {
			t.Errorf("expected IsAdministrator=false, IsActive=true; got %+v", row.Object.Base)
		}
	})

	t.Run("SetPassword_UsernameAndEmailModes", func(t *testing.T) {
		Auth.LoginWithEmail = false
		mUser := &mockCommandManager{
			inputErrOnce:    true,
			protErrOnce:     true,
			inputs:          []string{"cmd_chg", "cmd_chg", "cmd_chg", "cmd_chg"},
			protectedInputs: []string{"Mismatch1!", "Mismatch2!", "weak", "weak", "NewPass123!", "NewPass123!"},
		}
		if err := command_set_password.Execute(mUser, nil, nil); err != nil {
			t.Fatalf("set_password username mode: %v", err)
		}
		row, err := GetUserQuerySet().WithContext(ctx).Filter("Username", "cmd_chg").Get()
		if err != nil {
			t.Fatalf("reload after set_password: %v", err)
		}
		if err := CheckPassword(row.Object, "NewPass123!"); err != nil {
			t.Errorf("new password check failed: %v", err)
		}

		Auth.LoginWithEmail = true
		mEmail := &mockCommandManager{
			inputs:          []string{"cmd_chg@example.com"},
			protectedInputs: []string{"EmailPass1!", "EmailPass1!"},
		}
		if err := command_set_password.Execute(mEmail, nil, nil); err != nil {
			t.Fatalf("set_password email mode: %v", err)
		}
		row2, err := GetUserQuerySet().WithContext(ctx).Filter("Username", "cmd_chg").Get()
		if err != nil {
			t.Fatalf("reload after email set_password: %v", err)
		}
		if err := CheckPassword(row2.Object, "EmailPass1!"); err != nil {
			t.Errorf("email new password check failed: %v", err)
		}
	})
}

func TestInitAuthEditForm_And_CallOrString(t *testing.T) {
	ctx := t.Context()

	// callOrString handles both string and func(*http.Request) string
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	if got := callOrString("/literal", req); got != "/literal" {
		t.Errorf("callOrString(string) = %q, want /literal", got)
	}
	if got := callOrString(func(r *http.Request) string { return "/fn:" + r.URL.Path }, req); got != "/fn:/test" {
		t.Errorf("callOrString(func) = %q, want /fn:/test", got)
	}

	cleanupInternalUsers(t, "editform_u")
	u := models.Setup(ctx, &User{
		Email:     drivers.MustParseEmail("editform_u@example.com"),
		Username:  "editform_u",
		FirstName: "Edit",
		LastName:  "Form",
		Password:  NewPassword("OrigPass1!"),
		Base: users.Base{
			IsActive: true,
		},
	})
	if _, err := u.Password.Value(); err != nil {
		t.Fatalf("hash password: %v", err)
	}
	created, err := queries.GetQuerySet(&User{}).WithContext(ctx).Create(u)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	newEditForm := func(data url.Values) modelforms.ModelForm[attrs.Definer] {
		mf := modelforms.NewBaseModelForm[attrs.Definer](ctx, created)
		mf.SetExclude("ID", "CreatedAt", "UpdatedAt", "LastLogin")
		mf.Load()
		initAuthEditForm(created, mf)
		mf.WithData(data, nil, req)
		return mf
	}

	t.Run("UnchangedHashWithoutConfirmClearsPassword", func(t *testing.T) {
		f := newEditForm(url.Values{
			"Email":           {created.Email.Address},
			"Username":        {created.Username},
			"FirstName":       {"Edit"},
			"LastName":        {"Form"},
			"IsActive":        {"on"},
			"Password":        {created.Password.Hash},
			"PasswordConfirm": {""},
		})
		if !forms.IsValid(ctx, f) {
			t.Fatalf("expected valid edit form when password hash unchanged, errs=%v list=%v", f.BoundErrors(), f.ErrorList())
		}
		cleaned := f.CleanedData()
		if _, ok := cleaned["Password"]; ok {
			t.Errorf("expected Password to be deleted from cleaned data when unchanged")
		}
		if _, ok := cleaned["PasswordConfirm"]; ok {
			t.Errorf("expected PasswordConfirm to be deleted from cleaned data when unchanged")
		}
	})

	t.Run("ChangedPasswordWithoutConfirmFails", func(t *testing.T) {
		f := newEditForm(url.Values{
			"Email":           {created.Email.Address},
			"Username":        {created.Username},
			"FirstName":       {"Edit"},
			"LastName":        {"Form"},
			"IsActive":        {"on"},
			"Password":        {"ChangedPass1!"},
			"PasswordConfirm": {""},
		})
		if forms.IsValid(ctx, f) {
			t.Fatalf("expected edit form to fail without PasswordConfirm")
		}
		if !stderrors.Is(stderrors.Join(f.ErrorList()...), autherrors.ErrPwdNoMatch) {
			t.Errorf("expected ErrPwdNoMatch, got %v", f.ErrorList())
		}
	})

	t.Run("MismatchedPasswordConfirmFails", func(t *testing.T) {
		f := newEditForm(url.Values{
			"Email":           {created.Email.Address},
			"Username":        {created.Username},
			"FirstName":       {"Edit"},
			"LastName":        {"Form"},
			"IsActive":        {"on"},
			"Password":        {"ChangedPass1!"},
			"PasswordConfirm": {"DifferentPass1!"},
		})
		if forms.IsValid(ctx, f) {
			t.Fatalf("expected edit form to fail on password mismatch")
		}
		if !stderrors.Is(stderrors.Join(f.ErrorList()...), autherrors.ErrPwdNoMatch) {
			t.Errorf("expected ErrPwdNoMatch, got %v", f.ErrorList())
		}
	})

	t.Run("MatchingNewPasswordUpdatesCleanedData", func(t *testing.T) {
		f := newEditForm(url.Values{
			"Email":           {created.Email.Address},
			"Username":        {created.Username},
			"FirstName":       {"Edit"},
			"LastName":        {"Form"},
			"IsActive":        {"on"},
			"Password":        {"ChangedPass1!"},
			"PasswordConfirm": {"ChangedPass1!"},
		})
		if !forms.IsValid(ctx, f) {
			t.Fatalf("expected valid edit form on matching new password, errs=%v list=%v", f.BoundErrors(), f.ErrorList())
		}
		pw, ok := f.CleanedData()["Password"].(*Password)
		if !ok || pw.Raw != "ChangedPass1!" {
			t.Errorf("expected new *Password with Raw='ChangedPass1!', got %#v", f.CleanedData()["Password"])
		}
	})

	t.Run("OmittedPasswordConfirmKeyInFormData", func(t *testing.T) {
		f := newEditForm(url.Values{
			"Email":     {created.Email.Address},
			"Username":  {created.Username},
			"FirstName": {"Edit"},
			"LastName":  {"Form"},
			"IsActive":  {"on"},
			"Password":  {created.Password.Hash},
		})
		if forms.IsValid(ctx, f) {
			t.Fatalf("expected omitted PasswordConfirm key to fail field validation")
		}
		if bErrs, ok := f.BoundErrors().Get("PasswordConfirm"); !ok || !stderrors.Is(stderrors.Join(bErrs...), errs.ErrInvalidType) {
			t.Errorf("expected ErrInvalidType on omitted PasswordConfirm, got %v", bErrs)
		}
	})

	t.Run("NilPasswordWithEmptyConfirmFails", func(t *testing.T) {
		f := newEditForm(url.Values{
			"Email":           {created.Email.Address},
			"Username":        {created.Username},
			"FirstName":       {"Edit"},
			"LastName":        {"Form"},
			"IsActive":        {"on"},
			"Password":        {""},
			"PasswordConfirm": {""},
		})
		// Empty Password is Required(true) on model field OR validator fails with ErrPwdNoMatch if optional
		if forms.IsValid(ctx, f) {
			t.Fatalf("expected empty password to fail edit form validation")
		}
	})
}

func TestAuthView_InternalEdgeCases(t *testing.T) {
	t.Run("GetContext_NilVsEmptyTemplateName", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)

		// Empty string TemplateName panics
		func() {
			defer func() {
				if rec := recover(); rec == nil {
					t.Errorf("expected panic for empty string TemplateName")
				}
			}()
			v := LoginView(&views.BaseView{TemplateName: ""})
			_, _ = v.GetContext(req)
		}()

		// Nil TemplateName (any(nil) != "") does not panic and GetTemplates returns empty slice
		vNil := LoginView(&views.BaseView{})
		if _, err := vNil.GetContext(req); err != nil {
			t.Fatalf("unexpected error for nil TemplateName GetContext: %v", err)
		}
		if tpls := vNil.GetTemplates(req); len(tpls) != 0 {
			t.Errorf("expected empty templates slice, got %v", tpls)
		}
	})

	t.Run("GetContext_MissingGetFormAndOnValidPanics", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)

		func() {
			defer func() {
				if rec := recover(); rec == nil {
					t.Errorf("expected panic when getForm is nil")
				}
			}()
			v := newAuthView[*BaseUserForm](&views.BaseView{TemplateName: "auth/login.tmpl"})
			v.onValid = func(req *http.Request, form *BaseUserForm) error { return nil }
			_, _ = v.GetContext(req)
		}()

		func() {
			defer func() {
				if rec := recover(); rec == nil {
					t.Errorf("expected panic when onValid is nil")
				}
			}()
			v := newAuthView[*BaseUserForm](&views.BaseView{TemplateName: "auth/login.tmpl"})
			v.getForm = func(req *http.Request) *BaseUserForm { return UserLoginForm(req) }
			_, _ = v.GetContext(req)
		}()
	})

	t.Run("Render_OnInvalidCallback", func(t *testing.T) {
		var onInvalidErr = stderrors.New("custom onInvalid override")

		// Case 1: Form is valid, onValid fails -> onInvalid overrides error and adds to form errors
		v1 := newAuthView[*BaseUserForm](&views.BaseView{
			BaseTemplateKey: "auth",
			TemplateName:    "auth/login.tmpl",
		})
		v1.getForm = func(req *http.Request) *BaseUserForm {
			return UserLoginForm(req)
		}
		v1.onValid = func(req *http.Request, form *BaseUserForm) error {
			return stderrors.New("valid handler error")
		}
		var onInvalidCalled1 bool
		v1.onInvalid = func(req *http.Request, form *BaseUserForm) error {
			onInvalidCalled1 = true
			return onInvalidErr
		}

		req1 := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader("username=validuser&password=ValidPass1!"))
		req1.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec1 := httptest.NewRecorder()
		ctx1, err := v1.GetContext(req1)
		if err != nil {
			t.Fatalf("GetContext1: %v", err)
		}
		if err := v1.Render(rec1, req1, "auth/login.tmpl", ctx1); err != nil {
			t.Fatalf("Render1: %v", err)
		}
		if !onInvalidCalled1 || !strings.Contains(rec1.Body.String(), "custom onInvalid override") {
			t.Errorf("expected onInvalid error rendered, called=%v body=%s", onInvalidCalled1, rec1.Body.String())
		}

		// Case 2: Form is invalid -> onInvalid runs and adds error
		v2 := newAuthView[*BaseUserForm](&views.BaseView{
			BaseTemplateKey: "auth",
			TemplateName:    "auth/login.tmpl",
		})
		v2.getForm = func(req *http.Request) *BaseUserForm {
			return UserLoginForm(req)
		}
		v2.onValid = func(req *http.Request, form *BaseUserForm) error { return nil }
		var onInvalidCalled2 bool
		v2.onInvalid = func(req *http.Request, form *BaseUserForm) error {
			onInvalidCalled2 = true
			return stderrors.New("invalid form callback")
		}

		req2 := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader("username=1&password=x"))
		req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec2 := httptest.NewRecorder()
		ctx2, _ := v2.GetContext(req2)
		if err := v2.Render(rec2, req2, "auth/login.tmpl", ctx2); err != nil {
			t.Fatalf("Render2: %v", err)
		}
		if !onInvalidCalled2 || !strings.Contains(rec2.Body.String(), "invalid form callback") {
			t.Errorf("expected invalid callback error rendered, called=%v body=%s", onInvalidCalled2, rec2.Body.String())
		}
	})
}

func TestCommandCreateUser_InvalidEmailThreeArgsEdgeCase(t *testing.T) {
	defer func() {
		rec := recover()
		if rec == nil {
			t.Errorf("expected panic when 3 args have invalid email (u.Email = nil fails non-null field validation)")
		}
	}()
	m := &mockCommandManager{}
	_ = command_create_user.Execute(m, createUserStorage{}, []string{"not-an-email", "user_bad_email", "ValidPass1!"})
}

func TestAppRouting_ConfigFlagsEdgeCases(t *testing.T) {
	origSettings := django.Global.Settings
	origAuth := *Auth
	t.Cleanup(func() {
		django.Global.Settings = origSettings
		*Auth = origAuth
	})

	appCfg := NewAppConfig().(*migrator.MigratorAppConfig).AppConfig.(*AuthApplication)

	t.Run("AllowUserRegisterFalse", func(t *testing.T) {
		django.Global.Settings = django.Config(map[string]any{
			APPVAR_REGISTER_AUTH_URLS:  true,
			APPVAR_ALLOW_USER_REGISTER: false,
		})
		m := mux.New()
		appCfg.Routing(m)

		if _, err := m.Reverse("auth:login"); err != nil {
			t.Errorf("expected auth:login to be registered: %v", err)
		}
		if _, err := m.Reverse("auth:logout"); err != nil {
			t.Errorf("expected auth:logout to be registered: %v", err)
		}
		if _, err := m.Reverse("auth:register"); err == nil {
			t.Errorf("expected auth:register NOT to be registered when ALLOW_USER_REGISTER=false")
		}
	})

	t.Run("RegisterAuthURLsFalse", func(t *testing.T) {
		django.Global.Settings = django.Config(map[string]any{
			APPVAR_REGISTER_AUTH_URLS: false,
		})
		m := mux.New()
		appCfg.Routing(m)

		if _, err := m.Reverse("auth:login"); err == nil {
			t.Errorf("expected auth:login NOT to be registered when REGISTER_AUTH_URLS=false")
		}
		if _, err := m.Reverse("auth:register"); err == nil {
			t.Errorf("expected auth:register NOT to be registered when REGISTER_AUTH_URLS=false")
		}
		if _, err := m.Reverse("auth:logout"); err == nil {
			t.Errorf("expected auth:logout NOT to be registered when REGISTER_AUTH_URLS=false")
		}
	})
}
