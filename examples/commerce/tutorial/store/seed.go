package store

import (
	"context"
	"fmt"

	"github.com/Nigel2392/go-django/contrib/auth"
	"github.com/Nigel2392/go-django/contrib/editor"
	auditlogs "github.com/Nigel2392/go-django/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/contrib/shop/models/carts"
	"github.com/Nigel2392/go-django/contrib/shop/models/categories"
	"github.com/Nigel2392/go-django/contrib/shop/models/coupons"
	"github.com/Nigel2392/go-django/contrib/shop/models/customers"
	"github.com/Nigel2392/go-django/contrib/shop/models/orders"
	"github.com/Nigel2392/go-django/contrib/shop/models/payments"
	"github.com/Nigel2392/go-django/contrib/shop/models/products"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/queries/src/migrator/sql/sqlite"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	"github.com/shopspring/decimal"
)

func initializeDemo(_ django.Settings, db drivers.Database) error {
	ctx := context.Background()
	// Build the disposable schema here so the tutorial runs without a migration step.
	schema := sqlite.NewSQLiteSchemaEditor(db)
	for _, model := range []attrs.Definer{
		&auth.User{}, &auditlogs.Entry{}, &categories.Category{}, &orders.SupplierModel{},
		&products.Product{}, &Product{}, &customers.Customer{}, &carts.Cart{}, &carts.CartItem{},
		&orders.OrderState{}, &orders.Order{}, &orders.OrderItem{}, &Delivery{},
		&coupons.Coupon{}, &coupons.CouponProductThrough{}, &coupons.Redemption{},
		&payments.ProviderAccount{}, &payments.Payment{},
	} {
		table := migrator.NewModelTable(model)
		if err := schema.CreateTable(ctx, table, true); err != nil {
			return fmt.Errorf("create %T table: %w", model, err)
		}
		for _, index := range table.Indexes() {
			if err := schema.AddIndex(ctx, table, index, true); err != nil {
				return fmt.Errorf("create %T index %s: %w", model, index.Name(), err)
			}
		}
	}
	if err := orders.SuppliersQuery().WithContext(ctx).Sync(); err != nil {
		return err
	}
	if err := payments.ProvidersQuery().WithContext(ctx).Sync(); err != nil {
		return err
	}

	category, err := queries.GetQuerySet(&categories.Category{}).WithContext(ctx).Create(&categories.Category{
		Name: "Stationery", Slug: "stationery",
	})
	if err != nil {
		return err
	}
	for _, item := range []struct {
		title, slug, supplier, availability string
		price                               int64
	}{
		{"Notebook (in stock)", "notebook", notebookSupplier, inStock, 10},
		{"Notebook (backorder)", "backorder-notebook", notebookSupplier, backordered, 10},
		{"Notebook (discontinued)", "discontinued-notebook", notebookSupplier, discontinued, 10},
		{"Pen", "pen", penSupplier, inStock, 5},
	} {
		product := &Product{
			Product: &products.Product{
				Title:    item.title,
				Slug:     item.slug,
				Category: category,
				Supplier: products.SupplierValue(item.supplier),
				Image:    &mediafiles.SimpleStoredObject{Filepath: item.slug + ".png"},
				Editor:   &editor.EditorJSBlockData{},
				Price:    decimal.NewFromInt(item.price),
				TaxRate:  decimal.NewFromInt(20),
			},
			Availability: item.availability,
		}
		product.Instance = product
		if _, err := queries.GetQuerySet(&Product{}).WithContext(ctx).Create(product); err != nil {
			return err
		}
		if _, err := queries.GetQuerySet(&products.Product{}).WithContext(ctx).Create(product.Product); err != nil {
			return err
		}
	}
	return nil
}
