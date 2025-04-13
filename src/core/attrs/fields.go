package attrs

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/mail"
	"reflect"
	"time"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/forms/widgets/chooser"
	"github.com/Nigel2392/go-django/src/internal/django_reflect"
	"github.com/Nigel2392/goldcrest"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var capCaser = cases.Title(language.English)

// FieldConfig is a configuration for a field.
//
// This defines how a field should behave and how it should be displayed in a form.
type FieldConfig struct {
	Null          bool                                          // Whether the field allows null values
	Blank         bool                                          // Whether the field allows blank values
	ReadOnly      bool                                          // Whether the field is read-only
	Primary       bool                                          // Whether the field is a primary key
	Label         string                                        // The label for the field
	HelpText      string                                        // The help text for the field
	RelForeignKey Definer                                       // The related object for the field (foreign key)
	RelManyToMany Definer                                       // The related objects for the field (many to many, not implemented
	RelOneToOne   Definer                                       // The related object for the field (one to one, not implemented)
	Default       any                                           // The default value for the field (or a function that returns the default value)
	Validators    []func(interface{}) error                     // Validators for the field
	FormField     func(opts ...func(fields.Field)) fields.Field // The form field for the field
	WidgetAttrs   map[string]string                             // The attributes for the widget
	FormWidget    func(FieldConfig) widgets.Widget              // The form widget for the field
	Setter        func(Definer, interface{}) error              // A custom setter for the field
	Getter        func(Definer) (interface{}, bool)             // A custom getter for the field
}

type FieldDef struct {
	attrDef        FieldConfig
	instance_t_ptr reflect.Type
	instance_v_ptr reflect.Value
	instance_t     reflect.Type
	instance_v     reflect.Value
	field_t        reflect.StructField
	field_v        reflect.Value
	formField      fields.Field
	fieldName      string
	// directlyInteractible bool
}

// NewField creates a new field definition for the given instance.
//
// This can then be used for managing the field in a more abstract way.
func NewField[T any](instance *T, name string, conf *FieldConfig) *FieldDef {
	var (
		instance_t_ptr = reflect.TypeOf(instance)
		instance_v_ptr = reflect.ValueOf(instance)
		instance_t     = instance_t_ptr.Elem()
		instance_v     = instance_v_ptr.Elem()
		field_t        reflect.StructField
		field_v        reflect.Value
		ok             bool
	)

	// var isRel = (conf != nil) && (conf.RelForeignKey != nil || conf.RelManyToMany != nil || conf.RelOneToOne != nil)
	field_t, ok = instance_t.FieldByName(name)
	// assert.True(ok || isRel, "field %q not found in %T", name, instance)
	assert.True(ok, "field %q not found in %T", name, instance)

	// var directlyInteractible = ok

	// if ok {
	field_v = instance_v.FieldByIndex(field_t.Index)
	assert.True(field_v.IsValid(), "field %q not found in %T", name, instance)
	// }

	if conf == nil {
		conf = &FieldConfig{}
	}

	return &FieldDef{
		attrDef:        *conf,
		instance_t_ptr: instance_t_ptr,
		instance_v_ptr: instance_v_ptr,
		instance_t:     instance_t,
		instance_v:     instance_v,
		field_t:        field_t,
		field_v:        field_v,
		fieldName:      name,
		// directlyInteractible: directlyInteractible,
	}
}

func (f *FieldDef) Label() string {
	if f.attrDef.Label != "" {
		return trans.T(f.attrDef.Label)
	}

	// if f.directlyInteractible {
	if labeler, ok := f.field_v.Interface().(Labeler); ok {
		return labeler.Label()
	}
	// }

	var rel = f.Rel()
	if rel != nil {
		var cTypeDef = contenttypes.DefinitionForObject(rel)
		if cTypeDef != nil {
			return cTypeDef.Label()
		}
	}

	return trans.T(capCaser.String(f.fieldName))
}

