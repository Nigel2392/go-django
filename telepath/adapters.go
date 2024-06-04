package telepath

import (
	"fmt"
	"reflect"

	"github.com/Nigel2392/django/forms/media"
)

type BaseTelepathAdapter struct {
}

func BaseAdapter() *BaseTelepathAdapter {
	return &BaseTelepathAdapter{}
}

func (m *BaseTelepathAdapter) BuildNode(value any, c Context) (Node, error) {
	return NewTelepathValueNode(value), nil
}

type StringTelepathAdapter struct {
}

func StringAdapter() *StringTelepathAdapter {
	return &StringTelepathAdapter{}
}

func (m *StringTelepathAdapter) BuildNode(value any, c Context) (Node, error) {
	return NewStringNode(value), nil
}

type SliceTelepathAdapter struct {
}

func SliceAdapter() *SliceTelepathAdapter {
	return &SliceTelepathAdapter{}
}

func (m *SliceTelepathAdapter) BuildNode(value any, c Context) (Node, error) {

	var (
		rTyp = reflect.TypeOf(value)
		rVal = reflect.ValueOf(value)
	)

	if rTyp.Kind() != reflect.Slice {
		return nil, fmt.Errorf("value is not a slice")
	}

	var nodes = make([]Node, 0, rVal.Len())
	for i := 0; i < rVal.Len(); i++ {
		var (
			item      = rVal.Index(i).Interface()
			node, err = c.BuildNode(item)
		)

		if err != nil {
			return nil, err
		}

		nodes = append(nodes, node)
	}

	return NewListNode(nodes), nil

}

type MapTelepathAdapter struct {
}

func MapAdapter() *MapTelepathAdapter {
	return &MapTelepathAdapter{}
}

func (m *MapTelepathAdapter) BuildNode(value any, c Context) (Node, error) {
	var (
		rTyp  = reflect.TypeOf(value)
		rVal  = reflect.ValueOf(value)
		nodes = make(map[string]Node)
	)

	if rTyp.Kind() != reflect.Map {
		return nil, fmt.Errorf("value is not a map: %v", rTyp.Kind())
	}

	for _, key := range rVal.MapKeys() {
		var (
			item      = rVal.MapIndex(key).Interface()
			node, err = c.BuildNode(item)
		)

		if err != nil {
			return nil, err
		}

		nodes[key.String()] = node
	}

	return NewDictNode(nodes), nil
}

type AutoTelepathAdapter struct {
}

func AutoAdapter() *AutoTelepathAdapter {
	return &AutoTelepathAdapter{}
}

func (m *AutoTelepathAdapter) BuildNode(value any, c Context) (Node, error) {
	var rTyp = reflect.TypeOf(value)
	var rVal = reflect.ValueOf(value)
	switch rTyp.Kind() {
	case reflect.String:
		return NewStringNode(value), nil
	case reflect.Slice:
		var nodes = make([]Node, 0, rVal.Len())
		for i := 0; i < rVal.Len(); i++ {
			var (
				item      = rVal.Index(i).Interface()
				node, err = c.BuildNode(item)
			)
			if err != nil {
				return nil, err
			}

			nodes = append(nodes, node)
		}
		return NewListNode(nodes), nil
	case reflect.Map:
		var nodes = make(map[string]Node)
		for _, key := range reflect.ValueOf(value).MapKeys() {
			var (
				item      = reflect.ValueOf(value).MapIndex(key).Interface()
				node, err = c.BuildNode(item)
			)

			if err != nil {
				return nil, err
			}

			nodes[key.String()] = node
		}
		return NewDictNode(nodes), nil
	default:
		return nil, fmt.Errorf("unsupported type %v", rTyp)
	}
}

type TelepathAdapter struct {
	JSConstructor string
	GetJSArgs     func(obj interface{}) []interface{}
}

func NewTelepathAdapter() *TelepathAdapter {
	return &TelepathAdapter{}
}

func (m *TelepathAdapter) GetMedia(obj interface{}) media.Media {
	return media.NewMedia()
}

func (m *TelepathAdapter) JSArgs(obj interface{}) []interface{} {
	if m.GetJSArgs != nil {
		return m.GetJSArgs(obj)
	} else {
		return make([]interface{}, 0)
	}
}

func (m *TelepathAdapter) Pack(obj interface{}, context Context) (string, []interface{}) {

	context.AddMedia(
		m.GetMedia(obj),
	)

	return m.JSConstructor, m.JSArgs(obj)
}

func (m *TelepathAdapter) BuildNode(value any, c Context) (Node, error) {
	var constructor, args = m.Pack(value, c)
	var newArgs = make([]Node, 0, len(args))
	for _, arg := range args {
		var node, err = c.BuildNode(arg)
		if err != nil {
			panic(err)
		}
		newArgs = append(newArgs, node)
	}
	return NewObjectNode(constructor, newArgs), nil
}
