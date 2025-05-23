// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: queries.sql

package openauth2_models_mysql

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	openauth2models "github.com/Nigel2392/go-django/src/contrib/openauth2/openauth2_models"
)

const createUser = `-- name: CreateUser :execlastid
INSERT INTO oauth2_users (unique_identifier, provider_name, data, access_token, refresh_token, token_type, expires_at, is_administrator, is_active)
VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
)
`

func (q *Queries) CreateUser(ctx context.Context, uniqueIdentifier string, providerName string, data json.RawMessage, accessToken string, refreshToken string, tokenType string, expiresAt time.Time, isAdministrator bool, isActive bool) (int64, error) {
	result, err := q.exec(ctx, q.createUserStmt, createUser,
		uniqueIdentifier,
		providerName,
		data,
		accessToken,
		refreshToken,
		tokenType,
		expiresAt,
		isAdministrator,
		isActive,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

const deleteUser = `-- name: DeleteUser :exec
DELETE FROM oauth2_users
WHERE id = ?
`

func (q *Queries) DeleteUser(ctx context.Context, id uint64) error {
	_, err := q.exec(ctx, q.deleteUserStmt, deleteUser, id)
	return err
}

const deleteUsers = `-- name: DeleteUsers :exec
DELETE FROM oauth2_users
WHERE id IN (/*SLICE:ids*/?)
`

func (q *Queries) DeleteUsers(ctx context.Context, ids []uint64) error {
	query := deleteUsers
	var queryParams []interface{}
	if len(ids) > 0 {
		for _, v := range ids {
			queryParams = append(queryParams, v)
		}
		query = strings.Replace(query, "/*SLICE:ids*/?", strings.Repeat(",?", len(ids))[1:], 1)
	} else {
		query = strings.Replace(query, "/*SLICE:ids*/?", "NULL", 1)
	}
	_, err := q.exec(ctx, nil, query, queryParams...)
	return err
}

const retrieveUserByID = `-- name: RetrieveUserByID :one
SELECT id, unique_identifier, provider_name, data, access_token, refresh_token, token_type, expires_at, created_at, updated_at, is_administrator, is_active FROM oauth2_users
WHERE id = ?
`

func (q *Queries) RetrieveUserByID(ctx context.Context, id uint64) (*openauth2models.User, error) {
	row := q.queryRow(ctx, q.retrieveUserByIDStmt, retrieveUserByID, id)
	var i openauth2models.User
	err := row.Scan(
		&i.ID,
		&i.UniqueIdentifier,
		&i.ProviderName,
		&i.Data,
		&i.AccessToken,
		&i.RefreshToken,
		&i.TokenType,
		&i.ExpiresAt,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.IsAdministrator,
		&i.IsActive,
	)
	return &i, err
}

const retrieveUserByIdentifier = `-- name: RetrieveUserByIdentifier :one
SELECT id, unique_identifier, provider_name, data, access_token, refresh_token, token_type, expires_at, created_at, updated_at, is_administrator, is_active FROM oauth2_users
WHERE unique_identifier = ? AND provider_name = ?
`

func (q *Queries) RetrieveUserByIdentifier(ctx context.Context, uniqueIdentifier string, providerName string) (*openauth2models.User, error) {
	row := q.queryRow(ctx, q.retrieveUserByIdentifierStmt, retrieveUserByIdentifier, uniqueIdentifier, providerName)
	var i openauth2models.User
	err := row.Scan(
		&i.ID,
		&i.UniqueIdentifier,
		&i.ProviderName,
		&i.Data,
		&i.AccessToken,
		&i.RefreshToken,
		&i.TokenType,
		&i.ExpiresAt,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.IsAdministrator,
		&i.IsActive,
	)
	return &i, err
}

const updateUser = `-- name: UpdateUser :exec
UPDATE oauth2_users
SET provider_name = ?,
    data = ?,
    access_token = ?,
    refresh_token = ?,
    token_type = ?,
    expires_at = ?,
    is_administrator = ?,
    is_active = ?
WHERE id = ?
`

func (q *Queries) UpdateUser(ctx context.Context, providerName string, data json.RawMessage, accessToken string, refreshToken string, tokenType string, expiresAt time.Time, isAdministrator bool, isActive bool, iD uint64) error {
	_, err := q.exec(ctx, q.updateUserStmt, updateUser,
		providerName,
		data,
		accessToken,
		refreshToken,
		tokenType,
		expiresAt,
		isAdministrator,
		isActive,
		iD,
	)
	return err
}
