package adminsite

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/admin/searches"
	"github.com/Nigel2392/go-django/contrib/shop/coupons"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
)

var CouponsAdmin = admin.ModelOptions{
	Name:                "coupons",
	MenuOrder:           900,
	Model:               &coupons.Coupon{},
	MenuLabel:           trans.S("Coupons"),
	RegisterToAdminMenu: true,
	DisallowDelete:      true,
	Labels: map[string]func(ctx context.Context) string{
		"ShortDescription": trans.S("Description"),
		"UsageCount":       trans.S("Uses"),
		"UsageLimit":       trans.S("Limit"),
		"ValidFrom":        trans.S("Valid From"),
		"ValidUntil":       trans.S("Valid Until"),
		"CreatedAt":        trans.S("Created"),
	},
	ListView: admin.ListViewOptions{
		Fields: []string{
			"ID",
			"Code",
			"Title",
			"ShortDescription",
			"UsageCount",
			"UsageLimit",
			"ValidFrom",
			"ValidUntil",
			"CreatedAt",
		},
		Search: &searches.SearchOptions{
			Fields: []searches.SearchField{
				{Name: "Code", Lookup: expr.LOOKUP_ICONTANS},
				{Name: "Title", Lookup: expr.LOOKUP_ICONTANS},
				{Name: "ShortDescription", Lookup: expr.LOOKUP_ICONTANS},
				{Name: "CreatedBy.Email", Lookup: expr.LOOKUP_ICONTANS},
			},
		},
	},
	AddView: admin.FormViewOptions{
		Panels: []admin.Panel{
			admin.TabbedPanel(
				admin.PanelTab(
					trans.S("Coupon"),
					admin.TitlePanel(admin.FieldPanel("Title")),
					admin.FieldPanel("Code"),
					admin.FieldPanel("ShortDescription"),
					admin.RowPanel(
						admin.FieldPanel("MaxDiscount"),
						admin.FieldPanel("IsPercentage"),
					),
				),
				admin.PanelTab(
					trans.S("Constraints"),
					admin.FieldPanel("UsageLimit"),
					admin.FieldPanel("MinOrderAmount"),
					admin.FieldPanel("ValidFrom"),
					admin.FieldPanel("ValidUntil"),
				),
				admin.PanelTab(
					trans.S("Products"),
					&admin.ModelFormPanel[*coupons.CouponProductThrough, modelforms.ModelForm[*coupons.CouponProductThrough]]{
						TargetType: &coupons.CouponProductThrough{},
						FieldName:  "Products",
						Classname:  "collapsible",
						MinNum:     1,
						Panels: []admin.Panel{
							admin.FieldPanel("Product"),
						},
					},
				),
			),
		},
	},
	EditView: admin.FormViewOptions{
		Panels: []admin.Panel{
			admin.TabbedPanel(
				admin.PanelTab(
					trans.S("Coupon"),
					admin.TitlePanel(admin.FieldPanel("Title")),
					admin.FieldPanel("Code"),
					admin.FieldPanel("ShortDescription"),
					admin.FieldPanel("CreatedBy"),
					admin.RowPanel(
						admin.FieldPanel("MaxDiscount"),
						admin.FieldPanel("IsPercentage"),
					),
				),
				admin.PanelTab(
					// Constraints for the coupon
					trans.S("Constraints"),
					admin.FieldPanel("UsageCount"),
					admin.FieldPanel("UsageLimit"),
					admin.FieldPanel("MinOrderAmount"),
					admin.FieldPanel("ValidFrom"),
					admin.FieldPanel("ValidUntil"),
					admin.FieldPanel("CreatedAt"),
					admin.FieldPanel("UpdatedAt"),
				),
				admin.PanelTab(
					trans.S("Products"),
					&admin.ModelFormPanel[*coupons.CouponProductThrough, modelforms.ModelForm[*coupons.CouponProductThrough]]{
						TargetType: &coupons.CouponProductThrough{},
						FieldName:  "Products",
						Classname:  "collapsible",
						MinNum:     1,
						Panels: []admin.Panel{
							admin.FieldPanel("Product"),
						},
					},
				),
			),
		},
	},
}
