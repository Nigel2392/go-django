package embedder

import (
	"reflect"
	"unsafe"
)

type flags uint32

const (
	flagNone       flags = iota
	flagAddrMethod flags = 1 << iota
	flagElemMethod
	flagReflectValue
	flagMethodErr
	flagNoCtx
)

type initNode struct {
	flags  flags
	method reflect.Method
}

type initializer struct {
	embedded reflect.StructField
	node     *initNode
	sub      []*initializer
}

func (r *initializer) count() int {
	var count int
	if r.node != nil {
		count++
	}
	for _, init := range r.sub {
		count += init.count()
	}
	return count
}

func (r *initializer) walk(depth int, walkFn func(depth int, field reflect.StructField) (stop bool)) (stopped bool) {
	if walkFn(depth, r.embedded) {
		return true
	}

	for _, sub := range r.sub {
		if sub.walk(depth+1, walkFn) {
			return true
		}
	}

	return false
}

func (i *initializer) bind(ctx, rv, el reflect.Value) (pathReverse []string, err error) {
	for _, init := range i.sub {
		pathReverse, err = init.bind(ctx, rv, el)
		if err != nil {
			return append(pathReverse, i.embedded.Name), err
		}
	}

	if i.node == nil {
		return nil, nil
	}

	// fmt.Printf("node %v for embedder %s.%s\n", i.node.embedded.Index, parent.Type(), i.node.embedded.Name)

	// fmt.Printf("\t%s\n\t%s\n\t%#v METHODADDR(%v)\n\n", val.Type(), parent.Type(), i, i.node.addrMethod)

	// walks the structfield index and sets any nil values
	parent, val := FieldByIndexNew(rv, i.embedded.Index)

	if i.node.flags&flagAddrMethod == flagAddrMethod {
		val = val.Addr()
	}

	// set embedded value to non-nil if is nil
	if val.Kind() == reflect.Pointer && val.IsNil() {
		if !val.CanSet() {
			panic("embedder field is not addressable or is unexported")
		}
		// Allocate a new zero-initialized instance of the pointed-to type
		newVal := reflect.New(val.Type().Elem())
		val.Set(newVal)
	}

	if i.node.flags&flagReflectValue == flagReflectValue {
		parent = reflect.ValueOf(parent)
	}

	var (
		out []reflect.Value
		nIn = i.node.method.Type.NumIn()
	)

	switch {
	case i.node.flags&flagNoCtx == flagNoCtx && nIn == 3:
		out = i.node.method.Func.Call([]reflect.Value{
			val, parent,
			reflect.ValueOf(i.embedded),
		})
	case i.node.flags&flagNoCtx == flagNoCtx:
		out = i.node.method.Func.Call([]reflect.Value{
			val, parent,
		})
	case i.node.flags&flagNoCtx != flagNoCtx && nIn == 4:
		out = i.node.method.Func.Call([]reflect.Value{
			val, ctx, parent,
			reflect.ValueOf(i.embedded),
		})
	case i.node.flags&flagNoCtx != flagNoCtx:
		out = i.node.method.Func.Call([]reflect.Value{
			val, ctx, parent,
		})
	}

	if i.node.flags&flagMethodErr == flagMethodErr {
		errV := out[len(out)-1]
		if !errV.IsValid() || errV.IsNil() {
			return nil, nil
		}

		return []string{i.embedded.Name}, errV.Interface().(error)
	}

	return nil, nil
}

type FieldValue struct {
	Struct reflect.Value
	Field  reflect.Value
}

func FieldByIndexNew(val reflect.Value, index []int) (parent, value reflect.Value) {
	parent = val

	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}

	for i, x := range index {

		if i > 0 {
			// set parent of val
			parent = val

			// set nil pointers
			if val.Kind() == reflect.Pointer && val.Type().Elem().Kind() == reflect.Struct {
				if val.IsNil() {
					if !val.CanSet() {
						panic("embedder field is not addressable or is unexported")
					}
					newVal := reflect.New(val.Type().Elem())
					val.Set(newVal)
				}
				val = val.Elem()
			}
		}

		// walk the structs' field tree
		val = val.Field(x)

		// removes the RO flag from unexported reflect.Value
		val = reflect.NewAt(val.Type(), unsafe.Pointer(val.UnsafeAddr())).Elem()
	}

	return parent, val
}
