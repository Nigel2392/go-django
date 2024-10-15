package models

import (
	"context"

	permissions_models "github.com/Nigel2392/go-django/src/contrib/auth/auth-permissions/permissions-models"
)

const allGroups = `-- name: AllGroups :many
SELECT id, name, description
FROM groups
ORDER BY id ASC
LIMIT ?1
OFFSET ?2
`

func (q *Queries) AllGroups(ctx context.Context, limit int32, offset int32) ([]*permissions_models.Group, error) {
	rows, err := q.db.QueryContext(ctx, allGroups, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*permissions_models.Group
	for rows.Next() {
		var i permissions_models.Group
		if err := rows.Scan(&i.ID, &i.Name, &i.Description); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const allPermissions = `-- name: AllPermissions :many
SELECT id, name, description
FROM permissions
ORDER BY id ASC
LIMIT ?1
OFFSET ?2
`

func (q *Queries) AllPermissions(ctx context.Context, limit int32, offset int32) ([]*permissions_models.Permission, error) {
	rows, err := q.db.QueryContext(ctx, allPermissions, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*permissions_models.Permission
	for rows.Next() {
		var i permissions_models.Permission
		if err := rows.Scan(&i.ID, &i.Name, &i.Description); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const deleteGroup = `-- name: DeleteGroup :exec
DELETE FROM groups
WHERE id = ?1
`

func (q *Queries) DeleteGroup(ctx context.Context, id uint64) error {
	_, err := q.db.ExecContext(ctx, deleteGroup, id)
	return err
}

const deleteGroupPermission = `-- name: DeleteGroupPermission :exec
DELETE FROM group_permissions
WHERE group_id = ?1
AND permission_id = ?2
`

func (q *Queries) DeleteGroupPermission(ctx context.Context, groupID uint64, permissionID uint64) error {
	_, err := q.db.ExecContext(ctx, deleteGroupPermission, groupID, permissionID)
	return err
}

const deletePermission = `-- name: DeletePermission :exec
DELETE FROM permissions
WHERE id = ?1
`

func (q *Queries) DeletePermission(ctx context.Context, id uint64) error {
	_, err := q.db.ExecContext(ctx, deletePermission, id)
	return err
}

const deleteUserGroup = `-- name: DeleteUserGroup :exec
DELETE FROM user_groups
WHERE user_id = ?1
AND group_id = ?2
`

func (q *Queries) DeleteUserGroup(ctx context.Context, userID uint64, groupID uint64) error {
	_, err := q.db.ExecContext(ctx, deleteUserGroup, userID, groupID)
	return err
}

const getGroupByID = `-- name: GetGroupByID :one
SELECT id, name, description
FROM groups
WHERE id = ?1
`

func (q *Queries) GetGroupByID(ctx context.Context, id uint64) (*permissions_models.Group, error) {
	row := q.db.QueryRowContext(ctx, getGroupByID, id)
	var i permissions_models.Group
	err := row.Scan(&i.ID, &i.Name, &i.Description)
	return &i, err
}

const getPermissionByID = `-- name: GetPermissionByID :one
SELECT id, name, description
FROM permissions
WHERE id = ?1
`

func (q *Queries) GetPermissionByID(ctx context.Context, id uint64) (*permissions_models.Permission, error) {
	row := q.db.QueryRowContext(ctx, getPermissionByID, id)
	var i permissions_models.Permission
	err := row.Scan(&i.ID, &i.Name, &i.Description)
	return &i, err
}

const insertGroup = `-- name: InsertGroup :execlastid
INSERT INTO groups (name, description) 
VALUES (?1, ?2)
`

func (q *Queries) InsertGroup(ctx context.Context, name string, description string) (int64, error) {
	result, err := q.db.ExecContext(ctx, insertGroup, name, description)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

const insertGroupPermission = `-- name: InsertGroupPermission :execlastid
INSERT INTO group_permissions (group_id, permission_id)
VALUES (?1, ?2)
`

func (q *Queries) InsertGroupPermission(ctx context.Context, groupID uint64, permissionID uint64) (int64, error) {
	result, err := q.db.ExecContext(ctx, insertGroupPermission, groupID, permissionID)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

const insertPermission = `-- name: InsertPermission :execlastid
INSERT INTO permissions (name, description) 
VALUES (?1, ?2)
`

func (q *Queries) InsertPermission(ctx context.Context, name string, description string) (int64, error) {
	result, err := q.db.ExecContext(ctx, insertPermission, name, description)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

const insertUserGroup = `-- name: InsertUserGroup :execlastid
INSERT INTO user_groups (user_id, group_id)
VALUES (?1, ?2)
`

func (q *Queries) InsertUserGroup(ctx context.Context, userID uint64, groupID uint64) (int64, error) {
	result, err := q.db.ExecContext(ctx, insertUserGroup, userID, groupID)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

const permissionsForUser = `-- name: PermissionsForUser :many
SELECT p.id, p.name, p.description 
FROM permissions p
JOIN group_permissions gp ON p.id = gp.permission_id
JOIN user_groups ug ON gp.group_id = ug.group_id
WHERE ug.user_id = ?1
`

func (q *Queries) PermissionsForUser(ctx context.Context, userID uint64) ([]*permissions_models.Permission, error) {
	rows, err := q.db.QueryContext(ctx, permissionsForUser, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*permissions_models.Permission
	for rows.Next() {
		var i permissions_models.Permission
		if err := rows.Scan(&i.ID, &i.Name, &i.Description); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateGroup = `-- name: UpdateGroup :exec
UPDATE groups
SET name = ?1,
    description = ?2
WHERE id = ?3
`

func (q *Queries) UpdateGroup(ctx context.Context, name string, description string, iD uint64) error {
	_, err := q.db.ExecContext(ctx, updateGroup, name, description, iD)
	return err
}

const updatePermission = `-- name: UpdatePermission :exec
UPDATE permissions
SET name = ?1,
    description = ?2
WHERE id = ?3
`

func (q *Queries) UpdatePermission(ctx context.Context, name string, description string, iD uint64) error {
	_, err := q.db.ExecContext(ctx, updatePermission, name, description, iD)
	return err
}

const userGroups = `-- name: UserGroups :many
SELECT g.id, g.name, g.description, p.id, p.name, p.description
FROM groups g
JOIN group_permissions gp ON g.id = gp.group_id
JOIN permissions p ON gp.permission_id = p.id
WHERE g.id = ?1
`

func (q *Queries) UserGroups(ctx context.Context, id uint64) (g permissions_models.Group, p []*permissions_models.Permission, err error) {
	rows, err := q.db.QueryContext(ctx, userGroups, id)
	if err != nil {
		return g, p, err
	}
	defer rows.Close()
	var group permissions_models.Group
	var items []*permissions_models.Permission
	for rows.Next() {
		var i permissions_models.Permission
		if err := rows.Scan(
			&group.ID,
			&group.Name,
			&group.Description,
			&i.ID,
			&i.Name,
			&i.Description,
		); err != nil {
			return g, p, err
		}
		items = append(items, &i)
	}
	if err := rows.Close(); err != nil {
		return g, p, err
	}
	if err := rows.Err(); err != nil {
		return g, p, err
	}
	return group, items, nil
}

const userHasPermission = `-- name: UserHasPermission :one
SELECT COUNT(*)
FROM permissions p
JOIN group_permissions gp ON p.id = gp.permission_id
JOIN user_groups ug ON gp.group_id = ug.group_id
WHERE ug.user_id = ?1
AND p.name = ?2
`

func (q *Queries) UserHasPermission(ctx context.Context, userID uint64, name string) (int64, error) {
	row := q.db.QueryRowContext(ctx, userHasPermission, userID, name)
	var count int64
	err := row.Scan(&count)
	return count, err
}
