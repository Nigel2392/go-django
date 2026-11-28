package carts

import (
	"context"
	stderrors "errors"
	"net/mail"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/auth/users"
	"github.com/Nigel2392/go-django/contrib/shop/models/coupons"
	"github.com/Nigel2392/go-django/contrib/shop/models/customers"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
)

var (
	_ queries.ActsBeforeCreate = (*Cart)(nil)

	ErrInvalidCustomerEmail = stderrors.New("a valid customer email address is required")
)

type Cart struct {
	ID          drivers.ULID                           `json:"id"`
	User        users.User                             `json:"-"`
	Customer    *customers.Customer                    `json:"customer,omitempty"`
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
				Config: attrs.Config{
					Primary:  true,
					ReadOnly: true,
				},
			}
		}),
		fattrs.Field(m, "User", &m.User, func() fattrs.PtrFieldConfig[*Cart, users.User] {
			return fattrs.PtrFieldConfig[*Cart, users.User]{
				Config: attrs.Config{
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
		fattrs.Field(m, "Customer", &m.Customer, func() fattrs.PtrFieldConfig[*Cart, *customers.Customer] {
			return fattrs.PtrFieldConfig[*Cart, *customers.Customer]{
				Config: attrs.Config{
					Column:        "customer_id",
					Null:          true,
					Blank:         true,
					RelForeignKey: attrs.Relate(&customers.Customer{}, "", nil),
				},
			}
		}),
		fattrs.Field(m, "Session", &m.Session, func() fattrs.PtrFieldConfig[*Cart, string] {
			return fattrs.PtrFieldConfig[*Cart, string]{
				Config: attrs.Config{
					Column:   "session_id",
					ReadOnly: true,
					Null:     true,
				},
			}
		}),
		fattrs.Field(m, "Active", &m.Active, func() fattrs.PtrFieldConfig[*Cart, bool] {
			return fattrs.PtrFieldConfig[*Cart, bool]{
				Config: attrs.Config{
					Blank: true,
				},
			}
		}),
		fattrs.Field(m, "CreatedAt", &m.CreatedAt, func() fattrs.PtrFieldConfig[*Cart, time.Time] {
			return fattrs.PtrFieldConfig[*Cart, time.Time]{
				Config: attrs.Config{
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
				Config: attrs.Config{
					Attributes: map[string]interface{}{
						migrator.AttrUseInDBKey: false,
					},
				},
			}
		}),
	)
}

type CustomerDetails struct {
	Email     string
	FirstName string
	LastName  string
	User      users.User
}

func SetCustomer(ctx context.Context, cart *Cart, details CustomerDetails) (*customers.Customer, error) {
	if cart == nil || cart.ID.IsZero() {
		return nil, errors.ValueError.Wrap("cart is required")
	}

	rawEmail := strings.TrimSpace(details.Email)
	address, err := mail.ParseAddress(rawEmail)
	if err != nil || address == nil || strings.TrimSpace(address.Address) == "" || len(address.Address) < 3 {
		return nil, ErrInvalidCustomerEmail
	}

	firstName := strings.TrimSpace(details.FirstName)
	lastName := strings.TrimSpace(details.LastName)

	ctx, tx, err := queries.StartTransaction(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var customerID drivers.ULID
	if cart.Customer != nil && !cart.Customer.ID.IsZero() {
		customerID = cart.Customer.ID
	} else {
		cartRow, err := queries.GetQuerySet(&Cart{}).
			WithContext(ctx).
			SelectRelated("Customer").
			Filter("ID", cart.ID).
			Get()
		if err != nil {
			return nil, err
		}
		if cartRow.Object.Customer != nil && !cartRow.Object.Customer.ID.IsZero() {
			customerID = cartRow.Object.Customer.ID
		}
	}

	var authUser users.User
	if details.User != nil && !attrs.IsZero(details.User) && details.User.IsAuthenticated() {
		authUser = details.User
	}

	var customer *customers.Customer
	if !customerID.IsZero() {
		existingRow, err := queries.GetQuerySet(&customers.Customer{}).
			WithContext(ctx).
			Filter("ID", customerID).
			Get()
		switch {
		case err == nil:
			customer = existingRow.Object
			customer.Email = (*drivers.Email)(address)
			customer.FirstName = firstName
			customer.LastName = lastName
			selectFields := []any{"Email", "FirstName", "LastName"}
			if authUser != nil && attrs.IsZero(customer.User) {
				customer.User = authUser
				selectFields = append(selectFields, "User")
			}
			if _, err := queries.GetQuerySet(&customers.Customer{}).
				WithContext(ctx).
				Filter("ID", customer.ID).
				Select(selectFields...).
				Update(customer); err != nil {
				return nil, err
			}
		case !errors.Is(err, errors.NoRows):
			return nil, err
		}
	}

	if customer == nil {
		customer, err = queries.GetQuerySet(&customers.Customer{}).
			WithContext(ctx).
			Create(&customers.Customer{
				Email:     (*drivers.Email)(address),
				FirstName: firstName,
				LastName:  lastName,
				User:      authUser,
			})
		if err != nil {
			return nil, err
		}
	}

	cart.Customer = customer
	if _, err := queries.GetQuerySet(&Cart{}).
		WithContext(ctx).
		Filter("ID", cart.ID).
		Select("Customer").
		Update(cart); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return customer, nil
}
