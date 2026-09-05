package fattrs

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/checks"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/versatile/bitcheck"
	"github.com/Nigel2392/versatile/scan"
)

type fieldFlag bitcheck.Flag

const (
	flagNone    fieldFlag = iota
	flagIsModel fieldFlag = 1 << iota
	flagIsValuer
	flagIsScanner
	flagIsPtrScanner
	flagIsBinder
	flagIsPtrBinder
	flagSetupNilVal

	// value types, anything that can be initialised with `var myVar [TYPE]`
	flagIsBasic
	flagHasColumn
	flagHasColumnName

	F_IS_REVERSE_FLD
)

type PtrFieldConfig[MODEL attrs.Definer, VALUE any] struct {
	Flags    fieldFlag
	Config   attrs.FieldConfig
	Default  interface{} // func() VALUE | VALUE
	Validate func(obj MODEL, value VALUE) error
	New      func() VALUE
	IsZero   func(VALUE) bool
}

type ptrFieldOpts[MODEL attrs.Definer, VALUE any] struct {
	attrs.FieldDefinition
	typ           reflect.Type
	flags         fieldFlag
	validateValue func(obj MODEL, value VALUE) error
	getDefault    func(obj MODEL) *VALUE
	newValue      func() VALUE
	bindError     func(error)
	isNil         func(*VALUE) bool
}

var (
	_DEFINER = reflect.TypeOf((*attrs.Definer)(nil)).Elem()
	_SCANNER = reflect.TypeOf((*sql.Scanner)(nil)).Elem()
	_VALUER  = reflect.TypeOf((*driver.Valuer)(nil)).Elem()
	_BINDER  = reflect.TypeOf((*attrs.Binder)(nil)).Elem()
)

func ptrConf_Value_CheckNil[VALUE any](v *VALUE) bool {
	val := reflect.ValueOf(*v)
	return !val.IsValid() || val.IsNil()
}

func ptrConf_Value_NeverNil[VALUE any](v *VALUE) bool {
	return false
}

func ptrConf_Validate_Nop[MODEL attrs.Definer, VALUE any](obj MODEL, value VALUE) error {
	return nil
}

func ptrConf_Default_Zero[MODEL attrs.Definer, VALUE any](obj MODEL) (v VALUE) {
	return v
}

// only to be used on value types that can be initialized by just
// referencing the type in a var statement: `var myVar [TYPE]`
func ptrConf_newZeroVal[TYPE any]() (v TYPE) {
	return v
}

