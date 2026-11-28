package handlers

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/shop/core/shared/signals"
	"github.com/Nigel2392/go-django/contrib/shop/models/products"
	"github.com/Nigel2392/go-django/src/core/attrs/changes"
	"github.com/Nigel2392/go-django/src/core/logger"
)

func ProductsSignalsCreated(ctx context.Context, s signals.Signal[products.ProductCreatedSignal], psd products.ProductCreatedSignal) error {
	return logAudit(ctx, "shop:product:add", logger.INF, psd.Product, map[string]any{
		"title": psd.Product.Title,
	})
}

func ProductsSignalsUpdated(ctx context.Context, s signals.Signal[products.ProductUpdatedSignal], psd products.ProductUpdatedSignal) error {
	changes, err := changes.Changes(ctx, psd.Product.Old, psd.Product.New)
	if err != nil {
		return err
	}

	return logAudit(ctx, "shop:product:edit", logger.INF, psd.Product.New, map[string]any{
		"title":   psd.Product.New.Title,
		"changes": changes,
	})
}
