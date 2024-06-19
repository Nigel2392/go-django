package core

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Nigel2392/django/contrib/admin"
	"github.com/Nigel2392/django/contrib/editor"
	"github.com/Nigel2392/django/contrib/pages"
	"github.com/Nigel2392/django/contrib/pages/models"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/contenttypes"
	"github.com/Nigel2392/django/forms/fields"
)

var pageMap = make(map[int64]*BlogPage)

func init() {
	pages.Register(&pages.PageDefinition{
		ContentTypeDefinition: &contenttypes.ContentTypeDefinition{
			GetLabel:      fields.S("Blog Page"),
			ContentObject: &BlogPage{},
		},
		Panels: func(r *http.Request, page pages.Page) []admin.Panel {
			return []admin.Panel{
				admin.TitlePanel(
					admin.FieldPanel("Title"),
				),
				admin.MultiPanel(
					admin.FieldPanel("CreatedAt"),
					admin.FieldPanel("UpdatedAt"),
				),
				admin.FieldPanel("Editor"),
			}
		},
		GetForID: func(ctx context.Context, ref models.PageNode, id int64) (pages.Page, error) {
			fmt.Printf("Getting blog page for id: %d %d %v\n", id, ref.PageID, pageMap)
			var page = pageMap[id]
			if page == nil {
				return &BlogPage{
					PageNode: &ref,
					Editor:   &editor.EditorJSBlockData{},
				}, nil
			}
			fmt.Printf("Returning blog page: %#v\n", page)
			return page, nil
		},
	})

	fmt.Println(contenttypes.NewContentType(&BlogPage{}).TypeName())
}

type BlogPage struct {
	*models.PageNode
	Editor *editor.EditorJSBlockData
}

func (b *BlogPage) Save(ctx context.Context) error {
	fmt.Printf("Saving blog page: %#v\n", b)
	var editorData = b.Editor
	for _, block := range editorData.Blocks {
		fmt.Printf("Block: %#v\n", block)
	}
	pageMap[b.PageID] = b
	return nil
}

func (b *BlogPage) ID() int64 {
	return b.PageID
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
		attrs.NewField(n.PageNode, "Title", nil),
		attrs.NewField(n, "Editor", &attrs.FieldConfig{
			Default: &editor.EditorJSBlockData{},
		}),
		attrs.NewField(n.PageNode, "CreatedAt", &attrs.FieldConfig{
			ReadOnly: true,
		}),
		attrs.NewField(n.PageNode, "UpdatedAt", &attrs.FieldConfig{
			ReadOnly: true,
		}),
	)
}
