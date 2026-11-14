//go:build test
// +build test

package chooser_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/auth"
	autherrors "github.com/Nigel2392/go-django/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/contrib/chooser"
	"github.com/Nigel2392/go-django/contrib/search"
	_ "github.com/Nigel2392/go-django/contrib/search/backends/searchqueries"
	"github.com/Nigel2392/go-django/contrib/session"
	"github.com/Nigel2392/go-django/djester"
	"github.com/Nigel2392/go-django/djester/quest"
	"github.com/Nigel2392/go-django/djester/testdb"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/models"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/mux/middleware/authentication"
	"github.com/PuerkitoBio/goquery"
)

const (
	ChooserKeyActiveOnly = "active_only"
	ChooserKeyCustomList = "custom_list"
)

type MailDomain struct {
	ID          int `attrs:"primary;readonly"`
	DomainName  string
	Description string
	IsActive    bool
}

func (m *MailDomain) String() string {
	return m.DomainName
}

func (m *MailDomain) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.
		AutoDefinitions[*MailDomain](ctx, m).(*attrs.ObjectDefinitions).
		WithTableName("chooser_test_maildomain")
}

func (m *MailDomain) Save() (err error) {
	if m.ID == 0 {
		_, err = queries.GetQuerySet(m).ExplicitSave().Create(m)
	} else {
		_, err = queries.GetQuerySet(m).ExplicitSave().Update(m)
	}
	return err
}

type Mailbox struct {
	models.Model `table:"chooser_test_mailbox"`
	ID           int `attrs:"primary;readonly"`
	Address      string
	Domain       *MailDomain
}

func (m *Mailbox) String() string {
	return m.Address
}

func (m *Mailbox) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.FieldConfig{
			Primary:  true,
			ReadOnly: true,
		}),
		attrs.NewField(m, "Address", &attrs.FieldConfig{
			Label: trans.S("Email Address"),
		}),
		attrs.NewField(m, "Domain", &attrs.FieldConfig{
			RelForeignKey: attrs.Relate(&MailDomain{}, "", nil),
			Column:        "domain_id",
			FormWidget: func(fc attrs.FieldConfig) widgets.Widget {
				return chooser.NewChooserWidget(
					fc.RelForeignKey.Model(),
					fc.WidgetAttrs,
					map[string]string{"data-custom-chooser-attr": "domain-selector"},
					chooser.DEFAULT_KEY,
				)
			},
			Label: trans.S("Mail Domain"),
		}),
	)
}

type UnregisteredModel struct {
	ID int `attrs:"primary"`
}

func (u *UnregisteredModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions[*UnregisteredModel](ctx, u)
}

