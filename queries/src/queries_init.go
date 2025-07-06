package queries

import (
	"context"
	"fmt"

	"github.com/Nigel2392/go-django/queries/internal"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/models"
	"github.com/Nigel2392/go-signals"
	"github.com/Nigel2392/goldcrest"
	"github.com/elliotchance/orderedmap/v2"
)

func init() {

	goldcrest.Register(models.MODEL_SAVE_HOOK, 0, models.ModelFunc(func(c context.Context, m attrs.Definer) (changed bool, err error) {
		if u, ok := m.(ForUseInQueries); ok && !u.ForUseInQueries() {
			return false, nil
		}

		var (
			defs         = m.FieldDefs()
			primaryField = defs.Primary()
		)

		primaryValue, err := primaryField.Value()
		if err != nil {
			return false, errors.ValueError.WithCause(err)
		}

		if primaryValue == nil || fields.IsZero(primaryValue) {
			var _, err = GetQuerySet(m).ExplicitSave().Create(m)
			if err != nil {
				return false, err
			}
			return true, nil
		}

		ct, err := GetQuerySet(m).
			ExplicitSave().
			Filter(
				primaryField.Name(), primaryValue,
			).
			Update(m)
		return ct > 0, err
	}))

	goldcrest.Register(models.MODEL_DELETE_HOOK, 0, models.ModelFunc(func(c context.Context, m attrs.Definer) (changed bool, err error) {
		if u, ok := m.(ForUseInQueries); ok && !u.ForUseInQueries() {
			return false, nil
		}

		var (
			defs         = m.FieldDefs()
			primaryField = defs.Primary()
		)

		primaryValue, err := primaryField.Value()
		if err != nil {
			return false, errors.ValueError.WithCause(err)
		}

		if primaryValue == nil || fields.IsZero(primaryValue) {
			return false, nil
		}

		ct, err := GetQuerySet(m).Filter(
			primaryField.Name(),
			primaryValue,
		).Delete()
		return ct > 0, err
	}))
}

var _, _ = attrs.OnBeforeModelRegister.Listen(func(s signals.Signal[attrs.SignalModelMeta], meta attrs.SignalModelMeta) error {

	var (
		def           = contenttypes.DefinitionForObject(meta.Definer)
		registerCType = false
		changeCType   = false
	)

	if def == nil {
		def = &contenttypes.ContentTypeDefinition{
			ContentObject:     meta.Definer,
			GetInstance:       CT_GetObject(meta.Definer),
			GetInstances:      CT_ListObjects(meta.Definer),
			GetInstancesByIDs: CT_ListObjectsByIDs(meta.Definer),
		}
		registerCType = true
	} else {
		if def.GetInstance == nil {
			def.GetInstance = CT_GetObject(meta.Definer)
			changeCType = true
		}
		if def.GetInstances == nil {
			def.GetInstances = CT_ListObjects(meta.Definer)
			changeCType = true
		}
		if def.GetInstancesByIDs == nil {
			def.GetInstancesByIDs = CT_ListObjectsByIDs(meta.Definer)
			changeCType = true
		}
	}

	switch {
	case changeCType:
		contenttypes.EditDefinition(def)
	case registerCType:
		contenttypes.Register(def)
	}

	return nil
})

// Return the unique fields and unique together fields
// for a model meta. This is used to generate a where clause
// for a model without a primary key defined.
//
// it is also used to generate a unique key for a model object
// when the model does not have a primary key defined.
//
// See [GetUniqueKey] and [GenerateObjectsWhereClause] for more details.
func getMetaUniqueFields(modelMeta attrs.ModelMeta) [][]string {
	var (
		modelDefs    = modelMeta.Definitions()
		uniqueFields [][]string
	)

	var uniqueTogetherObj, ok = modelMeta.Storage(MetaUniqueTogetherKey)
	if ok {
		var fields = make([][]string, 0, 1)
		switch uqFields := uniqueTogetherObj.(type) {
		case []string:
			fields = append(fields, uqFields)
		case [][]string:
			fields = append(fields, uqFields...)
		default:
			panic(fmt.Sprintf("unexpected type for ModelMeta.Storage(%q): %T, expected []string or [][]string", MetaUniqueTogetherKey, uqFields))
		}
		uniqueFields = append(uniqueFields, fields...)
	}

	for _, field := range modelDefs.Fields() {
		var attributes = field.Attrs()
		var isUnique, _ = internal.GetFromAttrs[bool](attributes, attrs.AttrUniqueKey)
		if isUnique {
			uniqueFields = append(uniqueFields, []string{field.Name()})
		}
	}

	return uniqueFields
}

