package main

import (
	"github.com/Nigel2392/go-django/contrib/shop/models/orders"
	"github.com/Nigel2392/go-django/src/core/trans"
)

func init() {
	orders.RegisterSupplier(&orders.Supplier{
		Name:            "apple",
		TextLabel:       trans.S("Apfel"),
		TextDescription: trans.S("Selling apples to your local farmers"),
	})
	orders.RegisterSupplier(&orders.Supplier{
		Name:            "microsoft",
		TextLabel:       trans.S("Microslop"),
		TextDescription: trans.S("Selling your data to local data farmers"),
	})
	orders.RegisterSupplier(&orders.Supplier{
		Name:            "google",
		TextLabel:       trans.S("Gooble"),
		TextDescription: trans.S("In the business of making mildly shitty chatbots"),
	})
}
