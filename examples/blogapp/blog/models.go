package blog

import (
	"context"
	"fmt"
	"net/http"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/contrib/editor"
	"github.com/Nigel2392/go-django/src/contrib/pages"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

type BlogPage struct {
	models.Model    `table:"blog_pages"`
	*pages.PageNode `proxy:"-"`
	Editor          *editor.EditorJSBlockData
}

func (b *BlogPage) ID() int64 {
	return b.PageNode.PageID
}

func (b *BlogPage) Reference() *pages.PageNode {
	return b.PageNode
}

func (b *BlogPage) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Serve the blog page here.
	// fmt.Fprintf(w, "Blog page: %s\n", b.Title)
	fmt.Fprintln(w, "<html><head><title>Blog Page</title></head><body>")
	fmt.Fprintf(w, "<h1>%s</h1><div>", b.Title)
	var ctx = context.Background()
	for _, block := range b.Editor.Blocks {
		if err := block.Render(ctx, w); err != nil && editor.RENDER_ERRORS {
			fmt.Fprintf(w, "Error (%s): %s", block.Type(), err)
		}
	}
	fmt.Fprintln(w, "</div></body></html>")

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

func (n *BlogPage) FieldDefs() attrs.Definitions {
	if n.PageNode == nil {
		n.PageNode = &pages.PageNode{}
	}
	return n.Model.Define(n,
		attrs.NewField(n.PageNode, "PageID", &attrs.FieldConfig{
			Primary:  true,
			ReadOnly: true,
			Column:   "id",
		}),
		&CantSelectField{attrs.NewField(n.PageNode, "Title", &attrs.FieldConfig{
			Label:    "Title",
			HelpText: "How do you want your post to be remembered?",
			Column:   "",
		})},
		&CantSelectField{attrs.NewField(n.PageNode, "UrlPath", &attrs.FieldConfig{
			ReadOnly: true,
			Label:    "URL Path",
			HelpText: "The URL path for this blog post.",
			Column:   "",
		})},
		&CantSelectField{attrs.NewField(n.PageNode, "Slug", &attrs.FieldConfig{
			Label:    "Slug",
			HelpText: "The slug for this blog post.",
			Blank:    true,
			Column:   "",
		})},
		editor.NewField(n, "Editor", editor.FieldConfig{
			Label:    "Editor",
			HelpText: "This is a rich text editor. You can add images, videos, and other media to your blog post.",
			//Features: []string{
			//	"paragraph",
			//	"text-align",
			//	"list",
			//},
		}),
		//attrs.NewField(n, "Editor", &attrs.FieldConfig{
		//	Default:  &editor.EditorJSBlockData{},
		//	Label:    "Rich Text Editor",
		//	HelpText: "This is a rich text editor. You can add images, videos, and other media to your blog post.",
		//	FormField: func(opts ...func(fields.Field)) fields.Field {
		//		var editor = editor.EditorJSField(
		//			[]string{
		//				// "paragraph",
		//				// "text-align",
		//				// "list",
		//			},
		//			opts...,
		//		)
		//		return editor
		//	},
		//}),
		&CantSelectField{attrs.NewField(n.PageNode, "CreatedAt", &attrs.FieldConfig{
			ReadOnly: true,
			Label:    "Created At",
			HelpText: "The date and time this blog post was created.",
			Column:   "",
		})},
		&CantSelectField{attrs.NewField(n.PageNode, "UpdatedAt", &attrs.FieldConfig{
			ReadOnly: true,
			Label:    "Updated At",
			HelpText: "The date and time this blog post was last updated.",
			Column:   "",
		})},
	)
}
