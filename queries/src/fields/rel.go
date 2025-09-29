package fields

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/src/core/attrs"

	_ "unsafe"
)

var (
	_ queries.TargetClauseField        = (*RelationField[any])(nil)
	_ queries.TargetClauseThroughField = (*RelationField[any])(nil)
	_ queries.ProxyField               = (*RelationField[any])(nil)
	_ queries.ProxyThroughField        = (*RelationField[any])(nil)
	_ queries.ForUseInQueriesField     = (*RelationField[any])(nil)
	_ attrs.CanRelatedName             = (*RelationField[any])(nil)

	_ attrs.LazyRelation = (*typedRelation)(nil)
	_ attrs.LazyRelation = (*wrappedRelation)(nil)
)

type RelationField[T any] struct {
	*DataModelField[T]
	cnf *FieldConfig
}

type typedRelation struct {
	attrs.Relation
	typ attrs.RelationType
}

func (r *typedRelation) Type() attrs.RelationType {
	return r.typ
}

func (r *typedRelation) ModelKey() string {
	if r.Relation == nil {
		return ""
	}
	if lr, ok := r.Relation.(attrs.LazyRelation); ok {
		return lr.ModelKey()
	}
	return ""
}

type wrappedRelation struct {
	attrs.Relation
	from attrs.RelationTarget
}

func (r *wrappedRelation) ModelKey() string {
	if r.Relation == nil {
		return ""
	}
	if lr, ok := r.Relation.(attrs.LazyRelation); ok {
		return lr.ModelKey()
	}
	return ""
}

func (r *wrappedRelation) From() attrs.RelationTarget {
	if r.from == nil {
		return r.Relation.From()
	}
	return r.from
}

type relationTarget struct {
	model    attrs.Definer
	field    attrs.FieldDefinition
	fieldStr string
	prev     attrs.RelationTarget
}

func (r *relationTarget) From() attrs.RelationTarget {
	return r.prev
}

func (r *relationTarget) Model() attrs.Definer {
	return r.model
}

