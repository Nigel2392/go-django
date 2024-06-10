package models

import (
	"context"
	"time"

	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/models"
)

type UserRow struct {
	User                  User   `json:"user"`
	GroupID               uint64 `json:"group_id"`
	GroupName             string `json:"group_name"`
	GroupDescription      string `json:"group_description"`
	PermissionID          uint64 `json:"permission_id"`
	PermissionName        string `json:"permission_name"`
	PermissionDescription string `json:"permission_description"`
}

var (
	_ models.Saver    = (*User)(nil)
	_ models.Updater  = (*User)(nil)
	_ models.Deleter  = (*User)(nil)
	_ models.Reloader = (*User)(nil)
)

type Password string

type User struct {
	ID              uint64    `json:"id" attrs:"primary;readonly"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	Email           string    `json:"email"`
	Username        string    `json:"username"`
	Password        Password  `json:"password"`
	FirstName       string    `json:"first_name"`
	LastName        string    `json:"last_name"`
	IsAdministrator bool      `json:"is_administrator"`
	IsActive        bool      `json:"is_active"`
	IsLoggedIn      bool      `json:"is_logged_in"`
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
		return queries.CreateUser(
			ctx,
			u.Email,
			u.Username,
			string(u.Password),
			u.FirstName,
			u.LastName,
			u.IsAdministrator,
			u.IsActive,
		)
	}
	return u.Update(ctx)
}

func (u *User) Update(ctx context.Context) error {
	return queries.UpdateUser(
		ctx,
		u.Email,
		u.Username,
		string(u.Password),
		u.FirstName,
		u.LastName,
		u.IsAdministrator,
		u.IsActive,
		u.ID,
	)
}

func (u *User) Delete(ctx context.Context) error {
	return queries.DeleteUser(ctx, u.ID)
}

func (u *User) Reload(ctx context.Context) error {
	row, err := queries.GetUserById(ctx, u.ID)
	if err != nil {
		return err
	}

	*u = row[0].User
	return nil
}

func (u *User) IsAuthenticated() bool {
	return u.IsLoggedIn && u.IsActive
}

func (u *User) IsAdmin() bool {
	return u.IsAdministrator && u.IsActive
}
