package attrs

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net/mail"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs/attrutils"
	"github.com/Nigel2392/go-django/src/core/checks"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/internal/django_reflect"
	"github.com/Nigel2392/goldcrest"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	capCaser            = cases.Title(language.English)
	ALLOW_METHOD_CHECKS = false // Whether to allow method checks for getters and setters

	_ Field = (*FieldDef)(nil) // Ensure FieldDef implements the Field interface
)

// FieldConfig is a configuration for a field.
//
// This defines how a field should behave and how it should be displayed in a form.
type FieldConfig struct {
	AutoInit             bool                                                // Whether the parent struct should be initialized automatically if the field is an embedded field
	Null                 bool                                                // Whether the field allows null values
	Blank                bool                                                // Whether the field allows blank values
	ReadOnly             bool                                                // Whether the field is read-only
	Primary              bool                                                // Whether the field is a primary key
	Embedded             bool                                                // Whether the field is an embedded field
	NameOverride         string                                              // An optional override for the field name
	Label                any                                                 // The label for the field
	HelpText             any                                                 // The help text for the field
	Column               string                                              // The name of the column in the database
	MinLength            int64                                               // The minimum length of the field
	MaxLength            int64                                               // The maximum length of the field
	MinValue             float64                                             // The minimum value of the field
	MaxValue             float64                                             // The maximum value of the field
	Attributes           map[string]interface{}                              // The attributes for the field
	RelForeignKey        Relation                                            // The related object for the field (foreign key)
	RelManyToMany        Relation                                            // The related objects for the field (many to many, not implemented
	RelOneToOne          Relation                                            // The related object for the field (one to one, not implemented)
	RelForeignKeyReverse Relation                                            // The reverse foreign key for the field (not implemented)
	Default              any                                                 // The default value for the field (or a function that takes in the object type and returns the default value)
	Validators           []func(interface{}) error                           // Validators for the field
	FormField            func(opts ...func(fields.Field)) fields.Field       // The form field for the field
	WidgetAttrs          map[string]string                                   // The attributes for the widget
	FormWidget           func(FieldConfig) widgets.Widget                    // The form widget for the field
	Setter               func(Definer, interface{}) error                    // A custom setter for the field
	Getter               func(Definer) (interface{}, bool)                   // A custom getter for the field
	OnInit               func(Definer, *FieldDef, *FieldConfig) *FieldConfig // A function that is called when the field is initialized
}

