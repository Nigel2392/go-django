-- name: RetrieveUserByID :one
SELECT * FROM oauth2_users
WHERE id = sqlc.arg(id);

-- name: RetrieveTokensByUserID :many
SELECT * FROM oauth2_tokens
WHERE user_id = sqlc.arg(user_id);

-- name: RetrieveUserByIdentifier :one
SELECT sqlc.embed(oauth2_users), sqlc.embed(oauth2_tokens) FROM oauth2_users
JOIN oauth2_tokens ON oauth2_users.id = oauth2_tokens.user_id
WHERE oauth2_users.unique_identifier = sqlc.arg(unique_identifier);

-- name: CreateUser :execlastid
INSERT INTO oauth2_users (unique_identifier, is_administrator, is_active)
VALUES (
    sqlc.arg(unique_identifier),
    sqlc.arg(is_administrator),
    sqlc.arg(is_active)
);

-- name: CreateUserToken :execlastid
INSERT INTO oauth2_tokens (user_id, provider_name, data, access_token, refresh_token, expires_at, scope, token_type)
VALUES (
    sqlc.arg(user_id),
    sqlc.arg(provider_name),
    sqlc.arg(data),
    sqlc.arg(access_token),
    sqlc.arg(refresh_token),
    sqlc.arg(expires_at),
    sqlc.arg(scope),
    sqlc.arg(token_type)
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


