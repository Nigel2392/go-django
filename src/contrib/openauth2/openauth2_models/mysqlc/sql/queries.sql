-- name: RetrieveUserByID :one
SELECT * FROM oauth2_users
WHERE id = sqlc.arg(id);

-- name: RetrieveUserByIdentifier :one
SELECT * FROM oauth2_users
WHERE unique_identifier = sqlc.arg(unique_identifier);

-- name: CreateUser :execlastid
INSERT INTO oauth2_users (unique_identifier, is_administrator, is_active)
VALUES (
    sqlc.arg(unique_identifier),
    sqlc.arg(is_administrator),
    sqlc.arg(is_active)
);

-- name: UpdateUser :exec
UPDATE oauth2_users
SET unique_identifier = sqlc.arg(unique_identifier),
    is_administrator = sqlc.arg(is_administrator),
    is_active = sqlc.arg(is_active)
WHERE id = sqlc.arg(id);

-- name: UpdateUserToken :exec
UPDATE oauth2_tokens
SET user_id = sqlc.arg(user_id),
    data = sqlc.arg(data),
    access_token = sqlc.arg(access_token),
    refresh_token = sqlc.arg(refresh_token),
    expires_at = sqlc.arg(expires_at),
    scope = sqlc.arg(scope),
    token_type = sqlc.arg(token_type)
WHERE provider_name = sqlc.arg(provider_name);

-- name: DeleteUser :exec
DELETE FROM oauth2_users
WHERE id = sqlc.arg(id);

-- name: DeleteUserToken :exec
DELETE FROM oauth2_tokens
WHERE user_id = sqlc.arg(user_id);

-- name: DeleteUserTokenByProvider :exec
DELETE FROM oauth2_tokens
WHERE user_id = sqlc.arg(user_id)
AND provider_name = sqlc.arg(provider_name);

-- name: DeleteUsers :exec
DELETE FROM oauth2_users
WHERE id IN (sqlc.slice(ids));

-- name: DeleteUserTokens :exec
DELETE FROM oauth2_tokens
WHERE user_id IN (sqlc.slice(user_ids));


