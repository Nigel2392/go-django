package openauth2

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"strings"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/admin/chooser"
	"github.com/Nigel2392/go-django/src/contrib/admin/components/columns"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/contrib/auth/users"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/checks"
	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/views/list"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware/authentication"
)

const (
	USER_ID_SESSION_KEY = "openauth2_user_id"

	ErrUnknownProvider errs.Error = "Unknown provider"
)

var (
	App *OpenAuth2AppConfig

	//go:embed assets/*
	assets embed.FS

	//go:embed migrations/*
	migrationFS embed.FS
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
	RedirectAfterLogin func(user *User, datastruct interface{}, r *http.Request) string

	// A function to generate the default URL after the user has logged out.
	RedirectAfterLogout func(r *http.Request) string
}

type OpenAuth2AppConfig struct {
	*apps.DBRequiredAppConfig
	Config *Config
	_cnfs  map[string]AuthConfig
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
		&User{},
		&users.Group{},
		&users.Permission{},
		&users.UserGroup{},
		&users.GroupPermission{},
		&users.UserPermission{},
	}

	App.Init = func(settings django.Settings, db drivers.Database) error {
		//if len(App.Config.AuthConfigurations) == 0 {
		//	return errors.New("OpenAuth2: No providers configured")
		//}

		if !django.AppInstalled("migrator") {
			var schemaEditor, err = migrator.GetSchemaEditor(db.Driver())
			if err != nil {
				return fmt.Errorf("failed to get schema editor: %w", err)
			}

			var table = migrator.NewModelTable(&User{})
			if err := schemaEditor.CreateTable(context.Background(), table, true); err != nil {
				return fmt.Errorf("failed to create pages table: %w", err)
			}

			for _, index := range table.Indexes() {
				if err := schemaEditor.AddIndex(context.Background(), table, index, true); err != nil {
					return fmt.Errorf("failed to create index %s: %w", index.Name(), err)
				}
			}
		}

		autherrors.RegisterHook("auth2:login")

		staticfiles.AddFS(
			filesystem.Sub(
				assets, "assets/static",
			),
			filesystem.MatchAnd(
				filesystem.MatchPrefix("oauth2/"),
				filesystem.MatchSuffix(".css"),
			),
		)

		// Register everything required to the admin if it is installed.
		if django.AppInstalled("admin") {
			admin.ConfigureAuth(admin.AuthConfig{
				GetLoginHandler: App.AdminLoginHandler,
				Logout:          Logout,
			})

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

			admin.RegisterAdminAppPageComponent("openauth2", newProviderComponent(App))

			admin.RegisterApp(
				"openauth2",
				admin.AppOptions{
					RegisterToAdminMenu: true,
					EnableIndexView:     true,
					AppLabel:            trans.S("Authentication and Authorization"),
					AppDescription: trans.S(
						"OpenAuth2 is an authentication backend for Go-Django. It allows you to authenticate users using OAuth2 providers such as Google, Facebook, GitHub, etc.",
					),
					MenuLabel: trans.S("OAuth 2"),
					MenuOrder: 995,
					MenuIcon: func() string {
						return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-shield-exclamation" viewBox="0 0 16 16">
    <!-- The MIT License (MIT) -->
    <!-- Copyright (c) 2011-2024 The Bootstrap Authors -->
	<path d="M5.338 1.59a61 61 0 0 0-2.837.856.48.48 0 0 0-.328.39c-.554 4.157.726 7.19 2.253 9.188a10.7 10.7 0 0 0 2.287 2.233c.346.244.652.42.893.533q.18.085.293.118a1 1 0 0 0 .101.025 1 1 0 0 0 .1-.025q.114-.034.294-.118c.24-.113.547-.29.893-.533a10.7 10.7 0 0 0 2.287-2.233c1.527-1.997 2.807-5.031 2.253-9.188a.48.48 0 0 0-.328-.39c-.651-.213-1.75-.56-2.837-.855C9.552 1.29 8.531 1.067 8 1.067c-.53 0-1.552.223-2.662.524zM5.072.56C6.157.265 7.31 0 8 0s1.843.265 2.928.56c1.11.3 2.229.655 2.887.87a1.54 1.54 0 0 1 1.044 1.262c.596 4.477-.787 7.795-2.465 9.99a11.8 11.8 0 0 1-2.517 2.453 7 7 0 0 1-1.048.625c-.28.132-.581.24-.829.24s-.548-.108-.829-.24a7 7 0 0 1-1.048-.625 11.8 11.8 0 0 1-2.517-2.453C1.928 10.487.545 7.169 1.141 2.692A1.54 1.54 0 0 1 2.185 1.43 63 63 0 0 1 5.072.56"/>
  	<path d="M7.001 11a1 1 0 1 1 2 0 1 1 0 0 1-2 0M7.1 4.995a.905.905 0 1 1 1.8 0l-.35 3.507a.553.553 0 0 1-1.1 0z"/>
</svg>`
					},
				},
				admin.ModelOptions{
					RegisterToAdminMenu: true,
					Name:                "users",
					Model:               &User{},
					MenuLabel:           trans.S("Users"),
					DisallowCreate:      true,
					DisallowDelete:      true,
					AddView: admin.FormViewOptions{
						ViewOptions: admin.ViewOptions{},
					},
					EditView: admin.FormViewOptions{
						Panels: []admin.Panel{
							admin.TabbedPanel(
								admin.PanelTab(
									trans.S("General"),
									admin.TitlePanel(admin.RowPanel(
										admin.FieldPanel("ID"),
										admin.FieldPanel("UniqueIdentifier"),
									)),
									admin.FieldPanel("ProviderName"),
									admin.FieldPanel("CreatedAt"),
									admin.FieldPanel("UpdatedAt"),
								),
								admin.PanelTab(
									trans.S("Authentication"),
									admin.FieldPanel("TokenType"),
									admin.FieldPanel("AccessToken"),
									admin.FieldPanel("RefreshToken"),
									admin.FieldPanel("ExpiresAt"),
								),
								admin.PanelTab(
									trans.S("Details"),
									&admin.AlertPanel{
										Type:  admin.ALERT_INFO,
										Label: trans.S("User Data"),
										HTML:  trans.S("User data retrieved from the OAuth2 provider."),
									},
									&admin.JSONDetailPanel{
										FieldName: "Data",
										Ordering: func(r *http.Request, fields map[string]forms.BoundField) []string {
											var providerField, _ = fields["ProviderName"]
											var providerName = providerField.Value()
											return App.getProviderDataFieldOrder(providerName.(string))
										},
										Labels: func(r *http.Request, fields map[string]forms.BoundField) map[string]any {
											var providerField, _ = fields["ProviderName"]
											var providerName = providerField.Value()
											return App.getProviderDataLabels(providerName.(string))
										},
										Widgets: func(r *http.Request, fields map[string]forms.BoundField) map[string]widgets.Widget {
											var providerField, _ = fields["ProviderName"]
											var providerName = providerField.Value()
											return App.getProviderDataWidgets(providerName.(string))
										},
									},
								),
								admin.PanelTab(
									trans.S("Authorization"),
									admin.RowPanel(
										admin.FieldPanel("IsAdministrator"),
										admin.FieldPanel("IsActive"),
									),
									admin.FieldPanel("Groups"),
									admin.FieldPanel("Permissions"),
								),
							),
						},
						ViewOptions: admin.ViewOptions{},
					},
					ListView: admin.ListViewOptions{
						PerPage: 20,
						Search: &admin.SearchOptions{
							ListFields: []string{
								"UniqueIdentifier",
								"Provider",
								"TokenType",
								"IsAdministrator",
								"IsActive",
								"HasRefreshToken",
								"CreatedAt",
								"UpdatedAt",
							},

							Fields: []admin.SearchField{
								{
									Name:   "ID",
									Lookup: expr.LOOKUP_EXACT,
								},
								{
									Name:   "UniqueIdentifier",
									Lookup: expr.LOOKUP_ICONTANS,
								},
								{
									Name:   "ProviderName",
									Lookup: expr.LOOKUP_EXACT,
								},
								{
									Name:   "Data",
									Lookup: expr.LOOKUP_EXACT,
								},
							},
						},
						ViewOptions: admin.ViewOptions{
							Labels: map[string]func(context.Context) string{
								"ProviderName": trans.S("Provider"),
							},
							Fields: []string{
								"ID",
								"UniqueIdentifier__Edit",
								"Provider",
								"TokenType",
								"IsAdministrator",
								"IsActive",
								"HasRefreshToken",
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
							"Provider": list.FuncColumn[attrs.Definer](
								trans.S("Provider"),
								func(r *http.Request, defs attrs.Definitions, row attrs.Definer) interface{} {
									var user = row.(*User)
									var provider, ok = App._cnfs[user.ProviderName]
									if !ok {
										return user.ProviderName
									}

									label, ok := trans.GetText(r.Context(), provider.ProviderInfo.ProviderLabel)
									if ok {
										return label
									}

									return provider.ProviderInfo.Provider

								},
							),
							"HasRefreshToken": list.BooleanColumn[attrs.Definer](
								trans.S("Has Refresh Token"),
								func(r *http.Request, defs attrs.Definitions, row attrs.Definer) bool {
									return row.(*User).RefreshToken != ""
								},
							),
							"IsActive": list.BooleanColumn[attrs.Definer](
								trans.S("Is Active"),
								func(r *http.Request, defs attrs.Definitions, row attrs.Definer) bool {
									return row.(*User).IsActive
								},
							),
							"IsAdministrator": list.BooleanColumn[attrs.Definer](
								trans.S("Is Admin"),
								func(r *http.Request, defs attrs.Definitions, row attrs.Definer) bool {
									return row.(*User).IsActive
								},
							),
							"CreatedAt": columns.TimeSinceColumn[attrs.Definer](
								trans.S("Created"),
								"CreatedAt",
							),
							"UpdatedAt": columns.TimeSinceColumn[attrs.Definer](
								trans.S("Last Updated"),
								"UpdatedAt",
							),
							"UniqueIdentifier__Edit": list.TitleFieldColumn(
								list.FieldColumn[attrs.Definer](
									trans.S("Unique Identifier"),
									"UniqueIdentifier",
								),
								func(r *http.Request, defs attrs.Definitions, row attrs.Definer) string {
									return django.Reverse(
										"admin:apps:model:edit",
										"openauth2", "users",
										row.(*User).ID,
									)
								},
							),
						},
					},
					//	DeleteView: admin.DeleteViewOptions{
					//
					//	},
				},
				admin.ModelOptions{
					MenuLabel:           trans.S("Groups"),
					Name:                "groups",
					Model:               &users.Group{},
					RegisterToAdminMenu: true,
					MenuOrder:           2,
				},
				admin.ModelOptions{
					MenuLabel:           trans.S("Permissions"),
					Name:                "permissions",
					Model:               &users.Permission{},
					RegisterToAdminMenu: true,
					MenuOrder:           3,
				},
			)

			chooser.Register(&chooser.ChooserDefinition[*User]{
				Title: trans.S("User Chooser"),
				Model: &User{},
				PreviewString: func(ctx context.Context, instance *User) string {
					return instance.UniqueIdentifier
				},
				ListPage: &chooser.ChooserListPage[*User]{
					Fields: []string{
						"ID",
						"UniqueIdentifier",
						"Provider",
						"IsAdministrator",
						"IsActive",
						"CreatedAt",
					},
					SearchFields: []admin.SearchField{
						{
							Name:   "ID",
							Lookup: expr.LOOKUP_EXACT,
						},
						{
							Name:   "UniqueIdentifier",
							Lookup: expr.LOOKUP_ICONTANS,
						},
						{
							Name:   "ProviderName",
							Lookup: expr.LOOKUP_EXACT,
						},
						{
							Name:   "Data",
							Lookup: expr.LOOKUP_EXACT,
						},
					},
					QuerySet: func(r *http.Request, model *User) *queries.QuerySet[*User] {
						var currentUser = authentication.Retrieve(r)
						var user = currentUser.(*User)
						return queries.GetQuerySet(&User{}).
							Filter(expr.Q("ID", user.ID).Not(true)).
							OrderBy("UniqueIdentifier")
					},
				},
			})
		}

		return nil
	}

	App.Ready = func() error {

		for _, c := range App.Config.AuthConfigurations {
			if c.ProviderInfo.Provider == "" {
				continue
			}

			if c.Oauth2 == nil {
				continue
			}

			if c.Oauth2.RedirectURL == "" {
				c.Oauth2.RedirectURL = fmt.Sprintf(
					"%s%s",
					strings.TrimSuffix(App.Config.BaseCallbackURL, "/"),
					django.Reverse("auth2:provider:callback", c.ProviderInfo.Provider),
				)
			}

			App._cnfs[c.ProviderInfo.Provider] = c
		}

		return nil
	}

	App.Routing = func(m mux.Multiplexer) {
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
		ContentObject:  &User{},
		GetLabel:       trans.S("User"),
		GetPluralLabel: trans.S("Users"),
		GetInstanceLabel: func(a any) string {
			var u = a.(*User)
			var providerConfig, ok = App._cnfs[u.ProviderName]
			if !ok {
				return u.String()
			}

			if providerConfig.UserToString != nil {
				var dataStruct, err = u.ContentObject()
				if err != nil {
					return u.String()
				}
				return providerConfig.UserToString(u, dataStruct)
			}

			return u.String()
		},
	})

	return &migrator.MigratorAppConfig{
		AppConfig: App,
		MigrationFS: filesystem.Sub(
			migrationFS,
			"migrations/openauth2",
		),
	}
}

func (a *OpenAuth2AppConfig) getProviderDataFieldOrder(providerName string) []string {
	var order []string
	var providerConfig, ok = a._cnfs[providerName]
	if !ok {
		return order
	}

	if providerConfig.DataFieldOrder != nil {
		return providerConfig.DataFieldOrder
	}

	return order
}

func (a *OpenAuth2AppConfig) getProviderDataLabels(providerName string) map[string]any {
	var labels = make(map[string]any)
	var providerConfig, ok = a._cnfs[providerName]
	if !ok {
		return labels
	}

	if providerConfig.DataLabels != nil {
		return providerConfig.DataLabels
	}

	return labels
}

func (a *OpenAuth2AppConfig) getProviderDataWidgets(providerName string) map[string]widgets.Widget {
	var widgets = make(map[string]widgets.Widget)
	var providerConfig, ok = a._cnfs[providerName]
	if !ok {
		return widgets
	}

	if providerConfig.DataWidgets != nil {
		return providerConfig.DataWidgets
	}

	return widgets
}

func (a *OpenAuth2AppConfig) Check(ctx context.Context, settings django.Settings) []checks.Message {
	var messages = a.DBRequiredAppConfig.Check(ctx, settings)

	if len(a.Config.AuthConfigurations) == 0 {
		messages = append(messages, checks.Error(
			"openauth2.no_providers",
			"OpenAuth2: No providers configured",
			nil,
		))
	}

	for _, cnf := range a.Config.AuthConfigurations {
		if cnf.ProviderInfo.Provider == "" {
			messages = append(messages, checks.Error(
				"openauth2.provider_missing",
				"OpenAuth2: Provider name is missing",
				cnf,
			))
			continue
		}

		if cnf.Oauth2 == nil {
			messages = append(messages, checks.Error(
				"openauth2.oauth2_missing",
				"OpenAuth2: OAuth2 configuration is missing",
				cnf.ProviderInfo.Provider,
			))
			continue
		}

		if cnf.Oauth2.ClientID == "" {
			messages = append(messages, checks.Error(
				"openauth2.client_id_missing",
				"OpenAuth2: Client ID is missing",
				cnf.ProviderInfo.Provider,
			))
			continue
		}

		if cnf.Oauth2.ClientSecret == "" {
			messages = append(messages, checks.Error(
				"openauth2.client_secret_missing",
				"OpenAuth2: Client Secret is missing for provider",
				cnf.ProviderInfo.Provider,
			))
			continue
		}
	}

	return messages
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