type FieldDef struct {
	defs           Definitions
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
func NewField(instance any, name string, conf ...*FieldConfig) *FieldDef {
	var (
		instance_t_ptr = reflect.TypeOf(instance)
		instance_v_ptr = reflect.ValueOf(instance)
		instance_t     = instance_t_ptr.Elem()
		instance_v     = instance_v_ptr.Elem()
	)

	var field_t, ok = attrutils.GetStructField(instance_t, name)
	assert.True(ok, "field %q not found in %T", name, instance)

	// var directlyInteractible = ok
	var cnf = &FieldConfig{}
	if len(conf) == 0 {
		var c = autoDefinitionStructTag(field_t)
		cnf = &c
	}
	if len(conf) > 0 && conf[0] != nil {
		cnf = conf[0]
	}

	// setupFieldValue:
	// make sure we can access the field
	var field_v = instance_v.Field(field_t.Index[0])
	var curr_t = field_v.Type()
	for i := 1; i < len(field_t.Index); i++ {
		var isNil = false
		if field_v.Kind() == reflect.Ptr {
			isNil = field_v.IsNil()
			if isNil {
				if !cnf.AutoInit {
					assert.Fail("field %q is nil (%s) and cannot be accessed", name, curr_t.Elem().Name())
				}
				isNil = false

				var newVal reflect.Value
				switch curr_t.Kind() {
				case reflect.Slice, reflect.Array:
					newVal = reflect.MakeSlice(curr_t, 0, 0)
				case reflect.Map:
					newVal = reflect.MakeMap(curr_t)
				case reflect.Interface:
					newVal = reflect.New(curr_t.Elem())
				default:
					newVal = reflect.New(curr_t.Elem())
				}

				if curr_t.Implements(reflect.TypeOf((*Definer)(nil)).Elem()) {
					newVal = reflect.ValueOf(
						NewObject[Definer](curr_t),
					)
				}

				field_v.Set(newVal)
			}

			field_v = field_v.Elem()
		}

		if !isNil {
			var ptr = field_v
			// ok because struct fields are always addressable
			if field_t.Type.Kind() != reflect.Ptr {
				ptr = field_v.Addr()
			}

			var iFace = ptr.Interface()
			if embedded, ok := iFace.(Embedded); ok {
				embedded.BindToEmbedder(instance.(Definer))
			}
		}

		field_v = field_v.Field(field_t.Index[i])
		curr_t = field_v.Type()
	}
	assert.True(field_v.IsValid(), "field %q not found in %T", name, instance)

	var f = &FieldDef{
		attrDef:        *cnf,
		instance_t_ptr: instance_t_ptr,
		instance_v_ptr: instance_v_ptr,
		instance_t:     instance_t,
		instance_v:     instance_v,
		field_t:        field_t,
		field_v:        field_v,
		fieldName:      name,
		// directlyInteractible: directlyInteractible,
	}

	if field_v.IsValid() && (field_t.Type.Kind() == reflect.Pointer && !field_v.IsNil()) {
		if err := BindValueToModel(instance.(Definer), f, field_v); err != nil {
			assert.Fail("failed to bind value to model: %v", err)
		}
	}

	if cnf.OnInit != nil {
		cnf = cnf.OnInit(
			any(instance).(Definer), f, cnf,
		)
		if cnf != nil {
			f.attrDef = *cnf
		}
	}

	return f
}

func (f *FieldDef) signalChanges(value interface{}) {
	if f.defs == nil {
		return
	}
	f.defs.SignalChange(f, value)
}

// Check checks if the field is valid and can be used.
func (f *FieldDef) Check(ctx context.Context) []checks.Message {
	var messages []checks.Message

	if f.attrDef.MaxLength < 0 || f.attrDef.MaxLength < f.attrDef.MinLength && f.attrDef.MaxLength > 0 {
		var messageText = fmt.Sprintf(
			"Field \"%s\" in model \"%T\" has an invalid max_length (%d)",
			f.Name(), f.instance_v_ptr.Interface(), f.attrDef.MaxLength,
		)
		messages = append(messages, checks.Error(
			"field.invalid_max_length",
			messageText,
			f, "",
		))
	}

	if f.attrDef.MinLength < 0 {
		var messageText = fmt.Sprintf(
			"Field \"%s\" in model \"%T\" has an invalid min_length (%d).",
			f.Name(), f.instance_v_ptr.Interface(), f.attrDef.MinLength,
		)
		messages = append(messages, checks.Error(
			"field.invalid_min_length",
			messageText,
			f, "",
		))
	}

	if f.attrDef.MaxLength > 0 && f.attrDef.MinLength > 0 && f.attrDef.MaxLength < f.attrDef.MinLength {
		var messageText = fmt.Sprintf(
			"Field \"%s\" in model \"%T\" has an invalid max_length (%d) less than min_length (%d).",
			f.Name(), f.instance_v_ptr.Interface(), f.attrDef.MaxLength, f.attrDef.MinLength,
		)
		messages = append(messages, checks.Error(
			"field.invalid_length",
			messageText,
			f, "",
		))
	}

	if f.attrDef.MaxValue < f.attrDef.MinValue && f.attrDef.MaxValue > 0 {
		var messageText = fmt.Sprintf(
			"Field \"%s\" in model \"%T\" has an invalid max_value (%f) less than min_value (%f).",
			f.Name(), f.instance_v_ptr.Interface(), f.attrDef.MaxValue, f.attrDef.MinValue,
		)
		messages = append(messages, checks.Error(
			"field.invalid_value",
			messageText,
			f, "",
		))
	}

	return messages
}

// model is equal to instance_t
func (f *FieldDef) OnModelRegister(model Definer) error {

	attrutils.AddStructField(f.instance_t_ptr, f.fieldName, f.field_t)

	if ALLOW_METHOD_CHECKS {
		var (
			defaultMethodName  = nameGetDefault(f)
			setValueMethodName = nameSetValue(f)
			getValueMethodName = nameGetValue(f)
		)

		if getDefaultMethod, ok := f.instance_t.MethodByName(defaultMethodName); ok {
			attrutils.AddStructMethod(f.instance_t_ptr, defaultMethodName, getDefaultMethod)
		}

		if setValueMethod, ok := f.instance_t.MethodByName(setValueMethodName); ok {
			attrutils.AddStructMethod(f.instance_t_ptr, setValueMethodName, setValueMethod)
		}

		if getValueMethod, ok := f.instance_t.MethodByName(getValueMethodName); ok {
			attrutils.AddStructMethod(f.instance_t_ptr, getValueMethodName, getValueMethod)
		}
	}

	return nil
}

func (f *FieldDef) BindToDefinitions(defs Definitions) {
	// allow for rebinding until i see a reason not to...
	//	assert.True(
	//		f.defs == nil,
	//		"Definitions for field %q (%T) are already set, the field was bound to the model multiple times",
	//		f.field_t.Name, f.field_v.Interface(),
	//	)
	f.defs = defs
}

func (f *FieldDef) FieldDefinitions() Definitions {
	assert.False(
		f.defs == nil,
		"Definitions for field %q (%T) are not set, the field was not properly bound to the model",
		f.field_t.Name, f.field_v.Interface(),
	)
	return f.defs
}

func (f *FieldDef) Type() reflect.Type {
	return f.field_t.Type
}

func (f *FieldDef) Label(ctx context.Context) string {
	if f.attrDef.Label != nil {
		switch label := f.attrDef.Label.(type) {
		case string:
			return trans.T(ctx, label)
		case func(ctx context.Context) string:
			return label(ctx)
		default:
			panic(fmt.Sprintf(
				"Label for field %q (%T) is not a `string` or `function(context) string`, got %T",
				f.field_t.Name, f.field_v.Interface(), label,
			))
		}
	}

	// if f.directlyInteractible {
	if labeler, ok := f.field_v.Interface().(Labeler); ok {
		return labeler.Label(ctx)
	}
	// }

	var rel = f.Rel()
	if rel != nil {
		var cTypeDef = contenttypes.DefinitionForObject(rel)
		if cTypeDef != nil {
			return cTypeDef.Label(ctx)
		}
	}

	return trans.T(ctx, capCaser.String(f.Name()))
}

func (f *FieldDef) HelpText(ctx context.Context) string {
	if helpTexter, ok := f.field_v.Interface().(Helper); ok {
		return helpTexter.HelpText(ctx)
	}

	if f.attrDef.HelpText != nil {
		switch helpText := f.attrDef.HelpText.(type) {
		case string:
			return trans.T(ctx, helpText)
		case func(ctx context.Context) string:
			return helpText(ctx)
		default:
			panic(fmt.Sprintf(
				"HelpText for field %q (%T) is not a `string` or `function(context) string`, got %T",
				f.field_t.Name, f.field_v.Interface(), helpText,
			))
		}
	}

	return ""
}

func (f *FieldDef) Name() string {
	if f.attrDef.NameOverride != "" {
		return f.attrDef.NameOverride
	}
	return f.fieldName
}

func (f *FieldDef) TypeString() string {
	return fmt.Sprintf("%s.%s", f.instance_t.Name(), f.field_t.Type.Name())
}

func (f *FieldDef) Tag(name string) string {
	return f.field_t.Tag.Get(name)
}

func (f *FieldDef) Attrs() map[string]interface{} {
	var attrs = f.attrDef.Attributes
	if attrs == nil {
		attrs = make(map[string]interface{})
	}
	attrs[AttrNameKey] = f.Name()
	attrs[AttrMaxLengthKey] = f.attrDef.MaxLength
	attrs[AttrMinLengthKey] = f.attrDef.MinLength
	attrs[AttrMinValueKey] = f.attrDef.MinValue
	attrs[AttrMaxValueKey] = f.attrDef.MaxValue
	attrs[AttrAllowNullKey] = f.AllowNull()
	attrs[AttrAllowBlankKey] = f.AllowBlank()
	attrs[AttrAllowEditKey] = f.AllowEdit()
	attrs[AttrIsPrimaryKey] = f.IsPrimary()
	return attrs
}

func (f *FieldDef) ColumnName() string {
	if f.attrDef.Column != "" {
		return f.attrDef.Column
	}

	return ColumnName(f.fieldName)
}

func relFromConfig[T FieldDefinition](f T, cnf *FieldConfig) Relation {
	var (
		rel Relation
		typ RelationType
	)

	switch {
	case cnf.RelForeignKey != nil:
		rel = cnf.RelForeignKey
		typ = RelManyToOne
	case cnf.RelManyToMany != nil:
		rel = cnf.RelManyToMany
		typ = RelManyToMany
	case cnf.RelOneToOne != nil:
		rel = cnf.RelOneToOne
		typ = RelOneToOne
	case cnf.RelForeignKeyReverse != nil:
		rel = cnf.RelForeignKeyReverse
		typ = RelOneToMany
	}

	if rel != nil {
		return &typedRelation{
			typ: typ,
			from: &relationTarget{
				model: f.Instance(),
				field: f,
			},
			Relation: rel,
		}
	}

	return nil
}

func (f *FieldDef) Rel() Relation {
	return relFromConfig(f, &f.attrDef)
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

func (f *FieldDef) Embedded() bool {
	return f.attrDef.Embedded
}

func (f *FieldDef) Validate() error {
	var v = f.GetValue()

	var rV = reflect.ValueOf(v)
	if f.attrDef.MinLength > 0 || f.attrDef.MaxLength > 0 {
		if slices.Contains([]reflect.Kind{reflect.String, reflect.Slice, reflect.Array, reflect.Map}, rV.Kind()) {

			if f.attrDef.MinLength > 0 && rV.Len() < int(f.attrDef.MinLength) {
				return fmt.Errorf(
					"field %q must be at least %d characters long, got %d",
					f.field_t.Name, f.attrDef.MinLength, rV.Len(),
				)
			}

			if f.attrDef.MaxLength > 0 && rV.Len() > int(f.attrDef.MaxLength) {
				return fmt.Errorf(
					"field %q must be at most %d characters long, got %d",
					f.field_t.Name, f.attrDef.MaxLength, rV.Len(),
				)
			}
		}
	}

	if f.attrDef.MinValue > 0 || f.attrDef.MinValue < 0 || f.attrDef.MaxValue > 0 || f.attrDef.MaxValue < 0 {
		if slices.Contains([]reflect.Kind{reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64}, rV.Kind()) {
			var num, err = attrutils.CastToNumber[float64](v)
			if err != nil {
				return fmt.Errorf(
					"field %q is not a number: %w",
					f.field_t.Name, err,
				)
			}
			if f.attrDef.MinValue > 0 && num < f.attrDef.MinValue {
				return fmt.Errorf(
					"field %q must be at least %f",
					f.field_t.Name, f.attrDef.MinValue,
				)
			}

			if f.attrDef.MaxValue > 0 && num > f.attrDef.MaxValue {
				return fmt.Errorf(
					"field %q must be at most %f",
					f.field_t.Name, f.attrDef.MaxValue,
				)
			}

			if f.attrDef.MinValue < 0 && num > f.attrDef.MinValue {
				return fmt.Errorf(
					"field %q must be at least %f",
					f.field_t.Name, f.attrDef.MinValue,
				)
			}

			if f.attrDef.MaxValue < 0 && num < f.attrDef.MaxValue {
				return fmt.Errorf(
					"field %q must be at most %f",
					f.field_t.Name, f.attrDef.MaxValue,
				)
			}
		}
	}

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

	var rv = reflect.ValueOf(v)
	return toString(rv, v)
}

func (f *FieldDef) GetDefault() interface{} {

	if f.attrDef.Default != nil {
		var v = reflect.ValueOf(f.attrDef.Default)
		if v.IsValid() && v.Kind() == reflect.Func {
			var out = v.Call([]reflect.Value{f.instance_v_ptr})
			assert.Gt(out, 0, "Default function did not return a value")
			var outVal = out[0].Interface()
			assert.Err(BindValueToModel(
				f.Instance(), f, outVal,
			))
			return outVal
		}

		var outVal = f.attrDef.Default
		assert.Err(BindValueToModel(
			f.Instance(), f, outVal,
		))
		return outVal
	}

	if ALLOW_METHOD_CHECKS {
		var funcName = nameGetDefault(f)
		var method, ok = attrutils.GetStructMethod(f.instance_t_ptr, funcName)
		if ok {
			var out []reflect.Value
			switch method.Type.In(0) {
			case f.instance_t_ptr:
				out = method.Func.Call([]reflect.Value{f.instance_v_ptr})
			case f.instance_t:
				out = method.Func.Call([]reflect.Value{f.instance_v})
			}
			assert.Gt(out, 0, "Method %q on raw did not return a value", funcName)
			var outVal = out[0].Interface()
			assert.Err(BindValueToModel(
				f.Instance(), f, outVal,
			))
			return outVal
		}
	}

	// if f.directlyInteractible {
	var typForNew = f.field_t.Type
	if f.field_t.Type.Kind() == reflect.Ptr {
		typForNew = f.field_t.Type.Elem()
	}

	var hooks = goldcrest.Get[DefaultGetter](DefaultForType)
	for _, hook := range hooks {
		if defaultValue, ok := hook(f, typForNew, f.field_v); ok {
			assert.Err(BindValueToModel(
				f.Instance(), f, defaultValue,
			))
			return defaultValue
		}
	}

	if !f.field_v.IsValid() {
		var zeroVal = reflect.Zero(f.field_t.Type).Interface()
		assert.Err(BindValueToModel(
			f.Instance(), f, zeroVal,
		))
		return zeroVal
	}

	var outVal = f.field_v.Interface()
	assert.Err(BindValueToModel(
		f.Instance(), f, outVal,
	))
	return outVal
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
				cTypeDef.Label,
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

	if f.attrDef.MinLength > 0 {
		opts = append(opts, fields.MinLength(int(f.attrDef.MinLength)))
	}

	if f.attrDef.MaxLength > 0 {
		opts = append(opts, fields.MaxLength(int(f.attrDef.MaxLength)))
	}

	if f.attrDef.MinValue > 0 || f.attrDef.MinValue < 0 {
		opts = append(opts, fields.MinValue(int(f.attrDef.MinValue)))
	}

	if f.attrDef.MaxValue > 0 || f.attrDef.MaxValue < 0 {
		opts = append(opts, fields.MaxValue(int(f.attrDef.MaxValue)))
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
	case Definer:
		formField = newDefinerField(f, opts...)
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
		formField = newDefinerField(f, opts...)
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
			assert.Err(BindValueToModel(
				f.Instance(), f, v,
			))
			return v
		}
	}

	if ALLOW_METHOD_CHECKS {
		var methodName = nameGetValue(f)
		var method, ok = attrutils.GetStructMethod(f.instance_t_ptr, methodName)
		if ok {
			var outVal any
			switch method.Type.In(0) {
			case f.instance_t_ptr:
				outVal = method.Func.Call([]reflect.Value{f.instance_v_ptr})[0].Interface()
			case f.instance_t:
				outVal = method.Func.Call([]reflect.Value{f.instance_v})[0].Interface()
			}
			assert.Err(BindValueToModel(
				f.Instance(), f, outVal,
			))
			return outVal
		}
	}

	var outVal = f.field_v.Interface()
	assert.Err(BindValueToModel(
		f.Instance(), f, outVal,
	))
	return outVal
}

func (f *FieldDef) SetValue(v interface{}, force bool) error {
	if f.attrDef.Setter != nil {
		return f.attrDef.Setter(f.instance_v_ptr.Interface().(Definer), v)
	}

	defer func() {
		assert.Err(BindValueToModel(
			f.Instance(), f, f.field_v.Interface(),
		))
		if !force {
			f.signalChanges(f.field_v.Interface())
		}
	}()

	rv := reflect.ValueOf(v)

	if !rv.IsValid() {
		if !f.AllowNull() {
			return assert.Fail("field %q (%q) is not valid", f.field_t.Name, f.field_t.Type)
		}
		rv = reflect.Zero(f.field_t.Type)
	}

	// Try user-defined setter method like Set<FieldName>
	if ALLOW_METHOD_CHECKS && !force {
		var setterName = nameSetValue(f)
		var method, ok = attrutils.GetStructMethod(f.instance_t_ptr, setterName)
		if ok {
			var arg, ok = django_reflect.RConvert(&rv, method.Type.In(1))
			assert.True(ok,
				"value %q not convertible to %q for field %q",
				rv.Type(), method.Type.In(1), f.field_t.Name,
			)
			var out = method.Func.Call([]reflect.Value{f.instance_v_ptr, *arg})
			if len(out) > 0 {
				if err, ok := out[len(out)-1].Interface().(error); ok {
					return err
				}
				return assert.Fail("setter returned a non-error value: %v", out)
			}
			return nil
		}
	}

	if !f.field_v.CanSet() || (!f.AllowEdit() && !force) {
		return assert.Fail("field %q is not editable", f.field_t.Name)
	}

	rvPtr, ok := django_reflect.RConvert(&rv, f.field_t.Type)
	if !ok {
		if scanner, ok := f.field_v.Interface().(Scanner); ok {

			if f.field_v.Kind() == reflect.Ptr && f.field_v.IsNil() {
				var newVal = reflect.New(f.field_t.Type.Elem())
				f.field_v.Set(newVal)
				scanner = newVal.Interface().(Scanner)
			}

			return assert.Err(scanner.ScanAttribute(
				rvPtr.Interface(),
			))
		} else if rv.Kind() == reflect.String {
			switch f.field_t.Type.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
				reflect.Float32, reflect.Float64:
				if rv.String() == "" {
					return assert.Fail("field %q cannot be blank", f.field_t.Name)
				}
				num := reflect.New(f.field_t.Type)
				if _, err := fmt.Sscan(rv.String(), num.Interface()); err != nil {
					return err
				}
				val := num.Elem()
				rvPtr = &val
			case reflect.Bool:
				b, err := strconv.ParseBool(rv.String())
				if err != nil {
					return assert.Fail("field %q cannot be converted to bool: %v", f.field_t.Name, err)
				}
				val := reflect.ValueOf(b)
				rvPtr = &val
			default:
				return assert.Fail("type mismatch for field %q: expected %q, got %q",
					f.field_t.Name, typeName(f.field_t.Type), typeName(rv.Type()),
				)
			}
		} else {
			return assert.Fail("value %q not convertible to %q for field %q",
				rv.Type(), f.field_t.Type, f.field_t.Name,
			)
		}
	}

	// Blank check
	if !force && rvPtr.IsZero() && !f.AllowBlank() {
		kind := reflect.Indirect(*rvPtr).Kind()
		if kind == reflect.Struct {
			switch reflect.Indirect(*rvPtr).Interface().(type) {
			case time.Time, sql.NullTime, sql.NullBool, sql.NullFloat64, sql.NullInt64, sql.NullString:
				break
			default:
				return assert.Fail("field %q must not be blank", f.field_t.Name)
			}
		} else if kind == reflect.String || kind == reflect.Slice || kind == reflect.Map || kind == reflect.Ptr {
			return assert.Fail("field %q must not be blank", f.field_t.Name)
		}
	}

	old := f.field_v
	django_reflect.RSet(rvPtr, &f.field_v, false)

	if err := f.Validate(); err != nil {
		f.field_v.Set(old)
		return err
	}

	return nil
}

