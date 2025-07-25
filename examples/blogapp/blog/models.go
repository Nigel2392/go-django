package blog

import (
	"net/http"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/contrib/editor"
	"github.com/Nigel2392/go-django/src/contrib/pages"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/trans"
)

var (
	_ models.SaveableObject   = (*BlogPage)(nil)
	_ models.DeleteableObject = (*BlogPage)(nil)
)

type BlogContext struct {
	ctx.ContextWithRequest
	Page pages.Page
}

type BlogPage struct {
	models.Model `table:"blog_pages"`
	Page         *pages.PageNode `proxy:"true"`
	Editor       *editor.EditorJSBlockData
}

func (b *BlogPage) ID() int64 {
	return b.Page.PageID
}

func (b *BlogPage) Reference() *pages.PageNode {
	return b.Page
}

func (b *BlogPage) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// Create a new RequestContext
	// Add the page object to the context
	var context ctx.ContextWithRequest = ctx.RequestContext(r)
	context = &BlogContext{
		ContextWithRequest: context,
		Page:               b,
	}

	// Render the template
	var err = tpl.FRender(
		w, context, "blog",
		"blog/page.tmpl",
	)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}

var (
	_ queries.ForDBEditableField   = (*CantSelectField)(nil)
	_ queries.ForUseInQueriesField = (*CantSelectField)(nil)
)

type CantSelectField struct {
	attrs.Field
}

func (f *CantSelectField) ForSelectAll() bool {
	return false
}

func (f *CantSelectField) AllowDBEdit() bool {
	return false
}

func (f *CantSelectField) CanMigrate() bool {
	return false
}

func (n *BlogPage) FieldDefs() attrs.Definitions {
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
			HelpText: "How do you want your post to be remembered?",
		}),
		attrs.NewField(n.Page, "UrlPath", &attrs.FieldConfig{
			Embedded: true,
			ReadOnly: true,
			Label:    "URL Path",
			HelpText: "The URL path for this blog post.",
		}),
		attrs.NewField(n.Page, "Slug", &attrs.FieldConfig{
			Embedded: true,
			Label:    "Slug",
			HelpText: "The slug for this blog post.",
			Blank:    true,
		}),
		editor.NewField(n, "Editor", editor.FieldConfig{
			Label:    trans.S("Editor"),
			HelpText: trans.S("This is a rich text editor. You can add images, videos, and other media to your blog post."),
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
			HelpText: "The date and time this blog post was created.",
		}),
		attrs.NewField(n.Page, "UpdatedAt", &attrs.FieldConfig{
			Embedded: true,
			ReadOnly: true,
			Label:    "Updated At",
			HelpText: "The date and time this blog post was last updated.",
		}),
	)
}
