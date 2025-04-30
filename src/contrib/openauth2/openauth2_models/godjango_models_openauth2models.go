package openauth2models

import (
	"context"
	"database/sql"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/models"
)

func getQuerySet() (Querier, error) {
	var db = django.ConfigGet[*sql.DB](
		django.Global.Settings,
		django.APPVAR_DATABASE,
	)
	if db == nil {
		return nil, errs.Error("No database connection")
	}
	var backend, err = GetBackend(db.Driver())
	if err != nil {
		return nil, err
	}
	qs, err := backend.NewQuerySet(db)
	if err != nil {
		return nil, err
	}

	return &SignalsQuerier{qs}, nil
}

func (o *User) Save(ctx context.Context) error {
	if o.ID == 0 {
		return o.Insert(ctx)
	}
	return o.Update(ctx)
}

func (o *User) Insert(ctx context.Context) error {
	qs, err := getQuerySet()
	if err != nil {
		return err
	}
	id, err := qs.CreateUser(
		ctx,
		o.UniqueIdentifier,
		o.ProviderName,
		o.Data,
		o.AccessToken,
		o.RefreshToken,
		o.TokenType,
		o.ExpiresAt,
		o.IsAdministrator,
		o.IsActive,
	)
	if err != nil {
		return err
	}
	o.ID = uint64(id)
	return nil
}

func (o *User) Update(ctx context.Context) error {
	qs, err := getQuerySet()
	if err != nil {
		return err
	}
	return qs.UpdateUser(
		ctx,
		o.ProviderName,
		o.Data,
		o.AccessToken,
		o.RefreshToken,
		o.TokenType,
		o.ExpiresAt,
		o.IsAdministrator,
		o.IsActive,
		o.ID,
	)
}

func (o *User) Delete(ctx context.Context) error {
	var qs, err = getQuerySet()
	if err != nil {
		return err
	}
	return qs.DeleteUser(ctx, o.ID)
}

func (o *User) Reload(ctx context.Context) error {
	var qs, err = getQuerySet()
	if err != nil {
		return err
	}
	user, err := qs.RetrieveUserByID(ctx, o.ID)
	if err != nil {
		return err
	}
	*o = *user
	return nil
}

var (
	_ models.Saver   = &User{}
	_ models.Deleter = &User{}
	_ attrs.Definer  = &User{}
)
