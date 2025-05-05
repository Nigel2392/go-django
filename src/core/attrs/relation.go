package attrs

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/elliotchance/orderedmap/v2"
)

type ThroughModel struct {
	This   Definer
	Source string
	Target string
}

// Model returns the through model itself.
func (t *ThroughModel) Model() Definer {
	return t.This
}

// SourceField returns the source field for the relation - this is the field in the source model.
func (t *ThroughModel) SourceField() string {
	return t.Source
}

// TargetField returns the target field for the relation - this is the field in the target model, or in the next through model.
func (t *ThroughModel) TargetField() string {
	return t.Target
}

type ModelMeta interface {
	// Model returns the model for this meta
	Model() Definer

	// Forward returns the forward relations for this model
	Forward(relField string) (Relation, bool)

	// ForwardMap returns the forward relations map for this model
	ForwardMap() *orderedmap.OrderedMap[string, Relation]

	// Reverse returns the reverse relations for this model
	Reverse(relField string) (Relation, bool)

	// ReverseMap returns the reverse relations map for this model
	ReverseMap() *orderedmap.OrderedMap[string, Relation]
}

type relationTarget struct {
	model    Definer
	field    Field
	fieldStr string
	prev     RelationTarget
}

func (r *relationTarget) From() RelationTarget {
	return r.prev
}

func (r *relationTarget) Model() Definer {
	return r.model
}

func (r *relationTarget) Field() Field {
	if r.field != nil {
		return r.field
	}

	var defs = r.model.FieldDefs()
	if r.fieldStr != "" {
		var ok bool
		r.field, ok = defs.Field(r.fieldStr)
		if !ok {
			panic(fmt.Errorf("field %q not found in model %T", r.fieldStr, r.model))
		}
	} else {
		r.field = defs.Primary()
	}

	return r.field
}

type relationMeta struct {
	from    RelationTarget
	typ     RelationType
	target  RelationTarget
	through Through
}

func (r *relationMeta) From() RelationTarget {
	return r.from
}

func (r *relationMeta) Type() RelationType {
	return r.typ
}

func (r *relationMeta) Model() Definer {
	return r.target.Model()
}

func (r *relationMeta) Field() Field {
	return r.target.Field()
}

func (r *relationMeta) Through() Through {
	return r.through
}

type modelMeta struct {
	model   Definer
	forward *orderedmap.OrderedMap[string, Relation] // forward orderedmap
	reverse *orderedmap.OrderedMap[string, Relation] // forward orderedmap
}

func (m *modelMeta) Model() Definer {
	return m.model
}

func (m *modelMeta) Forward(relField string) (Relation, bool) {
	if rel, ok := m.forward.Get(relField); ok {
		return rel, true
	}
	return nil, false
}

func (m *modelMeta) ForwardMap() *orderedmap.OrderedMap[string, Relation] {
	return m.forward
}

func (m *modelMeta) ReverseMap() *orderedmap.OrderedMap[string, Relation] {
	return m.reverse
}

func (m *modelMeta) Reverse(relField string) (Relation, bool) {
	if rel, ok := m.reverse.Get(relField); ok {
		return rel, true
	}
	return nil, false
}

var modelReg = make(map[reflect.Type]*modelMeta)

func newReverseAlias(rev Relation) string {
	var name string
	var model = rev.Model()
	switch rev.Type() {
	case RelManyToOne, RelOneToOne:
		name = fmt.Sprintf("%T", model)
	case RelOneToMany, RelManyToMany:
		name = fmt.Sprintf("%TSet", model)
	default:
		panic(fmt.Errorf("unknown relation type %d", rev.Type()))
	}
	var parts = strings.Split(name, ".")
	if len(parts) > 1 {
		name = parts[len(parts)-1]
	}
	return name
}

func getRelatedName(f Field, default_ string) string {
	if f == nil {
		return default_
	}

	var alias string
	if reverseName, ok := f.(CanRelatedName); ok {
		alias = reverseName.RelatedName()
	}

	if alias == "" {
		var atts = f.Attrs()
		var s, ok = atts[AttrReverseAliasKey]
		if ok {
			alias = s.(string)
		}
	}

	if alias != "" {
		return alias
	}

	return default_
}

