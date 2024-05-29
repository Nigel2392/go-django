package blocks

import "github.com/Nigel2392/django/forms/fields"

type BlockFormField struct {
	*fields.BaseField
}

func BlockField(block Block, opts ...func(fields.Field)) *BlockFormField {
	var bf = &BlockFormField{
		BaseField: fields.NewField(fields.S("block"), opts...),
	}

	if bf.FormWidget == nil {
		bf.FormWidget = NewBlockWidget(block)
	}

	return bf
}