func ptrFieldConfigToOptions[MODEL attrs.Definer, VALUE any](fld *ptrField[MODEL, VALUE], conf PtrFieldConfig[MODEL, VALUE]) *ptrFieldOpts[MODEL, VALUE] {
	var opts = &ptrFieldOpts[MODEL, VALUE]{
		typ:           reflect.TypeFor[VALUE](),
		validateValue: conf.Validate,
		newValue:      conf.New,
		flags:         conf.Flags,
	}

	var (
		rTPtr = reflect.TypeOf(new(VALUE))
		rT    = rTPtr.Elem()
	)

	if rTPtr.Implements(_SCANNER) {
		opts.flags = bitcheck.Set(opts.flags, flagIsPtrScanner, true)
	}
	if rT.Implements(_SCANNER) {
		opts.flags = bitcheck.Set(opts.flags, flagIsScanner, true)
	}
	if rTPtr.Implements(_BINDER) {
		opts.flags = bitcheck.Set(opts.flags, flagIsPtrBinder, true)
	}
	if rT.Implements(_BINDER) {
		opts.flags = bitcheck.Set(opts.flags, flagIsBinder, true)
	}
	if rT.Implements(_VALUER) {
		opts.flags = bitcheck.Set(opts.flags, flagIsValuer, true)
	}
	if rT == _DEFINER || rT.Implements(_DEFINER) {
		opts.flags = bitcheck.Set(opts.flags, flagIsModel, true)
	}

	/*

		The following block is supposed to and should always adhere to
		the structure in [attrs.relFromConfig].

	*/
	var rel attrs.Relation
	switch {
	case conf.Config.RelForeignKey != nil:
		//	attrs.RelManyToOne
		rel = conf.Config.RelForeignKey
		opts.flags = bitcheck.Set(opts.flags, flagHasColumn, true)

	case conf.Config.RelManyToMany != nil:
		//	attrs.RelManyToMany
		rel = conf.Config.RelManyToMany
		opts.flags = bitcheck.Set(opts.flags, flagHasColumn, false)

	case conf.Config.RelOneToOne != nil:
		//	attrs.RelOneToOne
		rel = conf.Config.RelOneToOne
		opts.flags = bitcheck.Set(opts.flags, flagHasColumn, rel.Through() == nil)

	case conf.Config.RelForeignKeyReverse != nil:
		//	attrs.RelOneToMany
		rel = conf.Config.RelForeignKeyReverse
		opts.flags = bitcheck.Set(opts.flags, flagHasColumn, false)
		opts.flags = bitcheck.Set(opts.flags, F_IS_REVERSE_FLD, true)

	default:
		opts.flags = bitcheck.Set(opts.flags, flagHasColumn, true)
	}

	if conf.Config.Column != "" {
		opts.flags = bitcheck.Set(opts.flags, flagHasColumnName, true)
	}

	if rel != nil {
		var relThrough = rel.Through()
		if relThrough != nil {
			opts.flags = bitcheck.Set(opts.flags, flagHasColumn, false)
		}
	} else {
		var rel, fwd, ok = attrs.GetRelationMeta(*new(MODEL), fld.fieldName)
		if ok && !fwd {
			switch rel.Type() {
			case attrs.RelManyToMany:
				conf.Config.RelManyToMany = rel
			case attrs.RelManyToOne:
				conf.Config.RelForeignKey = rel
			case attrs.RelOneToMany:
				conf.Config.RelForeignKeyReverse = rel
			case attrs.RelOneToOne:
				conf.Config.RelOneToOne = rel
			}

			opts.flags = bitcheck.Set(opts.flags, F_IS_REVERSE_FLD, true)
		}
	}

	if bitcheck.Is(opts.flags, flagIsBinder) || bitcheck.Is(opts.flags, flagIsPtrBinder) {
		opts.flags = bitcheck.Set(opts.flags, flagSetupNilVal, true)
	}

	switch _default := conf.Default.(type) {
	case func(MODEL) *VALUE:
		opts.getDefault = _default

	case func(attrs.Definer) *VALUE:
		opts.getDefault = func(obj MODEL) *VALUE {
			d := _default(obj)
			return d
		}

	case func(MODEL) VALUE:
		opts.getDefault = func(obj MODEL) *VALUE {
			d := _default(obj)
			return &d
		}

	case func(attrs.Definer) VALUE:
		opts.getDefault = func(obj MODEL) *VALUE {
			d := _default(obj)
			return &d
		}

	case *VALUE:
		opts.getDefault = func(obj MODEL) *VALUE {
			if _default == nil {
				var z VALUE
				return &z
			}
			d := *_default
			return &d
		}

	case VALUE:
		opts.getDefault = func(obj MODEL) *VALUE {
			v := _default
			return &v
		}

	case nil, NULL:
		opts.getDefault = ptrConf_Default_Zero

	default:
		panic(fmt.Sprintf("Invalid default type %T", _default))

	}

	switch rT.Kind() {
	case reflect.Pointer,
		reflect.Interface,
		reflect.Map,
		reflect.Slice,
		reflect.Chan,
		reflect.Func:

		// set nil check
		opts.isNil = ptrConf_Value_CheckNil

		if opts.newValue == nil {
			switch rT.Kind() {
			case reflect.Pointer:
				opts.newValue = func() (v VALUE) {
					return reflect.New(rT.Elem()).Interface().(VALUE)
				}
			case reflect.Map:
				opts.newValue = func() VALUE {
					return reflect.MakeMap(rT).Interface().(VALUE)
				}
			case reflect.Slice:
				opts.newValue = func() VALUE {
					return reflect.MakeSlice(rT, 0, 0).Interface().(VALUE)
				}
			case reflect.Interface, reflect.Chan, reflect.Func:
				opts.newValue = ptrConf_newZeroVal[VALUE] // typed nil by default
			}
		}

	default:
		// can never be nil
		opts.isNil = ptrConf_Value_NeverNil

		opts.flags = bitcheck.Set(opts.flags, flagIsBasic, true)

		if opts.newValue == nil {
			opts.newValue = ptrConf_newZeroVal[VALUE]
		}
	}

	if opts.validateValue == nil {
		opts.validateValue = ptrConf_Validate_Nop
	}

	if opts.newValue == nil {
		panic(fmt.Sprintf("opts.newValue should not be nil for %q", fld.fieldName))
	}

	opts.FieldDefinition = attrs.NewField(
		attrs.NewObject[MODEL](context.Background(), reflect.TypeFor[MODEL]()),
		fld.fieldName, &conf.Config,
	)

	return opts
}

