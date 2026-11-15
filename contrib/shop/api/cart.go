package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Nigel2392/go-django/contrib/auth/users"
	"github.com/Nigel2392/go-django/contrib/shop/carts"
	"github.com/Nigel2392/go-django/contrib/shop/products"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-signals"
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

type CartAPISignal struct {
	Cart    *carts.Cart
	Request *http.Request
	User    authentication.User
}

type CartItemSignal struct {
	CartAPISignal
	CartItem *carts.CartItem
}

type cartItemSignals struct {
	Add    signals.Signal[CartItemSignal]
	Remove signals.Signal[CartItemSignal]
}

type cartSignals struct {
	pool    *signals.Pool[any]
	Created signals.Signal[CartAPISignal]
	Cleared signals.Signal[CartAPISignal]
	Item    cartItemSignals
}

type CartAPI struct {
	Cart cartSignals
}

func NewCartAPI() CartAPI {
	pool := signals.NewGPool()
	return CartAPI{
		Cart: cartSignals{
			pool:    (*signals.Pool[any])(pool),
			Created: pool.Get[CartAPISignal]("CartApi.Cart.Created"),
			Cleared: pool.Get[CartAPISignal]("CartApi.Cart.Cleared"),
			Item: cartItemSignals{
				Add:    pool.Get[CartItemSignal]("CartApi.Cart.Item.Add"),
				Remove: pool.Get[CartItemSignal]("CartApi.Cart.Item.Remove"),
			},
		},
	}
}

func (c CartAPI) respondError(w http.ResponseWriter, statusCode int, message string, args ...any) {
	w.WriteHeader(statusCode)

	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}

	enc := json.NewEncoder(w)
	err := enc.Encode(ErrorResponse{
		Message: message,
	})
	if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"failed to encode the error response: %v", err,
		)
	}
}

func (c CartAPI) cartQuery(ctx context.Context, r *http.Request, u authentication.User) *queries.QuerySet[*carts.Cart] {
	var session = sessions.Retrieve(r)
	var q expr.ClauseExpression = expr.Q("Session", session.Token())
	if u.IsAuthenticated() {
		q = q.Or(expr.Q("User", u))
	}

	q = q.And(expr.Q("Active", true))

	return queries.
		GetQuerySetWithContext(ctx, &carts.Cart{}).
		Preload(queries.NoJoins("CartItems")).
		Filter(q).
		OrderBy("CreatedAt")
}

