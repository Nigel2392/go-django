package public

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Nigel2392/go-django/contrib/auth/users"
	"github.com/Nigel2392/go-django/contrib/shop/api"
	"github.com/Nigel2392/go-django/contrib/shop/carts"
	"github.com/Nigel2392/go-django/contrib/shop/products"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/mux/middleware/authentication"
	"github.com/Nigel2392/mux/middleware/sessions"
)

type CartItemRequest struct {
	ProductID string `json:"product_id"`
	Amount    int    `json:"amount"`
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

func (c CartAPI) cartQuery(ctx context.Context, r *http.Request, u authentication.User) *queries.QuerySet[*carts.Cart] {
	var session = sessions.Retrieve(r)
	var q expr.ClauseExpression = expr.Q("Session", session.Token())
	if u.IsAuthenticated() {
		q = q.Or(expr.Q("User", u))
	}

	q = q.And(expr.Q("Active", true))

	return queries.
		GetQuerySetWithContext(ctx, &carts.Cart{}).
		Filter(q).
		OrderBy("CreatedAt")
}

func (c CartAPI) GetOrCreateCart(w http.ResponseWriter, r *http.Request) {
	u := authentication.UserFromContext(r.Context())
	q := c.cartQuery(r.Context(), r, u).
		Preload(queries.Preload{
			Path:    "CartItems",
			NoJoins: true,
			QuerySet: queries.
				GetQuerySet[attrs.Definer](&carts.CartItem{}).
				Select(
					"*",
					"Product.ID",
					"Product.Title",
					"Product.Image",
					"Product.Price",
					"Product.TaxRate",
					"Product.DiscountPrice",
				),
		})

	cart, created, err := q.GetOrCreate(&carts.Cart{
		Active:  true,
		User:    u.(users.User),
		Session: sessions.Retrieve(r).Token(),
	})
	if err != nil {
		api.RespondError(w,
			http.StatusInternalServerError,
			"failed to get or create the cart: %v", err,
		)
		return
	}

	if created {
		err = carts.Signals.Created.Send(r.Context(), carts.CartAPISignal{
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

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	err = enc.Encode(CartResponse{
		Cart:  cart,
		Items: cart.CartItems.AsList(),
	})
	if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"failed to encode the cart response: %v", err,
		)
	}
}

func (c CartAPI) AddCartItem(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	request := CartItemRequest{}
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&request)
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
	u := authentication.UserFromContext(r.Context())
	q := c.cartQuery(ctx, r, u)
	cart := &carts.Cart{
		Active:  true,
		Session: sessions.Retrieve(r).Token(),
	}

	if u.IsAuthenticated() {
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
	enc := json.NewEncoder(w)
	err = enc.Encode(CartItemResponse{
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
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&request)
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
	q := c.cartQuery(ctx, r, u)
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
	enc := json.NewEncoder(w)
	err = enc.Encode(CartItemResponse{
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
	u := authentication.UserFromContext(r.Context())
	q := c.cartQuery(r.Context(), r, u)
	cartRow, err := q.Get()
	switch {
	case errors.Is(err, errors.NoRows):

	case err != nil:
		api.RespondError(w,
			http.StatusInternalServerError,
			"failed to update the cart: %v", err,
		)
		return

	default:
		_, err := q.Select("Active").BulkUpdate(
			expr.As("Active", expr.Value(false)),
		)
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

	c.GetOrCreateCart(w, r)
}
