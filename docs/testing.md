# Testing (`djester` and `httptest`)

Go-Django includes a dedicated testing module at `github.com/Nigel2392/go-django/djester`, plus a `(*django.Application).TestServe` helper in `src/django_httptest.go`. All test helpers in `djester` and `src/django_httptest.go` are gated behind the `test` build tag.

---

## Running Tests and Build Tags

Because `djester` and `TestServe` use `//go:build test`, always pass `-tags test` when compiling or running tests that import them:

```bash
go test -tags test ./...
```

The `djester/testdb` package also supports optional, mutually exclusive database backend build tags (defaulting to in-memory SQLite when none is specified):

- Default / `-tags "test sqlite"`: In-memory SQLite (`sqlite3`)
- `-tags "test postgres"`: PostgreSQL on `127.0.0.1:5432`
- `-tags "test mysql"`: MySQL on `127.0.0.1:3306`
- `-tags "test mariadb"`: MariaDB on `127.0.0.1:3307`
- `-tags "test mysql_local"`: Embedded in-memory MySQL server (`github.com/dolthub/go-mysql-server`) started on a free localhost port

---

## `djester.Tester` and HTTP Testing

The core `github.com/Nigel2392/go-django/djester` package sets up a `django.Application`, boots an `httptest.Server`, wraps HTTP client calls with cookie-jar support, and executes structured `Test` suites.

### Core Types

```go
package djester

type BaseTB interface { /* *testing.T / *testing.B subset */ }
type TB interface {
	BaseTB
	Run(name string, f func(TB)) bool
}

func TW(t *testing.T) TB
func BW(b *testing.B) TB

type TesterAuth struct {
	UnauthenticatedUser func() authentication.User
	Users               map[string]authentication.User
}

type Tester struct {
	Verbose      bool
	BeforeSetup  func(dj *Tester) error
	Database     Database
	Auth         *TesterAuth
	Flags        []django.AppFlag
	Settings     map[string]any
	ExtraOptions []django.Option
	Apps         []AppInitFuncOrAppConfig
	Handlers     []Handler
	Tests        []Test
	ExtraModels  []attrs.Definer
	App          *django.Application
}
```

### `Tester` Methods

- **`Setup(t TB) error`**: Configures settings, opens `Database` (if `Engine != ""`), mounts any `Handlers` / `ExtraModels` under an automatic `"djester"` app, appends `django.FlagSkipCmds` and `django.FlagSkipChecks`, initializes `App`, attaches `Auth` middleware and `/djester/auth/login/<<map_key>>` + `/djester/auth/logout` routes if `Auth != nil`, and starts the test server via `App.TestServe(true)`.
- **`Close() error`**: Closes `DB()`, shuts down `Server()`, and calls `App.Quit()`.
- **`Server() *django.HTTPTestServer`** / **`Client() *django.HTTPTestClient`** / **`DB() drivers.Database`**: Accessors for the underlying test server, cookie-enabled HTTP client, and database connection.
- **`Get(url, headers, params)`** / **`GetJson(url, headers, params, scanTo)`**: Performs GET requests and optionally decodes JSON responses.
- **`Post(url, headers, params, body)`** / **`PostForm(url, headers, params, form)`** / **`PostFile(url, headers, params, fileName, file)`** / **`PostJson(url, headers, params, body, scanTo)`**: Performs POST requests with raw bodies, URL-encoded or multipart forms, file uploads, or JSON payloads.
- **`Login(userKey string)`** / **`Logout()`**: Authenticates or clears the session user against `Tester.Auth.Users`.
- **`Assert(t BaseTB, verbose bool) Assertion`**: Returns an assertion helper bound to `t`.
- **`Test(t *testing.T)`** / **`Bench(b *testing.B)`**: Calls `Setup(...)` and runs each item in `Tests` as a Go subtest or sub-benchmark.

### Authentication Requirements

When setting `Tester.Auth`:

