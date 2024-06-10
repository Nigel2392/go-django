-- name: CreateUser :exec
INSERT INTO users (email, username, password, first_name, last_name, is_administrator, is_active)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: UserByID :one
SELECT * FROM users WHERE id = ?;

-- name: UserByEmail :one
SELECT * FROM users WHERE email = ?;

-- name: UserByUsername :one
SELECT * FROM users WHERE username = ?;

-- name: UpdateUser :exec
UPDATE users SET email = ?, username = ?, password = ?, first_name = ?, last_name = ?, is_administrator = ?, is_active = ? WHERE id = ?;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = ?;

-- name: CreateGroup :exec
INSERT INTO `groups` (name, description)
VALUES (?, ?);

-- name: GetGroupByID :one
SELECT * FROM `groups` WHERE id = ?;

-- name: UpdateGroup :exec
UPDATE `groups` SET name = ?, description = ? WHERE id = ?;

-- name: DeleteGroup :exec
DELETE FROM `groups` WHERE id = ?;

-- name: CreatePermission :exec
INSERT INTO permissions (name, description)
VALUES (?, ?);

-- name: GetPermissionByID :one
SELECT * FROM permissions WHERE id = ?;

-- name: UpdatePermission :exec
UPDATE permissions SET name = ?, description = ? WHERE id = ?;

-- name: DeletePermission :exec
DELETE FROM permissions WHERE id = ?;

-- name: AddUserToGroup :exec
INSERT INTO user_groups (user_id, group_id)
VALUES (?, ?);

-- name: RemoveUserFromGroup :exec
DELETE FROM user_groups WHERE user_id = ? AND group_id = ?;

-- name: ListUsersInGroup :many
SELECT u.* FROM users u JOIN user_groups ug ON u.id = ug.user_id WHERE ug.group_id = ?;

-- name: AddPermissionToGroup :exec
INSERT INTO group_permissions (group_id, permission_id)
VALUES (?, ?);

-- name: RemovePermissionFromGroup :exec
DELETE FROM group_permissions WHERE group_id = ? AND permission_id = ?;

-- name: ListPermissionsInGroup :many
SELECT p.* FROM permissions p JOIN group_permissions gp ON p.id = gp.permission_id WHERE gp.group_id = ?;

-- name: GetAllUsers :many
SELECT * FROM users;

-- name: GetAllGroups :many
SELECT * FROM `groups`;

-- name: GetAllPermissions :many
SELECT * FROM permissions;

-- name: GetGroupsByUserID :many
SELECT g.*
FROM `groups` g
JOIN user_groups ug ON g.id = ug.group_id
WHERE ug.user_id = ?;

-- name: GetUsersByPermissionID :many
SELECT DISTINCT u.*
FROM users u
JOIN user_groups ug ON u.id = ug.user_id
JOIN group_permissions gp ON ug.group_id = gp.group_id
WHERE gp.permission_id = ?;

-- name: GetPermissionsByUserID :many
SELECT DISTINCT p.*
FROM permissions p
JOIN group_permissions gp ON p.id = gp.permission_id
JOIN user_groups ug ON gp.group_id = ug.group_id
WHERE ug.user_id = ?;

-- name: GetPermissionByName :one
SELECT p.*
FROM permissions p
WHERE p.name = ?;

-- name: GetPermissionsByUserIDAndPermissionNames :many
SELECT DISTINCT p.*
FROM permissions p
JOIN group_permissions gp ON p.id = gp.permission_id
JOIN user_groups ug ON gp.group_id = ug.group_id
WHERE ug.user_id = ? AND p.name IN (sqlc.slice(PermissionNames));

-- name: GetUsersWithPagination :many
SELECT * FROM users
ORDER BY id
LIMIT ? OFFSET ?;

-- name: GetGroupsWithPagination :many
SELECT * FROM `groups`
ORDER BY id
LIMIT ? OFFSET ?;

-- name: GetPermissionsWithPagination :many
SELECT * FROM permissions
ORDER BY id
LIMIT ? OFFSET ?;

-- name: GetUserById :many
SELECT
    sqlc.embed(u),
    g.id AS group_id, g.name AS group_name, g.description AS group_description,
    p.id AS permission_id, p.name AS permission_name, p.description AS permission_description
FROM
    users u
JOIN
    user_groups ug ON u.id = ug.user_id
JOIN
    `groups` g ON ug.group_id = g.id
JOIN
    group_permissions gp ON g.id = gp.group_id
JOIN
    permissions p ON gp.permission_id = p.id
WHERE
    u.id = ?
ORDER BY
    u.id, g.name, p.name;

-- name: GetUserByEmail :one
SELECT
    sqlc.embed(u),
    g.id AS group_id, g.name AS group_name, g.description AS group_description,
    p.id AS permission_id, p.name AS permission_name, p.description AS permission_description
FROM
    users u
JOIN
    user_groups ug ON u.id = ug.user_id
JOIN
    `groups` g ON ug.group_id = g.id
JOIN
    group_permissions gp ON g.id = gp.group_id
JOIN
    permissions p ON gp.permission_id = p.id
WHERE
    u.email = ?
ORDER BY
    u.id, g.name, p.name
LIMIT 1;

-- name: GetUserByName :one
SELECT
    sqlc.embed(u),
    g.id AS group_id, g.name AS group_name, g.description AS group_description,
    p.id AS permission_id, p.name AS permission_name, p.description AS permission_description
FROM
    users u
JOIN
    user_groups ug ON u.id = ug.user_id
JOIN
    `groups` g ON ug.group_id = g.id
JOIN
    group_permissions gp ON g.id = gp.group_id
JOIN
    permissions p ON gp.permission_id = p.id
WHERE
    u.username = ?
ORDER BY
    u.id
LIMIT 1;

-- name: CheckUserHasPermissions :one
SELECT EXISTS (
    SELECT 1
    FROM users u
    JOIN user_groups ug ON u.id = ug.user_id
    JOIN `groups` g ON ug.group_id = g.id
    JOIN group_permissions gp ON g.id = gp.group_id
    JOIN permissions p ON gp.permission_id = p.id
    WHERE u.id = ? AND p.name IN (sqlc.slice(PermissionNames))
) as has_permissions;

-- name: PermissionsNotInUser :many
SELECT p.*
FROM permissions p
WHERE p.id NOT IN (
    SELECT gp.permission_id
    FROM user_groups ug
    JOIN group_permissions gp ON ug.group_id = gp.group_id
    WHERE ug.user_id = ?
);

-- name: PermissionsNotInGroup :many
SELECT p.*
FROM permissions p
WHERE p.id NOT IN (
    SELECT permission_id
    FROM group_permissions
    WHERE group_id = ?
);

-- name: GroupsDoNotBelongTo :many
SELECT g.*
FROM `groups` g
WHERE g.id NOT IN (
    SELECT ug.group_id
    FROM user_groups ug
    WHERE ug.user_id = ?
);

-- name: DeleteUserGroups :exec
DELETE FROM user_groups
WHERE user_id = ?;

-- name: AddUserToGroups :exec
INSERT INTO user_groups (user_id, group_id)
SELECT ? AS user_id, group_id
FROM (SELECT * FROM `groups` WHERE id IN (sqlc.slice('group_ids'))) AS t;