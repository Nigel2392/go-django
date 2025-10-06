package drivers

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
)

var queryContextKey = dbContextKey{"db.query.context"}

type QueryFlag int

func (f QueryFlag) String() string {
	var n, ok = flagNames[f]
	if !ok {
		var parts = make([]string, 0)
		for k, v := range flagNames {
			if f&k != 0 {
				parts = append(parts, v)
			}
		}
		if len(parts) == 0 {
			return "UNKNOWN"
		}
		slices.Sort(parts)
		n = strings.Join(parts, "|")
	}
	return n
}

const (
	Q_UNKNOWN QueryFlag = 0
	Q_QUERY   QueryFlag = 1 << iota
	Q_QUERYROW
	Q_EXEC
	Q_PING
	Q_TSTART
	Q_TCOMMIT
	Q_TROLLBACK
	Q_MULTIPLE
)

var flagNames = map[QueryFlag]string{
	Q_UNKNOWN:   "UNKNOWN",
	Q_QUERY:     "QUERY",
	Q_QUERYROW:  "QUERYROW",
	Q_EXEC:      "EXEC",
	Q_PING:      "PING",
	Q_TSTART:    "TRANSACTION_START",
	Q_TCOMMIT:   "TRANSACTION_COMMIT",
	Q_TROLLBACK: "TRANSACTION_ROLLBACK",
	Q_MULTIPLE:  "MULTIPLE",
}

type Query struct {
	Context   *QueryInformation
	Driver    string
	Flags     QueryFlag
	Query     string
	Args      []any
	Error     error
	Start     time.Time
	TimeTaken time.Duration
}

func (q *Query) Explain(c context.Context, db DB) (string, error) {
	var d, ok = drivers.byName[q.Driver]
	if !ok {
		panic(fmt.Errorf("unknown driver %q", q.Driver))
	}
	if d.ExplainQuery == nil {
		return "", errors.NotImplemented.Wrapf(
			"explain not implemented for driver %q", d.Name,
		)
	}
	return d.ExplainQuery(c, db, q.Query, q.Args)
}

type QueryInformation struct {
	Queries []*Query
	Start   time.Time
}

func ContextWithQueryInfo(ctx context.Context) (context.Context, *QueryInformation) {
	var qi = &QueryInformation{
		Queries: make([]*Query, 0),
		Start:   time.Now(),
	}
	return context.WithValue(ctx, queryContextKey, qi), qi
}

func ContextQueryInfo(ctx context.Context) (*QueryInformation, bool) {
	var v = ctx.Value(queryContextKey)
	if v == nil {
		return nil, false
	}
	return v.(*QueryInformation), true
}

func ContextQueryExec[T any](ctx context.Context, driver string, query string, args []any, flags QueryFlag, fn func(ctx context.Context, query string, args ...any) (T, error)) (T, error) {
	var qi, ok = ContextQueryInfo(ctx)
	if !ok {
		return fn(ctx, query, args...)
	}

	var start = time.Now()
	var result, err = fn(ctx, query, args...)
	var ql = &Query{
		Context:   qi,
		Driver:    driver,
		Query:     query,
		Args:      args,
		Error:     err,
		Start:     start,
		TimeTaken: time.Since(start),
		Flags:     flags,
	}
	qi.Queries = append(qi.Queries, ql)
	return result, err
}

func (q *QueryInformation) TotalExecutionTime() time.Duration {
	var s = q.Start
	if len(q.Queries) == 0 {
		return 0
	}
	var latest = q.Queries[len(q.Queries)-1]
	return latest.Start.Add(latest.TimeTaken).Sub(s)
}

func (q *QueryInformation) TotalTime() time.Duration {
	var total time.Duration
	for _, ql := range q.Queries {
		total += ql.TimeTaken
	}
	return total
}

func (q *QueryInformation) Slowest() *Query {
	if len(q.Queries) == 0 {
		return nil
	}
	var maxIdx = 0
	for i, ql := range q.Queries {
		if i == 0 {
			continue
		}

		if ql.TimeTaken > q.Queries[maxIdx].TimeTaken {
			maxIdx = i
		}
	}
	return q.Queries[maxIdx]
}

func (q *QueryInformation) AverageTime() time.Duration {
	if len(q.Queries) == 0 {
		return 0
	}
	return q.TotalTime() / time.Duration(len(q.Queries))
}
