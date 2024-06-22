package blog

import (
	"encoding/json"

	"github.com/Nigel2392/django/contrib/editor"
	"github.com/Nigel2392/django/contrib/pages/models"
	"github.com/pkg/errors"
)

const (
	// createTable = `CREATE TABLE IF NOT EXISTS blog_pages (
	// id INTEGER PRIMARY KEY AUTOINCREMENT,
	// title TEXT,
	// editor TEXT
	// )`
	insertPage = `INSERT INTO blog_pages (title, editor) VALUES (?, ?)`
	updatePage = `UPDATE blog_pages SET title = ?, editor = ? WHERE id = ?`
	selectPage = `SELECT id, title, editor FROM blog_pages WHERE id = ?`
)

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

	res, err := blog.DB.Exec(insertPage, title, string(data))
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

	_, err = blog.DB.Exec(updatePage, title, string(data), id)
	return err
}

func getBlogPage(parentNode models.PageNode, id int64) (*BlogPage, error) {
	var page = &BlogPage{
		PageNode: &parentNode,
	}
	if blog.DB == nil {
		return nil, errors.New("blog.DB is nil")
	}
	var editorData = ""
	var row = blog.DB.QueryRow(selectPage, id)
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
