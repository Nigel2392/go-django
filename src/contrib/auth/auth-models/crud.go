package models

import (
	"context"

	django_signals "github.com/Nigel2392/go-django/src/signals"
)

func CreateUser(ctx context.Context, u *User) (int64, error) {
	django_signals.SIGNAL_BEFORE_USER_CREATE.Send(u)

	var id, err = queries.CreateUser(ctx, u.Email.Address, u.Username, string(u.Password), u.FirstName, u.LastName, u.IsAdministrator, u.IsActive)
	if err != nil {
		return 0, err
	}

	u.ID = uint64(id)

	django_signals.SIGNAL_AFTER_USER_CREATE.Send(u)
	return id, nil
}

func DeleteUser(ctx context.Context, u *User) error {
	django_signals.SIGNAL_BEFORE_USER_DELETE.Send(u.ID)

	err := queries.DeleteUser(ctx, u.ID)
	if err != nil {
		return err
	}

	django_signals.SIGNAL_AFTER_USER_DELETE.Send(u.ID)
	return nil

}

func UpdateUser(ctx context.Context, u *User) error {
	django_signals.SIGNAL_BEFORE_USER_UPDATE.Send(u)

	err := queries.UpdateUser(ctx, u.Email.Address, u.Username, string(u.Password), u.FirstName, u.LastName, u.IsAdministrator, u.IsActive, u.ID)
	if err != nil {
		return err
	}

	django_signals.SIGNAL_AFTER_USER_UPDATE.Send(u)
	return nil
}
