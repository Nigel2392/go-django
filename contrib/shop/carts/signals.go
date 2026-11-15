package carts

import (
	"net/http"

	"github.com/Nigel2392/go-django/contrib/shop/util/signals"
	"github.com/Nigel2392/mux/middleware/authentication"
)

type CartAPISignal struct {
	Cart    *Cart
	Request *http.Request
	User    authentication.User
}

type CartItemAPISignal struct {
	CartAPISignal
	CartItem *CartItem
}

type cartItemSignals struct {
	Add    signals.Signal[CartItemAPISignal]
	Remove signals.Signal[CartItemAPISignal]
}

type cartSignals struct {
	Created signals.Signal[CartAPISignal]
	Cleared signals.Signal[CartAPISignal]
	Item    cartItemSignals
}

var Signals = cartSignals{
	Created: signals.Pool.Get[CartAPISignal]("shop.carts.Cart.Created"),
	Cleared: signals.Pool.Get[CartAPISignal]("shop.carts.Cart.Cleared"),
	Item: cartItemSignals{
		Add:    signals.Pool.Get[CartItemAPISignal]("shop.carts.CartItem.Add"),
		Remove: signals.Pool.Get[CartItemAPISignal]("shop.carts.CartItem.Remove"),
	},
}
