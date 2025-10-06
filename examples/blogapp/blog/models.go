package blog

import (
	"fmt"
	"net/http"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/contrib/admin/chooser"
	"github.com/Nigel2392/go-django/src/contrib/auth/users"
	"github.com/Nigel2392/go-django/src/contrib/blocks"
	"github.com/Nigel2392/go-django/src/contrib/documents"
	"github.com/Nigel2392/go-django/src/contrib/editor"
	"github.com/Nigel2392/go-django/src/contrib/images"
	"github.com/Nigel2392/go-django/src/contrib/pages"
	"github.com/Nigel2392/go-django/src/contrib/search"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

var (
	_ models.SaveableObject   = (*BlogPage)(nil)
	_ models.DeleteableObject = (*BlogPage)(nil)

	_ attrs.Embedded           = (*OrderableMixin[*BlogImage])(nil)
	_ attrs.FieldUnpackerMixin = (*OrderableMixin[*BlogImage])(nil)
)

type BlogContext struct {
	ctx.ContextWithRequest
	Page pages.Page
}

type OrderableMixin[T attrs.Definer] struct {
	Reference T
	Ordering  int
}

func (m *OrderableMixin[T]) IsModelMixin() {}

func (m *OrderableMixin[T]) BindToEmbedder(embedder attrs.Definer) error {
	m.Reference = embedder.(T)
	return nil
}

func (m *OrderableMixin[T]) FieldDefs() attrs.Definitions {
	return nil
}

func (m *OrderableMixin[T]) ObjectFields(object attrs.Definer, base_fields *attrs.FieldsMap) error {
	var orderingField, ok = base_fields.Get("Ordering")
	if !ok {
		orderingField = attrs.NewField(m, "Ordering", &attrs.FieldConfig{
			Default: 0,
			Column:  "ordering",
		})
		base_fields.Set("Ordering", orderingField)
	}
	return nil
}

type BlogImage struct {
	models.Model `table:"blog_images"`
	OrderableMixin[*BlogImage]
	ID       int64
	Image    *images.Image
	BlogPage *BlogPage
	Content  blocks.ListBlockData
}

func (b *BlogImage) UniqueTogether() [][]string {
	return [][]string{
		{"BlogPage", "Image"},
	}
}

func (b *BlogImage) GetContentBlock() *blocks.ListBlock {
	var sb = blocks.NewStructBlock()
	sb.AddField("Caption", blocks.CharBlock(
		blocks.WithLabel[*blocks.FieldBlock](
			trans.S("Text"),
		),
		blocks.WithHelpText[*blocks.FieldBlock](
			trans.S("Some text for this image."),
		),
		blocks.WithDefault[*blocks.FieldBlock]("Default caption text"),
	))
	sb.AddField("Attribution", blocks.CharBlock(
		blocks.WithLabel[*blocks.FieldBlock](
			trans.S("Attribution"),
		),
		blocks.WithHelpText[*blocks.FieldBlock](
			trans.S("Some text for the attribution."),
		),
		blocks.WithDefault[*blocks.FieldBlock]("Default attribution text"),
	))

	var block = blocks.NewListBlock(sb)

	block.Min = 2
	block.Max = 3

	return block
}

func (b *BlogImage) FieldDefs() attrs.Definitions {
	return b.Model.Define(b,
		attrs.NewField(b, "ID", &attrs.FieldConfig{
			Primary:  true,
			ReadOnly: true,
			Column:   "id",
		}),
		attrs.NewField(b, "BlogPage", &attrs.FieldConfig{
			RelForeignKey: attrs.Relate(
				&BlogPage{}, "", nil,
			),
			Blank: true,
			FormField: func(opts ...func(fields.Field)) fields.Field {
				return fields.CharField(
					fields.Hide(true),
				)
			},
		}),
		attrs.NewField(b, "Image", &attrs.FieldConfig{
			RelForeignKey: attrs.Relate(
				&images.Image{}, "", nil,
			),
			FormWidget: func(fc attrs.FieldConfig) widgets.Widget {
				return chooser.NewChooserWidget(
					fc.RelForeignKey.Model(), fc.WidgetAttrs,
				)
			},
			Label:    trans.S("Image"),
			HelpText: trans.S("The image for this blog post."),
		}),
		attrs.NewField(b, "Content", &attrs.FieldConfig{
			Null:  true,
			Blank: true,
		}),
	)
}

type BlogPage struct {
	models.Model `table:"blog_pages"`
	Page         *pages.PageNode `proxy:"true"`
	Image        *mediafiles.SimpleStoredObject
	Editor       *editor.EditorJSBlockData
	Thumbnail    *images.Image
	Document     *documents.Document
	User         users.User
}

func (b *BlogPage) ID() int64 {
	return b.Page.PageID
}

func (b *BlogPage) Reference() *pages.PageNode {
	return b.Page
}

func (b *BlogPage) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	fmt.Printf("Serving blog page %q from %s\n", b.Page.Title, r.URL.Path)

	// Create a new RequestContext
	// Add the page object to the context
	var context ctx.ContextWithRequest = ctx.RequestContext(r)
	context = &BlogContext{
		ContextWithRequest: context,
		Page:               b,
	}

	context.Set("blog_page", b)

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

func (n *BlogPage) SearchableFields() []search.SearchField {
	return []search.SearchField{
		search.NewSearchField(4, "Editor", expr.LOOKUP_ICONTANS),
	}
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
			HelpText: trans.S("How do you want your post to be remembered?"),
		}),
		attrs.NewField(n.Page, "UrlPath", &attrs.FieldConfig{
			Embedded: true,
			ReadOnly: true,
			Label:    "URL Path",
			HelpText: trans.S("The URL path for this blog post."),
		}),
		attrs.NewField(n.Page, "Slug", &attrs.FieldConfig{
			Embedded: true,
			Label:    "Slug",
			HelpText: trans.S("The slug for this blog post."),
			Blank:    true,
		}),
		attrs.NewField(n, "Image", &attrs.FieldConfig{
			Label:    "Image",
			HelpText: trans.S("The image for this blog post."),
			Null:     true,
			Blank:    true,
		}),
		attrs.NewField(n, "Thumbnail", &attrs.FieldConfig{
			Null:     true,
			Label:    trans.S("Thumbnail"),
			HelpText: trans.S("The thumbnail for this blog post."),
			RelForeignKey: attrs.Relate(
				&images.Image{}, "", nil,
			),
			FormWidget: func(fc attrs.FieldConfig) widgets.Widget {
				return chooser.NewChooserWidget(
					fc.RelForeignKey.Model(), fc.WidgetAttrs,
				)
			},
		}),
		attrs.NewField(n, "Document", &attrs.FieldConfig{
			Null:     true,
			Label:    trans.S("Document"),
			HelpText: trans.S("The document for this blog post."),
			RelForeignKey: attrs.Relate(
				&documents.Document{}, "", nil,
			),
			FormWidget: func(fc attrs.FieldConfig) widgets.Widget {
				return chooser.NewChooserWidget(
					fc.RelForeignKey.Model(), fc.WidgetAttrs,
				)
			},
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
		attrs.NewField(n, "User", &attrs.FieldConfig{
			Label:    trans.S("User"),
			HelpText: trans.S("The user who created this blog post."),
			Null:     true,
			RelForeignKey: attrs.RelatedDeferred(
				attrs.RelManyToOne,
				users.MODEL_KEY,
				"", nil,
			),
			FormWidget: func(fc attrs.FieldConfig) widgets.Widget {
				return chooser.NewChooserWidget(
					fc.RelForeignKey.Model(), fc.WidgetAttrs,
				)
			},
		}),
		attrs.NewField(n.Page, "CreatedAt", &attrs.FieldConfig{
			Embedded: true,
			ReadOnly: true,
			Label:    "Created At",
			HelpText: trans.S("The date and time this blog post was created."),
		}),
		attrs.NewField(n.Page, "UpdatedAt", &attrs.FieldConfig{
			Embedded: true,
			ReadOnly: true,
			Label:    "Updated At",
			HelpText: trans.S("The date and time this blog post was last updated."),
		}),
	)
}
