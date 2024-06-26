package models

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"net/mail"
	"time"

	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/models"
)

var (
	_ models.Saver    = (*User)(nil)
	_ models.Updater  = (*User)(nil)
	_ models.Deleter  = (*User)(nil)
	_ models.Reloader = (*User)(nil)
)

type Password string

var _ (sql.Scanner) = (*Email)(nil)
var _ (driver.Valuer) = (*Email)(nil)

type Email mail.Address

func (e Email) String() string {
	return e.Address
}

func (e *Email) ScanAttribute(src interface{}) error {
	return e.Scan(src)
}

func (e *Email) Scan(src interface{}) error {
	switch v := src.(type) {
	case string:
		a, err := mail.ParseAddress(v)
		if err != nil {
			return err
		}
		*e = Email(*a)
		return nil
	case []byte:
		a, err := mail.ParseAddress(string(v))
		if err != nil {
			return err
		}
		*e = Email(*a)
		return nil
	default:
		return errors.New("invalid email type")
	}
}

func (e Email) Value() (driver.Value, error) {
	var addr = e.Address
	return addr, nil
}

type User struct {
	ID              uint64    `json:"id" attrs:"primary;readonly"`
	CreatedAt       time.Time `json:"created_at" attrs:"readonly"`
	UpdatedAt       time.Time `json:"updated_at" attrs:"readonly"`
	Email           *Email    `json:"email"`
	Username        string    `json:"username"`
	Password        Password  `json:"password"`
	FirstName       string    `json:"first_name"`
	LastName        string    `json:"last_name"`
	IsAdministrator bool      `json:"is_administrator" attrs:"blank;default=true"`
	IsActive        bool      `json:"is_active" attrs:"blank;default=true"`
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
			u.Email.Address,
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
		u.Email.Address,
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
