package attrs

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/elliotchance/orderedmap/v2"
)

type modelMeta struct {
	setup       bool
	model       Definer
	definitions StaticDefinitions
	forward     *orderedmap.OrderedMap[string, Relation] // forward orderedmap
	reverse     *orderedmap.OrderedMap[string, Relation] // forward orderedmap
	stored      *orderedmap.OrderedMap[string, any]      // stored (possible configuration) values
}

func (m *modelMeta) Model() Definer {
	return m.model
}

func (m *modelMeta) Definitions() StaticDefinitions {
	if m.definitions == nil {
		m.definitions = newStaticDefinitions(m.model)
	}
	return m.definitions
}

func (m *modelMeta) Forward(relField string) (Relation, bool) {
	if rel, ok := m.forward.Get(relField); ok {
		return rel, true
	}
	return nil, false
}

func (m *modelMeta) Reverse(relField string) (Relation, bool) {
	if rel, ok := m.reverse.Get(relField); ok {
		return rel, true
	}
	return nil, false
}

func (m *modelMeta) ContentType() contenttypes.ContentType {
	return contenttypes.NewContentType[any](m.model)
}

func (m *modelMeta) ForwardMap() *orderedmap.OrderedMap[string, Relation] {
	return m.forward.Copy()
}

func (m *modelMeta) ReverseMap() *orderedmap.OrderedMap[string, Relation] {
	return m.reverse.Copy()
}

func (m *modelMeta) Storage(key string) (any, bool) {
	return m.stored.Get(key)
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

	// Step 1: Get the target model and type
	// Create a new instance of the target target model
	var targetModel = forward.Model()
	var targetType = reflect.TypeOf(targetModel)
	targetModel = NewObject[Definer](targetType.Elem())

	// Step 2: Get or init ModelMeta for the target
	meta, ok := modelReg[targetType]
	if !ok {
		RegisterModel(targetModel)
		meta = modelReg[targetType]
	}

	// Step 3: Build reversed chain
	// This is the relation that will be used to access the source model from the target model
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

	var storageKey = fmt.Sprintf(
		"relation.%T.%s.%s",
		fromModel,
		fromField.Name(),
		reverseAlias,
	)

	if _, ok := meta.stored.Get(storageKey); ok {
		// Cannot register the same reverse relation twice
		// No need to panic here - since the relation was already registered
		// we can just skip it
		return
	}

	// Step 5: Store in reverseRelations
	if _, ok := meta.reverse.Get(reverseAlias); ok {
		// Cannot register a reverse relation with the same name twice
		// This is a programming error and can happen if you have two reverse relations
		// from two different models to the same model with the same name
		//
		// e.g. if you have two models A and B, and both have a reverse relation to C with the same name
		panic(fmt.Errorf(
			"reverse relation %q from %T on %T was already registered, please use a different related name",
			reverseAlias, fromModel, targetModel,
		))
	}

	meta.reverse.Set(reverseAlias, reversed)
	meta.stored.Set(storageKey, nil)

	modelReg[targetType] = meta
}

