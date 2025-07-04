package queries

import (
	"iter"
	"strings"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/errs"
)

const errStopIteration errs.Error = "stop iteration"

// A row represents a single row in the result set of a QuerySet.
//
// It contains the model object, a map of annotations, and a pointer to the QuerySet.
//
// The annotations map contains additional data that is not part of the model object,
// such as calculated fields or additional information derived from the query.
type Row[T attrs.Definer] struct {
	Object      T
	Through     attrs.Definer // The through model instance, if applicable
	Annotations map[string]any
	QuerySet    *QuerySet[T]
}

// A collection of Row[T] objects, where T is a type that implements attrs.Definer.
//
// This collection is used to represent the result set of a QuerySet.
type Rows[T attrs.Definer] []*Row[T]

func (r Rows[T]) Len() int {
	return len(r)
}

// Objects returns a sequence of model objects from the rows.
func (rows Rows[T]) Objects() iter.Seq[T] {
	return iter.Seq[T](func(yield func(T) bool) {
		for _, row := range rows {
			if !yield(row.Object) {
				return // Stop yielding if the yield function returns false
			}
		}
	})
}

// Pluck returns a sequence of field values from the rows based on the provided path to the field.
//
// The path to the field is a dot-separated string that specifies the field to pluck from each row.
//
// It traverses the field definitions of each row's object to find the specified field,
// and yields the index of that field.
// The index will be increased by one for each field processed.
func (rows Rows[T]) Pluck(pathToField string) iter.Seq2[int, attrs.Field] {
	var (
		idx        = 0
		fieldNames = strings.Split(pathToField, ".")
	)

	return iter.Seq2[int, attrs.Field](func(yield func(int, attrs.Field) bool) {
		var yieldFn = func(w walkInfo) bool {
			return yield(idx, w.field)
		}

		for _, row := range rows {
			var err = walkFieldValues(row.Object.FieldDefs(), fieldNames, &idx, 0, yieldFn)
			if errors.Is(err, errStopIteration) {
				return // Stop iteration if the yield function returned false
			}
			if err != nil && errors.Is(err, errors.FieldNotFound) {
				panic(errors.Wrapf(err, "error getting field %s from row", pathToField))
			} else if err != nil {
				panic(errors.Wrapf(err, "error getting field %s from row", pathToField))
			}
		}
	})
}

// PluckRowValues returns a sequence of values from the rows based on the provided path to the field.
//
// The path to the field is a dot-separated string that specifies the field to pluck from each row.
// It traverses the field definitions of each row's object to find the specified field,
// and yields the index of that field along with its value.
// The index will be increased by one for each field processed.
func PluckRowValues[ValueT any, ModelT attrs.Definer](rows Rows[ModelT], pathToField string) iter.Seq2[int, ValueT] {
	return func(yield func(int, ValueT) bool) {
		for idx, field := range rows.Pluck(pathToField) {
			var value = field.GetValue()
			if value == nil {
				if !yield(idx, *new(ValueT)) {
					return // Stop yielding if the yield function returns false
				}
			}
			if v, ok := value.(ValueT); ok {
				if !yield(idx, v) {
					return // Stop yielding if the yield function returns false
				}
			} else {
				panic(errors.Errorf(errors.CodeTypeMismatch, "type mismatch on pluckRows: %v (%T != %T)", value, *new(ValueT), value))
			}
		}
	}
}
