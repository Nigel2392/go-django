# Authentication App (`auth`)

The `auth` app provides session-backed user authentication, password validation and hashing, login/registration forms and views, authentication error redirection hooks, admin integration, and CLI user management commands.

For user model internals, custom user models, and RBAC permission evaluation, see the companion pages:

- [User Model, Passwords, and Custom Users](./user.md)
- [Permissions Backend and RBAC](./permissions.md)

---

## Setup and Dependencies

`auth.NewAppConfig()` configures the `AuthApplication` singleton (`auth.Auth`) and wraps it in a `*migrator.MigratorAppConfig` with embedded migrations for `User`, `Group`, `Permission`, `UserGroup`, `GroupPermission`, and `UserPermission`:

```go
type AuthApplication struct {
	*apps.AppConfig
	Session         *scs.SessionManager
	LoginWithEmail  bool
	LoginReverseURL string
}

var Auth *AuthApplication = &AuthApplication{}
```

The `auth` app declares `Deps = []string{"session"}` and requires both `django.APPVAR_DATABASE` (`drivers.Database`) and `django.APPVAR_SESSION_MANAGER` (`*scs.SessionManager`, initialized by [Sessions](../sessions.md)) in `settings`:

```go
package main

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/auth"
	"github.com/Nigel2392/go-django/contrib/auth/users"
	"github.com/Nigel2392/go-django/contrib/messages"
	"github.com/Nigel2392/go-django/contrib/session"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/permissions"
)

func main() {
	db, err := drivers.Open(context.Background(), "sqlite3", "./db.sqlite3")
	if err != nil {
		panic(err)
	}

	// Enable database/cache permission checks (not set automatically by auth).
	permissions.Tester = users.NewPermissionsBackend()

	app := django.App(
		django.Configure(map[string]any{
			django.APPVAR_ALLOWED_HOSTS:      []string{"127.0.0.1", "localhost"},
			django.APPVAR_DATABASE:           db,
			migrator.APPVAR_MIGRATION_DIR:    "./migrations",
			auth.APPVAR_AUTH_EMAIL_LOGIN:     true,
			auth.APPVAR_LOGIN_REDIRECT_URL:   "/",
			auth.APPVAR_LOGIN_VIEW_REVERSE_URL: "auth:login",
		}),
		django.Apps(
			session.NewAppConfig,
			auth.NewAppConfig,
			messages.NewAppConfig,
			admin.NewAppConfig,
			migrator.NewAppConfig,
		),
	)

	if err := app.Initialize(); err != nil {
		panic(err)
	}
	_ = app.Serve()
}
```

### Migrations vs. Non-Migrator Startup

- When `migrator` is installed, `auth` exposes embedded schema migrations (`migrations/auth`) for all six models (`auth_users`, `auth_groups`, `auth_permissions`, `auth_user_groups`, `auth_group_permissions`, and `auth_user_permissions`).
- When `migrator` is **not** installed (`!django.AppInstalled("migrator")`), `auth.Init` automatically creates only the `auth_users` table and its unique indexes (`CreateTable(..., true)` / `AddIndex(..., true)`). If you use groups or permissions without `migrator`, you must create those five tables yourself or include [Migrations](../../../queries/docs/migrations.md).

---

## Configuration Settings and Checks

All five `auth` settings are defined in `contrib/auth/settings.go`:

| Constant | Key | Type | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `APPVAR_AUTH_EMAIL_LOGIN` | `"AUTH_EMAIL_LOGIN"` | `bool` | `false` | Authenticate via `email` instead of `username` in login forms and CLI lookups. |
| `APPVAR_REGISTER_AUTH_URLS` | `"REGISTER_AUTH_URLS"` | `bool` | `true` | Mount `/auth/login`, `/auth/register`, and `/auth/logout` routes in `app.Routing`. |
| `APPVAR_ALLOW_USER_REGISTER` | `"ALLOW_USER_REGISTER"` | `bool` | `true` | Mount `/auth/register` when `REGISTER_AUTH_URLS` is `true`. |
| `APPVAR_LOGIN_REDIRECT_URL` | `"LOGIN_REDIRECT_URL"` | `string` or `func(*http.Request) string` | `DEFAULT_LOGIN_REDIRECT_URL` (`"/"`) | Destination after login/register when no `?next=` query parameter is present, or when visiting login while already authenticated. |
| `APPVAR_LOGIN_VIEW_REVERSE_URL` | `"LOGIN_VIEW_REVERSE_URL"` | `string` | `"auth:login"` | Named route used by `autherrors.RegisterHook` and `AuthView` to build login redirect URLs. |

