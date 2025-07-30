package images

import (
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/trans"
)

var _ = contenttypes.Register(&contenttypes.ContentTypeDefinition{
	GetLabel:       trans.S("Image"),
	GetPluralLabel: trans.S("Images"),
	GetDescription: trans.S("An image file"),
	GetInstanceLabel: func(a any) string {
		var image = a.(*Image)
		if image.Title != "" {
			return image.Title
		}
		return ""
	},
	ContentObject: &Image{},
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