// Registers a function to generate a where clause for a model
// without a primary key defined.
//
// This function will be called when a model object needs to be referenced
// in the queryset, for example when updating or deleting objects.
//
// See [GenerateObjectsWhereClause] for the implementation details when a primary key is defined.
var _, _ = attrs.OnModelRegister.Listen(func(s signals.Signal[attrs.SignalModelMeta], meta attrs.SignalModelMeta) error {
	var (
		modelDefs = meta.Meta.Definitions()
		primary   = modelDefs.Primary()
	)

	// Store the unique together fields on the model meta
	if def, ok := meta.Meta.Model().(UniqueTogetherDefiner); ok {
		var uqFields = def.UniqueTogether()
		if len(uqFields) > 0 {
			attrs.StoreOnMeta(
				meta.Meta.Model(), MetaUniqueTogetherKey, uqFields,
			)
		}
	}

	// See [GenerateObjectsWhereClause] for the implementation details
	// when a primary key is defined.
	if primary != nil {
		return nil
	}

	var uniqueFields = getMetaUniqueFields(meta.Meta)
	if len(uniqueFields) == 0 {
		//return fmt.Errorf(
		//	"model %T has no unique fields or unique together fields, cannot generate where clause: %w",
		//	meta.Definer, errors.ErrFieldNotFound,
		//)
		return errors.FieldNotFound.WithCause(fmt.Errorf(
			"model %T has no unique fields or unique together fields, cannot generate where clause",
			meta.Definer,
		))
	}

	// Create a generator function to generate a where clause
	//
	// It will use unique fields or unique together fields
	// to generate a where clause for the model.
	attrs.StoreOnMeta(meta.Definer, __GENERATE_WHERE_CLAUSE_FOR_OBJECTS, func(objects []attrs.Definer) ([]expr.ClauseExpression, error) {
		var orExprs = make([]expr.Expression, 0, len(uniqueFields))
		for _, object := range objects {
			var (
				defs    = object.FieldDefs()
				objExpr expr.ClauseExpression
			)
		uniqueFieldsLoop:
			for _, fieldNames := range uniqueFields {

				// List of and expressions for unique together fields
				// this also works for single unique fields
				var and = make([]expr.Expression, 0, len(fieldNames))
				for _, fieldName := range fieldNames {
					var field, ok = defs.Field(fieldName)
					if !ok {
						panic(fmt.Sprintf("field %q not found in model %T", fieldName, meta.Definer))
					}

					var val, err = field.Value()
					if err != nil {
						return nil, fmt.Errorf(
							"error getting value for field %q in model %T: %w",
							fieldName, meta.Definer, err,
						)
					}

					// If the value is nil or zero, we cannot use this field
					// to generate a where clause,
					//
					// we assume zero values are never unique, even
					// when the field is marked as unique together.
					if val == nil || fields.IsZero(val) {
						continue uniqueFieldsLoop
					}

					and = append(and, expr.Q(fieldName, val))
				}

				if len(and) == 0 {
					continue uniqueFieldsLoop
				}

				// set the object expression and break the loop
				objExpr = expr.And(and...)
				break uniqueFieldsLoop
			}

			// If we have no unique fields, we cannot generate a where clause
			if objExpr == nil {
				return nil, errors.NoWhereClause.WithCause(fmt.Errorf(
					"model %T has does not have enough unique fields or unique together fields set to generate a where clause",
					meta.Definer,
				))
			}

			// Add the object expression to the list of or expressions
			orExprs = append(orExprs, objExpr)
		}

		return []expr.ClauseExpression{expr.Or(orExprs...)}, nil
	})
	return nil
})

