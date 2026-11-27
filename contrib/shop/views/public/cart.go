package public

import (
	"context"
	stderrors "errors"
	"net/http"

	"github.com/Nigel2392/go-django/contrib/auth/users"
	"github.com/Nigel2392/go-django/contrib/shop/carts"
	"github.com/Nigel2392/go-django/contrib/shop/coupons"
	"github.com/Nigel2392/go-django/contrib/shop/products"
	"github.com/Nigel2392/go-django/contrib/shop/shared/api"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/mux/v2/middleware/authentication"
	"github.com/Nigel2392/mux/v2/middleware/sessions"
)

type CartItemRequest struct {
	ProductID string `json:"product_id"`
	Amount    int    `json:"amount"`
}

type CartCustomerRequest struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type CartResponse struct {
	Cart  *carts.Cart       `json:"cart"`
	Items []*carts.CartItem `json:"items"`
}

type CartItemResponse struct {
	Cart    *carts.Cart       `json:"cart"`
	Item    *carts.CartItem   `json:"item"`
	Product *products.Product `json:"product"`
}

type CartAPI struct{}

func cartQuery(ctx context.Context, r *http.Request, preloads bool) *carts.CartQuery {
	qs := carts.Query().
		WithContext(ctx).
		CartForRequest(r)

	if preloads {
		qs = qs.PreloadProducts(
			"ID",
			"Title",
			"Image",
			"Price",
			"TaxRate",
			"DiscountPrice",
		)
	}

	return qs
}

func (c CartAPI) getOrCreateCart(r *http.Request, preloads bool) (*carts.CartRow, bool, error) {
	if err := ensureCartSession(r); err != nil {
		return nil, false, err
	}
	u := authentication.UserFromContext(r.Context())
	q := cartQuery(r.Context(), r, preloads)

	newCart := &carts.Cart{
		Active:  true,
		Session: sessions.Retrieve(r).Token(),
	}
	if u != nil && u.IsAuthenticated() {
		newCart.User = u.(users.User)
	}
	cart, created, err := q.CalcOneOrCreate(newCart)
	if err != nil {
		return nil, created, errors.Wrap(err, "getOrCreateCart")
	}

	if created {
		err = carts.Signals.Created.Send(r.Context(), carts.CartAPISignal{
			Cart:    cart.Object,
			Request: r,
			User:    u,
		})
		if err != nil {
			return cart, created, errors.Wrap(err, "error processing cart created signal")
		}
	}

	return cart, created, nil
}

func ensureCartSession(r *http.Request) error {
	s := sessions.Retrieve(r)
	if s == nil {
		return errors.NilPointer.Wrap("cart requires session middleware")
	}
	if s.Token() == "" {
		return s.RenewToken()
	}
	return nil
}

func (c CartAPI) GetOrCreateCart(w http.ResponseWriter, r *http.Request) {

	cart, _, err := c.getOrCreateCart(r, true)
	if err != nil {
		api.RespondError(w,
			http.StatusInternalServerError,
			"failed to encode the cart response: %v", err,
		)
		return
	}

	err = api.Respond(w, cart)
	if err != nil {
		api.RespondError(w,
			http.StatusInternalServerError,
			"failed to encode the cart response: %v", err,
		)
	}
}

func (c CartAPI) SetCustomer(w http.ResponseWriter, r *http.Request) {
	request := CartCustomerRequest{}
	if err := api.Decode(r, &request); err != nil {
		api.RespondError(w,
			http.StatusBadRequest,
			"failed to read or decode request body: %v", err,
		)
		return
	}

	if err := ensureCartSession(r); err != nil {
		api.RespondError(w, http.StatusServiceUnavailable, "cart session unavailable")
		return
	}

	ctx, tx, err := queries.StartTransaction(r.Context())
	if err != nil {
		api.RespondError(w,
			http.StatusInternalServerError,
			"failed to start transaction: %v", err,
		)
		return
	}
	defer tx.Rollback(ctx)

	cart, _, err := c.getOrCreateCart(r.WithContext(ctx), true)
	if err != nil {
		api.RespondError(w,
			http.StatusInternalServerError,
			"failed to get or create the cart: %v", err,
		)
		return
	}

	var authUser users.User
	if u, ok := authentication.Retrieve(r).(users.User); ok && !attrs.IsZero(u) && u.IsAuthenticated() {
		authUser = u
	} else if u, ok := authentication.UserFromContext(ctx).(users.User); ok && !attrs.IsZero(u) && u.IsAuthenticated() {
		authUser = u
	}

	customer, err := carts.SetCustomer(ctx, cart.Object, carts.CustomerDetails{
		Email:     request.Email,
		FirstName: request.FirstName,
		LastName:  request.LastName,
		User:      authUser,
	})
	switch {
	case stderrors.Is(err, carts.ErrInvalidCustomerEmail):
		api.RespondError(w,
			http.StatusUnprocessableEntity,
			"a valid customer email address is required",
		)
		return
	case err != nil:
		api.RespondError(w,
			http.StatusInternalServerError,
			"failed to save customer details: %v", err,
		)
		return
	}

	cart.Object.Customer = customer

	if err := tx.Commit(ctx); err != nil {
		api.RespondError(w,
			http.StatusInternalServerError,
			"failed to commit the changes to the database: %v", err,
		)
		return
	}

	if err := api.Respond(w, cart); err != nil {
		api.RespondError(w,
			http.StatusInternalServerError,
			"failed to encode the cart response: %v", err,
		)
	}
}

