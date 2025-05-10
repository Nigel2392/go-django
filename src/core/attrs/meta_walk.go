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

func WalkMetaFields(m Definer, path string) (PathMetaChain, error) {
	//var cacheKey = fmt.Sprintf("%T.%s", m, path)
	//if CACHE_TRAVERSAL_RESULTS {
	//	if result, ok := walkFieldPathsCache[cacheKey]; ok {
	//		return result, nil
	//	}
	//}

	var parts = strings.Split(path, ".")
	var root = make(PathMetaChain, len(parts))
	var current = m
	for i, part := range parts {
		var modelMeta = GetModelMeta(current)
		var defs = modelMeta.Definitions()
		var meta = &pathMeta{
			Object:      current,
			Definitions: defs,
		}

		var f, ok = defs.Field(part)
		if !ok {
			return nil, fmt.Errorf("field %q not found in %T", part, meta.Object)
		}

		var (
			relation, fwd, rev Relation
			ok1, ok2           bool
		)

		if fwd, ok1 = modelMeta.Forward(part); ok1 {
			relation = fwd
		}

		if rev, ok2 = modelMeta.Reverse(part); ok2 {
			relation = rev
		}

		if (!ok1 && !ok2) && i != len(parts)-1 {
			return nil, fmt.Errorf("field %q is not a relation in %T", part, meta.Object)
		}

		meta.idx = i
		meta.root = root
		meta.Field = f
		meta.Relation = relation

		root[i] = meta

		// meta.TableAlias = pathMetaTableAlias(meta)

		if i == len(parts)-1 {
			break
		}

		// This is required to avoid FieldNotFound errors - some objects might cache
		// their field definitions, meaning any dynamic changes to the field will not be reflected
		// in the field definitions. This is a workaround to avoid that issue.
		var newTyp = reflect.TypeOf(relation.Model())
		var newObj = reflect.New(newTyp.Elem())
		current = newObj.Interface().(Definer)
	}

	//if CACHE_TRAVERSAL_RESULTS {
	//	walkFieldPathsCache[cacheKey] = root
	//}

	return root, nil
}
