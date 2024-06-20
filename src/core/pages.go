package core

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Nigel2392/django/contrib/admin"
	"github.com/Nigel2392/django/contrib/editor"
	"github.com/Nigel2392/django/contrib/pages"
	"github.com/Nigel2392/django/contrib/pages/models"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/contenttypes"
	"github.com/Nigel2392/django/core/logger"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/pkg/errors"
)

const createTable = `CREATE TABLE IF NOT EXISTS blog_pages (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	title TEXT,
	editor TEXT
)`

const insertPage = `INSERT INTO blog_pages (title, editor) VALUES (?, ?)`
const updatePage = `UPDATE blog_pages SET title = ?, editor = ? WHERE id = ?`
const selectPage = `SELECT id, title, editor FROM blog_pages WHERE id = ?`

func toBlockData(richText *editor.EditorJSBlockData) editor.EditorJSData {
	var blocks = make([]editor.BlockData, 0)
	for _, block := range richText.Blocks {
		var data = block.Data()
		blocks = append(blocks, data)
	}
	var d = editor.EditorJSData{
		Time:    richText.Time,
		Version: richText.Version,
		Blocks:  blocks,
	}
	return d
}

func createBlogPage(title string, richText *editor.EditorJSBlockData) (id int64, err error) {
	var editorData = toBlockData(richText)
	data, err := json.Marshal(editorData)
	if err != nil {
		return 0, err
	}

	res, err := globalDB.Exec(insertPage, title, string(data))
	if err != nil {
		return 0, err
	}

	id, err = res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

func updateBlogPage(id int64, title string, richText *editor.EditorJSBlockData) error {
	var editorData = toBlockData(richText)
	var data, err = json.Marshal(editorData)
	if err != nil {
		return err
	}

	_, err = globalDB.Exec(updatePage, title, string(data), id)
	return err
}

func getBlogPage(parentNode models.PageNode, id int64) (*BlogPage, error) {
	var page = &BlogPage{
		PageNode: &parentNode,
	}
	var editorData = ""
	var row = globalDB.QueryRow(selectPage, id)
	var err = row.Err()
	if err != nil {
		return nil, errors.Wrapf(
			err, "Error getting blog page with id %d (%T)", id, id,
		)
	}

	err = row.Scan(&page.PageNode.PageID, &page.Title, &editorData)
	if err != nil {
		return nil, errors.Wrapf(
			err, "Error scanning blog page with id %d (%T)", id, id,
		)
	}

	var richText = editor.EditorJSData{}
	err = json.Unmarshal([]byte(editorData), &richText)
	if err != nil {
		return nil, err
	}

	page.Editor, err = editor.ValueToGo(
		nil, richText,
	)

	return page, err
}

func init() {
	pages.Register(&pages.PageDefinition{
		ContentTypeDefinition: &contenttypes.ContentTypeDefinition{
			GetLabel:       fields.S("Blog Page"),
			GetDescription: fields.S("A blog page with a rich text editor."),
			ContentObject:  &BlogPage{},
		},
		AddPanels: func(r *http.Request, page pages.Page) []admin.Panel {
			return []admin.Panel{
				admin.TitlePanel(
					admin.FieldPanel("Title"),
				),
				admin.FieldPanel("Editor"),
			}
		},
		EditPanels: func(r *http.Request, page pages.Page) []admin.Panel {
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
			return getBlogPage(ref, id)
		},
	})
}

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
		attrs.NewField(n, "Editor", &attrs.FieldConfig{
			Default:  &editor.EditorJSBlockData{},
			Label:    "Rich Text Editor",
			HelpText: "This is a rich text editor. You can add images, videos, and other media to your blog post. ",
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
