package formfields

import (
	"context"
	"fmt"
	"io"
	"net/url"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/forms/widgets/chooser"
)

var _ widgets.Widget = &ModelSelect{}

type ModelSelect struct {
	*chooser.BaseChooser
	ExcludeBlank bool
	BlankLabel   string
}

func ModelSelectWidget(allowBlank bool, blankLabel string, opts chooser.BaseChooserOptions, attrs map[string]string) *ModelSelect {
	var chooser = chooser.BaseChooserWidget(opts, attrs)
	chooser.BaseWidget.Type = "select"
	chooser.BaseWidget.TemplateName = "forms/widgets/model-select.html"
	return &ModelSelect{
		BaseChooser:  chooser,
		ExcludeBlank: !allowBlank,
		BlankLabel:   blankLabel,
	}
}
func (f *ModelSelect) ValueToForm(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	if attrs.IsZero(value) {
		return nil
	}

	switch v := value.(type) {
	case attrs.Definer:
		var defs = v.FieldDefs()
		var prim = defs.Primary()
		return prim.GetValue()
	default:
		return value
	}
}

func (f *ModelSelect) ValueToGo(value interface{}) (interface{}, error) {

	if _, ok := value.(attrs.Definer); ok {
		return value, nil
	}

	var newObj = attrs.NewObject[attrs.Definer](f.Opts.TargetObject)
	var defs = newObj.FieldDefs()
	var prim = defs.Primary()
	var err = prim.Scan(value)
	if err != nil {
		return nil, err
	}

	return newObj, nil
}

func (o *ModelSelect) GetContextData(ctx context.Context, id, name string, value interface{}, widgetAttrs map[string]string) ctx.Context {
	var base_context = o.BaseWidget.GetContextData(ctx, id, name, value, widgetAttrs)
	var modelInstances, err = o.QuerySet(ctx)
	if err != nil {
		logger.Errorf(
			"error getting model instances for model: %s, %s",
			o.ModelDefinition().Name(),
			err,
		)
		return base_context
	}

	var choices = make([]widgets.Option, 0, len(modelInstances))
	for _, modelInstance := range modelInstances {

		var value = o.Opts.GetPrimaryKey(ctx, modelInstance)
		var labelStr = o.ModelDefinition().InstanceLabel(
			modelInstance,
		)
		var valueStr = fmt.Sprintf("%v", value)
		var option = widgets.NewOption(
			valueStr, labelStr, valueStr,
		)

		choices = append(choices, option)
	}

	var values []string
	if value != nil {
		switch v := value.(type) {
		case string:
			values = []string{v}
		case []string:
			values = v
		default:
			values = []string{fmt.Sprintf("%v", v)}
		}
	}

	base_context.Set(
		"choices",
		widgets.WrapOptions(choices, values),
	)

	base_context.Set("include_blank", !o.ExcludeBlank)
	if o.BlankLabel == "" {
		o.BlankLabel = "---------"
	}
	base_context.Set("blank_label", o.BlankLabel)
	return base_context
}

func (b *ModelSelect) Render(ctx context.Context, w io.Writer, id, name string, value interface{}, attrs map[string]string) error {
	return b.RenderWithErrors(ctx, w, id, name, value, nil, attrs, b.GetContextData(ctx, id, name, value, attrs))
}

type MultiSelectWidget[T attrs.Definer] struct {
	*widgets.BaseWidget
	Queryset     func() *queries.QuerySet[T]
	Relation     attrs.Relation
	FieldDef     attrs.FieldDefinition
	IncludeBlank bool
	BlankLabel   string
}

func (o *MultiSelectWidget[T]) ValueToGo(value interface{}) (interface{}, error) {
	var objects []attrs.Definer
	if value == nil || fields.IsZero(value) {
		return nil, nil
	}

	switch v := value.(type) {
	case []string:
		var meta = attrs.GetModelMeta(o.Relation.Model())
		var metaDefs = meta.Definitions()
		var primary = metaDefs.Primary()
		var qs *queries.QuerySet[T]
		if o.Queryset != nil {
			qs = o.Queryset()
		} else {
			qs = queries.GetQuerySet(o.Relation.Model().(T))
		}
		qs = qs.Filter(
			fmt.Sprintf("%s__in", primary.Name()),
			v,
		)
		rowsCount, rowsIter, err := qs.IterAll()
		if err != nil {
			return nil, err
		}

		objects = make([]attrs.Definer, 0, rowsCount)
		for row, err := range rowsIter {
			if err != nil {
				return nil, err
			}

			objects = append(objects, row.Object)
		}
	case []attrs.Definer:
		objects = v
	default:
		return nil, errors.TypeMismatch.Wrapf(
			"Value %v (%T) is not a []string or []attrs.Definer",
			value, value,
		)
	}
	return objects, nil
}

