package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/Nigel2392/go-datastructures/linkedlist"
)

// Thank you SQLC

type AuthQuerier interface {
	// Groups
	CountGroups(ctx context.Context) (int64, error)
	GetAllGroups(ctx context.Context) (*linkedlist.Doubly[Group], error)
	CreateGroup(ctx context.Context, arg *Group) error
	UpdateGroup(ctx context.Context, arg *Group) error
	DeleteGroup(ctx context.Context, id int64) error
	GetGroupByID(ctx context.Context, id int64) (*Group, error)
	GetGroupsWithPagination(ctx context.Context, arg PaginationParams) (*linkedlist.Doubly[Group], error)
	GroupsNotInUser(ctx context.Context, userID int64) (*linkedlist.Doubly[Group], error)
	// Permissions
	CountPermissions(ctx context.Context) (int64, error)
	GetAllPermissions(ctx context.Context) (*linkedlist.Doubly[Permission], error)
	CreatePermission(ctx context.Context, arg *Permission) error
	UpdatePermission(ctx context.Context, arg *Permission) error
	DeletePermission(ctx context.Context, id int64) error
	GetPermissionByID(ctx context.Context, id int64) (*Permission, error)
	GetPermissionByName(ctx context.Context, name string) (Permission, error)
	GetPermissionsByUserID(ctx context.Context, userID int64) (*linkedlist.Doubly[Permission], error)
	GetPermissionsByUserIDAndPermissionNames(ctx context.Context, arg UserPermissionParams) (*linkedlist.Doubly[Permission], error)
	GetPermissionsWithPagination(ctx context.Context, arg PaginationParams) (*linkedlist.Doubly[Permission], error)
	PermissionsNotInGroup(ctx context.Context, groupID int64) (*linkedlist.Doubly[Permission], error)
	PermissionsNotInUser(ctx context.Context, userID int64) (*linkedlist.Doubly[Permission], error)
	GetPermissionsByGroupID(ctx context.Context, groupID int64) (*linkedlist.Doubly[Permission], error)
	// Users
	CountUsers(ctx context.Context) (int64, error)
	GetAllUsers(ctx context.Context) (*linkedlist.Doubly[User], error)
	CreateUser(ctx context.Context, arg *User) error
	UpdateUser(ctx context.Context, arg *User) error
	DeleteUser(ctx context.Context, id int64) error
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id int64) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetUsersByPermissionID(ctx context.Context, permissionID int64) (*linkedlist.Doubly[User], error)
	GetUsersWithPagination(ctx context.Context, arg PaginationParams) (*linkedlist.Doubly[User], error)
	GetGroupsByUserID(ctx context.Context, userID int64) (*linkedlist.Doubly[Group], error)
	// m2m
	CheckUserHasPermissions(ctx context.Context, arg UserPermissionParams) (bool, error)
	AddPermissionToGroup(ctx context.Context, groupID, permissionID int64) error
	AddUserToGroup(ctx context.Context, userID, groupID int64) error
	OverrideUserGroups(ctx context.Context, userID int64, groupIDs []int64) error
	OverrideGroupPermissions(ctx context.Context, groupID int64, permissionIDs []int64) error
	ListPermissionsInGroup(ctx context.Context, groupID int64) (*linkedlist.Doubly[Permission], error)
	ListUsersInGroup(ctx context.Context, groupID int64) (*linkedlist.Doubly[User], error)
	RemovePermissionFromGroup(ctx context.Context, groupID, permissionID int64) error
	RemoveUserFromGroup(ctx context.Context, userID, groupID int64) error
}

const deleteUserGroups = `-- name: DeleteUserGroups :exec
DELETE FROM user_groups
WHERE user_id = ?
`

func (q *Queries) DeleteUserGroups(ctx context.Context, userID int64) error {
	_, err := q.db.ExecContext(ctx, deleteUserGroups, userID)
	return err
}

const addUserToGroups = `-- name: AddUserToGroups :exec
INSERT INTO user_groups (user_id, group_id)
SELECT ? AS user_id, id AS group_id
FROM ` + "`" + `groups` + "`" + `
WHERE id IN (/*SLICE:group_ids*/?)
`

