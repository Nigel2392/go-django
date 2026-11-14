//go:build test
// +build test

package settings_test

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/auth"
	autherrors "github.com/Nigel2392/go-django/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/contrib/blocks"
	"github.com/Nigel2392/go-django/contrib/messages"
	"github.com/Nigel2392/go-django/contrib/session"
	"github.com/Nigel2392/go-django/contrib/settings"
	"github.com/Nigel2392/go-django/djester"
	"github.com/Nigel2392/go-django/djester/quest"
	"github.com/Nigel2392/go-django/djester/testdb"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/forms/widgets/options"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/mux/middleware/authentication"
	"github.com/PuerkitoBio/goquery"
)

type BaseSetting struct {
	Type               dbtype.Type
	Default            any
	Permission         []string
	DisplayIcon        func(ctx context.Context) template.HTML
	DisplayLabel       func(ctx context.Context) string
	DisplayDescription func(ctx context.Context) string
	Field              func(ctx context.Context) forms.Field
}

func (b BaseSetting) DBType() dbtype.Type {
	return b.Type
}

func (b BaseSetting) Permissions() []string {
	return b.Permission
}

func (b BaseSetting) DefaultValue() any {
	return b.Default
}

func (b BaseSetting) Icon(ctx context.Context) template.HTML {
	if b.DisplayIcon != nil {
		return b.DisplayIcon(ctx)
	}
	return ""
}

func (b BaseSetting) Label(ctx context.Context) string {
	if b.DisplayLabel != nil {
		return b.DisplayLabel(ctx)
	}
	return ""
}

func (b BaseSetting) Description(ctx context.Context) string {
	if b.DisplayDescription != nil {
		return b.DisplayDescription(ctx)
	}
	return ""
}

func (b BaseSetting) FormField(ctx context.Context) forms.Field {
	if b.Field == nil {
		return nil
	}
	return b.Field(ctx)
}

func (b BaseSetting) ValueFromDB(ctx context.Context, val json.RawMessage) (any, error) {
	var (
		typ  = drivers.DBToDefaultGoType(b.Type)
		nval = reflect.New(typ)
	)

	err := json.Unmarshal(val, nval.Interface())
	return nval.Elem().Interface(), err
}

type User struct {
	Name            string `attrs:"primary"`
	LoggedIn        bool
	IsAdministrator bool
	Superuser       bool
	Perms           []string
}

func (u *User) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions[*User](ctx, u)
}

func (u *User) IsAuthenticated() bool {
	return u.LoggedIn
}

func (u *User) IsAdmin() bool {
	return u.IsAdministrator
}

type userPermissionTester struct{}

func (t *userPermissionTester) HasObjectPermission(r *http.Request, obj interface{}, perms ...string) bool {
	return t.HasPermission(r, perms...)
}

func (t *userPermissionTester) HasPermission(r *http.Request, perms ...string) bool {
	if len(perms) == 0 {
		return true
	}
	u, ok := authentication.Retrieve(r).(*User)
	if !ok || u == nil || !u.IsAuthenticated() {
		return false
	}
	if u.Superuser {
		return true
	}
	for _, perm := range perms {
		if !slices.Contains(u.Perms, perm) {
			return false
		}
	}
	return true
}

const (
	GroupGeneral   = "general"
	GroupSuppliers = "suppliers"
	GroupSecurity  = "security"

	SettingGeneralSiteName    = "general.site_name"
	SettingGeneralMaintenance = "general.maintenance"
	SettingGeneralMaxItems    = "general.max_items"

	SettingSupplierEnabled = "commerce.dbsupplier.backend"
	SettingSupplierStorage = "commerce.dbsupplier.backend.storage"

	SettingSecuritySecretKey = "security.secret_key"
	SettingSecurityAuditLog  = "security.audit_log"
)

