package queries

import (
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
)

var (
	_ CompiledQuery[int64] = &QueryObject[int64]{}
	// _ CompiledQuery[[][]interface{}] = (*CombinedQuery[[]interface{}])(nil)

	LogQueries = true
)

type QueryInformation struct {
	Object  attrs.Definer
	Params  []any
	Stmt    string
	Builder QueryCompiler
}

func (q *QueryInformation) SQL() string {
	return q.Stmt
}

func (q *QueryInformation) Args() []any {
	return q.Params
}

func (q *QueryInformation) Model() attrs.Definer {
	return q.Object
}

func (q *QueryInformation) Compiler() QueryCompiler {
	return q.Builder
}

type QueryObject[T1 any] struct {
	QueryInformation
	Execute func(sql string, args ...any) (T1, error)
}

func (q *QueryObject[T1]) Exec() (T1, error) {
	var result, err = q.Execute(q.Stmt, q.Params...)
	if LogQueries {
		if err != nil {
			logger.Errorf("Query (%T, %T): %s: %s %v", q.Model(), *new(T1), err.Error(), q.Stmt, q.Params)
			return result, err
		}
		logger.Debugf("Query (%T, %T): %s %v", q.Model(), *new(T1), q.Stmt, q.Params)
	}
	return result, err
}
