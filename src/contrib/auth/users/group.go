package users

import (
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/fields"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
)

type Group struct {
	models.Model `table:"auth_groups" json:"-"`
	ID           uint64                                         `json:"id"`
	Name         string                                         `json:"name"`
	Description  string                                         `json:"description"`
	Permissions  *queries.RelM2M[*Permission, *GroupPermission] `json:"-"`
}

func (g *Group) String() string {
	return g.Name
}

func (g *Group) FieldDefs() attrs.Definitions {
	var fields = []attrs.Field{
		attrs.NewField(g, "ID", &attrs.FieldConfig{
			ReadOnly: true,
			Primary:  true,
		}),
		attrs.NewField(g, "Name", &attrs.FieldConfig{
			Label:     "Group Name",
			HelpText:  trans.S("Name of the group. This is the name that will be displayed in the UI."),
			MaxLength: 255,
		}),
		attrs.NewField(g, "Description", &attrs.FieldConfig{
			Blank:     true,
			Label:     "Description",
			HelpText:  trans.S("Description of the group. This is the description that will be displayed in the UI."),
			MaxLength: 1024,
		}),
		fields.NewManyToManyField[*queries.RelM2M[*Permission, *GroupPermission]](
			g, "Permissions", &fields.FieldConfig{
				ScanTo:      &g.Permissions,
				ReverseName: "GroupPermissions",
				ColumnName:  "id",
				Rel: attrs.Relate(
					&Permission{}, "",
					&attrs.ThroughModel{
						This:   &GroupPermission{},
						Source: "GroupID",
						Target: "PermissionID",
					},
				),
			},
		),
	}
	return g.Model.Define(g, fields)
}
