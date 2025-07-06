package queries

import (
	"fmt"

	"github.com/Nigel2392/go-django/queries/internal"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/forms/fields"
)

type throughProxy struct {
	throughDefinition attrs.Through
	object            attrs.Definer
	defs              attrs.Definitions
	sourceField       attrs.Field
	targetField       attrs.Field
}

func newThroughProxy(throughDefinition attrs.Through) *throughProxy {
	var (
		ok              bool
		sourceFieldStr  = throughDefinition.SourceField()
		targetFieldStr  = throughDefinition.TargetField()
		throughInstance = throughDefinition.Model()
		defs            = throughInstance.FieldDefs()
		proxy           = &throughProxy{
			defs:              defs,
			object:            throughInstance,
			throughDefinition: throughDefinition,
		}
	)

	if proxy.sourceField, ok = defs.Field(sourceFieldStr); !ok {
		panic(fmt.Errorf(
			"source field %s not found in through model %T: %w",
			sourceFieldStr, throughInstance, errors.FieldNotFound,
		))
	}

	if proxy.targetField, ok = defs.Field(targetFieldStr); !ok {
		panic(fmt.Errorf(
			"target field %s not found in through model %T: %w",
			targetFieldStr, throughInstance, errors.FieldNotFound,
		))
	}

	return proxy
}

type relatedQuerySet[T attrs.Definer, T2 any] struct {
	embedder         T2
	source           *ParentInfo
	JoinDefCondition *JoinDefCondition
	rel              attrs.Relation
	originalQs       QuerySet[T]
	qs               *QuerySet[T]
}

// NewrelatedQuerySet creates a new relatedQuerySet for the given model type.
func newRelatedQuerySet[T attrs.Definer, T2 any](embedder T2, rel attrs.Relation, source *ParentInfo) *relatedQuerySet[T, T2] {
	if rel == nil {
		panic("Relation is nil, cannot create relatedQuerySet")
	}

	return &relatedQuerySet[T, T2]{
		embedder: embedder,
		rel:      rel,
		source:   source,
	}
}

func (t *relatedQuerySet[T, T2]) setup() {
	if t.qs != nil {
		return // Already set up
	}

	var (
		condition *JoinDefCondition

		newTargetObj     = internal.NewObjectFromIface(t.rel.Model())
		newTargetObjDefs = newTargetObj.FieldDefs()

		// probably should not use a queryset from the model's getqueryset method? right?
		qs           = Objects(newTargetObj.(T))
		throughModel = t.rel.Through()
	)

	var targetFieldInfo = &FieldInfo[attrs.FieldDefinition]{
		Model: qs.internals.Model.Object,
		Table: Table{
			Name: qs.internals.Model.Table,
		},
		Fields: ForSelectAllFields[attrs.FieldDefinition](
			newTargetObjDefs,
		),
	}

	if throughModel != nil {
		var throughObject = newThroughProxy(throughModel)

		targetFieldInfo.Through = &FieldInfo[attrs.FieldDefinition]{
			Model: throughObject.object,
			Table: Table{
				Name: throughObject.defs.TableName(),
				Alias: fmt.Sprintf(
					"%s_through",
					qs.internals.Model.Table,
				),
			},
			Fields: ForSelectAllFields[attrs.FieldDefinition](throughObject.defs),
		}

		condition = &JoinDefCondition{
			Operator: expr.EQ,
			ConditionA: expr.TableColumn{
				TableOrAlias: targetFieldInfo.Table.Name,
				FieldColumn:  t.source.Field,
			},
			ConditionB: expr.TableColumn{
				TableOrAlias: targetFieldInfo.Through.Table.Alias,
				FieldColumn:  throughObject.targetField,
			},
		}

		condition.Next = &JoinDefCondition{
			Operator: expr.EQ,
			ConditionA: expr.TableColumn{
				TableOrAlias: targetFieldInfo.Through.Table.Alias,
				FieldColumn:  throughObject.sourceField,
			},
			ConditionB: expr.TableColumn{
				TableOrAlias: "",
				FieldColumn:  nil,
				Value: t.source.Object.
					FieldDefs().
					Primary().
					GetValue(),
			},
		}

		var join = JoinDef{
			TypeJoin: TypeJoinInner,
			Table: Table{
				Name: throughObject.defs.TableName(),
				Alias: fmt.Sprintf(
					"%s_through",
					qs.internals.Model.Table,
				),
			},
			JoinDefCondition: condition,
		}

		qs.internals.AddJoin(join)
	} else {
		var targetField = t.rel.Field()
		if targetField == nil {
			targetField = newTargetObjDefs.Primary()
		}

		qs.internals.Where = append(qs.internals.Where, expr.Expr(
			targetField.Name(),
			expr.LOOKUP_EXACT,
			t.source.Object.FieldDefs().Primary().GetValue(),
		))
	}

	qs.internals.Fields = append(
		qs.internals.Fields, targetFieldInfo,
	)

	t.originalQs = *qs.clone()
	t.qs = qs
	t.JoinDefCondition = condition

}

