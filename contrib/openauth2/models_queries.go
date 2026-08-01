package openauth2

import (
	"context"

	queries "github.com/Nigel2392/go-django/queries/src"
)

func CreateUser(ctx context.Context, user *User) (*User, error) {
	return queries.GetQuerySetWithContext(ctx, &User{}).Create(user)
}

func GetUserByID(ctx context.Context, id uint64) (*User, error) {
	var row, err = queries.GetQuerySetWithContext(ctx, &User{}).
		Select("*").
		Filter("ID", id).
		Get()
	if err != nil {
		return nil, err
	}
	return row.Object, nil
}

func GetUserByIdentifier(ctx context.Context, provider, identifier string) (*User, error) {
	var row, err = queries.GetQuerySetWithContext(ctx, &User{}).
		Select("*").
		Filter(map[string]interface{}{
			"ProviderName":     provider,
			"UniqueIdentifier": identifier,
		}).
		Get()
	if err != nil {
		return nil, err
	}
	return row.Object, nil
}
