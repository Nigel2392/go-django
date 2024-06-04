package telepath

import (
	"reflect"

	"github.com/Nigel2392/django/forms/media"
)

var (
	specificAdapterMap = make(map[reflect.Kind]map[reflect.Type]Adapter)
	defaultAdapterMap  = make(map[reflect.Kind]Adapter)
)

func init() {
	var (
		rTypBool = reflect.TypeOf(bool(false))

		rTypInt   = reflect.TypeOf(int(0))
		rTypInt8  = reflect.TypeOf(int8(0))
		rTypInt16 = reflect.TypeOf(int16(0))
		rTypInt32 = reflect.TypeOf(int32(0))
		rTypInt64 = reflect.TypeOf(int64(0))

		rTypUint   = reflect.TypeOf(uint(0))
		rTypUint8  = reflect.TypeOf(uint8(0))
		rTypUint16 = reflect.TypeOf(uint16(0))
		rTypUint32 = reflect.TypeOf(uint32(0))
		rTypUint64 = reflect.TypeOf(uint64(0))

		rTypFloat32 = reflect.TypeOf(float32(0))
		rTypFloat64 = reflect.TypeOf(float64(0))

		rTypString = reflect.TypeOf(string(""))

		rTypSlice = reflect.TypeOf([]interface{}{})
		rTypMap   = reflect.TypeOf(map[string]interface{}{})
	)

	specificAdapterMap[rTypBool.Kind()] = make(map[reflect.Type]Adapter)
	specificAdapterMap[rTypBool.Kind()][rTypBool] = BaseAdapter()
	defaultAdapterMap[rTypBool.Kind()] = BaseAdapter()

	specificAdapterMap[rTypInt.Kind()] = make(map[reflect.Type]Adapter)
	specificAdapterMap[rTypInt.Kind()][rTypInt] = BaseAdapter()
	defaultAdapterMap[rTypInt.Kind()] = BaseAdapter()

	specificAdapterMap[rTypInt8.Kind()] = make(map[reflect.Type]Adapter)
	specificAdapterMap[rTypInt8.Kind()][rTypInt8] = BaseAdapter()
	defaultAdapterMap[rTypInt8.Kind()] = BaseAdapter()

	specificAdapterMap[rTypInt16.Kind()] = make(map[reflect.Type]Adapter)
	specificAdapterMap[rTypInt16.Kind()][rTypInt16] = BaseAdapter()
	defaultAdapterMap[rTypInt16.Kind()] = BaseAdapter()

	specificAdapterMap[rTypInt32.Kind()] = make(map[reflect.Type]Adapter)
	specificAdapterMap[rTypInt32.Kind()][rTypInt32] = BaseAdapter()
	defaultAdapterMap[rTypInt32.Kind()] = BaseAdapter()

	specificAdapterMap[rTypInt64.Kind()] = make(map[reflect.Type]Adapter)
	specificAdapterMap[rTypInt64.Kind()][rTypInt64] = BaseAdapter()
	defaultAdapterMap[rTypInt64.Kind()] = BaseAdapter()

	specificAdapterMap[rTypUint.Kind()] = make(map[reflect.Type]Adapter)
	specificAdapterMap[rTypUint.Kind()][rTypUint] = BaseAdapter()
	defaultAdapterMap[rTypUint.Kind()] = BaseAdapter()

	specificAdapterMap[rTypUint8.Kind()] = make(map[reflect.Type]Adapter)
	specificAdapterMap[rTypUint8.Kind()][rTypUint8] = BaseAdapter()
	defaultAdapterMap[rTypUint8.Kind()] = BaseAdapter()

	specificAdapterMap[rTypUint16.Kind()] = make(map[reflect.Type]Adapter)
	specificAdapterMap[rTypUint16.Kind()][rTypUint16] = BaseAdapter()
	defaultAdapterMap[rTypUint16.Kind()] = BaseAdapter()

	specificAdapterMap[rTypUint32.Kind()] = make(map[reflect.Type]Adapter)
	specificAdapterMap[rTypUint32.Kind()][rTypUint32] = BaseAdapter()
	defaultAdapterMap[rTypUint32.Kind()] = BaseAdapter()

	specificAdapterMap[rTypUint64.Kind()] = make(map[reflect.Type]Adapter)
	specificAdapterMap[rTypUint64.Kind()][rTypUint64] = BaseAdapter()
	defaultAdapterMap[rTypUint64.Kind()] = BaseAdapter()

	specificAdapterMap[rTypFloat32.Kind()] = make(map[reflect.Type]Adapter)
	specificAdapterMap[rTypFloat32.Kind()][rTypFloat32] = BaseAdapter()
	defaultAdapterMap[rTypFloat32.Kind()] = BaseAdapter()

	specificAdapterMap[rTypFloat64.Kind()] = make(map[reflect.Type]Adapter)
	specificAdapterMap[rTypFloat64.Kind()][rTypFloat64] = BaseAdapter()
	defaultAdapterMap[rTypFloat64.Kind()] = BaseAdapter()

	specificAdapterMap[rTypString.Kind()] = make(map[reflect.Type]Adapter)
	specificAdapterMap[rTypString.Kind()][rTypString] = StringAdapter()
	defaultAdapterMap[rTypString.Kind()] = StringAdapter()

	specificAdapterMap[rTypSlice.Kind()] = make(map[reflect.Type]Adapter)
	specificAdapterMap[rTypSlice.Kind()][rTypSlice] = SliceAdapter()
	defaultAdapterMap[rTypSlice.Kind()] = SliceAdapter()

	specificAdapterMap[rTypMap.Kind()] = make(map[reflect.Type]Adapter)
	specificAdapterMap[rTypMap.Kind()][rTypMap] = MapAdapter()
	defaultAdapterMap[rTypMap.Kind()] = MapAdapter()

}

