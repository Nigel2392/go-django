package openauth2

import (
	"bytes"
	"context"
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
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/views/list"
	"github.com/Nigel2392/mux"
)

const (
	USER_ID_SESSION_KEY = "openauth2_user_id"

	ErrUnknownProvider errs.Error = "Unknown provider"
)

var (
	App *OpenAuth2AppConfig

	//go:embed assets/*
	assets embed.FS
)

type Config struct {
	// The base URL for the callback URL. This is used to generate the redirect URL for the OAuth2 provider.
	//
	// This should be the base URL of your application, e.g. "https://example.com/"
	BaseCallbackURL string

	// A list of authentication configurations for the providers.
	AuthConfigurations []AuthConfig

	// If the user's state should be inactive by default.
	UserDefaultIsDisabled bool

	// A function to generate the default URL after the user has logged in.
	//
	// Note:
	//  If this is not set, the default URL will be "/".
	// 	A redirect URL might also be stored in a HTTP-only cookie, if present the cookie's URL will be used instead.
	RedirectAfterLogin func(user *openauth2models.User, datastruct interface{}, r *http.Request) string

	// A function to generate the default URL after the user has logged out.
	RedirectAfterLogout func(r *http.Request) string
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

	App.Cmd = []command.Command{
		command_create_user,
		command_change_user,
	}

	App.ModelObjects = []attrs.Definer{
		&openauth2models.User{},
	}

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

		staticfiles.AddFS(
			filesystem.Sub(
				assets, "assets/static",
			),
			filesystem.MatchAnd(
				filesystem.MatchPrefix("oauth2/"),
				filesystem.MatchSuffix(".css"),
			),
		)

		tpl.Add(tpl.Config{
			AppName: "openauth2",
			FS: filesystem.NewMultiFS(
				filesystem.Sub(
					assets, "assets/templates",
				),
				admin.AdminSite.TemplateConfig.FS,
			),
			Matches: filesystem.MatchOr(
				filesystem.MatchAnd(
					filesystem.MatchPrefix("oauth2"),
					filesystem.MatchSuffix(".tmpl"),
				),
				admin.AdminSite.TemplateConfig.Matches,
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

	contenttypes.Register(&contenttypes.ContentTypeDefinition{
		ContentObject:  &openauth2models.User{},
		GetLabel:       trans.S("User"),
		GetPluralLabel: trans.S("Users"),
		GetInstanceLabel: func(a any) string {
			var u = a.(*openauth2models.User)
			var providerConfig, ok = App._cnfs[u.ProviderName]
			if !ok {
				return u.String()
			}

			if providerConfig.UserToString != nil {
				var dataStruct, err = providerConfig.ScanStruct(
					bytes.NewReader(u.Data),
				)
				if err != nil {
					return u.String()
				}
				return providerConfig.UserToString(u, dataStruct)
			}

			return u.String()
		},
		GetInstance: func(i interface{}) (interface{}, error) {
			var u, err = attrs.CastToNumber[uint64](i)
			if err != nil {
				return nil, err
			}

			instance, err := App.queryset.RetrieveUserByID(
				context.Background(), u,
			)
			return instance, err
		},
		GetInstances: func(amount, offset uint) ([]interface{}, error) {
			var instances, err = App.queryset.RetrieveUsers(
				context.Background(), int32(amount), int32(offset),
			)
			return attrs.InterfaceList(instances), err
		},
	})

	admin.RegisterAdminAppPageComponent("openauth2", newProviderComponent(App))

	admin.RegisterApp(
		"openauth2",
		admin.AppOptions{
			RegisterToAdminMenu: true,
			EnableIndexView:     true,
			AppLabel:            trans.S("OpenAuth2"),
			AppDescription: trans.S(
				"OpenAuth2 is an authentication backend for Go-Django. It allows you to authenticate users using OAuth2 providers such as Google, Facebook, GitHub, etc.",
			),
			MenuLabel: trans.S("Authentication"),
			MenuOrder: 1000,
		},
		admin.ModelOptions{
			RegisterToAdminMenu: true,
			Name:                "OAuth2User",
			Model:               &openauth2models.User{},
			MenuLabel:           trans.S("Users"),
			DisallowCreate:      true,
			DisallowDelete:      true,
			AddView: admin.FormViewOptions{
				ViewOptions: admin.ViewOptions{},
			},
			EditView: admin.FormViewOptions{
				ViewOptions: admin.ViewOptions{},
			},
			ListView: admin.ListViewOptions{
				PerPage: 20,
				ViewOptions: admin.ViewOptions{
					Labels: map[string]func() string{
						"ProviderName": trans.S("Provider"),
					},
					Fields: []string{
						"ID",
						"UniqueIdentifier",
						"ProviderName",
						"HasRefreshToken",
						"TokenType",
						"IsAdministrator",
						"IsActive",
						"CreatedAt",
						"UpdatedAt",
					},
				},
				Format: map[string]func(v any) any{
					"TokenType": func(v any) any {
						return strings.ToUpper(v.(string))
					},
				},
				Columns: map[string]list.ListColumn[attrs.Definer]{
					"UniqueIdentifier": list.TitleFieldColumn(
						list.FieldColumn[attrs.Definer](
							trans.S("Unique Identifier"),
							"UniqueIdentifier",
						),
						func(r *http.Request, defs attrs.Definitions, row attrs.Definer) string {
							return django.Reverse(
								"admin:apps:model:edit",
								"openauth2", "OAuth2User",
								row.(*openauth2models.User).ID,
							)
						},
					),
					"HasRefreshToken": list.FuncColumn(
						trans.S("Has Refresh Token"),
						func(r *http.Request, defs attrs.Definitions, row attrs.Definer) interface{} {
							var u = row.(*openauth2models.User)
							return u.RefreshToken != ""
						},
					),
				},
			},
			//	DeleteView: admin.DeleteViewOptions{
			//
			//	},
		},
	)

	return App
}

func (a *OpenAuth2AppConfig) Provider(name string) (*AuthConfig, error) {
	var authConfig, ok = a._cnfs[name]
	if !ok {
		return nil, ErrUnknownProvider
	}
	return &authConfig, nil
}

func (a *OpenAuth2AppConfig) Providers() []AuthConfig {
	return a.Config.AuthConfigurations
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