func (q *Queries) AddUserToGroups(ctx context.Context, userID int64, groupIds []int64) error {
	sql := addUserToGroups
	var queryParams []interface{}
	queryParams = append(queryParams, userID)
	if len(groupIds) > 0 {
		for _, v := range groupIds {
			queryParams = append(queryParams, v)
		}
		sql = strings.Replace(sql, "/*SLICE:group_ids*/?", strings.Repeat(",?", len(groupIds))[1:], 1)
	} else {
		sql = strings.Replace(sql, "/*SLICE:group_ids*/?", "NULL", 1)
	}
	_, err := q.db.ExecContext(ctx, sql, queryParams...)
	return err
}

func (q *Queries) OverrideUserGroups(ctx context.Context, userID int64, groupIDs []int64) error {
	q.DeleteUserGroups(ctx, userID)
	return q.AddUserToGroups(ctx, userID, groupIDs)
}

const deleteGroupPermissions = `-- name: DeleteGroupPermissions :exec
DELETE FROM group_permissions
WHERE group_id = ?
`

func (q *Queries) DeleteGroupPermissions(ctx context.Context, groupID int64) error {
	_, err := q.db.ExecContext(ctx, deleteGroupPermissions, groupID)
	return err
}

const addPermissionsToGroup = `-- name: AddPermissionsToGroup :exec
INSERT INTO group_permissions (group_id, permission_id)
SELECT ? AS group_id, id AS permission_id
FROM permissions
WHERE id IN (/*SLICE:permission_ids*/?)
`

func (q *Queries) AddPermissionsToGroup(ctx context.Context, groupID int64, permissionIds []int64) error {
	sql := addPermissionsToGroup
	var queryParams []interface{}
	queryParams = append(queryParams, groupID)
	if len(permissionIds) > 0 {
		for _, v := range permissionIds {
			queryParams = append(queryParams, v)
		}
		sql = strings.Replace(sql, "/*SLICE:permission_ids*/?", strings.Repeat(",?", len(permissionIds))[1:], 1)
	} else {
		sql = strings.Replace(sql, "/*SLICE:permission_ids*/?", "NULL", 1)
	}
	_, err := q.db.ExecContext(ctx, sql, queryParams...)
	return err
}

func (q *Queries) OverrideGroupPermissions(ctx context.Context, groupID int64, permissionIDs []int64) error {
	q.DeleteGroupPermissions(ctx, groupID)
	return q.AddPermissionsToGroup(ctx, groupID, permissionIDs)
}

var _ AuthQuerier = (*Queries)(nil)

