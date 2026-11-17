package products

import (
	"github.com/Nigel2392/go-django/contrib/shop/products/supplier"
	"github.com/Nigel2392/go-django/contrib/shop/shared/signals"
)

type ProductCreatedSignal struct {
	signals.BaseSignal
	Product  *Product
	Specific supplier.ProductModel
}

type ProductUpdatedSignal struct {
	signals.BaseSignal
	Product  signals.Changed[*Product]
	Specific signals.Changed[supplier.ProductModel]
}

type productSignalManager struct {
	Created signals.Signal[ProductCreatedSignal]
	Updated signals.Signal[ProductUpdatedSignal]
}

var Signals = productSignalManager{
	Created: signals.Pool.Get[ProductCreatedSignal]("shop.products.Product.Created"),
	Updated: signals.Pool.Get[ProductUpdatedSignal]("shop.products.Product.Updated"),
}
