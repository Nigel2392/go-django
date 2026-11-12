package embedder

import (
	"context"
	"reflect"
	"runtime"
	"strings"
	"unsafe"

	"github.com/pkg/errors"
)

/*
type _setupEmbedded1 interface {
	<MethodName>(ctx context.Context, embedder any, structField reflect.StructField)
}
type _setupEmbedded2 interface {
	<MethodName>(ctx context.Context, embedder T, structField reflect.StructField)
}
*/

type modelRegistry struct {
	// map of underlying eface type ptr to initializer func
	typeReg map[uintptr][]*initializer

	// the name of the method to look for
	method string
}

func NewRegistry(methodName string) Registry {
	return new(modelRegistry{
		method:  methodName,
		typeReg: make(map[uintptr][]*initializer),
	})
}

// Bind ensures any embedded types can be properly initialized.
//
// when the obj param does not embed any such types, this function will be as fast as a single map read.
//
// !!! WARNING DANGER !!!
//
// Any objects that utilize the context cannot utilize the [context.Context]
// variable any longer than the lifetime of the bind function.
func (r *modelRegistry) Bind(ctx context.Context, obj any) error {
	_rootCtx := RootValueContext{
		Context: ctx,
		Root:    obj,
	}
	rootCtx := &_rootCtx

	err := r.bind(rootCtx, obj)
	runtime.KeepAlive(_rootCtx)
	return err
}

// Register an object that might have an embedded type which requires setup.
//
// If no supported embedded types are found, the type is left out of the registry.
func (r *modelRegistry) Register(obj any) {
	r.register(obj)
}

// Count returns the amount of embedders that will be initialized.
//
// The obj must be registered before the count of initializers can be retrieved.
//
// When the obj param does not embed any supported embedded types,
// this function will return 0, and be as fast as a single map read.
func (r *modelRegistry) Count(obj any) int {
	return r.countOf(obj)
}

// Walk walks the initializer tree, calling a function with the depth and structfield of each initializer.
//
// If no initializers are defined for param `obj`, this function will do nothing.
func (r *modelRegistry) Walk(obj any, walkFn func(depth int, field reflect.StructField) (stop bool)) bool {
	return r.walk(obj, walkFn)
}

type reflectValue struct {
	typ uintptr // type
	_   uintptr // data
	_   uintptr // flag
}

type eface struct {
	typ  uintptr
	data unsafe.Pointer
}

func keyOf[T any]() uintptr {
	var v any = *new(T)
	return (*eface)(unsafe.Pointer(&v)).typ
}

var (
	_reflect_keyOfValue      = keyOf[reflect.Value]()
	_type_reflectValue       = reflect.TypeFor[reflect.Value]()
	_type_contextContext     = reflect.TypeFor[context.Context]()
	_type_reflectStructField = reflect.TypeFor[reflect.StructField]()
)

// countOf returns the amount of embedders that will be initialized
func (r *modelRegistry) countOf(obj any) int {
	var (
		face         = (*eface)(unsafe.Pointer(&obj))
		key  uintptr = face.typ
	)

	if key == _reflect_keyOfValue { // faster than type switches
		// [reflect.Value] type pointer is equal to [eface.typ] of the same value
		key = (*reflectValue)(face.data).typ
	}

	inits, ok := r.typeReg[key]
	if !ok {
		return 0
	}

	var count int
	for _, init := range inits {
		count += init.count()
	}

	return count
}

// walk takes a function and will walk each initializer
func (r *modelRegistry) walk(obj any, walkFn func(depth int, field reflect.StructField) (stop bool)) (stopped bool) {
	var (
		face         = (*eface)(unsafe.Pointer(&obj))
		key  uintptr = face.typ
	)

	if key == _reflect_keyOfValue { // faster than type switches
		// [reflect.Value] type pointer is equal to [eface.typ] of the same value
		key = (*reflectValue)(face.data).typ
	}

	inits, ok := r.typeReg[key]
	if !ok {
		return false
	}

	var depth = 0
	for _, init := range inits {
		if init.walk(depth, walkFn) {
			return true
		}
	}

	return false
}

