package chooser

import (
	"reflect"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

var _ widgets.Widget = &BaseChooser{}

type BaseChooserOptions struct {
	TargetObject  interface{}
	GetPrimaryKey func(interface{}) interface{}
	Queryset      func() ([]interface{}, error)
}

type BaseChooser struct {
	*widgets.BaseWidget
	Opts               BaseChooserOptions
	forModelDefinition *contenttypes.ContentTypeDefinition
}

func BaseChooserWidget(opts BaseChooserOptions, attrs map[string]string) *BaseChooser {
	var def = contenttypes.DefinitionForObject(opts.TargetObject)
	assert.True(
		def != nil,
		"content type definition not found for model: %T",
		opts.TargetObject,
	)

	return &BaseChooser{
		BaseWidget: widgets.NewBaseWidget(
			"chooser",
			"forms/widgets/select.html",
			attrs,
		),
		Opts:               opts,
		forModelDefinition: def,
	}
}

func (o *BaseChooser) QuerySet() ([]interface{}, error) {
	if o.Opts.Queryset != nil {
		return o.Opts.Queryset()
	}
	return o.forModelDefinition.Instances(1000, 0)
}

func (o *BaseChooser) Validate(value interface{}) []error {
	if value == nil {
		return nil
	}

	var (
		errors []error
		rType  = reflect.TypeOf(value)
		rVal   = reflect.ValueOf(value)
	)

	if rType.Kind() == reflect.Ptr {
		rType = rType.Elem()
		rVal = rVal.Elem()
	}

	switch rType.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < rVal.Len(); i++ {
			var val = rVal.Index(i).Interface()
			errors = append(errors, o.validateValue(val)...)
		}
	default:
		errors = append(errors, o.validateValue(value)...)
	}

	return errors
}

func (o *BaseChooser) validateValue(value interface{}) []error {
	var errors []error
	var modelInstance, err = o.forModelDefinition.Instance(
		value,
	)
	if err != nil {
		errors = append(
			errors,
			errs.Wrap(err, "error retrieving model instance"),
		)
	}

	if modelInstance == nil {
		errors = append(
			errors,
			errs.Error("model instance not found"),
		)
	}

	return errors
}
