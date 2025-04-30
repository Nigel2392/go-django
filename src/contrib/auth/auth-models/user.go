package models

import (
	"context"
	"time"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/models"
)

var (
	_ models.Saver   = (*User)(nil)
	_ models.Deleter = (*User)(nil)
)

type Password string

type User struct {
	ID              uint64        `json:"id" attrs:"primary;readonly"`
	CreatedAt       time.Time     `json:"created_at" attrs:"readonly"`
	UpdatedAt       time.Time     `json:"updated_at" attrs:"readonly"`
	Email           *models.Email `json:"email"`
	Username        string        `json:"username"`
	Password        Password      `json:"password"`
	FirstName       string        `json:"first_name"`
	LastName        string        `json:"last_name"`
	IsAdministrator bool          `json:"is_administrator" attrs:"blank;default=true"`
	IsActive        bool          `json:"is_active" attrs:"blank;default=true"`
	IsLoggedIn      bool          `json:"is_logged_in"`
}

func (u *User) String() string {
	return u.Username
}

func (u *User) FieldDefs() attrs.Definitions {
	return attrs.AutoDefinitions(u,
		"ID",
		"Email",
		"Username",
		"FirstName",
		"LastName",
		"Password",
		"IsAdministrator",
		"IsActive",
		"CreatedAt",
		"UpdatedAt",
	)
}

//
//func (u *User) SetPassword(password string) error {
//	return SetPassword(u, password)
//}

func (u *User) Save(ctx context.Context) error {
	if u.ID == 0 {
		var id, err = CreateUser(
			ctx, u,
		)
		if err != nil {
			return err
		}
		u.ID = uint64(id)
	}
	return u.Update(ctx)
}

func (u *User) Update(ctx context.Context) error {
	return UpdateUser(
		ctx, u,
	)
}

func (u *User) Delete(ctx context.Context) error {
	return DeleteUser(ctx, u)
}

func (u *User) Reload(ctx context.Context) error {
	row, err := queries.RetrieveByID(ctx, u.ID)
	if err != nil {
		return err
	}

	*u = *row
	return nil
}

func (u *User) IsAuthenticated() bool {
	return u.IsLoggedIn && u.IsActive
}

func (u *User) IsAdmin() bool {
	return u.IsAdministrator && u.IsActive
}
