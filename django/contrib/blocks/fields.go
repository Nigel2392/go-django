package blocks

import "github.com/Nigel2392/django/forms/fields"

type BlockFormField struct {
	*fields.BaseField
	Block Block
}

func BlockField(block Block, opts ...func(fields.Field)) *BlockFormField {
	var bf = &BlockFormField{
		BaseField: fields.NewField(fields.S("block"), opts...),
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

func (bw *BlockFormField) Validate(value interface{}) []error {
	return bw.Block.Validate(value)
}

func (bw *BlockFormField) Clean(value interface{}) (interface{}, error) {
	return bw.Block.Clean(value)
}
