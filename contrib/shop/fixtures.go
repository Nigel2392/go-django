package shop

import (
	"github.com/Nigel2392/go-django/contrib/blocks"
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
		Name:      "microsoft",
		TextLabel: trans.S("Microslop"),
		ProductBlock: func() *blocks.StructBlock {
			block := blocks.NewStructBlock()
			block.AddField("social_sec_num", blocks.CharBlock(
				blocks.WithLabel[*blocks.FieldBlock](trans.S("Social Security Number")),
			))
			block.AddField("agree_to_share", blocks.BooleanBlock(
				blocks.WithLabel[*blocks.FieldBlock](trans.S("Notice")),
				blocks.WithHelpText[*blocks.FieldBlock](trans.S("Do you agree to allow Microslop Inc. to gobble up and share all your data with the great big eye of sauron?")),
			))
			return block
		},
		TextDescription: trans.S("Selling your data to local data farmers"),
	})
	supplier.Register(&supplier.Supplier{
		Name:            "google",
		TextLabel:       trans.S("Gooble"),
		TextDescription: trans.S("In the business of making mildly shitty chatbots"),
	})
}