func registerTestSettings() {
	// Group 1: General (mixed public and permission-restricted settings)
	settings.RegisterGroup(1, GroupGeneral, settings.SettingsGroup{
		Label:       trans.S("General"),
		Description: trans.S("General site settings"),
	})
	settings.RegisterSetting(1, GroupGeneral, SettingGeneralSiteName, BaseSetting{
		Type:               dbtype.String,
		Default:            "My Go-Django Site",
		DisplayLabel:       trans.S("Site Name"),
		DisplayDescription: trans.S("The public name of the site."),
		Field: func(ctx context.Context) forms.Field {
			return fields.CharField()
		},
	})
	settings.RegisterSetting(2, GroupGeneral, SettingGeneralMaintenance, BaseSetting{
		Type:               dbtype.Bool,
		Default:            false,
		Permission:         []string{"settings:general:maintenance"},
		DisplayLabel:       trans.S("Maintenance Mode"),
		DisplayDescription: trans.S("Enable or disable maintenance mode."),
		Field: func(ctx context.Context) forms.Field {
			return fields.BooleanField()
		},
	})
	settings.RegisterSetting(3, GroupGeneral, SettingGeneralMaxItems, BaseSetting{
		Type:               dbtype.Int,
		Default:            int64(25),
		Permission:         []string{"settings:general:max_items"},
		DisplayLabel:       trans.S("Max Items"),
		DisplayDescription: trans.S("Maximum number of items per page."),
		Field: func(ctx context.Context) forms.Field {
			return fields.NumberField[int64]()
		},
	})

	// Group 2: Suppliers (mirroring examples/commerce/src/suppliers/dbsupplier/app.go)
	settings.RegisterGroup(2, GroupSuppliers, settings.SettingsGroup{
		Label:       trans.S("Suppliers"),
		Description: trans.S("Supplier specific settings"),
	})
	settings.RegisterSetting(1, GroupSuppliers, SettingSupplierEnabled, BaseSetting{
		Type:               dbtype.Bool,
		Default:            false,
		Permission:         []string{"settings:suppliers:backend"},
		DisplayLabel:       trans.S("Database Supplier Enabled"),
		DisplayDescription: trans.S("Whether to enable or disable the database supplier backend."),
		Field: func(ctx context.Context) forms.Field {
			return fields.BooleanField()
		},
	})
	settings.RegisterSetting(2, GroupSuppliers, SettingSupplierStorage, BaseSetting{
		Type:               dbtype.String,
		Default:            "local",
		Permission:         []string{"settings:suppliers:storage"},
		DisplayLabel:       trans.S("Database Supplier Storage"),
		DisplayDescription: trans.S("Which storage backend to use for the database supplier."),
		Field: func(ctx context.Context) forms.Field {
			return fields.CharField(fields.Widget(options.NewSelectInput(nil, func(ctx context.Context) []widgets.Option {
				return []widgets.Option{
					widgets.NewOption("local", trans.T(ctx, "Local"), "local"),
					widgets.NewOption("cloud", trans.T(ctx, "Cloud"), "cloud"),
				}
			})))
		},
	})

	// Group 3: Security (high-privilege settings requiring multiple/specific permissions)
	settings.RegisterGroup(3, GroupSecurity, settings.SettingsGroup{
		Label:       trans.S("Security"),
		Description: trans.S("Security and audit settings"),
	})
	settings.RegisterSetting(1, GroupSecurity, SettingSecuritySecretKey, BaseSetting{
		Type:               dbtype.String,
		Default:            "default-secret",
		Permission:         []string{"settings:security:view", "settings:security:secret"},
		DisplayLabel:       trans.S("Secret Key"),
		DisplayDescription: trans.S("Secret signing key."),
		Field: func(ctx context.Context) forms.Field {
			return fields.CharField()
		},
	})
	settings.RegisterSetting(2, GroupSecurity, SettingSecurityAuditLog, BaseSetting{
		Type:               dbtype.Bool,
		Default:            false,
		Permission:         []string{"settings:security:view"},
		DisplayLabel:       trans.S("Audit Logging"),
		DisplayDescription: trans.S("Enable security audit logging."),
		Field: func(ctx context.Context) forms.Field {
			return fields.BooleanField()
		},
	})
}

func initAdminApp() django.AppConfig {
	admin.ConfigureAuth(admin.AuthConfig{
		GetLoginForm: func(r *http.Request, formOpts ...func(forms.Form)) admin.LoginForm {
			return auth.UserLoginForm(r, formOpts...)
		},
		Logout: auth.Logout,
	})

	attrs.RegisterModel(&User{})
	admin.RegisterApp(
		"settings_test",
		admin.AppOptions{
			EnableIndexView: true,
		},
		admin.ModelOptions{Name: "User", Model: &User{}},
	)
	autherrors.RegisterHook("index")
	return admin.NewAppConfig()
}

// assertGroupHasSettingInHTML checks the rendered StructBlock telepath arguments on the page
// to verify whether a specific setting is present or absent in a given group's block widget,
// and optionally checks its rendered value.
func assertGroupSettingHTML(groupName, settingName string, shouldExist bool, expectedVal ...any) djester.HTMLAssertFunc {
	return func(doc *goquery.Document) error {
		sel := doc.Find(fmt.Sprintf("#id_%s[data-struct-block]", groupName))
		if sel.Length() == 0 {
			return fmt.Errorf("expected struct block element #id_%s[data-struct-block], found none", groupName)
		}

		classArgsAttr, ok := sel.Attr("data-block-class-args-value")
		if !ok {
			return fmt.Errorf("missing data-block-class-args-value on #id_%s", groupName)
		}

		inClassArgs := strings.Contains(classArgsAttr, fmt.Sprintf(`"name":%q`, settingName))
		if shouldExist && !inClassArgs {
			return fmt.Errorf("expected setting %q to be present in group %q class args, got: %s", settingName, groupName, classArgsAttr)
		}
		if !shouldExist && inClassArgs {
			return fmt.Errorf("expected setting %q NOT to be present in group %q class args, got: %s", settingName, groupName, classArgsAttr)
		}

		argsAttr, ok := sel.Attr("data-block-arguments-value")
		if !ok {
			return fmt.Errorf("missing data-block-arguments-value on #id_%s", groupName)
		}

		var parsedArgs []json.RawMessage
		if err := json.Unmarshal([]byte(argsAttr), &parsedArgs); err != nil {
			return fmt.Errorf("failed to unmarshal data-block-arguments-value on #id_%s: %w", groupName, err)
		}
		if len(parsedArgs) < 2 {
			return fmt.Errorf("unexpected data-block-arguments-value length on #id_%s: %s", groupName, argsAttr)
		}

		var valueMap map[string]any
		if err := json.Unmarshal(parsedArgs[1], &valueMap); err != nil {
			return fmt.Errorf("failed to unmarshal valueMap in data-block-arguments-value on #id_%s: %w", groupName, err)
		}

		actualVal, existsInValues := valueMap[settingName]
		if shouldExist && !existsInValues {
			return fmt.Errorf("expected setting %q in valueMap of group %q, got keys: %v", settingName, groupName, valueMap)
		}
		if !shouldExist && existsInValues {
			return fmt.Errorf("expected setting %q NOT in valueMap of group %q, but found value: %v", settingName, groupName, actualVal)
		}

		if shouldExist && len(expectedVal) > 0 {
			wantBytes, _ := json.Marshal(expectedVal[0])
			gotBytes, _ := json.Marshal(actualVal)
			if string(wantBytes) != string(gotBytes) && fmt.Sprintf("%v", expectedVal[0]) != fmt.Sprintf("%v", actualVal) {
				return fmt.Errorf("setting %q in group %q: expected HTML value %s, got %s", settingName, groupName, string(wantBytes), string(gotBytes))
			}
		}

		return nil
	}
}

