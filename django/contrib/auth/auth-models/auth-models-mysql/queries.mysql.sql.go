package models_mysql

import (
	"context"

	models "github.com/Nigel2392/django/contrib/auth/auth-models"
)

const count = `-- name: Count :one
SELECT COUNT(*) FROM users
`

func (q *Queries) Count(ctx context.Context) (int64, error) {
	row := q.queryRow(ctx, q.countStmt, count)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const countMany = `-- name: CountMany :one
SELECT COUNT(*) FROM users
WHERE is_active = ?
AND is_administrator = ?
`

func (q *Queries) CountMany(ctx context.Context, isActive bool, isAdministrator bool) (int64, error) {
	row := q.queryRow(ctx, q.countManyStmt, countMany, isActive, isAdministrator)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const createUser = `-- name: CreateUser :exec
INSERT INTO users (
    email,
    username,
    password,
    first_name,
    last_name,
    is_administrator,
    is_active
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
)
`

func (q *Queries) CreateUser(ctx context.Context, email string, username string, password string, firstName string, lastName string, isAdministrator bool, isActive bool) error {
	_, err := q.exec(ctx, q.createUserStmt, createUser,
		email,
		username,
		password,
		firstName,
		lastName,
		isAdministrator,
		isActive,
	)
	return err
}

const deleteUser = `-- name: DeleteUser :exec
DELETE FROM users WHERE id = ?
`

func (q *Queries) DeleteUser(ctx context.Context, id uint64) error {
	_, err := q.exec(ctx, q.deleteUserStmt, deleteUser, id)
	return err
}

const retrieve = `-- name: Retrieve :many
SELECT id, created_at, updated_at, email, username, password, first_name, last_name, is_administrator, is_active FROM users
ORDER BY updated_at DESC
LIMIT ?
OFFSET ?
`

func (q *Queries) Retrieve(ctx context.Context, limit int32, offset int32) ([]*models.User, error) {
	rows, err := q.query(ctx, q.retrieveStmt, retrieve, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*models.User
	for rows.Next() {
		var i models.User
		if err := rows.Scan(
			&i.ID,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.Email,
			&i.Username,
			&i.Password,
			&i.FirstName,
			&i.LastName,
			&i.IsAdministrator,
			&i.IsActive,
		); err != nil {
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

const retrieveByEmail = `-- name: RetrieveByEmail :one
SELECT id, created_at, updated_at, email, username, password, first_name, last_name, is_administrator, is_active FROM users 
WHERE LOWER(email) = LOWER(?)
LIMIT 1
`

func (q *Queries) RetrieveByEmail(ctx context.Context, email string) (*models.User, error) {
	row := q.queryRow(ctx, q.retrieveByEmailStmt, retrieveByEmail, email)
	var i models.User
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Email,
		&i.Username,
		&i.Password,
		&i.FirstName,
		&i.LastName,
		&i.IsAdministrator,
		&i.IsActive,
	)
	return &i, err
}

const retrieveByID = `-- name: RetrieveByID :one
SELECT id, created_at, updated_at, email, username, password, first_name, last_name, is_administrator, is_active FROM users WHERE id = ?
`

func (q *Queries) RetrieveByID(ctx context.Context, id uint64) (*models.User, error) {
	row := q.queryRow(ctx, q.retrieveByIDStmt, retrieveByID, id)
	var i models.User
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Email,
		&i.Username,
		&i.Password,
		&i.FirstName,
		&i.LastName,
		&i.IsAdministrator,
		&i.IsActive,
	)
	return &i, err
}

const retrieveByUsername = `-- name: RetrieveByUsername :one
SELECT id, created_at, updated_at, email, username, password, first_name, last_name, is_administrator, is_active FROM users
WHERE LOWER(username) = LOWER(?)
LIMIT 1
`

func (q *Queries) RetrieveByUsername(ctx context.Context, username string) (*models.User, error) {
	row := q.queryRow(ctx, q.retrieveByUsernameStmt, retrieveByUsername, username)
	var i models.User
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Email,
		&i.Username,
		&i.Password,
		&i.FirstName,
		&i.LastName,
		&i.IsAdministrator,
		&i.IsActive,
	)
	return &i, err
}

const retrieveMany = `-- name: RetrieveMany :many
SELECT id, created_at, updated_at, email, username, password, first_name, last_name, is_administrator, is_active FROM users
WHERE is_active = ? 
AND is_administrator = ?
ORDER BY updated_at DESC
LIMIT ?
OFFSET ?
`

func (q *Queries) RetrieveMany(ctx context.Context, isActive bool, isAdministrator bool, limit int32, offset int32) ([]*models.User, error) {
	rows, err := q.query(ctx, q.retrieveManyStmt, retrieveMany,
		isActive,
		isAdministrator,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*models.User
	for rows.Next() {
		var i models.User
		if err := rows.Scan(
			&i.ID,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.Email,
			&i.Username,
			&i.Password,
			&i.FirstName,
			&i.LastName,
			&i.IsAdministrator,
			&i.IsActive,
		); err != nil {
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

const updateUser = `-- name: UpdateUser :exec
UPDATE users SET
    email = ?,
    username = ?,
    password = ?,
    first_name = ?,
    last_name = ?,
    is_administrator = ?,
    is_active = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
`

func (q *Queries) UpdateUser(ctx context.Context, email string, username string, password string, firstName string, lastName string, isAdministrator bool, isActive bool, iD uint64) error {
	_, err := q.exec(ctx, q.updateUserStmt, updateUser,
		email,
		username,
		password,
		firstName,
		lastName,
		isAdministrator,
		isActive,
		iD,
	)
	return err
}