func (f *FieldDef) HelpText() string {
	// if f.directlyInteractible {
	if helpTexter, ok := f.field_v.Interface().(Helper); ok {
		return trans.T(helpTexter.HelpText())
	}
	// }
	if f.attrDef.HelpText != "" {
		return trans.T(f.attrDef.HelpText)
	}
	return ""
}

func (f *FieldDef) Name() string {
	return f.fieldName
}

func (f *FieldDef) Tag(name string) string {
	return f.field_t.Tag.Get(name)
}

func (f *FieldDef) Rel() Definer {
	if f.ForeignKey() != nil {
		return f.ForeignKey()
	}
	if f.ManyToMany() != nil {
		return f.ManyToMany()
	}
	if f.OneToOne() != nil {
		return f.OneToOne()
	}
	return nil
}

func (f *FieldDef) ForeignKey() Definer {
	if f.attrDef.RelForeignKey != nil {
		return f.attrDef.RelForeignKey
	}
	return nil
}

func (f *FieldDef) ManyToMany() Definer {
	if f.attrDef.RelManyToMany != nil {
		return f.attrDef.RelManyToMany
	}
	return nil
}

func (f *FieldDef) OneToOne() Definer {
	if f.attrDef.RelOneToOne != nil {
		return f.attrDef.RelOneToOne
	}
	return nil
}

func (f *FieldDef) IsPrimary() bool {
	return f.attrDef.Primary
}

func (f *FieldDef) AllowNull() bool {
	return f.attrDef.Null
}

func (f *FieldDef) AllowBlank() bool {
	return f.attrDef.Blank
}

func (f *FieldDef) AllowEdit() bool {
	return !f.attrDef.ReadOnly
}

func (f *FieldDef) Validate() error {
	var v = f.GetValue()
	for _, validator := range f.attrDef.Validators {
		if err := validator(v); err != nil {
			return err
		}
	}
	return nil
}

func (f *FieldDef) Instance() Definer {
	return f.instance_v_ptr.Interface().(Definer)
}

func (f *FieldDef) ToString() string {
	var v = f.GetValue()
	if v == nil {
		v = f.GetDefault()
	}

	var funcName = fmt.Sprintf("%sToString", f.Name())
	if method, ok := f.instance_t.MethodByName(funcName); ok {
		var out = method.Func.Call([]reflect.Value{f.instance_v_ptr})
		assert.Gt(out, 0, "Method %q on raw did not return a value", funcName)
		return out[0].String()
	}

	return toString(v)
}

func (f *FieldDef) GetDefault() interface{} {

	if f.attrDef.Default != nil {
		var v = reflect.ValueOf(f.attrDef.Default)
		if v.IsValid() && v.Kind() == reflect.Func {
			var out = v.Call([]reflect.Value{})
			assert.Gt(out, 0, "Default function did not return a value")
			return out[0].Interface()
		}
		return f.attrDef.Default
	}

	var funcName = fmt.Sprintf("GetDefault%s", f.Name())
	if method, ok := f.instance_t.MethodByName(funcName); ok {
		var out = method.Func.Call([]reflect.Value{f.instance_v_ptr})
		assert.Gt(out, 0, "Method %q on ptr did not return a value", funcName)
		return out[0].Interface()
	}

	if method, ok := f.instance_t_ptr.MethodByName(funcName); ok {
		var out = method.Func.Call([]reflect.Value{f.instance_v_ptr})
		assert.Gt(out, 0, "Method %q on raw did not return a value", funcName)
		return out[0].Interface()
	}

	// if f.directlyInteractible {
	var typForNew = f.field_t.Type
	if f.field_t.Type.Kind() == reflect.Ptr {
		typForNew = f.field_t.Type.Elem()
	}

	var hooks = goldcrest.Get[DefaultGetter](DefaultForType)
	for _, hook := range hooks {
		if defaultValue, ok := hook(f, typForNew, f.field_v); ok {
			return defaultValue
		}
	}

	if !f.field_v.IsValid() {
		return reflect.Zero(f.field_t.Type).Interface()
	}

	return f.field_v.Interface()
	// }

	// var rel = f.Rel()
	// if rel != nil {
	// var cTypeDef = contenttypes.DefinitionForObject(rel)
	// if cTypeDef != nil {
	// return cTypeDef.Object()
	// }
	// }
	//
	// return nil
}

