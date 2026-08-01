package auditlogs

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/logger"
)

func FilterType(values ...string) expr.Expression {
	var v = make([]interface{}, len(values))
	for i, value := range values {
		v[i] = value
	}
	if len(v) > 1 {
		return expr.Q("Type__in", v)
	}
	return expr.Q("Type", v[0])
}

func FilterLevel(values ...logger.LogLevel) expr.Expression {
	var v = make([]interface{}, len(values))
	for i, value := range values {
		v[i] = value
	}
	if len(v) > 1 {
		return expr.Q("Level__in", v)
	}
	return expr.Q("Level", v[0])
}

func FilterLevelGT(value logger.LogLevel) expr.Expression {
	return expr.Q("Level__gt", value)
}

func FilterLevelLT(value logger.LogLevel) expr.Expression {
	return expr.Q("Level__lt", value)
}

func FilterTimestamp(values ...time.Time) expr.Expression {
	var v = make([]interface{}, len(values))
	for i, value := range values {
		v[i] = value
	}
	if len(v) > 1 {
		return expr.Q("Timestamp__in", v)
	}
	return expr.Q("Timestamp", v[0])
}

func FilterUserID(values ...interface{}) expr.Expression {
	if len(values) > 1 {
		return expr.Q("UserID__in", values)
	}
	return expr.Q("UserID", values[0])
}

func FilterObjectID(values ...interface{}) expr.Expression {
	if len(values) > 1 {
		return expr.Q("ObjectID__in", values)
	}
	return expr.Q("ObjectID", values[0])
}

func FilterContentType(values ...contenttypes.ContentType) expr.Expression {
	var v = make([]interface{}, len(values))
	for i, value := range values {
		v[i] = value.TypeName()
	}
	if len(v) > 1 {
		return expr.Q("ContentType__in", v)
	}
	return expr.Q("ContentType", v[0])
}

func FilterData(values ...interface{}) expr.Expression {
	var v = make([]interface{}, len(values))
	for i, value := range values {
		var b = new(bytes.Buffer)
		if err := json.NewEncoder(b).Encode(value); err != nil {
			return nil
		}
		v[i] = b.Bytes()
	}
	if len(v) > 1 {
		return expr.Q("Data__in", v)
	}
	return expr.Q("Data", v[0])
}
