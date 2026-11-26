//go:build test

package orders_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Nigel2392/go-django/contrib/blocks"
	"github.com/Nigel2392/go-django/contrib/shop/carts"
	"github.com/Nigel2392/go-django/contrib/shop/customers"
	"github.com/Nigel2392/go-django/contrib/shop/orders"
	"github.com/Nigel2392/go-django/contrib/shop/products"
	queries "github.com/Nigel2392/go-django/queries/src"
	qerrors "github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/stretchr/testify/require"
)

func productWithSpecificID(product *products.Product, id string) *products.Product {
	copy := *product
	copy.SpecificID = id
	return &copy
}

func testSupplierSync(t *testing.T) {
	ctx := t.Context()
	qs := orders.SuppliersQuery().WithContext(ctx)
	_, err := qs.ExplicitSave().BulkCreate([]*orders.SupplierModel{
		{InternalName: "order_test_supplier", Enabled: true},
		{InternalName: "retired_test_supplier", Enabled: true},
	})
	require.NoError(t, err)
	_, err = qs.Filter("InternalName", "order_test_supplier").
		BulkUpdate(expr.As("Enabled", expr.V(false)))
	require.NoError(t, err)

	orders.RegisterSupplier(&orders.Supplier{Name: "new_test_supplier"})
	for range 2 {
		require.NoError(t, qs.Sync())
		rows, err := qs.All()
		require.NoError(t, err)
		got := make(map[string]bool, len(rows))
		for _, row := range rows {
			got[row.Object.InternalName] = row.Object.Enabled
		}
		require.Equal(t, map[string]bool{
			"order_test_supplier":   false,
			"retired_test_supplier": false,
			"new_test_supplier":     true,
		}, got)
	}
}

func testOrderSupplierSnapshot(t *testing.T, customer *customers.Customer, product *products.Product) {
	clearOrderAndCartTables(t)
	t.Cleanup(func() { clearOrderAndCartTables(t) })
	req := httptest.NewRequest(http.MethodPost, "/checkout", nil).WithContext(t.Context())
	order, items, err := orders.CreateOrder(req, customer, nil, &carts.CartItem{Product: product, Amount: 2})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, product.Supplier, items[0].Supplier)
	require.NotSame(t, product, items[0].Product.Data.Product)

	qs := func() *queries.QuerySet[*orders.OrderItem] {
		return queries.GetQuerySet(&orders.OrderItem{}).Filter("ID", items[0].ID)
	}
	t.Run("concrete product survives database reload", func(t *testing.T) {
		row, err := qs().Get()
		require.NoError(t, err)
		require.Equal(t, product.Supplier, row.Object.Supplier)
		require.False(t, row.Object.SupplierData.HasData())
		require.Equal(t, product.ID, row.Object.Product.Data.Product.ID)
		specific, ok := row.Object.Product.Data.Specific.(*testOrderSupplierProduct)
		require.True(t, ok, "decoded product has type %T", row.Object.Product.Data.Specific)
		require.Equal(t, "SKU-KB-01", specific.SKU)
	})

	t.Run("metadata binds to the supplier block", func(t *testing.T) {
		backend, ok := orders.RetrieveSupplier(product.Supplier)
		require.True(t, ok)
		t.Cleanup(func() { orders.RegisterSupplier(backend) })
		block := blocks.NewStructBlock()
		block.AddField("tracking", blocks.CharBlock())
		supplier := *backend.(*orders.Supplier)
		supplier.OrderBlockType = func(_ context.Context, gotOrder *orders.Order, item *orders.OrderItem) (blocks.Block, error) {
			require.Equal(t, order.ID, gotOrder.ID)
			require.Equal(t, items[0].ID, item.ID)
			return block, nil
		}
		orders.RegisterSupplier(&supplier)

		row, err := qs().Get()
		require.NoError(t, err)
		require.False(t, row.Object.SupplierData.HasData())

		_, err = qs().BulkUpdate(expr.As("SupplierData", expr.V(`{"tracking":"DELIVERY-12"}`)))
		require.NoError(t, err)
		row, err = qs().Get()
		require.NoError(t, err)
		require.Same(t, block, row.Object.SupplierData.Block())
		tracking, ok := row.Object.SupplierData.V["tracking"].(blocks.BoundBlockValue)
		require.True(t, ok, "tracking field has type %T", row.Object.SupplierData.V["tracking"])
		require.Equal(t, "DELIVERY-12", tracking.Data())

		supplier.OrderBlockType = func(context.Context, *orders.Order, *orders.OrderItem) (blocks.Block, error) {
			return nil, errors.New("supplier failed to build block")
		}
		_, err = qs().Get()
		require.ErrorContains(t, err, "supplier failed to build block")

		row.Object.Supplier = "missing_test_supplier"
		require.ErrorIs(t, row.Object.AfterQuery(t.Context()), qerrors.CorruptedData)
	})

	t.Run("legacy orders keep the base product and supplier", func(t *testing.T) {
		legacyProduct := *product
		legacyProduct.Supplier = "retired_test_supplier"
		baseJSON, err := json.Marshal(&legacyProduct)
		require.NoError(t, err)
		_, err = qs().BulkUpdate(
			expr.As("Product", expr.V(string(baseJSON))),
			expr.As("Supplier", expr.V("")),
			expr.As("SupplierData", expr.V(nil)),
		)
		require.NoError(t, err)
		row, err := qs().Get()
		require.NoError(t, err)
		require.Equal(t, product.ID, row.Object.Product.Data.Product.ID)
		require.Equal(t, legacyProduct.Supplier, row.Object.Supplier)
		require.Nil(t, row.Object.Product.Data.Specific)
	})
}
