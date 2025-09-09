package queries

import (
	"fmt"
	"iter"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-signals"
	"github.com/elliotchance/orderedmap/v2"
)

const _PROXY_FIELDS_KEY = "models.embed.proxy.fields"

// Build out the proxy field map when a model is registered.
//
// This will create a tree structure that contains all the proxy fields
// and their respective sub-proxy fields.
var _, _ = attrs.OnModelRegister.Listen(func(s signals.Signal[attrs.SignalModelMeta], meta attrs.SignalModelMeta) error {
	var newDefiner = attrs.NewObject[attrs.Definer](meta.Definer)
	var proxyFields = buildProxyFieldMap(newDefiner)
	attrs.StoreOnMeta(
		meta.Definer,
		_PROXY_FIELDS_KEY,
		proxyFields,
	)

	return nil
})

type ProxyFieldObject struct {
	Parent      attrs.Definer
	ParentField attrs.Field
	Path        []string
	Field       ProxyField
	Value       attrs.Definer
}

type proxyTree struct {
	object  attrs.Definer
	defs    attrs.Definitions
	proxies *orderedmap.OrderedMap[string, *proxyFieldNode]
	fields  *orderedmap.OrderedMap[string, ProxyField]
}

func (n *proxyTree) WalkObjectProxies(instance attrs.Definer, walkZeroValues bool) iter.Seq2[ProxyFieldObject, error] {
	return func(yield func(ProxyFieldObject, error) bool) {
		n.walkObjectProxies(instance, []string{}, walkZeroValues, nil, yield)
	}
}

func (n *proxyTree) walkObjectProxies(instance attrs.Definer, path []string, walkZeroValues bool, parent attrs.Definer, yield func(ProxyFieldObject, error) bool) bool {
	for head := n.fields.Front(); head != nil; head = head.Next() {
		var (
			fieldName = head.Key
			fieldNode = head.Value
		)

		var defs = instance.FieldDefs()
		var field, ok = defs.Field(fieldName)
		if !ok {
			return yield(ProxyFieldObject{}, errors.FieldNotFound.Wrapf(
				"field %s not found on instance of type %T", fieldName, instance,
			))
		}

		var value = field.GetValue()
		var isZero = attrs.IsZero(value)
		if isZero && !walkZeroValues {
			continue
		}

		var proxyInstance attrs.Definer
		if !isZero {
			proxyInstance, ok = value.(attrs.Definer)
			if !ok {
				return yield(ProxyFieldObject{}, errors.TypeMismatch.Wrapf(
					"expected field %s to be of type attrs.Definer, got %T",
					fieldName, value,
				))
			}
		} else {
			proxyInstance = attrs.NewObject[attrs.Definer](field.Type())
		}

		var obj = ProxyFieldObject{
			Path:        append(path, fieldName),
			Field:       fieldNode,
			Parent:      parent,
			ParentField: field,
			Value:       proxyInstance,
		}

		if !yield(obj, nil) {
			return false
		}

		var subTree, exists = n.proxies.Get(fieldName)
		if !exists {
			continue
		}

		if !subTree.tree.walkObjectProxies(proxyInstance, obj.Path, walkZeroValues, proxyInstance, yield) {
			return false
		}
	}
	return true
}

type proxyFieldNode struct {
	sourceField ProxyField
	tree        *proxyTree
}

func (n *proxyTree) Object() attrs.Definer {
	return n.object
}

func (n *proxyTree) FieldsLen() int {
	return n.fields.Len()
}

func (n *proxyTree) hasProxies() bool {
	return n.proxies.Len() > 0 || n.fields.Len() > 0
}

func (n *proxyTree) Get(name string) (ProxyField, bool) {
	return n.fields.Get(name)
}

func (n *proxyTree) IsProxyField(fld any) bool {
	switch v := fld.(type) {
	case ProxyField:
		return v.IsProxy()
	case attrs.Field:
		_, ok := n.fields.Get(v.Name())
		return ok
	case string:
		_, ok := n.fields.Get(v)
		return ok
	default:
		panic(fmt.Sprintf("unknown type %T for IsProxyField", fld))
	}
}

func buildProxyFieldMap(definer attrs.Definer) *proxyTree {
	if attrs.IsModelRegistered(definer) {
		var (
			meta     = attrs.GetModelMeta(definer)
			vals, ok = meta.Storage(_PROXY_FIELDS_KEY)
		)
		if ok {
			return vals.(*proxyTree)
		}
	}

	var newDefiner = attrs.NewObject[attrs.Definer](definer)
	var node = &proxyTree{
		object:  newDefiner,
		defs:    newDefiner.FieldDefs(),
		fields:  orderedmap.NewOrderedMap[string, ProxyField](),
		proxies: orderedmap.NewOrderedMap[string, *proxyFieldNode](),
	}
	for _, field := range node.defs.Fields() {
		var proxyField, ok = field.(ProxyField)
		if !ok || !proxyField.IsProxy() {
			continue
		}

		node.fields.Set(
			field.Name(),
			proxyField,
		)

		var rel = field.Rel()
		var relType = rel.Type()
		if relType == attrs.RelOneToOne || relType == attrs.RelManyToOne {
			var model = rel.Model()
			var subTree = buildProxyFieldMap(model)
			if !subTree.hasProxies() {
				continue
			}

			node.proxies.Set(
				field.Name(),
				&proxyFieldNode{
					tree:        subTree,
					sourceField: proxyField,
				},
			)

			continue
		}
	}

	return node
}

func ProxyFields(definer attrs.Definer) *proxyTree {
	if !attrs.IsModelRegistered(definer) {
		return &proxyTree{
			object:  definer,
			fields:  orderedmap.NewOrderedMap[string, ProxyField](),
			proxies: orderedmap.NewOrderedMap[string, *proxyFieldNode](),
		}
	}

	var (
		meta     = attrs.GetModelMeta(definer)
		vals, ok = meta.Storage(_PROXY_FIELDS_KEY)
	)

	if !ok {
		panic(fmt.Errorf("no proxy fields found for model %T", definer))
	}

	return vals.(*proxyTree)
}