func registerReverseRelation(
	fromModel Definer,
	fromField Field,
	forward Relation,
) {
	//// Step 1: Get final target in the chain (the destination model)
	//var last = forward
	//for last.To() != nil {
	//	last = last.To()
	//}

	var targetModel = forward.Model()
	var targetType = reflect.TypeOf(targetModel)
	targetModel = reflect.New(targetType.Elem()).Interface().(Definer)

	// Step 2: Get or init ModelMeta for the target
	meta, ok := modelReg[targetType]
	if !ok {
		RegisterModel(targetModel)
		meta = modelReg[targetType]
	}

	var reversed = ReverseRelation(
		fromModel,
		fromField,
		forward,
	)

	// Step 4: Determine a reverse name
	// Prefer something explicit if available (you could add support for a "related_name" tag in field config)
	var reverseAlias = getRelatedName(fromField, "")
	if reverseAlias == "" {
		reverseAlias = newReverseAlias(reversed)
	}

	// Step 5: Store in reverseRelations
	if _, ok := meta.reverse.Get(reverseAlias); ok {
		panic(fmt.Errorf(
			"reverse relation %q from %T on %T was already registered, please use a different related name",
			reverseAlias, fromModel, targetModel,
		))
	}

	meta.reverse.Set(reverseAlias, reversed)

	modelReg[targetType] = meta
}

func RegisterModel(model Definer) {
	var t = reflect.TypeOf(model)
	if _, ok := modelReg[t]; ok {
		//var stackFrame [10]uintptr
		//n := runtime.Callers(2, stackFrame[:])
		//frames := runtime.CallersFrames(stackFrame[:n])
		//
		//frame, _ := frames.Next()
		//
		//logger.Warnf(
		//	"model %T already registered, skipping registration (called from %s:%d)",
		//	model, frame.File, frame.Line,
		//)
		return
	}

	var meta = &modelMeta{
		model:   reflect.New(t.Elem()).Interface().(Definer),
		forward: orderedmap.NewOrderedMap[string, Relation](),
		reverse: orderedmap.NewOrderedMap[string, Relation](),
	}

	// set the model in the registry early - reverse relations may need it
	// if the model is self-referential (e.g. a tree structure)
	modelReg[t] = meta

	var defs = reflect.New(t.Elem()).Interface().(Definer).FieldDefs()
	if defs == nil {
		panic(fmt.Errorf("model %T has no field definitions", model))
	}

	var fields = defs.Fields()
	for _, field := range fields {
		var name = field.Name()
		if name == "" {
			panic(fmt.Errorf("field %T has no name", field))
		}

		var rel = field.Rel()
		if rel == nil {
			continue
		}

		meta.forward.Set(
			name, rel,
		)

		registerReverseRelation(
			model, field, rel,
		)
	}
}

func GetModelMeta(model Definer) ModelMeta {
	if meta, ok := modelReg[reflect.TypeOf(model)]; ok {
		return meta
	}
	panic(fmt.Errorf("model %T not registered with `queries.RegisterModel`", model))
}

func GetRelationMeta(m Definer, name string) (Relation, bool) {
	var (
		meta ModelMeta
		ok   bool
	)
	if meta, ok = modelReg[reflect.TypeOf(m)]; !ok {
		return nil, false
	}
	if rel, ok := meta.Forward(name); ok {
		return rel, true
	}
	if rel, ok := meta.Reverse(name); ok {
		return rel, true
	}
	return nil, false
}

type typedRelation struct {
	Relation
	from RelationTarget
	typ  RelationType
}

func (r *typedRelation) Type() RelationType {
	return r.typ
}

func (r *typedRelation) From() RelationTarget {
	if r.from != nil {
		return r.from
	}
	return r.Relation.From()
}

// Relate creates a new relation between two models.
//
// This can be used to define all kinds of relations between models,
// such as one to one, one to many, many to many, many to one.
//
// The target model is the model that is being related to.
// The target field is the field in the target model that is being related to, it can be an empty string,
// in which case the primary field of the target model is used.
//
// The through model is the model that is used to link the two models together, it can be nil if not needed.
func Relate(target Definer, targetField string, through Through) Relation {
	var rel = &relationMeta{
		target:  &relationTarget{model: target, field: nil, fieldStr: targetField},
		through: through,
	}
	return rel
}

func ReverseRelation(
	fromModel Definer,
	fromField Field,
	forward Relation,
) Relation {
	var targetModel = forward.Model()
	// Step 3: Build reversed chain
	var relTyp RelationType
	switch forward.Type() {
	case RelOneToOne:
		relTyp = RelOneToOne
	case RelOneToMany:
		relTyp = RelManyToOne
	case RelManyToOne:
		relTyp = RelOneToMany
	case RelManyToMany:
		relTyp = RelManyToMany
	}

	var reversed = &relationMeta{
		typ:    relTyp,
		from:   &relationTarget{model: targetModel, field: forward.Field()},
		target: &relationTarget{model: fromModel, field: fromField},
	}

	var through = forward.Through()
	if through != nil {
		reversed.through = &ThroughModel{
			This:   through.Model(),
			Source: through.TargetField(),
			Target: through.SourceField(),
		}
	}

	return &typedRelation{
		Relation: reversed,
		from:     reversed.from,
		typ:      relTyp,
	}
}
