package handlers

import (
	"context"

	auditlogs "github.com/Nigel2392/go-django/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/contrib/shop/carts"
	"github.com/Nigel2392/go-django/contrib/shop/shared/signals"
	"github.com/Nigel2392/go-django/src/core/logger"
)

func CartsSignalsCreated(ctx context.Context, s signals.Signal[carts.CartAPISignal], ca carts.CartAPISignal) error {
	_, err := auditlogs.Log(ctx, "shop:carts:create", logger.INF, ca.Cart, map[string]any{})
	return err
}

func CartsSignalsCleared(ctx context.Context, s signals.Signal[carts.CartAPISignal], ca carts.CartAPISignal) error {
	_, err := auditlogs.Log(ctx, "shop:carts:clear", logger.INF, ca.Cart, map[string]any{})
	return err
}

func CartsSignalsItemAdd(ctx context.Context, s signals.Signal[carts.CartItemAPISignal], ca carts.CartItemAPISignal) error {
	_, err := auditlogs.Log(ctx, "shop:carts:items:add", logger.INF, ca.CartItem, map[string]any{
		"cart_id": ca.Cart.ID,
	})
	return err
}

func CartsSignalsItemRemoved(ctx context.Context, s signals.Signal[carts.CartItemAPISignal], ca carts.CartItemAPISignal) error {
	_, err := auditlogs.Log(ctx, "shop:carts:items:remove", logger.INF, ca.CartItem, map[string]any{
		"cart_id": ca.Cart.ID,
	})
	return err
}
