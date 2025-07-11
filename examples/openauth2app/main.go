package main

import (
	"context"
	"net/http"
	"os"

	"github.com/Nigel2392/go-django/examples/blogapp/blog"
	"github.com/Nigel2392/go-django/examples/todoapp/todos"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/editor"
	"github.com/Nigel2392/go-django/src/contrib/editor/features/images"
	_ "github.com/Nigel2392/go-django/src/contrib/editor/features/links"
	"github.com/Nigel2392/go-django/src/contrib/messages"
	"github.com/Nigel2392/go-django/src/contrib/openauth2"
	"github.com/Nigel2392/go-django/src/contrib/pages"
	"github.com/Nigel2392/go-django/src/contrib/reports"
	auditlogs "github.com/Nigel2392/go-django/src/contrib/reports/audit_logs"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"

	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles/fs"
	"github.com/Nigel2392/go-django/src/core/logger"

	"github.com/Nigel2392/go-django/src/contrib/session"

	_ "github.com/mattn/go-sqlite3"
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

type GitHubUser struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
	Email     string `json:"email"`
}

func main() {

	godotenv.Load("./.private/.env")

	var db, err = drivers.Open(context.Background(), "sqlite3", "./.private/openauth2app/db.sqlite3")
	if err != nil {
		panic(err)
	}

	var app = django.App(
		django.Configure(map[string]interface{}{
			django.APPVAR_ALLOWED_HOSTS: []string{"*"},
			// django.APPVAR_DEBUG:         false,
			django.APPVAR_HOST:            "127.0.0.1",
			django.APPVAR_PORT:            "8080",
			django.APPVAR_DATABASE:        db,
			django.APPVAR_RECOVERER:       false,
			migrator.APPVAR_MIGRATION_DIR: "./.private/openauth2app/migrations",
		}),
		django.AppLogger(&logger.Logger{
			Level:       logger.INF,
			OutputTime:  true,
			WrapPrefix:  logger.ColoredLogWrapper,
			OutputDebug: os.Stdout,
			OutputInfo:  os.Stdout,
			OutputWarn:  os.Stdout,
			OutputError: os.Stdout,
		}),
		django.Apps(
			session.NewAppConfig,
			messages.NewAppConfig,
			openauth2.NewAppConfig(openauth2.Config{
				AuthConfigurations: []openauth2.AuthConfig{
					{
						Provider:         "google",
						ProviderNiceName: "Google",
						DocumentationURL: "https://developers.google.com/identity/protocols/oauth2",
						ProviderLogoURL: func() string {
							return "https://t1.gstatic.com/faviconV2?client=SOCIAL&type=FAVICON&fallback_opts=TYPE,SIZE,URL&url=http://google.com&size=128"
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
						DataStructURL: "https://www.googleapis.com/oauth2/v3/userinfo",
						DataStructIdentifier: func(token *oauth2.Token, dataStruct interface{}) (string, error) {
							var user = dataStruct.(*GoogleUser)
							return user.Email, nil
						},
						DataStruct: &GoogleUser{},
						UserToString: func(user *openauth2.User, dataStruct interface{}) string {
							var googleUser = dataStruct.(*GoogleUser)
							return googleUser.Email
						},
					},
					{
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
						UserToString: func(user *openauth2.User, dataStruct interface{}) string {
							var u = dataStruct.(*GitHubUser)
							return u.Email
						},
					},
				},
				BaseCallbackURL:       "http://127.0.0.1:8080",
				UserDefaultIsDisabled: false,
				RedirectAfterLogin: func(user *openauth2.User, datastruct interface{}, r *http.Request) string {
					return django.Reverse("index")
				},
				RedirectAfterLogout: func(r *http.Request) string {
					return django.Reverse("index")
				},
			}),
			// auth.NewAppConfig,
			admin.NewAppConfig,
			pages.NewAppConfig,
			editor.NewAppConfig,
			todos.NewAppConfig,
			blog.NewAppConfig,
			reports.NewAppConfig,
			auditlogs.NewAppConfig,
			migrator.NewAppConfig,
			images.NewAppConfig(&images.Options{
				MediaBackend: fs.NewBackend(
					"./.web/__images__", 5,
				),
			}),
		),
	)

	pages.SetRoutePrefix("/pages")

	if err := app.Serve(); err != nil {
		panic(err)
	}
}