type ptrField[MODEL attrs.Definer, VALUE any] struct {
	obj       MODEL
	val       *VALUE
	dst       reflect.Value
	defs      attrs.Definitions
	fieldName string
	opts      *ptrFieldOpts[MODEL, VALUE]
	confFunc  func() PtrFieldConfig[MODEL, VALUE]
}

func Field[MODEL attrs.Definer, VALUE any](obj MODEL, name string, ptr *VALUE, cnf func() PtrFieldConfig[MODEL, VALUE]) attrs.Field {
	var opts, ok = GetOptions[*ptrFieldOpts[MODEL, VALUE]](reflect.TypeOf(obj), name)
	if !ok && cnf == nil {
		panic(fmt.Sprintf("options were not configured for field %q in type %T", name, obj))
	}

	var f = &ptrField[MODEL, VALUE]{
		obj:       obj,
		val:       ptr,
		fieldName: name,
		opts:      opts,
		dst:       reflect.ValueOf(ptr),
	}

	if !ok {
		f.confFunc = cnf
	}

	return f
}

func (r *ptrField[M, V]) useOpts() *ptrFieldOpts[M, V] {
	if r.opts != nil {
		return r.opts
	}

	if r.confFunc != nil {
		r.opts = ptrFieldConfigToOptions(r, r.confFunc())
	}

	return r.opts
}

func (f *ptrField[M, V]) Check(ctx context.Context) []checks.Message {
	field := f.useOpts().FieldDefinition
	if checker, ok := field.(checks.Checker); ok {
		return checker.Check(ctx)
	}
	return []checks.Message{}
}

func (r *ptrField[M, V]) OnModelRegister(model attrs.Definer, outer attrs.FieldDefinition) error {
	if r.confFunc != nil {
		AddOptions(
			reflect.TypeOf(r.obj),
			r.fieldName,
			ptrFieldConfigToOptions(r, r.confFunc()),
		)
	}
	return nil
}

func (r *ptrField[M, V]) Embedded() bool {
	var f, ok = r.useOpts().FieldDefinition.(attrs.Embedder)
	if ok {
		return f.Embedded()
	}
	return false
}

func (r *ptrField[M, V]) ForSelectAll() bool {
	if bitcheck.Is(r.opts.flags, F_IS_REVERSE_FLD) {
		return false
	}
	return bitcheck.Is(r.opts.flags, flagHasColumn)
}

func (r *ptrField[M, V]) CanMigrate() bool {
	return bitcheck.Is(r.opts.flags, flagHasColumn)
}

func (r *ptrField[M, V]) AllowDBEdit() bool {
	return bitcheck.Is(r.opts.flags, flagHasColumn) && !bitcheck.Is(r.opts.flags, F_IS_REVERSE_FLD)
}

func (r *ptrField[M, V]) IsReverse() bool {
	return bitcheck.Is(r.opts.flags, F_IS_REVERSE_FLD)
}

//	func (r *ptrField[M, V]) AllowReverseRelation() bool {
//		return bitcheck.Is(r.opts.flags, flagHasColumn)
//	}

func (r *ptrField[M, V]) StructField() *reflect.StructField {
	var rv = reflect.TypeOf(r.obj)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}

	var fld, ok = rv.FieldByName(r.fieldName)
	if !ok {
		return nil
	}

	return &fld
}

func (r *ptrField[M, V]) Name() string                                    { return r.fieldName }
func (r *ptrField[M, V]) FieldDefinitions() attrs.Definitions             { return r.defs }
func (r *ptrField[M, V]) BindToDefinitions(definitions attrs.Definitions) { r.defs = definitions }
func (r *ptrField[M, V]) HelpText(ctx context.Context) string             { return r.useOpts().HelpText(ctx) }
func (r *ptrField[M, V]) Label(ctx context.Context) string                { return r.useOpts().Label(ctx) }
func (r *ptrField[M, V]) Instance() attrs.Definer                         { return r.obj }

