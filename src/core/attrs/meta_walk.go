package attrs

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
)

type PathMetaChain []*pathMeta

func (c PathMetaChain) First() *pathMeta {
	if len(c) == 0 {
		return nil
	}
	return c[0]
}

func (c PathMetaChain) Last() *pathMeta {
	if len(c) == 0 {
		return nil
	}
	return c[len(c)-1]
}

type pathMeta struct {
	idx         int
	root        PathMetaChain
	Object      Definer
	Definitions StaticDefinitions
	Field       FieldDefinition
	Relation    Relation
	// TableAlias  string
}

func (m *pathMeta) String() string {
	var sb strings.Builder
	for i, part := range m.root[:m.idx+1] {
		if i > 0 {
			sb.WriteString(".")
		}
		sb.WriteString(part.Field.Name())
	}
	return sb.String()
}

//
// func pathMetaTableAlias(m *pathMeta) string {
// if len(m.root) == 1 {
// return m.Definitions.TableName()
// }
// var c = m.CutAt()
// var s = make([]string, len(c))
// for i, part := range c {
// s[i] = part.Field.Name()
// }
//
// var field FieldDefinition
// if m.idx > 0 {
// field = m.root[m.idx-1].Field
// }
//
// return NewJoinAlias(field, m.Definitions.TableName(), s)
// }

func (m *pathMeta) Parent() *pathMeta {
	if m.idx == 0 {
		return nil
	}
	return m.root[m.idx-1]
}

func (m *pathMeta) Child() *pathMeta {
	if m.idx >= len(m.root)-1 {
		return nil
	}
	return m.root[m.idx+1]
}

func (m *pathMeta) CutAt() []*pathMeta {
	return slices.Clone(m.root)[:m.idx+1]
}

//var walkFieldPathsCache = make(map[string]PathMetaChain)

type WalkFieldsFunc func(meta ModelMeta, object Definer, field FieldDefinition, relation Relation, path string, parts []string, idx int) (stop bool, err error)

func WalkMetaFieldsFunc(m Definer, path []string, fn WalkFieldsFunc) error {
	var current = m
	for i, part := range path {
		var (
			modelMeta = GetModelMeta(current)
			defs      = modelMeta.Definitions()
			f, ok     = defs.Field(part)
		)
		if !ok {
			return fmt.Errorf("field %q not found in %T", part, current)
		}

		var (
			relation, fwd, rev Relation
			ok1, ok2           bool
		)

		if fwd, ok1 = modelMeta.Forward(part); ok1 {
			relation = fwd
		}

		if rev, ok2 = modelMeta.Reverse(part); ok2 {

			if ok1 {
				return fmt.Errorf("field %q is both a forward and reverse relation in %T", part, current)
			}

			relation = rev
		}

		if (!ok1 && !ok2) && i != len(path)-1 {
			return fmt.Errorf("field %q is not a relation in %T, cannot traverse further", part, current)
		}

		if stop, err := fn(modelMeta, current, f, relation, part, path, i); stop || err != nil {
			return err
		}

		if i == len(path)-1 {
			break
		}

		var newTyp = reflect.TypeOf(relation.Model())
		var newObj = reflect.New(newTyp.Elem())
		current = newObj.Interface().(Definer)
	}

	return nil
}

func WalkMetaFields(m Definer, path string) (PathMetaChain, error) {

	var parts = strings.Split(path, ".")
	var root = make(PathMetaChain, len(parts))
	var walk = func(meta ModelMeta, object Definer, field FieldDefinition, relation Relation, path string, parts []string, idx int) (bool, error) {
		var pM = &pathMeta{
			Object:      object,
			Definitions: meta.Definitions(),
			Field:       field,
			Relation:    relation,
			idx:         idx,
			root:        root,
		}

		root[idx] = pM

		if idx == len(parts)-1 {
			return true, nil
		}

		return false, nil
	}

	if err := WalkMetaFieldsFunc(m, parts, walk); err != nil {
		return nil, err
	}

	return root, nil
}
