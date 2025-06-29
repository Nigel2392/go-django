package fields

import (
	"fmt"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
)

var (
	_ queries.TargetClauseField = (*genericForeignKeyField[attrs.Definer])(nil)
	// _ queries.SaveableDependantField = (*genericForeignKeyField[attrs.Definer])(nil)
)

type GenericForeignKeyField interface {
	queries.TargetClauseField
	queries.SaveableDependantField
}

type genericForeignKeyField[T attrs.Definer] struct {
	attrs.Field
	cnf  *GenericForeignKeyConfig
	obj  attrs.Definer
	defs attrs.Definitions

	targetPrimaryField attrs.Field
	targetCtypeField   attrs.Field
}

type GenericForeignKeyConfig struct {
	Nullable         bool
	Target           attrs.Definer
	RelPrimary       string
	ContentTypeField string
	TargetField      string
}

type GenericForeignKeyFieldDefiner interface {
	ContentTypeField() attrs.Field
	PrimaryField() attrs.Field
}

func GenericForeignKey[T attrs.Definer](definer T, name string, cnf *GenericForeignKeyConfig) interface{} {
	if cnf == nil || cnf.Target == nil {
		panic("GenericForeignKeyConfig.Target must not be nil")
	}

	return &genericForeignKeyField[T]{
		Field: attrs.NewField(
			definer,
			name,
			&attrs.FieldConfig{
				Null:     cnf.Nullable,
				ReadOnly: true,
				RelOneToOne: attrs.Relate(
					cnf.Target,
					cnf.RelPrimary, nil,
				),
			},
		),
		obj: definer,
		cnf: cnf,
	}
}

func (f *genericForeignKeyField[T]) BindToDefinitions(definitions attrs.Definitions) {
	f.Field.BindToDefinitions(definitions)
	f.defs = definitions
}

func (f *genericForeignKeyField[T]) ForSelectAll() bool {
	return false
}

func (f *genericForeignKeyField[T]) AllowReverseRelation() bool {
	return false
}

func (f *genericForeignKeyField[T]) SetValue(value interface{}, force bool) error {
	if err := f.Field.SetValue(value, force); err != nil {
		return fmt.Errorf(
			"failed to set value for generic foreign key field %s in model %T: %w",
			f.Name(), f.obj, err,
		)
	}

	var (
		definer            = value.(attrs.Definer)
		targetCtypeField   = f.ContentTypeField()
		targetPrimaryField = f.PrimaryField()
		targetDefs         = definer.FieldDefs()
		targetcType        = contenttypes.NewContentType(definer)
	)

	if err := targetCtypeField.SetValue(targetcType.TypeName(), force); err != nil {
		return fmt.Errorf(
			"failed to set value for target CType field %s in model %T: %w",
			targetCtypeField.Name(), f.obj, err,
		)
	}

	var relField = f.GetTargetPrimary()
	var relPrimaryInst, ok = targetDefs.Field(relField.Name())
	if !ok {
		return fmt.Errorf(
			"target model %T does not have a field with name %s",
			definer, relField.Name(),
		)
	}

	if err := targetPrimaryField.SetValue(relPrimaryInst.GetValue(), force); err != nil {
		return fmt.Errorf(
			"failed to set value for target ID field %s in model %T: %w",
			targetPrimaryField.Name(), f.obj, err,
		)
	}

	return nil
}

func (f *genericForeignKeyField[T]) GetTargetPrimary() attrs.FieldDefinition {
	if relField := f.Rel().Field(); relField != nil {
		return relField
	}

	var meta = attrs.GetModelMeta(f.cnf.Target)
	var defs = meta.Definitions()
	return defs.Primary()
}

func (f *genericForeignKeyField[T]) PrimaryField() attrs.Field {
	return f.targetPrimaryField
}

func (f *genericForeignKeyField[T]) ContentTypeField() attrs.Field {
	return f.targetCtypeField
}

func (f *genericForeignKeyField[T]) setupRelatedFields() {
	var (
		targetCtypeField   attrs.Field
		targetPrimaryField attrs.Field
	)

	if targetDefiner, ok := f.obj.(GenericForeignKeyFieldDefiner); ok {
		targetCtypeField = targetDefiner.ContentTypeField()
		targetPrimaryField = targetDefiner.PrimaryField()
	} else {
		var objDefs = f.obj.FieldDefs()
		targetCtypeField, ok = objDefs.Field(f.cnf.ContentTypeField)
		if !ok {
			panic(fmt.Sprintf(
				"generic foreign key field in model %T does not have a field with ctype name %q",
				f.cnf.Target, f.cnf.ContentTypeField,
			))
		}

		targetPrimaryField, ok = objDefs.Field(f.cnf.TargetField)
		if !ok {
			panic(fmt.Sprintf(
				"generic foreign key field in model %T does not have a field with target name %q",
				f.cnf.Target, f.cnf.TargetField,
			))
		}
	}

	f.targetCtypeField = targetCtypeField
	f.targetPrimaryField = targetPrimaryField
}

//
//func (f *genericForeignKeyField[T]) Save(ctx context.Context, parent attrs.Definer) error {
//
//	f.setupRelatedFields()
//
//	var objDefs = f.obj.FieldDefs()
//	var (
//		targetIDFieldDef = f.PrimaryField()
//		targetIDField, _ = objDefs.Field(
//			targetIDFieldDef.Name(),
//		)
//		targetCTypeFieldDef = f.ContentTypeField()
//		targetCTypeField, _ = objDefs.Field(
//			targetCTypeFieldDef.Name(),
//		)
//	)
//
//	var parentDefs = parent.FieldDefs()
//	var parentPrimary = parentDefs.Primary()
//
//	if err := targetIDField.SetValue(parentPrimary.GetValue(), true); err != nil {
//		return fmt.Errorf(
//			"failed to set value for target ID field %s in model %T: %w",
//			targetIDFieldDef.Name(), f.obj, err,
//		)
//	}
//
//	if err := targetCTypeField.SetValue(contenttypes.NewContentType(parent).TypeName(), true); err != nil {
//		return fmt.Errorf(
//			"failed to set value for target CType field %s in model %T: %w",
//			targetCTypeFieldDef.Name(), f.obj, err,
//		)
//	}
//
//	return nil
//}

func (f *genericForeignKeyField[T]) GenerateTargetClause(qs *queries.QuerySet[attrs.Definer], inter *queries.QuerySetInternals, lhs queries.ClauseTarget, rhs queries.ClauseTarget) queries.JoinDef {
	if f.cnf == nil || f.cnf.Target == nil {
		panic("GenericForeignKeyConfig.Target must not be nil")
	}

	f.setupRelatedFields()

	var sourceType = contenttypes.NewContentType(f.obj)
	var cond = queries.JoinDefCondition{
		ConditionA: expr.TableColumn{
			TableOrAlias: lhs.Table.Alias,
			FieldColumn:  f.PrimaryField(),
		},
		Operator: expr.EQ,
		ConditionB: expr.TableColumn{
			TableOrAlias: rhs.Table.Alias,
			FieldColumn:  f.GetTargetPrimary(),
		},
		Next: &queries.JoinDefCondition{
			ConditionA: expr.TableColumn{
				TableOrAlias: lhs.Table.Alias,
				FieldColumn:  f.ContentTypeField(),
			},
			Operator: expr.EQ,
			ConditionB: expr.TableColumn{
				RawSQL: "?",
				Value:  sourceType.TypeName(),
			},
		},
	}

	return queries.JoinDef{
		Table:            rhs.Table,
		TypeJoin:         queries.TypeJoinInner,
		JoinDefCondition: &cond,
	}
}