func (c CartAPI) AddCartItem(w http.ResponseWriter, r *http.Request) {
	request := CartItemRequest{}
	err := api.Decode(r, &request)
	if err != nil {
		api.RespondError(w,
			http.StatusBadRequest,
			"failed to read or decode request body: %v", err,
		)
		return
	}

	if request.Amount <= 0 {
		api.RespondError(w,
			http.StatusBadRequest,
			"amount cannot be less than 0: %v", request.Amount,
		)
		return
	}

	ctx, tx, err := queries.StartTransaction(r.Context())
	if err != nil {
		api.RespondError(w,
			http.StatusInternalServerError,
			"failed to start transaction: %v", err,
		)
		return
	}

	defer tx.Rollback(ctx)

	// retrieve the requested product
	productRow, err := queries.
		GetQuerySet(&products.Product{}).
		WithContext(ctx).
		Select(
			"ID",
			"Title",
			"Image",
			"Price",
			"TaxRate",
			"DiscountPrice",
		).
		Filter("ID", request.ProductID).
		Get()
	if err != nil {
		if errors.Is(err, errors.NoRows) {
			api.RespondError(w,
				http.StatusNotFound,
				"failed to retrieve product to add to cart: %v", err,
			)
			return
		}

		api.RespondError(w,
			http.StatusInternalServerError,
			"failed to retrieve product to add to cart: %v", err,
		)
		return
	}

	// retrieve or create the users' cart
	if err := ensureCartSession(r); err != nil {
		api.RespondError(w, http.StatusServiceUnavailable, "cart session unavailable")
		return
	}
	u := authentication.UserFromContext(r.Context())
	q := cartQuery(ctx, r, false)
	cart := &carts.Cart{
		Active:  true,
		Session: sessions.Retrieve(r).Token(),
	}

	if u != nil && u.IsAuthenticated() {
		cart.User = u.(users.User)
	}

	cart, cartAdded, err := q.GetOrCreate(cart)
	if err != nil {
		api.RespondError(w,
			http.StatusInternalServerError,
			"failed to get or create the cart: %v", err,
		)
		return
	}

	var (
		item    *carts.CartItem
		created bool
	)

	if !cartAdded {
		// try to retrieve or create the existing cart item
		item, created, err = queries.
			GetQuerySet(&carts.CartItem{}).
			WithContext(ctx).
			Filter(
				expr.Q("Cart", cart),
				expr.Q("Product", productRow.Object),
			).
			GetOrCreate(&carts.CartItem{
				Cart:    cart,
				Product: productRow.Object,
				Amount:  uint64(request.Amount),
			})
		if err != nil {
			api.RespondError(w,
				http.StatusInternalServerError,
				"failed to get or create the cart item: %v", err,
			)
			return
		}

		// if a cart item wasn't newly added, update the existing cart item
		// directly in the database.
		if !created {

			// this wont get saved, it is just for the user
			item.Amount += uint64(request.Amount)

			// update atomically in the DB
			_, err = queries.
				GetQuerySet(&carts.CartItem{}).
				WithContext(ctx).
				Filter(
					expr.Q("Cart", cart),
					expr.Q("Product", productRow.Object),
				).
				BulkUpdate(
					expr.As("Amount", expr.Logical("Amount").ADD(request.Amount)),
				)
			if err != nil {
				api.RespondError(w,
					http.StatusInternalServerError,
					"failed to update the existing cart item: %v", err,
				)
				return
			}
		}

		goto returnCartResponse
	}

	// create a new product when the cart was newly added
	// saves a SELECT query, skipping GetOrCreate
	item, err = queries.
		GetQuerySet(&carts.CartItem{}).
		WithContext(ctx).
		Create(&carts.CartItem{
			Cart:    cart,
			Product: productRow.Object,
			Amount:  uint64(request.Amount),
		})
	if err != nil {
		api.RespondError(w,
			http.StatusInternalServerError,
			"failed to create a new cart item: %v", err,
		)
		return
	}

returnCartResponse:
	if cartAdded {
		err = carts.Signals.Created.Send(ctx, carts.CartAPISignal{
			Cart:    cart,
			Request: r,
			User:    u,
		})
		if err != nil {
			api.RespondError(w,
				http.StatusInternalServerError,
				"error processing cart created signal: %v", err,
			)
			return
		}
	}

	err = carts.Signals.Item.Add.Send(ctx, carts.CartItemAPISignal{
		Cart:     cart,
		Request:  r,
		User:     u,
		CartItem: item,
	})
	if err != nil {
		api.RespondError(w,
			http.StatusInternalServerError,
			"error processing cart item added signal: %v", err,
		)
		return
	}

	err = tx.Commit(ctx)
	if err != nil {
		api.RespondError(w,
			http.StatusInternalServerError,
			"failed to commit the changes to the database: %v", err,
		)
		return
	}

	// encode the response
	err = api.Respond(w, CartItemResponse{
		Cart:    cart,
		Item:    item,
		Product: productRow.Object,
	})
	if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"failed to encode the cart item response: %v", err,
		)
	}
}