type relWithField struct {
	attrs.RelationTarget
	fld attrs.FieldDefinition
}

func (r *relWithField) Field() attrs.FieldDefinition {
	return r.fld
}

type relWithFromField struct {
	attrs.Relation
	fld attrs.FieldDefinition
}

func (r *relWithFromField) From() attrs.RelationTarget {
	return &relWithField{
		RelationTarget: r.Relation.From(),
		fld:            r.fld,
	}
}

func (r *ptrField[M, V]) Rel() attrs.Relation {
	rel := r.useOpts().Rel()
	if rel != nil {
		return &relWithFromField{Relation: rel, fld: r}
	}
	return nil
}

func (r *ptrField[M, V]) ColumnName() string {

	if bitcheck.Is(r.useOpts().flags, flagHasColumnName) {
		return r.opts.ColumnName()
	}

	var rel = r.Rel()
	if rel == nil {
		return attrs.ColumnName(r.Name())
	}

	var from = rel.From()
	if from != nil && from.Field() != r {
		return from.Field().ColumnName()
	}

	switch rel.Type() {
	case attrs.RelOneToOne, attrs.RelOneToMany, attrs.RelManyToMany:
		var primary = r.defs.Primary()
		return primary.ColumnName()
	}

	return attrs.ColumnName(r.Name())
}
func (r *ptrField[M, V]) Type() reflect.Type      { return r.useOpts().Type() }
func (r *ptrField[M, V]) Attrs() map[string]any   { return r.useOpts().Attrs() }
func (r *ptrField[M, V]) IsPrimary() bool         { return r.useOpts().IsPrimary() }
func (r *ptrField[M, V]) AllowNull() bool         { return r.useOpts().AllowNull() }
func (r *ptrField[M, V]) AllowBlank() bool        { return r.useOpts().AllowBlank() }
func (r *ptrField[M, V]) AllowEdit() bool         { return r.useOpts().AllowEdit() }
func (r *ptrField[M, V]) FormField() fields.Field { return r.useOpts().FormField() }
func (r *ptrField[M, V]) ToString() string        { return attrs.ToString(r.GetValue()) }

func (r *ptrField[M, V]) SetValue(v interface{}, force bool) error {
	if bitcheck.Is(r.useOpts().flags, flagIsModel) {
		var rt reflect.Type
		d, ok := v.(attrs.Definer)
		if !ok {
			goto scanNow
		}

		rt = reflect.TypeOf(d)
		if rt == nil {
			goto scanNow
		}

		if rt == r.opts.typ {
			*r.val = d.(V)
			return r.wasSet(nil)
		}

		if rt.AssignableTo(r.opts.typ) {
			*r.val = d.(V)
			return r.wasSet(nil)
		}

		if rt.ConvertibleTo(r.opts.typ) {
			rv := reflect.ValueOf(v).Convert(r.opts.typ)
			*r.val = rv.Interface().(V)
			return r.wasSet(nil)
		}
	}

scanNow:
	wasSet, err := scan.Scan(r.val, v, scan.SF_DEFAULT)
	if wasSet {
		return r.wasSet(err)
	}

	var (
		src = reflect.ValueOf(v)
		typ = src.Type()
	)

	if typ == nil || !src.IsValid() {
		return nil
	}

	if typ.ConvertibleTo(r.opts.typ) {
		src = src.Convert(r.opts.typ)
		typ = src.Type()
	}

	if typ.AssignableTo(r.opts.typ) {
		return r.set(src)
	}

	// if is model we can create new value of type V
	// and set the scanned v as it's primary key.
	// this wont work if the field type itself is an interface.
	if bitcheck.Is(r.opts.flags, flagIsModel) && r.opts.typ != _DEFINER {
		var newModel = reflect.New(r.opts.typ.Elem())
		var newModelV = newModel.Interface().(attrs.Definer)
		if err := attrs.SetPrimaryKey(r.defs.Context(), newModelV, v); err != nil {
			return err
		}
		return r.set(newModel)
	}

	return errors.TypeMismatch.Wrapf(
		"%s is not assignable nor convertible to %s",
		typ, r.opts.typ,
	)
}

