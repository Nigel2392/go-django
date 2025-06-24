package fields

import (
	"github.com/Nigel2392/go-django/src/core/attrs"
)

type OneToOneField[T any] struct {
	*RelationField[T]
}

func NewOneToOneField[T any](forModel attrs.Definer, name string, conf *FieldConfig) *OneToOneField[T] {

	if conf == nil {
		panic("NewForeignKeyField: config is nil")
	}

	if conf.Rel != nil {
		conf.Rel = &typedRelation{
			Relation: conf.Rel,
			typ:      attrs.RelOneToOne,
		}
	}

	var f = &OneToOneField[T]{
		RelationField: NewRelatedField[T](
			forModel,
			name,
			conf,
		),
	}
	f.DataModelField.fieldRef = f // Set the field reference to itself
	f.DataModelField.setupInitialVal()
	return f
}

type OneToOneReverseField[T any] struct {
	*RelationField[T]
}

func NewOneToOneReverseField[T any](forModel attrs.Definer, name string, conf *FieldConfig) *OneToOneReverseField[T] {
	if conf == nil {
		panic("NewForeignKeyField: config is nil")
	}

	if conf.Rel != nil {
		conf.Rel = &typedRelation{
			Relation: conf.Rel,
			typ:      attrs.RelOneToOne,
		}
	}

	var f = &OneToOneReverseField[T]{
		RelationField: NewRelatedField[T](
			forModel,
			name,
			conf,
		),
	}
	f.DataModelField.fieldRef = f // Set the field reference to itself
	f.DataModelField.setupInitialVal()
	return f
}
