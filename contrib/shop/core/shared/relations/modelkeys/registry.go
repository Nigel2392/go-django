package modelkeys

import (
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/utils/djangoapps"
)

var (
	_ = contenttypes.NewLazyRegistry(ProductModelKey, func(ctd *contenttypes.ContentTypeDefinition) bool {
		var app, ok = djangoapps.GetAppForModel(ctd.ContentObject)
		if !ok {
			return false
		}
		return app.Name() == "shop" && ctd.Name() == "Product"
	})
	_ = contenttypes.NewLazyRegistry(OrderModelKey, func(ctd *contenttypes.ContentTypeDefinition) bool {
		var app, ok = djangoapps.GetAppForModel(ctd.ContentObject)
		if !ok {
			return false
		}
		return app.Name() == "shop" && ctd.Name() == "Order"
	})
	_ = contenttypes.NewLazyRegistry(CartModelKey, func(ctd *contenttypes.ContentTypeDefinition) bool {
		var app, ok = djangoapps.GetAppForModel(ctd.ContentObject)
		if !ok {
			return false
		}
		return app.Name() == "shop" && ctd.Name() == "Cart"
	})
	_ = contenttypes.NewLazyRegistry(CategoryModelKey, func(ctd *contenttypes.ContentTypeDefinition) bool {
		var app, ok = djangoapps.GetAppForModel(ctd.ContentObject)
		if !ok {
			return false
		}
		return app.Name() == "shop" && ctd.Name() == "Category"
	})
	_ = contenttypes.NewLazyRegistry(PaymentModelKey, func(ctd *contenttypes.ContentTypeDefinition) bool {
		var app, ok = djangoapps.GetAppForModel(ctd.ContentObject)
		if !ok {
			return false
		}
		return app.Name() == "shop" && ctd.Name() == "Payment"
	})
)
