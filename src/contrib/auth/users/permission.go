package users

import (
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
)

type Permission struct {
	models.Model `table:"auth_permissions" json:"-"`
	ID           uint64 `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
}

func (p *Permission) String() string {
	return p.Name
}

func (p *Permission) FieldDefs() attrs.Definitions {
	var fields = []attrs.Field{
		attrs.NewField(p, "ID", &attrs.FieldConfig{
			ReadOnly: true,
			Primary:  true,
		}),
		attrs.NewField(p, "Name", &attrs.FieldConfig{
			Label:     "Permission Name",
			HelpText:  trans.S("Name of the permission. This is the name that will be displayed in the UI."),
			MaxLength: 255,
		}),
		attrs.NewField(p, "Description", &attrs.FieldConfig{
			Blank:     true,
			Label:     "Description",
			HelpText:  trans.S("Description of the permission. This is the description that will be displayed in the UI."),
			MaxLength: 1024,
		}),
	}
	return p.Model.Define(p, fields)
}
