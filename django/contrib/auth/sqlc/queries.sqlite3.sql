-- name: Retrieve :many
SELECT * FROM users
ORDER BY updated_at DESC
LIMIT sqlc.arg(limit)
OFFSET sqlc.arg(offset);

-- name: RetrieveMany :many
SELECT * FROM users
WHERE is_active = sqlc.arg(is_active) 
AND is_administrator = sqlc.arg(is_administrator)
ORDER BY updated_at DESC
LIMIT sqlc.arg(limit)
OFFSET sqlc.arg(offset);

-- name: Count :one
SELECT COUNT(*) FROM users;

-- name: CountMany :one
SELECT COUNT(*) FROM users
WHERE is_active = sqlc.arg(is_active)
AND is_administrator = sqlc.arg(is_administrator);

-- name: RetrieveByID :one
SELECT * FROM users WHERE id = sqlc.arg(id);

-- name: RetrieveByEmail :one
SELECT * FROM users 
WHERE LOWER(email) = LOWER(sqlc.arg(email))
LIMIT 1;

-- name: RetrieveByUsername :one
SELECT * FROM users
WHERE LOWER(username) = LOWER(sqlc.arg(username))
LIMIT 1;

-- name: CreateUser :exec
INSERT INTO users (
    email,
    username,
    password,
    first_name,
    last_name,
    is_administrator,
    is_active
) VALUES (
    sqlc.arg(email),
    sqlc.arg(username),
    sqlc.arg(password),
    sqlc.arg(first_name),
    sqlc.arg(last_name),
    sqlc.arg(is_administrator),
    sqlc.arg(is_active)
);

-- name: UpdateUser :exec
UPDATE users SET
    email = sqlc.arg(email),
    username = sqlc.arg(username),
    password = sqlc.arg(password),
    first_name = sqlc.arg(first_name),
    last_name = sqlc.arg(last_name),
    is_administrator = sqlc.arg(is_administrator),
    is_active = sqlc.arg(is_active),
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id);

-- name: DeleteUser :exec
DELETE FROM users WHERE id = sqlc.arg(id);