func (r *relationTarget) Field() attrs.FieldDefinition {
	if r.field != nil {
		return r.field
	}

	var meta = attrs.GetModelMeta(r.model)
	var defs = meta.Definitions()
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

func NewRelatedField[T any](forModel attrs.Definer, name string, cnf *FieldConfig) *RelationField[T] {
	//var (
	//	inst = field.Instance()
	//	defs = inst.FieldDefs()
	//)

	if cnf == nil {
		panic(fmt.Sprintf("NewRelatedField: config is nil for field %q in model %T", name, forModel))
	}

	if cnf.Rel == nil {
		panic(fmt.Sprintf("NewRelatedField: relation is nil for field %q in model %T", name, forModel))
	}

	if cnf.IsProxy && (cnf.Rel.Type() != attrs.RelOneToOne && cnf.Rel.Type() != attrs.RelManyToOne) {
		panic(fmt.Sprintf(
			"NewRelatedField: relation type %s is not supported for proxy fields in model %T",
			cnf.Rel.Type(), forModel,
		))
	}

	if cnf.IsProxy && cnf.Nullable {
		panic(fmt.Sprintf(
			"NewRelatedField: proxy field %q in model %T cannot be nullable",
			name, forModel,
		))
	}

	if cnf.TargetField != "" && (cnf.Rel.Field() == nil || cnf.Rel.Field().Name() != cnf.TargetField) {
		cnf.Rel = &typedRelation{
			Relation: attrs.Relate(
				cnf.Rel.Model(),
				cnf.TargetField,
				cnf.Rel.Through(),
			),
			typ: cnf.Rel.Type(),
		}
	}

	if cnf.ReverseName == "" {
		var nameParts = strings.Split(reflect.TypeOf(forModel).Elem().Name(), ".")
		var modelName = nameParts[len(nameParts)-1]
		switch cnf.Rel.Type() {
		case attrs.RelOneToOne:
			cnf.ReverseName = fmt.Sprintf("%sReverse", modelName)
		case attrs.RelManyToOne:
			cnf.ReverseName = fmt.Sprintf("%sSet", modelName)
		case attrs.RelManyToMany:
			cnf.ReverseName = fmt.Sprintf("%sSet", modelName)
		case attrs.RelOneToMany:
			cnf.ReverseName = fmt.Sprintf("%sReverse", modelName)
		default:
			panic(fmt.Sprintf(
				"NewRelatedField: unsupported relation type %s for field %q in model %T",
				cnf.Rel.Type(), name, forModel,
			))
		}
	}

	var f = &RelationField[T]{
		DataModelField: NewDataModelField[T](forModel, cnf.ScanTo, name, cnf.DataModelFieldConfig),
		cnf:            cnf,
	}
	f.DataModelField.fieldRef = f // Set the field reference to itself
	f.DataModelField.setupInitialVal()
	return f
}

func (m *RelationField[T]) Name() string {
	return m.DataModelField.Name()
}

func (m *RelationField[T]) AllowNull() bool {
	// Relations are never nullable, they always have a value
	// even if it's an empty slice or nil for one-to-one relations.
	return m.cnf.Nullable
}

func (m *RelationField[T]) ForSelectAll() bool {
	return false
}

func (r *RelationField[T]) CanMigrate() bool {
	var relType = r.cnf.Rel.Type()
	return !(relType == attrs.RelManyToMany) && !(relType == attrs.RelOneToMany) &&
		!(relType == attrs.RelOneToOne && r.cnf.Through != nil)
}

func (r *RelationField[T]) ColumnName() string {
	if r.cnf.ColumnName == "" {
		var from = r.cnf.Rel.From()
		if from != nil {
			return from.Field().ColumnName()
		}

		if !r.IsProxy() && r.cnf.ColumnName == "" {
			switch r.cnf.Rel.Type() {
			case attrs.RelOneToOne, attrs.RelOneToMany, attrs.RelManyToMany:
				var meta = attrs.GetModelMeta(r.Model)
				var defs = meta.Definitions()
				var primary = defs.Primary()

				return primary.ColumnName()
			}
		}

		return attrs.ColumnName(r.Name())
	}
	return r.cnf.ColumnName
}

func (r *RelationField[T]) AllowReverseRelation() bool {
	// If the field is a proxy, it cannot have a reverse relation
	if r.cnf.IsProxy {
		return false
	}

	return !r.cnf.NoReverseRelation
}

type (
	saveableDefiner interface {
		attrs.Definer
		Save(ctx context.Context) error
	}
	saveableRelation interface {
		queries.Relation
		Save(ctx context.Context, parent attrs.Definer) error
	}
	canSetup interface {
		Setup(def attrs.Definer) error
	}
)

func (r *RelationField[T]) AllowEdit() bool {
	switch r.cnf.Rel.Type() {
	case attrs.RelOneToOne, attrs.RelManyToOne:
		return r.cnf.IsProxy && r.cnf.AllowEdit
	}
	return r.cnf.AllowEdit
}

func (r *RelationField[T]) AllowDBEdit() bool {
	switch r.cnf.Rel.Type() {
	case attrs.RelOneToOne, attrs.RelManyToOne:
		return r.cnf.AllowEdit
	}
	return false
}

func (r *RelationField[T]) Save(ctx context.Context) error {
	var val = r.GetValue()
	if val == nil {
		return nil
	}

	if canSetup, ok := val.(canSetup); ok {
		if err := canSetup.Setup(val.(attrs.Definer)); err != nil {
			return fmt.Errorf("failed to setup value for relation %s: %w", r.Name(), err)
		}
	}

	switch r.cnf.Rel.Type() {
	case attrs.RelManyToMany, attrs.RelOneToMany:
		return errors.NotImplemented.WithCause(fmt.Errorf(
			"cannot save relation %s with type %s", r.Name(), r.cnf.Rel.Type(),
		))

	//	case attrs.RelOneToMany:
	//		v := val.(*queries.RelRevFK[attrs.Definer])
	//		var createList = make([]attrs.Definer, 0)
	//		var updateList = make([]attrs.Definer, 0)
	//		for _, item := range v.GetValues() {
	//			if fields.IsZero(attrs.PrimaryKey(item)) {
	//				createList = append(createList, item)
	//			} else {
	//				updateList = append(updateList, item)
	//			}
	//		}
	//		if len(createList) > 0 {
	//			queries.GetQuerySet(r.cnf.Rel.Model()).WithContext(ctx).BulkCreate(createList)
	//		}
	//		if len(updateList) > 0 {
	//			queries.GetQuerySet(r.cnf.Rel.Model()).WithContext(ctx).BulkUpdate(updateList)
	//		}
	//		return nil

	case attrs.RelOneToOne:

		switch v := val.(type) {
		case saveableDefiner:
			return v.Save(ctx)
		case saveableRelation:
			return v.Save(ctx, r.Instance())
		}

	case attrs.RelManyToOne:
		if v, ok := val.(saveableDefiner); ok {
			return v.Save(ctx)
		}
	}

	return errors.NotImplemented.WithCause(fmt.Errorf(
		"cannot save relation %s with type %s, value %T does not implement saveableDefiner or saveableRelation",
		r.Name(), r.cnf.Rel.Type(), val,
	))
}

func (r *RelationField[T]) GetTargetField() attrs.FieldDefinition {
	var targetField = r.cnf.Rel.Field()
	if targetField == nil {
		var defs = r.cnf.Rel.Model().FieldDefs()
		return defs.Primary()
	}
	return targetField
}

func (r *RelationField[T]) IsReverse() bool {
	if r.cnf.IsProxy {
		return false
	}

	if r.cnf.IsReverse != nil {
		switch v := r.cnf.IsReverse.(type) {
		case bool:
			return v
		case func() bool:
			return v()
		case func(field attrs.Field) bool:
			return v(r)
		default:
			panic(fmt.Sprintf("IsReverse: unsupported type %T for field %q in model %T", v, r.Name(), r.Instance()))
		}
	}

	if r.cnf.Rel != nil && r.cnf.Rel.Type() == attrs.RelManyToOne {
		return false
	}

	var targetField = r.GetTargetField()
	if targetField == nil || targetField.IsPrimary() {
		return false
	}

	return true
}

func (r *RelationField[T]) Attrs() map[string]any {
	var atts = make(map[string]any)
	atts[attrs.AttrNameKey] = r.Name()
	atts[migrator.AttrUseInDBKey] = r.cnf.Rel.Through() == nil && !r.IsReverse()
	return atts
}

func (r *RelationField[T]) RelatedName() string {
	return r.cnf.ReverseName
}

func (r *RelationField[T]) Rel() attrs.Relation {
	return &wrappedRelation{
		Relation: r.cnf.Rel,
		from: &relationTarget{
			model: r.Instance(),
			field: r,
		},
	}
}

func (f *RelationField[T]) hasMany() bool {
	if f.cnf.Rel == nil {
		return false
	}
	var relType = f.cnf.Rel.Type()
	return !(relType == attrs.RelOneToOne || relType == attrs.RelManyToOne)
}

func (r *RelationField[T]) IsProxy() bool {
	return r.cnf.IsProxy
}

func (f *RelationField[T]) GenerateTargetClause(qs *queries.QuerySet[attrs.Definer], inter *queries.QuerySetInternals, lhs queries.ClauseTarget, rhs queries.ClauseTarget) queries.JoinDef {

	var joinType = expr.TypeJoinLeft
	if !f.cnf.Nullable && !f.hasMany() {
		joinType = expr.TypeJoinInner
	}

	return queries.JoinDef{
		Table:    rhs.Table,
		TypeJoin: joinType,
		JoinDefCondition: &queries.JoinDefCondition{
			ConditionA: expr.TableColumn{
				TableOrAlias: lhs.Table.Alias,
				FieldColumn:  lhs.Field,
			},
			Operator: expr.EQ,
			ConditionB: expr.TableColumn{
				TableOrAlias: rhs.Table.Alias,
				FieldColumn:  rhs.Field,
			},
		},
	}
}

func (f *RelationField[T]) GenerateTargetThroughClause(qs *queries.QuerySet[attrs.Definer], inter *queries.QuerySetInternals, lhs queries.ClauseTarget, thru queries.ThroughClauseTarget, rhs queries.ClauseTarget) (queries.JoinDef, queries.JoinDef) {

	var joinType = expr.TypeJoinLeft
	if !f.cnf.Nullable && !f.hasMany() {
		joinType = expr.TypeJoinInner
	}

	var sourceToThrough = queries.JoinDef{
		TypeJoin: joinType,
		Table:    thru.Table,
		JoinDefCondition: &queries.JoinDefCondition{
			Operator: expr.EQ,
			ConditionA: expr.TableColumn{
				TableOrAlias: lhs.Table.Alias,
				FieldColumn:  lhs.Field,
			},
			ConditionB: expr.TableColumn{
				TableOrAlias: thru.Table.Alias,
				FieldColumn:  thru.Left,
			},
		},
	}

	// JOIN target table
	var throughToTarget = queries.JoinDef{
		TypeJoin: joinType,
		Table:    rhs.Table,
		JoinDefCondition: &queries.JoinDefCondition{
			Operator: expr.EQ,
			ConditionA: expr.TableColumn{
				TableOrAlias: thru.Table.Alias,
				FieldColumn:  thru.Right,
			},
			ConditionB: expr.TableColumn{
				TableOrAlias: rhs.Table.Alias,
				FieldColumn:  rhs.Field,
			},
		},
	}

	return sourceToThrough, throughToTarget
}
