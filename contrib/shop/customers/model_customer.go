package customers

import (
	"context"
	"time"

	"github.com/Nigel2392/go-django/contrib/auth/users"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/fields"
)

type Customer struct {
	ID        drivers.ULID
	Email     *drivers.Email
	FirstName string
	LastName  string
	User      users.User
	FirstSeen time.Time
	Orders    *queries.RelRevFK[attrs.Definer]
}

func (m *Customer) BeforeSave(ctx context.Context) error {
	if m.ID.IsZero() {
		m.ID = drivers.NewULID()
	}
	return nil
}

func (m *Customer) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*Customer, drivers.ULID] {
			return fattrs.PtrFieldConfig[*Customer, drivers.ULID]{
				Config: attrs.FieldConfig{
					Primary:  true,
					ReadOnly: true,
				},
			}
		}),
		fattrs.Field(m, "Email", &m.Email, func() fattrs.PtrFieldConfig[*Customer, *drivers.Email] {
			return fattrs.PtrFieldConfig[*Customer, *drivers.Email]{
				Config: attrs.FieldConfig{
					Column:    "email",
					MaxLength: 255,
					MinLength: 3,
					FormField: fields.EmailField,
					Label:     trans.S("Email"),
					HelpText:  trans.S("The email address of the user."),
				},
			}
		}),
		fattrs.Field(m, "FirstName", &m.FirstName, func() fattrs.PtrFieldConfig[*Customer, string] {
			return fattrs.PtrFieldConfig[*Customer, string]{
				Config: attrs.FieldConfig{
					Column:    "first_name",
					MaxLength: 75,
					Label:     trans.S("First Name"),
					HelpText:  trans.S("The first name of the user."),
				},
			}
		}),
		fattrs.Field(m, "LastName", &m.LastName, func() fattrs.PtrFieldConfig[*Customer, string] {
			return fattrs.PtrFieldConfig[*Customer, string]{
				Config: attrs.FieldConfig{
					Column:    "last_name",
					MaxLength: 75,
					Label:     trans.S("Last Name"),
					HelpText:  trans.S("The last name of the user."),
				},
			}
		}),
		fattrs.Field(m, "User", &m.User, func() fattrs.PtrFieldConfig[*Customer, users.User] {
			return fattrs.PtrFieldConfig[*Customer, users.User]{
				Config: attrs.FieldConfig{
					Column:   "user_id",
					ReadOnly: true,
					Null:     true,
					RelForeignKey: attrs.RelatedDeferred(
						attrs.RelManyToOne,
						users.MODEL_KEY,
						"", nil,
					),
				},
			}
		}),
		fattrs.Field(m, "FirstSeen", &m.FirstSeen, func() fattrs.PtrFieldConfig[*Customer, time.Time] {
			return fattrs.PtrFieldConfig[*Customer, time.Time]{
				Config: attrs.FieldConfig{
					Attributes: map[string]any{
						attrs.AttrAutoNowAddKey: true,
					},
					ReadOnly: true,
					Null:     false,
					Blank:    false,
				},
			}
		}),
		//	fattrs.Field(m, "Orders", &m.Orders, func() fattrs.PtrFieldConfig[*Customer, *queries.RelRevFK[attrs.Definer]] {
		//		return fattrs.PtrFieldConfig[*Customer, *queries.RelRevFK[attrs.Definer]]{}
		//	}),
	)
}
