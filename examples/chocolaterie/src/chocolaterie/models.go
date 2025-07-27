package chocolaterie

import (
	"net/http"
	"slices"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/contrib/editor"
	"github.com/Nigel2392/go-django/src/contrib/pages"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/shopspring/decimal"
)

type ChocolateContext struct {
	ctx.ContextWithRequest
	Page       *ChocolateListPage
	Chocolates []*Chocolate
}

type Chocolate struct {
	models.Model `table:"chocolaterie_chocolate"`
	ID           int             `json:"id"`
	Name         string          `json:"name"`
	Description  drivers.Text    `json:"description"`
	Price        decimal.Decimal `json:"price"`
	Weight       int             `json:"weight"` // in grams
	Ingredients  drivers.Text    `json:"ingredients"`
}

func (c *Chocolate) FieldDefs() attrs.Definitions {
	return c.Model.Define(c,
		attrs.Unbound("ID", &attrs.FieldConfig{
			Primary:  true,
			ReadOnly: true,
		}),
		attrs.Unbound("Name", &attrs.FieldConfig{
			MinLength: 2,
			MaxLength: 255,
		}),
		attrs.Unbound("Description", &attrs.FieldConfig{
			MinLength: 10,
			MaxLength: 5000,
			FormWidget: func(fc attrs.FieldConfig) widgets.Widget {
				return widgets.NewTextarea(fc.WidgetAttrs)
			},
		}),
		attrs.Unbound("Price", &attrs.FieldConfig{}),
		attrs.Unbound("Weight", &attrs.FieldConfig{
			MinValue: 1,
			MaxValue: 10000, // 10kg max
		}),
		attrs.Unbound("Ingredients", &attrs.FieldConfig{}),
	)
}

type ChocolateListPage struct {
	models.Model
	Page        *pages.PageNode           `proxy:"true"`
	Description *editor.EditorJSBlockData `json:"description"`
}

func (b *ChocolateListPage) ID() int64 {
	return b.Page.PageID
}

func (b *ChocolateListPage) Reference() *pages.PageNode {
	return b.Page
}

func (n *ChocolateListPage) FieldDefs() attrs.Definitions {
	if n.Page == nil {
		n.Page = &pages.PageNode{}
	}
	return n.Model.Define(n,
		attrs.NewField(n.Page, "PageID", &attrs.FieldConfig{
			Primary:  true,
			ReadOnly: true,
			Column:   "id",
		}),
		attrs.NewField(n.Page, "Title", &attrs.FieldConfig{
			Embedded: true,
			Label:    "Title",
			HelpText: trans.S("How do you want your page to be remembered?"),
		}),
		attrs.NewField(n.Page, "UrlPath", &attrs.FieldConfig{
			Embedded: true,
			ReadOnly: true,
			Label:    "URL Path",
			HelpText: trans.S("The URL path for this chocolate list page."),
		}),
		attrs.NewField(n.Page, "Slug", &attrs.FieldConfig{
			Embedded: true,
			Label:    "Slug",
			HelpText: trans.S("The slug for this chocolate list page."),
			Blank:    true,
		}),
		editor.NewField(n, "Description", editor.FieldConfig{
			Label:    trans.S("Description"),
			HelpText: trans.S("This is a rich text editor. You can add images, videos, and other media to your chocolate list page."),
			//Features: []string{
			//	"paragraph",
			//	"text-align",
			//	"list",
			//},
		}),
		attrs.NewField(n.Page, "CreatedAt", &attrs.FieldConfig{
			Embedded: true,
			ReadOnly: true,
			Label:    "Created At",
			HelpText: trans.S("The date and time this chocolate list page was created."),
		}),
		attrs.NewField(n.Page, "UpdatedAt", &attrs.FieldConfig{
			Embedded: true,
			ReadOnly: true,
			Label:    "Updated At",
			HelpText: trans.S("The date and time this chocolate list page was last updated."),
		}),
	)
}

func (b *ChocolateListPage) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var chocolateRows, err = queries.GetQuerySet(&Chocolate{}).
		Filter("Weight__gt", 0).
		All()
	if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"Failed to fetch chocolates: %v", err,
		)
		return
	}

	// Create a new RequestContext
	// Add the page object to the context
	var context ctx.ContextWithRequest = ctx.RequestContext(r)
	context = &ChocolateContext{
		ContextWithRequest: context,
		Page:               b,
		Chocolates: slices.Collect(
			chocolateRows.Objects(),
		),
	}

	// Render the template
	err = tpl.FRender(
		w, context, "chocolaterie",
		"chocolaterie/page.tmpl",
	)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}
