package drivers

import (
	"context"
	"fmt"
	"time"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
)

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