// RegisterModel registers a model to be used for any ORM- type operations.
//
// Models are registered automatically in [django.Initialize], but you can also register them manually if needed.
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

	// set the model in the registry early - reverse relations may need it
	// if the model is self-referential (e.g. a tree structure)
	var meta = &modelMeta{
		model:   NewObject[Definer](model),
		forward: orderedmap.NewOrderedMap[string, Relation](),
		reverse: orderedmap.NewOrderedMap[string, Relation](),
		stored:  orderedmap.NewOrderedMap[string, any](),
	}
	modelReg[t] = meta

	var defs = meta.model.FieldDefs()
	if defs == nil {
		panic(fmt.Errorf("error getting model definitions: model %T has no field definitions", model))
	}

	// Send signal that the model is being registered
	var staticDefs = wrapDefinitions(meta.model, defs)
	OnBeforeModelRegister.Send(SignalModelMeta{
		Definer:     meta.model,
		Definitions: staticDefs,
		Meta:        meta,
	})

	if mInfo, ok := meta.model.(CanModelInfo); ok {
		// If the model has a meta, we need to set it
		// This is used for things like unique_together, ordering, etc.
		var modelMeta = mInfo.ModelMetaInfo(meta.model)
		for k, v := range modelMeta {
			meta.stored.Set(k, v)
		}
	}

	var fields = defs.Fields()
	for _, field := range fields {
		var name = field.Name()
		if name == "" {
			panic(fmt.Errorf("error creating meta: field %T has no name", field))
		}

		// fields can get a callback when the model they are defined on is registered
		if registrar, ok := field.(CanOnModelRegister); ok {
			var err = registrar.OnModelRegister(meta.model)
			if err != nil {
				panic(fmt.Errorf(
					"field.OnModelRegister: error registering field %q on model %T: %w",
					name, meta.model, err,
				))
			}
		}

		var rel = field.Rel()
		if rel == nil {
			continue
		}

		if rel.Through() != nil {
			// If the relation has a through model, we need to register it
			// This is used for many-to-many relations
			var through = rel.Through()
			if through.Model() == nil {
				panic(fmt.Errorf("error creating meta: relation %q on %T has no through model", name, model))
			}

			var throughMeta, ok = modelReg[reflect.TypeOf(through.Model())]
			if !ok {
				// If the through model is not registered, we need to register it
				RegisterModel(through.Model())
				throughMeta = modelReg[reflect.TypeOf(through.Model())]
			}

			var storageKey = fmt.Sprintf(
				"through.%s",
				meta.ContentType().TypeName(),
			)
			if _, wasSent := throughMeta.stored.Get(storageKey); wasSent {
				goto setRel
			}

			// Send signal that the through model is being registered
			OnThroughModelRegister.Send(SignalThroughModelMeta{
				Source:      meta.model,
				Target:      rel.Model(),
				ThroughInfo: through,
				Meta:        throughMeta,
			})
		}

	setRel:
		meta.forward.Set(
			name, rel,
		)

		var canReverse, ok = field.(CanReverseRelate)
		if !ok || canReverse.AllowReverseRelation() {
			registerReverseRelation(
				model, field, rel,
			)
		}
	}

	// Set the model as setup
	meta.setup = true

	// Send signal that the model has been registered
	OnModelRegister.Send(SignalModelMeta{
		Definer:     meta.model,
		Definitions: staticDefs,
		Meta:        meta,
	})
}

func GetModelMeta(model any) ModelMeta {
	var (
		meta ModelMeta
		ok   bool
	)
	switch model := model.(type) {
	case Definer:
		meta, ok = modelReg[reflect.TypeOf(model)]
	case reflect.Type:
		if model.Kind() != reflect.Ptr {
			model = reflect.PointerTo(model)
		}
		meta, ok = modelReg[model]
	case reflect.Value:
		var t = model.Type()
		if t.Kind() != reflect.Ptr {
			t = reflect.PointerTo(t)
		}
		meta, ok = modelReg[t]
	default:
		panic(fmt.Errorf("GetModelMeta: expected Definer, reflect.Type or reflect.Value, got %T", model))
	}

	if ok {
		return meta
	}

	panic(fmt.Errorf("model %T not registered with `attrs.RegisterModel`, could not retrieve meta", model))
}

func IsModelRegistered(model Definer) bool {
	var mdl, ok = modelReg[reflect.TypeOf(model)]
	if !ok {
		return false
	}
	return mdl.setup
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

func StoreOnMeta(m Definer, key string, value any) {
	var rType = reflect.TypeOf(m)
	if meta, ok := modelReg[rType]; ok {
		meta.stored.Set(key, value)
	} else {
		panic(fmt.Errorf("model %T not registered with `attrs.RegisterModel`, cannot store value %q", m, key))
	}
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
