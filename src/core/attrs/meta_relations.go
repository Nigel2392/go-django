package attrs

import (
	"fmt"

	"github.com/Nigel2392/go-django/src/core/contenttypes"
)

var (
	_ Relation = &deferredRelation{}
	_ Relation = &typedRelation{}
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

type relationTarget struct {
	model    Definer
	defs     Definitions
	field    FieldDefinition
	fieldStr string
	prev     RelationTarget
}

func (r *relationTarget) From() RelationTarget {
	return r.prev
}

func (r *relationTarget) Model() Definer {
	return r.model
}

func (r *relationTarget) Field() FieldDefinition {
	if r.field != nil {
		return r.field
	}

	if r.defs == nil {
		r.defs = GetModelMeta(r.model).Model().FieldDefs()
	}

	if r.fieldStr != "" {
		var ok bool
		r.field, ok = r.defs.Field(r.fieldStr)
		if !ok {
			panic(fmt.Errorf("field %q not found in model %T", r.fieldStr, r.model))
		}
	} else {
		r.field = r.defs.Primary()
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

func (r *relationMeta) Field() FieldDefinition {
	return r.target.Field()
}

func (r *relationMeta) Through() Through {
	return r.through
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

var deferredMap = make(map[string]*deferredRelation)

func deferredMapKey(d *deferredRelation) string {

	var (
		typ         = d.typ
		modelType   = d.model_type
		targetField = d.target_field
	)

	return fmt.Sprintf(
		"%d:%s:%s",
		typ, modelType, targetField,
	)
}

func ThroughDeferred(
	contentType string,
	sourceField string,
	targetField string,
) Through {
	return &deferredThroughModel{
		contentType: contentType,
		sourceField: sourceField,
		targetField: targetField,
	}
}

func RelatedDeferred(
	typ RelationType,
	modelContentType string,
	targetField string,
	through Through,
) Relation {
	return &deferredRelation{
		typ:          typ,
		model_type:   modelContentType,
		target_field: targetField,
		through:      through,
	}
}

type deferredThroughModel struct {
	contentType string
	sourceField string
	targetField string

	mdl Definer
}

func (d *deferredThroughModel) setup() {
	if d.mdl != nil {
		return // already set up
	}

	var cType = contenttypes.DefinitionForType(d.contentType)
	if cType == nil {
		panic(fmt.Errorf("content type %q not found for deferring through model", d.contentType))
	}

	d.mdl = cType.Object().(Definer)
}

func (d *deferredThroughModel) Model() Definer {
	if d.mdl != nil {
		return d.mdl
	}

	d.setup()
	return d.mdl
}

func (d *deferredThroughModel) SourceField() string {
	return d.sourceField
}

func (d *deferredThroughModel) TargetField() string {
	return d.targetField
}

// deferredRelation is a relation that is deferred until it is needed.
//
// this exists to avoid unescessary work when the relation is not used.
//
// it is currently only used when registering with [AutoDefinitions],
// as the contenttype might not be immediately available.
type deferredRelation struct {
	typ          RelationType
	model_type   string
	target_field string
	mdl          Definer
	field        Field
	through      Through
}

func (d *deferredRelation) setup() {

	//	var k = deferredMapKey(d)
	//	if existing, ok := deferredMap[k]; ok {
	//		d.mdl = existing.mdl
	//		d.field = existing.field
	//		d.through = existing.through
	//		return
	//	}

	var cType = contenttypes.DefinitionForType(d.model_type)
	if cType == nil {
		panic(fmt.Errorf("content type %q not found for deferring relation", d.model_type))
	}

	d.mdl = cType.Object().(Definer)
	if d.target_field != "" {
		var defs = d.mdl.FieldDefs()
		var f, ok = defs.Field(d.target_field)
		if !ok {
			panic(fmt.Errorf("field %q not found in model %T", d.target_field, d.mdl))
		}
		d.field = f
	}

	// deferredMap[k] = d
}

func (d *deferredRelation) Type() RelationType {
	return d.typ
}

func (d *deferredRelation) From() RelationTarget {
	return nil
}

func (d *deferredRelation) Model() Definer {
	if d.mdl != nil {
		return d.mdl
	}

	d.setup()
	return d.mdl
}

func (d *deferredRelation) Field() FieldDefinition {
	if d.field != nil {
		return d.field
	}

	if d.target_field == "" {
		return nil
	}

	d.setup()
	return d.field
}

func (d *deferredRelation) Through() Through {
	if d.through != nil {
		return d.through
	}

	d.setup()
	return d.through
}