`AuthApplication.Check` emits two startup warnings via [Checks](../../checks.md) when defaults are left unset:

- `auth.login_redirect_url_not_set`: `APPVAR_LOGIN_REDIRECT_URL` is not configured; falls back to `"/"`.
- `auth.login_reverse_url_not_set`: `APPVAR_LOGIN_VIEW_REVERSE_URL` is not configured; falls back to `"auth:login"`.

---

## Middleware, Request Helpers, and Sessions

During routing, `auth` installs `AddUserMiddleware()` globally on the multiplexer. During `Init`, it also registers a `tpl.RequestProcessors` function that sets `"User"` to `authentication.Retrieve(rc.Request())` on every request-backed template context.

```go
const SESSION_COOKIE_NAME = "user_authentication"

func AddUserMiddleware() middleware.Middleware
func UserFromRequest(r *http.Request) *User
func UnAuthenticatedUser() *User
func Login(r *http.Request, u *User) (*User, error)
func Logout(r *http.Request) error
```

- **`AddUserMiddleware()`**: Wrapped with `django.NonStaticMiddleware` so static asset requests skip DB lookups. For each non-static request, it reads `SESSION_COOKIE_NAME` (`"user_authentication"`) from the SCS session, queries `GetUserQuerySet()` by `ID` while preloading `Permissions` and `Groups.Permissions` via `queries.NoJoins(...)`, sets `user.IsLoggedIn = true`, and attaches the user to the request context. If no valid session or matching row exists, it attaches `UnAuthenticatedUser()`.
- **`UnAuthenticatedUser()`**: Returns an inactive, logged-out `*User` with `Username: "AnonymousUser"` and `Email: &drivers.Email{Address: "<unknown>"}`.
- **`UserFromRequest(r)`**: Returns the `*User` already stored on the request context by `AddUserMiddleware`, or loads it from the session if called before the middleware ran. Panics if SCS session middleware is missing.
- **`Login(r, u)`**: Renews the session token (`session.RenewToken()`), sets `u.IsLoggedIn = true`, writes `u.ID` under `SESSION_COOKIE_NAME`, fires `core.SIGNAL_USER_LOGGED_IN`, sets `u.LastLogin = time.Now()`, and registers a `session.AddFinalizer` callback that persists `LastLogin` before the response finishes unless the caller already saved `u` (`!u.State().Changed(true)`).
- **`Logout(r)`**: Returns `autherrors.ErrNoSession` if no session is present, otherwise calls `session.Destroy()` and fires `core.SIGNAL_USER_LOGGED_OUT`.

### Testing Without Session Middleware (`testing_auth`)

Passing `-tags testing_auth` compiles `auth_sessions_testing_auth.go` instead of `auth_sessions.go`. In `testing_auth` mode, `Login` and `Logout` emit the same `core` signals and toggle `u.IsLoggedIn`, while skipping `sessions.Retrieve(r)` and `LastLogin` database finalizers so unit tests can run without HTTP session middleware.

---

## Built-in Routes, Template Requirement, and Reusable Views

When `APPVAR_REGISTER_AUTH_URLS` is `true` (the default), `auth` registers the following named routes under `/auth`:

| Route Name | Methods | Path | Notes |
| :--- | :--- | :--- | :--- |
| `auth:login` | `GET`, `POST` | `/auth/login` | Redirects `302` to `LOGIN_REDIRECT_URL` if already authenticated |
| `auth:register` | `GET`, `POST` | `/auth/register` | Mounted only when `APPVAR_ALLOW_USER_REGISTER` is `true` |
| `auth:logout` | `POST` | `/auth/logout` | `LogoutView`; rejects direct `GET` calls with `405` (router returns `404`) |

