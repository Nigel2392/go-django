package queries

import (
	"context"

	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

var _ BaseQuerySet[attrs.Definer, *QuerySet[attrs.Definer]] = (*WrappedQuerySet[attrs.Definer, *GenericQuerySet, *QuerySet[attrs.Definer]])(nil)

type WrappedQuerySet[T attrs.Definer, CONV BaseQuerySet[T, CONV], ORIG BaseQuerySet[T, ORIG]] struct {
	BaseQuerySet[T, ORIG]
	original ORIG
	embedder CONV
}

func WrapQuerySet[T attrs.Definer, CONV BaseQuerySet[T, CONV], ORIG BaseQuerySet[T, ORIG]](qs ORIG, embedder CONV) *WrappedQuerySet[T, CONV, ORIG] {
	return &WrappedQuerySet[T, CONV, ORIG]{
		BaseQuerySet: qs,
		original:     qs.Clone(),
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
)

func (w *WrappedQuerySet[T, CONV, ORIG]) Reset() CONV {
	w.BaseQuerySet = w.original.Clone().WithContext(
		w.BaseQuerySet.Context(),
	)
	return w.embedder
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
	w.BaseQuerySet = w.BaseQuerySet.WithContext(ctx)
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) BuildExpression() expr.Expression {
	w.setup()
	return w.BaseQuerySet.BuildExpression()
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Clone() CONV {
	w.BaseQuerySet = w.BaseQuerySet.Clone()
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Distinct() CONV {
	w.BaseQuerySet = w.BaseQuerySet.Distinct()
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Select(fields ...any) CONV {
	w.setup()
	w.BaseQuerySet = w.BaseQuerySet.Select(fields...)
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Preload(fields ...any) CONV {
	w.setup()
	w.BaseQuerySet = w.BaseQuerySet.Preload(fields...)
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Filter(key interface{}, vals ...interface{}) CONV {
	w.setup()
	w.BaseQuerySet = w.BaseQuerySet.Filter(key, vals...)
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) GroupBy(fields ...any) CONV {
	w.setup()
	w.BaseQuerySet = w.BaseQuerySet.GroupBy(fields...)
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Limit(n int) CONV {
	w.BaseQuerySet = w.BaseQuerySet.Limit(n)
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Offset(n int) CONV {
	w.BaseQuerySet = w.BaseQuerySet.Offset(n)
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) OrderBy(fields ...string) CONV {
	w.setup()
	w.BaseQuerySet = w.BaseQuerySet.OrderBy(fields...)
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Reverse() CONV {
	w.BaseQuerySet = w.BaseQuerySet.Reverse()
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) ExplicitSave() CONV {
	w.BaseQuerySet = w.BaseQuerySet.ExplicitSave()
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Annotate(aliasOrAliasMap interface{}, exprs ...expr.Expression) CONV {
	w.setup()
	w.BaseQuerySet = w.BaseQuerySet.Annotate(aliasOrAliasMap, exprs...)
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) ForUpdate() CONV {
	w.BaseQuerySet = w.BaseQuerySet.ForUpdate()
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Prefix(prefix string) CONV {
	w.setup()
	w.BaseQuerySet = w.BaseQuerySet.Prefix(prefix)
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Having(key interface{}, vals ...interface{}) CONV {
	w.setup()
	w.BaseQuerySet = w.BaseQuerySet.Having(key, vals...)
	return w.embedder
}

func (w *WrappedQuerySet[T, CONV, ORIG]) All() (Rows[T], error) {
	w.setup()

	if err := w.beforeReadExec(); err != nil {
		return nil, err
	}

	res, err := w.BaseQuerySet.All()
	if err != nil {
		return nil, err
	}

	err = w.afterReadExec(res)
	return res, err
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Exists() (bool, error) {
	w.setup()
	if err := w.beforeReadExec(); err != nil {
		return false, err
	}

	res, err := w.BaseQuerySet.Exists()
	if err != nil {
		return false, err
	}

	err = w.afterReadExec(res)
	return res, err

}

func (w *WrappedQuerySet[T, CONV, ORIG]) Count() (int64, error) {
	w.setup()
	if err := w.beforeReadExec(); err != nil {
		return 0, err
	}

	res, err := w.BaseQuerySet.Count()
	if err != nil {
		return 0, err
	}

	err = w.afterReadExec(res)
	return res, err

}

func (w *WrappedQuerySet[T, CONV, ORIG]) First() (*Row[T], error) {
	w.setup()
	if err := w.beforeReadExec(); err != nil {
		return nil, err
	}

	res, err := w.BaseQuerySet.First()
	if err != nil {
		return nil, err
	}

	err = w.afterReadExec(res)
	return res, err

}

func (w *WrappedQuerySet[T, CONV, ORIG]) Last() (*Row[T], error) {
	w.setup()
	if err := w.beforeReadExec(); err != nil {
		return nil, err
	}

	res, err := w.BaseQuerySet.Last()
	if err != nil {
		return nil, err
	}

	err = w.afterReadExec(res)
	return res, err

}

func (w *WrappedQuerySet[T, CONV, ORIG]) Get() (*Row[T], error) {
	w.setup()
	if err := w.beforeReadExec(); err != nil {
		return nil, err
	}

	res, err := w.BaseQuerySet.Get()
	if err != nil {
		return nil, err
	}

	err = w.afterReadExec(res)
	return res, err

}

func (w *WrappedQuerySet[T, CONV, ORIG]) Values(fields ...any) ([]map[string]any, error) {
	w.setup()
	if err := w.beforeReadExec(); err != nil {
		return nil, err
	}
	res, err := w.BaseQuerySet.Values(fields...)
	if err != nil {
		return nil, err
	}
	err = w.afterReadExec(res)
	return res, err
}

func (w *WrappedQuerySet[T, CONV, ORIG]) ValuesList(fields ...any) ([][]interface{}, error) {
	w.setup()
	if err := w.beforeReadExec(); err != nil {
		return nil, err
	}
	res, err := w.BaseQuerySet.ValuesList(fields...)
	if err != nil {
		return nil, err
	}
	err = w.afterReadExec(res)
	return res, err
}

func (w *WrappedQuerySet[T, CONV, ORIG]) Aggregate(annotations map[string]expr.Expression) (map[string]any, error) {
	w.setup()
	if err := w.beforeReadExec(); err != nil {
		return nil, err
	}
	res, err := w.BaseQuerySet.Aggregate(annotations)
	if err != nil {
		return nil, err
	}
	err = w.afterReadExec(res)
	return res, err
}

// this method is pretty much only used in subquery expressions.
func (w *WrappedQuerySet[T, CONV, ORIG]) QueryAll(fields ...any) CompiledQuery[[][]interface{}] {
	return w.BaseQuerySet.QueryAll(fields...)
}

// this method is pretty much only used in subquery expressions.
func (w *WrappedQuerySet[T, CONV, ORIG]) QueryAggregate() CompiledQuery[[][]interface{}] {
	return w.BaseQuerySet.QueryAggregate()
}

// this method is pretty much only used in subquery expressions.
func (w *WrappedQuerySet[T, CONV, ORIG]) QueryCount() CompiledQuery[int64] {
	return w.BaseQuerySet.QueryCount()
}
