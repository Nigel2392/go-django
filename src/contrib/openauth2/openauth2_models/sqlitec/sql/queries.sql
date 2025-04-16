-- name: RetrieveUserByID :one
SELECT * FROM oauth2_users
WHERE id = sqlc.arg(id);

-- name: RetrieveUserByIdentifier :one
SELECT * FROM oauth2_users
WHERE unique_identifier = sqlc.arg(unique_identifier);

-- name: CreateUser :execlastid
INSERT INTO oauth2_users (unique_identifier, provider_name, data, access_token, refresh_token, token_type, expires_at, is_administrator, is_active)
VALUES (
    sqlc.arg(unique_identifier),
    sqlc.arg(provider_name),
    sqlc.arg(data),
    sqlc.arg(access_token),
    sqlc.arg(refresh_token),
    sqlc.arg(token_type),
    sqlc.arg(expires_at),
    sqlc.arg(is_administrator),
    sqlc.arg(is_active)
);

-- name: UpdateUser :exec
UPDATE oauth2_users
SET provider_name = sqlc.arg(provider_name),
    data = sqlc.arg(data),
    access_token = sqlc.arg(access_token),
    refresh_token = sqlc.arg(refresh_token),
    token_type = sqlc.arg(token_type),
    expires_at = sqlc.arg(expires_at),
    is_administrator = sqlc.arg(is_administrator),
    is_active = sqlc.arg(is_active)
WHERE id = sqlc.arg(id);

-- name: DeleteUser :exec
DELETE FROM oauth2_users
WHERE id = sqlc.arg(id);

-- name: DeleteUsers :exec
DELETE FROM oauth2_users
WHERE id IN (sqlc.slice(ids));