func (c CartAPI) GetOrCreateCart(w http.ResponseWriter, r *http.Request) {
	u := users.UserFromRequest(r)
	q := c.cartQuery(r.Context(), r, u)
	cart, created, err := q.GetOrCreate(&carts.Cart{
		Active:  true,
		User:    u.(users.User),
		Session: sessions.Retrieve(r).Token(),
	})
	if err != nil {
		c.respondError(w,
			http.StatusInternalServerError,
			"failed to get or create the cart: %v", err,
		)
		return
	}

	if created {
		err = c.Cart.Created.Send(r.Context(), CartAPISignal{
			Cart:    cart,
			Request: r,
			User:    u,
		})
		if err != nil {
			c.respondError(w,
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
		c.respondError(w,
			http.StatusBadRequest,
			"failed to read or decode request body: %v", err,
		)
		return
	}

	if request.Amount <= 0 {
		c.respondError(w,
			http.StatusBadRequest,
			"amount cannot be less than 0: %v", request.Amount,
		)
		return
	}

	ctx, tx, err := queries.StartTransaction(r.Context())
	if err != nil {
		c.respondError(w,
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
		Filter("ID", request.ProductID).
		Get()
	if err != nil {
		if errors.Is(err, errors.NoRows) {
			c.respondError(w,
				http.StatusNotFound,
				"failed to retrieve product to add to cart: %v", err,
			)
			return
		}

		c.respondError(w,
			http.StatusInternalServerError,
			"failed to retrieve product to add to cart: %v", err,
		)
		return
	}

	// retrieve or create the users' cart
	u := users.UserFromRequest(r)
	q := c.cartQuery(ctx, r, u)
	cart, cartAdded, err := q.GetOrCreate(&carts.Cart{
		Active:  true,
		User:    u.(users.User),
		Session: sessions.Retrieve(r).Token(),
	})
	if err != nil {
		c.respondError(w,
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
			c.respondError(w,
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
				c.respondError(w,
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
		c.respondError(w,
			http.StatusInternalServerError,
			"failed to create a new cart item: %v", err,
		)
		return
	}

returnCartResponse:
	if cartAdded {
		err = c.Cart.Created.Send(ctx, CartAPISignal{
			Cart:    cart,
			Request: r,
			User:    u,
		})
		if err != nil {
			c.respondError(w,
				http.StatusInternalServerError,
				"error processing cart created signal: %v", err,
			)
			return
		}
	}

	err = c.Cart.Item.Add.Send(ctx, CartItemSignal{
		Cart:     cart,
		Request:  r,
		User:     u,
		CartItem: item,
	})
	if err != nil {
		c.respondError(w,
			http.StatusInternalServerError,
			"error processing cart item added signal: %v", err,
		)
		return
	}

	err = tx.Commit(ctx)
	if err != nil {
		c.respondError(w,
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
		c.respondError(w,
			http.StatusBadRequest,
			"failed to read or decode request body: %v", err,
		)
		return
	}

	if request.Amount <= 0 {
		c.respondError(w,
			http.StatusBadRequest,
			"amount cannot be less than 0: %v", request.Amount,
		)
		return
	}

	ctx, tx, err := queries.StartTransaction(r.Context())
	if err != nil {
		c.respondError(w,
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
		Filter("ID", request.ProductID).
		Get()
	if err != nil {
		if errors.Is(err, errors.NoRows) {
			c.respondError(w,
				http.StatusNotFound,
				"failed to retrieve product to remove from cart: %v", err,
			)
			return
		}

		c.respondError(w,
			http.StatusInternalServerError,
			"failed to retrieve product to remove from cart: %v", err,
		)
		return
	}

	// retrieve or create the users' cart
	u := users.UserFromRequest(r)
	q := c.cartQuery(ctx, r, u)
	cart, err := q.Get()
	if err != nil {
		if errors.Is(err, errors.NoRows) {
			c.respondError(w,
				http.StatusNotFound,
				"cart does not exist, cannot remove items: %v", err,
			)
			return
		}

		c.respondError(w,
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
			c.respondError(w,
				http.StatusNotFound,
				"cart item doesn't exist, cannot remove item: %v", err,
			)
			return
		}

		c.respondError(w,
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
			c.respondError(w,
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
		c.respondError(w,
			http.StatusInternalServerError,
			"failed to update the existing cart item: %v", err,
		)
		return
	}

returnCartResponse:
	err = c.Cart.Item.Remove.Send(ctx, CartItemSignal{
		Cart:     cart.Object,
		Request:  r,
		User:     u,
		CartItem: item.Object,
	})
	if err != nil {
		c.respondError(w,
			http.StatusInternalServerError,
			"error processing cart item removed signal: %v", err,
		)
		return
	}

	err = tx.Commit(ctx)
	if err != nil {
		c.respondError(w,
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
	u := users.UserFromRequest(r)
	q := c.cartQuery(r.Context(), r, u)

	_, err := q.Select("Active").BulkUpdate(
		expr.As("Active", expr.Value(false)),
	)
	if err != nil {
		c.respondError(w,
			http.StatusInternalServerError,
			"failed to update the cart: %v", err,
		)
		return
	}

	err = c.Cart.Cleared.Send(r.Context(), CartAPISignal{
		Cart:    nil,
		Request: r,
		User:    u,
	})
	if err != nil {
		c.respondError(w,
			http.StatusInternalServerError,
			"error processing cart cleared signal: %v", err,
		)
		return
	}

	c.GetOrCreateCart(w, r)
}
