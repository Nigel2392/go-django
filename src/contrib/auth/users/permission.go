package users

import (
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

type Permission struct {
	models.Model `table:"auth_permissions" json:"-"`
	ID           uint64 `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
}

func (p *Permission) FieldDefs() attrs.Definitions {
	var fields = []attrs.Field{
		attrs.NewField(p, "ID", &attrs.FieldConfig{
			ReadOnly: true,
			Primary:  true,
		}),
		attrs.NewField(p, "Name", &attrs.FieldConfig{
			Label:    "Permission Name",
			HelpText: "Name of the permission. This is the name that will be displayed in the UI.",
		}),
		attrs.NewField(p, "Description", &attrs.FieldConfig{
			Blank:    true,
			Label:    "Description",
			HelpText: "Description of the permission. This is the description that will be displayed in the UI.",
		}),
	}
	return p.Model.Define(p, fields)
}