var (
	_ sql.Scanner   = (*FieldDef)(nil)
	_ driver.Valuer = (*FieldDef)(nil)
)

func (f *FieldDef) Scan(value any) error {

	if f.field_v.Kind() == reflect.Ptr && f.field_v.IsNil() {
		f.field_v.Set(reflect.New(f.field_t.Type.Elem()))
	}

	defer func() {
		assert.Err(BindValueToModel(
			f.Instance(), f, f.field_v.Interface(),
		))
		f.signalChanges(f.field_v.Interface())
	}()

	rv := reflect.ValueOf(value)

	if !rv.IsValid() {
		f.field_v.Set(reflect.Zero(f.field_t.Type))
		return nil
	}

	if scanner, ok := f.field_v.Interface().(sql.Scanner); ok {
		return scanner.Scan(value)
	}

	if scanner, ok := f.field_v.Addr().Interface().(sql.Scanner); ok {
		return scanner.Scan(value)
	}

	if rv.Type() == f.field_v.Type() || (f.field_t.Type.Kind() == reflect.Interface && rv.Type().Implements(f.field_t.Type)) {
		f.field_v.Set(rv)
		return nil
	}

	var rel = f.Rel()
	// Custom handling for definer types - this is required in case field_v is an interface type instead of a concrete type.
	if f.field_t.Type.Kind() == reflect.Interface && f.field_t.Type.Implements(reflect.TypeOf((*Definer)(nil)).Elem()) && rel != nil && (rel.Type() == RelManyToOne || rel.Type() == RelOneToOne) {
		if f.field_v.IsNil() {
			var model = NewObject[Definer](rel.Model())
			var defs = model.FieldDefs()
			var prim = defs.Primary()
			if err := prim.Scan(value); err != nil {
				return fmt.Errorf("failed to scan primary key for %T: %w", rel.Model(), err)
			}
			f.field_v.Set(reflect.ValueOf(model))
		} else {
			var model = f.field_v.Interface().(Definer)
			var defs = model.FieldDefs()
			var prim = defs.Primary()
			if err := prim.Scan(value); err != nil {
				return fmt.Errorf("failed to scan primary key for %T: %w", rel.Model(), err)
			}
			f.field_v.Set(reflect.ValueOf(model))
		}
		return nil
	}

	if definer, ok := f.field_v.Interface().(Definer); ok {
		return definer.FieldDefs().Primary().Scan(value)
	}

	rvPtr, ok := django_reflect.RConvert(&rv, f.field_t.Type)
	if !ok {
		typ := f.field_t.Type
		if typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}

		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			switch typ.Kind() {
			case reflect.Bool:
				if rv.Int() == 0 {
					f.field_v.SetBool(false)
				} else {
					f.field_v.SetBool(true)
				}
			}

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			switch typ.Kind() {
			case reflect.Bool:
				if rv.Uint() == 0 {
					f.field_v.SetBool(false)
				} else {
					f.field_v.SetBool(true)
				}
				return nil
			default:
				return fmt.Errorf("value of type %q not convertible to %q for field %q",
					rv.Type(), f.field_t.Type, f.field_t.Name)
			}

		case reflect.String:
			switch typ.Kind() {
			case reflect.Struct, reflect.Map, reflect.Slice:
				t := reflect.New(typ).Interface()
				if err := json.Unmarshal([]byte(rv.String()), t); err != nil {
					return err
				}
				f.field_v.Set(reflect.ValueOf(t))
				return nil

			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
				reflect.Float32, reflect.Float64:
				if rv.String() == "" {
					return fmt.Errorf("value is empty")
				}
				n := reflect.New(f.field_t.Type)
				if _, err := fmt.Sscan(rv.String(), n.Interface()); err != nil {
					return err
				}
				val := n.Elem()
				rvPtr = &val
			default:
				return fmt.Errorf("value of type %q not convertible to %q for field %q",
					rv.Type(), f.field_t.Type, f.field_t.Name)
			}

		default:
			return fmt.Errorf("value of type %q not convertible to %q for field %q",
				rv.Type(), f.field_t.Type, f.field_t.Name)
		}

	}

	old := f.field_v
	django_reflect.RSet(rvPtr, &f.field_v, false)

	if err := f.Validate(); err != nil {
		f.field_v.Set(old)
		return err
	}

	return nil
}

