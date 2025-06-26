package queries

import (
	"github.com/Nigel2392/go-django/src/core/attrs"
)

var (
	_ CompiledQuery[int64] = &QueryObject[int64]{}
	// _ CompiledQuery[[][]interface{}] = (*CombinedQuery[[]interface{}])(nil)
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
	return q.Execute(q.Stmt, q.Params...)
}
