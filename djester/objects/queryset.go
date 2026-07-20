package objects

import (
	"context"
	"fmt"
	"iter"
	"reflect"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/djester"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

var _ djester.Test = (*QuerySetTest)(nil)

type QuerySetTest struct {
	Label string

	// Any and all objects created MUST use the queryset to do so.
	// This is so the queries can be reverted later.
	// []T.(attrs.Definer), func(ctx context.Context, t *testing.T)
	Create any

	Execute func(dj *djester.Tester, t *testing.T, ctx context.Context)

	Teardown func(*djester.Tester, *testing.T, context.Context)
}

func (q *QuerySetTest) Name() string { return q.Label }

func (f *QuerySetTest) Bench(dj *djester.Tester, b *testing.B) {
	b.Skipf("%T(%s) does not implenent benchmarks", f, f.Label)
}

func isSlice(r any) bool {
	switch chk := r.(type) {
	case reflect.Type:
		return (chk.Kind() == reflect.Slice || chk.Kind() == reflect.Array)
	case reflect.Value:
		return (chk.Kind() == reflect.Slice || chk.Kind() == reflect.Array)
	}
	panic(fmt.Sprintf("type %T not reflect.Type or reflect.Value", r))
}

func unpackSlice[T any](rv reflect.Value) (iter.Seq2[reflect.Value, error], error) {
	if !isSlice(rv) {
		return nil, errors.TypeMismatch.Wrapf("%s is not a slice or array type", rv.Type().String())
	}

	var zero int
	var t_typ = reflect.TypeFor[T]()
	var fn = func(yield func(reflect.Value, error) bool) {
		if !_unpackSlice(rv, t_typ, &zero, yield) {
			return
		}
	}
	return fn, nil
}

var __any = reflect.TypeFor[interface{}]()

func _unpackSlice(rv reflect.Value, wantedType reflect.Type, yielded *int, yield func(reflect.Value, error) bool) bool {
	if !rv.IsValid() {
		goto isZero
	}

	if rv.IsZero() || rv.IsNil() {
		goto isZero
	}

	if rv.Type() == __any {
		rv = rv.Elem()
	}

	if isSlice(rv) {
		for i := 0; i < rv.Len(); i++ {
			if !_unpackSlice(rv.Index(i), wantedType, yielded, yield) {
				return false
			}
		}
		return true
	}

	switch {
	case wantedType == rv.Type():
	case wantedType.Kind() == reflect.Interface && rv.Type().Implements(wantedType):
	case rv.Type().ConvertibleTo(wantedType):
		rv = rv.Convert(wantedType)
	default:
		yield(reflect.Value{}, errors.TypeMismatch.Wrapf(
			"[%d] %T does not implement or is convertible not to %s or []%s",
			*yielded, rv.Interface(), wantedType.String(), wantedType.String(),
		))
		return false
	}

	*yielded++
	return yield(rv, nil)

isZero:
	yield(reflect.Value{}, errors.ValueError.Wrapf(
		"[%d] invalid value provided, not convertible to %s",
		*yielded, wantedType,
	))
	return false
}

func orderedDefinerSlice(rv reflect.Value) iter.Seq2[[]attrs.Definer, error] {
	var (
		idx         = 0
		currentType reflect.Type
		currentList = make([]attrs.Definer, 0)
		iterFn, err = unpackSlice[attrs.Definer](rv)
	)

	if err != nil {
		return func(yield func([]attrs.Definer, error) bool) {
			yield(nil, err)
		}
	}

	return func(yield func([]attrs.Definer, error) bool) {
		for objectVal, err := range iterFn {
			if err != nil {
				yield(nil, err)
				return
			}

			var objValType = objectVal.Type()
			if idx == 0 {
				currentType = objValType
				currentList = append(
					currentList,
					objectVal.Interface().(attrs.Definer),
				)
				idx++
				continue
			}

			if objValType != currentType {
				if !yield(currentList, nil) {
					return
				}

				currentList = make([]attrs.Definer, 0)
				currentType = objValType
			}

			currentList = append(
				currentList,
				objectVal.Interface().(attrs.Definer),
			)

			idx++
		}

		if len(currentList) > 0 {
			yield(currentList, nil)
		}
	}
}

func (q *QuerySetTest) createObjectsFunc() func(ctx context.Context, t *testing.T) {
	if q.Create == nil {
		return func(ctx context.Context, t *testing.T) {
			t.Logf("[%T] No create function provided", q)
		}
	}

	switch fn := q.Create.(type) {
	case func(ctx context.Context, t *testing.T):
		return fn
	}

	return func(ctx context.Context, t *testing.T) {

		t.Logf("creating objects...")

		for objectList, err := range orderedDefinerSlice(reflect.ValueOf(q.Create)) {
			if err != nil {
				t.Fatalf("error whilst unpacking object list: %v", err)
			}

			var (
				err        error
				listString strings.Builder
				obj        = attrs.NewObject[attrs.Definer](ctx, reflect.TypeOf(objectList[0]))
				objQs      = queries.GetQuerySet(obj).WithContext(ctx)
			)

			for idx, obj := range objectList {
				if idx > 0 {
					listString.WriteString(", ")
				}

				fmt.Fprintf(&listString, "%v", obj)
			}

			t.Logf("Creating %v...", listString.String())

			_, err = objQs.BulkCreate(objectList)
			if err != nil {
				t.Fatalf("error while setting up %T, could not create objects: %v", q, err)
			}
		}
	}
}

func (q *QuerySetTest) Test(dj *djester.Tester, t *testing.T) {

	var ctx, tx, err = queries.StartTransaction(t.Context())
	if err != nil {
		t.Fatalf("faield to start transaction: %v", err)
	}

	// roll back at the end
	// commit will never be called
	defer tx.Rollback(ctx)

	var createFn = q.createObjectsFunc()
	createFn(ctx, t)

	q.Execute(dj, t, ctx)

	if q.Teardown != nil {
		q.Teardown(dj, t, ctx)
	}
}