func (t *relatedQuerySet[T, T2]) createTargets(targets []T) ([]T, error) {
	t.setup()
	return t.originalQs.Clone().BulkCreate(targets)
}

func (t *relatedQuerySet[T, T2]) createThroughObjects(targets []T) (rels []Relation, created int64, _ error) {
	if t.rel == nil {
		panic("Relation is nil, cannot create through object")
	}

	t.setup()
	var throughModel = t.rel.Through()
	if throughModel == nil {
		panic("Through model is nil, cannot create through object")
	}

	var (
		targetsToSave   = make([]T, 0, len(targets))
		existingTargets = make([]T, 0, len(targets))
	)
	for _, target := range targets {
		var (
			defs    = target.FieldDefs()
			primary = defs.Primary()
			pkValue = primary.GetValue()
		)

		if fields.IsZero(pkValue) {
			targetsToSave = append(targetsToSave, target)
		} else {
			existingTargets = append(existingTargets, target)
		}
	}

	if len(targetsToSave) > 0 {
		var err error
		targetsToSave, err = t.createTargets(targetsToSave)
		if err != nil {
			return nil, 0, errors.SaveFailed.WithCause(fmt.Errorf(
				"failed to create targets %T: %w", targetsToSave, err,
			))
		}
		created = int64(len(targetsToSave))
	}

	targets = append(existingTargets, targetsToSave...)

	// Create a new instance of the through model
	var (
		throughObj            = throughModel.Model()
		throughSourceFieldStr = throughModel.SourceField()
		throughTargetFieldStr = throughModel.TargetField()

		sourceObject        = t.source.Object.FieldDefs()
		sourceObjectPrimary = sourceObject.Primary()
		sourceObjectPk, err = sourceObjectPrimary.Value()

		relations     = make([]Relation, 0, len(targets))
		throughModels = make([]attrs.Definer, 0, len(targets))
	)
	if err != nil {
		// return nil, created, fmt.Errorf("failed to get primary key for source object %T: %w", t.source.Object, err)
		return nil, created, errors.ValueError.WithCause(fmt.Errorf(
			"failed to get primary key for source object %T: %w: %w",
			t.source.Object, err, errors.NoUniqueKey,
		))
	}

	for _, target := range targets {
		var (
			// target related values
			targetDefs    = target.FieldDefs()
			targetPrimary = targetDefs.Primary()
			targetPk, err = targetPrimary.Value()

			// through model values
			ok          bool
			sourceField attrs.Field
			targetField attrs.Field
			newInstance = internal.NewObjectFromIface(throughObj)
			fieldDefs   = newInstance.FieldDefs()
		)
		if err != nil {
			return nil, created, errors.ValueError.WithCause(fmt.Errorf(
				"failed to get primary key for target %T: %w: %w",
				target, err, errors.NoUniqueKey,
			))
		}

		if sourceField, ok = fieldDefs.Field(throughSourceFieldStr); !ok {
			return nil, created, errors.FieldNotFound.WithCause(fmt.Errorf(
				"source field %s not found in through model %T: %w",
				throughSourceFieldStr, throughObj, errors.FieldNotFound,
			))
		}
		if targetField, ok = fieldDefs.Field(throughTargetFieldStr); !ok {
			return nil, created, errors.FieldNotFound.WithCause(fmt.Errorf(
				"target field %s not found in through model %T: %w",
				throughTargetFieldStr, throughObj, errors.FieldNotFound,
			))
		}

		if err := sourceField.Scan(sourceObjectPk); err != nil {
			return nil, created, errors.ValueError.WithCause(err)
		}
		if err := targetField.Scan(targetPk); err != nil {
			return nil, created, errors.ValueError.WithCause(err)
		}

		// Create a new relation object
		var rel = &baseRelation{
			uniqueValue: targetPk,
			object:      target,
			through:     newInstance,
		}

		throughModels = append(throughModels, newInstance)
		relations = append(relations, rel)
	}

	_, err = GetQuerySet(throughObj).BulkCreate(throughModels)
	if err != nil {
		return nil, created, err
	}

	return relations, created, nil
}

