package auth

import (
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/fields"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

type GroupPermission struct {
	GroupID      drivers.Uint `json:"group_id" attrs:"primary;readonly"`
	PermissionID drivers.Uint `json:"permission_id" attrs:"primary;readonly"`
}

func (gp *GroupPermission) FieldDefs() attrs.Definitions {
	return attrs.Define(gp,
		attrs.Unbound("GroupID", &attrs.FieldConfig{
			ReadOnly: true,
			Column:   "group_id",
		}),
		attrs.Unbound("PermissionID", &attrs.FieldConfig{
			ReadOnly: true,
			Column:   "permission_id",
		}),
	).WithTableName("auth_group_permissions")
}

func (gp *GroupPermission) UniqueTogether() [][]string {
	return [][]string{
		{"GroupID", "PermissionID"},
	}
}

type Group struct {
	models.Model `table:"auth_groups" json:"-"`
	ID           uint64                                         `json:"id"`
	Name         string                                         `json:"name"`
	Description  string                                         `json:"description"`
	Permissions  *queries.RelM2M[*Permission, *GroupPermission] `json:"-"`
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
