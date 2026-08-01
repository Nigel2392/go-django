package models

import (
	"context"
	"time"

	"github.com/Nigel2392/go-django/contrib/auth/users"
	"github.com/Nigel2392/go-django/contrib/session"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
)

var _ queries.ActsBeforeCreate = (*Cart)(nil)

type Cart struct {
	ID        drivers.ULID
	User      users.User
	Session   *session.Session
	CartItems *queries.RelRevFK[*CartItem]
	CreatedAt time.Time
}

func (m *Cart) BeforeCreate(ctx context.Context) error {
	m.ID = drivers.NewULID()
	return nil
}

func (m *Cart) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*Cart, drivers.ULID] {
			return fattrs.PtrFieldConfig[*Cart, drivers.ULID]{
				Config: attrs.FieldConfig{
					Primary: true,
				},
			}
		}),
		fattrs.Field(m, "User", &m.User, func() fattrs.PtrFieldConfig[*Cart, users.User] {
			return fattrs.PtrFieldConfig[*Cart, users.User]{
				Config: attrs.FieldConfig{
					Column: "user_id",
					Null:   true,
					RelForeignKey: attrs.RelatedDeferred(
						attrs.RelManyToOne,
						users.MODEL_KEY,
						"", nil,
					),
				},
			}
		}),
		fattrs.Field(m, "Session", &m.Session, func() fattrs.PtrFieldConfig[*Cart, *session.Session] {
			return fattrs.PtrFieldConfig[*Cart, *session.Session]{
				Config: attrs.FieldConfig{
					Column:        "session_id",
					Null:          true,
					RelForeignKey: attrs.Relate(&session.Session{}, "", nil),
				},
			}
		}),
		fattrs.Field(m, "CartItems", &m.CartItems, func() fattrs.PtrFieldConfig[*Cart, *queries.RelRevFK[*CartItem]] {
			return fattrs.PtrFieldConfig[*Cart, *queries.RelRevFK[*CartItem]]{}
		}),
		fattrs.Field(m, "CreatedAt", &m.CreatedAt, func() fattrs.PtrFieldConfig[*Cart, time.Time] {
			return fattrs.PtrFieldConfig[*Cart, time.Time]{}
		}),
	)
}
