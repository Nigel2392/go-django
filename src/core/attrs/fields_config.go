package attrs

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"maps"
	"net/mail"
	"reflect"
	"time"

	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/forms/widgets/options"
	"github.com/Nigel2392/goldcrest"
	"github.com/shopspring/decimal"
)

type Option struct {
	Label trans.Translatable
	Val   any
}

func (o Option) Value() (driver.Value, error) {
	if o.Val == nil {
		return nil, nil
	}

	if val, ok := o.Val.(driver.Valuer); ok {
		return val.Value()
	}

	return o.Val, nil
}

// []Option
// func() []Option
// func(MODEL Definer) []Option
// func(context.Context) []Option
// func(context.Context, MODEL Definer) []Option
// func() ([]Option, error)
// func(MODEL Definer) ([]Option, error)
// func(context.Context) ([]Option, error)
// func(context.Context, MODEL Definer) ([]Option, error)
type OptionsType any

func IsOptionsType(optVal any) (acceptsModel, isOptions bool) {
	if optVal == nil {
		return false, false
	}

	switch optVal.(type) {
	case []Option,
		func() []Option,
		func(context.Context) []Option,
		func() ([]Option, error),
		func(context.Context) ([]Option, error):
		return false, true

	case func(Definer) []Option,
		func(context.Context, Definer) []Option,
		func(Definer) ([]Option, error),
		func(context.Context, Definer) ([]Option, error):
		return true, true
	}

	return false, false

}

func TryOptions(ctx context.Context, optVal any, modelType Definer) (opts []Option, ok bool, err error) {
	if optVal == nil {
		return []Option{}, false, nil
	}

	switch v := optVal.(type) {
	case []Option:
		return v, true, nil

	case func() []Option:
		return v(), true, nil

	case func(Definer) []Option:
		return v(modelType), true, nil

	case func(context.Context) []Option:
		return v(ctx), true, nil

	case func(context.Context, Definer) []Option:
		return v(ctx, modelType), true, nil

	case func() ([]Option, error):
		opts, err = v()
		return opts, true, err

	case func(Definer) ([]Option, error):
		opts, err = v(modelType)
		return opts, true, err

	case func(context.Context) ([]Option, error):
		opts, err = v(ctx)
		return opts, true, err

	case func(context.Context, Definer) ([]Option, error):
		opts, err = v(ctx, modelType)
		return opts, true, err
	}

	return []Option{}, false, nil
}

// Config is a configuration for a field.
//
// This defines the general behaviour of fields, albeit in forms or queries.
type Config struct {
	AutoInit             bool                                          `json:"auto_init,omitempty"`     // Whether the parent struct should be initialized automatically if the field is an embedded field
	Null                 bool                                          `json:"allow_null"`              // Whether the field allows null values
	Blank                bool                                          `json:"allow_blank"`             // Whether the field allows blank values
	ReadOnly             bool                                          `json:"read_only,omitempty"`     // Whether the field is read-only
	Primary              bool                                          `json:"is_primary,omitempty"`    // Whether the field is a primary key
	IsEmbedded           bool                                          `json:"is_embedded,omitempty"`   // Whether the field is an embedded field
	AutoNowAdd           bool                                          `json:"auto_now_add,omitempty"`  // Wether a time value should be automatically populated with time.Now on model creation
	AutoNow              bool                                          `json:"auto_now,omitempty"`      // Wether a time value should be automatically populated with time.Now on model update
	Precision            int                                           `json:"precision,omitempty"`     // Decimal precision
	Scale                int                                           `json:"scale,omitempty"`         // Decimal scale
	NameOverride         string                                        `json:"name_override,omitempty"` // An optional override for the field name
	Label                any                                           `json:"-"`                       // The label for the field
	HelpText             any                                           `json:"-"`                       // The help text for the field
	Column               string                                        `json:"column,omitempty"`        // The name of the column in the database
	MinLength            int64                                         `json:"min_length,omitempty"`    // The minimum length of the field
	MaxLength            int64                                         `json:"max_length,omitempty"`    // The maximum length of the field
	MinValue             float64                                       `json:"min_value,omitempty"`     // The minimum value of the field
	MaxValue             float64                                       `json:"max_value,omitempty"`     // The maximum value of the field
	Attributes           map[string]interface{}                        `json:"attributes,omitempty"`    // The attributes for the field
	ReverseAlias         string                                        `json:"reverse_alias,omitempty"` // The reverse alias for the field
	RelForeignKey        Relation                                      `json:"-"`                       // The related object for the field (foreign key)
	RelManyToMany        Relation                                      `json:"-"`                       // The related objects for the field (many to many, not implemented
	RelOneToOne          Relation                                      `json:"-"`                       // The related object for the field (one to one, not implemented)
	RelForeignKeyReverse Relation                                      `json:"-"`                       // The reverse foreign key for the field (not implemented)
	Default              any                                           `json:"default,omitempty"`       // The default value for the field (or a function that takes in the object type and returns the default value)
	Options              OptionsType                                   `json:"-"`                       // options for the field
	WidgetAttrs          map[string]string                             `json:"widgetAttrs,omitempty"`   // The attributes for the widget
	Validators           []func(interface{}) error                     `json:"-"`                       // Validators for the field
	FormField            func(opts ...func(fields.Field)) fields.Field `json:"-"`                       // The form field for the field
	FormWidget           func(Config) widgets.Widget                   `json:"-"`                       // The form widget for the field
	OnInit               func(Definer, *FieldDef, *Config) *Config     `json:"-"`                       // A function that is called when the field is initialized
}