var _DRIVER_VALUE = reflect.TypeOf((*driver.Valuer)(nil)).Elem()
var maxInt64 uint64 = 1<<63 - 1

func (f *FieldDef) _driverValue(value any) (driver.Value, error) {
	var v reflect.Value
	switch val := value.(type) {
	case reflect.Value:
		v = val
	case *reflect.Value:
		v = *val
	default:
		v = reflect.ValueOf(val)
	}

	if !v.IsValid() {
		return nil, nil
	}

	if v.IsZero() {
		return v.Interface(), nil
	}

	if (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) && v.IsNil() {
		return nil, nil
	}

	assert.Err(BindValueToModel(
		f.Instance(), f, v,
	))

	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}

	if v.Type().Implements(_DRIVER_VALUE) {
		var valuer = v.Interface().(driver.Valuer)
		return valuer.Value()
	}

	switch v.Kind() {
	case reflect.String:
		return v.String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint(), nil
	case reflect.Float32, reflect.Float64:
		return v.Float(), nil
	case reflect.Bool:
		return v.Bool(), nil
	case reflect.Slice, reflect.Array:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			// byte slice, return as []byte
			return v.Bytes(), nil
		}
		return v.Interface(), nil
	case reflect.Struct:
		switch val := f.field_v.Interface().(type) {
		case driver.Valuer:
			var valuer = val.(driver.Valuer)
			return valuer.Value()
		case Definer:
			var def = val.(Definer)
			var pk = PrimaryKey(def)
			return pk, nil
		}
	}

	if v.CanInterface() {
		return v.Interface(), nil
	}

	return nil, fmt.Errorf(
		"cannot convert %q to driver.Value",
		f.field_v.Type().Name(),
	)
}

