//go:build test

package orders_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Nigel2392/go-django/contrib/blocks"
	"github.com/Nigel2392/go-django/contrib/editor"
	"github.com/Nigel2392/go-django/contrib/shop/core/shared/signals"
	"github.com/Nigel2392/go-django/contrib/shop/models/carts"
	"github.com/Nigel2392/go-django/contrib/shop/models/categories"
	"github.com/Nigel2392/go-django/contrib/shop/models/coupons"
	"github.com/Nigel2392/go-django/contrib/shop/models/customers"
	"github.com/Nigel2392/go-django/contrib/shop/models/orders"
	"github.com/Nigel2392/go-django/contrib/shop/models/products"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	qerrors "github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	formfields "github.com/Nigel2392/go-django/src/forms/fields"
	gosignals "github.com/Nigel2392/go-signals"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func clearWorkflowTables(t *testing.T) {
	t.Helper()
	clearOrderAndCartTables(t)
	if _, err := queries.GetQuerySet(&coupons.CouponProductThrough{}).Delete(); err != nil {
		t.Fatalf("failed to clear CouponProductThrough table: %v", err)
	}
	if _, err := queries.GetQuerySet(&coupons.Coupon{}).Delete(); err != nil {
		t.Fatalf("failed to clear Coupon table: %v", err)
	}
	if _, err := orders.SuppliersQuery().Delete(); err != nil {
		t.Fatalf("failed to clear SupplierModel table: %v", err)
	}
}

func seedWorkflowProduct(t *testing.T, cat *categories.Category, supplierName, title, slug, sku, price string) *products.Product {
	t.Helper()
	spec := &testOrderSupplierProduct{
		Product: &products.Product{
			Image:         &mediafiles.SimpleStoredObject{Filepath: slug + ".png"},
			Title:         title,
			Slug:          slug,
			Category:      cat,
			Editor:        &editor.EditorJSBlockData{},
			Supplier:      products.SupplierValue(supplierName),
			Price:         decimal.RequireFromString(price),
			TaxRate:       decimal.RequireFromString("21.00"),
			DiscountPrice: decimal.Zero,
			Ranking:       1,
		},
		SKU: sku,
	}
	spec.Product.Instance = spec

	if _, err := queries.GetQuerySet(&testOrderSupplierProduct{}).Create(spec); err != nil {
		t.Fatalf("failed to create specific product %q: %v", title, err)
	}
	prod, err := queries.GetQuerySet(&products.Product{}).Create(spec.Product)
	if err != nil {
		t.Fatalf("failed to create base product %q: %v", title, err)
	}
	return prod
}

func seedWorkflowCouponRedemption(t *testing.T, order *orders.Order, customer *customers.Customer, code, discount string) (*coupons.Coupon, *coupons.Redemption) {
	t.Helper()
	coupon, err := queries.GetQuerySet(&coupons.Coupon{}).Create(&coupons.Coupon{
		Title:       "Workflow " + code,
		Code:        coupons.CouponCode(code),
		MaxDiscount: decimal.RequireFromString(discount),
	})
	require.NoError(t, err)

	redemption, err := queries.GetQuerySet(&coupons.Redemption{}).Create(&coupons.Redemption{
		Coupon:         coupon,
		Customer:       customer,
		Order:          order,
		DiscountAmount: decimal.RequireFromString(discount),
	})
	require.NoError(t, err)
	return coupon, redemption
}

func listenOrderAccepted(
	t *testing.T,
	fn func(context.Context, signals.Signal[orders.OrderAcceptedSignal], orders.OrderAcceptedSignal) error,
) {
	t.Helper()
	sigCtx, cancelSig := context.WithCancel(t.Context())
	recv, err := orders.Signals.Accepted.Listen(sigCtx, fn)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, orders.Signals.Accepted.Disconnect(context.Background(), recv))
		cancelSig()
	})
}

func listenOrderProcessed(
	t *testing.T,
	fn func(context.Context, signals.Signal[orders.OrderProcessedSignal], orders.OrderProcessedSignal) error,
) {
	t.Helper()
	sigCtx, cancelSig := context.WithCancel(t.Context())
	recv, err := orders.Signals.Processed.Listen(sigCtx, fn)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, orders.Signals.Processed.Disconnect(context.Background(), recv))
		cancelSig()
	})
}

func signalErrorContains(err error, target error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, target) {
		return true
	}
	if unwrapped := errors.Unwrap(err); unwrapped != nil && signalErrorContains(unwrapped, target) {
		return true
	}
	sigErr, ok := gosignals.SignalError(err)
	if !ok {
		return false
	}
	if sigErr.Reason != nil && sigErr.Reason != err && signalErrorContains(sigErr.Reason, target) {
		return true
	}
	for _, rel := range sigErr.Related {
		if signalErrorContains(rel, target) {
			return true
		}
	}
	return false
}

func requireSignalErrorIs(t *testing.T, err error, target error) {
	t.Helper()
	require.Error(t, err)
	if signalErrorContains(err, target) {
		return
	}
	t.Fatalf("expected error (or nested SignalError.Related) to wrap %v, got %v", target, err)
}