1. You **must** include `session.NewAppConfig` (`github.com/Nigel2392/go-django/contrib/session`) in `Apps`—otherwise `Setup` fatals with `"cannot initialize authentication without app 'session'"`.
2. Set `django.APPVAR_DISABLE_NOSURF: true` in `Settings` so `Login`, `Logout`, and `Post*` helpers work without submitting CSRF tokens.

### `TestResponse` and DOM Assertions

HTTP methods return `*djester.TestResponse`, which wraps `*http.Response` with cached body reading, JSON decoding, and `goquery` HTML assertions:

```go
type TestResponse struct {
	*http.Response
	// ...
}

func (r *TestResponse) Assert(t BaseTB, verbose bool) ResponseAssertion
func (t *TestResponse) JSON(scanTo interface{}) error
func (t *TestResponse) HTML() (string, error)
func (t *TestResponse) DOM() (*goquery.Document, error)
```

Available `HTMLAssertFunc` helpers:

- **`HasElement(selector string)`**: Ensures at least one element matches `selector`.
- **`DoesNotHaveElement(selector string)`**: Ensures zero elements match `selector`.
- **`HasText(selector, expectedText string)`**: Ensures a matched element's text contains `expectedText`.
- **`HasAttribute(selector, attrName, expectedValue string)`**: Ensures a matched element has `attrName == expectedValue`.
- **`HasElementCount(selector string, expectedCount int)`**: Ensures exact match count.

```go
//go:build test

package myapp_test

import (
	"net/http"
	"testing"

	"github.com/Nigel2392/go-django/contrib/session"
	"github.com/Nigel2392/go-django/djester"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware/authentication"
)

type testUser struct{ auth, admin bool }

func (u testUser) IsAuthenticated() bool { return u.auth }
func (u testUser) IsAdmin() bool         { return u.admin }

func TestDashboard(t *testing.T) {
	d := &djester.Tester{
		Settings: map[string]any{
			django.APPVAR_ALLOWED_HOSTS:  []string{"*"},
			django.APPVAR_DISABLE_NOSURF: true,
		},
		Apps: []djester.AppInitFuncOrAppConfig{session.NewAppConfig},
		Handlers: []djester.Handler{
			{
				Name:   "home",
				Method: "GET",
				Route:  "/",
				Handler: mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "text/html")
					w.Write([]byte(`<h1 class="title">Welcome</h1>`))
				}),
			},
		},
		Auth: &djester.TesterAuth{
			UnauthenticatedUser: func() authentication.User { return testUser{} },
			Users: map[string]authentication.User{
				"admin": testUser{auth: true, admin: true},
			},
		},
		Tests: []djester.Test{
			&djester.BasicTest{
				Label: "Renders welcome heading",
				Function: func(dj *djester.Tester, t *testing.T) {
					_, err := dj.Login("admin")
					dj.Assert(t, true).Assert(err == nil, "login failed: %v", err)

					resp, err := dj.Get("/", nil, nil)
					dj.Assert(t, true).Assert(err == nil, "GET / failed: %v", err)
					defer resp.Body.Close()

					resp.Assert(t, true).AssertHTML(
						djester.HasElement("h1.title"),
						djester.HasText("h1.title", "Welcome"),
					)
				},
			},
		},
	}

	t.Cleanup(func() { _ = d.Close() })
	d.Test(t)
}
```

---

## `djester/objects`

The `github.com/Nigel2392/go-django/djester/objects` package provides reusable `djester.Test` implementations for form fields and ORM queries:

- **`FieldTest[FORMFIELD forms.Field]`**: Tests form field validation (`ExpectsValid`, `ExpectedErrors`), `CleanedData`, default values, and rendered widget HTML (`ExpectedHTML`), with helpers:
  - `AssertInputPresent(fieldName)`
  - `AssertInputAttributes(fieldName, checks...)` with `AttributePresent`, `AttributeNotPresent`, `AttributeValueEQ`, `AttributeValueNEQ`
  - `AssertInputValueMatches(fieldName, provided...)`
