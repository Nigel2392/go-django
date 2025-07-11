package yml

import (
	"fmt"
	"iter"
	"reflect"

	"github.com/elliotchance/orderedmap/v2"
	"gopkg.in/yaml.v3"
)

type OrderedMap[K comparable, V any] struct {
	*yaml.Node                   `yaml:",inline"`
	*orderedmap.OrderedMap[K, V] `yaml:"-"`
}

func (n *OrderedMap[K, V]) Iter() iter.Seq2[K, V] {
	if n.OrderedMap == nil {
		return func(yield func(K, V) bool) {}
	}

	return func(yield func(K, V) bool) {
		for front := n.OrderedMap.Front(); front != nil; front = front.Next() {
			if !yield(front.Key, front.Value) {
				break
			}
		}
	}
}

func reflectScanNode[T any](valueNode *yaml.Node, rValue reflect.Value) error {
	if rValue.Kind() == reflect.Pointer {
		return valueNode.Decode(rValue.Interface())
	}
	return valueNode.Decode(rValue.Addr().Interface())
}

func (n *OrderedMap[K, V]) scanNode(keyNode *yaml.Node, valueNode *yaml.Node) error {
	var (
		kV K
		vV V
		// make sure the values are adressable
		rKey   = reflect.ValueOf(&kV).Elem()
		rValue = reflect.ValueOf(&vV).Elem()
	)

	switch rValue.Kind() {
	case reflect.Slice:
		rValue = reflect.MakeSlice(rValue.Type(), 0, 0)
	case reflect.Map:
		rValue = reflect.MakeMap(rValue.Type())
	}

	for _, v := range []*reflect.Value{&rKey, &rValue} {
		if v.Kind() == reflect.Ptr {
			*v = reflect.New(v.Type().Elem())
		}
	}

	if err := reflectScanNode[K](keyNode, rKey); err != nil {
		return fmt.Errorf("error scanning key node %s: %w", keyNode.Value, err)
	}

	if err := reflectScanNode[V](valueNode, rValue); err != nil {
		return fmt.Errorf("error scanning value node %s: %w", valueNode.Value, err)
	}

	n.OrderedMap.Set(
		rKey.Interface().(K),
		rValue.Interface().(V),
	)
	return nil
}

func (n *OrderedMap[K, V]) unmarshalYAML_Mapping(node *yaml.Node) error {
	for i := 0; i < len(node.Content); i += 2 {
		var keyNode = node.Content[i]
		var valueNode = node.Content[i+1]
		if keyNode.Kind != yaml.ScalarNode {
			return fmt.Errorf("expected scalar node for key %s, got %d (%+v)", keyNode.Value, keyNode.Kind, keyNode)
		}
		if keyNode.Tag != "!!str" {
			return fmt.Errorf("expected string tag for key, got %s", keyNode.Tag)
		}

		if err := n.scanNode(keyNode, valueNode); err != nil {
			return fmt.Errorf("error scanning node %s: %w", keyNode.Value, err)
		}
	}

	return nil
}

func (n *OrderedMap[K, V]) unmarshalYAML_Sequence(node *yaml.Node) error {
	for i := 0; i < len(node.Content); i++ {
		var root = node.Content[i]
		var keyNode *yaml.Node
		var valueNode *yaml.Node
		switch root.Kind {
		case yaml.MappingNode:
			if len(root.Content) != 2 {
				return fmt.Errorf("expected 2 content items in sequence, got %d", len(root.Content))
			}
			keyNode = root.Content[0]
			valueNode = root.Content[1]
		case yaml.ScalarNode:
			if len(root.Content) != 1 {
				return fmt.Errorf("expected 1 content item in sequence, got %d", len(root.Content))
			}
			keyNode = root
			valueNode = root
		default:
			return fmt.Errorf("unexpected node kind in OrderedMap: %+v: %w", root, ErrInvalidKind)
		}

		if keyNode.Kind != yaml.ScalarNode {
			return fmt.Errorf("expected scalar node for key %s, got %d (%+v)", keyNode.Value, keyNode.Kind, keyNode)
		}

		if keyNode.Tag != "!!str" {
			return fmt.Errorf("expected string tag for key, got %s", keyNode.Tag)
		}

		if err := n.scanNode(keyNode, valueNode); err != nil {
			return fmt.Errorf("error scanning node %s: %w", keyNode.Value, err)
		}
	}

	return nil
}

func (n *OrderedMap[K, V]) UnmarshalYAML(node *yaml.Node) error {

	if n.OrderedMap == nil {
		n.OrderedMap = orderedmap.NewOrderedMap[K, V]()
	}

	n.Node = node

	switch node.Kind {
	case yaml.MappingNode:
		return n.unmarshalYAML_Mapping(node)
	case yaml.SequenceNode:
		return n.unmarshalYAML_Sequence(node)
	}

	return ErrInvalidKind
}