func (f *Config) FormOptions[FIELD Field](field FIELD) []func(fields.Field) {
	var opts = make([]func(fields.Field), 0)
	var rel = field.Rel()
	if rel != nil {
		var cTypeDef = contenttypes.DefinitionForObject(rel)
		if cTypeDef != nil {
			opts = append(opts, fields.Label(
				cTypeDef.Label,
			))
		}
	} else {
		opts = append(opts, fields.Label(field.Label))
	}

	opts = append(opts, fields.HelpText(field.HelpText))

	if f.ReadOnly {
		opts = append(opts, fields.ReadOnly(true))
	}

	if !field.AllowBlank() {
		opts = append(opts, fields.Required(true))
	}

	if f.MinLength > 0 {
		opts = append(opts, fields.MinLength(int(f.MinLength)))
	}

	if f.MaxLength > 0 {
		opts = append(opts, fields.MaxLength(int(f.MaxLength)))
	}

	if f.MinValue > 0 || f.MinValue < 0 {
		opts = append(opts, fields.MinValue(int(f.MinValue)))
	}

	if f.MaxValue > 0 || f.MaxValue < 0 {
		opts = append(opts, fields.MaxValue(int(f.MaxValue)))
	}

	return opts
}

func (f *Config) GetFormField[FIELD Field](field FIELD, value reflect.Value, opts ...func(fields.Field)) fields.Field {
	var (
		formField fields.Field
		origTyp   = field.Type()
		typForNew = origTyp
	)

	if typForNew.Kind() == reflect.Ptr {
		typForNew = typForNew.Elem()
	}

	var hooks []FormFieldGetter
	if f.FormField != nil {
		formField = f.FormField(opts...)
		goto returnField
	}

	hooks = goldcrest.Get[FormFieldGetter](HookFormFieldForType)
	for _, hook := range hooks {
		if field, ok := hook(field, typForNew, value, opts...); ok {
			formField = field
			goto returnField
		}
	}

	switch reflect.New(typForNew).Elem().Interface().(type) {
	case time.Time:
		formField = fields.DateField(widgets.DateWidgetTypeDateTime, opts...)
	case json.RawMessage:
		formField = fields.JSONField[map[string]interface{}](opts...)
	case mail.Address:
		formField = fields.EmailField(opts...)
	case decimal.Decimal:
		formField = fields.DecimalField(opts...)
	case sql.NullBool:
		formField = fields.SQLNullField[bool](opts...)
	case sql.NullByte:
		formField = fields.SQLNullField[byte](opts...)
	case sql.NullInt16:
		formField = fields.SQLNullField[int16](opts...)
	case sql.NullInt32:
		formField = fields.SQLNullField[int32](opts...)
	case sql.NullInt64:
		formField = fields.SQLNullField[int64](opts...)
	case sql.NullString:
		formField = fields.SQLNullField[string](opts...)
	case sql.NullFloat64:
		formField = fields.SQLNullField[float64](opts...)
	case sql.NullTime:
		formField = fields.SQLNullField[time.Time](opts...)
	case Definer:
		formField = newDefinerField(field, opts...)
	}

	if formField != nil {
		goto returnField
	}

	switch reflect.New(typForNew).Interface().(type) {
	case *time.Time:
		formField = fields.DateField(widgets.DateWidgetTypeDateTime, opts...)
	case *mail.Address:
		formField = fields.EmailField(opts...)
	case Definer:
		formField = newDefinerField(field, opts...)
	}

	if formField != nil {
		goto returnField
	}

	switch origTyp.Kind() {
	case reflect.String:
		formField = fields.CharField(opts...)
	case reflect.Bool:
		formField = fields.BooleanField(opts...)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		formField = fields.NumberField[int](opts...)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		formField = fields.NumberField[uint](opts...)
	case reflect.Float32, reflect.Float64:
		formField = fields.NumberField[float64](opts...)
	default:
		formField = fields.CharField(opts...)
	}

returnField:
	if formField == nil {
		return nil
	}

	formField.SetName(field.Name())

	if f.WidgetAttrs != nil {
		formField.SetAttrs(
			f.WidgetAttrs,
		)
	}

	_, isOpts := IsOptionsType(f.Options)
	switch {
	case f.FormWidget != nil:
		formField.SetWidget(
			f.FormWidget(*f),
		)
	case isOpts:
		var widgetOpts []func(*options.OptionsWidget)

		if f.Blank {
			widgetOpts = append(
				widgetOpts,
				options.IncludeBlank(true),
			)
		}

		getWidgetOptions := func(ctx context.Context) []widgets.Option {
			opts, _, err := TryOptions(ctx, f.Options, field.Instance())
			if err != nil {
				logger.Errorf("could not retrieve options for %q in %T", field.Name(), field.Instance())
				return []widgets.Option{}
			}

			wopts := make([]widgets.Option, 0, len(opts))
			for _, opt := range opts {
				var l any
				var ok bool

				l, ok = trans.GetText(ctx, opt.Label)
				if !ok {
					l = opt.Label
				}

				v, err := opt.Value()
				if err != nil {
					logger.Errorf(
						"could not retrieve options raw value for %q in %T (%v)",
						field.Name(), field.Instance(), l,
					)
					return []widgets.Option{}
				}

				wopts = append(wopts, &widgets.FormOption{
					OptLabel: l,
					OptValue: fmt.Sprint(v),
				})
			}

			return wopts
		}

		formField.SetWidget(options.NewSelectInput(
			nil, getWidgetOptions,
			widgetOpts...,
		))
	}

	return formField
}

