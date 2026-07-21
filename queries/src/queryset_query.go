package queries

import (
	"iter"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

var (
	_ CompiledQuery[int64]     = &QueryObject[int64]{}
	_ CompiledRowQuery[int64]  = &QueryRowObject[int64]{}
	_ CompiledRowsQuery[int64] = &QueryRowsObject[int64]{}
	// _ CompiledQuery[[][]interface{}] = (*CombinedQuery[[]interface{}])(nil)
)

type Query struct {
	Object  attrs.Definer
	Params  []any
	Stmt    string
	Builder QueryCompiler
}

func (q *Query) SQL() string {
	return q.Stmt
}

func (q *Query) Args() []any {
	return q.Params
}

func (q *Query) Model() attrs.Definer {
	return q.Object
}

func (q *Query) Compiler() QueryCompiler {
	return q.Builder
}

func ErrorQueryObject[T1 any](object attrs.Definer, builder QueryCompiler, possibleError error) *QueryObject[T1] {
	return &QueryObject[T1]{
		QueryInfo: &Query{
			Object:  object,
			Builder: builder,
		},
		Execute: func(sql string, args ...any) (T1, error) {
			return *new(T1), possibleError
		},
	}
}

type QueryObject[T1 any] struct {
	QueryInfo
	Execute func(sql string, args ...any) (T1, error)
}

func (q *QueryObject[T1]) Exec() (T1, error) {
	return q.Execute(q.SQL(), q.Args()...)
}

type SQLQueryObject[RESULT any, SQL any] struct {
	QueryInfo
	Error        error
	ExecSQL      func(sql string, args ...any) (SQL, error)
	Execute      func(SQL) (RESULT, error)
	RawResultSQL *SQL
}

func (q *SQLQueryObject[RESULT, SQL]) setupExec() error {
	if q.Error != nil {
		return q.Error
	}

	if q.RawResultSQL != nil || q.ExecSQL == nil {
		return nil
	}

	sql, err := q.ExecSQL(q.SQL(), q.Args()...)
	if err != nil {
		return err
	}
	q.RawResultSQL = &sql
	return nil
}

func (q *SQLQueryObject[RESULT, SQL]) Reset() {
	q.Error = nil
	q.RawResultSQL = nil
}

func (q *SQLQueryObject[RESULT, SQL]) Exec() (result RESULT, err error) {
	if err := q.setupExec(); err != nil {
		return result, err
	}
	return q.Execute(*q.RawResultSQL)
}

/*
ROW
*/
type QueryRowObject[RESULT any] SQLQueryObject[RESULT, drivers.SQLRow]

func (q *QueryRowObject[RESULT]) Reset() {
	(*SQLQueryObject[RESULT, drivers.SQLRow])(q).Reset()
}

func (q *QueryRowObject[RESULT]) Err() (err error) {
	if err := (*SQLQueryObject[RESULT, drivers.SQLRow])(q).setupExec(); err != nil {
		return err
	}
	return (*q.RawResultSQL).Err()
}

func (q *QueryRowObject[RESULT]) Scan(dest ...any) error {
	if err := (*SQLQueryObject[RESULT, drivers.SQLRow])(q).setupExec(); err != nil {
		return err
	}
	return (*q.RawResultSQL).Scan(dest...)
}

func (q *QueryRowObject[RESULT]) Exec() (result RESULT, err error) {
	return (*SQLQueryObject[RESULT, drivers.SQLRow])(q).Exec()
}

/*
ROWS
*/
type QueryRowsObject[RESULT any] SQLQueryObject[RESULT, drivers.SQLRows]

func (q *QueryRowsObject[RESULT]) Reset() {
	(*SQLQueryObject[RESULT, drivers.SQLRows])(q).Reset()
}

func (q *QueryRowsObject[RESULT]) Err() (err error) {
	if err := (*SQLQueryObject[RESULT, drivers.SQLRows])(q).setupExec(); err != nil {
		return err
	}
	return (*q.RawResultSQL).Err()
}

func (q *QueryRowsObject[RESULT]) Scan(dest ...any) error {
	if err := (*SQLQueryObject[RESULT, drivers.SQLRows])(q).setupExec(); err != nil {
		return err
	}
	return (*q.RawResultSQL).Scan(dest...)
}

func (q *QueryRowsObject[RESULT]) Next() bool {
	if err := (*SQLQueryObject[RESULT, drivers.SQLRows])(q).setupExec(); err != nil {
		q.Error = err
		return false
	}
	return (*q.RawResultSQL).Next()
}

func (q *QueryRowsObject[RESULT]) Close() error {
	if err := (*SQLQueryObject[RESULT, drivers.SQLRows])(q).setupExec(); err != nil {
		return err
	}
	return (*q.RawResultSQL).Close()
}
func (q *QueryRowsObject[RESULT]) Columns() ([]string, error) {
	if err := (*SQLQueryObject[RESULT, drivers.SQLRows])(q).setupExec(); err != nil {
		return nil, err
	}
	return (*q.RawResultSQL).Columns()
}
func (q *QueryRowsObject[RESULT]) NextResultSet() bool {
	if err := (*SQLQueryObject[RESULT, drivers.SQLRows])(q).setupExec(); err != nil {
		q.Error = err
		return false
	}
	return (*q.RawResultSQL).NextResultSet()
}

func (q *QueryRowsObject[RESULT]) Exec() (result RESULT, err error) {
	return (*SQLQueryObject[RESULT, drivers.SQLRows])(q).Exec()
}

type QueryIterRowsObject[RESULT any] struct {
	QueryRowsObject[[]RESULT]
	IterExecute func(drivers.SQLRows) iter.Seq2[RESULT, error]
}

func (q *QueryIterRowsObject[RESULT]) Exec() (result []RESULT, err error) {
	var s = make([]RESULT, 0)
	for row, err := range q.Iter() {
		if err != nil {
			return nil, err
		}
		s = append(s, row)
	}
	return s, nil
}

func (q *QueryIterRowsObject[RESULT]) Iter() iter.Seq2[RESULT, error] {

	assert.Assert(
		q.IterExecute != nil,
		"an iterator needs to be provided to %T", q,
	)

	if err := (*SQLQueryObject[[]RESULT, drivers.SQLRows])(&q.QueryRowsObject).setupExec(); err != nil {
		return func(yield func(RESULT, error) bool) {
			var zero RESULT
			yield(zero, err)
		}
	}

	return q.IterExecute(*q.RawResultSQL)
}
