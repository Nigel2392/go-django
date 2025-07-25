package pages

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/contrib/pages/validators"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/mux"
)

const DEFAULT_SITE_KEY = "default_site"

type siteContextKey struct{}

//	func siteForRequestQuerySet(r *http.Request) *queries.QuerySet[*Site] {
//		return queries.GetQuerySet(&Site{}).
//			WithContext(r.Context()).
//			SelectRelated("Root").
//			Annotate("includedInFilter", expr.Case(
//				expr.When("Domain", mux.GetHost(r)).Then(1),
//				expr.When("Default", true).Then(2),
//				expr.When(expr.EXISTS(queries.GetQuerySet(&Site{}).
//					Filter("Domain", mux.GetHost(r)).
//					Filter("AllowFallback", true),
//				)).Then(3),
//			).Default(0)).
//			Filter("includedInFilter__gt", 0).
//			OrderBy("includedInFilter")
//	}

func siteForRequestQuerySet(r *http.Request) *queries.QuerySet[*Site] {
	return queries.GetQuerySet(&Site{}).
		WithContext(r.Context()).
		SelectRelated("Root").
		Filter(expr.Or(
			expr.Q("Domain", mux.GetHost(r)),
			expr.Q("Default", true),
		)).
		OrderBy("Default")
}

func SiteForRequest(r *http.Request, fn ...func(qs *queries.QuerySet[*Site]) *queries.QuerySet[*Site]) (*http.Request, *Site, error) {
	var siteVal = r.Context().Value(siteContextKey{})
	if siteVal != nil {
		var site, ok = siteVal.(*Site)
		if ok {
			return r, site, nil
		}
	}

	var qs = siteForRequestQuerySet(r)
	for _, f := range fn {
		qs = f(qs)
	}

	var row, err = qs.First()
	if err != nil {
		if errors.Is(err, errors.NoRows) {
			return r, nil, errors.NoRows.Wrapf(
				"no site found for request %s", r.URL.String(),
			)
		}
		return r, nil, err
	}

	r = r.WithContext(
		context.WithValue(r.Context(), siteContextKey{}, row.Object),
	)

	return r, row.Object, nil
}

type Site struct {
	models.Model `table:"sites"`
	ID           int64
	Name         string
	Domain       string
	Port         int
	Default      bool
	Root         *PageNode
}

func (n *Site) URL() string {
	var sb = &strings.Builder{}
	sb.WriteString(n.Domain)
	if n.Port != 80 && n.Port != 443 {
		sb.WriteString(fmt.Sprintf(":%d", n.Port))
	}
	return sb.String()
}

func (n *Site) FieldDefs() attrs.Definitions {
	return n.Model.Define(n, n.Fields)
}

func (n *Site) DatabaseIndexes(obj attrs.Definer) []migrator.Index {
	if reflect.TypeOf(obj) != reflect.TypeOf(n) {
		return nil
	}

	return []migrator.Index{
		{Fields: []string{"Domain"}, Unique: true},
	}
}

func (n *Site) Fields(d attrs.Definer) []attrs.Field {
	return []attrs.Field{
		attrs.NewField(n, "ID", &attrs.FieldConfig{
			HelpText: "The unique identifier for the site.",
			Primary:  true,
			Column:   "id",
			ReadOnly: true,
		}),
		attrs.NewField(n, "Name", &attrs.FieldConfig{
			HelpText:  "The name of the site.",
			Column:    "site_name",
			Null:      false,
			Blank:     false,
			MinLength: 2,
			MaxLength: 64,
		}),
		attrs.NewField(n, "Domain", &attrs.FieldConfig{
			HelpText:  "The domain of the site, e.g. example.com.",
			Column:    "domain",
			Null:      false,
			Blank:     false,
			MinLength: 2,
			MaxLength: 256,
			Validators: []func(interface{}) error{
				validators.ValidateSettingsURL,
			},
		}),
		attrs.NewField(n, "Port", &attrs.FieldConfig{
			HelpText: "The port of the site, e.g. 80 for HTTP or 443 for HTTPS.",
			Column:   "port",
			Null:     false,
			Blank:    false,
			Default:  80,
			MinValue: 1,
			MaxValue: 65535,
		}),
		attrs.NewField(n, "Default", &attrs.FieldConfig{
			HelpText: "Whether this site is the default site. Only one site can be the default site.",
			Column:   "is_default_site",
			Null:     false,
			Blank:    true,
			Default:  false,
		}),
		attrs.NewField(n, "Root", &attrs.FieldConfig{
			HelpText: "The root page of the site. This is the page that will be displayed when the site is accessed.",
			Column:   "root_page_id",
			Null:     true,
			RelForeignKey: attrs.Relate(
				&PageNode{}, "", nil,
			),
		}),
	}
}

func (n *Site) Validate(ctx context.Context) error {
	if !n.Default {
		return nil
	}

	var validatorContextVal = ctx.Value(queries.ValidationContext{})
	var validatorCtx, ok = validatorContextVal.(*queries.ValidationContext)
	if !ok {
		return errors.TypeMismatch.Wrapf(
			"expected %T, got %T (%v)",
			&queries.ValidationContext{},
			validatorContextVal,
			validatorContextVal,
		)
	}

	val, ok := validatorCtx.Data[DEFAULT_SITE_KEY]
	if ok {
		var existingSite, ok = val.(*Site)
		if !ok {
			return errors.TypeMismatch.Wrapf(
				"expected %T, got %T", &Site{}, val,
			)
		}

		if existingSite.ID == n.ID {
			return nil
		}

		return errors.CheckFailed.Wrapf(
			"site with ID %d is already set as the default site", existingSite.ID,
		)
	}

	var row, err = queries.GetQuerySet(&Site{}).
		WithContext(validatorCtx).
		Filter("Default", true).
		Get()

	if err != nil && !errors.Is(err, errors.NoRows) {
		return err
	}

	if errors.Is(err, errors.NoRows) || row.Object == nil || (row.Object != nil && row.Object.ID == n.ID) {
		validatorCtx.SetValue(DEFAULT_SITE_KEY, row.Object)
		return nil
	}

	validatorCtx.SetValue(DEFAULT_SITE_KEY, row.Object)
	return errors.CheckFailed.Wrapf(
		"site with ID %d is already set as the default site", row.Object.ID,
	)
}

func (n *Site) BeforeSave(ctx context.Context) error {
	if n.Name == "" {
		n.Name = "Default Settings"
	}

	if n.Domain == "" {
		n.Domain = "localhost"
	}

	return nil
}

func (n *Site) BeforeDelete(ctx context.Context) error {
	if n.Default {
		return errors.CheckFailed.Wrap(
			"cannot delete the default site",
		)
	}

	return nil
}