### Template Requirement Caveat

`contrib/auth` does **not** bundle any HTML templates. The built-in handlers render `"auth/login.tmpl"` and `"auth/register.tmpl"` under `BaseTemplateKey: "auth"`, and pass `"FormURL"`, `"NextURL"`, and `"Form"` in the template context.

If you keep `APPVAR_REGISTER_AUTH_URLS` enabled and serve `/auth/login` or `/auth/register`, you must register matching templates with `tpl.Add` (see [Rendering](../../rendering.md)):

```go
tpl.Add(tpl.Config{
	AppName: "auth",
	FS:      authTplFS,
	Matches: filesystem.MatchPrefix("auth/"),
})
```

```html
<!-- templates/auth/login.tmpl -->
{{ define "login" }}
<form method="post" action="{{ .Get "FormURL" }}">
    <input type="hidden" name="next" value="{{ .Get "NextURL" }}">
    {{ $f := .Get "Form" }}
    {{ range $f.ErrorList }}
        <div class="form-error">{{ . }}</div>
    {{ end }}
    {{ $f.AsP }}
    <button type="submit">Log in</button>
</form>
{{ end }}
```

If your application only uses the [Admin](../admin/app.md) login page or custom routes, set `auth.APPVAR_REGISTER_AUTH_URLS: false` in `django.Configure(...)`.

### Reusable `AuthView` Helpers

You can mount login, registration, and logout views on your own routes with custom templates or `OnSuccess` callbacks:

```go
type AuthView[T forms.Form] struct {
	*views.BaseView
	OnSuccess func(w http.ResponseWriter, req *http.Request, form T) error
	// ...
}

func LoginView(baseView *views.BaseView, opts ...func(forms.Form)) *AuthView[*BaseUserForm]
func RegisterView(baseView *views.BaseView, cfg RegisterFormConfig, opts ...func(forms.Form)) *AuthView[*BaseUserForm]
func LogoutView(w http.ResponseWriter, r *http.Request)
```

For example, mounting a custom registration endpoint that asks for first/last names and logs the user in automatically:

```go
regView := auth.RegisterView(&views.BaseView{
	BaseTemplateKey: "myapp",
	TemplateName:    "myapp/register.tmpl",
}, auth.RegisterFormConfig{
	AlwaysAllLoginFields: true,
	AskForNames:          true,
	AutoLogin:            true,
})

app.Mux.Handle(mux.ANY, "/accounts/register", views.Serve(regView), "accounts_register")
```

After a valid submission, `AuthView` runs `OnSuccess` (if set) and redirects `303 See Other` to `?next=` or `APPVAR_LOGIN_REDIRECT_URL`. `LogoutView` requires `POST`, logs the user out if authenticated, and redirects `303 See Other` to `?next=` or the reversed `Auth.LoginReverseURL`.

---

## Forms and Password Fields

### `BaseUserForm`, `UserLoginForm`, and `UserRegisterForm`

```go
type RegisterFormConfig struct {
	AlwaysAllLoginFields bool
	AutoLogin            bool
	AskForNames          bool
	IsInactive           bool
}

type BaseUserForm struct {
	*forms.BaseForm
	Request  *http.Request
	Instance *User
	// ...
}

func (f *BaseUserForm) SetRequest(r *http.Request)
func (f *BaseUserForm) Save() (*User, error)
func (f *BaseUserForm) Login() error

func UserLoginForm(r *http.Request, formOpts ...func(forms.Form)) *BaseUserForm
func UserRegisterForm(r *http.Request, registerConfig RegisterFormConfig, formOpts ...func(forms.Form)) *BaseUserForm
```

