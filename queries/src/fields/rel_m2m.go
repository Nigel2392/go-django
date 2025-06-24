package fields

import (
	"github.com/Nigel2392/go-django/src/core/attrs"
)

//
//type RelatedQuerySet[T any] interface {
//	Filter(any, ...any) RelatedQuerySet[T]
//	OrderBy(...string) RelatedQuerySet[T]
//	Reverse() RelatedQuerySet[T]
//	Limit(int) RelatedQuerySet[T]
//	Offset(int) RelatedQuerySet[T]
//
//	All() ([]T, error)
//	Get() (T, bool)
//	First() (T, error)
//	Last() (T, error)
//	Exists() (bool, error)
//	Count() (int64, error)
//}
//
//type ManyToManyRelation[T any] struct {
//	cached  orderedmap.OrderedMap[any, T]
//	latestQ queries.QueryInfo
//}
//
//func (r *ManyToManyRelation[T]) Set(objs []T) error {
//
//}
//
//func (r *ManyToManyRelation[T]) Add(obj T) error {
//
//}
//
//func (r *ManyToManyRelation[T]) Remove(obj T) error {
//
//}
//
//func (r *ManyToManyRelation[T]) Clear() error {
//
//}

type ManyToManyField[T any] struct {
	*RelationField[T]
}

func NewManyToManyField[T any](forModel attrs.Definer, name string, conf *FieldConfig) *ManyToManyField[T] {
	if conf == nil {
		panic("NewForeignKeyField: config is nil")
	}

	if conf.Rel != nil {
		conf.Rel = &typedRelation{
			Relation: conf.Rel,
			typ:      attrs.RelManyToMany,
		}
	}

	var f = &ManyToManyField[T]{
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