func (o *MultiSelectWidget[T]) Choices(ctx context.Context) []widgets.Option {
	var choices = make([]widgets.Option, 0)
	var chosen = make([]any, 0)

	var backRef = queries.RelM2M[attrs.Definer, attrs.Definer]{
		Parent: &queries.ParentInfo{
			Object: o.FieldDef.Instance(),
			Field:  o.FieldDef.(attrs.Field),
		},
	}

	var selectedRows, err = backRef.Objects().WithContext(ctx).All()
	if err != nil {
		except.Fail(
			500, "error getting queryset for MultiSelectWidget: %s", err,
		)
		return choices
	}

	for _, row := range selectedRows {
		var value = attrs.PrimaryKey(row.Object)
		var labelStr = attrs.ToString(row.Object)
		var valueStr = fmt.Sprintf("%v", value)
		chosen = append(chosen, value)
		choices = append(choices, &widgets.WrappedOption{
			Option:   widgets.NewOption(labelStr, labelStr, valueStr),
			Selected: true,
		})
	}

	var querySet *queries.QuerySet[T]
	if o.Queryset != nil {
		querySet = o.Queryset()
	} else {
		querySet = queries.GetQuerySet(o.Relation.Model().(T))
	}

	querySet = querySet.WithContext(ctx)

	var meta = attrs.GetModelMeta(o.Relation.Model())
	var metaDefs = meta.Definitions()
	var primary = metaDefs.Primary()

	if len(chosen) > 0 {
		querySet = querySet.Filter(
			expr.Q(fmt.Sprintf("%s__in", primary.Name()), chosen).Not(true),
		)
	}

	unSelected, err := querySet.All()
	if err != nil {
		except.Fail(
			500, "error getting queryset for MultiSelectWidget: %s", err,
		)
		return choices
	}

	for _, row := range unSelected {
		var value = attrs.PrimaryKey(row.Object)
		var labelStr = attrs.ToString(row.Object)
		var valueStr = fmt.Sprintf("%v", value)
		choices = append(choices, &widgets.WrappedOption{
			Option:   widgets.NewOption(labelStr, labelStr, valueStr),
			Selected: false,
		})
	}

	return choices
}

func (o *MultiSelectWidget[T]) GetContextData(ctx context.Context, id, name string, value interface{}, attrs map[string]string) ctx.Context {
	var base_context = o.BaseWidget.GetContextData(ctx, id, name, value, attrs)
	var choices = o.Choices(ctx)

	base_context.Set(
		"choices",
		choices,
	)

	base_context.Set("include_blank", o.IncludeBlank)
	if o.BlankLabel == "" {
		o.BlankLabel = "---------"
	}
	base_context.Set("blank_label", o.BlankLabel)
	return base_context
}

func (o *MultiSelectWidget[T]) Validate(ctx context.Context, value interface{}) []error {
	if value == nil {
		return nil
	}

	var (
		errors  []error
		choices = o.Choices(ctx)
		values  []string
	)

	switch v := value.(type) {
	case string:
		values = []string{v}
	case []string:
		values = v
	}

	if (len(values) == 0 || len(values) == 1 && values[0] == "") && o.IncludeBlank {
		return nil
	}

	var valuesMap = make(map[string]struct{})
	for _, valueStr := range values {
		valuesMap[valueStr] = struct{}{}
	}
	for _, choice := range choices {
		if _, ok := valuesMap[choice.Value()]; !ok {
			errors = append(errors, errs.ErrInvalidValue)
		}
	}

	return errors
}

func (m *MultiSelectWidget[T]) ValueFromDataDict(ctx context.Context, data url.Values, files map[string][]filesystem.FileHeader, name string) (interface{}, []error) {
	var values, ok = data[name]
	if !ok {
		return nil, nil
	}
	return values, nil
}

func (m *MultiSelectWidget[T]) Media() media.Media {
	var formMedia = media.NewMedia()
	formMedia.AddCSS(media.CSS(django.Static(
		"forms/css/model-multiple-select.css",
	)))
	formMedia.AddJS(&media.JSAsset{
		URL: django.Static(
			"forms/js/index.js",
		),
	})
	return formMedia
}
