package carts

import (
	"encoding/json"
	"net/http"

	"github.com/Nigel2392/go-django/contrib/auth/users"
	"github.com/Nigel2392/go-django/contrib/shop/models"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-signals"
	"github.com/Nigel2392/mux/middleware/authentication"
	"github.com/Nigel2392/mux/middleware/sessions"
)

type CartAPISignal struct {
	Cart    *models.Cart
	Request *http.Request
	User    authentication.User
}

type sigs struct {
	pool      signals.SignalPool[CartAPISignal]
	Created   signals.Signal[CartAPISignal]
	Increment signals.Signal[CartAPISignal]
	Decrement signals.Signal[CartAPISignal]
	AddItem   signals.Signal[CartAPISignal]
	DelItem   signals.Signal[CartAPISignal]
	Cleared   signals.Signal[CartAPISignal]
}

type CartAPI struct {
	Signals sigs
}

func NewCartAPI() CartAPI {
	pool := signals.NewPool[CartAPISignal]()
	return CartAPI{
		Signals: sigs{
			pool:      pool,
			Created:   pool.Get("CartApi.Cart.Created"),
			Increment: pool.Get("CartApi.CartItem.Increment"),
			Decrement: pool.Get("CartApi.CartItem.Decrement"),
			AddItem:   pool.Get("CartApi.CartItem.Add"),
			DelItem:   pool.Get("CartApi.CartItem.Delete"),
			Cleared:   pool.Get("CartApi.Cart.Cleared"),
		},
	}
}

type CartResponse struct {
	Cart  *models.Cart       `json:"cart,omitempty"`
	Items []*models.CartItem `json:"items,omitempty"`
}

func (c CartAPI) cartQuery(r *http.Request, u authentication.User) *queries.QuerySet[*models.Cart] {
	var session = sessions.Retrieve(r)
	var q expr.ClauseExpression = expr.Q("Session", session.Token())
	if u.IsAuthenticated() {
		q = q.Or(expr.Q("User", u))
	}

	q = q.And(expr.Q("Active", true))

	return queries.
		GetQuerySetWithContext(r.Context(), &models.Cart{}).
		Preload(queries.NoJoins("CartItems")).
		Filter(q).
		OrderBy("CreatedAt")
}

func (c CartAPI) GetOrCreateCart(w http.ResponseWriter, r *http.Request) {
	u := users.UserFromRequest(r)
	q := c.cartQuery(r, u)
	cart, created, err := q.GetOrCreate(&models.Cart{
		Active:  true,
		User:    u.(users.User),
		Session: sessions.Retrieve(r).Token(),
	})
	if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"failed to get or create the cart: %v", err,
		)
	}

	if created {
		c.Signals.Created.Send(r.Context(), CartAPISignal{
			Cart:    cart,
			Request: r,
			User:    u,
		})
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
			"failed to encode the cart: %v", err,
		)
	}
}

func (c CartAPI) AddCartItem(w http.ResponseWriter, r *http.Request) {

}

func (c CartAPI) RemoveCartItem(w http.ResponseWriter, r *http.Request) {

}

func (c CartAPI) IncrementCartItem(w http.ResponseWriter, r *http.Request) {

}

func (c CartAPI) DecrementCartItem(w http.ResponseWriter, r *http.Request) {

}

func (c CartAPI) ClearCart(w http.ResponseWriter, r *http.Request) {
	u := users.UserFromRequest(r)
	q := c.cartQuery(r, u)
	_, err := q.Select("Active").BulkUpdate(
		expr.As("Active", expr.Value(false)),
	)
	if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"failed to encode the cart: %v", err,
		)
	}

	c.Signals.Cleared.Send(r.Context(), CartAPISignal{
		Cart:    nil,
		Request: r,
		User:    u,
	})

	c.GetOrCreateCart(w, r)
}
