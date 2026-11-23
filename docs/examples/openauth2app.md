# OpenAuth2 Application Example

The `openauth2app` example (`examples/openauth2app`) demonstrates OAuth2 authentication in Go-Django using [`contrib/openauth2`](../apps/oauth2.md) together with `session`, `messages`, `admin`, `pages`, `editor`, `documents`, `images`, `migrator`, and the `todos` and `blog` example apps.

## Table of Contents

- [Prerequisites and Environment Configuration](#prerequisites-and-environment-configuration)
- [Running the Example (Required Migrations)](#running-the-example-required-migrations)
- [OAuth2 Provider and Display Configuration](#oauth2-provider-and-display-configuration)
- [Mounted Apps and Storage](#mounted-apps-and-storage)
- [Notes and Caveats](#notes-and-caveats)

---

## Prerequisites and Environment Configuration

`examples/openauth2app/main.go` loads environment variables from `./.private/.env` using `godotenv.Load("./.private/.env")`. Create `.private/.env` inside `examples/openauth2app` with your Google OAuth2 credentials:

```env
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret
```

Configure your Google OAuth2 client in the Google Cloud Console to allow the callback URL matching `BaseCallbackURL`:

```text
http://127.0.0.1:8080/openauth2/google/callback
```

---

## Running the Example (Required Migrations)

Because `openauth2app` installs `migrator.NewAppConfig` with `migrator.APPVAR_MIGRATION_DIR: "./.private/openauth2app/migrations"`, you **must** generate and apply database migrations before starting the server.

From `examples/openauth2app`:

```bash
mkdir -p .private
go run . makemigrations
go run . migrate
go run .
```

If running from the repository root (`go run ./examples/openauth2app ...`), ensure `.private/.env` is present in the working directory you run from.

Once the server is running on `http://127.0.0.1:8080`:

- Sign in via the admin login page at `http://127.0.0.1:8080/admin/login/`.
- Use the `createuser` or `changeuser` CLI commands from `contrib/openauth2` to grant administrator access (`-s`), for example:

  ```bash
  go run . changeuser -s
  ```

See [OpenAuth2](../apps/oauth2.md) and [CLI](../cli.md) for user management command flags.

---

## OAuth2 Provider and Display Configuration

`examples/openauth2app/main.go` configures `openauth2.NewAppConfig(openauth2.Config{...})` using the current `ProviderInfo` and `DataConfig` structs:

- **`BaseCallbackURL`**: Set to `"http://127.0.0.1:8080"`.
- **`ProviderInfo` (`openauth2.ConfigInfo`)**: Sets `Provider: "google"`, `ProviderLabel: "Google"`, `ProviderLogoURL`, `DocumentationURL`, and `PrivacyPolicyURL`. (A commented-out GitHub provider template is also included in `main.go`.)
- **`Oauth2` (`*oauth2.Config`)**: Configures `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, scopes (`"openid"`, `"email"`, `"profile"`), and `google.Endpoint`.
- **`DataConfig` (`openauth2.DataConfig`)**:
  - `Object: &GoogleUser{}` and `DetailsURL: "https://www.googleapis.com/oauth2/v3/userinfo"`.
  - `GetUniqueIdentifier`: Extracts `user.Email` as the unique database identifier.
  - `EditDisplayOrder`: Orders the JSON fields shown on the admin user edit page (`picture`, `email`, `verified_email`, `name`, `given_name`, `first_name`, `last_name`, `family_name`, `locale`, `sub`).
  - `EditDisplayLabels`: Supplies translatable field labels (`trans.S(...)`).
  - `EditDisplayWidgets`: Supplies read-only display widgets for each profile field, including a custom `imageWidget` that renders an `<img>` element for `"picture"`.
- **`RedirectAfterLogin` / `RedirectAfterLogout`**: Configured to resolve `django.Reverse("index")` (see caveat below).

---

## Mounted Apps and Storage

In addition to `openauth2`, `main.go` configures:

- **Database**: SQLite at `./.private/openauth2app/db.sqlite3`.
- **Media backend**: Local filesystem backend (`mediafiles/fs`) at `./.private/openauth2app/media`, registered as `"fs"` and set as the default backend.
- **Mounted apps**:
  - Core & auth/admin: `session`, `messages`, `openauth2`, `admin` (note `contrib/auth` is omitted because `openauth2` provides the user model).
  - CMS & reports: `pages`, `contrib/pages/settings`, `editor`, `reports`, `auditlogs`, `migrator`.
  - Media apps: `documents` (`MediaDir: "__documents__"`) and `images` (`MediaDir: "__images__"`).
  - Example sub-apps: `todos.NewAppConfig` and `blog.NewAppConfig`, with `pages.SetRoutePrefix("/pages")`.

See [Apps](../apps.md), [Filesystem](../filesystem.md), and [Pages](../apps/pages/readme.md).

---

## Notes and Caveats

- **`django.Reverse("index")` panics as configured**: Both `RedirectAfterLogin` and `RedirectAfterLogout` call `django.Reverse("index")`, but none of the apps registered in `examples/openauth2app/main.go` defines a route named `"index"`. Completing OAuth2 login or logout triggers `assert.Fail("Error reversing URL index: ...")` unless you mount a route named `"index"` or update those callbacks (for example, to `"/admin/"` or `django.Reverse("todos:list")`).
- **`blog` is mounted without `blocks`, `revisions`, or `searchqueries`**: `examples/openauth2app/main.go` registers `blog.NewAppConfig`, yet omits `blocks.NewAppConfig` (needed for `blocks` static assets, templates, and `render_block`/`block_value` template functions used by `BlogPage`), `revisions.NewAppConfig`, and `_ "github.com/Nigel2392/go-django/contrib/search/backends/searchqueries"`. Add those to `main.go` if you render or search `BlogPage` entries inside this example; see the [Blog App Example](./blog.md).
- **Working-directory-relative paths**: `./.private/.env`, `./.private/openauth2app/db.sqlite3`, `./.private/openauth2app/media`, and `./.private/openauth2app/migrations` are all resolved relative to the current working directory. Run commands consistently from `examples/openauth2app`.