func (f *Config) Attrs() map[string]interface{} {
	attrs := make(map[string]interface{})
	attrs[AttrMaxLengthKey] = f.MaxLength
	attrs[AttrMinLengthKey] = f.MinLength
	attrs[AttrMinValueKey] = f.MinValue
	attrs[AttrMaxValueKey] = f.MaxValue
	attrs[AttrAutoNowAddKey] = f.AutoNowAdd
	attrs[AttrAutoNowKey] = f.AutoNow
	attrs[AttrPrecisionKey] = f.Precision
	attrs[AttrScaleKey] = f.Scale
	attrs[AttrReverseAliasKey] = f.ReverseAlias
	attrs[AttrAllowNullKey] = f.Null
	attrs[AttrAllowBlankKey] = f.Blank
	attrs[AttrAllowEditKey] = !f.ReadOnly
	attrs[AttrIsPrimaryKey] = f.Primary
	maps.Copy(attrs, f.Attributes)
	return attrs
}

func (f *Config) IsPrimary() bool {
	return f.Primary
}

func (f *Config) AllowNull() bool {
	return f.Null
}

func (f *Config) AllowBlank() bool {
	return f.Blank
}

func (f *Config) AllowEdit() bool {
	return !f.ReadOnly
}

func (f *Config) Embedded() bool {
	return f.IsEmbedded
}
