package handlers

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/shop/core/shared/signals"
	"github.com/Nigel2392/go-django/contrib/shop/models/carts"
	"github.com/Nigel2392/go-django/src/core/logger"
)

func CartsSignalsCreated(ctx context.Context, s signals.Signal[carts.CartAPISignal], ca carts.CartAPISignal) error {
	return logAudit(ctx, "shop:carts:create", logger.INF, ca.Cart, map[string]any{})
}

func CartsSignalsCleared(ctx context.Context, s signals.Signal[carts.CartAPISignal], ca carts.CartAPISignal) error {
	return logAudit(ctx, "shop:carts:clear", logger.INF, ca.Cart, map[string]any{})
}

func CartsSignalsItemAdd(ctx context.Context, s signals.Signal[carts.CartItemAPISignal], ca carts.CartItemAPISignal) error {
	return logAudit(ctx, "shop:carts:items:add", logger.INF, ca.CartItem, map[string]any{
		"cart_id": ca.Cart.ID,
	})
}

func CartsSignalsItemRemoved(ctx context.Context, s signals.Signal[carts.CartItemAPISignal], ca carts.CartItemAPISignal) error {
	return logAudit(ctx, "shop:carts:items:remove", logger.INF, ca.CartItem, map[string]any{
		"cart_id": ca.Cart.ID,
	})
}