func assertNoFormErrors() djester.HTMLAssertFunc {
	return func(doc *goquery.Document) error {
		s := doc.Find(".panel__error, .errorlist li")
		if s.Length() > 0 {
			text := strings.TrimSpace(s.Text())
			if text != "" {
				return fmt.Errorf("unexpected form error(s) in HTML: %s", text)
			}
		}
		return nil
	}
}

func assertDBSettingValue[T comparable](t *testing.T, asserter djester.Assertion, internalName string, expected T) {
	t.Helper()
	row, err := queries.GetQuerySet(&settings.RegisteredSetting{}).
		Filter("InternalName", internalName).
		Get()
	asserter.Assert(err == nil, "failed to fetch RegisteredSetting %q from DB: %v", internalName, err)
	asserter.Assert(len(row.Object.Data) > 0, "expected non-empty Data for setting %q", internalName)

	var actual T
	err = json.Unmarshal(row.Object.Data, &actual)
	asserter.Assert(err == nil, "failed to unmarshal Data for setting %q (%s): %v", internalName, string(row.Object.Data), err)
	asserter.AssertEqual(expected, actual)
}

func resetSettingsDBData(t *testing.T, asserter djester.Assertion) {
	t.Helper()
	_, err := queries.GetQuerySet(&settings.RegisteredSetting{}).Delete()
	asserter.Assert(err == nil, "failed to clear RegisteredSetting table: %v", err)

	err = settings.SyncSettingsCmd.Execute(nil, nil, nil)
	asserter.Assert(err == nil, "SyncSettingsCmd failed: %v", err)
}

