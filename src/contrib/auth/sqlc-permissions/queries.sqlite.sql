-- name: InsertGroup :execlastid
INSERT INTO groups (name, description) 
VALUES (?1, ?2);

-- name: InsertPermission :execlastid
INSERT INTO permissions (name, description) 
VALUES (?1, ?2);

-- name: InsertGroupPermission :execlastid
INSERT INTO group_permissions (group_id, permission_id)
VALUES (?1, ?2);

-- name: InsertUserGroup :execlastid
INSERT INTO user_groups (user_id, group_id)
VALUES (?1, ?2);

-- name: GetGroupByID :one
SELECT *
FROM groups
WHERE id = ?1;

-- name: GetPermissionByID :one
SELECT *
FROM permissions
WHERE id = ?1;

-- name: UpdateGroup :exec
UPDATE groups
SET name = ?1,
    description = ?2
WHERE id = ?3;

-- name: UpdatePermission :exec
UPDATE permissions
SET name = ?1,
    description = ?2
WHERE id = ?3;

-- name: DeleteGroup :exec
DELETE FROM groups
WHERE id = ?1;

-- name: DeletePermission :exec
DELETE FROM permissions
WHERE id = ?1;

-- name: DeleteGroupPermission :exec
DELETE FROM group_permissions
WHERE group_id = ?1
AND permission_id = ?2;

-- name: DeleteUserGroup :exec
DELETE FROM user_groups
WHERE user_id = ?1
AND group_id = ?2;

-- name: AllGroups :many
SELECT *
FROM groups
ORDER BY id ASC
LIMIT ?1
OFFSET ?2;

-- name: AllPermissions :many
SELECT *
FROM permissions
ORDER BY id ASC
LIMIT ?1
OFFSET ?2;

-- name: PermissionsForUser :many
SELECT p.* 
FROM permissions p
JOIN group_permissions gp ON p.id = gp.permission_id
JOIN user_groups ug ON gp.group_id = ug.group_id
WHERE ug.user_id = ?1;

-- name: UserHasPermission :one
SELECT COUNT(*)
FROM permissions p
JOIN group_permissions gp ON p.id = gp.permission_id
JOIN user_groups ug ON gp.group_id = ug.group_id
WHERE ug.user_id = ?1
AND p.name = ?2;

-- name: UserGroups :many
SELECT sqlc.embed(g), sqlc.embed(p)
FROM groups g
JOIN group_permissions gp ON g.id = gp.group_id
JOIN permissions p ON gp.permission_id = p.id
WHERE g.id = ?1;
