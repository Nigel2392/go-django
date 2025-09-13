package compare

import (
	"context"
	"fmt"
	"html/template"
	"reflect"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/utils/text"
)

func init() {
	RegisterDefaultComparison(FieldComparison)
	RegisterComparisonKind(reflect.String, FieldComparison)
	RegisterComparisonKind(reflect.Int, FieldComparison)
	RegisterComparisonKind(reflect.Float64, FieldComparison)
	RegisterComparisonKind(reflect.Bool, FieldComparison)
	RegisterComparisonKind(reflect.Slice, FieldComparison)
	RegisterComparisonKind(reflect.Map, FieldComparison)
	RegisterComparisonKind(reflect.Struct, FieldComparison)
	RegisterComparisonKind(reflect.Ptr, FieldComparison)
	RegisterComparisonKind(reflect.Interface, FieldComparison)
	RegisterComparisonKind(reflect.Array, FieldComparison)
}

var (
	_ Comparison = (*fieldComparison)(nil)
)

type fieldComparison struct {
	ctx       context.Context
	LabelText any // string or func() string
	MetaField attrs.FieldDefinition
	Old, New  interface{}
}

func FieldComparison(ctx context.Context, label func(context.Context) string, fieldName string, modelMeta attrs.ModelMeta, old, new attrs.Definer) (Comparison, error) {
	var defs = modelMeta.Definitions()
	var field, ok = defs.Field(fieldName)
	if !ok {
		return nil, errors.FieldNotFound.Wrapf(
			"field %q not found in model %T",
			fieldName, modelMeta.Model(),
		)
	}

	var oldValue = attrs.Get[any](old, fieldName)
	var newValue = attrs.Get[any](new, fieldName)
	var fc = &fieldComparison{
		ctx:       ctx,
		LabelText: label,
		MetaField: field,
		Old:       oldValue,
		New:       newValue,
	}

	return fc, nil
}

func (fc *fieldComparison) Label() string {
	var t, ok = trans.GetText(fc.ctx, fc.LabelText)
	if !ok {
		return fc.MetaField.Label(fc.ctx)
	}
	return t
}

func (fc *fieldComparison) HasChanged() (bool, error) {
	var formField = fc.MetaField.FormField()
	return formField.HasChanged(fc.Old, fc.New), nil
}

func (fc *fieldComparison) HTMLDiff() (template.HTML, error) {
	var diff = TextDiff{
		Changes: []Differential{
			{Type: DIFF_REMOVED, Value: fc.Old},
			{Type: DIFF_ADDED, Value: fc.New},
		},
	}
	return diff.HTML(), nil
}

type multipleComparison struct {
	ctx         context.Context
	Comparisons []Comparison
}

func MultipleComparison(ctx context.Context, comparisons ...Comparison) Comparison {
	return &multipleComparison{
		ctx:         ctx,
		Comparisons: comparisons,
	}
}

func (mc *multipleComparison) Unwrap() []Comparison {
	return mc.Comparisons
}

func (mc *multipleComparison) Label() string {
	return ""
}

func (mc *multipleComparison) HasChanged() (bool, error) {
	for _, comp := range mc.Comparisons {
		changed, err := comp.HasChanged()
		if err != nil {
			return false, err
		}
		if changed {
			return true, nil
		}
	}
	return false, nil
}

func (mc *multipleComparison) HTMLDiff() (template.HTML, error) {
	var argList = make([][]any, 0, len(mc.Comparisons))
	for _, comp := range mc.Comparisons {

		changed, err := comp.HasChanged()
		if err != nil {
			return "", err
		}

		if !changed {
			continue
		}

		html, err := comp.HTMLDiff()
		if err != nil {
			return "", err
		}

		argList = append(argList, []any{comp.Label(), html})
	}

	var inner = text.JoinFormat(
		"\n", "<dt>%s</dt><dd>%s</dd>", argList...,
	)

	return template.HTML(fmt.Sprintf("<dl>%s</dl>", inner)), nil
}