func TestSettings(t *testing.T) {
	var _, db = testdb.Open()
	defer db.Close()

	permissions.Tester = &userPermissionTester{}
	defer func() {
		permissions.Tester = nil
	}()

	registerTestSettings()

	d := &djester.Tester{
		Settings: map[string]any{
			django.APPVAR_ALLOWED_HOSTS:                     []string{"*"},
			django.APPVAR_DISABLE_NOSURF:                    true,
			django.APPVAR_DEBUG:                             true,
			django.APPVAR_DATABASE:                          db,
			"contrib.settings.SettingsForRequestMiddleware": false,
		},
		Flags: []django.AppFlag{
			django.FlagSkipCmds,
			django.FlagSkipChecks,
			django.FlagSkipDepsCheck,
		},
		Apps: []djester.AppInitFuncOrAppConfig{
			session.NewAppConfig,
			messages.NewAppConfig,
			blocks.NewAppConfig,
			initAdminApp,
			settings.NewAppConfig,
			&apps.AppConfig{
				AppName: "settings_test",
				ModelObjects: []attrs.Definer{
					&User{},
				},
			},
		},
		Auth: &djester.TesterAuth{
			UnauthenticatedUser: func() authentication.User {
				return &User{LoggedIn: false, IsAdministrator: false}
			},
			Users: map[string]authentication.User{
				"normal_user": &User{
					Name:            "normal_user",
					LoggedIn:        true,
					IsAdministrator: false,
				},
				"admin_no_settings_perms": &User{
					Name:            "admin_no_settings_perms",
					LoggedIn:        true,
					IsAdministrator: true,
					Perms:           []string{},
				},
				"admin_view_public_only": &User{
					Name:            "admin_view_public_only",
					LoggedIn:        true,
					IsAdministrator: true,
					Perms:           []string{"settings:view", "settings:edit"},
				},
				"admin_general_editor": &User{
					Name:            "admin_general_editor",
					LoggedIn:        true,
					IsAdministrator: true,
					Perms: []string{
						"settings:view",
						"settings:edit",
						"settings:general:maintenance",
						"settings:general:max_items",
					},
				},
				"admin_suppliers_editor": &User{
					Name:            "admin_suppliers_editor",
					LoggedIn:        true,
					IsAdministrator: true,
					Perms: []string{
						"settings:view",
						"settings:edit",
						"settings:suppliers:backend",
						"settings:suppliers:storage",
					},
				},
				"admin_partial_security": &User{
					Name:            "admin_partial_security",
					LoggedIn:        true,
					IsAdministrator: true,
					Perms: []string{
						"settings:view",
						"settings:edit",
						"settings:suppliers:backend",
						"settings:security:view", // has view, but NOT settings:security:secret
					},
				},
				"superuser": &User{
					Name:            "superuser",
					LoggedIn:        true,
					IsAdministrator: true,
					Superuser:       true,
				},
			},
		},
		Tests: []djester.Test{
			&djester.BasicTest{
				Label: "Unauthenticated and unauthorized access redirects",
				Function: func(d *djester.Tester, t *testing.T) {
					asserter := djester.Asserter(t, true)
					endpoints := []string{
						"/admin/settings",
						"/admin/settings/change",
					}

					// 1. Unauthenticated user -> redirects to /admin/login/
					for _, ep := range endpoints {
						t.Run("unauthenticated_"+ep, func(t *testing.T) {
							subAssert := djester.Asserter(t, true)
							resp, err := d.Get(ep, nil, nil)
							subAssert.Assert(err == nil, "GET %s failed: %v", ep, err)
							defer resp.Body.Close()
							subAssert.AssertEqual(http.StatusOK, resp.StatusCode)
							subAssert.Assert(
								strings.HasPrefix(resp.Request.URL.Path, "/admin/login/"),
								"expected redirect to /admin/login/, got %s", resp.Request.URL.Path,
							)
						})
					}

					// 2. Non-admin authenticated user -> redirects to /admin/relogin/
					_, err := d.Login("normal_user")
					asserter.Assert(err == nil, "Login failed: %v", err)
					for _, ep := range endpoints {
						t.Run("normal_user_"+ep, func(t *testing.T) {
							subAssert := djester.Asserter(t, true)
							resp, err := d.Get(ep, nil, nil)
							subAssert.Assert(err == nil, "GET %s failed: %v", ep, err)
							defer resp.Body.Close()
							subAssert.AssertEqual(http.StatusOK, resp.StatusCode)
							subAssert.Assert(
								strings.HasPrefix(resp.Request.URL.Path, "/admin/relogin/"),
								"expected redirect to /admin/relogin/, got %s", resp.Request.URL.Path,
							)
						})
					}
					d.Logout()

					// 3. Admin user without settings:view or settings:edit permissions -> redirects to /admin/login/ -> /admin/
					_, err = d.Login("admin_no_settings_perms")
					asserter.Assert(err == nil, "Login failed: %v", err)
					for _, ep := range endpoints {
						t.Run("admin_no_settings_perms_"+ep, func(t *testing.T) {
							subAssert := djester.Asserter(t, true)
							resp, err := d.Get(ep, nil, nil)
							subAssert.Assert(err == nil, "GET %s failed: %v", ep, err)
							defer resp.Body.Close()
							subAssert.AssertEqual(http.StatusOK, resp.StatusCode)
							subAssert.AssertEqual("/admin/", resp.Request.URL.Path)
						})
					}
					d.Logout()
				},
			},
			&djester.BasicTest{
				Label: "Visibility of settings across groups with different permissions",
				Function: func(d *djester.Tester, t *testing.T) {
					asserter := djester.Asserter(t, true)
					resetSettingsDBData(t, asserter)

					type visibilityCase struct {
						userKey     string
						visible     map[string][]string
						notVisible  map[string][]string
						defaultVals map[string]map[string]any
					}

					cases := []visibilityCase{
						{
							userKey: "admin_view_public_only",
							visible: map[string][]string{
								GroupGeneral:   {SettingGeneralSiteName},
								GroupSuppliers: {},
								GroupSecurity:  {},
							},
							notVisible: map[string][]string{
								GroupGeneral:   {SettingGeneralMaintenance, SettingGeneralMaxItems},
								GroupSuppliers: {SettingSupplierEnabled, SettingSupplierStorage},
								GroupSecurity:  {SettingSecuritySecretKey, SettingSecurityAuditLog},
							},
							defaultVals: map[string]map[string]any{
								GroupGeneral: {SettingGeneralSiteName: "My Go-Django Site"},
							},
						},
						{
							userKey: "admin_general_editor",
							visible: map[string][]string{
								GroupGeneral:   {SettingGeneralSiteName, SettingGeneralMaintenance, SettingGeneralMaxItems},
								GroupSuppliers: {},
								GroupSecurity:  {},
							},
							notVisible: map[string][]string{
								GroupGeneral:   {},
								GroupSuppliers: {SettingSupplierEnabled, SettingSupplierStorage},
								GroupSecurity:  {SettingSecuritySecretKey, SettingSecurityAuditLog},
							},
							defaultVals: map[string]map[string]any{
								GroupGeneral: {
									SettingGeneralSiteName:    "My Go-Django Site",
									SettingGeneralMaintenance: false,
									SettingGeneralMaxItems:    "25",
								},
							},
						},
						{
							userKey: "admin_suppliers_editor",
							visible: map[string][]string{
								GroupGeneral:   {SettingGeneralSiteName},
								GroupSuppliers: {SettingSupplierEnabled, SettingSupplierStorage},
								GroupSecurity:  {},
							},
							notVisible: map[string][]string{
								GroupGeneral:   {SettingGeneralMaintenance, SettingGeneralMaxItems},
								GroupSuppliers: {},
								GroupSecurity:  {SettingSecuritySecretKey, SettingSecurityAuditLog},
							},
							defaultVals: map[string]map[string]any{
								GroupGeneral: {SettingGeneralSiteName: "My Go-Django Site"},
								GroupSuppliers: {
									SettingSupplierEnabled: false,
									SettingSupplierStorage: "local",
								},
							},
						},
						{
							userKey: "admin_partial_security",
							visible: map[string][]string{
								GroupGeneral:   {SettingGeneralSiteName},
								GroupSuppliers: {SettingSupplierEnabled},
								GroupSecurity:  {SettingSecurityAuditLog},
							},
							notVisible: map[string][]string{
								GroupGeneral:   {SettingGeneralMaintenance, SettingGeneralMaxItems},
								GroupSuppliers: {SettingSupplierStorage},
								GroupSecurity:  {SettingSecuritySecretKey}, // requires both settings:security:view AND settings:security:secret
							},
							defaultVals: map[string]map[string]any{
								GroupGeneral:   {SettingGeneralSiteName: "My Go-Django Site"},
								GroupSuppliers: {SettingSupplierEnabled: false},
								GroupSecurity:  {SettingSecurityAuditLog: false},
							},
						},
						{
							userKey: "superuser",
							visible: map[string][]string{
								GroupGeneral:   {SettingGeneralSiteName, SettingGeneralMaintenance, SettingGeneralMaxItems},
								GroupSuppliers: {SettingSupplierEnabled, SettingSupplierStorage},
								GroupSecurity:  {SettingSecuritySecretKey, SettingSecurityAuditLog},
							},
							notVisible: map[string][]string{
								GroupGeneral:   {},
								GroupSuppliers: {},
								GroupSecurity:  {},
							},
							defaultVals: map[string]map[string]any{
								GroupGeneral: {
									SettingGeneralSiteName:    "My Go-Django Site",
									SettingGeneralMaintenance: false,
									SettingGeneralMaxItems:    25,
								},
								GroupSuppliers: {
									SettingSupplierEnabled: false,
									SettingSupplierStorage: "local",
								},
								GroupSecurity: {
									SettingSecuritySecretKey: "default-secret",
									SettingSecurityAuditLog:  false,
								},
							},
						},
					}

					for _, tc := range cases {
						t.Run(tc.userKey, func(t *testing.T) {
							asserter := djester.Asserter(t, true)
							_, err := d.Login(tc.userKey)
							asserter.Assert(err == nil, "Login(%s) failed: %v", tc.userKey, err)
							defer d.Logout()

							for _, pageURL := range []string{"/admin/settings", "/admin/settings/change"} {
								resp, err := d.Get(pageURL, nil, nil)
								asserter.Assert(err == nil, "GET %s failed: %v", pageURL, err)
								asserter.AssertEqual(http.StatusOK, resp.StatusCode)
								asserter.AssertEqual(pageURL, resp.Request.URL.Path)

								var htmlChecks []djester.HTMLAssertFunc
								htmlChecks = append(htmlChecks,
									djester.HasText(".group-list-item--heading", "General"),
									djester.HasText(".group-list-item--heading", "Suppliers"),
									djester.HasText(".group-list-item--heading", "Security"),
								)

								for grp, settingsList := range tc.visible {
									for _, sName := range settingsList {
										if expMap, ok := tc.defaultVals[grp]; ok {
											if expVal, hasVal := expMap[sName]; hasVal {
												htmlChecks = append(htmlChecks, assertGroupSettingHTML(grp, sName, true, expVal))
												continue
											}
										}
										htmlChecks = append(htmlChecks, assertGroupSettingHTML(grp, sName, true))
									}
								}

								for grp, settingsList := range tc.notVisible {
									for _, sName := range settingsList {
										htmlChecks = append(htmlChecks, assertGroupSettingHTML(grp, sName, false))
									}
								}

								resp.Assert(t, true).AssertHTML(htmlChecks...)
								resp.Body.Close()
							}
						})
					}
				},
			},
			&djester.BasicTest{
				Label: "Form submission and data change for single group (General)",
				Function: func(d *djester.Tester, t *testing.T) {
					asserter := djester.Asserter(t, true)
					resetSettingsDBData(t, asserter)

					_, err := d.Login("admin_general_editor")
					asserter.Assert(err == nil, "Login failed: %v", err)
					defer d.Logout()

					form := map[string]interface{}{
						fmt.Sprintf("%s-%s", GroupGeneral, SettingGeneralSiteName):    "Updated Commerce Portal",
						fmt.Sprintf("%s-%s", GroupGeneral, SettingGeneralMaintenance): "true",
						fmt.Sprintf("%s-%s", GroupGeneral, SettingGeneralMaxItems):    "100",
					}

					resp, err := d.PostForm("/admin/settings/change", nil, nil, form)
					asserter.Assert(err == nil, "POST /admin/settings/change failed: %v", err)
					asserter.AssertEqual(http.StatusOK, resp.StatusCode)
					asserter.AssertEqual("/admin/settings/", resp.Request.URL.Path)
					resp.Assert(t, true).AssertHTML(
						assertNoFormErrors(),
						assertGroupSettingHTML(GroupGeneral, SettingGeneralSiteName, true, "Updated Commerce Portal"),
						assertGroupSettingHTML(GroupGeneral, SettingGeneralMaintenance, true, true),
						assertGroupSettingHTML(GroupGeneral, SettingGeneralMaxItems, true, "100"),
						// Suppliers and Security settings must still be hidden for this user
						assertGroupSettingHTML(GroupSuppliers, SettingSupplierEnabled, false),
						assertGroupSettingHTML(GroupSuppliers, SettingSupplierStorage, false),
						assertGroupSettingHTML(GroupSecurity, SettingSecuritySecretKey, false),
					)
					resp.Body.Close()

					// Verify database changes for General group settings
					assertDBSettingValue(t, asserter, SettingGeneralSiteName, "Updated Commerce Portal")
					assertDBSettingValue(t, asserter, SettingGeneralMaintenance, true)
					assertDBSettingValue(t, asserter, SettingGeneralMaxItems, int64(100))

					// Verify untouched groups still have nil/empty Data in DB
					for _, untouched := range []string{SettingSupplierEnabled, SettingSupplierStorage, SettingSecuritySecretKey, SettingSecurityAuditLog} {
						row, err := queries.GetQuerySet(&settings.RegisteredSetting{}).
							Filter("InternalName", untouched).
							Get()
						asserter.Assert(err == nil, "failed to get %q: %v", untouched, err)
						asserter.AssertEqual(0, len(row.Object.Data))
					}
				},
			},
			&djester.BasicTest{
				Label: "Form submission and data change across different groups and permission combinations",
				Function: func(d *djester.Tester, t *testing.T) {
					asserter := djester.Asserter(t, true)
					resetSettingsDBData(t, asserter)

					// Step 1: Suppliers editor updates General public setting + Suppliers group settings,
					// and attempts to inject unauthorized Security group fields in the POST payload.
					_, err := d.Login("admin_suppliers_editor")
					asserter.Assert(err == nil, "Login failed: %v", err)

					supplierForm := map[string]interface{}{
						fmt.Sprintf("%s-%s", GroupGeneral, SettingGeneralSiteName):    "Supplier Managed Shop",
						fmt.Sprintf("%s-%s", GroupGeneral, SettingGeneralMaintenance): "true", // unauthorized for suppliers editor
						fmt.Sprintf("%s-%s", GroupSuppliers, SettingSupplierEnabled):  "true",
						fmt.Sprintf("%s-%s", GroupSuppliers, SettingSupplierStorage):  "cloud",
						fmt.Sprintf("%s-%s", GroupSecurity, SettingSecuritySecretKey): "hacked-secret", // unauthorized for suppliers editor
					}

					resp, err := d.PostForm("/admin/settings/change", nil, nil, supplierForm)
					asserter.Assert(err == nil, "POST /admin/settings/change failed: %v", err)
					asserter.AssertEqual(http.StatusOK, resp.StatusCode)
					asserter.AssertEqual("/admin/settings/", resp.Request.URL.Path)
					resp.Assert(t, true).AssertHTML(
						assertNoFormErrors(),
						assertGroupSettingHTML(GroupGeneral, SettingGeneralSiteName, true, "Supplier Managed Shop"),
						assertGroupSettingHTML(GroupSuppliers, SettingSupplierEnabled, true, true),
						assertGroupSettingHTML(GroupSuppliers, SettingSupplierStorage, true, "cloud"),
						assertGroupSettingHTML(GroupGeneral, SettingGeneralMaintenance, false),
						assertGroupSettingHTML(GroupSecurity, SettingSecuritySecretKey, false),
					)
					resp.Body.Close()
					d.Logout()

					// Verify permitted settings were saved in DB
					assertDBSettingValue(t, asserter, SettingGeneralSiteName, "Supplier Managed Shop")
					assertDBSettingValue(t, asserter, SettingSupplierEnabled, true)
					assertDBSettingValue(t, asserter, SettingSupplierStorage, "cloud")

					// Verify unauthorized settings in the POST payload were NOT saved in DB
					for _, unauthorized := range []string{SettingGeneralMaintenance, SettingGeneralMaxItems, SettingSecuritySecretKey, SettingSecurityAuditLog} {
						row, err := queries.GetQuerySet(&settings.RegisteredSetting{}).
							Filter("InternalName", unauthorized).
							Get()
						asserter.Assert(err == nil, "failed to get %q: %v", unauthorized, err)
						asserter.AssertEqual(0, len(row.Object.Data))
					}

					// Step 2: Partial security user logs in, sees updated values for visible settings,
					// and updates SupplierEnabled + SecurityAuditLog across groups.
					_, err = d.Login("admin_partial_security")
					asserter.Assert(err == nil, "Login failed: %v", err)

					resp, err = d.Get("/admin/settings/change", nil, nil)
					asserter.Assert(err == nil, "GET /admin/settings/change failed: %v", err)
					resp.Assert(t, true).AssertHTML(
						assertGroupSettingHTML(GroupGeneral, SettingGeneralSiteName, true, "Supplier Managed Shop"),
						assertGroupSettingHTML(GroupSuppliers, SettingSupplierEnabled, true, true),
						assertGroupSettingHTML(GroupSuppliers, SettingSupplierStorage, false),
						assertGroupSettingHTML(GroupSecurity, SettingSecurityAuditLog, true, false),
						assertGroupSettingHTML(GroupSecurity, SettingSecuritySecretKey, false),
					)
					resp.Body.Close()

					partialSecurityForm := map[string]interface{}{
						// Keep General unchanged ("Supplier Managed Shop")
						fmt.Sprintf("%s-%s", GroupGeneral, SettingGeneralSiteName): "Supplier Managed Shop",
						// Toggle SupplierEnabled to false (omitting checkbox value or passing "false")
						fmt.Sprintf("%s-%s", GroupSuppliers, SettingSupplierEnabled): "false",
						// Toggle SecurityAuditLog to true
						fmt.Sprintf("%s-%s", GroupSecurity, SettingSecurityAuditLog): "true",
					}

					resp, err = d.PostForm("/admin/settings/change", nil, nil, partialSecurityForm)
					asserter.Assert(err == nil, "POST /admin/settings/change failed: %v", err)
					asserter.AssertEqual(http.StatusOK, resp.StatusCode)
					asserter.AssertEqual("/admin/settings/", resp.Request.URL.Path)
					resp.Assert(t, true).AssertHTML(
						assertNoFormErrors(),
						assertGroupSettingHTML(GroupGeneral, SettingGeneralSiteName, true, "Supplier Managed Shop"),
						assertGroupSettingHTML(GroupSuppliers, SettingSupplierEnabled, true, false),
						assertGroupSettingHTML(GroupSecurity, SettingSecurityAuditLog, true, true),
					)
					resp.Body.Close()
					d.Logout()

					// Verify DB state after partial security user's submission:
					// - SettingSupplierEnabled changed to false
					// - SettingSupplierStorage remained "cloud" (since partial_security user didn't have permission to touch it)
					// - SettingSecurityAuditLog changed to true
					// - SettingSecuritySecretKey remained untouched (empty)
					assertDBSettingValue(t, asserter, SettingGeneralSiteName, "Supplier Managed Shop")
					assertDBSettingValue(t, asserter, SettingSupplierEnabled, false)
					assertDBSettingValue(t, asserter, SettingSupplierStorage, "cloud")
					assertDBSettingValue(t, asserter, SettingSecurityAuditLog, true)

					secretRow, err := queries.GetQuerySet(&settings.RegisteredSetting{}).
						Filter("InternalName", SettingSecuritySecretKey).
						Get()
					asserter.Assert(err == nil, "failed to get %q: %v", SettingSecuritySecretKey, err)
					asserter.AssertEqual(0, len(secretRow.Object.Data))

					// Step 3: Superuser logs in, sees all groups & settings (both previously updated and default ones),
					// and updates all groups at once.
					_, err = d.Login("superuser")
					asserter.Assert(err == nil, "Login failed: %v", err)
					defer d.Logout()

					resp, err = d.Get("/admin/settings", nil, nil)
					asserter.Assert(err == nil, "GET /admin/settings failed: %v", err)
					resp.Assert(t, true).AssertHTML(
						assertGroupSettingHTML(GroupGeneral, SettingGeneralSiteName, true, "Supplier Managed Shop"),
						assertGroupSettingHTML(GroupGeneral, SettingGeneralMaintenance, true, false),
						assertGroupSettingHTML(GroupGeneral, SettingGeneralMaxItems, true, "25"),
						assertGroupSettingHTML(GroupSuppliers, SettingSupplierEnabled, true, false),
						assertGroupSettingHTML(GroupSuppliers, SettingSupplierStorage, true, "cloud"),
						assertGroupSettingHTML(GroupSecurity, SettingSecuritySecretKey, true, "default-secret"),
						assertGroupSettingHTML(GroupSecurity, SettingSecurityAuditLog, true, true),
					)
					resp.Body.Close()

					superuserForm := map[string]interface{}{
						fmt.Sprintf("%s-%s", GroupGeneral, SettingGeneralSiteName):    "Global Enterprise Store",
						fmt.Sprintf("%s-%s", GroupGeneral, SettingGeneralMaintenance): "true",
						fmt.Sprintf("%s-%s", GroupGeneral, SettingGeneralMaxItems):    "250",
						fmt.Sprintf("%s-%s", GroupSuppliers, SettingSupplierEnabled):  "true",
						fmt.Sprintf("%s-%s", GroupSuppliers, SettingSupplierStorage):  "local",
						fmt.Sprintf("%s-%s", GroupSecurity, SettingSecuritySecretKey): "rotated-prod-secret-999",
						fmt.Sprintf("%s-%s", GroupSecurity, SettingSecurityAuditLog):  "true",
					}

					resp, err = d.PostForm("/admin/settings/change", nil, nil, superuserForm)
					asserter.Assert(err == nil, "POST /admin/settings/change failed: %v", err)
					asserter.AssertEqual(http.StatusOK, resp.StatusCode)
					asserter.AssertEqual("/admin/settings/", resp.Request.URL.Path)
					resp.Assert(t, true).AssertHTML(
						assertNoFormErrors(),
						assertGroupSettingHTML(GroupGeneral, SettingGeneralSiteName, true, "Global Enterprise Store"),
						assertGroupSettingHTML(GroupGeneral, SettingGeneralMaintenance, true, true),
						assertGroupSettingHTML(GroupGeneral, SettingGeneralMaxItems, true, "250"),
						assertGroupSettingHTML(GroupSuppliers, SettingSupplierEnabled, true, true),
						assertGroupSettingHTML(GroupSuppliers, SettingSupplierStorage, true, "local"),
						assertGroupSettingHTML(GroupSecurity, SettingSecuritySecretKey, true, "rotated-prod-secret-999"),
						assertGroupSettingHTML(GroupSecurity, SettingSecurityAuditLog, true, true),
					)
					resp.Body.Close()

					assertDBSettingValue(t, asserter, SettingGeneralSiteName, "Global Enterprise Store")
					assertDBSettingValue(t, asserter, SettingGeneralMaintenance, true)
					assertDBSettingValue(t, asserter, SettingGeneralMaxItems, int64(250))
					assertDBSettingValue(t, asserter, SettingSupplierEnabled, true)
					assertDBSettingValue(t, asserter, SettingSupplierStorage, "local")
					assertDBSettingValue(t, asserter, SettingSecuritySecretKey, "rotated-prod-secret-999")
					assertDBSettingValue(t, asserter, SettingSecurityAuditLog, true)
				},
			},
			&djester.BasicTest{
				Label: "SettingsForRequest programmatic API reflects permissions and updated DB data",
				Function: func(d *djester.Tester, t *testing.T) {
					asserter := djester.Asserter(t, true)

					req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/admin/settings", nil)
					asserter.Assert(err == nil, "NewRequest failed: %v", err)

					// Attach admin_suppliers_editor user to request context
					supplierUser := d.Auth.Users["admin_suppliers_editor"]
					req = req.WithContext(authentication.ContextWithUser(req.Context(), supplierUser))

					groupsMap, err := settings.SettingsForRequest(req, true)
					asserter.Assert(err == nil, "SettingsForRequest failed: %v", err)
					asserter.AssertEqual(3, groupsMap.Len())

					// Check General group: only SettingGeneralSiteName is visible and has updated value
					genGroup, ok := groupsMap.Get(GroupGeneral)
					asserter.Assert(ok, "expected group %q", GroupGeneral)
					asserter.AssertEqual(1, len(genGroup.Settings))
					asserter.AssertEqual(SettingGeneralSiteName, genGroup.Settings[0].Name())
					val, err := genGroup.Settings[0].Value()
					asserter.Assert(err == nil, "Value() failed: %v", err)
					asserter.AssertEqual(drivers.String("Global Enterprise Store"), val)

					// Check Suppliers group: both settings visible with updated values
					supGroup, ok := groupsMap.Get(GroupSuppliers)
					asserter.Assert(ok, "expected group %q", GroupSuppliers)
					asserter.AssertEqual(2, len(supGroup.Settings))
					asserter.AssertEqual(SettingSupplierEnabled, supGroup.Settings[0].Name())
					val, err = supGroup.Settings[0].Value()
					asserter.Assert(err == nil, "Value() failed: %v", err)
					asserter.AssertEqual(drivers.Bool(true), val)

					asserter.AssertEqual(SettingSupplierStorage, supGroup.Settings[1].Name())
					val, err = supGroup.Settings[1].Value()
					asserter.Assert(err == nil, "Value() failed: %v", err)
					asserter.AssertEqual(drivers.String("local"), val)

					// Check Security group: 0 settings visible for supplier user
					secGroup, ok := groupsMap.Get(GroupSecurity)
					asserter.Assert(ok, "expected group %q", GroupSecurity)
					asserter.AssertEqual(0, len(secGroup.Settings))
				},
			},
		},
	}

	if err := d.Setup(djester.TW(t)); err != nil {
		t.Fatalf("d.Setup failed: %v", err)
	}

	var tables = quest.Table(t, &settings.RegisteredSetting{})
	tables.IfNotExists = true
	tables.Create()
	defer tables.Drop()

	attrs.ResetDefinitions.Send(t.Context(), nil)

	if err := settings.SyncSettingsCmd.Execute(nil, nil, nil); err != nil {
		t.Fatalf("Failed to sync settings: %v", err)
	}

	d.Test(t)
}
