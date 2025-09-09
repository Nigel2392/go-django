package revisions

import (
	"context"
	"encoding/json"
	"maps"
	"reflect"
	"slices"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
)

type QueryInfoFunc func(ctx context.Context, obj attrs.Definer) (pk any, contentType string, err error)

type MarshallerOption func(*marshallerOption)

type RevisionInfoDefiner interface {
	GetRevisionInfo(ctx context.Context) (pk any, contentType string, err error)
}

type RevisionDataMarshaller interface {
	MarshalRevisionData() (map[string]any, error)
}

type RevisionDataUnMarshaller interface {
	UnmarshalRevisionData(data []byte) error
}

type RevisionDataFieldExcluder interface {
	ExcludeFromRevisionData() []string
}

func MarshallerSkipProxyFields(b bool) MarshallerOption {
	return func(o *marshallerOption) {
		o.SkipProxyFields = b
	}
}

func MarshallerAllowReadOnlyFields(b bool) MarshallerOption {
	return func(o *marshallerOption) {
		o.AllowReadOnlyFields = b
	}
}

func MarshallerGetFieldValue(fn func(def attrs.Field) (any, error)) MarshallerOption {
	return func(o *marshallerOption) {
		o.GetFieldValue = fn
	}
}

type marshallerOption struct {
	SkipProxyFields     bool
	AllowReadOnlyFields bool
	GetFieldValue       func(def attrs.Field) (any, error)
	AllowDataMarshaller bool
}

func newMarshallerOptions(opts ...MarshallerOption) *marshallerOption {
	var o = &marshallerOption{}
	for _, fn := range opts {
		fn(o)
	}
	return o
}

func (m *marshallerOption) getFieldValue(field attrs.Field) (any, error) {
	if m.GetFieldValue != nil {
		return m.GetFieldValue(field)
	}
	return field.Value()
}

func getExcludedFields(obj attrs.Definer) map[string]struct{} {
	var excludedMap map[string]struct{}
	if excluder, ok := obj.(RevisionDataFieldExcluder); ok {
		var excluded = excluder.ExcludeFromRevisionData()
		excludedMap = make(map[string]struct{}, len(excluded))
		for _, name := range excluded {
			excludedMap[name] = struct{}{}
		}
	}
	return excludedMap
}

func getIdAndContentType(ctx context.Context, obj attrs.Definer, getter ...QueryInfoFunc) (pk string, contentType string, err error) {
	var (
		objKey any
		fn     QueryInfoFunc
	)

	if oi, ok := obj.(RevisionInfoDefiner); ok {
		objKey, contentType, err = oi.GetRevisionInfo(ctx)
		if err != nil {
			return "", "", err
		}
		goto marshalKey
	}

	if len(getter) > 0 && getter[0] != nil {
		fn = getter[0]
	} else {
		fn = func(ctx context.Context, obj attrs.Definer) (pk any, contentType string, err error) {
			objKey := attrs.PrimaryKey(obj)
			cTypeDef := contenttypes.DefinitionForObject(obj)
			cType := cTypeDef.ContentType()
			return objKey, cType.TypeName(), nil
		}
	}

	objKey, contentType, err = fn(ctx, obj)
	if err != nil {
		return "", "", err
	}

marshalKey:
	objectID, err := json.Marshal(objKey)
	if err != nil {
		return "", "", err
	}

	return string(objectID), contentType, nil
}