func (f *FieldDef) FormField() fields.Field {
	if f.formField != nil {
		return f.formField
	}

	var opts = make([]func(fields.Field), 0)
	var rel = f.Rel()
	if rel != nil {
		var cTypeDef = contenttypes.DefinitionForObject(rel)
		if cTypeDef != nil {
			opts = append(opts, fields.Label(
				cTypeDef.Label(),
			))
		}
	} else {
		opts = append(opts, fields.Label(f.Label))
	}
	opts = append(opts, fields.HelpText(f.HelpText))

	if f.attrDef.ReadOnly {
		opts = append(opts, fields.ReadOnly(true))
	}

	if !f.AllowBlank() {
		opts = append(opts, fields.Required(true))
	}

	var formField fields.Field
	var typForNew reflect.Type = f.field_t.Type
	if f.field_t.Type.Kind() == reflect.Ptr {
		typForNew = f.field_t.Type.Elem()
	}

	var hooks []FormFieldGetter
	if f.attrDef.FormField != nil {
		formField = f.attrDef.FormField(opts...)
		goto returnField
	}

	hooks = goldcrest.Get[FormFieldGetter](HookFormFieldForType)
	for _, hook := range hooks {
		if field, ok := hook(f, typForNew, f.field_v, opts...); ok {
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
	}

	if formField != nil {
		goto returnField
	}

	switch f.field_t.Type.Kind() {
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
	formField.SetName(f.Name())

	if f.attrDef.WidgetAttrs != nil {
		formField.SetAttrs(
			f.attrDef.WidgetAttrs,
		)
	}

	switch {
	case f.attrDef.FormWidget != nil:
		formField.SetWidget(
			f.attrDef.FormWidget(f.attrDef),
		)
	case f.ForeignKey() != nil:
		formField.SetWidget(
			chooser.SelectWidget(
				f.AllowBlank(),
				"--------",
				chooser.BaseChooserOptions{
					TargetObject: f.ForeignKey(),
					GetPrimaryKey: func(i interface{}) interface{} {
						var def = i.(Definer)
						return PrimaryKey(def)
					},
				},
				nil,
			),
		)
	case f.ManyToMany() != nil:

	case f.OneToOne() != nil:

	}

	f.formField = formField
	return formField
}

func (f *FieldDef) GetValue() interface{} {

	if f.attrDef.Getter != nil {
		var v, ok = f.attrDef.Getter(
			f.instance_v_ptr.Interface().(Definer),
		)
		if ok {
			return v
		}
	}

	var (
		b        []byte
		firstArg reflect.Value
		method   reflect.Method
		field    reflect.StructField
		ok       bool
	)
	b = make([]byte, 0, len(f.field_t.Name)+3)
	b = append(b, "Get"...)
	b = append(b, f.field_t.Name...)
	firstArg = f.instance_v
	method, ok = f.instance_t.MethodByName(string(b))
	if !ok {
		method, ok = f.instance_t_ptr.MethodByName(string(b))
		firstArg = f.instance_v_ptr
	}
	if ok {
		return method.Func.Call([]reflect.Value{firstArg})[0].Interface()
	}

	field, ok = f.instance_t.FieldByName(string(b))
	if ok {
		return f.instance_v.FieldByIndex(field.Index).Interface()
	}

	return f.field_v.Interface()
}

func (f *FieldDef) SetValue(v interface{}, force bool) error {

	if f.attrDef.Setter != nil {
		return f.attrDef.Setter(
			f.instance_v_ptr.Interface().(Definer), v,
		)
	}

	var r_v = reflect.ValueOf(v)
	if err := assert.True(
		r_v.IsValid() || f.AllowNull(),
		"field %q (%q) is not valid", f.field_t.Name, f.field_t.Type,
	); err != nil {
		return err
	}

	// Set r_v to zero value if it is nil and field allows null
	if !r_v.IsValid() && f.AllowNull() {
		if f.field_v.Kind() == reflect.Ptr {
			r_v = reflect.New(f.field_t.Type.Elem())
		} else {
			r_v = reflect.Zero(f.field_t.Type)
		}
	}

	if !force {
		// Check if field has a setter
		var b = make([]byte, 0, len(f.field_t.Name)+3)
		b = append(b, "Set"...)
		b = append(b, f.field_t.Name...)
		var method, ok = f.instance_t_ptr.MethodByName(string(b))
		// Call setter if it exists
		if ok {
			var r_v = reflect.ValueOf(v)
			var r_v_ptr, ok = django_reflect.RConvert(&r_v, method.Type.In(1))
			if !ok {
				return assert.Fail(
					fmt.Sprintf("value of type %q is not convertible to %q for field %q",
						r_v.Type(),
						method.Type.In(1),
						f.field_t.Name,
					),
				)
			}
			var out = method.Func.Call([]reflect.Value{f.instance_v_ptr, *r_v_ptr})
			if len(out) > 0 {
				var out = out[len(out)-1].Interface()
				if err, ok := out.(error); ok {
					return err
				}
			}
			return nil
		}
	}

	// Check if field is editable
	if err := assert.True(
		f.field_v.CanSet() && (f.AllowEdit() || force),
		"field %q is not editable", f.field_t.Name,
	); err != nil {
		return err
	}

	// Convert to field type if possible
	r_v_ptr, ok := django_reflect.RConvert(&r_v, f.field_t.Type)
	if !ok {
		scanner, ok := f.field_v.Interface().(Scanner)
		if ok {
			return scanner.ScanAttribute(r_v_ptr.Interface())
		}

		// Try converting the string to a number if necessary
		if r_v.Kind() == reflect.String {
			switch f.field_t.Type.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
				reflect.Float32, reflect.Float64:
				var v = r_v.String()
				if v == "" {
					goto fail
				}

				var n = reflect.New(f.field_t.Type)
				// Scan the value into the field
				if _, err := fmt.Sscan(v, n.Interface()); err != nil {
					goto fail
				}

				// Set the field value
				n = n.Elem()
				*r_v_ptr = n
				goto success
			}
		}

	fail:
		return assert.Fail(
			fmt.Sprintf("value of type %q is not convertible to %q for field %q",
				r_v.Type(),
				f.field_t.Type,
				f.field_t.Name,
			),
		)
	}

success:
	if r_v_ptr.IsZero() && !f.AllowBlank() {
		switch reflect.Indirect(*r_v_ptr).Kind() {
		case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
			reflect.Float32, reflect.Float64,
			reflect.Complex64, reflect.Complex128:
		case reflect.Struct:
			switch reflect.Indirect(*r_v_ptr).Interface().(type) {
			case time.Time, sql.NullBool, sql.NullByte, sql.NullInt16, sql.NullInt32, sql.NullInt64,
				sql.NullString, sql.NullFloat64, sql.NullTime:
			default:
				return assert.Fail(
					fmt.Sprintf("field %q must not be blank", f.field_t.Name),
				)
			}
		default:
			return assert.Fail(
				fmt.Sprintf("field %q must not be blank", f.field_t.Name),
			)
		}
	}

	var old = f.field_v
	django_reflect.RSet(r_v_ptr, &f.field_v, false)

	if err := f.Validate(); err != nil {
		f.field_v.Set(old)
		return err
	}

	return nil
}
