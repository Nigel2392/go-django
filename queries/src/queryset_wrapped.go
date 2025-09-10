package queries

import (
	"context"
	"database/sql"
	"iter"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

var (
	_ BaseReadQuerySet[attrs.Definer, *QuerySet[attrs.Definer]] = (*WrappedQuerySet[attrs.Definer, *GenericQuerySet, *QuerySet[attrs.Definer]])(nil)
	_ BaseQuerySet[attrs.Definer, *QuerySet[attrs.Definer]]     = (*WrappedQuerySet[attrs.Definer, *GenericQuerySet, *QuerySet[attrs.Definer]])(nil)
)

type WrappedQuerySet[T attrs.Definer, CONV any, ORIG NullQuerySet[T, ORIG]] struct {
	NullQuerySet[T, ORIG]
	embedder CONV
}

func WrapQuerySet[T attrs.Definer, CONV any, ORIG NullQuerySet[T, ORIG]](qs ORIG, embedder CONV) *WrappedQuerySet[T, CONV, ORIG] {
	if _, ok := any(embedder).(QuerySetCanClone[T, CONV, ORIG]); !ok {
		panic("embedder must implement QuerySetCanClone[T, CONV, ORIG]")
	}

	return &WrappedQuerySet[T, CONV, ORIG]{
		NullQuerySet: qs,
		embedder:     embedder,
	}
}

type (
	QuerySetCanSetup interface {
		Setup()
	}
	QuerySetCanBeforeExec interface {
		BeforeExec() error
	}
	QuerySetCanAfterExec interface {
		AfterExec(res any) error
	}
	QuerySetCanClone[T attrs.Definer, CONV any, ORIG NullQuerySet[T, ORIG]] interface {
		CloneQuerySet(*WrappedQuerySet[T, CONV, ORIG]) CONV
	}
)

func (w *WrappedQuerySet[T, CONV, ORIG]) Base() ORIG {
	return w.NullQuerySet.Clone()
}

func (w *WrappedQuerySet[T, CONV, ORIG]) setup() {
	if canSetup, ok := any(w.embedder).(attrs.CanSetup); ok {
		canSetup.Setup()
	}
}

func (w *WrappedQuerySet[T, CONV, ORIG]) beforeReadExec() error {
	if canBeforeExec, ok := any(w.embedder).(QuerySetCanBeforeExec); ok {
		return canBeforeExec.BeforeExec()
	}
	return nil
}

func (w *WrappedQuerySet[T, CONV, ORIG]) afterReadExec(res any) error {
	if canAfterExec, ok := any(w.embedder).(QuerySetCanAfterExec); ok {
		return canAfterExec.AfterExec(res)
	}
	return nil
}

func (w *WrappedQuerySet[T, CONV, ORIG]) WithContext(ctx context.Context) CONV {
	w.NullQuerySet = w.NullQuerySet.WithContext(ctx)
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) BuildExpression() expr.Expression {
	w.setup()
	return w.NullQuerySet.BuildExpression()
}

func (w *WrappedQuerySet[T, CONV, ORIG]) clone() *WrappedQuerySet[T, CONV, ORIG] {
	var wrapped = &WrappedQuerySet[T, CONV, ORIG]{
		NullQuerySet: w.NullQuerySet.Clone(),
	}
	var cloner = any(w.embedder).(QuerySetCanClone[T, CONV, ORIG])
	wrapped.embedder = cloner.CloneQuerySet(wrapped)
	return wrapped
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Clone() CONV {
	w = w.clone()
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Distinct() CONV {
	w = w.clone()
	w.NullQuerySet = w.NullQuerySet.Distinct()
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Select(fields ...any) CONV {
	w = w.clone()
	w.setup()
	w.NullQuerySet = w.NullQuerySet.Select(fields...)
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Preload(fields ...any) CONV {
	w = w.clone()
	w.setup()
	w.NullQuerySet = w.NullQuerySet.Preload(fields...)
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Filter(key interface{}, vals ...interface{}) CONV {
	w = w.clone()
	w.setup()
	w.NullQuerySet = w.NullQuerySet.Filter(key, vals...)
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) GroupBy(fields ...any) CONV {
	w = w.clone()
	w.setup()
	w.NullQuerySet = w.NullQuerySet.GroupBy(fields...)
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Limit(n int) CONV {
	w = w.clone()
	w.NullQuerySet = w.NullQuerySet.Limit(n)
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Offset(n int) CONV {
	w = w.clone()
	w.NullQuerySet = w.NullQuerySet.Offset(n)
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) OrderBy(fields ...string) CONV {
	w = w.clone()
	w.setup()
	// w.BaseQuerySet = w.BaseQuerySet.OrderBy(fields...)
	w.NullQuerySet = w.NullQuerySet.OrderBy(fields...)
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Reverse() CONV {
	w = w.clone()
	w.NullQuerySet = w.NullQuerySet.Reverse()
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) ExplicitSave() CONV {
	w = w.clone()
	w.NullQuerySet = w.NullQuerySet.ExplicitSave()
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Annotate(aliasOrAliasMap interface{}, exprs ...expr.Expression) CONV {
	w = w.clone()
	w.setup()
	w.NullQuerySet = w.NullQuerySet.Annotate(aliasOrAliasMap, exprs...)
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) ForUpdate() CONV {
	w = w.clone()
	w.NullQuerySet = w.NullQuerySet.ForUpdate()
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Prefix(prefix string) CONV {
	w = w.clone()
	w.setup()
	w.NullQuerySet = w.NullQuerySet.Prefix(prefix)
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Having(key interface{}, vals ...interface{}) CONV {
	w = w.clone()
	w.setup()
	w.NullQuerySet = w.NullQuerySet.Having(key, vals...)
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) All() (Rows[T], error) {
	w.setup()

	if err := w.beforeReadExec(); err != nil {
		return nil, err
	}

	type readQs interface {
		All() (Rows[T], error)
	}

	if readQs, ok := w.NullQuerySet.(readQs); ok {
		res, err := readQs.All()
		if err != nil {
			return nil, err
		}
		return res, w.afterReadExec(res)
	}

	panic("All method not implemented on underlying queryset")
}

func (w *WrappedQuerySet[T, CONV, ORIG]) IterAll() (int, iter.Seq2[*Row[T], error], error) {
	w.setup()
	type readQs interface {
		IterAll() (int, iter.Seq2[*Row[T], error], error)
	}
	if readQs, ok := w.NullQuerySet.(readQs); ok {
		return readQs.IterAll()
	}
	panic("IterAll method not implemented on underlying queryset")
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Exists() (bool, error) {
	w.setup()
	if err := w.beforeReadExec(); err != nil {
		return false, err
	}

	type readQs interface {
		Exists() (bool, error)
	}

	if readQs, ok := w.NullQuerySet.(readQs); ok {
		exists, err := readQs.Exists()
		if err != nil {
			return false, err
		}
		return exists, w.afterReadExec(exists)
	}

	panic("Exists method not implemented on underlying queryset")
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Count() (int64, error) {
	w.setup()
	if err := w.beforeReadExec(); err != nil {
		return 0, err
	}

	type readQs interface {
		Count() (int64, error)
	}
	if readQs, ok := w.NullQuerySet.(readQs); ok {
		count, err := readQs.Count()
		if err != nil {
			return 0, err
		}
		return count, w.afterReadExec(count)
	}

	panic("Count method not implemented on underlying queryset")
}

func (w *WrappedQuerySet[T, CONV, ORIG]) First() (*Row[T], error) {
	w.setup()
	if err := w.beforeReadExec(); err != nil {
		return nil, err
	}

	type readQs interface {
		First() (*Row[T], error)
	}

	if readQs, ok := w.NullQuerySet.(readQs); ok {
		res, err := readQs.First()
		if err != nil {
			return nil, err
		}
		return res, w.afterReadExec(res)
	}

	panic("First method not implemented on underlying queryset")
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Last() (*Row[T], error) {
	w.setup()
	if err := w.beforeReadExec(); err != nil {
		return nil, err
	}

	type readQs interface {
		Last() (*Row[T], error)
	}

	if readQs, ok := w.NullQuerySet.(readQs); ok {
		res, err := readQs.Last()
		if err != nil {
			return nil, err
		}
		return res, w.afterReadExec(res)
	}

	panic("Last method not implemented on underlying queryset")

}

func (w *WrappedQuerySet[T, CONV, ORIG]) Get() (*Row[T], error) {
	w.setup()
	if err := w.beforeReadExec(); err != nil {
		return nil, err
	}

	type readQs interface {
		Get() (*Row[T], error)
	}

	if readQs, ok := w.NullQuerySet.(readQs); ok {
		res, err := readQs.Get()
		if err != nil {
			return nil, err
		}
		return res, w.afterReadExec(res)
	}

	panic("Get method not implemented on underlying queryset")
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Values(fields ...any) ([]map[string]any, error) {
	w.setup()
	if err := w.beforeReadExec(); err != nil {
		return nil, err
	}

	type readQs interface {
		Values(fields ...any) ([]map[string]any, error)
	}
	if readQs, ok := w.NullQuerySet.(readQs); ok {
		res, err := readQs.Values(fields...)
		if err != nil {
			return nil, err
		}
		return res, w.afterReadExec(res)
	}

	panic("Values method not implemented on underlying queryset")
}

func (w *WrappedQuerySet[T, CONV, ORIG]) ValuesList(fields ...any) ([][]interface{}, error) {
	w.setup()
	if err := w.beforeReadExec(); err != nil {
		return nil, err
	}
	type readQs interface {
		ValuesList(fields ...any) ([][]interface{}, error)
	}
	if readQs, ok := w.NullQuerySet.(readQs); ok {
		res, err := readQs.ValuesList(fields...)
		if err != nil {
			return nil, err
		}
		return res, w.afterReadExec(res)
	}

	panic("ValuesList method not implemented on underlying queryset")
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Aggregate(annotations map[string]expr.Expression) (map[string]any, error) {
	w.setup()
	if err := w.beforeReadExec(); err != nil {
		return nil, err
	}

	type readQs interface {
		Aggregate(annotations map[string]expr.Expression) (map[string]any, error)
	}

	if readQs, ok := w.NullQuerySet.(readQs); ok {
		res, err := readQs.Aggregate(annotations)
		if err != nil {
			return nil, err
		}
		return res, w.afterReadExec(res)
	}

	panic("Aggregate method not implemented on underlying queryset")
}

// this method is pretty much only used in subquery expressions.
func (w *WrappedQuerySet[T, CONV, ORIG]) QueryAll(fields ...any) CompiledQuery[[][]interface{}] {
	if q, ok := w.NullQuerySet.(interface {
		QueryAll(fields ...any) CompiledQuery[[][]interface{}]
	}); ok {
		return q.QueryAll(fields...)
	}
	panic("QueryAll method not implemented on underlying queryset")
}

// this method is pretty much only used in subquery expressions.
func (w *WrappedQuerySet[T, CONV, ORIG]) QueryAggregate() CompiledQuery[[][]interface{}] {
	if q, ok := w.NullQuerySet.(interface {
		QueryAggregate() CompiledQuery[[][]interface{}]
	}); ok {
		return q.QueryAggregate()
	}
	panic("QueryAggregate method not implemented on underlying queryset")
}

// this method is pretty much only used in subquery expressions.
func (w *WrappedQuerySet[T, CONV, ORIG]) QueryCount() CompiledQuery[int64] {
	if q, ok := w.NullQuerySet.(interface {
		QueryCount() CompiledQuery[int64]
	}); ok {
		return q.QueryCount()
	}
	panic("QueryCount method not implemented on underlying queryset")
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Create(value T) (T, error) {
	if q, ok := w.NullQuerySet.(interface {
		Create(value T) (T, error)
	}); ok {
		return q.Create(value)
	}
	panic("Create method not implemented on underlying queryset")
}
func (w *WrappedQuerySet[T, CONV, ORIG]) Update(value T, expressions ...any) (int64, error) {
	if q, ok := w.NullQuerySet.(interface {
		Update(value T, expressions ...any) (int64, error)
	}); ok {
		return q.Update(value, expressions...)
	}
	panic("Update method not implemented on underlying queryset")
}
func (w *WrappedQuerySet[T, CONV, ORIG]) GetOrCreate(value T) (T, bool, error) {
	if q, ok := w.NullQuerySet.(interface {
		GetOrCreate(value T) (T, bool, error)
	}); ok {
		return q.GetOrCreate(value)
	}
	panic("GetOrCreate method not implemented on underlying queryset")
}
func (w *WrappedQuerySet[T, CONV, ORIG]) BulkCreate(objects []T) ([]T, error) {
	if q, ok := w.NullQuerySet.(interface {
		BulkCreate(objects []T) ([]T, error)
	}); ok {
		return q.BulkCreate(objects)
	}
	panic("BulkCreate method not implemented on underlying queryset")
}
func (w *WrappedQuerySet[T, CONV, ORIG]) BatchCreate(objects []T) ([]T, error) {
	if q, ok := w.NullQuerySet.(interface {
		BatchCreate(objects []T) ([]T, error)
	}); ok {
		return q.BatchCreate(objects)
	}
	panic("BatchCreate method not implemented on underlying queryset")
}
func (w *WrappedQuerySet[T, CONV, ORIG]) BulkUpdate(params ...any) (int64, error) {
	if q, ok := w.NullQuerySet.(interface {
		BulkUpdate(params ...any) (int64, error)
	}); ok {
		return q.BulkUpdate(params...)
	}
	panic("BulkUpdate method not implemented on underlying queryset")
}
func (w *WrappedQuerySet[T, CONV, ORIG]) BatchUpdate(params ...any) (int64, error) {
	if q, ok := w.NullQuerySet.(interface {
		BatchUpdate(params ...any) (int64, error)
	}); ok {
		return q.BatchUpdate(params...)
	}
	panic("BatchUpdate method not implemented on underlying queryset")
}
func (w *WrappedQuerySet[T, CONV, ORIG]) Delete(objects ...T) (int64, error) {
	if q, ok := w.NullQuerySet.(interface {
		Delete(objects ...T) (int64, error)
	}); ok {
		return q.Delete(objects...)
	}
	panic("Delete method not implemented on underlying queryset")
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Row(sqlStr string, args ...interface{}) drivers.SQLRow {
	if q, ok := w.NullQuerySet.(interface {
		Row(sqlStr string, args ...interface{}) drivers.SQLRow
	}); ok {
		return q.Row(sqlStr, args...)
	}
	panic("Row method not implemented on underlying queryset")
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Rows(sqlStr string, args ...interface{}) (drivers.SQLRows, error) {
	if q, ok := w.NullQuerySet.(interface {
		Rows(sqlStr string, args ...interface{}) (drivers.SQLRows, error)
	}); ok {
		return q.Rows(sqlStr, args...)
	}
	panic("Rows method not implemented on underlying queryset")
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Exec(sqlStr string, args ...interface{}) (sql.Result, error) {
	if q, ok := w.NullQuerySet.(interface {
		Exec(sqlStr string, args ...interface{}) (sql.Result, error)
	}); ok {
		return q.Exec(sqlStr, args...)
	}
	panic("Exec method not implemented on underlying queryset")
}