func (f *FieldDef) driverValue() (driver.Value, error) {
	var v = f.field_v
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if !v.IsValid() || v.IsNil() {
			var def = f.GetDefault()
			f.SetValue(def, true)
			return f._driverValue(def)
		}
		v = v.Elem()
	}

	if IsZero(v) {
		return f._driverValue(f.GetDefault())
	}

	return f._driverValue(v)
}

// Returns the value of the field as a driver.Value.
//
// This value should be used for storing the field in a database.
//
// If the field is nil or the zero value, the default value is returned.
func (f *FieldDef) Value() (driver.Value, error) {
	var val, err = f.driverValue()
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get value for field %q (%T): %w",
			f.field_t.Name, f.field_v.Interface(), err,
		)
	}

	//	if reflect.ValueOf(val).IsZero() && !f.AllowBlank() {
	//		return nil, fmt.Errorf(
	//			"field %q (%T) is not allowed to be blank",
	//			f.field_t.Name, f.field_v.Interface(),
	//		)
	//	}

	return val, nil
}

func nameGetDefault(f *FieldDef) string {
	return fmt.Sprintf("GetDefault%s", f.field_t.Name)
}

func nameSetValue(f *FieldDef) string {
	return fmt.Sprintf("Set%s", f.field_t.Name)
}