- **`QuerySetTest`**: Wraps database tests in `queries.StartTransaction(t.Context())`, runs `Create` (either a `func(context.Context, *testing.T)` or a slice/nested slice of `attrs.Definer` models inserted via `BulkCreate`), executes `Execute(dj, t, ctx)`, runs optional `Teardown`, and rolls back the transaction at the end so changes are reverted. Note that benchmarks on both `FieldTest` and `QuerySetTest` call `b.Skipf(...)`.

---

## `djester/quest`

The `github.com/Nigel2392/go-django/djester/quest` package provides schema and fixture helpers for model tests:

- **`Table[T TB](t T, model ...attrs.Definer) *DBTables[T]`**: Registers models with `attrs`, resets definitions, and prepares `migrator.ModelTable` entries using `django.APPVAR_DATABASE` (or an in-memory SQLite database if `django.Global` is uninitialized). Call `.Create()` and `.Drop()` (drops in reverse table order).
- **`CreateObjects[T attrs.Definer](t LogFatal, objects []T)`**: Bulk-creates `objects` via `queries.GetQuerySet` and returns `(created, deleteFn)`.
- **`StartTransaction[TEST djester.BaseTB](t TEST, database ...string)`**: Starts a `queries` transaction and automatically registers `tx.Rollback` with `t.Cleanup`.
- **`MySQLDatabase(cnf DatabaseConfig) (QuestDatabase, error)`** (`//go:build test && mysql_local`): Spins up an in-memory `go-mysql-server` instance (`DB()`, `Start()`, `Stop()`), defaulting to `tcp` on `127.0.0.1:13306` with user `root` and no password.

---

## `djester/testdb`

The `github.com/Nigel2392/go-django/djester/testdb` package exports `ENGINE` and `Open() (which string, db drivers.Database)` using the selected build tag:

| Build Tag | `ENGINE` | Target / Credentials |
| :--- | :--- | :--- |
| Default / `sqlite` | `"sqlite3"` | `file::memory:?cache=shared&_loc=auto&mode=memory` |
| `postgres` | `"postgres"` | `127.0.0.1:5432`, DB `django-test`, `root` / `my-secret-pw` |
| `mysql` | `"mysql"` | `127.0.0.1:3306`, `root` / `my-secret-pw` (creates & drops an isolated `django_test_<ULID>` DB) |
| `mariadb` | `"mariadb"` | `127.0.0.1:3307`, DB `django-test`, `root` / `my-secret-pw` |
| `mysql_local` | `"mysql_local"` | Embedded `quest.MySQLDatabase` on a free port on `127.0.0.1` |

---

## `(*django.Application).TestServe`

If you want to run your own `httptest.Server` without `djester.Tester`, `src/django_httptest.go` (behind `//go:build test`) exposes `TestServe`:

```go
func (a *Application) TestServe(autoStart bool) (*HTTPTestServer, error)
func (s *HTTPTestServer) Client() *HTTPTestClient
```

`TestServe` initializes `a` if needed, wraps `a.Mux` with `a.middlewareBuiltins()`, creates an `httptest.Server` (started when `autoStart` is `true`), executes `HOOK_SERVER_STARTUP` hooks, and hooks `HOOK_SERVER_SHUTDOWN` + `server.Close()` into `a.Quit()`.

---

## Caveats

- **Package-level singletons**: `django.App(...)` initializes and reuses the global `django.Global` singleton when non-nil, and `testdb.Open()` caches its database handle in a package-level singleton. Avoid running multiple `Tester` setups with conflicting apps or settings in parallel within the same test package.
- **Manual `Close()` cleanup**: `Tester.Test` and `Tester.Bench` do not automatically call `d.Close()`. Register `t.Cleanup(func() { _ = d.Close() })` (or `defer d.Close()`) if you want database connections, `httptest.Server`, and shutdown hooks cleaned up after the test finishes.
- **`-tags test` is required**: Compiling tests that reference `djester`, `djester/quest`, or `(*django.Application).TestServe` without `-tags test` fails with undefined symbol errors.