func (c CartAPI) RemoveCartItem(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	request := CartItemRequest{}
	err := api.Decode(r, &request)
	if err != nil {
		api.RespondError(w,
			http.StatusBadRequest,
			"failed to read or decode request body: %v", err,
		)
		return
	}

	if request.Amount <= 0 {
		api.RespondError(w,
			http.StatusBadRequest,
			"amount cannot be less than 0: %v", request.Amount,
		)
		return
	}

	ctx, tx, err := queries.StartTransaction(r.Context())
	if err != nil {
		api.RespondError(w,
			http.StatusInternalServerError,
			"failed to start transaction: %v", err,
		)
		return
	}

	defer tx.Rollback(ctx)

	// retrieve the requested product
	productRow, err := queries.
		GetQuerySet(&products.Product{}).
		Select(
			"ID",
			"Title",
			"Image",
			"Price",
			"TaxRate",
			"DiscountPrice",
		).
		WithContext(ctx).
		Filter("ID", request.ProductID).
		Get()
	if err != nil {
		if errors.Is(err, errors.NoRows) {
			api.RespondError(w,
				http.StatusNotFound,
				"failed to retrieve product to remove from cart: %v", err,
			)
			return
		}

		api.RespondError(w,
			http.StatusInternalServerError,
			"failed to retrieve product to remove from cart: %v", err,
		)
		return
	}

	// retrieve or create the users' cart
	u := authentication.UserFromContext(r.Context())
	q := cartQuery(ctx, r, false)
	cart, err := q.Get()
	if err != nil {
		if errors.Is(err, errors.NoRows) {
			api.RespondError(w,
				http.StatusNotFound,
				"cart does not exist, cannot remove items: %v", err,
			)
			return
		}

		api.RespondError(w,
			http.StatusInternalServerError,
			"failed to retrieve the cart: %v", err,
		)
		return
	}

	// retrieve the specified cart item
	item, err := queries.
		GetQuerySet(&carts.CartItem{}).
		WithContext(ctx).
		Filter(
			expr.Q("Cart", cart.Object),
			expr.Q("Product", productRow.Object),
		).
		Get()
	if err != nil {
		if errors.Is(err, errors.NoRows) {
			api.RespondError(w,
				http.StatusNotFound,
				"cart item doesn't exist, cannot remove item: %v", err,
			)
			return
		}

		api.RespondError(w,
			http.StatusInternalServerError,
			"failed to retrieve cart item to remove: %v", err,
		)
		return
	}

	// if the amount to subtract is greater than the current amount
	// of requested items, it is safe to delete the entire cartitem instance
	if request.Amount >= int(item.Object.Amount) {
		// remove the cart item from the cart entirely
		_, err = queries.GetQuerySet(&carts.CartItem{}).
			WithContext(ctx).
			Filter("ID", item.Object.ID).
			Delete()
		if err != nil {
			api.RespondError(w,
				http.StatusInternalServerError,
				"failed to remove cart item: %v", err,
			)
			return
		}

		item.Object.Amount = 0
		goto returnCartResponse
	}

	// this wont get saved, it is just for the user
	item.Object.Amount -= uint64(request.Amount)

	// subtract the amount atomically in the DB
	_, err = queries.
		GetQuerySet(&carts.CartItem{}).
		WithContext(ctx).
		Filter("ID", item.Object.ID).
		BulkUpdate(
			expr.As("Amount", expr.Logical("Amount").SUB(request.Amount)),
		)
	if err != nil {
		api.RespondError(w,
			http.StatusInternalServerError,
			"failed to update the existing cart item: %v", err,
		)
		return
	}

returnCartResponse:
	err = carts.Signals.Item.Remove.Send(ctx, carts.CartItemAPISignal{
		Cart:     cart.Object,
		Request:  r,
		User:     u,
		CartItem: item.Object,
	})
	if err != nil {
		api.RespondError(w,
			http.StatusInternalServerError,
			"error processing cart item removed signal: %v", err,
		)
		return
	}

	err = tx.Commit(ctx)
	if err != nil {
		api.RespondError(w,
			http.StatusInternalServerError,
			"failed to commit the changes to the database: %v", err,
		)
		return
	}

	// encode the response
	err = api.Respond(w, CartItemResponse{
		Cart:    cart.Object,
		Item:    item.Object,
		Product: productRow.Object,
	})
	if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"failed to encode the cart item response: %v", err,
		)
	}
}

