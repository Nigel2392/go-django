package carts

import (
	"context"
	"time"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/auth/users"
	"github.com/Nigel2392/go-django/contrib/shop/coupons"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
)

var (
	_ queries.ActsBeforeCreate = (*Cart)(nil)
)

type Cart struct {
	ID          drivers.ULID                           `json:"id"`
	User        users.User                             `json:"-"`
	Session     string                                 `json:"-"`
	CreatedAt   time.Time                              `json:"created_at"`
	Active      bool                                   `json:"active"`
	CartItems   *queries.RelRevFK[*CartItem]           `json:"-"`
	Redemptions *queries.RelRevFK[*coupons.Redemption] `json:"-"`
}

func (m *Cart) BeforeCreate(ctx context.Context) error {
	m.ID = drivers.NewULID()
	return nil
}

func (m *Cart) EditPanels(ctx context.Context) []admin.Panel {
	return []admin.Panel{
		admin.FieldPanel("ID"),
		&admin.ModelFormPanel[*CartItem, modelforms.ModelForm[*CartItem]]{
			TargetType: &CartItem{},
			FieldName:  "CartItems",
			Classname:  "collapsible",
			// SubClassname: "collapsed",
			MinNum:         0,
			DisallowAdd:    true,
			DisallowRemove: true,
			Panels: []admin.Panel{
				admin.FieldPanel("Product"),
				admin.FieldPanel("Amount"),
			},
		},
	}
}

func (m *Cart) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*Cart, drivers.ULID] {
			return fattrs.PtrFieldConfig[*Cart, drivers.ULID]{
				Config: attrs.FieldConfig{
					Primary:  true,
					ReadOnly: true,
				},
			}
		}),
		fattrs.Field(m, "User", &m.User, func() fattrs.PtrFieldConfig[*Cart, users.User] {
			return fattrs.PtrFieldConfig[*Cart, users.User]{
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
		fattrs.Field(m, "Session", &m.Session, func() fattrs.PtrFieldConfig[*Cart, string] {
			return fattrs.PtrFieldConfig[*Cart, string]{
				Config: attrs.FieldConfig{
					Column:   "session_id",
					ReadOnly: true,
					Null:     true,
				},
			}
		}),
		fattrs.Field(m, "Active", &m.Active, func() fattrs.PtrFieldConfig[*Cart, bool] {
			return fattrs.PtrFieldConfig[*Cart, bool]{
				Config: attrs.FieldConfig{
					Blank: true,
				},
			}
		}),
		fattrs.Field(m, "CreatedAt", &m.CreatedAt, func() fattrs.PtrFieldConfig[*Cart, time.Time] {
			return fattrs.PtrFieldConfig[*Cart, time.Time]{
				Config: attrs.FieldConfig{
					ReadOnly:   true,
					AutoNowAdd: true,
					Null:       false,
					Blank:      false,
				},
			}
		}),
		fattrs.Field(m, "CartItems", &m.CartItems, func() fattrs.PtrFieldConfig[*Cart, *queries.RelRevFK[*CartItem]] {
			return fattrs.PtrFieldConfig[*Cart, *queries.RelRevFK[*CartItem]]{}
		}),
		fattrs.Field(m, "Redemptions", &m.Redemptions, func() fattrs.PtrFieldConfig[*Cart, *queries.RelRevFK[*coupons.Redemption]] {
			return fattrs.PtrFieldConfig[*Cart, *queries.RelRevFK[*coupons.Redemption]]{
				Flags: fattrs.F_IS_REVERSE_FLD,
				Config: attrs.FieldConfig{
					Attributes: map[string]interface{}{
						migrator.AttrUseInDBKey: false,
					},
				},
			}
		}),
	)
}