- **`UserLoginForm`**: Adds `"email"` (when `Auth.LoginWithEmail` is `true`) or `"username"` (when `false`) plus `"password"`. All field validation errors are masked behind `autherrors.ErrInvalidLogin` via `fields.Protect(...)`, and invalid submissions fire `core.SIGNAL_LOGIN_FAILED`. Calling `form.Login()` verifies credentials with `CheckPassword`, requires `user.IsActive` (`autherrors.ErrIsActive`), and calls `auth.Login`.
- **`UserRegisterForm`**: Adds `"email"` and/or `"username"` (both if `AlwaysAllLoginFields` is `true`), optional `"firstName"`/`"lastName"` (`AskForNames`), `"password"`, and `"passwordConfirm"`. Validates password matching (`autherrors.ErrPwdNoMatch`), fires `SignalPreValidateLogonField`, checks for duplicate email/username (`autherrors.ErrUserExists`), fires `SignalPostValidateLogonField`, and deletes password values from `InvalidDefaults` on invalid submits. Calling `form.Save()` hashes and persists the user (`IsAdministrator = false`, `IsActive = !IsInactive`), optionally logging them in if `AutoLogin` is `true`.

### `PasswordField` and Character Validation

`auth` registers `NewPasswordField` as the default `attrs` form field for `*auth.Password`, and exposes character validation flags and validators for custom forms:

```go
type PasswordCharacterFlag uint8

const (
	ChrFlagSpecial PasswordCharacterFlag = 1 << iota
	ChrFlagDigit
	ChrFlagLower
	ChrFlagUpper
	ChrFlagAll = ChrFlagSpecial | ChrFlagDigit | ChrFlagLower | ChrFlagUpper
)

var ChrFlagDEFAULT = ChrFlagAll

type PasswordCharValidator struct {
	GenericError error
	Flags        PasswordCharacterFlag
}

type PasswordFieldOptions struct {
	Flags             PasswordCharacterFlag
	IsRegistering     bool
	UseDefaultOptions bool
	Options           []func(fields.Field)
}

type PasswordField struct {
	*fields.BaseField
	Validators []func(*Password) error
}

func NewPasswordField(config PasswordFieldOptions, opts ...func(fields.Field)) *PasswordField
func PasswordValidators(fn ...func(*Password) error) func(*PasswordField)
func ValidateCharacters(isRegister bool, flags PasswordCharacterFlag) func(fields.Field)
```

- Setting `UseDefaultOptions: true` adds `fields.MinLength(8)` and `fields.MaxLength(64)`.
- Setting `Flags` attaches `ValidateCharacters(isRegistering, flags)`. When `isRegister == false`, `PasswordCharValidator.GenericError` is set to `autherrors.ErrGenericAuthFail` so login forms do not leak complexity rules.
- `PasswordCharValidator.Validate` requires at least one matching character without consisting exclusively of that class (for example, `"ALLUPPER"` triggers `ErrPwdCasingUpper`, `"12345678"` triggers `ErrPwdDigits`), and always rejects whitespace (`ErrPwdSpaces`).

---

## Error Handling (`contrib/auth/auth_errors`)

The `github.com/Nigel2392/go-django/contrib/auth/auth_errors` package centralizes auth errors and HTTP redirection when authentication fails:

```go
type AuthenticationError struct {
	Message string
	NextURL string
	Status  int
}

type AuthErrorHook func(w http.ResponseWriter, r *http.Request, app *django.Application, err *AuthenticationError) (written bool)

const AUTH_ERROR_HOOK = "auth.errors.AuthenticationError"

func Fail(code int, msg string, next ...string)
func RegisterHook(reversePath string)
func OnAuthenticationError(h AuthErrorHook)
```

Calling `autherrors.Fail(http.StatusUnauthorized, "Login required", r.URL.RequestURI())` panics with an `*AuthenticationError` (status `0` defaults to `401`). During `auth.Init`, `autherrors.RegisterHook(Auth.LoginReverseURL)` attaches a `django.HOOK_SERVER_ERROR` handler that:

