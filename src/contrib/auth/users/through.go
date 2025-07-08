package users

import (
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

type UserGroup struct {
	UserID  drivers.Uint `json:"user_id" attrs:"primary;readonly"`
	GroupID drivers.Uint `json:"group_id" attrs:"primary;readonly"`
}

func (up *UserGroup) FieldDefs() attrs.Definitions {
	return attrs.Define(up,
		attrs.Unbound("UserID", &attrs.FieldConfig{
			ReadOnly: true,
			Column:   "user_id",
		}),
		attrs.Unbound("GroupID", &attrs.FieldConfig{
			ReadOnly: true,
			Column:   "group_id",
		}),
	).WithTableName("auth_user_groups")
}

func (up *UserGroup) UniqueTogether() [][]string {
	return [][]string{
		{"UserID", "GroupID"},
	}
}

type UserPermission struct {
	UserID       drivers.Uint `json:"user_id" attrs:"primary;readonly"`
	PermissionID drivers.Uint `json:"permission_id" attrs:"primary;readonly"`
}

func (up *UserPermission) FieldDefs() attrs.Definitions {
	return attrs.Define(up,
		attrs.Unbound("UserID", &attrs.FieldConfig{
			ReadOnly: true,
			Column:   "user_id",
		}),
		attrs.Unbound("PermissionID", &attrs.FieldConfig{
			ReadOnly: true,
			Column:   "permission_id",
		}),
	).WithTableName("auth_user_permissions")
}

func (up *UserPermission) UniqueTogether() [][]string {
	return [][]string{
		{"UserID", "PermissionID"},
	}
}

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