type AdapterRegistry struct {
	adapters map[reflect.Kind]map[reflect.Type]Adapter
	defaults map[reflect.Kind]Adapter
}

func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{
		adapters: specificAdapterMap,
		defaults: defaultAdapterMap,
	}
}

func (r *AdapterRegistry) RegisterAdapter(k reflect.Kind, t reflect.Type, a Adapter) {
	if _, ok := r.adapters[k]; !ok {
		r.adapters[k] = make(map[reflect.Type]Adapter)
	}

	r.adapters[k][t] = a
}

func (r *AdapterRegistry) RegisterDefaultAdapter(k reflect.Kind, a Adapter) {
	r.defaults[k] = a
}

func (r *AdapterRegistry) Context() *JSContext {
	var c = &JSContext{
		Media:           media.NewMedia(),
		AdapterRegistry: r,
	}
	return c
}

func (r *AdapterRegistry) Register(a any, forType ...interface{}) {
	var v interface{}

	if len(forType) == 0 {
		v = a
	} else {
		v = forType[0]
	}

	if getter, ok := v.(AdapterGetter); ok {
		a = getter.Adapter()
	}

	var adapter = a.(Adapter)

	var t = reflect.TypeOf(v)
	var k = t.Kind()

	r.RegisterAdapter(k, t, adapter)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		r.RegisterAdapter(t.Kind(), t, adapter)
	}
}

func (r *AdapterRegistry) Find(value interface{}) (Adapter, bool) {
	var v = reflect.ValueOf(value)
	var k = v.Kind()
	var t = v.Type()

	if _, ok := r.adapters[k]; ok {
		if a, ok := r.adapters[k][t]; ok {
			return a, true
		} else {
			if a, ok := r.defaults[k]; ok {
				return a, true
			}
		}
	} else {
		if a, ok := r.defaults[k]; ok {
			return a, true
		}
	}

	return nil, false
}
