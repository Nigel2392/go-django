package permissions_models

type Group struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type GroupPermission struct {
	ID           uint64 `json:"id"`
	GroupID      uint64 `json:"group_id"`
	PermissionID uint64 `json:"permission_id"`
}

type Permission struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UserGroup struct {
	ID      uint64 `json:"id"`
	UserID  uint64 `json:"user_id"`
	GroupID uint64 `json:"group_id"`
}

type UserGroupsRow struct {
	Group      Group      `json:"group"`
	Permission Permission `json:"permission"`
}