func (c CartAPI) ClearCart(w http.ResponseWriter, r *http.Request) {
	var tx drivers.Transaction
	var u = authentication.UserFromContext(r.Context())
	var q = cartQuery(r.Context(), r, true)
	var cartRow, err = q.Get()
	switch {
	case errors.Is(err, errors.NoRows):

	case err != nil:
		api.RespondError(w,
			http.StatusInternalServerError,
			"failed to update the cart: %v", err,
		)
		return

	default:

		// start a transaction to ensure everything is done atomically
		var ctx context.Context
		ctx, tx, err = queries.StartTransaction(r.Context())
		if err != nil {
			api.RespondError(w,
				http.StatusInternalServerError,
				"failed to start transaction: %v", err,
			)
			return
		}

		defer tx.Rollback(ctx)

		r = r.WithContext(ctx)

		// Close the active cart before releasing its coupon reservations.
		closed, err := q.
			WithContext(ctx).
			Filter("ID", cartRow.Object.ID).
			BulkUpdate(expr.As("Active", expr.Value(false)))
		if err != nil {
			api.RespondError(w,
				http.StatusInternalServerError,
				"failed to update the cart: %v", err,
			)
			return
		}

		if closed == 0 {
			break
		}

		_, err = queries.
			GetQuerySet(&coupons.Redemption{}).
			WithContext(ctx).
			Filter(
				expr.Q("Cart", cartRow.Object.ID),
				expr.Q("Order__isnull", true),
				expr.Q("Invalidated", false),
			).
			BulkUpdate(expr.As("Invalidated", expr.Value(true)))
		if err != nil {
			api.RespondError(w,
				http.StatusInternalServerError,
				"failed to update the cart: %v", err,
			)
			return
		}

		err = carts.Signals.Cleared.Send(r.Context(), carts.CartAPISignal{
			Cart:    cartRow.Object,
			Request: r,
			User:    u,
		})
		if err != nil {
			api.RespondError(w,
				http.StatusInternalServerError,
				"error processing cart cleared signal: %v", err,
			)
			return
		}
	}

	// create a new cart
	cart, _, err := c.getOrCreateCart(r, false)
	if err != nil {
		api.RespondError(w,
			http.StatusInternalServerError,
			"failed to create new cart after clearing: %v", err,
		)
		return
	}

	if tx != nil {
		err = tx.Commit(r.Context())
		if err != nil {
			api.RespondError(w,
				http.StatusInternalServerError,
				"failed to commit the transaction: %v", err,
			)
			return
		}
	}

	err = api.Respond(w, cart)
	if err != nil {
		api.RespondError(w,
			http.StatusInternalServerError,
			"failed to encode the cart response: %v", err,
		)
	}
}
