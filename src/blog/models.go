package blog

import (
	"context"

	"github.com/Nigel2392/django/contrib/editor"
	"github.com/Nigel2392/django/contrib/pages/models"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/logger"
	"github.com/Nigel2392/django/forms/fields"
)

type BlogPage struct {
	*models.PageNode
	Editor *editor.EditorJSBlockData
}

func (b *BlogPage) Save(ctx context.Context) error {
	var err error
	if b.ID() == 0 {
		var id int64
		id, err = createBlogPage(b.Title, b.Editor)
		b.PageID = id
	} else {
		err = updateBlogPage(b.PageNode.PageID, b.Title, b.Editor)
	}
	if err != nil {
		logger.Errorf("Error saving blog page: %v\n", err)
	}
	return err
}

func (b *BlogPage) ID() int64 {
	return b.PageNode.PageID
}

func (b *BlogPage) Reference() *models.PageNode {
	return b.PageNode
}

func (n *BlogPage) FieldDefs() attrs.Definitions {
	if n.PageNode == nil {
		n.PageNode = &models.PageNode{}
	}
	return attrs.Define(n,
		attrs.NewField(n.PageNode, "PageID", &attrs.FieldConfig{
			Primary:  true,
			ReadOnly: true,
		}),
		attrs.NewField(n.PageNode, "Title", &attrs.FieldConfig{
			Label:    "Title",
			HelpText: "How do you want your post to be remembered?",
		}),
		attrs.NewField(n.PageNode, "UrlPath", &attrs.FieldConfig{
			ReadOnly: true,
			Label:    "URL Path",
			HelpText: "The URL path for this blog post.",
		}),
		attrs.NewField(n.PageNode, "Slug", &attrs.FieldConfig{
			Label:    "Slug",
			HelpText: "The slug for this blog post.",
			Blank:    true,
		}),
		attrs.NewField(n, "Editor", &attrs.FieldConfig{
			Default:  &editor.EditorJSBlockData{},
			Label:    "Rich Text Editor",
			HelpText: "This is a rich text editor. You can add images, videos, and other media to your blog post.",
			FormField: func(opts ...func(fields.Field)) fields.Field {
				var editor = editor.EditorJSField(
					[]string{
						// "paragraph",
						// "text-align",
						// "list",
					},
					opts...,
				)
				return editor
			},
		}),
		attrs.NewField(n.PageNode, "CreatedAt", &attrs.FieldConfig{
			ReadOnly: true,
			Label:    "Created At",
			HelpText: "The date and time this blog post was created.",
		}),
		attrs.NewField(n.PageNode, "UpdatedAt", &attrs.FieldConfig{
			ReadOnly: true,
			Label:    "Updated At",
			HelpText: "The date and time this blog post was last updated.",
		}),
	)
}
