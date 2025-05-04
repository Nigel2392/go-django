package attrs

import "github.com/Nigel2392/go-django/src/core/assert"

var (
	_ Through  = &ThroughModel{}
	_ Relation = &relation{}
)

type ThroughModel struct {
	This   Definer
	Source string
	Target string

	defs        Definitions
	sourceField Field
	targetField Field
}

// Model returns the through model itself.
func (t *ThroughModel) Model() Definer {
	return t.This
}

func (t *ThroughModel) define() {
	if t.defs == nil {
		t.defs = t.This.FieldDefs()
	}
}

// SourceField returns the source field for the relation - this is the field in the source model.
func (t *ThroughModel) SourceField() Field {
	if t.sourceField == nil {
		t.define()
		var ok bool
		t.sourceField, ok = t.defs.Field(t.Source)
		if !ok {
			assert.Fail("Source field %q not found in model %T", t.Source, t.This)
		}
	}
	return t.sourceField
}

// TargetField returns the target field for the relation - this is the field in the target model, or in the next through model.
func (t *ThroughModel) TargetField() Field {
	if t.targetField == nil {
		t.define()
		var ok bool
		t.targetField, ok = t.defs.Field(t.Target)
		if !ok {
			assert.Fail("Target field %q not found in model %T", t.Target, t.This)
		}
	}
	return t.targetField
}

type relation struct {
	target         Definer
	targetDefs     Definitions
	targetFieldStr string
	targetField    Field
	through        Through
}

func (t *relation) define() {
	if t.targetDefs == nil {
		t.targetDefs = t.target.FieldDefs()
	}
}

// Target returns the target model for the relationship.
func (r *relation) Target() Definer {
	return r.target
}

// TargetField retrieves the field in the target model for the relationship.
//
// This can be nil, in such cases the relationship should use the primary field of the target model.
//
// If a through model is used, the target field should still target the actual target model,
// the through model should then use this field to link to the target model.
func (r *relation) TargetField() Field {
	if r.targetField == nil {
		r.define()
		var ok bool
		if r.targetFieldStr == "" {
			r.targetField = r.targetDefs.Primary()
			ok = true
		} else {
			r.targetField, ok = r.targetDefs.Field(r.targetFieldStr)
		}
		if !ok {
			assert.Fail("Target field %q not found in model %T", r.targetFieldStr, r.target)
		}
	}
	return r.targetField
}

// Through returns the through model for the relationship.
//
// This can be nil, but does not have to be.
//
// It can support a one to one relationship with or without a through model,
// or a many to many relationship with a through model.
//
func (r *relation) Through() Through {
	return r.through
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
	var rel = &relation{
		target:         target,
		targetFieldStr: targetField,
		through:        through,
	}

	return rel
}

// Reverse creates a new reverse relation between two models.
//
// It does this by taking the relation and reversing the source and target models,
// and optionally any through models.
//
// The sourceObj is the model that is being related from in the TypedRelation.
// The sourceFieldName is the field in the forward model that is being related from in the TypedRelation.
func ReverseRelation(sourceObj Definer, sourceFieldName string, rel TypedRelation) TypedRelation {
	var r = &relation{
		target:         sourceObj,
		targetFieldStr: sourceFieldName,
	}

	var through = rel.Through()
	if through != nil {
		r.through = &ThroughModel{
			This:        through.Model(),
			sourceField: through.TargetField(),
			targetField: through.SourceField(),
		}
	}

	var relTyp RelationType
	switch rel.Type() {
	case RelOneToOne:
		relTyp = RelOneToOne
	case RelOneToMany:
		relTyp = RelManyToOne
	case RelManyToOne:
		relTyp = RelOneToMany
	case RelManyToMany:
		relTyp = RelManyToMany
	default:
		assert.Fail("Unknown relation type %q", rel.Type())
	}

	return &typedRelation{
		typ:      relTyp,
		Relation: r,
	}
}

type typedRelation struct {
	Relation
	typ RelationType
}

func (r *typedRelation) Type() RelationType {
	return r.typ
}