// Registers a function to generate a where clause for a through model
// without a primary key defined.
//
// This function will be called when a through object needs to be referenced
// in the queryset, for example when deleting through objects.
//
// See [GenerateObjectsWhereClause] for the implementation details
// when a primary key is defined.
//
// It generates a where clause for a list of through model objects
// that match the source and target fields of the through model meta.
var _, _ = attrs.OnThroughModelRegister.Listen(func(s signals.Signal[attrs.SignalThroughModelMeta], d attrs.SignalThroughModelMeta) error {

	var (
		throughModel   = d.ThroughInfo.Model()
		sourceFieldStr = d.ThroughInfo.SourceField()
		targetFieldStr = d.ThroughInfo.TargetField()
		throughDefs    = throughModel.FieldDefs()
	)

	if _, ok := throughDefs.Field(sourceFieldStr); !ok {
		panic(fmt.Sprintf(
			"source field %q not found in through model meta of %T (%T %+v)",
			sourceFieldStr, throughModel, d.ThroughInfo.(*attrs.ThroughModel).This, d.ThroughInfo,
		))
	}

	if _, ok := throughDefs.Field(targetFieldStr); !ok {
		panic(fmt.Sprintf(
			"target field %q not found in through model meta of %T (%T %+v)",
			targetFieldStr, throughModel, d.ThroughInfo.(*attrs.ThroughModel).This, d.ThroughInfo,
		))
	}

	// See [GenerateObjectsWhereClause] for the implementation details
	// when a primary key is defined.
	if throughDefs.Primary() == nil {

		// Gennerate a where clause for the through model
		// that matches the source and target fields.
		//
		// these fields are autimatically assumed to be unique together,
		// so we can use them to generate a where clause for the through model.
		//
		// this is used when deleting through objects for many-to-many or one-to-one relations
		// where the through model does not have a primary key defined.
		attrs.StoreOnMeta(throughModel, __GENERATE_WHERE_CLAUSE_FOR_OBJECTS, func(objects []attrs.Definer) ([]expr.ClauseExpression, error) {

			// groups of source object ids to target ids
			//
			// this is used to generate a where clause
			// that matches the source and target fields of the through model.
			//
			// i.e:
			//   source_id = 1 AND target_id IN (2, 3, 4)
			//   source_id = 4 AND target_id = 5
			var groups = orderedmap.NewOrderedMap[any, []any]()
			for _, object := range objects {
				var (
					err                      error
					ok                       bool
					sourceField, targetField attrs.Field
					sourceVal, targetVal     any
					group                    []any

					instDefs = object.FieldDefs()
				)

				// Get the values from the actual model instance
				if sourceField, ok = instDefs.Field(sourceFieldStr); !ok {
					panic("source field not found in through model meta")
				}

				if targetField, ok = instDefs.Field(targetFieldStr); !ok {
					panic("target field not found in through model meta")
				}

				sourceVal, err = sourceField.Value()
				if err != nil {
					panic(err)
				}

				targetVal, err = targetField.Value()
				if err != nil {
					panic(err)
				}

				if sourceVal == nil || targetVal == nil {
					return nil, fmt.Errorf(
						"source or target field value is nil for object %T, source: %v, target: %v",
						object, sourceVal, targetVal,
					)
				}

				group, ok = groups.Get(sourceVal)
				if !ok {
					group = make([]any, 0, 1)
					groups.Set(sourceVal, group)
				}

				group = append(group, targetVal)
				groups.Set(sourceVal, group)
			}

			// Generate the where clause expressions
			// based on the grouped source and target values.
			var expressions = make([]expr.ClauseExpression, 0, groups.Len())
			for head := groups.Front(); head != nil; head = head.Next() {
				var (
					source  = head.Key
					targets = head.Value
				)

				if len(targets) == 0 {
					continue
				}

				var sourceExpr = expr.Q(d.ThroughInfo.SourceField(), source)
				if len(targets) == 1 {
					var targetExpr = expr.Q(d.ThroughInfo.TargetField(), targets[0])
					expressions = append(
						expressions,
						expr.Express(sourceExpr, targetExpr)...,
					)
					continue
				}

				expressions = append(expressions, sourceExpr.And(
					expr.Expr(d.ThroughInfo.TargetField(), "in", targets...),
				))
			}

			return expressions, nil
		})
	}

	return nil
})
