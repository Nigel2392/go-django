package compare

import (
	"context"
	"html/template"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
)

var (
	_ Comparison = (*FieldComparison)(nil)
)

type FieldComparison struct {
	LabelText any // string or func(ctx context.Context) string
	MetaField attrs.FieldDefinition
	Old, New  interface{}
}

func (fc *FieldComparison) Label(ctx context.Context) string {
	var t, ok = trans.GetText(ctx, fc.LabelText)
	if !ok {
		return fc.MetaField.Label(ctx)
	}
	return t
}

func (fc *FieldComparison) HasChanged(ctx context.Context) (bool, error) {
	var formField = fc.MetaField.FormField()
	return formField.HasChanged(fc.Old, fc.New), nil
}

func (fc *FieldComparison) HTMLDiff(ctx context.Context) (template.HTML, error) {
	var diff = TextDiff{
		Changes: []Differential{
			{Type: DIFF_REMOVED, Value: fc.Old},
			{Type: DIFF_ADDED, Value: fc.New},
		},
	}
	return diff.HTML(), nil
}