func GetRevisionData(obj attrs.Definer, opts ...MarshallerOption) (map[string]any, error) {
	var o = newMarshallerOptions(opts...)
	if m, ok := obj.(RevisionDataMarshaller); ok && o.AllowDataMarshaller {
		return m.MarshalRevisionData()
	}

	var tree = queries.ProxyFields(obj)
	var seen = make(map[string]struct{})
	var objMap = make(map[string]any)

	if o.SkipProxyFields {
		goto marshalObject
	} else {
		// Add option to skip proxy fields when walking the tree
		// This is because all proxy fields should already be included
		// in the below loop - we dont want to do extra work.
		opts = append(
			opts, MarshallerSkipProxyFields(true),
		)
	}

proxyLoop:
	for obj, err := range tree.WalkObjectProxies(obj, false) {
		if err != nil {
			return nil, errors.Wrapf(
				err, "walking object proxies for %T", obj,
			)
		}

		if len(obj.Path) == 1 {
			seen[obj.Path[0]] = struct{}{}
		}

		var walkMap = objMap
		for i, name := range obj.Path {
			if i == len(obj.Path)-1 {
				var data, err = GetRevisionData(obj.Value, opts...)
				if err != nil {
					return nil, err
				}
				walkMap[name] = data
				continue proxyLoop
			}

			var obj, exists = walkMap[name]
			if !exists {
				obj = make(map[string]any)
			}

			m, isMap := obj.(map[string]any)
			if !isMap {
				return nil, errors.TypeMismatch.Wrapf(
					"expected field %s to be a map[string]any, got %T",
					name, obj,
				)
			}

			walkMap = m
		}
	}

marshalObject:
	// Merge seen and excluded maps - they serve the same purpose
	// from here on.
	var excluded = getExcludedFields(obj)
	maps.Copy(seen, excluded)

	var defs = obj.FieldDefs()
	for _, def := range defs.Fields() {
		var name = def.Name()
		if _, isSeen := seen[name]; isSeen {
			continue
		}

		if !o.AllowReadOnlyFields && !def.AllowEdit() {
			continue
		}

		var rel = def.Rel()
		if rel != nil && slices.Contains([]attrs.RelationType{attrs.RelManyToMany, attrs.RelOneToMany}, rel.Type()) {
			continue
		}

		var value, err = o.getFieldValue(def)
		if err != nil {
			return nil, err
		}

		objMap[name] = value
	}

	return objMap, nil
}

func MarshalRevisionData(obj attrs.Definer, opts ...MarshallerOption) ([]byte, error) {
	var objMap, err = GetRevisionData(obj, opts...)
	if err != nil {
		return nil, err
	}

	return json.Marshal(objMap)
}

func UnmarshalRevisionData(obj attrs.Definer, data []byte, opts ...MarshallerOption) error {
	var o = newMarshallerOptions(opts...)
	if m, ok := obj.(RevisionDataUnMarshaller); ok {
		return m.UnmarshalRevisionData(data)
	}

	var dataMap = make(map[string]json.RawMessage)
	var err = json.Unmarshal(data, &dataMap)
	if err != nil {
		return errors.Wrap(err, "unmarshaling revision data")
	}

	var defs = obj.FieldDefs()
	var fields = defs.Fields()
	var excluded = getExcludedFields(obj)
	for _, def := range fields {
		if _, isExcluded := excluded[def.Name()]; isExcluded {
			continue
		}

		var name = def.Name()
		var value, ok = dataMap[name]
		if !ok {
			continue
		}

		if !o.AllowReadOnlyFields && !def.AllowEdit() {
			continue
		}

		proxyField, ok := def.(queries.ProxyField)
		if !o.SkipProxyFields && ok && proxyField.IsProxy() {
			var newObjFace = def.GetValue()
			var newObj attrs.Definer
			newObj, ok = newObjFace.(attrs.Definer)
			if !ok {
				return errors.TypeMismatch.Wrapf(
					"expected proxy field %q to be of type attrs.Definer, got %T",
					name, newObjFace,
				)
			}

			if attrs.IsZero(newObj) {
				newObj = attrs.NewObject[attrs.Definer](def.Type())
			}

			if err = UnmarshalRevisionData(newObj, value); err != nil {
				return errors.Wrapf(
					err, "unmarshaling proxy field %q", name,
				)
			}

			err = def.SetValue(newObj, true)
			if err != nil {
				return errors.Wrapf(
					err, "setting value for proxy field %q", name,
				)
			}

			continue
		}

		var rel = def.Rel()
		if rel != nil && slices.Contains([]attrs.RelationType{attrs.RelManyToMany, attrs.RelOneToMany}, rel.Type()) {
			continue
		}

		dbTyp, ok := drivers.DBType(def)
		if !ok {
			return errors.TypeMismatch.Wrapf(
				"cannot determine database type for field %q", name,
			)
		}

		var fT = drivers.DBToDefaultGoType(dbTyp)
		if fT.Kind() == reflect.Ptr {
			fT = fT.Elem()
		}

		var rValuePtr = reflect.New(fT)
		var valuePtr = rValuePtr.Interface()
		err = json.Unmarshal(value, valuePtr)
		if err != nil {
			return errors.Wrapf(
				err, "unmarshaling field %q", name,
			)
		}

		if err = def.Scan(rValuePtr.Elem().Interface()); err != nil {
			return errors.Wrapf(
				err, "scanning field %q for %T", name, rValuePtr.Elem().Interface(),
			)
		}
	}

	return nil
}
