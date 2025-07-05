package auth

import (
	"context"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
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

type User struct {
	models.Model    `table:"users" json:"-"`
	ID              uint64           `json:"id" attrs:"primary;readonly"`
	CreatedAt       drivers.DateTime `json:"created_at" attrs:"readonly"`
	UpdatedAt       drivers.DateTime `json:"updated_at" attrs:"readonly"`
	Email           *drivers.Email   `json:"email"`
	Username        string           `json:"username"`
	Password        Password         `json:"password"`
	FirstName       string           `json:"first_name"`
	LastName        string           `json:"last_name"`
	IsAdministrator bool             `json:"is_administrator" attrs:"blank"`
	IsActive        bool             `json:"is_active" attrs:"blank;default=true"`
	IsLoggedIn      bool             `json:"is_logged_in"`

	//	Permissions *queries.RelM2M[*Permission, *UserPermission] `json:"-"`
}

func (u *User) String() string {
	return u.Username
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
		//	fields.NewManyToManyField[*Permission](
		//		u, "Permissions", &fields.FieldConfig{
		//			ScanTo: &u.Permissions,
		//			Rel: attrs.Relate(
		//				&Permission{}, "",
		//				&attrs.ThroughModel{
		//					This:   &UserPermission{},
		//					Source: "UserID",
		//					Target: "PermissionID",
		//				},
		//			),
		//		},
		//	),
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
