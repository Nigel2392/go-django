package openauth2models

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Nigel2392/go-django-queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core"
)

type Querier interface {
	Close() error
	WithTx(tx drivers.Transaction) Querier

	RetrieveUsers(ctx context.Context, limit int32, offset int32, ordering ...string) ([]*User, error)
	RetrieveUserByID(ctx context.Context, id uint64) (*User, error)
	RetrieveUserByIdentifier(ctx context.Context, uniqueIdentifier string, providerName string) (*User, error)

	CreateUser(ctx context.Context, uniqueIdentifier string, providerName string, data json.RawMessage, accessToken string, refreshToken string, tokenType string, expiresAt time.Time, isAdministrator bool, isActive bool) (int64, error)
	DeleteUser(ctx context.Context, id uint64) error
	DeleteUsers(ctx context.Context, ids []uint64) error
	UpdateUser(ctx context.Context, providerName string, data json.RawMessage, accessToken string, refreshToken string, tokenType string, expiresAt time.Time, isAdministrator bool, isActive bool, iD uint64) error
}

type SignalsQuerier struct {
	Querier
}

func (q *SignalsQuerier) CreateUser(ctx context.Context, uniqueIdentifier string, providerName string, data json.RawMessage, accessToken string, refreshToken string, tokenType string, expiresAt time.Time, isAdministrator bool, isActive bool) (int64, error) {
	var u = &User{
		UniqueIdentifier: uniqueIdentifier,
		ProviderName:     providerName,
		Data:             data,
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		ExpiresAt:        expiresAt,
		TokenType:        tokenType,
		IsAdministrator:  isAdministrator,
		IsActive:         isActive,
		IsLoggedIn:       true,
	}
	var err = core.SIGNAL_BEFORE_USER_CREATE.Send(u)
	if err != nil {
		return 0, err
	}

	id, err := q.Querier.CreateUser(ctx, uniqueIdentifier, providerName, data, accessToken, refreshToken, tokenType, expiresAt, isAdministrator, isActive)
	if err != nil {
		return 0, err
	}

	u.ID = uint64(id)
	core.SIGNAL_AFTER_USER_CREATE.Send(u)

	return id, nil
}

func (q *SignalsQuerier) UpdateUser(ctx context.Context, providerName string, data json.RawMessage, accessToken string, refreshToken string, tokenType string, expiresAt time.Time, isAdministrator bool, isActive bool, iD uint64) error {
	var u = &User{
		ID:              iD,
		ProviderName:    providerName,
		Data:            data,
		AccessToken:     accessToken,
		RefreshToken:    refreshToken,
		TokenType:       tokenType,
		ExpiresAt:       expiresAt,
		IsAdministrator: isAdministrator,
		IsActive:        isActive,
		IsLoggedIn:      true,
	}
	var err = core.SIGNAL_BEFORE_USER_UPDATE.Send(u)
	if err != nil {
		return err
	}

	err = q.Querier.UpdateUser(ctx, providerName, data, accessToken, refreshToken, tokenType, expiresAt, isAdministrator, isActive, iD)
	if err != nil {
		return err
	}

	core.SIGNAL_AFTER_USER_UPDATE.Send(u)
	return nil
}

func (q *SignalsQuerier) DeleteUser(ctx context.Context, id uint64) error {
	err := core.SIGNAL_BEFORE_USER_DELETE.Send(uint64(id))
	if err != nil {
		return err
	}

	err = q.Querier.DeleteUser(ctx, id)
	if err != nil {
		return err
	}

	core.SIGNAL_AFTER_USER_DELETE.Send(uint64(id))
	return nil
}

func (q *SignalsQuerier) DeleteUsers(ctx context.Context, ids []uint64) error {
	err := core.SIGNAL_BEFORE_USER_DELETE.Send(ids)
	if err != nil {
		return err
	}

	err = q.Querier.DeleteUsers(ctx, ids)
	if err != nil {
		return err
	}

	core.SIGNAL_AFTER_USER_DELETE.Send(ids)
	return nil
}
