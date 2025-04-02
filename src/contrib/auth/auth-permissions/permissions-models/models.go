package permissions_models

import "github.com/Nigel2392/go-django/src/core/attrs"

type Group struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (g *Group) FieldDefs() attrs.Definitions {
	var fields = []attrs.Field{
		attrs.NewField(g, "ID", &attrs.FieldConfig{
			ReadOnly: true,
			Primary:  true,
		}),
		attrs.NewField(g, "Name", &attrs.FieldConfig{
			Label:    "Group Name",
			HelpText: "Name of the group. This is the name that will be displayed in the UI.",
		}),
		attrs.NewField(g, "Description", &attrs.FieldConfig{
			Blank:    true,
			Label:    "Description",
			HelpText: "Description of the group. This is the description that will be displayed in the UI.",
		}),
	}
	return attrs.Define(g, fields...)
}

type Permission struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
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
	return attrs.Define(p, fields...)
}