type PaginationParams struct {
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

type UserPermissionParams struct {
	UserID          int64    `json:"user_id"`
	Permissionnames []string `json:"permissionnames"`
}

type userRow struct {
	ID              int64           `json:"id"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	Email           string          `json:"email"`
	Username        string          `json:"username"`
	Password        string          `json:"password"`
	FirstName       string          `json:"first_name"`
	LastName        string          `json:"last_name"`
	IsAdministrator bool            `json:"is_administrator"`
	IsActive        bool            `json:"is_active"`
	Groups          json.RawMessage `json:"groups"`
	Permissions     json.RawMessage `json:"permissions"`
}

const getUserByEmail = `-- name: GetUserByEmail :one
WITH user_groups AS (
  SELECT ug.user_id, g.id as group_id, g.name as group_name, g.description as group_description
  FROM user_groups ug
  JOIN ` + "`" + `groups` + "`" + ` g ON ug.group_id = g.id
),
user_permissions AS (
  SELECT ug.user_id, p.id as permission_id, p.name as permission_name, p.description as permission_description
  FROM user_groups ug
  JOIN group_permissions gp ON ug.group_id = gp.group_id
  JOIN permissions p ON gp.permission_id = p.id
)
SELECT u.id, u.created_at, u.updated_at, u.email, u.username, u.password, u.first_name, u.last_name, u.is_administrator, u.is_active,
       JSON_ARRAYAGG(JSON_OBJECT('id', ug.group_id, 'name', ug.group_name, 'description', ug.group_description)) as ` + "`" + `groups` + "`" + `,
       JSON_ARRAYAGG(JSON_OBJECT('id', up.permission_id, 'name', up.permission_name, 'description', up.permission_description)) as permissions
FROM users u
LEFT JOIN user_groups ug ON u.id = ug.user_id
LEFT JOIN user_permissions up ON u.id = up.user_id
WHERE LOWER(u.email) = LOWER(?) 
GROUP BY u.id
`

const getUserByID = `-- name: GetUserByID :one
WITH user_groups AS (
  SELECT ug.user_id, g.id as group_id, g.name as group_name, g.description as group_description
  FROM user_groups ug
  JOIN ` + "`" + `groups` + "`" + ` g ON ug.group_id = g.id
),
user_permissions AS (
  SELECT ug.user_id, p.id as permission_id, p.name as permission_name, p.description as permission_description
  FROM user_groups ug
  JOIN group_permissions gp ON ug.group_id = gp.group_id
  JOIN permissions p ON gp.permission_id = p.id
)
SELECT u.id, u.created_at, u.updated_at, u.email, u.username, u.password, u.first_name, u.last_name, u.is_administrator, u.is_active,
       JSON_ARRAYAGG(JSON_OBJECT('id', ug.group_id, 'name', ug.group_name, 'description', ug.group_description)) as ` + "`" + `groups` + "`" + `,
       JSON_ARRAYAGG(JSON_OBJECT('id', up.permission_id, 'name', up.permission_name, 'description', up.permission_description)) as permissions
FROM users u
LEFT JOIN user_groups ug ON u.id = ug.user_id
LEFT JOIN user_permissions up ON u.id = up.user_id
WHERE u.id = ? 
GROUP BY u.id
`

const getUserByUsername = `-- name: GetUserByUsername :one
WITH user_groups AS (
  SELECT ug.user_id, g.id as group_id, g.name as group_name, g.description as group_description
  FROM user_groups ug
  JOIN ` + "`" + `groups` + "`" + ` g ON ug.group_id = g.id
),
user_permissions AS (
  SELECT ug.user_id, p.id as permission_id, p.name as permission_name, p.description as permission_description
  FROM user_groups ug
  JOIN group_permissions gp ON ug.group_id = gp.group_id
  JOIN permissions p ON gp.permission_id = p.id
)
SELECT u.id, u.created_at, u.updated_at, u.email, u.username, u.password, u.first_name, u.last_name, u.is_administrator, u.is_active,
       JSON_ARRAYAGG(JSON_OBJECT('id', ug.group_id, 'name', ug.group_name, 'description', ug.group_description)) as ` + "`" + `groups` + "`" + `,
       JSON_ARRAYAGG(JSON_OBJECT('id', up.permission_id, 'name', up.permission_name, 'description', up.permission_description)) as permissions
FROM users u
LEFT JOIN user_groups ug ON u.id = ug.user_id
LEFT JOIN user_permissions up ON u.id = up.user_id
WHERE LOWER(u.username) = LOWER(?) 
GROUP BY u.id
`

func (q *Queries) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	row := q.db.QueryRowContext(ctx, getUserByEmail, email)
	return makeUser(row)
}

func (q *Queries) GetUserByID(ctx context.Context, id int64) (*User, error) {
	row := q.db.QueryRowContext(ctx, getUserByID, id)
	return makeUser(row)
}

func (q *Queries) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	row := q.db.QueryRowContext(ctx, getUserByUsername, username)
	return makeUser(row)
}

func makeUser(row *sql.Row) (*User, error) {
	var i userRow
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Email,
		&i.Username,
		&i.Password,
		&i.FirstName,
		&i.LastName,
		&i.IsAdministrator,
		&i.IsActive,
		&i.Groups,
		&i.Permissions,
	)
	if err != nil {
		return nil, err
	}
	var groups []Group
	var permissions []Permission
	if len(i.Groups) > 0 {
		if err := json.Unmarshal(i.Groups, &groups); err != nil {
			return nil, err
		}
	}
	if len(i.Permissions) > 0 {
		if err := json.Unmarshal(i.Permissions, &permissions); err != nil {
			return nil, err
		}
	}
	return &User{
		ID:              i.ID,
		Email:           EmailField(i.Email),
		Username:        i.Username,
		Password:        PasswordField(i.Password),
		FirstName:       i.FirstName,
		LastName:        i.LastName,
		IsAdministrator: i.IsAdministrator,
		IsActive:        i.IsActive,
		Groups:          groups,
		Permissions:     permissions,
	}, nil
}

var countGroups = `-- name: CountGroups :one
SELECT COUNT(*) FROM ` + "`" + `groups` + "`"

func (q *Queries) CountGroups(ctx context.Context) (int64, error) {
	var count int64
	err := q.db.QueryRowContext(ctx, countGroups).Scan(&count)
	return count, err
}

var countPermissions = `-- name: CountPermissions :one
SELECT COUNT(*) FROM permissions
`

func (q *Queries) CountPermissions(ctx context.Context) (int64, error) {
	var count int64
	err := q.db.QueryRowContext(ctx, countPermissions).Scan(&count)
	return count, err
}

var countUsers = `-- name: CountUsers :one
SELECT COUNT(*) FROM users
`

func (q *Queries) CountUsers(ctx context.Context) (int64, error) {
	var count int64
	err := q.db.QueryRowContext(ctx, countUsers).Scan(&count)
	return count, err
}

const addPermissionToGroup = `-- name: AddPermissionToGroup :exec
INSERT INTO group_permissions (group_id, permission_id)
VALUES (?, ?)
`

func (q *Queries) AddPermissionToGroup(ctx context.Context, groupID, permissionID int64) error {
	_, err := q.db.ExecContext(ctx, addPermissionToGroup, groupID, permissionID)
	return err
}

const addUserToGroup = `-- name: AddUserToGroup :exec
INSERT INTO user_groups (user_id, group_id)
VALUES (?, ?)
`

func (q *Queries) AddUserToGroup(ctx context.Context, userID, groupID int64) error {
	_, err := q.db.ExecContext(ctx, addUserToGroup, userID, groupID)
	return err
}

const checkUserHasPermissions = `-- name: CheckUserHasPermissions :one
SELECT EXISTS (
    SELECT 1
    FROM users u
    JOIN user_groups ug ON u.id = ug.user_id
    JOIN ` + "`" + `groups` + "`" + ` g ON ug.group_id = g.id
    JOIN group_permissions gp ON g.id = gp.group_id
    JOIN permissions p ON gp.permission_id = p.id
    WHERE u.id = ? AND p.name IN (/*SLICE:permissionnames*/?)
) as has_permissions
`

func (q *Queries) CheckUserHasPermissions(ctx context.Context, arg UserPermissionParams) (bool, error) {
	sql := checkUserHasPermissions
	var queryParams []interface{}
	queryParams = append(queryParams, arg.UserID)
	if len(arg.Permissionnames) > 0 {
		for _, v := range arg.Permissionnames {
			queryParams = append(queryParams, v)
		}
		sql = strings.Replace(sql, "/*SLICE:permissionnames*/?", strings.Repeat(",?", len(arg.Permissionnames))[1:], 1)
	} else {
		sql = strings.Replace(sql, "/*SLICE:permissionnames*/?", "NULL", 1)
	}
	row := q.db.QueryRowContext(ctx, sql, queryParams...)
	var has_permissions bool
	err := row.Scan(&has_permissions)
	return has_permissions, err
}

const createGroup = `-- name: CreateGroup :exec
INSERT INTO ` + "`" + `groups` + "`" + ` (name, description)
VALUES (?, ?)
`

func (q *Queries) CreateGroup(ctx context.Context, arg *Group) error {
	SIGNAL_BEFORE_GROUP_CREATE.Send(arg)
	result, err := q.db.ExecContext(ctx, createGroup, arg.Name, arg.Description)
	if err == nil {
		var id, _ = result.LastInsertId()
		arg.ID = id
		SIGNAL_AFTER_GROUP_CREATE.Send(arg)
	}
	return err
}

const createPermission = `-- name: CreatePermission :exec
INSERT INTO permissions (name, description)
VALUES (?, ?)
`

func (q *Queries) CreatePermission(ctx context.Context, arg *Permission) error {
	SIGNAL_BEFORE_PERMISSION_CREATE.Send(arg)
	result, err := q.db.ExecContext(ctx, createPermission, arg.Name, arg.Description)
	if err == nil {
		var id, _ = result.LastInsertId()
		arg.ID = id
		SIGNAL_AFTER_PERMISSION_CREATE.Send(arg)
	}
	return err
}

const createUser = `-- name: CreateUser :exec
INSERT INTO users (email, username, password, first_name, last_name, is_administrator, is_active)
VALUES (?, ?, ?, ?, ?, ?, ?)
`

func (q *Queries) CreateUser(ctx context.Context, arg *User) error {
	SIGNAL_BEFORE_USER_CREATE.Send(arg)
	result, err := q.db.ExecContext(ctx, createUser,
		arg.Email,
		arg.Username,
		arg.Password,
		arg.FirstName,
		arg.LastName,
		arg.IsAdministrator,
		arg.IsActive,
	)
	if err == nil {
		var id, _ = result.LastInsertId()
		arg.ID = id
		SIGNAL_AFTER_USER_CREATE.Send(arg)
	}
	return err
}

const deleteGroup = `-- name: DeleteGroup :exec
DELETE FROM ` + "`" + `groups` + "`" + ` WHERE id = ?
`

func (q *Queries) DeleteGroup(ctx context.Context, id int64) error {
	SIGNAL_BEFORE_GROUP_DELETE.Send(id)
	_, err := q.db.ExecContext(ctx, deleteGroup, id)
	if err == nil {
		SIGNAL_AFTER_GROUP_DELETE.Send(id)
	}
	return err
}

const deletePermission = `-- name: DeletePermission :exec
DELETE FROM permissions WHERE id = ?
`

func (q *Queries) DeletePermission(ctx context.Context, id int64) error {
	SIGNAL_BEFORE_PERMISSION_DELETE.Send(id)
	_, err := q.db.ExecContext(ctx, deletePermission, id)
	if err == nil {
		SIGNAL_AFTER_PERMISSION_DELETE.Send(id)
	}
	return err
}

const deleteUser = `-- name: DeleteUser :exec
DELETE FROM users WHERE id = ?
`

func (q *Queries) DeleteUser(ctx context.Context, id int64) error {
	SIGNAL_BEFORE_USER_DELETE.Send(id)
	_, err := q.db.ExecContext(ctx, deleteUser, id)
	if err == nil {
		SIGNAL_AFTER_USER_DELETE.Send(id)
	}
	return err
}

const getAllGroups = `-- name: GetAllGroups :many
SELECT id, name, description FROM ` + "`" + `groups` + "`" + `
`

func (q *Queries) GetAllGroups(ctx context.Context) (*linkedlist.Doubly[Group], error) {
	rows, err := q.db.QueryContext(ctx, getAllGroups)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items *linkedlist.Doubly[Group]
	for rows.Next() {
		var i Group
		if err := rows.Scan(&i.ID, &i.Name, &i.Description); err != nil {
			return nil, err
		}
		items.Append(i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getAllPermissions = `-- name: GetAllPermissions :many
SELECT id, name, description FROM permissions
`

func (q *Queries) GetAllPermissions(ctx context.Context) (*linkedlist.Doubly[Permission], error) {
	rows, err := q.db.QueryContext(ctx, getAllPermissions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items = &linkedlist.Doubly[Permission]{}
	for rows.Next() {
		var i Permission
		if err := rows.Scan(&i.ID, &i.Name, &i.Description); err != nil {
			return nil, err
		}
		items.Append(i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getAllUsers = `-- name: GetAllUsers :many
SELECT id, created_at, updated_at, email, username, password, first_name, last_name, is_administrator, is_active FROM users
`

func (q *Queries) GetAllUsers(ctx context.Context) (*linkedlist.Doubly[User], error) {
	rows, err := q.db.QueryContext(ctx, getAllUsers)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items = &linkedlist.Doubly[User]{}
	for rows.Next() {
		var i User
		if err := rows.Scan(
			&i.ID,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.Email,
			&i.Username,
			&i.Password,
			&i.FirstName,
			&i.LastName,
			&i.IsAdministrator,
			&i.IsActive,
		); err != nil {
			return nil, err
		}
		items.Append(i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getGroupByID = `-- name: GetGroupByID :one
SELECT id, name, description FROM ` + "`" + `groups` + "`" + ` WHERE id = ?
`

func (q *Queries) GetGroupByID(ctx context.Context, id int64) (*Group, error) {
	row := q.db.QueryRowContext(ctx, getGroupByID, id)
	var i Group
	err := row.Scan(&i.ID, &i.Name, &i.Description)
	return &i, err
}

const getGroupsByUserID = `-- name: GetGroupsByUserID :many
SELECT g.id, g.name, g.description
FROM ` + "`" + `groups` + "`" + ` g
JOIN user_groups ug ON g.id = ug.group_id
WHERE ug.user_id = ?
`

func (q *Queries) GetGroupsByUserID(ctx context.Context, userID int64) (*linkedlist.Doubly[Group], error) {
	rows, err := q.db.QueryContext(ctx, getGroupsByUserID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items = &linkedlist.Doubly[Group]{}
	for rows.Next() {
		var i Group
		if err := rows.Scan(&i.ID, &i.Name, &i.Description); err != nil {
			return nil, err
		}
		items.Append(i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getPermissionsByGroupID = `-- name: GetPermissionsByGroupID :many
SELECT p.id, p.name, p.description
FROM permissions p
JOIN group_permissions gp ON p.id = gp.permission_id
WHERE gp.group_id = ?
`

func (q *Queries) GetPermissionsByGroupID(ctx context.Context, groupID int64) (*linkedlist.Doubly[Permission], error) {
	rows, err := q.db.QueryContext(ctx, getPermissionsByGroupID, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items = &linkedlist.Doubly[Permission]{}
	for rows.Next() {
		var i Permission
		if err := rows.Scan(&i.ID, &i.Name, &i.Description); err != nil {
			return nil, err
		}
		items.Append(i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getGroupsWithPagination = `-- name: GetGroupsWithPagination :many
SELECT id, name, description FROM ` + "`" + `groups` + "`" + `
ORDER BY id
LIMIT ? OFFSET ?
`

func (q *Queries) GetGroupsWithPagination(ctx context.Context, arg PaginationParams) (*linkedlist.Doubly[Group], error) {
	rows, err := q.db.QueryContext(ctx, getGroupsWithPagination, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items = &linkedlist.Doubly[Group]{}
	for rows.Next() {
		var i Group
		if err := rows.Scan(&i.ID, &i.Name, &i.Description); err != nil {
			return nil, err
		}
		items.Append(i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const groupsNotInUser = `-- name: GroupsDoNotBelongTo :many
SELECT g.id, g.name, g.description
FROM ` + "`" + `groups` + "`" + ` g
WHERE g.id NOT IN (
    SELECT ug.group_id
    FROM user_groups ug
    WHERE ug.user_id = ?
)
`

func (q *Queries) GroupsNotInUser(ctx context.Context, userID int64) (*linkedlist.Doubly[Group], error) {
	rows, err := q.db.QueryContext(ctx, groupsNotInUser, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items = &linkedlist.Doubly[Group]{}
	for rows.Next() {
		var i Group
		if err := rows.Scan(&i.ID, &i.Name, &i.Description); err != nil {
			return nil, err
		}
		items.Append(i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getPermissionByID = `-- name: GetPermissionByID :one
SELECT id, name, description FROM permissions WHERE id = ?
`

func (q *Queries) GetPermissionByID(ctx context.Context, id int64) (*Permission, error) {
	row := q.db.QueryRowContext(ctx, getPermissionByID, id)
	var i Permission
	err := row.Scan(&i.ID, &i.Name, &i.Description)
	return &i, err
}

const getPermissionByName = `-- name: GetPermissionByName :one
SELECT p.id, p.name, p.description
FROM permissions p
WHERE p.name = ?
`

func (q *Queries) GetPermissionByName(ctx context.Context, name string) (Permission, error) {
	row := q.db.QueryRowContext(ctx, getPermissionByName, name)
	var i Permission
	err := row.Scan(&i.ID, &i.Name, &i.Description)
	return i, err
}

const getPermissionsByUserID = `-- name: GetPermissionsByUserID :many
SELECT DISTINCT p.id, p.name, p.description
FROM permissions p
JOIN group_permissions gp ON p.id = gp.permission_id
JOIN user_groups ug ON gp.group_id = ug.group_id
WHERE ug.user_id = ?
`

func (q *Queries) GetPermissionsByUserID(ctx context.Context, userID int64) (*linkedlist.Doubly[Permission], error) {
	rows, err := q.db.QueryContext(ctx, getPermissionsByUserID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items = &linkedlist.Doubly[Permission]{}
	for rows.Next() {
		var i Permission
		if err := rows.Scan(&i.ID, &i.Name, &i.Description); err != nil {
			return nil, err
		}
		items.Append(i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getPermissionsByUserIDAndPermissionNames = `-- name: GetPermissionsByUserIDAndPermissionNames :many
SELECT DISTINCT p.id, p.name, p.description
FROM permissions p
JOIN group_permissions gp ON p.id = gp.permission_id
JOIN user_groups ug ON gp.group_id = ug.group_id
WHERE ug.user_id = ? AND p.name IN (/*SLICE:permissionnames*/?)
`

func (q *Queries) GetPermissionsByUserIDAndPermissionNames(ctx context.Context, arg UserPermissionParams) (*linkedlist.Doubly[Permission], error) {
	sql := getPermissionsByUserIDAndPermissionNames
	var queryParams []interface{}
	queryParams = append(queryParams, arg.UserID)
	if len(arg.Permissionnames) > 0 {
		for _, v := range arg.Permissionnames {
			queryParams = append(queryParams, v)
		}
		sql = strings.Replace(sql, "/*SLICE:permissionnames*/?", strings.Repeat(",?", len(arg.Permissionnames))[1:], 1)
	} else {
		sql = strings.Replace(sql, "/*SLICE:permissionnames*/?", "NULL", 1)
	}
	rows, err := q.db.QueryContext(ctx, sql, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items = &linkedlist.Doubly[Permission]{}
	for rows.Next() {
		var i Permission
		if err := rows.Scan(&i.ID, &i.Name, &i.Description); err != nil {
			return nil, err
		}
		items.Append(i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getPermissionsWithPagination = `-- name: GetPermissionsWithPagination :many
SELECT id, name, description FROM permissions
ORDER BY id
LIMIT ? OFFSET ?
`

func (q *Queries) GetPermissionsWithPagination(ctx context.Context, arg PaginationParams) (*linkedlist.Doubly[Permission], error) {
	rows, err := q.db.QueryContext(ctx, getPermissionsWithPagination, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items = &linkedlist.Doubly[Permission]{}
	for rows.Next() {
		var i Permission
		if err := rows.Scan(&i.ID, &i.Name, &i.Description); err != nil {
			return nil, err
		}
		items.Append(i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getUsersByPermissionID = `-- name: GetUsersByPermissionID :many
SELECT DISTINCT u.id, u.created_at, u.updated_at, u.email, u.username, u.password, u.first_name, u.last_name, u.is_administrator, u.is_active
FROM users u
JOIN user_groups ug ON u.id = ug.user_id
JOIN group_permissions gp ON ug.group_id = gp.group_id
WHERE gp.permission_id = ?
`

func (q *Queries) GetUsersByPermissionID(ctx context.Context, permissionID int64) (*linkedlist.Doubly[User], error) {
	rows, err := q.db.QueryContext(ctx, getUsersByPermissionID, permissionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items = &linkedlist.Doubly[User]{}
	for rows.Next() {
		var i User
		if err := rows.Scan(
			&i.ID,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.Email,
			&i.Username,
			&i.Password,
			&i.FirstName,
			&i.LastName,
			&i.IsAdministrator,
			&i.IsActive,
		); err != nil {
			return nil, err
		}
		items.Append(i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getUsersWithPagination = `-- name: GetUsersWithPagination :many
SELECT id, created_at, updated_at, email, username, password, first_name, last_name, is_administrator, is_active FROM users
ORDER BY id
LIMIT ? OFFSET ?
`

func (q *Queries) GetUsersWithPagination(ctx context.Context, arg PaginationParams) (*linkedlist.Doubly[User], error) {
	rows, err := q.db.QueryContext(ctx, getUsersWithPagination, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items = &linkedlist.Doubly[User]{}
	for rows.Next() {
		var i User
		if err := rows.Scan(
			&i.ID,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.Email,
			&i.Username,
			&i.Password,
			&i.FirstName,
			&i.LastName,
			&i.IsAdministrator,
			&i.IsActive,
		); err != nil {
			return nil, err
		}
		items.Append(i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listPermissionsInGroup = `-- name: ListPermissionsInGroup :many
SELECT p.id, p.name, p.description FROM permissions p JOIN group_permissions gp ON p.id = gp.permission_id WHERE gp.group_id = ?
`

func (q *Queries) ListPermissionsInGroup(ctx context.Context, groupID int64) (*linkedlist.Doubly[Permission], error) {
	rows, err := q.db.QueryContext(ctx, listPermissionsInGroup, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items = &linkedlist.Doubly[Permission]{}
	for rows.Next() {
		var i Permission
		if err := rows.Scan(&i.ID, &i.Name, &i.Description); err != nil {
			return nil, err
		}
		items.Append(i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listUsersInGroup = `-- name: ListUsersInGroup :many
SELECT u.id, u.created_at, u.updated_at, u.email, u.username, u.password, u.first_name, u.last_name, u.is_administrator, u.is_active FROM users u JOIN user_groups ug ON u.id = ug.user_id WHERE ug.group_id = ?
`

func (q *Queries) ListUsersInGroup(ctx context.Context, groupID int64) (*linkedlist.Doubly[User], error) {
	rows, err := q.db.QueryContext(ctx, listUsersInGroup, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items = &linkedlist.Doubly[User]{}
	for rows.Next() {
		var i User
		if err := rows.Scan(
			&i.ID,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.Email,
			&i.Username,
			&i.Password,
			&i.FirstName,
			&i.LastName,
			&i.IsAdministrator,
			&i.IsActive,
		); err != nil {
			return nil, err
		}
		items.Append(i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const removePermissionFromGroup = `-- name: RemovePermissionFromGroup :exec
DELETE FROM group_permissions WHERE group_id = ? AND permission_id = ?
`

func (q *Queries) RemovePermissionFromGroup(ctx context.Context, groupID, permID int64) error {
	_, err := q.db.ExecContext(ctx, removePermissionFromGroup, groupID, permID)
	return err
}

const removeUserFromGroup = `-- name: RemoveUserFromGroup :exec
DELETE FROM user_groups WHERE user_id = ? AND group_id = ?
`

func (q *Queries) RemoveUserFromGroup(ctx context.Context, userID, groupID int64) error {
	_, err := q.db.ExecContext(ctx, removeUserFromGroup, userID, groupID)
	return err
}

const updateGroup = `-- name: UpdateGroup :exec
UPDATE ` + "`" + `groups` + "`" + ` SET name = ?, description = ? WHERE id = ?
`

func (q *Queries) UpdateGroup(ctx context.Context, arg *Group) error {
	SIGNAL_BEFORE_GROUP_UPDATE.Send(arg)
	_, err := q.db.ExecContext(ctx, updateGroup, arg.Name, arg.Description, arg.ID)
	if err == nil {
		SIGNAL_AFTER_GROUP_UPDATE.Send(arg)
	}
	return err
}

const updatePermission = `-- name: UpdatePermission :exec
UPDATE permissions SET name = ?, description = ? WHERE id = ?
`

func (q *Queries) UpdatePermission(ctx context.Context, arg *Permission) error {
	SIGNAL_BEFORE_PERMISSION_UPDATE.Send(arg)
	_, err := q.db.ExecContext(ctx, updatePermission, arg.Name, arg.Description, arg.ID)
	if err == nil {
		SIGNAL_AFTER_PERMISSION_UPDATE.Send(arg)
	}
	return err
}

const updateUser = `-- name: UpdateUser :exec
UPDATE users SET email = ?, username = ?, password = ?, first_name = ?, last_name = ?, is_administrator = ?, is_active = ? WHERE id = ?
`

func (q *Queries) UpdateUser(ctx context.Context, arg *User) error {
	SIGNAL_BEFORE_USER_UPDATE.Send(arg)
	_, err := q.db.ExecContext(ctx, updateUser,
		arg.Email,
		arg.Username,
		arg.Password,
		arg.FirstName,
		arg.LastName,
		arg.IsAdministrator,
		arg.IsActive,
		arg.ID,
	)
	if err == nil {
		SIGNAL_AFTER_USER_UPDATE.Send(arg)
	}
	return err
}

const permissionsNotInGroup = `-- name: PermissionsNotInGroup :many
SELECT p.id, p.name, p.description
FROM permissions p
WHERE p.id NOT IN (
    SELECT permission_id
    FROM group_permissions
    WHERE group_id = ?
)
`

func (q *Queries) PermissionsNotInGroup(ctx context.Context, groupID int64) (*linkedlist.Doubly[Permission], error) {
	rows, err := q.db.QueryContext(ctx, permissionsNotInGroup, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items = &linkedlist.Doubly[Permission]{}
	for rows.Next() {
		var i Permission
		if err := rows.Scan(&i.ID, &i.Name, &i.Description); err != nil {
			return nil, err
		}
		items.Append(i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const permissionsNotInUser = `-- name: PermissionsNotInUser :many
SELECT p.id, p.name, p.description
FROM permissions p
WHERE p.id NOT IN (
    SELECT gp.permission_id
    FROM user_groups ug
    JOIN group_permissions gp ON ug.group_id = gp.group_id
    WHERE ug.user_id = ?
)
`

func (q *Queries) PermissionsNotInUser(ctx context.Context, userID int64) (*linkedlist.Doubly[Permission], error) {
	rows, err := q.db.QueryContext(ctx, permissionsNotInUser, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items = &linkedlist.Doubly[Permission]{}
	for rows.Next() {
		var i Permission
		if err := rows.Scan(&i.ID, &i.Name, &i.Description); err != nil {
			return nil, err
		}
		items.Append(i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
