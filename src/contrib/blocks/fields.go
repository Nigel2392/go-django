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

	block.SetLabel(bf.FormLabel)
	block.SetHelpText(bf.FormHelpText)

	if bf.FormWidget == nil {
		bf.FormWidget = NewBlockWidget(block)
	}

	//	bf.SetName(block.Name())
	//	bf.SetLabel(block.Label)
	//	bf.SetHelpText(block.HelpText)

	return bf
}

func (bw *BlockFormField) ValueToGo(value interface{}) (interface{}, error) {
	var v, err = bw.Block.ValueToGo(value)
	return v, err
}

func (bw *BlockFormField) ValueToForm(value interface{}) interface{} {
	return bw.Block.ValueToForm(value)

}

func (bw *BlockFormField) Validate(ctx context.Context, value interface{}) []error {
	var errs = bw.Block.Validate(ctx, value)
	return errs
}

func (bw *BlockFormField) Clean(ctx context.Context, value interface{}) (interface{}, error) {
	var v, err = bw.Block.Clean(ctx, value)
	return v, err
}

func (bw *BlockFormField) SetLabel(label func(ctx context.Context) string) {
	bw.BaseField.SetLabel(label)
	bw.Block.SetLabel(label)
}

func (bw *BlockFormField) SetHelpText(helpText func(ctx context.Context) string) {
	bw.BaseField.SetHelpText(helpText)
	bw.Block.SetHelpText(helpText)
}