func (t *relatedQuerySet[T, T2]) Filter(key any, vals ...any) T2 {
	t.setup()
	t.qs = t.qs.Filter(key, vals...)
	return t.embedder
}

func (t *relatedQuerySet[T, T2]) OrderBy(fields ...string) T2 {
	t.setup()
	t.qs = t.qs.OrderBy(fields...)
	return t.embedder
}

func (t *relatedQuerySet[T, T2]) Limit(limit int) T2 {
	t.setup()
	t.qs = t.qs.Limit(limit)
	return t.embedder
}

func (t *relatedQuerySet[T, T2]) Offset(offset int) T2 {
	t.setup()
	t.qs = t.qs.Offset(offset)
	return t.embedder
}

func (t *relatedQuerySet[T, T2]) Get() (*Row[T], error) {
	t.setup()
	if t.qs == nil {
		panic("QuerySet is nil, cannot call Get()")
	}
	return t.qs.Get()
}

func (t *relatedQuerySet[T, T2]) All() (Rows[T], error) {
	t.setup()
	if t.qs == nil {
		panic("QuerySet is nil, cannot call All()")
	}
	return t.qs.All()
}

type RelManyToOneQuerySet[T attrs.Definer] struct {
	backRef                                       MultiRelationValue
	*relatedQuerySet[T, *RelManyToOneQuerySet[T]] // Embedding the relatedQuerySet to inherit its methods
}

func ManyToOneQuerySet[T attrs.Definer](backRef MultiRelationValue) *RelManyToOneQuerySet[T] {
	var parentInfo = backRef.ParentInfo()
	var mQs = &RelManyToOneQuerySet[T]{
		backRef: backRef,
	}
	mQs.relatedQuerySet = newRelatedQuerySet[T](mQs, parentInfo.Field.Rel(), parentInfo)
	return mQs
}

type RelManyToManyQuerySet[T attrs.Definer] struct {
	backRef                                        MultiThroughRelationValue
	*relatedQuerySet[T, *RelManyToManyQuerySet[T]] // Embedding the relatedQuerySet to inherit its methods
}

func ManyToManyQuerySet[T attrs.Definer](backRef MultiThroughRelationValue) *RelManyToManyQuerySet[T] {
	var parentInfo = backRef.ParentInfo()
	var mQs = &RelManyToManyQuerySet[T]{
		backRef: backRef,
	}
	mQs.relatedQuerySet = newRelatedQuerySet[T](mQs, parentInfo.Field.Rel(), parentInfo)
	return mQs
}

func (r *RelManyToManyQuerySet[T]) AddTarget(target T) (created bool, err error) {
	added, err := r.AddTargets(target)
	if err != nil {
		return false, err
	}
	return added == 1, nil
}

func (r *RelManyToManyQuerySet[T]) AddTargets(targets ...T) (int64, error) {
	if r.backRef == nil {
		return 0, fmt.Errorf("back reference is nil, cannot add targets")
	}

	var relations, added, err = r.createThroughObjects(targets)
	if err != nil {
		return 0, fmt.Errorf("failed to create through objects: %w", err)
	}

	if len(relations) == 0 {
		return added, fmt.Errorf("no relations created for targets %T", targets)
	}

	var relList = r.backRef.GetValues()
	if relList == nil {
		relList = make([]Relation, 0)
	}

	relList = append(relList, relations...)

	r.backRef.SetValues(relList)
	return added, nil
}

func (r *RelManyToManyQuerySet[T]) SetTargets(targets []T) (added int64, err error) {
	if r.backRef == nil {
		return 0, fmt.Errorf("back reference is nil, cannot set targets")
	}

	_, err = r.ClearTargets()
	if err != nil {
		return 0, fmt.Errorf("failed to clear targets: %w", err)
	}

	relations, added, err := r.createThroughObjects(targets)
	if err != nil {
		return 0, fmt.Errorf("failed to create through objects: %w", err)
	}

	if len(relations) == 0 {
		return added, fmt.Errorf("no relations created for targets %T", targets)
	}

	r.backRef.SetValues(relations)
	return added, nil
}

