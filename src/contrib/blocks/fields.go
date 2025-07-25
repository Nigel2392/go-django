package blocks

import (
	"context"

	"github.com/Nigel2392/go-django/src/forms/fields"
)

type BlockFormField struct {
	*fields.BaseField
	Block Block
}

func BlockField(block Block, opts ...func(fields.Field)) *BlockFormField {
	var bf = &BlockFormField{
		BaseField: fields.NewField(opts...),
		Block:     block,
	}

	if bf.FormWidget == nil {
		bf.FormWidget = NewBlockWidget(block)
	}

	bf.FormLabel = block.Label
	bf.FormHelpText = block.HelpText

	return bf
}

func (bw *BlockFormField) ValueToGo(value interface{}) (interface{}, error) {
	return bw.Block.ValueToGo(value)
}

func (bw *BlockFormField) ValueToForm(value interface{}) interface{} {
	return bw.Block.ValueToForm(value)
}

func (bw *BlockFormField) Validate(ctx context.Context, value interface{}) []error {
	return bw.Block.Validate(ctx, value)
}

func (bw *BlockFormField) Clean(ctx context.Context, value interface{}) (interface{}, error) {
	return bw.Block.Clean(ctx, value)
}
