package main

import (
	"github.com/Nigel2392/go-django/contrib/shop/supplier"
	"github.com/Nigel2392/go-django/src/core/trans"
)

func init() {
	supplier.Register(&supplier.Supplier{
		Name:            "apple",
		TextLabel:       trans.S("Apfel"),
		TextDescription: trans.S("Selling apples to your local farmers"),
	})
	supplier.Register(&supplier.Supplier{
		Name:            "microsoft",
		TextLabel:       trans.S("Microslop"),
		TextDescription: trans.S("Selling your data to local data farmers"),
	})
	supplier.Register(&supplier.Supplier{
		Name:            "google",
		TextLabel:       trans.S("Gooble"),
		TextDescription: trans.S("In the business of making mildly shitty chatbots"),
	})
}
