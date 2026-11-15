package handlers

import (
	"context"

	auditlogs "github.com/Nigel2392/go-django/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/contrib/shop/products"
	"github.com/Nigel2392/go-django/contrib/shop/util/signals"
	"github.com/Nigel2392/go-django/src/core/attrs/changes"
	"github.com/Nigel2392/go-django/src/core/logger"
)

func ProductsSignalsCreated(ctx context.Context, s signals.Signal[products.ProductCreatedSignal], psd products.ProductCreatedSignal) error {
	_, err := auditlogs.Log(ctx, "shop:product:add", logger.INF, psd.Product, map[string]any{
		"title": psd.Product.Title,
	})
	return err
}

func ProductsSignalsUpdated(ctx context.Context, s signals.Signal[products.ProductUpdatedSignal], psd products.ProductUpdatedSignal) error {
	changes, err := changes.Changes(ctx, psd.Product.Old, psd.Product.New)
	if err != nil {
		return err
	}

	_, err = auditlogs.Log(ctx, "shop:product:edit", logger.INF, psd.Product.New, map[string]any{
		"title":   psd.Product.New.Title,
		"changes": changes,
	})
	return err
}