type User struct {
	Name            string `attrs:"primary"`
	LoggedIn        bool
	IsAdministrator bool
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

type customWrappedList struct {
	Models []*chooser.WrappedModel[*MailDomain]
}

func (c *customWrappedList) Render() string {
	var sb strings.Builder
	sb.WriteString(`<ul class="custom-wrapped-chooser-list">`)
	for _, wm := range c.Models {
		dataBytes, _ := json.Marshal(wm.ExtraData())
		sb.WriteString(fmt.Sprintf(
			`<li class="godjango-chooser-list-group" data-chooser-value="%d" data-chooser-preview="%s" data-chooser-data='%s'>%s</li>`,
			wm.Model.ID,
			wm.PreviewHTML(),
			string(dataBytes),
			wm.Model.DomainName,
		))
	}
	sb.WriteString(`</ul>`)
	return sb.String()
}

var defaultDomainChooserDef *chooser.ChooserDefinition[*MailDomain]

func registerTestChoosers() {
	defaultDomainChooserDef = &chooser.ChooserDefinition[*MailDomain]{
		Model: &MailDomain{},
		Title: trans.S("Choose Mail Domain"),
		Labels: map[string]func(ctx context.Context) string{
			"DomainName": trans.S("Domain Name"),
		},
		PreviewString: func(ctx context.Context, instance *MailDomain) string {
			return fmt.Sprintf("Domain(%s)", instance.DomainName)
		},
		ExtraData: func(ctx context.Context, instance *MailDomain) map[string]any {
			return map[string]any{
				"domain": instance.DomainName,
				"active": instance.IsActive,
			}
		},
		MediaFn: func() media.Media {
			m := media.NewMedia()
			m.AddCSS(media.CSS("/static/chooser/custom-domain.css"))
			return m
		},
		ListPage: &chooser.ChooserListPage[*MailDomain]{
			PerPage: 2,
			Fields:  []string{"ID", "DomainName", "Description"},
			Labels: map[string]func(ctx context.Context) string{
				"Description": trans.S("Domain Description"),
				"search":      trans.S("Filter Domains"),
				"search...":   trans.S("Type domain name..."),
			},
			Format: map[string]func(v any) any{
				"Description": func(v any) any {
					if s, ok := v.(string); ok {
						return strings.ToUpper(s)
					}
					return v
				},
			},
			SearchFields: []search.SearchField{
				search.NewSearchField(2, "DomainName", expr.LOOKUP_ICONTANS),
				search.NewSearchField(1, "Description", expr.LOOKUP_ICONTANS),
			},
			QuerySet: func(r *http.Request, model *MailDomain) (*queries.QuerySet[*MailDomain], error) {
				return queries.GetQuerySet(model).Select("*").OrderBy("ID"), nil
			},
		},
	}

	chooser.Register(defaultDomainChooserDef)

	// Second chooser on the same model with a custom key: filters to active domains only, custom SearchQueryVar, and custom Filter hook
	chooser.Register(&chooser.ChooserDefinition[*MailDomain]{
		Model: &MailDomain{},
		Title: "Choose Active Domain",
		PreviewString: func(ctx context.Context, instance *MailDomain) string {
			return fmt.Sprintf("ActiveDomain(%s)", instance.DomainName)
		},
		ListPage: &chooser.ChooserListPage[*MailDomain]{
			PerPage:        10,
			SearchQueryVar: "q",
			Fields:         []string{"ID", "DomainName"},
			SearchFields: []search.SearchField{
				search.NewSearchField(1, "DomainName", expr.LOOKUP_ICONTANS),
			},
			QuerySet: func(r *http.Request, model *MailDomain) (*queries.QuerySet[*MailDomain], error) {
				return queries.GetQuerySet(model).Select("*").Filter("IsActive", true).OrderBy("ID"), nil
			},
			Filter: func(qs *queries.QuerySet[*MailDomain], req *http.Request, searchValue string) *queries.QuerySet[*MailDomain] {
				if exclude := req.URL.Query().Get("exclude_domain"); exclude != "" {
					qs = qs.Filter(expr.Q("DomainName", exclude).Not(true))
				}
				return qs
			},
		},
	}, ChooserKeyActiveOnly)

	// Third chooser on the same model using NewList + WrapModels
	chooser.Register(&chooser.ChooserDefinition[*MailDomain]{
		Model: &MailDomain{},
		Title: trans.S("Custom Wrapped Domain Chooser"),
		PreviewString: func(ctx context.Context, instance *MailDomain) string {
			return fmt.Sprintf("Wrapped(%s)", instance.DomainName)
		},
		ExtraData: func(ctx context.Context, instance *MailDomain) map[string]any {
			return map[string]any{"wrapped_id": instance.ID}
		},
		ListPage: &chooser.ChooserListPage[*MailDomain]{
			PerPage: 10,
			QuerySet: func(r *http.Request, model *MailDomain) (*queries.QuerySet[*MailDomain], error) {
				return queries.GetQuerySet(model).Select("*").OrderBy("ID"), nil
			},
			NewList: func(req *http.Request, results []*MailDomain, def *chooser.ChooserDefinition[*MailDomain]) any {
				return &customWrappedList{
					Models: chooser.WrapModels(req.Context(), def, results),
				}
			},
		},
	}, ChooserKeyCustomList)

	// Chooser for Mailbox model with minimal/default ListPage configuration
	chooser.Register(&chooser.ChooserDefinition[*Mailbox]{
		Model: &Mailbox{},
		Title: trans.S("Choose Mailbox"),
	})
}

func initAdminApp() django.AppConfig {
	admin.ConfigureAuth(admin.AuthConfig{
		GetLoginForm: func(r *http.Request, formOpts ...func(forms.Form)) admin.LoginForm {
			return auth.UserLoginForm(r, formOpts...)
		},
		Logout: auth.Logout,
	})

	attrs.RegisterModel(&MailDomain{})
	attrs.RegisterModel(&Mailbox{})
	attrs.RegisterModel(&User{})

	admin.RegisterApp(
		"chooser_test",
		admin.AppOptions{
			EnableIndexView: true,
		},
		admin.ModelOptions{Name: "MailDomain", Model: &MailDomain{}},
		admin.ModelOptions{Name: "Mailbox", Model: &Mailbox{}},
		admin.ModelOptions{Name: "User", Model: &User{}},
	)

	autherrors.RegisterHook("index")
	return admin.NewAppConfig()
}

func seedDomains(t *testing.T, asserter djester.Assertion) []*MailDomain {
	t.Helper()
	_, err := queries.GetQuerySet(&Mailbox{}).Delete()
	asserter.Assert(err == nil, "Failed to clear mailboxes: %v", err)
	_, err = queries.GetQuerySet(&MailDomain{}).Delete()
	asserter.Assert(err == nil, "Failed to clear domains: %v", err)

	seed := []*MailDomain{
		{DomainName: "alpha.example.com", Description: "primary alpha mail", IsActive: true},
		{DomainName: "beta.example.org", Description: "secondary beta relay", IsActive: true},
		{DomainName: "gamma.internal.net", Description: "legacy internal domain", IsActive: false},
		{DomainName: "delta.example.com", Description: "delta transactional mail", IsActive: true},
		{DomainName: "epsilon.archive.io", Description: "archived cold storage", IsActive: false},
	}

	for _, d := range seed {
		err := d.Save()
		asserter.Assert(err == nil, "Failed to save seed domain %q: %v", d.DomainName, err)
	}
	return seed
}

func TestChooser(t *testing.T) {
	var _, db = testdb.Open()
	defer db.Close()

	registerTestChoosers()

	d := &djester.Tester{
		Settings: map[string]any{
			django.APPVAR_ALLOWED_HOSTS:  []string{"*"},
			django.APPVAR_DISABLE_NOSURF: true,
			django.APPVAR_DEBUG:          true,
			django.APPVAR_DATABASE:       db,
		},
		Flags: []django.AppFlag{
			django.FlagSkipCmds,
			django.FlagSkipChecks,
			django.FlagSkipDepsCheck,
		},
		Apps: []djester.AppInitFuncOrAppConfig{
			session.NewAppConfig,
			initAdminApp,
			chooser.NewAppConfig,
			&apps.AppConfig{
				AppName: "chooser_test",
				ModelObjects: []attrs.Definer{
					&MailDomain{},
					&Mailbox{},
					&User{},
				},
			},
		},
		Auth: &djester.TesterAuth{
			UnauthenticatedUser: func() authentication.User {
				return &User{LoggedIn: false, IsAdministrator: false}
			},
			Users: map[string]authentication.User{
				"admin_user": &User{
					Name:            "admin_user",
					LoggedIn:        true,
					IsAdministrator: true,
				},
			},
		},
		Tests: []djester.Test{
			&djester.BasicTest{
				Label: "Registration panics and ChooserDefinition Setup validation",
				Function: func(d *djester.Tester, t *testing.T) {
					asserter := djester.Asserter(t, true)

					// 1. Duplicate registration with same model and key must panic
					dupDef := &chooser.ChooserDefinition[*MailDomain]{
						Model: &MailDomain{},
						Title: "Duplicate Test Key",
					}
					asserter.Assert(dupDef.Setup("dup_test_key") == nil, "Setup for dup_test_key failed")
					chooser.Register(dupDef, "dup_test_key")

					didPanic := false
					func() {
						defer func() {
							if r := recover(); r != nil {
								didPanic = true
							}
						}()
						chooser.Register(dupDef, "dup_test_key")
					}()
					asserter.Assert(didPanic, "expected chooser.Register to panic when registering duplicate key for same model")

					// 2. Setup validation: nil Title returns ValueError
					defNilTitle := &chooser.ChooserDefinition[*MailDomain]{
						Model: &MailDomain{},
						Title: nil,
					}
					err := defNilTitle.Setup("nil_title")
					asserter.Assert(err != nil, "expected error when ChooserDefinition.Title is nil")
					asserter.Assert(errors.Is(err, errors.ValueError), "expected ValueError for nil Title, got: %v", err)

					// 3. Setup validation: nil Model returns TypeMismatch
					defNilModel := &chooser.ChooserDefinition[*MailDomain]{
						Model: nil,
						Title: "Nil Model",
					}
					err = defNilModel.Setup("nil_model")
					asserter.Assert(err != nil, "expected error when ChooserDefinition.Model is nil")
					asserter.Assert(errors.Is(err, errors.TypeMismatch), "expected TypeMismatch for nil Model, got: %v", err)

					// 4. Setup validation: Model not registered to any Django app returns TypeMismatch
					defUnregistered := &chooser.ChooserDefinition[*UnregisteredModel]{
						Model: &UnregisteredModel{},
						Title: "Unregistered Model",
					}
					err = defUnregistered.Setup("unregistered")
					asserter.Assert(err != nil, "expected error when Model is not attached to a Django app")
					asserter.Assert(errors.Is(err, errors.TypeMismatch), "expected TypeMismatch for unregistered model, got: %v", err)

					// 5. WrapModel and WrapModels behavior
					asserter.Assert(chooser.WrapModel(t.Context(), defaultDomainChooserDef, nil) == nil, "expected WrapModel with nil model to return nil")
					sample := &MailDomain{ID: 42, DomainName: "wrapped.example.com", IsActive: true}
					wrappedSlice := chooser.WrapModels(t.Context(), defaultDomainChooserDef, []*MailDomain{nil, sample})
					asserter.AssertEqual(1, len(wrappedSlice))
					asserter.AssertEqual("Domain(wrapped.example.com)", wrappedSlice[0].PreviewHTML())
					asserter.AssertEqual("wrapped.example.com", wrappedSlice[0].ExtraData()["domain"])
					asserter.AssertEqual(true, wrappedSlice[0].ExtraData()["active"])
				},
			},
			&djester.BasicTest{
				Label: "Chooser list view rendering, extra data, column formatting, and pagination",
				Function: func(d *djester.Tester, t *testing.T) {
					asserter := djester.Asserter(t, true)
					domains := seedDomains(t, asserter)

					// Page 1 (PerPage = 2 -> should contain domains[0] and domains[1])
					var page1Resp chooser.ChooserResponse
					resp, err := d.Get("/chooser/chooser_test.MailDomain/default/list/", nil, nil)
					asserter.Assert(err == nil, "GET /chooser/chooser_test.MailDomain/default/list/ failed: %v", err)
					asserter.AssertEqual(http.StatusOK, resp.StatusCode)
					err = resp.JSON(&page1Resp)
					asserter.Assert(err == nil, "Failed to decode ChooserResponse JSON: %v", err)

					asserter.AssertHTMLString(
						page1Resp.HTML,
						// Search form is enabled with custom placeholder and button text
						djester.HasElement(`form.godjango-chooser-list-form input.godjango-chooser-list-search-input[name="search"][placeholder="Type domain name..."]`),
						djester.HasText(`button.godjango-chooser-list-search-button`, "Filter Domains"),
						// Custom labels on headers
						djester.HasText(`th`, "Domain Name"),
						djester.HasText(`th`, "Domain Description"),
						// Formatted uppercase descriptions for page 1 rows
						djester.HasText(`td`, "PRIMARY ALPHA MAIL"),
						djester.HasText(`td`, "SECONDARY BETA RELAY"),
						// Pagination links with chooser-link class
						djester.HasElement(`.pagination a`),
						func(doc *goquery.Document) error {
							rows := doc.Find("tr.godjango-chooser-list-group")
							if rows.Length() != 2 {
								return fmt.Errorf("expected 2 rows on page 1 (PerPage=2), got %d", rows.Length())
							}

							for i := 0; i < 2; i++ {
								expected := domains[i]
								rowSel := doc.Find(fmt.Sprintf(`tr.godjango-chooser-list-group[data-chooser-value="%d"]`, expected.ID))
								if rowSel.Length() != 1 {
									return fmt.Errorf("expected row with data-chooser-value=%d", expected.ID)
								}
								preview, _ := rowSel.Attr("data-chooser-preview")
								wantPreview := fmt.Sprintf("Domain(%s)", expected.DomainName)
								if preview != wantPreview {
									return fmt.Errorf("expected data-chooser-preview=%q, got %q", wantPreview, preview)
								}
								dataAttr, ok := rowSel.Attr("data-chooser-data")
								if !ok {
									return fmt.Errorf("missing data-chooser-data attribute on row %d", expected.ID)
								}
								var extra map[string]any
								if err := json.Unmarshal([]byte(dataAttr), &extra); err != nil {
									return fmt.Errorf("failed to unmarshal data-chooser-data %q: %w", dataAttr, err)
								}
								if extra["domain"] != expected.DomainName || extra["active"] != expected.IsActive {
									return fmt.Errorf("unexpected extra data for %s: %v", expected.DomainName, extra)
								}
							}

							// Ensure page 2/3 items are not on page 1
							for i := 2; i < len(domains); i++ {
								if doc.Find(fmt.Sprintf(`tr.godjango-chooser-list-group[data-chooser-value="%d"]`, domains[i].ID)).Length() > 0 {
									return fmt.Errorf("did not expect domain %d (%s) on page 1", domains[i].ID, domains[i].DomainName)
								}
							}
							return nil
						},
					)

					// Page 2 (should contain domains[2] and domains[3])
					var page2Resp chooser.ChooserResponse
					resp, err = d.Get("/chooser/chooser_test.MailDomain/default/list/", nil, url.Values{"page": {"2"}})
					asserter.Assert(err == nil, "GET page 2 failed: %v", err)
					asserter.AssertEqual(http.StatusOK, resp.StatusCode)
					err = resp.JSON(&page2Resp)
					asserter.Assert(err == nil, "Failed to decode page 2 JSON: %v", err)

					asserter.AssertHTMLString(
						page2Resp.HTML,
						djester.HasText(`td`, "LEGACY INTERNAL DOMAIN"),
						djester.HasText(`td`, "DELTA TRANSACTIONAL MAIL"),
						func(doc *goquery.Document) error {
							rows := doc.Find("tr.godjango-chooser-list-group")
							if rows.Length() != 2 {
								return fmt.Errorf("expected 2 rows on page 2, got %d", rows.Length())
							}
							for _, idx := range []int{2, 3} {
								if doc.Find(fmt.Sprintf(`tr.godjango-chooser-list-group[data-chooser-value="%d"]`, domains[idx].ID)).Length() != 1 {
									return fmt.Errorf("expected domain %d (%s) on page 2", domains[idx].ID, domains[idx].DomainName)
								}
							}
							return nil
						},
					)

					// Page 3 (should contain domains[4] only)
					var page3Resp chooser.ChooserResponse
					resp, err = d.Get("/chooser/chooser_test.MailDomain/default/list/", nil, url.Values{"page": {"3"}})
					asserter.Assert(err == nil, "GET page 3 failed: %v", err)
					asserter.AssertEqual(http.StatusOK, resp.StatusCode)
					err = resp.JSON(&page3Resp)
					asserter.Assert(err == nil, "Failed to decode page 3 JSON: %v", err)

					asserter.AssertHTMLString(
						page3Resp.HTML,
						djester.HasText(`td`, "ARCHIVED COLD STORAGE"),
						func(doc *goquery.Document) error {
							rows := doc.Find("tr.godjango-chooser-list-group")
							if rows.Length() != 1 {
								return fmt.Errorf("expected 1 row on page 3, got %d", rows.Length())
							}
							if doc.Find(fmt.Sprintf(`tr.godjango-chooser-list-group[data-chooser-value="%d"]`, domains[4].ID)).Length() != 1 {
								return fmt.Errorf("expected domain %d (%s) on page 3", domains[4].ID, domains[4].DomainName)
							}
							return nil
						},
					)
				},
			},
			&djester.BasicTest{
				Label: "Chooser list search, custom QuerySet, custom Filter, and multiple chooser keys",
				Function: func(d *djester.Tester, t *testing.T) {
					asserter := djester.Asserter(t, true)
					domains := seedDomains(t, asserter)

					// 1. Search on default chooser (?search=example.com -> matches alpha.example.com and delta.example.com)
					var searchResp chooser.ChooserResponse
					resp, err := d.Get("/chooser/chooser_test.MailDomain/default/list/", nil, url.Values{"search": {"example.com"}})
					asserter.Assert(err == nil, "GET search failed: %v", err)
					asserter.AssertEqual(http.StatusOK, resp.StatusCode)
					err = resp.JSON(&searchResp)
					asserter.Assert(err == nil, "Decode search JSON failed: %v", err)

					asserter.AssertHTMLString(
						searchResp.HTML,
						djester.HasElement(`input.godjango-chooser-list-search-input[name="search"][value="example.com"]`),
						func(doc *goquery.Document) error {
							rows := doc.Find("tr.godjango-chooser-list-group")
							if rows.Length() != 2 {
								return fmt.Errorf("expected 2 matching rows for 'example.com', got %d", rows.Length())
							}
							if doc.Find(fmt.Sprintf(`tr.godjango-chooser-list-group[data-chooser-value="%d"]`, domains[0].ID)).Length() != 1 {
								return fmt.Errorf("expected alpha.example.com in search results")
							}
							if doc.Find(fmt.Sprintf(`tr.godjango-chooser-list-group[data-chooser-value="%d"]`, domains[3].ID)).Length() != 1 {
								return fmt.Errorf("expected delta.example.com in search results")
							}
							return nil
						},
					)

					// 2. Second chooser key ("active_only") filters to IsActive=true via QuerySet (3 active domains)
					var activeResp chooser.ChooserResponse
					resp, err = d.Get(fmt.Sprintf("/chooser/chooser_test.MailDomain/%s/list/", ChooserKeyActiveOnly), nil, nil)
					asserter.Assert(err == nil, "GET active_only chooser failed: %v", err)
					asserter.AssertEqual(http.StatusOK, resp.StatusCode)
					err = resp.JSON(&activeResp)
					asserter.Assert(err == nil, "Decode active_only JSON failed: %v", err)

					asserter.AssertHTMLString(
						activeResp.HTML,
						djester.HasElement(`input.godjango-chooser-list-search-input[name="q"]`),
						func(doc *goquery.Document) error {
							rows := doc.Find("tr.godjango-chooser-list-group")
							if rows.Length() != 3 {
								return fmt.Errorf("expected 3 active rows, got %d", rows.Length())
							}
							for _, dom := range domains {
								found := doc.Find(fmt.Sprintf(`tr.godjango-chooser-list-group[data-chooser-value="%d"]`, dom.ID)).Length() > 0
								if dom.IsActive && !found {
									return fmt.Errorf("expected active domain %s to be present", dom.DomainName)
								}
								if !dom.IsActive && found {
									return fmt.Errorf("expected inactive domain %s NOT to be present", dom.DomainName)
								}
							}
							return nil
						},
					)

					// 3. Custom Filter + custom SearchQueryVar ("?q=example&exclude_domain=alpha.example.com") on "active_only"
					// Active domains matching "example" are alpha.example.com, beta.example.org, delta.example.com;
					// excluding alpha.example.com leaves beta.example.org and delta.example.com.
					var filteredResp chooser.ChooserResponse
					resp, err = d.Get(fmt.Sprintf("/chooser/chooser_test.MailDomain/%s/list/", ChooserKeyActiveOnly), nil, url.Values{
						"q":              {"example"},
						"exclude_domain": {"alpha.example.com"},
					})
					asserter.Assert(err == nil, "GET filtered active_only chooser failed: %v", err)
					asserter.AssertEqual(http.StatusOK, resp.StatusCode)
					err = resp.JSON(&filteredResp)
					asserter.Assert(err == nil, "Decode filtered active_only JSON failed: %v", err)

					asserter.AssertHTMLString(
						filteredResp.HTML,
						djester.HasElement(`input.godjango-chooser-list-search-input[name="q"][value="example"]`),
						func(doc *goquery.Document) error {
							rows := doc.Find("tr.godjango-chooser-list-group")
							if rows.Length() != 2 {
								return fmt.Errorf("expected 2 rows after q=example & exclude_domain=alpha.example.com, got %d", rows.Length())
							}
							if doc.Find(fmt.Sprintf(`tr.godjango-chooser-list-group[data-chooser-value="%d"]`, domains[1].ID)).Length() != 1 {
								return fmt.Errorf("expected beta.example.org to be present")
							}
							if doc.Find(fmt.Sprintf(`tr.godjango-chooser-list-group[data-chooser-value="%d"]`, domains[3].ID)).Length() != 1 {
								return fmt.Errorf("expected delta.example.com to be present")
							}
							return nil
						},
					)

					// 4. Third chooser key ("custom_list") using custom NewList + WrapModels
					var customListResp chooser.ChooserResponse
					resp, err = d.Get(fmt.Sprintf("/chooser/chooser_test.MailDomain/%s/list/", ChooserKeyCustomList), nil, nil)
					asserter.Assert(err == nil, "GET custom_list chooser failed: %v", err)
					asserter.AssertEqual(http.StatusOK, resp.StatusCode)
					err = resp.JSON(&customListResp)
					asserter.Assert(err == nil, "Decode custom_list JSON failed: %v", err)

					asserter.AssertHTMLString(
						customListResp.HTML,
						djester.HasElement(`ul.custom-wrapped-chooser-list`),
						func(doc *goquery.Document) error {
							// SearchFields is empty on custom_list, so search form must not be rendered
							if doc.Find("form.godjango-chooser-list-form").Length() != 0 {
								return fmt.Errorf("expected no search form when SearchFields is empty")
							}
							items := doc.Find("ul.custom-wrapped-chooser-list li.godjango-chooser-list-group")
							if items.Length() != len(domains) {
								return fmt.Errorf("expected %d custom list items, got %d", len(domains), items.Length())
							}
							first := doc.Find(fmt.Sprintf(`li.godjango-chooser-list-group[data-chooser-value="%d"]`, domains[0].ID))
							preview, _ := first.Attr("data-chooser-preview")
							if preview != fmt.Sprintf("Wrapped(%s)", domains[0].DomainName) {
								return fmt.Errorf("expected Wrapped preview, got %q", preview)
							}
							return nil
						},
					)

					// 5. Unknown model_key and chooser_key return 404
					resp404Model, err := d.Get("/chooser/NonExistentModel/default/list/", nil, nil)
					asserter.Assert(err == nil, "GET unknown model failed: %v", err)
					asserter.AssertEqual(http.StatusNotFound, resp404Model.StatusCode)
					resp404Model.Body.Close()

					resp404Key, err := d.Get("/chooser/chooser_test.MailDomain/nonexistent_key/list/", nil, nil)
					asserter.Assert(err == nil, "GET unknown chooser key failed: %v", err)
					asserter.AssertEqual(http.StatusNotFound, resp404Key.StatusCode)
					resp404Key.Body.Close()
				},
			},
			&djester.BasicTest{
				Label: "ChooserWidget rendering, ValueToGo, Validate, Media, and form submission",
				Function: func(d *djester.Tester, t *testing.T) {
					asserter := djester.Asserter(t, true)
					domains := seedDomains(t, asserter)
					targetDomain := domains[1] // beta.example.org

					widget := chooser.NewChooserWidget(
						&MailDomain{},
						map[string]string{"class": "domain-input-field"},
						map[string]string{"data-extra-test": "chooser-custom"},
						chooser.DEFAULT_KEY,
					)

					// 1. Media includes chooser CSS, JS, and definition's custom CSS
					m := widget.Media()
					cssList := m.CSS()
					jsList := m.JS()
					asserter.Assert(len(cssList) >= 2, "expected at least 2 CSS assets on ChooserWidget, got %d", len(cssList))
					asserter.Assert(len(jsList) >= 1, "expected at least 1 JS asset on ChooserWidget, got %d", len(jsList))

					// 2. ValueToGo conversions
					goValNil, err := widget.ValueToGo("")
					asserter.Assert(err == nil, "ValueToGo(\"\") failed: %v", err)
					asserter.Assert(goValNil == nil, "expected nil for empty string ValueToGo, got %v", goValNil)

					goValFromPK, err := widget.ValueToGo(fmt.Sprintf("%d", targetDomain.ID))
					asserter.Assert(err == nil, "ValueToGo(PK string) failed: %v", err)
					domainFromPK, ok := goValFromPK.(*MailDomain)
					asserter.Assert(ok, "expected *MailDomain from ValueToGo, got %T", goValFromPK)
					asserter.AssertEqual(targetDomain.ID, domainFromPK.ID)

					goValPassthrough, err := widget.ValueToGo(targetDomain)
					asserter.Assert(err == nil, "ValueToGo(Definer) failed: %v", err)
					asserter.AssertEqual(targetDomain, goValPassthrough.(*MailDomain))

					// 3. Validate against DB
					errsEmpty := widget.Validate(t.Context(), nil)
					asserter.AssertEqual(0, len(errsEmpty))

					errsValid := widget.Validate(t.Context(), targetDomain.ID)
					asserter.AssertEqual(0, len(errsValid))

					errsInvalid := widget.Validate(t.Context(), 999999)
					asserter.AssertEqual(1, len(errsInvalid))
					asserter.Assert(errors.Is(errsInvalid[0], errors.NoRows), "expected errors.NoRows for non-existent PK, got: %v", errsInvalid[0])

					// 4. Render widget with empty value
					var bufEmpty bytes.Buffer
					err = widget.Render(t.Context(), &bufEmpty, "id_domain", "domain", nil, nil)
					asserter.Assert(err == nil, "widget.Render(nil) failed: %v", err)
					asserter.AssertHTMLString(
						bufEmpty.String(),
						djester.HasElement(`div.chooser-widget.id_domain-chooser[data-controller="chooser"][data-chooser-title-value="Choose Mail Domain"][data-extra-test="chooser-custom"]`),
						djester.HasElement(`div.chooser-widget[data-chooser-listurl-value="/chooser/chooser_test.MailDomain/default/list/"]`),
						djester.HasElement(`input[type="hidden"][data-chooser-target="input"][name="domain"][id="id_domain"]`),
						djester.HasElement(`button.chooser-choose[data-chooser-target="open"]`),
						djester.HasElement(`button.chooser-reset[data-chooser-target="clear"]`),
						func(doc *goquery.Document) error {
							input := doc.Find(`input#id_domain`)
							if _, hasVal := input.Attr("value"); hasVal {
								return fmt.Errorf("expected no value attribute on empty chooser widget input")
							}
							preview := strings.TrimSpace(doc.Find(`[data-chooser-target="preview"]`).Text())
							if preview != "" {
								return fmt.Errorf("expected empty preview on empty chooser widget, got %q", preview)
							}
							return nil
						},
					)

					// 5. Render widget with existing DB PK value (resolves preview string from DB)
					var bufWithVal bytes.Buffer
					err = widget.Render(t.Context(), &bufWithVal, "id_domain", "domain", targetDomain.ID, nil)
					asserter.Assert(err == nil, "widget.Render(targetDomain.ID) failed: %v", err)
					asserter.AssertHTMLString(
						bufWithVal.String(),
						djester.HasElement(fmt.Sprintf(`input#id_domain[value="%d"]`, targetDomain.ID)),
						djester.HasText(`[data-chooser-target="preview"]`, fmt.Sprintf("Domain(%s)", targetDomain.DomainName)),
					)

					// 6. Render widget with non-existent DB PK value (falls back to empty value & preview)
					var bufMissing bytes.Buffer
					err = widget.Render(t.Context(), &bufMissing, "id_domain", "domain", 888888, nil)
					asserter.Assert(err == nil, "widget.Render(888888) failed: %v", err)
					asserter.AssertHTMLString(
						bufMissing.String(),
						func(doc *goquery.Document) error {
							if _, hasVal := doc.Find(`input#id_domain`).Attr("value"); hasVal {
								return fmt.Errorf("expected no value attribute when DB row does not exist")
							}
							if strings.TrimSpace(doc.Find(`[data-chooser-target="preview"]`).Text()) != "" {
								return fmt.Errorf("expected empty preview when DB row does not exist")
							}
							return nil
						},
					)

					// 7. End-to-end form submission with ChooserWidget (valid PK vs non-existent PK)
					validReq, err := http.NewRequestWithContext(
						t.Context(),
						http.MethodPost,
						"/test-form",
						strings.NewReader(fmt.Sprintf("domain=%d", targetDomain.ID)),
					)
					asserter.Assert(err == nil, "NewRequest failed: %v", err)
					validReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

					validForm := forms.NewBaseForm(
						t.Context(),
						forms.WithRequestData(http.MethodPost, validReq),
						forms.WithFields(fields.CharField(
							fields.Name("domain"),
							fields.Required(true),
							fields.Widget(chooser.NewChooserWidget(&MailDomain{}, nil, nil, chooser.DEFAULT_KEY)),
						)),
					)
					asserter.Assert(forms.IsValid(t.Context(), validForm), "expected form with valid chooser PK to be valid, errors: %v", validForm.BoundErrors())
					cleanedDomain, ok := validForm.CleanedData()["domain"].(*MailDomain)
					asserter.Assert(ok, "expected CleanedData['domain'] to be *MailDomain, got %T", validForm.CleanedData()["domain"])
					asserter.AssertEqual(targetDomain.ID, cleanedDomain.ID)

					invalidReq, err := http.NewRequestWithContext(
						t.Context(),
						http.MethodPost,
						"/test-form",
						strings.NewReader("domain=999999"),
					)
					asserter.Assert(err == nil, "NewRequest failed: %v", err)
					invalidReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

					invalidForm := forms.NewBaseForm(
						t.Context(),
						forms.WithRequestData(http.MethodPost, invalidReq),
						forms.WithFields(fields.CharField(
							fields.Name("domain"),
							fields.Required(true),
							fields.Widget(chooser.NewChooserWidget(&MailDomain{}, nil, nil, chooser.DEFAULT_KEY)),
						)),
					)
					asserter.Assert(!forms.IsValid(t.Context(), invalidForm), "expected form with non-existent chooser PK to fail validation")
					domainErrs, hasDomainErrs := invalidForm.BoundErrors().Get("domain")
					asserter.Assert(hasDomainErrs && len(domainErrs) > 0, "expected field error on 'domain' for non-existent PK")
				},
			},
			&djester.BasicTest{
				Label: "Admin ModelForm integration with ForeignKey field using ChooserWidget",
				Function: func(d *djester.Tester, t *testing.T) {
					asserter := djester.Asserter(t, true)
					domains := seedDomains(t, asserter)
					chosenDomain := domains[3] // delta.example.com

					_, err := d.Login("admin_user")
					asserter.Assert(err == nil, "Login failed: %v", err)
					defer d.Logout()

					// 1. GET admin add page for Mailbox -> renders ChooserWidget for Domain field
					resp, err := d.Get("/admin/apps/chooser_test/Mailbox/add", nil, nil)
					asserter.Assert(err == nil, "GET Mailbox add page failed: %v", err)
					asserter.AssertEqual(http.StatusOK, resp.StatusCode)
					resp.Assert(t, true).AssertHTML(
						djester.HasElement(`div.chooser-widget[data-controller="chooser"][data-chooser-listurl-value="/chooser/chooser_test.MailDomain/default/list/"][data-custom-chooser-attr="domain-selector"]`),
						djester.HasElement(`input[type="hidden"][name="Domain"]`),
					)
					resp.Body.Close()

					// 2. POST admin add page for Mailbox selecting chosenDomain.ID via ChooserWidget
					resp, err = d.PostForm("/admin/apps/chooser_test/Mailbox/add", nil, nil, map[string]any{
						"Address": "postmaster@delta.example.com",
						"Domain":  fmt.Sprintf("%d", chosenDomain.ID),
					})
					asserter.Assert(err == nil, "POST Mailbox add failed: %v", err)
					asserter.AssertEqual(http.StatusOK, resp.StatusCode)
					resp.Body.Close()

					// 3. Verify Mailbox was saved in DB with the chosen MailDomain ForeignKey
					mbRow, err := queries.GetQuerySet(&Mailbox{}).
						Select("ID", "Address", "Domain.*").
						Filter("Address", "postmaster@delta.example.com").
						Get()
					asserter.Assert(err == nil, "Failed to query created Mailbox: %v", err)
					asserter.AssertEqual("postmaster@delta.example.com", mbRow.Object.Address)
					asserter.Assert(mbRow.Object.Domain != nil, "Expected Mailbox.Domain to be populated")
					asserter.AssertEqual(chosenDomain.ID, mbRow.Object.Domain.ID)
					asserter.AssertEqual(chosenDomain.DomainName, mbRow.Object.Domain.DomainName)

					// 4. GET admin edit page for the created Mailbox -> renders ChooserWidget pre-populated with chosenDomain PK and preview
					editURL := fmt.Sprintf("/admin/apps/chooser_test/Mailbox/edit/%d", mbRow.Object.ID)
					resp, err = d.Get(editURL, nil, nil)
					asserter.Assert(err == nil, "GET Mailbox edit page failed: %v", err)
					asserter.AssertEqual(http.StatusOK, resp.StatusCode)
					resp.Assert(t, true).AssertHTML(
						djester.HasElement(fmt.Sprintf(`input[type="hidden"][name="Domain"][value="%d"]`, chosenDomain.ID)),
						djester.HasText(`[data-chooser-target="preview"]`, fmt.Sprintf("Domain(%s)", chosenDomain.DomainName)),
					)
					resp.Body.Close()
				},
			},
		},
	}

	if err := d.Setup(djester.TW(t)); err != nil {
		t.Fatalf("d.Setup failed: %v", err)
	}

	var tables = quest.Table(t, &MailDomain{}, &Mailbox{})
	tables.IfNotExists = true
	tables.Create()
	defer tables.Drop()

	attrs.ResetDefinitions.Send(t.Context(), nil)

	d.Test(t)
}