func nameGetValue(f *FieldDef) string {
	return fmt.Sprintf("Get%s", f.field_t.Name)
}

func typeName(t reflect.Type) string {
	var sb strings.Builder
	_typeName(&sb, t)
	return sb.String()
}

func _typeName(sb *strings.Builder, t reflect.Type) {
	if t.Kind() == reflect.Ptr {
		sb.WriteString("*")
		_typeName(sb, t.Elem())
		return
	}

	switch t.Kind() {
	case reflect.Map:
		sb.WriteString("map[")
		_typeName(sb, t.Key())
		sb.WriteString("]")
	case reflect.Slice, reflect.Array:
		sb.WriteString("[]")
		_typeName(sb, t.Elem())
	case reflect.Chan:
		sb.WriteString("chan<")
		_typeName(sb, t.Elem())
		sb.WriteString(">")
	case reflect.Func:
		sb.WriteString("func(")
		for i := 0; i < t.NumIn(); i++ {
			if i > 0 {
				sb.WriteString(", ")
			}
			_typeName(sb, t.In(i))
		}
		sb.WriteString(") ")
		if t.NumOut() > 0 {
			sb.WriteString(" ")
			_typeName(sb, t.Out(0))
		}
	default:
		sb.WriteString(t.Name())
	}
}
