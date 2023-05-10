package auth

// Thank you SQLC

type GroupPermission struct {
	GroupID      int64 `json:"group_id" gorm:"primary_key"`
	PermissionID int64 `json:"permission_id" gorm:"primary_key"`
}

type UserGroup struct {
	UserID  int64 `json:"user_id" gorm:"primary_key"`
	GroupID int64 `json:"group_id" gorm:"primary_key"`
}
