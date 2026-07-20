package drivers

import (
	"slices"
	"strings"

	"github.com/Nigel2392/go-django/internal/bitch"
)

type QueryFlag bitch.Flag

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
