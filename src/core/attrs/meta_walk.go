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

type WalkFieldsFunc func(source Relation, meta ModelMeta, object Definer, field FieldDefinition, fieldRel Relation, part string, parts []string, idx int) (stop bool, err error)

func WalkMetaFieldsFunc(m Definer, path []string, fn WalkFieldsFunc) error {
	var (
		parentRel Relation
		current   = m
	)
	for i, part := range path {
		var (
			modelMeta = GetModelMeta(current)
			defs      = modelMeta.Definitions()
			f, ok     = defs.Field(part)
		)
		if !ok {
			return fmt.Errorf("field %q not found in %T (%v)", part, current, FieldNames(defs.Fields(), nil))
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

		if stop, err := fn(parentRel, modelMeta, current, f, relation, part, path, i); stop || err != nil {
			return err
		}

		if i == len(path)-1 {
			break
		}

		var newTyp = reflect.TypeOf(relation.Model())
		var newObj = reflect.New(newTyp.Elem())
		current = newObj.Interface().(Definer)
		parentRel = relation
	}

	return nil
}

func WalkMetaFields(m Definer, path []string) (PathMetaChain, error) {
	var root = make(PathMetaChain, len(path))
	var walk = func(source Relation, meta ModelMeta, object Definer, field FieldDefinition, relation Relation, part string, parts []string, idx int) (bool, error) {
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

	if err := WalkMetaFieldsFunc(m, path, walk); err != nil {
		return nil, err
	}

	return root, nil
}

type RelationChainPart struct {
	chain     *RelationChain     // the chain this part belongs to
	Next      *RelationChainPart // the next part in the chain
	Prev      *RelationChainPart // the previous part in the chain
	ChainPart string             // the name of the field in the chain
	FieldRel  Relation           // the relation of the field in the current part
	Model     Definer            // the current target model
	Through   Through            // the through relation to get to the target model, if any
	Field     FieldDefinition    // the field in the current target model
	Depth     int                // corresponds to the index in chain.Chain
}

type RelationChain struct {
	Root   *RelationChainPart
	Final  *RelationChainPart
	Fields []FieldDefinition
	Chain  []string
}

func WalkRelationChain(m Definer, includeFinalRel bool, path []string) (*RelationChain, error) {
	var last *RelationChainPart
	var chain = &RelationChain{
		Chain: make([]string, 0, len(path)),
	}
	var walk = func(sourceRel Relation, meta ModelMeta, object Definer, field FieldDefinition, fieldRel Relation, part string, parts []string, idx int) (bool, error) {
		var node = &RelationChainPart{
			chain: chain,

			Next: nil,
			Prev: chain.Final, // the previous part in the chain

			Depth:     idx,
			ChainPart: part,
			FieldRel:  fieldRel,
			Model:     object,
			Field:     field,
		}

		if last == nil {
			chain.Root = node
		} else {
			last.Next = node
		}
		last = node
		chain.Final = node

		if idx == len(parts)-1 {
			// always set the final part’s Source to fieldRel
			if fieldRel != nil && includeFinalRel {
				// include the relation in the chain
				chain.Chain = append(chain.Chain, part)

				// rebuild p to carry through-model info
				*node = RelationChainPart{
					chain:     chain,
					Next:      nil,
					Depth:     idx,
					Prev:      chain.Final, // the previous part in the chain
					ChainPart: field.Name(),

					FieldRel: fieldRel,
					Through:  fieldRel.Through(),
					Model:    fieldRel.Model(),
					Field:    fieldRel.Field(),
				}

				// all the target’s fields
				chain.Fields = GetModelMeta(fieldRel.Model()).Definitions().Fields()
			} else {
				// just expose the last field itself
				chain.Fields = []FieldDefinition{field}
			}

			return true, nil
		}

		chain.Chain = append(chain.Chain, part)

		return false, nil
	}

	if err := WalkMetaFieldsFunc(m, path, walk); err != nil {
		return nil, err
	}

	return chain, nil
}
