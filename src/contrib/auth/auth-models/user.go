package models

import (
	"context"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/fields"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	django_models "github.com/Nigel2392/go-django/src/models"
)

var (
	_ django_models.ContextSaver = (*User)(nil)
	_ queries.ActsBeforeSave     = (*User)(nil)
	_ queries.ActsBeforeCreate   = (*User)(nil)
)

type Password string

type UserPermission struct {
	UserID       drivers.Uint `json:"user_id" attrs:"primary;readonly"`
	PermissionID drivers.Uint `json:"permission_id" attrs:"primary;readonly"`
}

func (up *UserPermission) FieldDefs() attrs.Definitions {
	return attrs.Define(up, []any{
		attrs.Unbound("UserID", &attrs.FieldConfig{
			ReadOnly: true,
			Column:   "user_id",
		}),
		attrs.Unbound("PermissionID", &attrs.FieldConfig{
			ReadOnly: true,
			Column:   "permission_id",
		}),
	})
}

func (up *UserPermission) UniqueTogether() [][]string {
	return [][]string{
		{"UserID", "PermissionID"},
	}
}

type Permission struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (p *Permission) FieldDefs() attrs.Definitions {
	var fields = []attrs.Field{
		attrs.NewField(p, "ID", &attrs.FieldConfig{
			ReadOnly: true,
			Primary:  true,
		}),
		attrs.NewField(p, "Name", &attrs.FieldConfig{
			Label:    "Permission Name",
			HelpText: "Name of the permission. This is the name that will be displayed in the UI.",
		}),
		attrs.NewField(p, "Description", &attrs.FieldConfig{
			Blank:    true,
			Label:    "Description",
			HelpText: "Description of the permission. This is the description that will be displayed in the UI.",
		}),
	}
	return attrs.Define(p, fields...)
}

type User struct {
	models.Model    `table:"users" json:"-"`
	ID              drivers.Uint     `json:"id" attrs:"primary;readonly"`
	CreatedAt       drivers.DateTime `json:"created_at" attrs:"readonly"`
	UpdatedAt       drivers.DateTime `json:"updated_at" attrs:"readonly"`
	Email           *drivers.Email   `json:"email"`
	Username        drivers.String   `json:"username"`
	Password        Password         `json:"password"`
	FirstName       drivers.String   `json:"first_name"`
	LastName        drivers.String   `json:"last_name"`
	IsAdministrator bool             `json:"is_administrator" attrs:"blank"`
	IsActive        bool             `json:"is_active" attrs:"blank;default=true"`
	IsLoggedIn      bool             `json:"is_logged_in"`

	Permissions *queries.RelM2M[*Permission, *UserPermission] `json:"-"`
}

func (u *User) String() string {
	return string(u.Username)
}

func (u *User) BeforeCreate(ctx context.Context) error {
	u.CreatedAt = drivers.CurrentDateTime()
	return nil
}

func (u *User) BeforeSave(ctx context.Context) error {
	u.UpdatedAt = drivers.CurrentDateTime()
	return nil
}

func (u *User) Fields() []any {
	return []any{
		attrs.Unbound("ID", &attrs.FieldConfig{
			Primary:  true,
			ReadOnly: true,
			Column:   "id",
		}),
		attrs.Unbound("Email", &attrs.FieldConfig{
			Column:    "email",
			MaxLength: 255,
		}),
		attrs.Unbound("Username", &attrs.FieldConfig{
			Column:    "username",
			MaxLength: 16,
		}),
		attrs.Unbound("FirstName", &attrs.FieldConfig{
			Column:    "first_name",
			MaxLength: 75,
		}),
		attrs.Unbound("LastName", &attrs.FieldConfig{
			Column:    "last_name",
			MaxLength: 75,
		}),
		attrs.Unbound("Password", &attrs.FieldConfig{
			Column:    "password",
			MaxLength: 255,
		}),
		attrs.Unbound("IsAdministrator", &attrs.FieldConfig{
			Column: "is_administrator",
		}),
		attrs.Unbound("IsActive", &attrs.FieldConfig{
			Column: "is_active",
		}),
		attrs.Unbound("CreatedAt", &attrs.FieldConfig{
			Column: "created_at",
		}),
		attrs.Unbound("UpdatedAt", &attrs.FieldConfig{
			Column: "updated_at",
		}),
		fields.NewManyToManyField[*Permission](
			u, "Permissions", &fields.FieldConfig{
				ScanTo: &u.Permissions,
				Rel: attrs.Relate(
					&Permission{}, "",
					&attrs.ThroughModel{
						This:   &UserPermission{},
						Source: "UserID",
						Target: "PermissionID",
					},
				),
			},
		),
	}
}

func (u *User) FieldDefs() attrs.Definitions {
	return u.Model.Define(u, u.Fields)
}

func (u *User) IsAuthenticated() bool {
	return u.IsLoggedIn && u.IsActive
}

func (u *User) IsAdmin() bool {
	return u.IsAdministrator && u.IsActive
}