1. Runs registered `OnAuthenticationError` hooks in order. If any returns `written == true`, handling stops (for example, [Admin](../admin/app.md) registers `RedirectLoginFailedToAdmin` to redirect `/admin/*` requests to `admin:login`, or an API app can return JSON `401`).
2. Otherwise reverses `reversePath` (default `"auth:login"`), appends `?next=<NextURL>` if present, and responds with `302 Found`.

Defined error constants in `autherrors`:

- `ErrPwdCasingUpper`, `ErrPwdCasingLower`, `ErrPwdDigits`, `ErrPwdSpaces`, `ErrPwdSpecial`
- `ErrInvalidLogin`, `ErrInvalidEmail`, `ErrInvalidUsername`, `ErrUserExists`, `ErrIsActive`
- `ErrPasswordInvalid`, `ErrPwdHashMismatch` (alias of `ErrPasswordInvalid`), `ErrPwdNoMatch`, `ErrGenericAuthFail`, `ErrNoSession`

---

## Signals

`contrib/auth` emits the request lifecycle signals defined in `src/core` (`github.com/Nigel2392/go-django/src/core`) plus form-level registration signals in `auth`:

| Signal | Payload Type | Emitted By |
| :--- | :--- | :--- |
| `core.SIGNAL_USER_LOGGED_IN` | `core.UserWithRequest` | `auth.Login` |
| `core.SIGNAL_USER_LOGGED_OUT` | `core.UserWithRequest` | `auth.Logout` (`User` is `nil`) |
| `core.SIGNAL_LOGIN_FAILED` | `*http.Request` | `UserLoginForm` on validation failure (and `admin.LoginHandler`) |
| `auth.SignalPreValidateLogonField` | `*auth.FormSignal` | `UserRegisterForm` before email/username DB uniqueness checks |
| `auth.SignalPostValidateLogonField` | `*auth.FormSignal` | `UserRegisterForm` after email/username DB uniqueness checks |

Note that `core` also defines `SIGNAL_BEFORE_USER_CREATE`, `SIGNAL_AFTER_USER_CREATE`, `SIGNAL_BEFORE_USER_UPDATE`, `SIGNAL_AFTER_USER_UPDATE`, `SIGNAL_BEFORE_USER_DELETE`, and `SIGNAL_AFTER_USER_DELETE` for custom user backends (used by [OAuth2](../oauth2.md)); `auth.User` itself does not emit those CRUD signals.

---

## Admin and Chooser Integration

During `Init`, `auth`:

- Calls `admin.ConfigureAuth` using `UserLoginForm` and `auth.Logout`.
- Registers the `"auth"` admin app (`MenuOrder: 995`, label `"Authentication and Authorization"`) with `User`, `users.Group`, and `users.Permission`.
- Registers a default modal chooser (`chooser.ChooserDefinition[*User]`) whose queryset excludes the currently logged-in user and searches across `Username`, `Email`, `FirstName`, and `LastName`.
- Customizes the admin edit form (`FormInit`) with a `"PasswordConfirm"` field so admins can keep an existing password hash untouched or enter and confirm a new password.

---

## Management Commands

The `auth` app registers three CLI commands (see [Commands](../../commands.md)):

### `createuser`

Creates a new `auth.User`. Supports positional arguments `[email] [username] [password]` and prompts interactively for missing fields while enforcing `PasswordCharValidator{Flags: ChrFlagAll}`.

- `-s`: Create as administrator (`IsAdministrator = true`).
- `-i`: Create as inactive (`IsActive = false`).

```bash
go run . createuser -s admin@example.com admin Secret123!
```

### `changeuser`

Updates `IsAdministrator` and `IsActive` on an existing user looked up interactively by email (when `APPVAR_AUTH_EMAIL_LOGIN` is `true`) or username (when `false`).

- `-s`: Set `IsAdministrator = true` (default `false`).
- `-i`: Set `IsActive = false` (default `true`).

```bash
go run . changeuser -s
```

### `set_password`

Prompts interactively for an email or username (matching `APPVAR_AUTH_EMAIL_LOGIN`) and a confirmed password validated against `ChrFlagAll`, then updates the user's hashed password in the database.

```bash
go run . set_password
```
