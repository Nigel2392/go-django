// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0

package models

type GroupPermission struct {
	GroupID      int64 `json:"group_id"`
	PermissionID int64 `json:"permission_id"`
}

type UserGroup struct {
	UserID  int64 `json:"user_id"`
	GroupID int64 `json:"group_id"`
}
