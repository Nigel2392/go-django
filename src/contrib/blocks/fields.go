package blocks

import (
	"bytes"
	"context"
	"encoding/json"

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

func (bw *BlockFormField) HasChanged(initial, data interface{}) bool {
	if fields.IsZero(initial) && fields.IsZero(data) {
		return false
	}

	if fields.IsZero(initial) != fields.IsZero(data) {
		return true
	}

	oldJSON, err := json.Marshal(initial)
	if err != nil {
		// If values cannot be serialized consistently, prefer treating it as changed.
		return true
	}

	newJSON, err := json.Marshal(data)
	if err != nil {
		// If values cannot be serialized consistently, prefer treating it as changed.
		return true
	}

	return !bytes.Equal(oldJSON, newJSON)
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

func (bw *BlockFormField) Default() interface{} {
	if bw.BaseField.GetDefault != nil {
		return bw.BaseField.Default()
	}
	return bw.Block.GetDefault()
}
