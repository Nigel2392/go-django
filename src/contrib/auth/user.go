package auth

import (
	"context"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/contrib/auth/users"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/forms/fields"
	django_models "github.com/Nigel2392/go-django/src/models"
)

var (
	_ django_models.ContextSaver    = (*User)(nil)
	_ queries.UniqueTogetherDefiner = (*User)(nil)
	_ migrator.IndexDefiner         = (*User)(nil)
	_ queries.ActsBeforeSave        = (*User)(nil)
	_ queries.ActsBeforeCreate      = (*User)(nil)
)

type UserQuerySet struct {
	*queries.WrappedQuerySet[*User, *UserQuerySet, *queries.QuerySet[*User]]
}

func GetUserQuerySet() *UserQuerySet {
	userQuerySet := &UserQuerySet{}
	userQuerySet.WrappedQuerySet = queries.WrapQuerySet(
		queries.GetQuerySet[*User](&User{}),
		userQuerySet,
	)
	return userQuerySet
}

type User struct {
	models.Model `table:"auth_users" json:"-"`
	users.Base

	ID        uint64           `json:"id" attrs:"primary;readonly"`
	Email     *drivers.Email   `json:"email"`
	Username  string           `json:"username"`
	Password  *Password        `json:"password"`
	FirstName string           `json:"first_name"`
	LastName  string           `json:"last_name"`
	CreatedAt drivers.DateTime `json:"created_at" attrs:"readonly"`
	UpdatedAt drivers.DateTime `json:"updated_at" attrs:"readonly"`
}

func (u *User) String() string {
	return u.Username
}

func (u *User) UniqueTogether() [][]string {
	return [][]string{
		{"Email"},
		{"Username"},
	}
}

func (u *User) DatabaseIndexes(obj attrs.Definer) []migrator.Index {
	return []migrator.Index{
		{
			Identifier: "auth_users_email_idx",
			Type:       "btree",
			Fields:     []string{"Email"},
			Unique:     true,
			// Comment:    "Index for email uniqueness",
		},
		{
			Identifier: "auth_users_username_idx",
			Type:       "btree",
			Fields:     []string{"Username"},
			Unique:     true,
			// Comment:    "Index for username uniqueness",
		},
	}
}

func (u *User) SetPassword(password string) *User {
	u.Password = NewPassword(password)
	return u
}

func (u *User) BeforeCreate(ctx context.Context) error {
	u.CreatedAt = drivers.CurrentDateTime()
	return nil
}

func (u *User) BeforeSave(ctx context.Context) error {

	if u.Password.IsZero() {
		return errors.ValueError.Wrapf("password cannot be empty: %+v", u.Password)
	}

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
			MinLength: 3,
			FormField: fields.EmailField,
		}),
		attrs.Unbound("Username", &attrs.FieldConfig{
			Column:    "username",
			MaxLength: 16,
			MinLength: 3,
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
		attrs.Unbound("CreatedAt", &attrs.FieldConfig{
			Column: "created_at",
		}),
		attrs.Unbound("UpdatedAt", &attrs.FieldConfig{
			Column: "updated_at",
		}),
		u.Base.Fields(u),
	}
}

func (u *User) FieldDefs() attrs.Definitions {
	return u.Model.Define(u, u.Fields)
}
