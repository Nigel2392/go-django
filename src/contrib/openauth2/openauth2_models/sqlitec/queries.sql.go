// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: queries.sql

package openauth2_models_sqlite

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	openauth2models "github.com/Nigel2392/go-django/src/contrib/openauth2/openauth2_models"
)

const createUser = `-- name: CreateUser :execlastid
INSERT INTO oauth2_users (unique_identifier, provider_name, data, access_token, refresh_token, expires_at, is_administrator, is_active)
VALUES (
    ?1,
    ?2,
    ?3,
    ?4,
    ?5,
    ?6,
    ?7,
    ?8
)
`

func (q *Queries) CreateUser(ctx context.Context, uniqueIdentifier string, providerName string, data json.RawMessage, accessToken string, refreshToken string, expiresAt time.Time, isAdministrator bool, isActive bool) (int64, error) {
	result, err := q.exec(ctx, q.createUserStmt, createUser,
		uniqueIdentifier,
		providerName,
		data,
		accessToken,
		refreshToken,
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
WHERE id = ?1
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
SELECT id, unique_identifier, provider_name, data, access_token, refresh_token, expires_at, created_at, updated_at, is_administrator, is_active FROM oauth2_users
WHERE id = ?1
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
		&i.ExpiresAt,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.IsAdministrator,
		&i.IsActive,
	)
	return &i, err
}

const retrieveUserByIdentifier = `-- name: RetrieveUserByIdentifier :one
SELECT id, unique_identifier, provider_name, data, access_token, refresh_token, expires_at, created_at, updated_at, is_administrator, is_active FROM oauth2_users
WHERE unique_identifier = ?1
`

func (q *Queries) RetrieveUserByIdentifier(ctx context.Context, uniqueIdentifier string) (*openauth2models.User, error) {
	row := q.queryRow(ctx, q.retrieveUserByIdentifierStmt, retrieveUserByIdentifier, uniqueIdentifier)
	var i openauth2models.User
	err := row.Scan(
		&i.ID,
		&i.UniqueIdentifier,
		&i.ProviderName,
		&i.Data,
		&i.AccessToken,
		&i.RefreshToken,
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
SET provider_name = ?1,
    data = ?2,
    access_token = ?3,
    refresh_token = ?4,
    expires_at = ?5,
    is_administrator = ?6,
    is_active = ?7
WHERE id = ?8
`

func (q *Queries) UpdateUser(ctx context.Context, providerName string, data json.RawMessage, accessToken string, refreshToken string, expiresAt time.Time, isAdministrator bool, isActive bool, iD uint64) error {
	_, err := q.exec(ctx, q.updateUserStmt, updateUser,
		providerName,
		data,
		accessToken,
		refreshToken,
		expiresAt,
		isAdministrator,
		isActive,
		iD,
	)
	return err
}
