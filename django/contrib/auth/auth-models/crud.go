package models

import "context"

func CreateUser(ctx context.Context, u *User) (int64, error) {
	SIGNAL_BEFORE_USER_CREATE.Send(u)

	var id, err = queries.CreateUser(ctx, u.Email.Address, u.Username, string(u.Password), u.FirstName, u.LastName, u.IsAdministrator, u.IsActive)
	if err != nil {
		return 0, err
	}

	u.ID = uint64(id)

	SIGNAL_AFTER_USER_CREATE.Send(u)
	return id, nil
}

func DeleteUser(ctx context.Context, u *User) error {
	SIGNAL_BEFORE_USER_DELETE.Send(u.ID)

	err := queries.DeleteUser(ctx, u.ID)
	if err != nil {
		return err
	}

	SIGNAL_AFTER_USER_DELETE.Send(u.ID)
	return nil

}

func UpdateUser(ctx context.Context, u *User) error {
	SIGNAL_BEFORE_USER_UPDATE.Send(u)

	err := queries.UpdateUser(ctx, u.Email.Address, u.Username, string(u.Password), u.FirstName, u.LastName, u.IsAdministrator, u.IsActive, u.ID)
	if err != nil {
		return err
	}

	SIGNAL_AFTER_USER_UPDATE.Send(u)
	return nil
}
