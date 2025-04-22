# OpenAuth2

The `openauth2` app provides a simple way to authenticate users using OAuth2 providers like Google, Facebook, and GitHub.

It uses the `golang.org/x/oauth2` package to handle the OAuth2 flow and provides a way to authenticate users and store their profile information in the database.

Currently, it is not implemented to let a user sign in with different oauth2 providers - this is also not planned.

When a user who is not registered tries to log in with a provider, the app will create a new user in the database with the profile information retrieved from the provider.

Dependencies for the `openauth2` app are:

- [`sessions`](./sessions.md) - storing session state such as the user's id

**Table of Contents**

- [Configuring openauth2](#configuring-openauth2)
- [Installing the openauth2 app](#installing-the-openauth2-app)
- [Using the user's tokens](#using-the-users-tokens)
  - [Refreshing the access and refresh tokens](#refreshing-the-access-and-refresh-tokens)
  - [Instantiating a new client with the user's tokens](#instantiating-a-new-client-with-the-users-tokens)
- [Working with the database](#working-with-the-database)
  - [Registering a custom querier](#registering-a-custom-querier)
- [Commands](#commands)
  - [Creating a new user](#creating-a-new-user)
  - [Changing a user](#changing-a-user)

## Configuring openauth2

The `openauth2` app can be configured using the `openauth2.AuthConfig` struct.

This is basically a wrapper around the `golang.org/x/oauth2.Config` struct, and provides some additional functionality to handle the OAuth2 flow,
along with retrieving additional profile information from the provider.

We will show examples for Google and GitHub, but the same principles apply for most other providers.

```go
package main

import (
    "os"
    "golang.org/x/oauth2"
    "golang.org/x/oauth2/google"
    "golang.org/x/oauth2/github"
    "github.com/Nigel2392/go-django/src"
    "github.com/Nigel2392/go-django/src/contrib/session"
    "github.com/Nigel2392/go-django/src/contrib/openauth2"
    openauth2models "github.com/Nigel2392/go-django/src/contrib/openauth2/openauth2_models"
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
    // The actual name for the provider, mainly used internally, in the database and in URLs
    // 
    // Can not be more than 255 characters long!
    Provider:         "google",

    // A label for the provider, used in the UI
    ProviderNiceName: "Google",

    // A URL to the provider's documentation for OAuth2
    //
    // This is used for display on the admin's index view for the openauth2 app.
    DocumentationURL: "https://developers.google.com/identity/protocols/oauth2",

    // A URL to the provider's logo, which can be used in the UI
    //
    // It is currently only used in the admin login view, but it can be used in other places as well.
    ProviderLogoURL: func() string {
        return "https://t1.gstatic.com/faviconV2?client=SOCIAL&type=FAVICON&fallback_opts=TYPE,SIZE,URL&url=http://google.com&size=128"
    },

    // The golang.org/x/oauth2.Config struct, which is used to handle the OAuth2 flow
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

    // The URL to redirect to after the user has authenticated with the provider
    //
    // This is where profile data will be retrieved from the provider.
    DataStructURL: "https://www.googleapis.com/oauth2/v3/userinfo",

    // A way to retrieve the user's primary identifier, this is used to identify the user appropriately in the database.
    // 
    // The return value must not be more than 255 characters long, and must be unique for the provider.
    DataStructIdentifier: func(token *oauth2.Token, dataStruct interface{}) (string, error) {
        var user = dataStruct.(*GoogleUser)
        return user.Email, nil
    },

    // The struct to use for the profile data, this is used to unmarshal the JSON response from the provider.
    DataStruct: &GoogleUser{},

    // A nice way to represent a user as a string
    UserToString: func(user *openauth2models.User, dataStruct interface{}) string {
        var googleUser = dataStruct.(*GoogleUser)
        return googleUser.Email
    },

    // The access type to request from the provider.
    //
    // If this is nil, it will be set to "oauth2.AccessTypeOffline".
    AccessType: oauth2.AccessTypeOffline,

    // The state to use when generating the URL
    // with Oauth2.AuthCodeURL.
    //
    // If this is left empty, it will default to "state"
    State: "state",

    // ExtraParams are extra parameters to be set on the URL when
    // generating the url with Oauth2.AuthCodeURL
    ExtraParams: nil,
}

type GitHubUser struct {
    ID        int    `json:"id"`
    Login     string `json:"login"`
    AvatarURL string `json:"avatar_url"`
    Email     string `json:"email"`
}

var ConfigGithub = openauth2.AuthConfig{
    Provider:         "github",
    ProviderNiceName: "Github",
    DocumentationURL: "https://docs.github.com/en/apps/oauth-apps",
    ProviderLogoURL: func() string {
        return "https://github.githubassets.com/assets/GitHub-Mark-ea2971cee799.png"
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
    DataStructURL: "https://api.github.com/user",
    DataStructIdentifier: func(token *oauth2.Token, dataStruct interface{}) (string, error) {
        var user = dataStruct.(*GitHubUser)
        return user.Email, nil
    },
    DataStruct: &GitHubUser{},
    UserToString: func(user *openauth2models.User, dataStruct interface{}) string {
        var u = dataStruct.(*GitHubUser)
        return u.Email
    },
}

var OAuth2Config = openauth2.Config{
    // The base URL for the callback URL. This is used to generate the redirect URL for the OAuth2 provider.
    //
    // This should be the base URL of your application, e.g. "https://example.com/"
    BaseCallbackURL: "127.0.0.1:8080"

    // A list of authentication configurations for the providers.
    AuthConfigurations: []openauth2.AuthConfig{
        ConfigGoogle,
        ConfigGithub,
    },

    // If the user's state should be inactive by default.
    //  UserDefaultIsDisabled bool

    // A function to generate the default URL after the user has logged in.
    //
    // Note:
    //    If this is not set, the default URL will be "/".
    //    A redirect URL might also be stored in a HTTP-only cookie, if present the cookie's URL will be used instead.
    //  RedirectAfterLogin func(user *openauth2models.User, datastruct interface{}, r *http.Request) string

    // A function to generate the default URL after the user has logged out.
    //  RedirectAfterLogout func(r *http.Request) string
}
```

## Installing the openauth2 app

The `openauth2` app can be installed using the `openauth2.NewAppConfig` function.

This function the previously created `openauth2.Config` as an argument, and returns a new `django.AppConfig` object.

```go
package main

import (
    "database/sql"

    "github.com/Nigel2392/go-django/src"
    "github.com/Nigel2392/go-django/src/contrib/session"
    "github.com/Nigel2392/go-django/src/contrib/messages"
    "github.com/Nigel2392/go-django/src/contrib/openauth2"
    "github.com/Nigel2392/go-django/src/contrib/admin"

    _ "github.com/mattn/go-sqlite3"
)

func main() {
    var db, err = sql.Open("sqlite3", "example.db")
    if err != nil {
        panic(err)
    }

    var app = django.App(
        django.Configure(map[string]interface{}{
            django.APPVAR_ALLOWED_HOSTS: []string{"*"},
            django.APPVAR_HOST: "127.0.0.1",
            django.APPVAR_PORT: "8080",
            django.APPVAR_DATABASE: func() *sql.DB {
                return db
            }(),
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

In the above example, the `openauth2` app is installed using the `openauth2.NewAppConfig` function.

We have also installed the admin app, which can be used to edit a user's administrator and active status.

Creating a new user through the admin interface is not supported.

You can now navigate to [`http://127.0.0.1:8080/admin/login`](http://127.0.0.1:8080/admin/login) and log in with your Google or GitHub account.

A user will be automatically created in the database with the profile information retrieved from the provider.

This user will not have an administrator status by default, [this can be changed with a command](#changing-a-user).

## Using the user's tokens

### Refreshing the access and refresh tokens

There is a simple way for refreshing a users' access token and refresh token if need be.

This is done by calling the `RefreshToken` function and passing in a `openauth2models.User` struct.

```go
var (
    myUser = // ... Get the user from the database
    token *oauth2.Token
    err error
)

token, err = openauth2.RefreshToken(myUser)
// ...
```

### Instantiating a new client with the user's tokens

The http client which is instantiated with the user's tokens can be used to make requests to the provider's API.

It will also automatically refresh the access and refresh tokens, and update the user's tokens in the database.

```go
var (
    myUser = // ... Get the user from the database
    client *http.Client
    err error
)

client, err = openauth2.Client(myUser)
// ...
```

## Working with the database

The `openauth2.App` variable exposes a `QuerySet()` method, which returns a `openauth2models.Querier` object that can be used to query the database.

```go
// openauth2models.Querier
type Querier interface {
    Close() error
    WithTx(tx *sql.Tx) Querier

    RetrieveUsers(ctx context.Context, limit int32, offset int32, ordering ...string) ([]*User, error)
    RetrieveUserByID(ctx context.Context, id uint64) (*User, error)
    RetrieveUserByIdentifier(ctx context.Context, uniqueIdentifier string, providerName string) (*User, error)

    CreateUser(ctx context.Context, uniqueIdentifier string, providerName string, data json.RawMessage, accessToken string, refreshToken string, tokenType string, expiresAt time.Time, isAdministrator bool, isActive bool) (int64, error)
    DeleteUser(ctx context.Context, id uint64) error
    DeleteUsers(ctx context.Context, ids []uint64) error
    UpdateUser(ctx context.Context, providerName string, data json.RawMessage, accessToken string, refreshToken string, tokenType string, expiresAt time.Time, isAdministrator bool, isActive bool, iD uint64) error
}
```

### Registering a custom querier

If you'd like to use a different database engine, i.e. postgres, you can register a custom querier for your database.

A simple way to register a custom querier for your database is to do so with `openauth2models.Register`

```go
import (
    "context"
    "database/sql"
    openauth2models "github.com/Nigel2392/go-django/src/contrib/openauth2/openauth2_models"
)

func init() {
    openauth2models.Register(
        &MyCustomDriver{}, &dj_models.BaseBackend[openauth2models.Querier]{
            CreateTableQuery: `CREATE TABLE IF NOT EXISTS oauth2_users (
    id                    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT                                 COMMENT 'readonly:true',
    unique_identifier     VARCHAR(255) NOT NULL                                                   COMMENT 'readonly:true',
    provider_name         VARCHAR(255) NOT NULL                                                   COMMENT 'readonly:true',
    data                  JSON NOT NULL                                                           COMMENT 'readonly:true',
    access_token          TEXT NOT NULL                                                           COMMENT 'readonly:true',
    refresh_token         TEXT NOT NULL                                                           COMMENT 'readonly:true',
    token_type            VARCHAR(60) NOT NULL                                                    COMMENT 'readonly:true',
    expires_at            DATETIME NOT NULL                                                       COMMENT 'readonly:true',
    created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP                             COMMENT 'readonly:true',
    updated_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'readonly:true',
    is_administrator      BOOLEAN NOT NULL,
    is_active             BOOLEAN NOT NULL,
    PRIMARY KEY (id)
);

ALTER TABLE oauth2_users ADD UNIQUE INDEX (unique_identifier(255), provider_name(255));
ALTER TABLE oauth2_users ADD INDEX (provider_name(255));`,
            NewQuerier: func(db *sql.DB) (openauth2models.Querier, error) {
                return MyNewQuerier(db), nil
            },
            PreparedQuerier: func(ctx context.Context, db *sql.DB) (openauth2models.Querier, error) {
                return PrepareMyNewQuerier(ctx, db)
            },
        },
    )
}
```

## Commands

The `openauth2` app provides a way to create a new user or edit an existing user in the database using a django command.

Both these commands take the following optional arguments:

- `-i` - Wether the user you are about to create should be inactive
- `-s` - Wether your user should be an administrator

Both of these commands will ask you for the unique identifier of the user and a provider.

This is the same unique identifier that is used to identify the user in the database, which we have [configured in the `DataStructIdentifier` function](#configuring-openauth2).

The provider is the name of the provider, in our case either `google` or `github`.

### Creating a new user

Creating a new user can be done by calling your main.go file with the `createuser` command.

Example:

```bash
go run main.go createuser -s # Create a new superuser
```

The command will then create a new user in the database with the given unique identifier and provider.

### Changing a user

Changing a user can be done by calling your main.go file with the `changeuser` command.

Example:

```bash
go run main.go changeuser -s # Make an existing user a superuser
```
