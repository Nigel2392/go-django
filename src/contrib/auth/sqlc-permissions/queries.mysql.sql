-- name: InsertGroup :execlastid
INSERT INTO `groups` (`name`, `description`) 
VALUES (sqlc.arg(name), sqlc.arg(description));

-- name: InsertPermission :execlastid
INSERT INTO `permissions` (`name`, `description`) 
VALUES (sqlc.arg(name), sqlc.arg(description));

-- name: InsertGroupPermission :execlastid
INSERT INTO `group_permissions` (`group_id`, `permission_id`)
VALUES (sqlc.arg(group_id), sqlc.arg(permission_id));

-- name: InsertUserGroup :execlastid
INSERT INTO `user_groups` (`user_id`, `group_id`)
VALUES (sqlc.arg(user_id), sqlc.arg(group_id));

-- name: GetGroupByID :one
SELECT *
FROM `groups`
WHERE `id` = sqlc.arg(id);

-- name: GetPermissionByID :one
SELECT *
FROM `permissions`
WHERE `id` = sqlc.arg(id);

-- name: UpdateGroup :exec
UPDATE `groups`
SET `name` = sqlc.arg(name),
    `description` = sqlc.arg(description)
WHERE `id` = sqlc.arg(id);

-- name: UpdatePermission :exec
UPDATE `permissions`
SET `name` = sqlc.arg(name),
    `description` = sqlc.arg(description)
WHERE `id` = sqlc.arg(id);

-- name: DeleteGroup :exec
DELETE FROM `groups`
WHERE `id` = sqlc.arg(id);

-- name: DeletePermission :exec
DELETE FROM `permissions`
WHERE `id` = sqlc.arg(id);

-- name: DeleteGroupPermission :exec
DELETE FROM `group_permissions`
WHERE `group_id` = sqlc.arg(group_id)
AND `permission_id` = sqlc.arg(permission_id);

-- name: DeleteUserGroup :exec
DELETE FROM `user_groups`
WHERE `user_id` = sqlc.arg(user_id)
AND `group_id` = sqlc.arg(group_id);

-- name: AllGroups :many
SELECT *
FROM `groups`
ORDER BY `id` ASC
LIMIT ?
OFFSET ?;

-- name: AllPermissions :many
SELECT *
FROM `permissions`
ORDER BY `id` ASC
LIMIT ?
OFFSET ?;

-- name: PermissionsForUser :many
SELECT p.* 
FROM `permissions` p
JOIN `group_permissions` gp ON p.id = gp.permission_id
JOIN `user_groups` ug ON gp.group_id = ug.group_id
WHERE ug.user_id = sqlc.arg(user_id);

-- name: UserHasPermission :one
SELECT COUNT(*)
FROM `permissions` p
JOIN `group_permissions` gp ON p.id = gp.permission_id
JOIN `user_groups` ug ON gp.group_id = ug.group_id
WHERE ug.user_id = sqlc.arg(user_id)
AND p.name = sqlc.arg(permission_name);

-- name: UserGroups :many
SELECT sqlc.embed(g), sqlc.embed(p)
FROM `groups` g
JOIN `group_permissions` gp ON g.id = gp.group_id
JOIN `permissions` p ON gp.permission_id = p.id
WHERE g.id = sqlc.arg(id);