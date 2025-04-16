package openauth2models

import (
	"database/sql"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/models"

	"context"
)

const (
	insertOauth2UserQuery = `INSERT INTO oauth2_users (
        unique_identifier, provider_name, data, access_token, refresh_token, token_type, expires_at, is_administrator, is_active
    ) VALUES (
        ?, ?, ?, ?, ?, ?, ?, ?, ?
    )`

	updateOauth2UserQuery = `UPDATE oauth2_users SET
        unique_identifier = ?, provider_name = ?, data = ?, access_token = ?, refresh_token = ?, token_type = ?, expires_at = ?, is_administrator = ?, is_active = ?
        WHERE id = ?`

	deleteOauth2UserQuery = `DELETE FROM oauth2_users WHERE id = ?`
)

func (o *User) Save(ctx context.Context) error {
	if o.ID == 0 {
		return o.Insert(ctx)
	}
	return o.Update(ctx)
}

func (o *User) Insert(ctx context.Context) error {
	var db = django.ConfigGet[*sql.DB](
		django.Global.Settings,
		django.APPVAR_DATABASE,
	)
	var result, err = db.ExecContext(
		ctx, insertOauth2UserQuery,
		o.ID, o.UniqueIdentifier, o.ProviderName, o.Data, o.AccessToken, o.RefreshToken, o.TokenType, o.ExpiresAt, o.CreatedAt, o.UpdatedAt, o.IsAdministrator, o.IsActive,
	)
	if err != nil {
		return err
	}

	dbId, err := result.LastInsertId()
	if err != nil {
		return err
	}

	o.ID = uint64(dbId)
	return nil
}

func (o *User) Update(ctx context.Context) error {
	var db = django.ConfigGet[*sql.DB](
		django.Global.Settings,
		django.APPVAR_DATABASE,
	)
	_, err := db.ExecContext(
		ctx, updateOauth2UserQuery,
		o.ID, o.UniqueIdentifier, o.ProviderName, o.Data, o.AccessToken, o.RefreshToken, o.TokenType, o.ExpiresAt, o.CreatedAt, o.UpdatedAt, o.IsAdministrator, o.IsActive,
		o.ID,
	)
	return err
}

func (o *User) Delete(ctx context.Context) error {
	var db = django.ConfigGet[*sql.DB](
		django.Global.Settings,
		django.APPVAR_DATABASE,
	)
	_, err := db.ExecContext(ctx, deleteOauth2UserQuery, o.ID)
	return err
}

var (
	_ models.Saver   = &User{}
	_ models.Updater = &User{}
	_ models.Deleter = &User{}
	_ attrs.Definer  = &User{}
)
