package telepath

import (
	"fmt"
	"reflect"

	"github.com/Nigel2392/django/forms/media"
)

var _ Context = (*ValueContext)(nil)

type JSContext struct {
	Media           media.Media
	AdapterRegistry *AdapterRegistry
}

func (c *JSContext) AddMedia(media media.Media) {
	c.Media = media.Merge(c.Media)
}

func (c *JSContext) Registry() *AdapterRegistry {
	return c.AdapterRegistry
}

func (c *JSContext) Pack(value interface{}) (interface{}, error) {
	var newCtx = NewValueContext(c)
	var v, err = newCtx.BuildNode(value)
	if err != nil {
		return nil, err
	}
	return v.Emit(), nil
}

type ValueContext struct {
	ParentContext   *JSContext
	AdapterRegistry *AdapterRegistry
	RawValues       map[uintptr]any
	Nodes           map[uintptr]Node
	NextID          int
}

func NewValueContext(c *JSContext) *ValueContext {
	return &ValueContext{
		ParentContext:   c,
		AdapterRegistry: c.Registry(),
		RawValues:       make(map[uintptr]any),
		Nodes:           make(map[uintptr]Node),
	}
}

func (c *ValueContext) AddMedia(media media.Media) {
	c.ParentContext.AddMedia(media)
}

func (c *ValueContext) Registry() *AdapterRegistry {
	return c.AdapterRegistry
}

func (c *ValueContext) buildNewNode(value interface{}) (Node, error) {

	var adapter, ok = c.AdapterRegistry.Find(value)
	if ok {
		return adapter.BuildNode(value, c)
	}

	var v = reflect.ValueOf(value)
	var rTyp = v.Type()

	switch rTyp.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Bool, reflect.Invalid:
		return BaseAdapter().BuildNode(value, c)
	case reflect.Slice, reflect.Array:
		var n, err = SliceAdapter().BuildNode(value, c)
		return n, err
	case reflect.Map:
		return MapAdapter().BuildNode(value, c)
	case reflect.String:
		return StringAdapter().BuildNode(value, c)
	}

	return nil, fmt.Errorf("no adapter found for value %v", value)
}

func (c *ValueContext) BuildNode(value interface{}) (Node, error) {
	var node, err = c.buildNewNode(value)
	if err != nil {
		return nil, err
	}

	return node, nil
}
