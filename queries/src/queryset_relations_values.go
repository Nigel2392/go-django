package queries

import (
	"fmt"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/elliotchance/orderedmap/v2"
)

var (
	_ canUniqueKey              = (*baseRelation)(nil)
	_ Relation                  = (*baseRelation)(nil)
	_ Relation                  = (*RelO2O[attrs.Definer, attrs.Definer])(nil)
	_ RelationValue             = (*RelFK[attrs.Definer])(nil)
	_ MultiRelationValue        = (*RelRevFK[attrs.Definer])(nil)
	_ ThroughRelationValue      = (*RelO2O[attrs.Definer, attrs.Definer])(nil)
	_ MultiThroughRelationValue = (*RelM2M[attrs.Definer, attrs.Definer])(nil)
	_ attrs.Binder              = (*RelFK[attrs.Definer])(nil)
	_ attrs.Binder              = (*RelRevFK[attrs.Definer])(nil)
	_ attrs.Binder              = (*RelO2O[attrs.Definer, attrs.Definer])(nil)
	_ attrs.Binder              = (*RelM2M[attrs.Definer, attrs.Definer])(nil)
)

// A base relation type that implements the Relation interface.
//
// It is used to set the related object and it's through object on a model.
type baseRelation struct {
	uniqueValue any
	object      attrs.Definer
	through     attrs.Definer
}

func (r *baseRelation) UniqueKey() any {
	return r.uniqueValue
}

func (r *baseRelation) Model() attrs.Definer {
	return r.object
}

func (r *baseRelation) Through() attrs.Definer {
	return r.through
}

type RelFK[ModelType attrs.Definer] struct {
	Parent *ParentInfo // The parent model instance
	Object ModelType
}

func (rl *RelFK[T]) ParentInfo() *ParentInfo {
	return rl.Parent
}

func (rl *RelFK[T]) BindToModel(parent attrs.Definer, parentField attrs.Field) error {
	rl.Parent = &ParentInfo{
		Object: parent,
		Field:  parentField,
	}
	return nil
}

func (rl *RelFK[T]) Model() attrs.Definer {
	if rl == nil {
		return nil
	}
	return rl.Object
}

// SetValue sets the related object on the relation.
func (rl *RelFK[T]) SetValue(instance attrs.Definer) {
	rl.Object = instance.(T)
}

// GetValue returns the related object on the relation.
func (rl *RelFK[T]) GetValue() attrs.Definer {
	if rl == nil {
		return nil
	}
	return rl.Object
}

type RelRevFK[ModelType attrs.Definer] struct {
	Parent          *ParentInfo                            // The parent model instance
	relations       *orderedmap.OrderedMap[any, ModelType] // The related objects
	relatedQuerySet *RelManyToOneQuerySet[ModelType]       // The query set for this relation
}

func (rl *RelRevFK[T]) ParentInfo() *ParentInfo {
	return rl.Parent
}

func (rl *RelRevFK[T]) BindToModel(parent attrs.Definer, parentField attrs.Field) error {
	rl.Parent = &ParentInfo{
		Object: parent,
		Field:  parentField,
	}
	if rl.relations == nil {
		rl.relations = orderedmap.NewOrderedMap[any, T]()
	}
	return nil
}

func (rl *RelRevFK[T]) Objects() *RelManyToOneQuerySet[T] {
	if rl.relatedQuerySet == nil {
		rl.relatedQuerySet = ManyToOneQuerySet[T](rl)
	}
	return rl.relatedQuerySet
}

func (rl *RelRevFK[T]) Cache() *orderedmap.OrderedMap[any, T] {
	if rl.relations == nil {
		rl.relations = orderedmap.NewOrderedMap[any, T]()
	}
	return rl.relations
}

// SetValues sets the related objects on the relation.
func (rl *RelRevFK[T]) SetValues(objects []attrs.Definer) {
	if rl.relations == nil {
		rl.relations = orderedmap.NewOrderedMap[any, T]()
	}

	for _, obj := range objects {
		if obj == nil {
			continue
		}

		var uniqueKey any
		if canPk, ok := obj.(canUniqueKey); ok {
			uniqueKey = canPk.UniqueKey()
		}

		if uniqueKey == nil {
			var err error
			uniqueKey, err = GetUniqueKey(obj)

			if err != nil {
				panic(fmt.Errorf("cannot set related object %T without a generated unique key: %w", obj, err))
			}
		}

		rl.relations.Set(uniqueKey, obj.(T))
	}
}

// GetValues returns the related objects on the relation.
func (rl *RelRevFK[T]) GetValues() []attrs.Definer {
	if rl == nil || rl.relations == nil {
		return nil
	}

	var relatedObjects = make([]attrs.Definer, 0, rl.relations.Len())
	for relHead := rl.relations.Front(); relHead != nil; relHead = relHead.Next() {
		relatedObjects = append(relatedObjects, relHead.Value)
	}
	return relatedObjects
}

// Objects returns the related objects as a slice of ModelType.
func (rl *RelRevFK[T]) AsList() []T {
	if rl == nil || rl.relations == nil {
		return nil
	}
	var relatedObjects = make([]T, 0, rl.relations.Len())
	for relHead := rl.relations.Front(); relHead != nil; relHead = relHead.Next() {
		relatedObjects = append(relatedObjects, relHead.Value)
	}
	return relatedObjects
}