// setup ensures any embedded types can be properly initialized
//
// when the obj does not embed any such types, this function will be as fast as a single map read
func (r *modelRegistry) bind(ctx context.Context, obj any) error {
	var (
		face         = (*eface)(unsafe.Pointer(&obj))
		key  uintptr = face.typ
		rv   reflect.Value
		isRv bool
	)

	if key == _reflect_keyOfValue { // faster than type switches
		isRv = true
		rv = *(*reflect.Value)(face.data)

		// [reflect.Value] type pointer is equal to [eface.typ] of the same value
		key = (*reflectValue)(face.data).typ
	}

	inits, ok := r.typeReg[key]
	if !ok || len(inits) == 0 {
		return nil
	}

	// key was not equal to [reflect_keyOfValue]
	if !isRv {
		rv = reflect.ValueOf(obj)
	}

	_ctx := reflect.ValueOf(ctx)
	el := rv.Elem()
	for _, i := range inits {
		pathReverse, err := i.bind(_ctx, rv, el)
		if err != nil {
			var sb strings.Builder
			for i := len(pathReverse) - 1; i >= 0; i-- {
				sb.WriteString(pathReverse[i])
				if i > 0 {
					sb.WriteRune('.')
				}
			}

			return errors.Wrap(
				err,
				sb.String(),
			)
		}
	}

	return nil
}

func (r *modelRegistry) register(obj any) {
	var (
		key     = (*eface)(unsafe.Pointer(&obj)).typ
		objT    = reflect.TypeOf(obj)
		objElem = objT
	)

	for objElem.Kind() == reflect.Pointer {
		objElem = objElem.Elem()
	}

	var (
		initialisers = make([]*initializer, 0)
		seen         = make(map[reflect.Type]struct{})
	)
	seen[objElem] = struct{}{}
	for index := range objElem.NumField() {

		init := new(initializer{
			sub: make([]*initializer, 0),
		})

		if objElem.Kind() == reflect.Interface {
			continue
		}

		if !registerInitializer(objElem, init, []int{index}, seen, r.method) {
			continue
		}

		initialisers = append(initialisers, init)
	}

	if len(initialisers) == 0 {
		return
	}

	r.typeReg[key] = initialisers
}

func registerInitializer(objElem reflect.Type, i *initializer, index []int, seen map[reflect.Type]struct{}, methodName string) (ok bool) {

	// fmt.Printf("at %s %v\n", objElem, index)

	field := objElem.FieldByIndex(index)
	field.Index = index
	fieldType := field.Type

	// fmt.Printf("walking %s => %s\n", objElem, fieldType)

	var isPtrFld bool
	if fieldType.Kind() == reflect.Pointer {
		fieldType = fieldType.Elem()
		isPtrFld = true
	}

	if _, ok := seen[fieldType]; ok {
		// fmt.Println("seen, returning", fieldType)
		return false
	}

	seen[fieldType] = struct{}{}

	if field.Tag.Get("embedder") == "-" {
		return false
	}

	if fieldType.Kind() == reflect.Interface {
		return false
	}

	if fieldType.Kind() != reflect.Struct {
		goto checkHasSetup
	}

	if isPtrFld {
		for currIdx := range fieldType.NumField() {
			var ni = new(initializer)
			if registerInitializer(objElem, ni, append(index, currIdx), seen, methodName) {
				ok = true
				i.embedded = field
				i.sub = append(i.sub, ni)
			}
		}

		if ok {
			return true
		}
	}

	// check if directly defined on type
checkHasSetup:
	method, ok := field.Type.MethodByName(methodName)
	if ok {
		i.embedded = field
		i.node = &initNode{
			method: method,
		}
		setupInitMethod(i)
		return true
	}

	var fl flags
	if field.Type.Kind() == reflect.Pointer {
		fl |= flagElemMethod
		method, ok = field.Type.
			Elem().
			MethodByName(methodName)
	} else {
		fl |= flagAddrMethod
		// check pointer of value type
		method, ok = reflect.
			PointerTo(field.Type).
			MethodByName(methodName)
	}

	if ok {
		i.embedded = field
		i.node = &initNode{
			method: method,
			flags:  fl,
		}
		setupInitMethod(i)
		return true
	}

	return false
}

func setupInitMethod(i *initializer) {
	if i.node.method.Type.NumOut() > 1 {
		panic("invalid SetupEmbedded function signature: no more than 1 error return parameter expected")
	}

	var (
		maxNum = 4
		valIdx = 2
	)
	if i.node.method.Type.In(1) != _type_contextContext {
		maxNum--
		valIdx--
		i.node.flags |= flagNoCtx
		// panic("invalid SetupEmbedded function signature: expected context.Context as first param")
	}

	if i.node.method.Type.NumIn() > maxNum {
		panic("invalid SetupEmbedded function signature: too many arguments provided")
	}

	if i.node.method.Type.NumIn() == maxNum && i.node.method.Type.In(maxNum-1) != _type_reflectStructField {
		panic("invalid SetupEmbedded function signature: expected none or reflect.StructField as last param")
	}

	if numOut := i.node.method.Type.NumOut(); numOut > 0 && i.node.method.Type.Out(numOut-1).AssignableTo(reflect.TypeFor[error]()) {
		i.node.flags |= flagMethodErr
	}

	if i.node.method.Type.In(valIdx) == _type_reflectValue {
		i.node.flags |= flagReflectValue
	}
}
