package models

import (
	"context"
	"fmt"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/models"
)

var (
	_ queries.TargetClauseField      = (*proxyField)(nil)
	_ queries.ProxyField             = (*proxyField)(nil)
	_ queries.SaveableDependantField = (*proxyField)(nil)
)

type proxyField struct {
	attrs.Field
	cnf   *ProxyFieldConfig
	obj   attrs.Definer
	model *Model
	meta  attrs.ModelMeta

	targetPrimaryField attrs.FieldDefinition
	targetCtypeField   attrs.FieldDefinition
}

type ProxyFieldConfig struct {
	Proxy            attrs.Definer
	ContentTypeField string
	TargetField      string
}

func newProxyField(model *Model, definer attrs.Definer, niceName string, internalName string, cnf *ProxyFieldConfig) *proxyField {
	if cnf == nil || cnf.Proxy == nil {
		panic("ProxyFieldConfig.Proxy must not be nil")
	}

	return &proxyField{
		Field: attrs.NewField(
			definer,
			niceName,
			&attrs.FieldConfig{
				// AutoInit:     true,
				ReadOnly:     true,
				NameOverride: internalName,
				RelOneToOne: attrs.Relate(
					cnf.Proxy,
					"", nil,
				),
			},
		),
		meta:  attrs.GetModelMeta(definer),
		model: model,
		obj:   definer,
		cnf:   cnf,
	}
}

func (f *proxyField) ForSelectAll() bool {
	return false
}

func (f *proxyField) AllowReverseRelation() bool {
	return false
}

func (f *proxyField) CanMigrate() bool {
	return false
}

func (f *proxyField) GetSourcePrimary() attrs.FieldDefinition {
	var primary = f.model.internals.defs.Primary()
	if primary == nil {
		panic(fmt.Errorf(
			"ProxyFieldConfig.Proxy: %T (%s) does not have a primary field defined (%T)",
			f.cnf.Proxy, f.Name(), f.obj,
		))
	}
	return primary
}

func (f *proxyField) LinkedPrimaryField() attrs.FieldDefinition {
	return f.targetPrimaryField
}

func (f *proxyField) LinkedCTypeField() attrs.FieldDefinition {
	return f.targetCtypeField
}

func (f *proxyField) setupRelatedFields() {
	var (
		targetCtypeField   attrs.FieldDefinition
		targetPrimaryField attrs.FieldDefinition
	)

	if targetDefiner, ok := f.cnf.Proxy.(CanTargetDefiner); ok {
		targetCtypeField = targetDefiner.TargetContentTypeField()
		targetPrimaryField = targetDefiner.TargetPrimaryField()
	} else {
		var meta = attrs.GetModelMeta(f.cnf.Proxy)
		var targetDefs = meta.Definitions()
		targetCtypeField, ok = targetDefs.Field(f.cnf.ContentTypeField)
		if !ok {
			panic(fmt.Sprintf(
				"proxy field in model %T does not have a field with ctype name %q",
				f.cnf.Proxy, f.cnf.ContentTypeField,
			))
		}

		targetPrimaryField, ok = targetDefs.Field(f.cnf.TargetField)
		if !ok {
			panic(fmt.Sprintf(
				"proxy field in model %T does not have a field with target name %q",
				f.cnf.Proxy, f.cnf.TargetField,
			))
		}
	}

	f.targetCtypeField = targetCtypeField
	f.targetPrimaryField = targetPrimaryField
}

func (f *proxyField) IsProxy() bool {
	return true
}

func (f *proxyField) Save(ctx context.Context, parent attrs.Definer) error {

	var proxyObject = f.GetValue()
	if proxyObject == nil {
		return fmt.Errorf(
			"cannot save proxy field %s in model %T: value is nil",
			f.Name(), f.obj,
		)
	}

	var proxy, ok = proxyObject.(attrs.Definer)
	if !ok {
		return fmt.Errorf(
			"cannot save proxy field %s in model %T: value is not a definer, got %T",
			f.Name(), f.obj, proxyObject,
		)
	}

	f.setupRelatedFields()

	var proxyDefs = proxy.FieldDefs()
	var (
		targetIDFieldDef = f.LinkedPrimaryField()
		targetIDField, _ = proxyDefs.Field(
			targetIDFieldDef.Name(),
		)
		targetCTypeFieldDef = f.LinkedCTypeField()
		targetCTypeField, _ = proxyDefs.Field(
			targetCTypeFieldDef.Name(),
		)
	)

	var parentDefs = parent.FieldDefs()
	var parentPrimary = parentDefs.Primary()

	if err := targetIDField.SetValue(parentPrimary.GetValue(), true); err != nil {
		return fmt.Errorf(
			"failed to set value for target ID field %s in proxy model %T: %w",
			targetIDFieldDef.Name(), proxyObject, err,
		)
	}
	if err := targetCTypeField.SetValue(contenttypes.NewContentType(parent).TypeName(), true); err != nil {
		return fmt.Errorf(
			"failed to set value for target CType field %s in proxy model %T: %w",
			targetCTypeFieldDef.Name(), proxyObject, err,
		)
	}

	saver, ok := proxyObject.(models.ContextSaver)
	if !ok {
		return fmt.Errorf(
			"cannot save proxy field %s in model %T: value does not implement models.ContextSaver, got %T",
			f.Name(), f.obj, proxyObject,
		)
	}

	return saver.Save(ctx)
}

func (f *proxyField) GenerateTargetClause(qs *queries.QuerySet[attrs.Definer], inter *queries.QuerySetInternals, lhs queries.ClauseTarget, rhs queries.ClauseTarget) queries.JoinDef {
	if f.cnf == nil || f.cnf.Proxy == nil {
		panic("ProxyFieldConfig.Proxy must not be nil")
	}
	f.setupRelatedFields()
	var sourceType = contenttypes.NewContentType(f.obj)
	var cond = queries.JoinDefCondition{
		ConditionA: expr.TableColumn{
			TableOrAlias: lhs.Table.Alias,
			FieldColumn:  f.GetSourcePrimary(),
		},
		Operator: expr.EQ,
		ConditionB: expr.TableColumn{
			TableOrAlias: rhs.Table.Alias,
			FieldColumn:  f.LinkedPrimaryField(),
		},
		Next: &queries.JoinDefCondition{
			ConditionA: expr.TableColumn{
				TableOrAlias: rhs.Table.Alias,
				FieldColumn:  f.LinkedCTypeField(),
			},
			Operator: expr.EQ,
			ConditionB: expr.TableColumn{
				RawSQL: "?",
				Values: []any{sourceType.TypeName()},
			},
		},
	}

	return queries.JoinDef{
		Table:            rhs.Table,
		TypeJoin:         queries.TypeJoinInner,
		JoinDefCondition: &cond,
	}
}