func TestSupplierWorkflow(t *testing.T) {
	setupOrderTestEnv(t)
	clearWorkflowTables(t)
	t.Cleanup(func() {
		clearWorkflowTables(t)
	})

	cat := seedTestCategory(t, "Workflow Digital", "workflow-digital")
	customer, err := queries.GetQuerySet(&customers.Customer{}).Create(&customers.Customer{
		Email:     drivers.MustParseEmail("workflow-buyer@example.com"),
		FirstName: "Alice",
		LastName:  "Tester",
	})
	require.NoError(t, err)

	alphaBlock := blocks.NewStructBlock()
	alphaBlock.AddField("pin", blocks.FormFieldBlock(formfields.CharField(formfields.Required(true)), ""))
	alphaBlock.AddField("serial", blocks.CharBlock())

	betaBlock := blocks.NewStructBlock()
	betaBlock.AddField("receipt", blocks.FormFieldBlock(formfields.CharField(formfields.Required(true)), ""))

	t.Run("Multi-item and multi-supplier happy path", func(t *testing.T) {
		clearWorkflowTables(t)

		var alphaCalls int
		var betaCalls int

		orders.RegisterSupplier(&orders.Supplier{
			Name: "wf_alpha",
			ProductModel: func(context.Context) products.ProductModel {
				return &testOrderSupplierProduct{}
			},
			OrderBlockType: func(context.Context, *orders.Order, *orders.OrderItem) (blocks.Block, error) {
				return alphaBlock, nil
			},
			OrderIsAccepted: func(_ context.Context, gotOrder *orders.Order, item *orders.OrderItem) (*blocks.StructBlockValue, error) {
				alphaCalls++
				require.NotNil(t, gotOrder.Customer, "Customer must be preloaded on Order passed to OrderAccepted")
				require.Equal(t, customer.ID, gotOrder.Customer.ID)
				require.Equal(t, customer.Email, gotOrder.Customer.Email)
				require.Equal(t, "Alice", gotOrder.Customer.FirstName)
				require.Equal(t, orders.OrderStateWorkflow, gotOrder.State)
				require.False(t, gotOrder.Completed)
				require.False(t, gotOrder.Processed)

				return blocks.NewBlockValue(alphaBlock, map[string]any{
					"pin":    fmt.Sprintf("PIN-%d", alphaCalls),
					"serial": fmt.Sprintf("SER-%s", item.ID),
				}), nil
			},
		})

		orders.RegisterSupplier(&orders.Supplier{
			Name: "wf_beta",
			ProductModel: func(context.Context) products.ProductModel {
				return &testOrderSupplierProduct{}
			},
			OrderBlockType: func(context.Context, *orders.Order, *orders.OrderItem) (blocks.Block, error) {
				return betaBlock, nil
			},
			OrderIsAccepted: func(_ context.Context, gotOrder *orders.Order, _ *orders.OrderItem) (*blocks.StructBlockValue, error) {
				betaCalls++
				require.NotNil(t, gotOrder.Customer, "Customer must be preloaded on Order passed to OrderAccepted")
				require.Equal(t, customer.ID, gotOrder.Customer.ID)
				require.Equal(t, orders.OrderStateWorkflow, gotOrder.State)
				require.False(t, gotOrder.Completed)

				// Return an unbound StructBlockValue to verify BindToBlock + validation.
				return &blocks.StructBlockValue{
					V: map[string]any{
						"receipt": "RCP-BETA-01",
					},
				}, nil
			},
		})

		require.NoError(t, orders.SuppliersQuery().Sync())

		prodAlpha1 := seedWorkflowProduct(t, cat, "wf_alpha", "Alpha Card 10", "wf-alpha-10", "SKU-WFA-10", "10.00")
		prodBeta := seedWorkflowProduct(t, cat, "wf_beta", "Beta Card 25", "wf-beta-25", "SKU-WFB-25", "25.00")
		prodAlpha2 := seedWorkflowProduct(t, cat, "wf_alpha", "Alpha Card 50", "wf-alpha-50", "SKU-WFA-50", "50.00")

		createReq := httptest.NewRequest(http.MethodPost, "/checkout", nil).WithContext(t.Context())
		order, createdItems, err := orders.CreateOrder(
			createReq,
			customer,
			nil,
			&carts.CartItem{Product: prodAlpha1, Amount: 2},
			&carts.CartItem{Product: prodBeta, Amount: 1},
			&carts.CartItem{Product: prodAlpha2, Amount: 1},
		)
		require.NoError(t, err)
		require.Len(t, createdItems, 3)
		_, err = queries.GetQuerySet(&orders.OrderItem{}).
			Filter("ID", createdItems[1].ID).
			Select("SupplierData").
			Update(&orders.OrderItem{SupplierData: blocks.NewBlockValue(betaBlock, map[string]any{
				"receipt": "STAGED",
			})})
		require.NoError(t, err)

		coupon, redemption := seedWorkflowCouponRedemption(t, order, customer, "WFSAVE10", "5.00")

		var signalSequence []string
		var acceptedSig orders.OrderAcceptedSignal
		var processedSig orders.OrderProcessedSignal
		var itemsVisibleInTx int64

		listenOrderAccepted(t, func(_ context.Context, _ signals.Signal[orders.OrderAcceptedSignal], sig orders.OrderAcceptedSignal) error {
			signalSequence = append(signalSequence, "accepted")
			acceptedSig = sig
			cnt, countErr := queries.GetQuerySet(&orders.OrderItem{}).
				WithContext(sig.Request.Context()).
				Filter("Order", sig.Order.ID).
				Count()
			if countErr != nil {
				return countErr
			}
			itemsVisibleInTx = cnt
			return nil
		})
		listenOrderProcessed(t, func(_ context.Context, _ signals.Signal[orders.OrderProcessedSignal], sig orders.OrderProcessedSignal) error {
			signalSequence = append(signalSequence, "processed")
			processedSig = sig
			return nil
		})

		completeReq := httptest.NewRequest(http.MethodPost, "/payments/webhook", nil).WithContext(t.Context())
		err = orders.Signals.Completed.Send(t.Context(), orders.OrderCompletedSignal{
			BaseSignal: signals.BaseSignal{Request: completeReq},
			Order:      order,
		})
		require.NoError(t, err)

		require.Equal(t, 2, alphaCalls)
		require.Equal(t, 1, betaCalls)
		require.Equal(t, []string{"accepted", "processed"}, signalSequence)
		require.Equal(t, int64(3), itemsVisibleInTx)

		// Verify caller order pointer was synchronized after commit.
		require.Equal(t, orders.OrderStateAccepted, order.State)
		require.True(t, order.Completed)
		require.True(t, order.Processed)
		require.NotNil(t, order.Customer)
		require.Equal(t, customer.ID, order.Customer.ID)

		// Verify Signals.Accepted and Signals.Processed payloads.
		require.NotNil(t, acceptedSig.Order)
		require.Equal(t, orders.OrderStateAccepted, acceptedSig.Order.State)
		require.True(t, acceptedSig.Order.Completed)
		require.True(t, acceptedSig.Order.Processed)
		require.NotNil(t, acceptedSig.Order.Customer)
		require.Equal(t, customer.ID, acceptedSig.Order.Customer.ID)
		require.Len(t, acceptedSig.Lines, 3)
		for _, line := range acceptedSig.Lines {
			require.True(t, line.SupplierData.HasData())
		}

		require.NotNil(t, processedSig.Order)
		require.Equal(t, order.ID, processedSig.Order.ID)
		require.Equal(t, orders.OrderStateAccepted, processedSig.Order.State)
		require.True(t, processedSig.Order.Completed)
		require.True(t, processedSig.Order.Processed)

		// Reload Order from DB.
		dbOrderRow, err := orders.Query().Select("*", "Customer.*").Filter("ID", order.ID).Get()
		require.NoError(t, err)
		require.Equal(t, orders.OrderStateAccepted, dbOrderRow.Object.State)
		require.True(t, dbOrderRow.Object.Completed)
		require.True(t, dbOrderRow.Object.Processed)

		// Reload OrderItems from DB and verify SupplierData is bound to OrderBlock.
		dbItemRows, err := queries.GetQuerySet(&orders.OrderItem{}).
			Filter("Order", order.ID).
			OrderBy("ID").
			All()
		require.NoError(t, err)
		require.Len(t, dbItemRows, 3)

		var seenAlphaPins []string
		for _, row := range dbItemRows {
			item := row.Object
			require.True(t, item.SupplierData.HasData())
			switch item.Supplier {
			case "wf_alpha":
				require.Same(t, alphaBlock, item.SupplierData.Block())
				pinField, ok := item.SupplierData.V["pin"].(blocks.BoundBlockValue)
				require.True(t, ok, "pin field has type %T", item.SupplierData.V["pin"])
				serialField, ok := item.SupplierData.V["serial"].(blocks.BoundBlockValue)
				require.True(t, ok, "serial field has type %T", item.SupplierData.V["serial"])
				require.Equal(t, fmt.Sprintf("SER-%s", item.ID), serialField.Data())
				seenAlphaPins = append(seenAlphaPins, fmt.Sprint(pinField.Data()))
			case "wf_beta":
				require.Same(t, betaBlock, item.SupplierData.Block())
				receiptField, ok := item.SupplierData.V["receipt"].(blocks.BoundBlockValue)
				require.True(t, ok, "receipt field has type %T", item.SupplierData.V["receipt"])
				require.Equal(t, "RCP-BETA-01", receiptField.Data())
			default:
				t.Fatalf("unexpected supplier %q", item.Supplier)
			}
		}
		require.ElementsMatch(t, []string{"PIN-1", "PIN-2"}, seenAlphaPins)

		// Verify coupon redemptions were reconciled.
		dbRedemption, err := queries.GetQuerySet(&coupons.Redemption{}).Filter("ID", redemption.ID).Get()
		require.NoError(t, err)
		require.True(t, dbRedemption.Object.Counted)
		require.False(t, dbRedemption.Object.Invalidated)

		dbCoupon, err := queries.GetQuerySet(&coupons.Coupon{}).Filter("ID", coupon.ID).Get()
		require.NoError(t, err)
		require.Equal(t, uint64(1), dbCoupon.Object.UsageCount)
	})

	t.Run("Supplier returning nil, nil still succeeds and marks order processed", func(t *testing.T) {
		clearWorkflowTables(t)

		var calls int
		orders.RegisterSupplier(&orders.Supplier{
			Name: "wf_nil_supplier",
			ProductModel: func(context.Context) products.ProductModel {
				return &testOrderSupplierProduct{}
			},
			OrderIsAccepted: func(context.Context, *orders.Order, *orders.OrderItem) (*blocks.StructBlockValue, error) {
				calls++
				return nil, nil
			},
		})
		require.NoError(t, orders.SuppliersQuery().Sync())

		prodNil := seedWorkflowProduct(t, cat, "wf_nil_supplier", "Nil Data Item", "wf-nil-item", "SKU-WFN-01", "12.00")
		req := httptest.NewRequest(http.MethodPost, "/checkout", nil).WithContext(t.Context())
		order, _, err := orders.CreateOrder(req, customer, nil, &carts.CartItem{Product: prodNil, Amount: 1})
		require.NoError(t, err)

		coupon, redemption := seedWorkflowCouponRedemption(t, order, customer, "WFNIL10", "2.00")

		var acceptedFired, processedFired bool
		listenOrderAccepted(t, func(context.Context, signals.Signal[orders.OrderAcceptedSignal], orders.OrderAcceptedSignal) error {
			acceptedFired = true
			return nil
		})
		listenOrderProcessed(t, func(context.Context, signals.Signal[orders.OrderProcessedSignal], orders.OrderProcessedSignal) error {
			processedFired = true
			return nil
		})

		err = orders.Signals.Completed.Send(t.Context(), orders.OrderCompletedSignal{Order: order})
		require.NoError(t, err)
		require.Equal(t, 1, calls)
		require.True(t, acceptedFired)
		require.True(t, processedFired)

		dbOrder, err := orders.Query().Filter("ID", order.ID).Get()
		require.NoError(t, err)
		require.Equal(t, orders.OrderStateAccepted, dbOrder.Object.State)
		require.True(t, dbOrder.Object.Completed)
		require.True(t, dbOrder.Object.Processed)

		dbItem, err := queries.GetQuerySet(&orders.OrderItem{}).Filter("Order", order.ID).Get()
		require.NoError(t, err)
		require.False(t, dbItem.Object.SupplierData.HasData())

		dbRedemption, err := queries.GetQuerySet(&coupons.Redemption{}).Filter("ID", redemption.ID).Get()
		require.NoError(t, err)
		require.True(t, dbRedemption.Object.Counted)

		dbCoupon, err := queries.GetQuerySet(&coupons.Coupon{}).Filter("ID", coupon.ID).Get()
		require.NoError(t, err)
		require.Equal(t, uint64(1), dbCoupon.Object.UsageCount)
	})

	t.Run("Supplier returning data when OrderBlock returns nil, nil persists SupplierData", func(t *testing.T) {
		clearWorkflowTables(t)

		orders.RegisterSupplier(&orders.Supplier{
			Name: "wf_nil_block_with_data",
			ProductModel: func(context.Context) products.ProductModel {
				return &testOrderSupplierProduct{}
			},
			OrderBlockType: func(context.Context, *orders.Order, *orders.OrderItem) (blocks.Block, error) {
				return nil, nil
			},
			OrderIsAccepted: func(context.Context, *orders.Order, *orders.OrderItem) (*blocks.StructBlockValue, error) {
				return &blocks.StructBlockValue{
					V: map[string]any{
						"note": "persisted-without-block",
					},
				}, nil
			},
		})
		require.NoError(t, orders.SuppliersQuery().Sync())

		prod := seedWorkflowProduct(t, cat, "wf_nil_block_with_data", "Nil Block With Data", "wf-nil-block-data", "SKU-WFNB-01", "12.00")
		req := httptest.NewRequest(http.MethodPost, "/checkout", nil).WithContext(t.Context())
		order, _, err := orders.CreateOrder(req, customer, nil, &carts.CartItem{Product: prod, Amount: 1})
		require.NoError(t, err)

		err = orders.Signals.Completed.Send(t.Context(), orders.OrderCompletedSignal{Order: order})
		require.NoError(t, err)

		dbOrder, err := orders.Query().Filter("ID", order.ID).Get()
		require.NoError(t, err)
		require.Equal(t, orders.OrderStateAccepted, dbOrder.Object.State)
		require.True(t, dbOrder.Object.Completed)
		require.True(t, dbOrder.Object.Processed)

		dbItem, err := queries.GetQuerySet(&orders.OrderItem{}).Filter("Order", order.ID).Get()
		require.NoError(t, err)
		require.True(t, dbItem.Object.SupplierData.HasData())
		require.Nil(t, dbItem.Object.SupplierData.Block())
		rawJSON, err := dbItem.Object.SupplierData.MarshalJSON()
		require.NoError(t, err)
		require.JSONEq(t, `{"note":"persisted-without-block"}`, string(rawJSON))
	})

	t.Run("Idempotency: firing Completed again when Processed is true does not re-invoke OrderAccepted", func(t *testing.T) {
		clearWorkflowTables(t)

		var calls int
		orders.RegisterSupplier(&orders.Supplier{
			Name: "wf_idem",
			ProductModel: func(context.Context) products.ProductModel {
				return &testOrderSupplierProduct{}
			},
			OrderBlockType: func(context.Context, *orders.Order, *orders.OrderItem) (blocks.Block, error) {
				return alphaBlock, nil
			},
			OrderIsAccepted: func(context.Context, *orders.Order, *orders.OrderItem) (*blocks.StructBlockValue, error) {
				calls++
				return blocks.NewBlockValue(alphaBlock, map[string]any{
					"pin":    "IDEM-PIN-1",
					"serial": "IDEM-SER-1",
				}), nil
			},
		})
		require.NoError(t, orders.SuppliersQuery().Sync())

		prod := seedWorkflowProduct(t, cat, "wf_idem", "Idem Card", "wf-idem-card", "SKU-WFI-01", "15.00")
		req := httptest.NewRequest(http.MethodPost, "/checkout", nil).WithContext(t.Context())
		order, _, err := orders.CreateOrder(req, customer, nil, &carts.CartItem{Product: prod, Amount: 1})
		require.NoError(t, err)

		coupon, _ := seedWorkflowCouponRedemption(t, order, customer, "WFIDEM1", "3.00")

		var acceptedCount, processedCount int
		listenOrderAccepted(t, func(context.Context, signals.Signal[orders.OrderAcceptedSignal], orders.OrderAcceptedSignal) error {
			acceptedCount++
			return nil
		})
		listenOrderProcessed(t, func(context.Context, signals.Signal[orders.OrderProcessedSignal], orders.OrderProcessedSignal) error {
			processedCount++
			return nil
		})

		require.NoError(t, orders.Signals.Completed.Send(t.Context(), orders.OrderCompletedSignal{Order: order}))
		require.Equal(t, 1, calls)
		require.Equal(t, 1, acceptedCount)
		require.Equal(t, 1, processedCount)

		// Second Completed signal with a sparse Order stub must be a safe no-op and sync state/Customer.
		sparseOrder := &orders.Order{ID: order.ID}
		require.NoError(t, orders.Signals.Completed.Send(t.Context(), orders.OrderCompletedSignal{Order: sparseOrder}))
		require.Equal(t, 1, calls)
		require.Equal(t, 1, acceptedCount)
		require.Equal(t, 1, processedCount)
		require.Equal(t, orders.OrderStateAccepted, sparseOrder.State)
		require.True(t, sparseOrder.Completed)
		require.True(t, sparseOrder.Processed)
		require.NotNil(t, sparseOrder.Customer)
		require.Equal(t, customer.ID, sparseOrder.Customer.ID)

		dbOrder, err := orders.Query().Filter("ID", order.ID).Get()
		require.NoError(t, err)
		require.Equal(t, orders.OrderStateAccepted, dbOrder.Object.State)
		require.True(t, dbOrder.Object.Completed)
		require.True(t, dbOrder.Object.Processed)

		dbCoupon, err := queries.GetQuerySet(&coupons.Coupon{}).Filter("ID", coupon.ID).Get()
		require.NoError(t, err)
		require.Equal(t, uint64(1), dbCoupon.Object.UsageCount)
	})

	t.Run("Guards", func(t *testing.T) {
		t.Run("nil order", func(t *testing.T) {
			err := orders.Signals.Completed.Send(t.Context(), orders.OrderCompletedSignal{Order: nil})
			requireSignalErrorIs(t, err, orders.ErrOrderRequired)
		})

		t.Run("zero order ID", func(t *testing.T) {
			err := orders.Signals.Completed.Send(t.Context(), orders.OrderCompletedSignal{Order: &orders.Order{}})
			requireSignalErrorIs(t, err, orders.ErrOrderRequired)
		})

		t.Run("missing order", func(t *testing.T) {
			err := orders.Signals.Completed.Send(t.Context(), orders.OrderCompletedSignal{
				Order: &orders.Order{ID: drivers.NewULID()},
			})
			requireSignalErrorIs(t, err, qerrors.NoRows)
		})

		t.Run("rejected order", func(t *testing.T) {
			clearWorkflowTables(t)

			var supplierCalled bool
			orders.RegisterSupplier(&orders.Supplier{
				Name: "wf_guard_rejected",
				ProductModel: func(context.Context) products.ProductModel {
					return &testOrderSupplierProduct{}
				},
				OrderIsAccepted: func(context.Context, *orders.Order, *orders.OrderItem) (*blocks.StructBlockValue, error) {
					supplierCalled = true
					return nil, nil
				},
			})
			require.NoError(t, orders.SuppliersQuery().Sync())

			prod := seedWorkflowProduct(t, cat, "wf_guard_rejected", "Guard Rejected", "wf-guard-rej", "SKU-WFR-01", "10.00")
			req := httptest.NewRequest(http.MethodPost, "/checkout", nil).WithContext(t.Context())
			order, _, err := orders.CreateOrder(req, customer, nil, &carts.CartItem{Product: prod, Amount: 1})
			require.NoError(t, err)

			order.State = orders.OrderStateRejected
			_, err = orders.Query().Select("State").Filter("ID", order.ID).Update(order)
			require.NoError(t, err)

			var acceptedFired, processedFired bool
			listenOrderAccepted(t, func(context.Context, signals.Signal[orders.OrderAcceptedSignal], orders.OrderAcceptedSignal) error {
				acceptedFired = true
				return nil
			})
			listenOrderProcessed(t, func(context.Context, signals.Signal[orders.OrderProcessedSignal], orders.OrderProcessedSignal) error {
				processedFired = true
				return nil
			})

			err = orders.Signals.Completed.Send(t.Context(), orders.OrderCompletedSignal{Order: order})
			requireSignalErrorIs(t, err, orders.ErrOrderRejected)
			require.False(t, supplierCalled)
			require.False(t, acceptedFired)
			require.False(t, processedFired)

			dbOrder, err := orders.Query().Filter("ID", order.ID).Get()
			require.NoError(t, err)
			require.Equal(t, orders.OrderStateRejected, dbOrder.Object.State)
			require.False(t, dbOrder.Object.Completed)
			require.False(t, dbOrder.Object.Processed)
		})

		t.Run("unregistered supplier backend", func(t *testing.T) {
			clearWorkflowTables(t)

			// Seed product referencing a supplier name that is NOT registered in orders registry.
			prod := seedWorkflowProduct(t, cat, "wf_unregistered_backend", "Unregistered Item", "wf-unreg", "SKU-WFU-01", "19.99")
			req := httptest.NewRequest(http.MethodPost, "/checkout", nil).WithContext(t.Context())
			order, _, err := orders.CreateOrder(req, customer, nil, &carts.CartItem{Product: prod, Amount: 1})
			require.NoError(t, err)

			err = orders.Signals.Completed.Send(t.Context(), orders.OrderCompletedSignal{Order: order})
			requireSignalErrorIs(t, err, orders.ErrSupplierNotFound)

			dbOrder, err := orders.Query().Filter("ID", order.ID).Get()
			require.NoError(t, err)
			require.Equal(t, orders.OrderStatePending, dbOrder.Object.State)
			require.False(t, dbOrder.Object.Completed)
			require.False(t, dbOrder.Object.Processed)
		})

		t.Run("disabled SupplierModel", func(t *testing.T) {
			clearWorkflowTables(t)

			var supplierCalled bool
			orders.RegisterSupplier(&orders.Supplier{
				Name: "wf_disabled",
				ProductModel: func(context.Context) products.ProductModel {
					return &testOrderSupplierProduct{}
				},
				OrderIsAccepted: func(context.Context, *orders.Order, *orders.OrderItem) (*blocks.StructBlockValue, error) {
					supplierCalled = true
					return nil, nil
				},
			})
			require.NoError(t, orders.SuppliersQuery().Sync())
			_, err := orders.SuppliersQuery().
				Filter(expr.Q("InternalName", "wf_disabled")).
				BulkUpdate(expr.As("Enabled", expr.V(false)))
			require.NoError(t, err)

			prod := seedWorkflowProduct(t, cat, "wf_disabled", "Disabled Supplier Item", "wf-disabled-item", "SKU-WFD-01", "14.00")
			req := httptest.NewRequest(http.MethodPost, "/checkout", nil).WithContext(t.Context())
			order, _, err := orders.CreateOrder(req, customer, nil, &carts.CartItem{Product: prod, Amount: 1})
			require.NoError(t, err)

			err = orders.Signals.Completed.Send(t.Context(), orders.OrderCompletedSignal{Order: order})
			requireSignalErrorIs(t, err, orders.ErrSupplierDisabled)
			require.False(t, supplierCalled)

			dbOrder, err := orders.Query().Filter("ID", order.ID).Get()
			require.NoError(t, err)
			require.Equal(t, orders.OrderStatePending, dbOrder.Object.State)
			require.False(t, dbOrder.Object.Completed)
			require.False(t, dbOrder.Object.Processed)
		})

		t.Run("invalid supplier data fails OrderBlock validation", func(t *testing.T) {
			clearWorkflowTables(t)

			orders.RegisterSupplier(&orders.Supplier{
				Name: "wf_invalid_data",
				ProductModel: func(context.Context) products.ProductModel {
					return &testOrderSupplierProduct{}
				},
				OrderBlockType: func(context.Context, *orders.Order, *orders.OrderItem) (blocks.Block, error) {
					return alphaBlock, nil
				},
				OrderIsAccepted: func(context.Context, *orders.Order, *orders.OrderItem) (*blocks.StructBlockValue, error) {
					// "pin" is required on alphaBlock, so empty string fails validation.
					return &blocks.StructBlockValue{
						V: map[string]any{
							"pin":    "",
							"serial": "SER-ONLY",
						},
					}, nil
				},
			})
			require.NoError(t, orders.SuppliersQuery().Sync())

			prod := seedWorkflowProduct(t, cat, "wf_invalid_data", "Invalid Data Item", "wf-invalid-item", "SKU-WFV-01", "14.00")
			req := httptest.NewRequest(http.MethodPost, "/checkout", nil).WithContext(t.Context())
			order, _, err := orders.CreateOrder(req, customer, nil, &carts.CartItem{Product: prod, Amount: 1})
			require.NoError(t, err)

			err = orders.Signals.Completed.Send(t.Context(), orders.OrderCompletedSignal{Order: order})
			requireSignalErrorIs(t, err, orders.ErrInvalidSupplierData)

			dbOrder, err := orders.Query().Filter("ID", order.ID).Get()
			require.NoError(t, err)
			require.Equal(t, orders.OrderStatePending, dbOrder.Object.State)
			require.False(t, dbOrder.Object.Completed)
			require.False(t, dbOrder.Object.Processed)

			dbItem, err := queries.GetQuerySet(&orders.OrderItem{}).Filter("Order", order.ID).Get()
			require.NoError(t, err)
			require.False(t, dbItem.Object.SupplierData.HasData())
		})
	})

	t.Run("All-or-nothing rollback", func(t *testing.T) {
		assertRolledBack := func(t *testing.T, order *orders.Order, coupon *coupons.Coupon, redemption *coupons.Redemption) {
			t.Helper()
			// Caller's order struct remains unchanged on error.
			require.Equal(t, orders.OrderStatePending, order.State)
			require.False(t, order.Completed)
			require.False(t, order.Processed)

			dbOrder, err := orders.Query().Filter("ID", order.ID).Get()
			require.NoError(t, err)
			require.Equal(t, orders.OrderStatePending, dbOrder.Object.State)
			require.False(t, dbOrder.Object.Completed)
			require.False(t, dbOrder.Object.Processed)

			dbItems, err := queries.GetQuerySet(&orders.OrderItem{}).Filter("Order", order.ID).All()
			require.NoError(t, err)
			require.NotEmpty(t, dbItems)
			for _, row := range dbItems {
				require.False(t, row.Object.SupplierData.HasData(), "expected SupplierData rolled back on item %s", row.Object.ID)
			}

			dbRedemption, err := queries.GetQuerySet(&coupons.Redemption{}).Filter("ID", redemption.ID).Get()
			require.NoError(t, err)
			require.False(t, dbRedemption.Object.Counted)
			require.False(t, dbRedemption.Object.Invalidated)

			dbCoupon, err := queries.GetQuerySet(&coupons.Coupon{}).Filter("ID", coupon.ID).Get()
			require.NoError(t, err)
			require.Equal(t, uint64(0), dbCoupon.Object.UsageCount)
		}

		t.Run("2nd item OrderAccepted failure rolls back 1st item SupplierData, Order, and coupons", func(t *testing.T) {
			clearWorkflowTables(t)

			errSecondItemFail := errors.New("supplier 2 upstream failure")
			var firstItemCalls, secondItemCalls int

			orders.RegisterSupplier(&orders.Supplier{
				Name: "wf_rb_ok",
				ProductModel: func(context.Context) products.ProductModel {
					return &testOrderSupplierProduct{}
				},
				OrderBlockType: func(context.Context, *orders.Order, *orders.OrderItem) (blocks.Block, error) {
					return alphaBlock, nil
				},
				OrderIsAccepted: func(context.Context, *orders.Order, *orders.OrderItem) (*blocks.StructBlockValue, error) {
					firstItemCalls++
					return blocks.NewBlockValue(alphaBlock, map[string]any{
						"pin":    "RB-PIN-1",
						"serial": "RB-SER-1",
					}), nil
				},
			})

			orders.RegisterSupplier(&orders.Supplier{
				Name: "wf_rb_fail",
				ProductModel: func(context.Context) products.ProductModel {
					return &testOrderSupplierProduct{}
				},
				OrderIsAccepted: func(context.Context, *orders.Order, *orders.OrderItem) (*blocks.StructBlockValue, error) {
					secondItemCalls++
					return nil, errSecondItemFail
				},
			})
			require.NoError(t, orders.SuppliersQuery().Sync())

			prod1 := seedWorkflowProduct(t, cat, "wf_rb_ok", "Rollback OK Item", "wf-rb-ok", "SKU-WFRB-1", "10.00")
			prod2 := seedWorkflowProduct(t, cat, "wf_rb_fail", "Rollback Fail Item", "wf-rb-fail", "SKU-WFRB-2", "20.00")

			req := httptest.NewRequest(http.MethodPost, "/checkout", nil).WithContext(t.Context())
			order, createdItems, err := orders.CreateOrder(
				req,
				customer,
				nil,
				&carts.CartItem{Product: prod1, Amount: 1},
				&carts.CartItem{Product: prod2, Amount: 1},
			)
			require.NoError(t, err)
			require.Len(t, createdItems, 2)

			// Ensure the failing item is ordered after the succeeding item (workflow processes items OrderBy("ID")).
			if createdItems[0].ID.String() > createdItems[1].ID.String() {
				_, err = queries.GetQuerySet(&orders.OrderItem{}).
					Filter("ID", createdItems[0].ID).
					BulkUpdate(expr.As("Supplier", expr.V(products.SupplierValue("wf_rb_ok"))))
				require.NoError(t, err)
				_, err = queries.GetQuerySet(&orders.OrderItem{}).
					Filter("ID", createdItems[1].ID).
					BulkUpdate(expr.As("Supplier", expr.V(products.SupplierValue("wf_rb_fail"))))
				require.NoError(t, err)
			}

			coupon, redemption := seedWorkflowCouponRedemption(t, order, customer, "WFRBFAIL", "5.00")

			var acceptedFired, processedFired bool
			listenOrderAccepted(t, func(context.Context, signals.Signal[orders.OrderAcceptedSignal], orders.OrderAcceptedSignal) error {
				acceptedFired = true
				return nil
			})
			listenOrderProcessed(t, func(context.Context, signals.Signal[orders.OrderProcessedSignal], orders.OrderProcessedSignal) error {
				processedFired = true
				return nil
			})

			err = orders.Signals.Completed.Send(t.Context(), orders.OrderCompletedSignal{Order: order})
			requireSignalErrorIs(t, err, errSecondItemFail)
			require.Equal(t, 1, firstItemCalls)
			require.Equal(t, 1, secondItemCalls)
			require.False(t, acceptedFired)
			require.False(t, processedFired)

			assertRolledBack(t, order, coupon, redemption)
		})

		t.Run("Signals.Accepted listener error rolls back everything", func(t *testing.T) {
			clearWorkflowTables(t)

			orders.RegisterSupplier(&orders.Supplier{
				Name: "wf_rb_sig_acc",
				ProductModel: func(context.Context) products.ProductModel {
					return &testOrderSupplierProduct{}
				},
				OrderBlockType: func(context.Context, *orders.Order, *orders.OrderItem) (blocks.Block, error) {
					return alphaBlock, nil
				},
				OrderIsAccepted: func(context.Context, *orders.Order, *orders.OrderItem) (*blocks.StructBlockValue, error) {
					return blocks.NewBlockValue(alphaBlock, map[string]any{
						"pin":    "RB-ACC-PIN",
						"serial": "RB-ACC-SER",
					}), nil
				},
			})
			require.NoError(t, orders.SuppliersQuery().Sync())

			prod := seedWorkflowProduct(t, cat, "wf_rb_sig_acc", "Rollback Accepted Signal", "wf-rb-acc", "SKU-WFRBA-1", "10.00")
			req := httptest.NewRequest(http.MethodPost, "/checkout", nil).WithContext(t.Context())
			order, _, err := orders.CreateOrder(req, customer, nil, &carts.CartItem{Product: prod, Amount: 1})
			require.NoError(t, err)

			coupon, redemption := seedWorkflowCouponRedemption(t, order, customer, "WFRBACC", "4.00")

			errAcceptedAbort := errors.New("accepted listener aborted")
			var processedFired bool
			listenOrderAccepted(t, func(context.Context, signals.Signal[orders.OrderAcceptedSignal], orders.OrderAcceptedSignal) error {
				return errAcceptedAbort
			})
			listenOrderProcessed(t, func(context.Context, signals.Signal[orders.OrderProcessedSignal], orders.OrderProcessedSignal) error {
				processedFired = true
				return nil
			})

			err = orders.Signals.Completed.Send(t.Context(), orders.OrderCompletedSignal{Order: order})
			requireSignalErrorIs(t, err, errAcceptedAbort)
			require.False(t, processedFired)

			assertRolledBack(t, order, coupon, redemption)
		})

		t.Run("Signals.Processed listener error rolls back everything", func(t *testing.T) {
			clearWorkflowTables(t)

			orders.RegisterSupplier(&orders.Supplier{
				Name: "wf_rb_sig_proc",
				ProductModel: func(context.Context) products.ProductModel {
					return &testOrderSupplierProduct{}
				},
				OrderBlockType: func(context.Context, *orders.Order, *orders.OrderItem) (blocks.Block, error) {
					return alphaBlock, nil
				},
				OrderIsAccepted: func(context.Context, *orders.Order, *orders.OrderItem) (*blocks.StructBlockValue, error) {
					return blocks.NewBlockValue(alphaBlock, map[string]any{
						"pin":    "RB-PROC-PIN",
						"serial": "RB-PROC-SER",
					}), nil
				},
			})
			require.NoError(t, orders.SuppliersQuery().Sync())

			prod := seedWorkflowProduct(t, cat, "wf_rb_sig_proc", "Rollback Processed Signal", "wf-rb-proc", "SKU-WFRBP-1", "10.00")
			req := httptest.NewRequest(http.MethodPost, "/checkout", nil).WithContext(t.Context())
			order, _, err := orders.CreateOrder(req, customer, nil, &carts.CartItem{Product: prod, Amount: 1})
			require.NoError(t, err)

			coupon, redemption := seedWorkflowCouponRedemption(t, order, customer, "WFRBPROC", "4.00")

			errProcessedAbort := errors.New("processed listener aborted")
			var acceptedFired bool
			listenOrderAccepted(t, func(context.Context, signals.Signal[orders.OrderAcceptedSignal], orders.OrderAcceptedSignal) error {
				acceptedFired = true
				return nil
			})
			listenOrderProcessed(t, func(context.Context, signals.Signal[orders.OrderProcessedSignal], orders.OrderProcessedSignal) error {
				return errProcessedAbort
			})

			err = orders.Signals.Completed.Send(t.Context(), orders.OrderCompletedSignal{Order: order})
			requireSignalErrorIs(t, err, errProcessedAbort)
			require.True(t, acceptedFired)

			assertRolledBack(t, order, coupon, redemption)
		})
	})

	t.Run("Caller-provided outer transaction", func(t *testing.T) {
		registerOuterTxSupplier := func(t *testing.T) *products.Product {
			t.Helper()
			orders.RegisterSupplier(&orders.Supplier{
				Name: "wf_outer_tx",
				ProductModel: func(context.Context) products.ProductModel {
					return &testOrderSupplierProduct{}
				},
				OrderBlockType: func(context.Context, *orders.Order, *orders.OrderItem) (blocks.Block, error) {
					return alphaBlock, nil
				},
				OrderIsAccepted: func(context.Context, *orders.Order, *orders.OrderItem) (*blocks.StructBlockValue, error) {
					return blocks.NewBlockValue(alphaBlock, map[string]any{
						"pin":    "OUTER-PIN-1",
						"serial": "OUTER-SER-1",
					}), nil
				},
			})
			require.NoError(t, orders.SuppliersQuery().Sync())
			return seedWorkflowProduct(t, cat, "wf_outer_tx", "Outer Tx Card", "wf-outer-tx", "SKU-WFOT-1", "30.00")
		}

		t.Run("rollback of outer transaction undoes workflow updates", func(t *testing.T) {
			clearWorkflowTables(t)
			prod := registerOuterTxSupplier(t)

			req := httptest.NewRequest(http.MethodPost, "/checkout", nil).WithContext(t.Context())
			order, _, err := orders.CreateOrder(req, customer, nil, &carts.CartItem{Product: prod, Amount: 1})
			require.NoError(t, err)

			coupon, redemption := seedWorkflowCouponRedemption(t, order, customer, "WFOUTERRB", "5.00")

			txCtx, tx, err := queries.StartTransaction(t.Context())
			require.NoError(t, err)
			defer tx.Rollback(txCtx)

			err = orders.Signals.Completed.Send(txCtx, orders.OrderCompletedSignal{Order: order})
			require.NoError(t, err)

			// Changes are visible inside the caller's outer transaction.
			inTxOrder, err := orders.Query().WithContext(txCtx).Filter("ID", order.ID).Get()
			require.NoError(t, err)
			require.Equal(t, orders.OrderStateAccepted, inTxOrder.Object.State)
			require.True(t, inTxOrder.Object.Completed)
			require.True(t, inTxOrder.Object.Processed)

			inTxItem, err := queries.GetQuerySet(&orders.OrderItem{}).WithContext(txCtx).Filter("Order", order.ID).Get()
			require.NoError(t, err)
			require.True(t, inTxItem.Object.SupplierData.HasData())

			require.NoError(t, tx.Rollback(txCtx))

			// Outside the rolled-back outer transaction, nothing was persisted.
			dbOrder, err := orders.Query().Filter("ID", order.ID).Get()
			require.NoError(t, err)
			require.Equal(t, orders.OrderStatePending, dbOrder.Object.State)
			require.False(t, dbOrder.Object.Completed)
			require.False(t, dbOrder.Object.Processed)

			dbItem, err := queries.GetQuerySet(&orders.OrderItem{}).Filter("Order", order.ID).Get()
			require.NoError(t, err)
			require.False(t, dbItem.Object.SupplierData.HasData())

			dbRedemption, err := queries.GetQuerySet(&coupons.Redemption{}).Filter("ID", redemption.ID).Get()
			require.NoError(t, err)
			require.False(t, dbRedemption.Object.Counted)

			dbCoupon, err := queries.GetQuerySet(&coupons.Coupon{}).Filter("ID", coupon.ID).Get()
			require.NoError(t, err)
			require.Equal(t, uint64(0), dbCoupon.Object.UsageCount)
		})

		t.Run("commit of outer transaction persists workflow updates", func(t *testing.T) {
			clearWorkflowTables(t)
			prod := registerOuterTxSupplier(t)

			req := httptest.NewRequest(http.MethodPost, "/checkout", nil).WithContext(t.Context())
			order, _, err := orders.CreateOrder(req, customer, nil, &carts.CartItem{Product: prod, Amount: 1})
			require.NoError(t, err)

			coupon, redemption := seedWorkflowCouponRedemption(t, order, customer, "WFOUTEROK", "5.00")

			txCtx, tx, err := queries.StartTransaction(t.Context())
			require.NoError(t, err)
			defer tx.Rollback(txCtx)

			err = orders.Signals.Completed.Send(txCtx, orders.OrderCompletedSignal{Order: order})
			require.NoError(t, err)
			require.NoError(t, tx.Commit(txCtx))

			dbOrder, err := orders.Query().Filter("ID", order.ID).Get()
			require.NoError(t, err)
			require.Equal(t, orders.OrderStateAccepted, dbOrder.Object.State)
			require.True(t, dbOrder.Object.Completed)
			require.True(t, dbOrder.Object.Processed)

			dbItem, err := queries.GetQuerySet(&orders.OrderItem{}).Filter("Order", order.ID).Get()
			require.NoError(t, err)
			require.True(t, dbItem.Object.SupplierData.HasData())
			require.Same(t, alphaBlock, dbItem.Object.SupplierData.Block())
			pinField, ok := dbItem.Object.SupplierData.V["pin"].(blocks.BoundBlockValue)
			require.True(t, ok)
			require.Equal(t, "OUTER-PIN-1", pinField.Data())

			dbRedemption, err := queries.GetQuerySet(&coupons.Redemption{}).Filter("ID", redemption.ID).Get()
			require.NoError(t, err)
			require.True(t, dbRedemption.Object.Counted)

			dbCoupon, err := queries.GetQuerySet(&coupons.Coupon{}).Filter("ID", coupon.ID).Get()
			require.NoError(t, err)
			require.Equal(t, uint64(1), dbCoupon.Object.UsageCount)
		})
	})
}