func (r *ptrField[M, V]) Scan(v any) error {
	if bitcheck.Is(r.useOpts().flags, flagIsPtrScanner) {
		this := r.val
		return r.wasSet(any(this).(sql.Scanner).Scan(v))
	}

	if bitcheck.Is(r.opts.flags, flagIsScanner) {
		if r.opts.isNil(r.val) {
			*r.val = r.newValue()
		}
		return r.wasSet(any(*r.val).(sql.Scanner).Scan(v))
	}

	wasSet, err := scan.Scan(r.val, v, scan.SF_DEFAULT)
	if err != nil {
		return err
	}
	if wasSet {
		return r.wasSet(nil)
	}

	var (
		src = reflect.ValueOf(v)
		typ = src.Type()
	)

	if typ == nil || !src.IsValid() {
		return nil
	}

	// if is model we can create new value of type V
	// and set the scanned v as it's primary key.
	// this wont work if the field type itself is an interface.
	if bitcheck.Is(r.opts.flags, flagIsModel) && r.opts.typ != _DEFINER {

		var newModelV any = r.opts.newValue()
		var newModel = reflect.ValueOf(newModelV)
		if err := attrs.SetPrimaryKey(r.defs.Context(), newModelV.(attrs.Definer), v); err != nil {
			return err
		}

		return r.set(newModel)
	}

	if typ.ConvertibleTo(r.opts.typ) {
		src = src.Convert(r.opts.typ)
		typ = src.Type()
	}

	if typ.AssignableTo(r.opts.typ) {
		return r.set(src)
	}

	return errors.TypeMismatch.Wrapf(
		"%s is not assignable nor convertible to %s",
		typ, r.opts.typ,
	)
}

// no op for now
func (r *ptrField[M, V]) wasSet(err error) error {
	if err != nil || (!bitcheck.Is(r.opts.flags, flagIsBinder) && !bitcheck.Is(r.opts.flags, flagIsPtrBinder)) {
		return err
	}

	if bitcheck.Is(r.opts.flags, flagIsPtrBinder) {
		binder := any(r.val).(attrs.Binder)
		return binder.BindToModel(r.obj, r)
	}

	if bitcheck.Is(r.opts.flags, flagIsBinder) && !r.opts.isNil(r.val) {
		binder := any(*r.val).(attrs.Binder)
		return binder.BindToModel(r.obj, r)
	}

	return nil
}

func (r *ptrField[K, V]) newValue() (v V) {
	if r.opts.newValue != nil {
		return r.opts.newValue()
	}

	// panic(fmt.Sprintf("should not be reachable: %s", r.opts.typ.Kind()))
	return v
}

func (r *ptrField[M, V]) set(rv reflect.Value) error {
	r.dst.Elem().Set(rv)
	return r.wasSet(nil)
}

func (r *ptrField[M, V]) get(val *V) V {

	if !bitcheck.Is(r.useOpts().flags, flagIsBinder) && !bitcheck.Is(r.opts.flags, flagIsPtrBinder) {
		return *val
	}

	isNil := r.opts.isNil(val)
	if isNil && bitcheck.Is(r.opts.flags, flagSetupNilVal) {
		*r.val = r.newValue()
		isNil = false
	}

	if isNil {
		return *val
	}

	if bitcheck.Is(r.opts.flags, flagIsPtrBinder) {
		if err := any(val).(attrs.Binder).BindToModel(r.obj, r); err != nil {
			r.opts.bindError(err)
		}
	}

	if bitcheck.Is(r.opts.flags, flagIsBinder) && !r.opts.isNil(r.val) {
		if err := any(*val).(attrs.Binder).BindToModel(r.obj, r); err != nil {
			r.opts.bindError(err)
		}
	}

	return *val
}

func (r *ptrField[M, V]) Value() (driver.Value, error) {
	return drivers.GetValue(*r.val)
}

func (r *ptrField[M, V]) GetValue() interface{} {
	return r.get(r.val)
}

func (r *ptrField[M, V]) GetDefault() interface{} {
	def := r.useOpts().getDefault(r.obj)

	if def == nil {
		return nil
	}

	if r.opts.isNil(def) && (bitcheck.Is(r.useOpts().flags, flagIsBinder) || bitcheck.Is(r.opts.flags, flagIsPtrBinder)) {
		d := r.opts.newValue()
		def = &d
	}

	return r.get(def)
}

func (r *ptrField[M, V]) Validate() error {
	return r.useOpts().validateValue(r.obj, *r.val)
}