func (r *RelManyToManyQuerySet[T]) RemoveTargets(targets ...any) (int64, error) {
	if r.backRef == nil {
		return 0, fmt.Errorf("back reference is nil, cannot remove targets")
	}

	targets = internal.ListUnpack(targets)

	var (
		pkValues = make([]any, 0, len(targets))
		pkMap    = make(map[any]struct{}, len(targets))
	)
targetLoop:
	for _, target := range targets {
		var pkValue any
		if canPk, ok := target.(canUniqueKey); ok {
			pkValue = canPk.UniqueKey()
		}

		if pkValue != nil {
			pkValues = append(pkValues, pkValue)
			pkMap[pkValue] = struct{}{}
			continue targetLoop
		}

		var val, err = GetUniqueKey(target)
		if err != nil {
			return 0, errors.ValueError.WithCause(fmt.Errorf(
				"failed to get unique key for target %T: %w: %w",
				target, err, errors.NoUniqueKey,
			))
		}

		pkValues = append(pkValues, val)
		pkMap[val] = struct{}{}
	}

	var throughModel = newThroughProxy(r.rel.Through())
	var throughQs = GetQuerySet(throughModel.object).
		Filter(
			expr.Q(
				throughModel.sourceField.Name(),
				r.source.Object.FieldDefs().Primary().GetValue(),
			),
			expr.Q(
				fmt.Sprintf(
					"%s__in",
					throughModel.targetField.Name(),
				),
				pkValues...,
			),
		)

	var deleted, err = throughQs.Delete()
	if err != nil {
		return 0, fmt.Errorf("failed to delete through objects: %w", err)
	}

	if deleted == 0 {
		// return 0, fmt.Errorf("no through objects deleted for targets %v", pkValues)
		return 0, errors.NoChanges.WithCause(fmt.Errorf(
			"no through objects deleted for targets %v", pkValues,
		))
	}

	var relList = r.backRef.GetValues()
	if len(relList) == 0 {
		return deleted, nil
	}

	var newRels = make([]Relation, 0, len(relList))
	for _, rel := range relList {
		var (
			model     = rel.Model()
			fieldDefs = model.FieldDefs()
			pkValue   = fieldDefs.Primary().GetValue()
		)

		if fields.IsZero(pkValue) {
			goto uniqueKeyCheck
		}

		if _, ok := pkMap[pkValue]; !ok {
			newRels = append(newRels, rel)
		}

	uniqueKeyCheck:
		var val, err = GetUniqueKey(model)
		if err != nil {
			return 0, errors.ValueError.WithCause(fmt.Errorf(
				"failed to get unique key for relation %T: %w: %w",
				model, err, errors.NoUniqueKey,
			))
		}

		if _, ok := pkMap[val]; !ok {
			newRels = append(newRels, rel)
		}
	}

	r.backRef.SetValues(newRels)
	return deleted, nil
}

func (r *RelManyToManyQuerySet[T]) ClearTargets() (int64, error) {
	if r.backRef == nil {
		return 0, fmt.Errorf("back reference is nil, cannot clear targets")
	}

	var throughModel = newThroughProxy(r.rel.Through())
	var throughIdsResult, err = r.qs.Select(r.qs.Meta().PrimaryKey().Name()).ValuesList()
	if err != nil {
		return 0, fmt.Errorf("failed to get through object IDs: %w", err)
	}

	var throughIds = make([]any, 0, len(throughIdsResult))
	for _, id := range throughIdsResult {
		throughIds = append(throughIds, id[0])
	}

	var throughQs = GetQuerySet(throughModel.object).
		Filter(
			expr.Q(
				throughModel.sourceField.Name(),
				r.source.Object.FieldDefs().Primary().GetValue(),
			),
			expr.Expr(
				throughModel.targetField.Name(), expr.LOOKUP_IN, throughIds,
			),
		)

	deleted, err := throughQs.Delete()
	if err != nil {
		return 0, fmt.Errorf("failed to delete through objects: %w", err)
	}

	r.backRef.SetValues([]Relation{}) // Clear the back reference values

	return deleted, nil
}
