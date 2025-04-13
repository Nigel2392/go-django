package openauth2

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"net/http"
	"strings"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	openauth2models "github.com/Nigel2392/go-django/src/contrib/openauth2/openauth2_models"
	_ "github.com/Nigel2392/go-django/src/contrib/openauth2/openauth2_models/mysqlc"
	_ "github.com/Nigel2392/go-django/src/contrib/openauth2/openauth2_models/sqlitec"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/mux"
)

const (
	USER_ID_SESSION_KEY = "openauth2_user_id"

	// TODO: Implement functionality later - model should store:
	// - UserID
	// - ProviderID (or name)
	// - AccessToken
	// - RefreshToken
	// - ExpiresIn
	// - Scope
	// - TokenType
	// - CreatedAt
	// - UpdatedAt
	//
	// The use of this struct is to easily access the token data, and refresh
	// the access token if needed.
	OAUTH_MODEL_SESSION_KEY = "openauth2_model"
)

var (
	App *OpenAuth2AppConfig

	//go:embed assets/*
	templates embed.FS
)

type Config struct {
	BaseCallbackURL       string
	AuthConfigurations    []AuthConfig
	UserDefaultIsDisabled bool
	RedirectAfterLogin    func(datastruct interface{}, r *http.Request) string
	RedirectAfterLogout   func(r *http.Request) string
}

type OpenAuth2AppConfig struct {
	*apps.DBRequiredAppConfig
	Config   *Config
	_cnfs    map[string]AuthConfig
	queryset openauth2models.Querier
}

func (oa *OpenAuth2AppConfig) Querier() openauth2models.Querier {
	return oa.queryset
}

func NewAppConfig(cnf Config) django.AppConfig {
	App = &OpenAuth2AppConfig{
		DBRequiredAppConfig: apps.NewDBAppConfig(
			"openauth2",
		),
		Config: &cnf,
		_cnfs:  make(map[string]AuthConfig),
	}

	App.Deps = []string{"session"}

	App.Init = func(settings django.Settings, db *sql.DB) error {
		if len(App.Config.AuthConfigurations) == 0 {
			return errors.New("OpenAuth2: No providers configured")
		}

		var backend, err = openauth2models.GetBackend(db.Driver())
		if err != nil {
			return err
		}

		err = backend.CreateTable(db)
		if err != nil {
			return err
		}

		queryset, err := backend.NewQuerySet(db)
		if err != nil {
			return err
		}
		App.queryset = &openauth2models.SignalsQuerier{Querier: queryset}

		autherrors.RegisterHook("auth2:login")

		admin.ConfigureAuth(admin.AuthConfig{
			GetLoginHandler: App.AdminLoginHandler,
			Logout:          Logout,
		})

		tpl.Add(tpl.Config{
			AppName: "openauth2",
			FS: filesystem.NewMultiFS(
				admin.AdminSite.TemplateConfig.FS,
				filesystem.Sub(
					templates, "assets/templates",
				),
			),
			Matches: filesystem.MatchAnd(
				filesystem.MatchPrefix("oauth2"),
				filesystem.MatchSuffix(".tmpl"),
			),
			Bases: admin.AdminSite.TemplateConfig.Bases,
			Funcs: admin.AdminSite.TemplateConfig.Funcs,
		})

		return nil
	}

	App.Ready = func() error {

		for _, c := range App.Config.AuthConfigurations {
			if c.Provider == "" {
				logger.Warnf("OpenAuth2: Missing provider name for %q, proider will not be used", c.Oauth2.ClientID)
				continue
			}

			if c.Oauth2 == nil {
				logger.Warnf("OpenAuth2: Missing Oauth2 config for %q, provider will not be used", c.Provider)
				continue
			}

			if c.Oauth2.RedirectURL == "" {
				c.Oauth2.RedirectURL = fmt.Sprintf(
					"%s%s",
					strings.TrimSuffix(App.Config.BaseCallbackURL, "/"),
					django.Reverse("auth2:provider:callback", c.Provider),
				)
			}

			App._cnfs[c.Provider] = c
		}

		return nil
	}

	App.Routing = func(m django.Mux) {
		m.Use(
			AddUserMiddleware(),
		)

		var base = m.Any("/auth2", nil, "auth2")
		base.Any("/logout", mux.NewHandler(App.LogoutHandler), "logout")
		base.Any("/login", mux.NewHandler(App.LoginHandler), "login")

		var rt = base.Any("/<<provider>>", App.handler(App.AuthHandler), "provider")
		rt.Any("/callback", App.handler(App.CallbackHandler), "callback")
	}
	return App
}

func (a *OpenAuth2AppConfig) handler(h func(http.ResponseWriter, *http.Request, *AuthConfig)) http.HandlerFunc {
	var fn = func(w http.ResponseWriter, r *http.Request) {
		var vars = mux.Vars(r)
		var provider = vars.Get("provider")
		if provider == "" {
			except.Fail(
				http.StatusBadRequest,
				"Missing provider name in URL",
			)
			return
		}

		var authConfig, ok = a._cnfs[provider]
		if !ok {
			except.Fail(
				http.StatusBadRequest,
				"Unknown provider name in URL",
			)
			return
		}

		h(w, r, &authConfig)
	}

	return http.HandlerFunc(fn)
}
