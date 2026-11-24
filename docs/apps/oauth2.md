# OpenAuth2

The `openauth2` app provides OAuth2 authentication for Go-Django using `golang.org/x/oauth2`. When a new user authenticates through a configured provider, `openauth2` creates an `openauth2.User` record in the database with their profile payload and tokens.

Users are uniquely scoped per `(UniqueIdentifier, ProviderName)` pair—logging in with different providers creates separate user records.

**Dependencies:**

- [`session`](./sessions.md) — stores the authenticated user ID in the session.
- `django.APPVAR_DATABASE` — stores users, tokens, groups, and permissions.
- [`admin`](./admin/app.md) *(optional)* — provides admin login integration, user management, groups, permissions, and a user chooser.

See also the [OpenAuth2 example guide](../examples/openauth2app.md).

## Table of Contents

- [Configuring OpenAuth2](#configuring-openauth2)
- [Installing the OpenAuth2 App](#installing-the-openauth2-app)
- [Routes and Authentication Flow](#routes-and-authentication-flow)
- [Admin, Groups, Permissions, and Chooser](#admin-groups-permissions-and-chooser)
- [Using User Tokens and Clients](#using-user-tokens-and-clients)
- [User Model, Helpers, and Extension Interfaces](#user-model-helpers-and-extension-interfaces)
- [Commands](#commands)
- [Signals and System Checks](#signals-and-system-checks)
- [Caveats](#caveats)

---

## Configuring OpenAuth2

`openauth2` is configured with an `openauth2.Config` containing one or more `openauth2.AuthConfig` entries. Each `AuthConfig` groups provider metadata in `ProviderInfo` (`openauth2.ConfigInfo`), profile-fetching settings in `DataConfig` (`openauth2.DataConfig`), and the underlying `*oauth2.Config`.

```go
package main

import (
	"net/http"
	"os"

	"github.com/Nigel2392/go-django/contrib/openauth2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

type GoogleUser struct {
	ID        string `json:"sub"`
	Email     string `json:"email"`
	Verified  bool   `json:"verified_email"`
	Name      string `json:"name"`
	FirstName string `json:"given_name"`
	LastName  string `json:"family_name"`
	Picture   string `json:"picture"`
	Locale    string `json:"locale"`
}

var ConfigGoogle = openauth2.AuthConfig{
	ProviderInfo: openauth2.ConfigInfo{
		// Internal provider identifier used in DB records and URLs (max 255 chars).
		Provider: "google",
		// Display label (string or func(context.Context) string for translations).
		ProviderLabel: "Google",
		// Optional provider logo URL callback.
		ProviderLogoURL: func(r *http.Request) string {
			return "https://t1.gstatic.com/faviconV2?client=SOCIAL&type=FAVICON&fallback_opts=TYPE,SIZE,URL&url=http://google.com&size=128"
		},
		// Optional provider documentation and privacy policy URL callbacks.
		DocumentationURL: func(r *http.Request) string {
			return "https://developers.google.com/identity/protocols/oauth2"
		},
		PrivacyPolicyURL: func(r *http.Request) string {
			return "https://policies.google.com/privacy"
		},
	},
	Oauth2: &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		Scopes: []string{
			"openid",
			"email",
			"profile",
		},
		Endpoint: google.Endpoint,
	},
	DataConfig: openauth2.DataConfig{
		// JSON endpoint used to fetch the user's profile after token exchange.
		DetailsURL: "https://www.googleapis.com/oauth2/v3/userinfo",
		// Returns the unique identifier for this user under this provider (max 255 chars).
		GetUniqueIdentifier: func(token *oauth2.Token, dataStruct interface{}) (string, error) {
			user := dataStruct.(*GoogleUser)
			return user.Email, nil
		},
		// Template struct cloned per scan via reflection for JSON decoding.
		Object: &GoogleUser{},
		// Optional ordering, labels, and widgets for the admin JSONDetailPanel.
		EditDisplayOrder:  nil,
		EditDisplayLabels: nil,
		EditDisplayWidgets: nil,
	},
	UserToString: func(user *openauth2.User, dataStruct interface{}) string {
		googleUser := dataStruct.(*GoogleUser)
		return googleUser.Email
	},
	AccessType:  oauth2.AccessTypeOffline,
	State:       "state",
	ExtraParams: nil,
}

type GitHubUser struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
	Email     string `json:"email"`
}

var ConfigGithub = openauth2.AuthConfig{
	ProviderInfo: openauth2.ConfigInfo{
		Provider:      "github",
		ProviderLabel: "GitHub",
		ProviderLogoURL: func(r *http.Request) string {
			return "https://github.githubassets.com/assets/GitHub-Mark-ea2971cee799.png"
		},
		DocumentationURL: func(r *http.Request) string {
			return "https://docs.github.com/en/apps/oauth-apps"
		},
	},
	Oauth2: &oauth2.Config{
		ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		Scopes: []string{
			"read:user",
			"user:email",
		},
		Endpoint: github.Endpoint,
	},
	DataConfig: openauth2.DataConfig{
		DetailsURL: "https://api.github.com/user",
		GetUniqueIdentifier: func(token *oauth2.Token, dataStruct interface{}) (string, error) {
			user := dataStruct.(*GitHubUser)
			return user.Email, nil
		},
		Object: &GitHubUser{},
	},
	UserToString: func(user *openauth2.User, dataStruct interface{}) string {
		u := dataStruct.(*GitHubUser)
		return u.Email
	},
}

var OAuth2Config = openauth2.Config{
	// Base URL used when auto-generating OAuth2 callback URLs.
	BaseCallbackURL: "http://127.0.0.1:8080",
	AuthConfigurations: []openauth2.AuthConfig{
		ConfigGoogle,
		ConfigGithub,
	},
	// Optional configuration:
	// UserDefaultIsDisabled: false,
	// RedirectAfterLogin: func(user *openauth2.User, datastruct interface{}, r *http.Request) string { return "/" },
	// RedirectAfterLogout: func(r *http.Request) string { return "/" },
}
```

If `c.Oauth2.RedirectURL` is left empty on an `AuthConfig`, `openauth2` automatically populates it during `Ready` by combining `BaseCallbackURL` with `django.Reverse("auth2:provider:callback", provider)` (e.g., `http://127.0.0.1:8080/auth2/google/callback`).

## Installing the OpenAuth2 App

Pass `openauth2.NewAppConfig(OAuth2Config)` to `django.Apps(...)`:

```go
package main

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/messages"
	"github.com/Nigel2392/go-django/contrib/openauth2"
	"github.com/Nigel2392/go-django/contrib/session"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := drivers.Open(context.Background(), "sqlite3", "example.db")
	if err != nil {
		panic(err)
	}

	app := django.App(
		django.Configure(map[string]any{
			django.APPVAR_ALLOWED_HOSTS: []string{"*"},
			django.APPVAR_HOST:          "127.0.0.1",
			django.APPVAR_PORT:          "8080",
			django.APPVAR_DATABASE:      db,
		}),
		django.Apps(
			session.NewAppConfig,
			messages.NewAppConfig,
			openauth2.NewAppConfig(OAuth2Config),
			admin.NewAppConfig,
		),
	)

	if err := app.Serve(); err != nil {
		panic(err)
	}
}
```

`openauth2.NewAppConfig` returns a `*migrator.MigratorAppConfig` bundling migrations for `&openauth2.User{}` as well as `Group`, `Permission`, `UserGroup`, `GroupPermission`, and `UserPermission` from `contrib/auth/users`. If `migrator` is not installed, `openauth2` automatically creates only the `openauth2_users` table during initialization.

Once started with `admin` installed, navigate to `http://127.0.0.1:8080/admin/login` to authenticate with a configured provider. New users default to non-administrator status; use the [`changeuser` command](#changing-a-user) to grant administrator access.

## Routes and Authentication Flow

`openauth2` registers `AddUserMiddleware()` globally and mounts the following routes under `/auth2`:

| Path | Reverse Name | Handler | Notes |
| --- | --- | --- | --- |
| `/auth2/login` | `"auth2:login"` | `App.LoginHandler` | Also hooked via `autherrors.RegisterHook("auth2:login")` |
| `/auth2/logout` | `"auth2:logout"` | `App.LogoutHandler` | Supports `?next=` or `Config.RedirectAfterLogout` |
| `/auth2/<<provider>>` | `"auth2:provider"` | `App.AuthHandler` | Stores optional `?next=` in a cookie and redirects to provider |
| `/auth2/<<provider>>/callback` | `"auth2:provider:callback"` | `App.CallbackHandler` | Exchanges code, fetches profile, creates/loads user, calls `Login` |

### Session and Request Helpers

- `openauth2.AddUserMiddleware() middleware.Middleware`: Attaches the current user to requests via `authentication.AddUserMiddleware`.
- `openauth2.UserFromRequest(r *http.Request) *User`: Returns the authenticated `*User` from request context or loads active users via session key `USER_ID_SESSION_KEY` (`"openauth2_user_id"`).
- `openauth2.UnauthenticatedUser() *User`: Returns a non-authenticated `*User` value.
- `openauth2.Login(r *http.Request, u *User) (*User, error)`: Renews the session token, sets `USER_ID_SESSION_KEY`, emits `core.SIGNAL_USER_LOGGED_IN`, and updates `LastLogin`.
- `openauth2.Logout(r *http.Request) error`: Destroys the session and emits `core.SIGNAL_USER_LOGGED_OUT` (returns `autherrors.ErrNoSession` if no session is present).

## Admin, Groups, Permissions, and Chooser

When the `admin` app is installed, `openauth2`:

- Configures admin authentication (`admin.ConfigureAuth`) with `App.AdminLoginHandler` (`oauth2/admin/login.tmpl`) and `openauth2.Logout`.
- Registers the `"openauth2"` admin app (with `providerComponent` showing configured providers on the app index page) and registers three models:
  - `users` (`&openauth2.User{}`): `DisallowCreate` and `DisallowDelete` are enabled; supports inline toggling of `IsActive` and `IsAdministrator`, searching on `ID`, `UniqueIdentifier`, `ProviderName`, and `Data`, and inspecting decoded provider JSON via `JSONDetailPanel` (driven by `DataConfig.EditDisplayOrder`, `EditDisplayLabels`, and `EditDisplayWidgets`).
  - `groups` (`&users.Group{}`)
  - `permissions` (`&users.Permission{}`)
- Registers a `chooser.ChooserDefinition[*openauth2.User]` that excludes the currently authenticated user.
- Registers a `contenttypes.ContentTypeDefinition` for `&openauth2.User{}` using `UserToString` for instance labels.

## Using User Tokens and Clients

`openauth2` provides helpers built around `golang.org/x/oauth2.TokenSource` that automatically persist refreshed tokens back to the database:

- `openauth2.TokenSource(u *User) *ProviderTokenSource`: Returns a `*ProviderTokenSource` wrapping `AuthConfig.TokenSource` (or `AuthConfig.GetTokenSource` if provided) in a mutex-guarded saving token source.
- `openauth2.RefreshTokens(u *User) (*oauth2.Token, error)`: Returns a valid token for `u`, refreshing and persisting it to the database if expired (returns `ErrUserNil` if `u == nil`).
- `openauth2.Client(u *User) (*http.Client, error)`: Returns an `*http.Client` backed by `TokenSource(u)` that refreshes expired tokens automatically.

### Refreshing Tokens

```go
var myUser *openauth2.User // Retrieved from database or request
token, err := openauth2.RefreshTokens(myUser)
if err != nil {
	// Handle error
}
```

### Instantiating an Authenticated HTTP Client

```go
var myUser *openauth2.User // Retrieved from database or request
client, err := openauth2.Client(myUser)
if err != nil {
	// Handle error
}
```

Note that `AuthConfig.TokenSource(ctx, token)` calls `AuthConfig.GetTokenSource` if set or falls back to `AuthConfig.Oauth2.TokenSource`; calling `AuthConfig.TokenSource` directly does not wrap the result with automatic database saving—use `openauth2.TokenSource(u)` or `openauth2.Client(u)` for persistence.

## User Model, Helpers, and Extension Interfaces

### `openauth2.User`

`openauth2.User` embeds `models.Model` (`table:"openauth2_users"`) and `users.Base`, and enforces uniqueness across `("UniqueIdentifier", "ProviderName")`:

- `Token() *oauth2.Token` / `SetToken(token *oauth2.Token)`: Converts between stored token fields (`AccessToken`, `RefreshToken`, `TokenType`, `ExpiresAt`) and `*oauth2.Token`.
- `Provider() (*AuthConfig, error)`: Looks up the `AuthConfig` matching `ProviderName`.
- `ContentObject() (interface{}, error)`: Decodes `Data` into a clone of `DataConfig.Object` (cached per `User` instance) and calls `UserBinder.BindUser` if implemented.
- `SetContext(ctx context.Context) *User` / `Context() context.Context`: Associates a `context.Context` with the user (defaults to `context.Background()`).
- `String() string`: Formats `"<UniqueIdentifier> (<ProviderName>)"`.

### Extension Interfaces

Provider data structs (`DataConfig.Object`) may optionally implement:

- **`UserBinder`**:
  ```go
  type UserBinder interface {
  	BindUser(user *User) error
  }
  ```
  Called by `User.ContentObject()` after JSON decoding.
- **`UserFormDefiner`**:
  ```go
  type UserFormDefiner interface {
  	UserFormFields(user *User) []attrs.Field
  }
  ```
  When implemented by `DataConfig.Object` and the user has `ProviderName`, `UniqueIdentifier`, and `Data` populated, its fields are appended to `User.FieldDefs`.

### Database Query Helpers

```go
func CreateUser(ctx context.Context, user *User) (*User, error)
func GetUserByID(ctx context.Context, id uint64) (*User, error)
func GetUserByIdentifier(ctx context.Context, provider, identifier string) (*User, error)
```

For example:

```go
user, err := openauth2.GetUserByID(context.Background(), 1)
```

## Commands

`openauth2` registers two CLI commands (both accept `-s` for administrator/superuser status and `-i` for inactive status):

### Creating a New User

```bash
go run main.go createuser -s
```

Prompts interactively for the user's `UniqueIdentifier` and `Provider` name, then inserts a new `openauth2.User`.

### Changing a User

```bash
go run main.go changeuser -s
```

Prompts interactively for the user's `UniqueIdentifier` and `Provider` name, then updates `IsAdministrator` and `IsActive` inside a transaction.

## Signals and System Checks

- **Signals**: `User` lifecycle hooks emit `core.SIGNAL_BEFORE_USER_CREATE`, `core.SIGNAL_AFTER_USER_CREATE`, `core.SIGNAL_BEFORE_USER_UPDATE`, `core.SIGNAL_AFTER_USER_UPDATE`, `core.SIGNAL_BEFORE_USER_DELETE` (ID payload), and `core.SIGNAL_AFTER_USER_DELETE` (ID payload). `Login` and `Logout` emit `core.SIGNAL_USER_LOGGED_IN` and `core.SIGNAL_USER_LOGGED_OUT`.
- **System checks**: `OpenAuth2AppConfig.Check` runs `DBRequiredAppConfig.Check` and emits `checks.Error` messages if `AuthConfigurations` is empty (`openauth2.no_providers`) or if any provider is missing `ProviderInfo.Provider` (`openauth2.provider_missing`), `Oauth2` (`openauth2.oauth2_missing`), `ClientID` (`openauth2.client_id_missing`), or `ClientSecret` (`openauth2.client_secret_missing`).

## Caveats

- **`LoginHandler` template is not bundled**: `/auth2/login` (`App.LoginHandler`) renders `"oauth2/login.tmpl"` with `BaseTemplateKey: "oauth2"`. The bundled assets in `contrib/openauth2` only include `oauth2/admin/login.tmpl` and `oauth2/admin/components/providers.tmpl`, so if you route users to `/auth2/login` directly you must supply and register an `oauth2/login.tmpl` template (or link directly to `/auth2/<provider>` or `/admin/login`).
- **Existing-user tokens and profile data are not updated on re-login**: In `CallbackHandler`, when an existing user signs in again, `Update(user)` runs on the previously fetched `user` without copying the newly fetched `rawData` or exchanged `token` fields onto `user` first.
- **Static `State` parameter**: `AuthHandler` defaults `State` to `"state"` (or the static `AuthConfig.State` value), and `CallbackHandler` does not validate a `state` query parameter on the callback request.
- **Redirect cookie and logout fallback**: The `openauth2.redirectURL` cookie storing `?next=` is created with `Secure: true` (requiring HTTPS or localhost secure-context handling), and `LogoutHandler` redirects to `""` if both `?next=` and `Config.RedirectAfterLogout` are empty.
