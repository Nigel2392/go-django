package attrs

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs/attrutils"
	django_reflect "github.com/Nigel2392/go-django/src/core/attrs/internals/django_refelct"
	"github.com/Nigel2392/go-django/src/core/checks"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/goldcrest"
	"github.com/Nigel2392/versatile/scan"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	capCaser            = cases.Title(language.English)
	ALLOW_METHOD_CHECKS = false // Whether to allow method checks for getters and setters

	_ Field = (*FieldDef)(nil) // Ensure FieldDef implements the Field interface

	_modelFields = make(map[reflect.Type]map[string][]func(*Config))
)

func relFromConfig(f FieldDefinition, cnf *Config) Relation {
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

func RegisterConfigChange(model Definer, field string, change func(*Config)) {
	rt := reflect.TypeOf(model)
	fm, ok := _modelFields[rt]
	if !ok {
		fm = make(map[string][]func(*Config))
		_modelFields[rt] = fm
	}

	fs, ok := fm[field]
	if !ok {
		fs = make([]func(*Config), 0)
	}

	fs = append(fs, change)
	fm[field] = fs
}

func GetFromAttributes[T any](attrMap map[string]any, key string) (T, bool) {
	var n T
	if v, ok := attrMap[key]; ok {
		if t, ok := v.(T); ok {
			return t, true
		}
		var (
			rT = reflect.TypeOf((*T)(nil)).Elem()
			vT = reflect.TypeOf(v)
			vV = reflect.ValueOf(v)
		)

		if vT.AssignableTo(rT) {
			return vV.Interface().(T), true
		}

		if vT.ConvertibleTo(rT) {
			return vV.Convert(rT).Interface().(T), true
		}

		return n, false
	}
	return n, false
}

type FieldDef struct {
	defs Definitions
	*Config
	instance_v_ptr reflect.Value
	field_t        *reflect.StructField
	field_v        reflect.Value
	formField      fields.Field
	fieldName      string
	bound          uintptr // internal use only, marks whether the value has been bound
	// directlyInteractible bool
}

var _ERROR = reflect.TypeOf((*error)(nil)).Elem()
var _DEFINER = reflect.TypeOf((*Definer)(nil)).Elem()
var _globalConf = &Config{}

// NewField creates a new field definition for the given instance.
//
// This can then be used for managing the field in a more abstract way.
func NewField(instance any, name string, conf ...*Config) *FieldDef {
	var (
		instance_t_ptr = reflect.TypeOf(instance)
		instance_v_ptr = reflect.ValueOf(instance)
	)

	var field_t, ok = attrutils.GetStructField(instance_t_ptr.Elem(), name)
	if !ok {
		panic(fmt.Sprintf("field %q not found in %T", name, instance))
	}

	var cnf *Config
	if len(conf) == 0 && field_t.Tag != "" {
		cnf = autoDefinitionStructTag(&Config{}, field_t)
	}

	if len(conf) > 0 && conf[0] != nil {
		cnf = conf[0]
	}

	fm, ok := _modelFields[instance_t_ptr]
	if ok {
		fs, _ := fm[name]

		if len(fs) > 0 && cnf == nil {
			cnf = &Config{}
		}

		for _, fn := range fs {
			fn(cnf)
		}
	}

	if cnf == nil {
		cnf = _globalConf
	}

	// setupFieldValue:
	// make sure we can access the field
	var field_v = instance_v_ptr.Elem().Field(field_t.Index[0])
	var curr_t = field_v.Type()
	for i := 1; i < len(field_t.Index); i++ {
		var isNil = false
		if field_v.Kind() == reflect.Ptr {
			isNil = field_v.IsNil()
			if isNil {
				if !cnf.AutoInit {
					panic(fmt.Sprintf("field %q is nil (%s) and cannot be accessed", name, curr_t.Elem().Name()))
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

				if curr_t.Implements(_DEFINER) {
					newVal = reflect.ValueOf(
						NewObject[Definer](context.Background(), curr_t),
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

	if !field_v.IsValid() {
		panic("field not found in type")
	}

	var f = &FieldDef{
		Config:         cnf,
		instance_v_ptr: instance_v_ptr,
		field_t:        field_t,
		field_v:        field_v,
		fieldName:      name,
		// directlyInteractible: directlyInteractible,
	}

	if field_v.IsValid() && (field_t.Type.Kind() == reflect.Pointer && !field_v.IsNil()) {
		if err := BindValueToModel(instance.(Definer), f, field_v); err != nil {
			panic("failed to bind value to model: " + err.Error())
		}
	}

	if cnf.OnInit != nil {
		cnf = cnf.OnInit(
			instance.(Definer), f, cnf,
		)
		if cnf != nil {
			f.Config = cnf
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

func (f *FieldDef) StructField() *reflect.StructField {
	if f.instance_v_ptr.Type() != reflect.TypeOf(f.defs.Instance()) {
		return nil
	}
	return f.field_t
}

func (f *FieldDef) Check(ctx context.Context) []checks.Message {
	var messages []checks.Message

	if f.Config.MaxLength < 0 || f.Config.MaxLength < f.Config.MinLength && f.Config.MaxLength > 0 {
		var messageText = fmt.Sprintf(
			"Field \"%s\" in model \"%T\" has an invalid max_length (%d)",
			f.Name(), f.instance_v_ptr.Interface(), f.Config.MaxLength,
		)
		messages = append(messages, checks.Error(
			"field.invalid_max_length",
			messageText,
			f, "",
		))
	}

	if f.Config.MinLength < 0 {
		var messageText = fmt.Sprintf(
			"Field \"%s\" in model \"%T\" has an invalid min_length (%d).",
			f.Name(), f.instance_v_ptr.Interface(), f.Config.MinLength,
		)
		messages = append(messages, checks.Error(
			"field.invalid_min_length",
			messageText,
			f, "",
		))
	}

	if f.Config.MaxLength > 0 && f.Config.MinLength > 0 && f.Config.MaxLength < f.Config.MinLength {
		var messageText = fmt.Sprintf(
			"Field \"%s\" in model \"%T\" has an invalid max_length (%d) less than min_length (%d).",
			f.Name(), f.instance_v_ptr.Interface(), f.Config.MaxLength, f.Config.MinLength,
		)
		messages = append(messages, checks.Error(
			"field.invalid_length",
			messageText,
			f, "",
		))
	}

	if f.Config.MaxValue < f.Config.MinValue && f.Config.MaxValue > 0 {
		var messageText = fmt.Sprintf(
			"Field \"%s\" in model \"%T\" has an invalid max_value (%f) less than min_value (%f).",
			f.Name(), f.instance_v_ptr.Interface(), f.Config.MaxValue, f.Config.MinValue,
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
func (f *FieldDef) OnModelRegister(model Definer, outer FieldDefinition) error {

	var typElem = f.instance_v_ptr.Type().Elem()
	attrutils.AddStructField(typElem, f.fieldName, f.field_t)

	if ALLOW_METHOD_CHECKS {
		var (
			defaultMethodName  = nameGetDefault(f.field_t)
			setValueMethodName = nameSetValue(f.field_t)
			getValueMethodName = nameGetValue(f.field_t)
		)

		if getDefaultMethod, ok := typElem.MethodByName(defaultMethodName); ok {
			attrutils.AddStructMethod(typElem, defaultMethodName, getDefaultMethod)
		}

		if setValueMethod, ok := typElem.MethodByName(setValueMethodName); ok {
			attrutils.AddStructMethod(typElem, setValueMethodName, setValueMethod)
		}

		if getValueMethod, ok := typElem.MethodByName(getValueMethodName); ok {
			attrutils.AddStructMethod(typElem, getValueMethodName, getValueMethod)
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

	if f.defs == nil {
		f.defs = defs
	}
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
	if f.Config.Label != nil {
		var text, ok = trans.GetText(ctx, f.Config.Label)
		if ok {
			return text
		}
	}

	// if f.directlyInteractible {
	if labeler, ok := f.field_v.Interface().(Labeler); ok {
		label := labeler.Label(ctx)
		if label != "" {
			return label
		}
	}
	// }

	var rel = f.Rel()
	if rel != nil {
		var cTypeDef = contenttypes.DefinitionForObject(rel)
		if cTypeDef != nil {
			return cTypeDef.Label(ctx)
		}
	}

	return trans.T(ctx, NiceName(f.field_t.Name))
}

func (f *FieldDef) HelpText(ctx context.Context) string {
	if helpTexter, ok := f.field_v.Interface().(Helper); ok {
		return helpTexter.HelpText(ctx)
	}

	if f.Config.HelpText != nil {
		t, ok := trans.GetText(ctx, f.Config.HelpText)
		if !ok {
			panic(fmt.Sprintf(
				"HelpText for field %q (%T) is not a `string` or `function(context) string`, got %T",
				f.field_t.Name, f.field_v.Interface(), f.Config.HelpText,
			))
		}
		return t
	}

	return ""
}

func (f *FieldDef) Name() string {
	if f.Config.NameOverride != "" {
		return f.Config.NameOverride
	}
	return f.fieldName
}

func (f *FieldDef) TypeString() string {
	return fmt.Sprintf("%s.%s", f.instance_v_ptr.Type().Elem().Name(), f.field_t.Type.Name())
}

func (f *FieldDef) Attrs() map[string]interface{} {
	var attrs map[string]interface{}
	if f.defs != nil && !ContextHasFlag(f.defs.Context(), CtxFlagRegistering) {
		var (
			_attrs   any
			fldAttrs map[string]map[string]any
		)

		reg, ok := modelReg[f.instance_v_ptr.Type()]
		if !ok || reg.stored == nil {
			goto createAttrs
		}

		_attrs, ok = reg.stored[MetaStorageKeyAttrs]
		if !ok {
			goto createAttrs
		}

		fldAttrs, ok = _attrs.(map[string]map[string]any)
		if !ok {
			goto createAttrs
		}

		attrs, ok = fldAttrs[f.Name()]
		if ok {
			return attrs
		}
	}

createAttrs:

	attrs = f.Config.Attrs()
	attrs[AttrNameKey] = f.Name()
	return attrs
}

func (f *FieldDef) ColumnName() string {
	if f.Config.Column != "" {
		return f.Config.Column
	}

	return ColumnName(f.fieldName)
}

func (f *FieldDef) Rel() Relation {

	if f.Config.RelForeignKey == nil &&
		f.Config.RelManyToMany == nil &&
		f.Config.RelOneToOne == nil &&
		f.Config.RelForeignKeyReverse == nil {
		return nil
	}

	if f.defs == nil || ContextHasFlag(f.defs.Context(), CtxFlagRegistering) {
		return relFromConfig(f, f.Config)
	}

	mdl, ok := modelReg[f.instance_v_ptr.Type()]
	if !ok {
		return relFromConfig(f, f.Config)
	}

	rel, ok := mdl.forward.Get(f.Name())
	if ok {
		return rel
	}

	rel, ok = mdl.reverse.Get(f.Name())
	if !ok {
		return relFromConfig(f, f.Config)
	}

	return rel
}

func (f *FieldDef) Validate() error {
	var v = f.GetValue()

	var rV = reflect.ValueOf(v)
	if f.Config.MinLength > 0 || f.Config.MaxLength > 0 {
		if slices.Contains([]reflect.Kind{reflect.String, reflect.Slice, reflect.Array, reflect.Map}, rV.Kind()) {

			if f.Config.MinLength > 0 && rV.Len() < int(f.Config.MinLength) {
				return fmt.Errorf(
					"field %q must be at least %d characters long, got %d",
					f.field_t.Name, f.Config.MinLength, rV.Len(),
				)
			}

			if f.Config.MaxLength > 0 && rV.Len() > int(f.Config.MaxLength) {
				return fmt.Errorf(
					"field %q must be at most %d characters long, got %d",
					f.field_t.Name, f.Config.MaxLength, rV.Len(),
				)
			}
		}
	}

	if f.Config.MinValue > 0 || f.Config.MinValue < 0 || f.Config.MaxValue > 0 || f.Config.MaxValue < 0 {
		if slices.Contains([]reflect.Kind{reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64}, rV.Kind()) {
			var num, err = attrutils.CastToNumber[float64](v)
			if err != nil {
				return fmt.Errorf(
					"field %q is not a number: %w",
					f.field_t.Name, err,
				)
			}
			if f.Config.MinValue > 0 && num < f.Config.MinValue {
				return fmt.Errorf(
					"field %q must be at least %f",
					f.field_t.Name, f.Config.MinValue,
				)
			}

			if f.Config.MaxValue > 0 && num > f.Config.MaxValue {
				return fmt.Errorf(
					"field %q must be at most %f",
					f.field_t.Name, f.Config.MaxValue,
				)
			}

			if f.Config.MinValue < 0 && num > f.Config.MinValue {
				return fmt.Errorf(
					"field %q must be at least %f",
					f.field_t.Name, f.Config.MinValue,
				)
			}

			if f.Config.MaxValue < 0 && num < f.Config.MaxValue {
				return fmt.Errorf(
					"field %q must be at most %f",
					f.field_t.Name, f.Config.MaxValue,
				)
			}
		}
	}

	for _, validator := range f.Config.Validators {
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

	if ALLOW_METHOD_CHECKS {
		var funcName = fmt.Sprintf("%sToString", f.Name())
		if method, ok := f.instance_v_ptr.Type().Elem().MethodByName(funcName); ok {
			var out = method.Func.Call([]reflect.Value{f.instance_v_ptr})
			assert.Gt(out, 0, "Method %q on raw did not return a value", funcName)
			return out[0].String()
		}
	}

	var rv = reflect.ValueOf(v)
	return toString(rv, v)
}

func (f *FieldDef) GetDefault() interface{} {

	if f.Config.Default != nil {
		var v = reflect.ValueOf(f.Config.Default)
		if v.IsValid() && v.Kind() == reflect.Func {

			var typ = v.Type()
			var out []reflect.Value
			switch typ.NumIn() {
			case 1:
				out = v.Call([]reflect.Value{f.instance_v_ptr})
			case 0:
				out = v.Call([]reflect.Value{})
			default:
				assert.Fail("Invalid function type %T for GetDefault function", f.Config.Default)
			}

			var outVal reflect.Value
			var numOut = typ.NumOut()
			switch {
			case numOut == 0:
				assert.Fail("No return value for GetDefault function: %T", f.Config.Default)
			case numOut == 1:
				outVal = out[0]
			case numOut == 2 && out[1].Type().Implements(_ERROR):
				outVal = out[0]
				err := out[1].Interface().(error)
				if err != nil {
					assert.Fail("Error retrieving default value: %v", err)
				}
			default:
				assert.Fail("Invalid function type %T for GetDefault function", f.Config.Default)
			}

			var val = outVal.Interface()
			assert.Err(BindValueToModel(
				f.Instance(), f, val,
			))
			return outVal
		}

		var outVal = f.Config.Default
		assert.Err(BindValueToModel(
			f.Instance(), f, outVal,
		))
		return outVal
	}

	if ALLOW_METHOD_CHECKS {
		var inst_t_ptr = f.instance_v_ptr.Type()
		var funcName = nameGetDefault(f.field_t)
		var method, ok = attrutils.GetStructMethod(inst_t_ptr, funcName)
		if ok {
			var out []reflect.Value
			switch method.Type.In(0) {
			case inst_t_ptr:
				out = method.Func.Call([]reflect.Value{f.instance_v_ptr})
			case inst_t_ptr.Elem():
				out = method.Func.Call([]reflect.Value{f.instance_v_ptr.Elem()})
			}
			assert.Gt(out, 0, "Method %q on raw did not return a value", funcName)
			var outVal = out[0].Interface()
			assert.Err(BindValueToModel(
				f.Instance(), f, outVal,
			))
			return outVal
		}
	}

	if f.Config.Default == nil {
		return nil
	}

	//	// required to make sure through models dont get a
	//	// default zero value for their source or target fields
	//	if f.FieldConfig.Default == nil {
	//		var meta = ThroughModelMeta(f.Instance())
	//		switch f.Name() {
	//		case meta.SourceField, meta.TargetField:
	//			return nil
	//		}
	//	}

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

	if f.field_v.Kind() == reflect.Pointer && f.field_v.IsNil() {
		return f.field_v.Interface()
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

	opts := f.Config.FormOptions(f)
	f.formField = f.Config.GetFormField(f, f.field_v, opts...)
	return f.formField
}

func (f *FieldDef) GetValue() interface{} {

	if ALLOW_METHOD_CHECKS {
		var methodName = nameGetValue(f.field_t)
		var inst_t_ptr = f.instance_v_ptr.Type()
		var method, ok = attrutils.GetStructMethod(inst_t_ptr, methodName)
		if ok {
			var outVal any
			switch method.Type.In(0) {
			case inst_t_ptr:
				outVal = method.Func.Call([]reflect.Value{f.instance_v_ptr})[0].Interface()
			case inst_t_ptr.Elem():
				outVal = method.Func.Call([]reflect.Value{f.instance_v_ptr.Elem()})[0].Interface()
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
		var setterName = nameSetValue(f.field_t)
		var method, ok = attrutils.GetStructMethod(f.instance_v_ptr.Type(), setterName)
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

	wasSet, err := scan.RScan(f.field_v.Addr(), v, scan.SF_CONVS)
	if err != nil {
		return err
	}

	if wasSet {
		if err := f.Validate(); err != nil {
			return err
		}
		return nil
	}

	rvPtr, ok := django_reflect.RConvert(&rv, f.field_t.Type)
	if !ok {
		if sqlScanner, ok := f.field_v.Addr().Interface().(sql.Scanner); ok {
			return assert.Err(sqlScanner.Scan(
				rvPtr.Interface(),
			))
		}

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
		if f.bound == f.field_v.UnsafeAddr() {
			return
		}
		assert.Err(BindValueToModel(
			f.Instance(), f, f.field_v.Interface(),
		))
		f.bound = f.field_v.UnsafeAddr()
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

	wasSet, err := scan.RScan(f.field_v.Addr(), value, scan.SF_CONVS)
	if err != nil {
		return err
	}

	if wasSet {
		return nil
	}

	if rv.Type() == f.field_v.Type() || (f.field_t.Type.Kind() == reflect.Interface && rv.Type().Implements(f.field_t.Type)) {
		f.field_v.Set(rv)
		return nil
	}

	var rel = f.Rel()
	// Custom handling for definer types - this is required in case field_v is an interface type instead of a concrete type.
	if f.field_t.Type.Kind() == reflect.Interface && f.field_t.Type.Implements(reflect.TypeOf((*Definer)(nil)).Elem()) && rel != nil && (rel.Type() == RelManyToOne || rel.Type() == RelOneToOne) {
		var model Definer
		if f.field_v.IsNil() {
			model = NewObject[Definer](f.defs.Context(), rel.Model())
			f.field_v.Set(reflect.ValueOf(model))
		} else {
			model = f.field_v.Interface().(Definer)
		}

		var defs = Define(f.defs.Context(), model)
		var prim = defs.Primary()
		if err := prim.Scan(value); err != nil {
			return fmt.Errorf("failed to scan primary key for %T: %w", rel.Model(), err)
		}

		return nil
	}

	if definer, ok := f.field_v.Interface().(Definer); ok {
		return Define(f.defs.Context(), definer).Primary().Scan(value)
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
					return fmt.Errorf("%T.%s: value is empty", f.Instance(), f.Name())
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
			return val.Value()
		case Definer:
			var pk = PrimaryKey(f.defs.Context(), val)
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

	if IsZero(v) && f.Config.Default != nil {
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

func nameGetDefault(f *reflect.StructField) string {
	return fmt.Sprintf("GetDefault%s", f.Name)
}

func nameSetValue(f *reflect.StructField) string {
	return fmt.Sprintf("Set%s", f.Name)
}

func nameGetValue(f *reflect.StructField) string {
	return fmt.Sprintf("Get%s", f.Name)
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
