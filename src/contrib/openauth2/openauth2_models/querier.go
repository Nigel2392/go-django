package openauth2models

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	django_signals "github.com/Nigel2392/go-django/src/signals"
)

type Querier interface {
	Close() error
	WithTx(tx *sql.Tx) Querier

	RetrieveUsers(ctx context.Context, limit int32, offset int32, ordering ...string) ([]*User, error)
	RetrieveUserByID(ctx context.Context, id uint64) (*User, error)
	RetrieveUserByIdentifier(ctx context.Context, uniqueIdentifier string) (*UserWithToken, error)
	RetrieveTokensByUserID(ctx context.Context, userID uint64) ([]*Token, error)

	CreateUser(ctx context.Context, uniqueIdentifier string, isAdministrator bool, isActive bool) (int64, error)
	UpdateUser(ctx context.Context, uniqueIdentifier string, isAdministrator bool, isActive bool, iD uint64) error

	DeleteUser(ctx context.Context, id uint64) error
	DeleteUsers(ctx context.Context, ids []uint64) error

	CreateUserToken(ctx context.Context, userID uint64, providerName string, data json.RawMessage, accessToken string, refreshToken string, expiresAt time.Time, scope sql.NullString, tokenType sql.NullString) (int64, error)
	UpdateUserToken(ctx context.Context, userID uint64, data json.RawMessage, accessToken string, refreshToken string, expiresAt time.Time, scope sql.NullString, tokenType sql.NullString, providerName string) error
	DeleteUserToken(ctx context.Context, userID uint64) error
	DeleteUserTokenByProvider(ctx context.Context, userID uint64, providerName string) error
	DeleteUserTokens(ctx context.Context, userIds []uint64) error
}

type SignalsQuerier struct {
	Querier
}

func (q *SignalsQuerier) CreateUser(ctx context.Context, uniqueIdentifier string, isAdministrator bool, isActive bool) (int64, error) {
	var u = &User{
		UniqueIdentifier: uniqueIdentifier,
		IsAdministrator:  isAdministrator,
		IsActive:         isActive,
	}
	var err = django_signals.SIGNAL_BEFORE_USER_CREATE.Send(u)
	if err != nil {
		return 0, err
	}

	id, err := q.Querier.CreateUser(ctx, uniqueIdentifier, isAdministrator, isActive)
	if err != nil {
		return 0, err
	}

	u.ID = uint64(id)
	django_signals.SIGNAL_AFTER_USER_CREATE.Send(u)

	return id, nil
}

func (q *SignalsQuerier) UpdateUser(ctx context.Context, uniqueIdentifier string, isAdministrator bool, isActive bool, iD uint64) error {
	var u = &User{
		ID:               uint64(iD),
		UniqueIdentifier: uniqueIdentifier,
		IsAdministrator:  isAdministrator,
		IsActive:         isActive,
	}
	var err = django_signals.SIGNAL_BEFORE_USER_UPDATE.Send(u)
	if err != nil {
		return err
	}

	err = q.Querier.UpdateUser(ctx, uniqueIdentifier, isAdministrator, isActive, iD)
	if err != nil {
		return err
	}

	django_signals.SIGNAL_AFTER_USER_UPDATE.Send(u)
	return nil
}

func (q *SignalsQuerier) DeleteUser(ctx context.Context, id uint64) error {
	err := django_signals.SIGNAL_BEFORE_USER_DELETE.Send(uint64(id))
	if err != nil {
		return err
	}

	err = q.Querier.DeleteUser(ctx, id)
	if err != nil {
		return err
	}

	django_signals.SIGNAL_AFTER_USER_DELETE.Send(uint64(id))
	return nil
}

func (q *SignalsQuerier) DeleteUsers(ctx context.Context, ids []uint64) error {
	err := django_signals.SIGNAL_BEFORE_USER_DELETE.Send(ids)
	if err != nil {
		return err
	}

	err = q.Querier.DeleteUsers(ctx, ids)
	if err != nil {
		return err
	}

	django_signals.SIGNAL_AFTER_USER_DELETE.Send(ids)
	return nil
}