// A value which can be used on models to represent a One-to-One relation
// with a through model.
//
// This implements the [SettableThroughRelation] interface, which allows setting
// the related object and its through object.
type RelO2O[ModelType, ThroughModelType attrs.Definer] struct {
	Parent        *ParentInfo // The parent model instance
	Object        ModelType
	ThroughObject ThroughModelType
}

func (rl *RelO2O[T1, T2]) ParentInfo() *ParentInfo {
	return rl.Parent
}

func (rl *RelO2O[T1, T2]) BindToModel(parent attrs.Definer, parentField attrs.Field) error {
	rl.Parent = &ParentInfo{
		Object: parent,
		Field:  parentField,
	}
	return nil
}

func (rl *RelO2O[T1, T2]) Model() attrs.Definer {
	if rl == nil {
		return nil
	}
	return rl.Object
}

func (rl *RelO2O[T1, T2]) Through() attrs.Definer {
	if rl == nil {
		return nil
	}
	return rl.ThroughObject
}

func (rl *RelO2O[T1, T2]) SetValue(instance attrs.Definer, through attrs.Definer) {
	if instance != nil {
		rl.Object = instance.(T1)
	}
	if through != nil {
		rl.ThroughObject = through.(T2)
	}
}

func (rl *RelO2O[T1, T2]) GetValue() (obj attrs.Definer, through attrs.Definer) {
	if rl == nil {
		return nil, nil
	}
	return rl.Object, rl.ThroughObject
}

// A value which can be used on models to represent a Many-to-Many relation
// with a through model.
//
// This implements the [SettableMultiThroughRelation] interface, which allows setting
// the related objects and their through objects.
type RelM2M[ModelType, ThroughModelType attrs.Definer] struct {
	Parent          *ParentInfo                                                      // The parent model instance
	relations       *orderedmap.OrderedMap[any, RelO2O[ModelType, ThroughModelType]] // can be changed to slice if needed
	relatedQuerySet *RelManyToManyQuerySet[ModelType]                                // The query set for this relation

	// relations []RelO2O[T1, T2] // can be changed to OrderedMap if needed
}

func (rl *RelM2M[T1, T2]) ParentInfo() *ParentInfo {
	return rl.Parent
}

func (rl *RelM2M[T1, T2]) BindToModel(parent attrs.Definer, parentField attrs.Field) error {
	rl.Parent = &ParentInfo{
		Object: parent,
		Field:  parentField,
	}
	return nil
}

func (rl *RelM2M[T1, T2]) Cache() *orderedmap.OrderedMap[any, RelO2O[T1, T2]] {
	if rl.relations == nil {
		rl.relations = orderedmap.NewOrderedMap[any, RelO2O[T1, T2]]()
	}
	return rl.relations
}

func (rl *RelM2M[T1, T2]) Objects() *RelManyToManyQuerySet[T1] {
	if rl.relatedQuerySet == nil {
		rl.relatedQuerySet = ManyToManyQuerySet[T1](rl)
	}
	return rl.relatedQuerySet
}

func (rl *RelM2M[T1, T2]) SetValues(rel []Relation) {
	if rl == nil {
		panic("cannot set values on nil RelM2M")
	}

	if len(rel) == 0 {
		rl.relations = orderedmap.NewOrderedMap[any, RelO2O[T1, T2]]()
		// rl.relations = make([]RelO2O[T1, T2], 0)
		return
	}

	rl.relations = orderedmap.NewOrderedMap[any, RelO2O[T1, T2]]()
	// rl.relations = make([]RelO2O[T1, T2], 0, len(rel))
	for _, r := range rel {
		if r == nil {
			continue
		}

		var o2o = RelO2O[T1, T2]{
			Parent:        rl.Parent,
			Object:        r.Model().(T1),
			ThroughObject: r.Through().(T2),
		}

		// rl.relations = append(rl.relations, o2o)

		var uniqueKey any
		if canPk, ok := r.(canUniqueKey); ok {
			uniqueKey = canPk.UniqueKey()
		}

		// First nil check we can get the primary key
		// from the relation's definitions.
		if uniqueKey == nil {
			var err error
			uniqueKey, err = GetUniqueKey(r.Model())
			if err != nil {
				panic(fmt.Errorf("cannot set related object %T without a generated unique key: %w", r.Model(), err))
			}
		}

		rl.relations.Set(uniqueKey, o2o)
	}
}

// GetValues returns the related objects and their through objects.
func (rl *RelM2M[T1, T2]) GetValues() []Relation {
	if rl == nil || rl.relations == nil {
		return nil
	}
	// var relatedObjects = make([]Relation, len(rl.relations))
	// for i, rel := range rl.relations {
	// relatedObjects[i] = &rel
	// }
	// return relatedObjects
	var relatedObjects = make([]Relation, 0, rl.relations.Len())
	for relHead := rl.relations.Front(); relHead != nil; relHead = relHead.Next() {
		relatedObjects = append(relatedObjects, &relHead.Value)
	}
	return relatedObjects
}

func (rl *RelM2M[T1, T2]) AsList() []RelO2O[T1, T2] {
	if rl == nil || rl.relations == nil {
		return nil
	}

	var relatedObjects = make([]RelO2O[T1, T2], 0, rl.relations.Len())
	for relHead := rl.relations.Front(); relHead != nil; relHead = relHead.Next() {
		relatedObjects = append(relatedObjects, relHead.Value)
	}
	return relatedObjects
}

func (rl *RelM2M[T1, T2]) Len() int {
	if rl == nil || rl.relations == nil {
		return 0
	}
	// return len(rl.relations)
	return rl.relations.Len()
}
