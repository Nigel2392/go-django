package images

import (
	"strings"

	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
)

var _ = contenttypes.Register(&contenttypes.ContentTypeDefinition{
	ContentObject: &Image{},
	GetInstance: func(id any) (interface{}, error) {
		return selectImageInterface(id)
	},
	GetInstances: func(limit, offset uint) ([]interface{}, error) {
		return listImagesInterface(limit, offset)
	},
	GetInstancesByIDs: func(ids []any) ([]interface{}, error) {
		return listImagesInterfaceForPrimaryKeys(ids)
	},
})

func AdminImageModelOptions() admin.ModelOptions {
	return admin.ModelOptions{
		RegisterToAdminMenu: true,
		Model:               &Image{},
		Name:                "Image",
		AddView: admin.FormViewOptions{
			Panels: []admin.Panel{
				admin.FieldPanel("ID"),
				admin.FieldPanel("Title"),
				admin.FieldPanel("Path"),
				admin.FieldPanel("CreatedAt"),
				admin.FieldPanel("FileSize"),
				admin.FieldPanel("FileHash"),
			},
		},
		EditView: admin.FormViewOptions{
			Panels: []admin.Panel{
				admin.FieldPanel("ID"),
				admin.FieldPanel("Title"),
				admin.FieldPanel("Path"),
				admin.FieldPanel("CreatedAt"),
				admin.FieldPanel("FileSize"),
				admin.FieldPanel("FileHash"),
			},
		},
		ListView: admin.ListViewOptions{
			ViewOptions: admin.ViewOptions{
				Fields: []string{
					"ID", "Title", "Path", "CreatedAt", "FileSize", "FileHash",
				},
			},
			PerPage: 10,
		},
	}
}

const (
	listImageAdminQuery = `SELECT id, title, path, created_at, file_size, file_hash
        FROM images
        ORDER BY id ASC
        LIMIT ? OFFSET ?`

	listImageForPrimaryKeysAdminQuery = `SELECT id, title, path, created_at, file_size, file_hash
        FROM images
        WHERE id IN ({%PLACEHOLDER%})`

	selectImageAdminQuery = `SELECT id, title, path, created_at, file_size, file_hash
        FROM images
        WHERE id = ?`
)

func SelectImage(id uint32) (*Image, error) {
	var c, err = selectImageInterface(id)
	if err != nil {
		return nil, err
	}
	return c.(*Image), nil
}

func ListImages(limit, offset int32) ([]*Image, error) {
	var rows, err = app.DB.Query(listImageAdminQuery, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var objects = make([]*Image, 0)
	for rows.Next() {
		var o = &Image{}
		err = rows.Scan(
			&o.ID, &o.Title, &o.Path, &o.CreatedAt, &o.FileSize, &o.FileHash,
		)
		if err != nil {
			return nil, err
		}
		objects = append(objects, o)
	}
	return objects, nil
}

func ListImagesForPrimaryKeys(ids []interface{}) ([]*Image, error) {
	var query = listImageForPrimaryKeysAdminQuery
	var placeholders = make([]string, len(ids))
	var values = make([]interface{}, len(ids)+2)
	for i := range ids {
		placeholders[i] = "?"
		values[i] = ids[i]
	}
	query = strings.ReplaceAll(
		query,
		"{%PLACEHOLDER%}",
		strings.Join(placeholders, ", "),
	)
	var rows, err = app.DB.Query(query, values...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var objects = make([]*Image, 0)
	for rows.Next() {
		var o = &Image{}
		err = rows.Scan(
			&o.ID, &o.Title, &o.Path, &o.CreatedAt, &o.FileSize, &o.FileHash,
		)
		if err != nil {
			return nil, err
		}
		objects = append(objects, o)
	}
	return objects, nil
}

func selectImageInterface(id any) (interface{}, error) {
	var o = &Image{}
	err := app.DB.QueryRow(selectImageAdminQuery, id).Scan(
		&o.ID, &o.Title, &o.Path, &o.CreatedAt, &o.FileSize, &o.FileHash,
	)
	if err != nil {
		return nil, err
	}
	return o, nil
}

func listImagesInterfaceForPrimaryKeys(ids []any) ([]interface{}, error) {
	var rows, err = ListImagesForPrimaryKeys(ids)
	if err != nil {
		return nil, err
	}
	var objects = make([]interface{}, len(rows))
	for i, row := range rows {
		objects[i] = row
	}
	return objects, nil
}

func listImagesInterface(limit, offset uint) ([]interface{}, error) {
	var rows, err = ListImages(int32(limit), int32(offset))
	if err != nil {
		return nil, err
	}
	var objects = make([]interface{}, len(rows))
	for i, row := range rows {
		objects[i] = row
	}
	return objects, nil
}
