package handlers

import (
	"context"
	"errors"
	"sync"

	auditlogs "github.com/Nigel2392/go-django/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/contrib/shop/carts"
	"github.com/Nigel2392/go-django/contrib/shop/products"
	"github.com/Nigel2392/go-django/contrib/shop/shared/signals"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
)

var registerAuditLogs sync.Once

// RegisterAuditLogs connects the shop's signal listeners and audit definitions.
func RegisterAuditLogs() {
	registerAuditLogs.Do(func() {
		ctx := context.Background()
		auditlogs.RegisterDefinition("shop:product:add", productAddedDefinition{})
		auditlogs.RegisterDefinition("shop:product:edit", productEditedDefinition{})
		products.Signals.Created.Listen(ctx, ProductsSignalsCreated)
		products.Signals.Updated.Listen(ctx, ProductsSignalsUpdated)

		auditlogs.RegisterDefinition("shop:carts:create", cartCreatedDefinition{})
		auditlogs.RegisterDefinition("shop:carts:clear", cartClearedDefinition{})
		auditlogs.RegisterDefinition("shop:carts:items:add", cartItemAddedDefinition{})
		auditlogs.RegisterDefinition("shop:carts:items:remove", cartItemRemovedDefinition{})
		carts.Signals.Created.Listen(ctx, CartsSignalsCreated)
		carts.Signals.Cleared.Listen(ctx, CartsSignalsCleared)
		carts.Signals.Item.Add.Listen(ctx, CartsSignalsItemAdd)
		carts.Signals.Item.Remove.Listen(ctx, CartsSignalsItemRemoved)

		registerCouponAudits()
		registerOrderAudits()
		registerPaymentAudits()
	})
}

type auditEvent struct {
	object attrs.Definer
	level  logger.LogLevel
	data   map[string]any
}

func listenAudit[T any](signal signals.Signal[T], kind string, definition auditlogs.Definition, event func(T) auditEvent) {
	auditlogs.RegisterDefinition(kind, definition)
	signal.Listen(context.Background(), func(ctx context.Context, _ signals.Signal[T], value T) error {
		entry := event(value)
		return logAudit(ctx, kind, entry.level, entry.object, entry.data)
	})
}

func logAudit(ctx context.Context, kind string, level logger.LogLevel, object attrs.Definer, data map[string]any) error {
	if attrs.IsZero(object) {
		object = nil
	}
	_, err := auditlogs.Log(ctx, kind, level, object, data)
	if errors.Is(err, auditlogs.ErrLogsNotReady) {
		return nil
	}
	return err
}
