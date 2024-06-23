package auditlogs

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/Nigel2392/django/core/contenttypes"
	"github.com/Nigel2392/django/core/logger"
)

type AuditLogFilter interface {
	Is(string) bool
	Name() string
	Value() []interface{}
}

const (
	AuditLogFilterID           = "id"
	AuditLogFilterType         = "type"
	AuditLogFilterLevel_EQ     = "level_eq"
	AuditLogFilterLevel_GT     = "level_gt"
	AuditLogFilterLevel_LT     = "level_lt"
	AuditLogFilterTimestamp_EQ = "timestamp_eq"
	AuditLogFilterTimestamp_GT = "timestamp_gt"
	AuditLogFilterTimestamp_LT = "timestamp_lt"
	AuditLogFilterUserID       = "user_id"
	AuditLogFilterObjectID     = "object_id"
	AuditLogFilterContentType  = "content_type"
	AuditLogFilterData         = "data"
)

func NewAuditLogFilter(name string, value ...interface{}) AuditLogFilter {
	return &auditlogFilter{
		name:  name,
		value: value,
	}
}

type auditlogFilter struct {
	name  string
	value []interface{}
}

func (f *auditlogFilter) Is(name string) bool {
	return f.name == name
}

func (f *auditlogFilter) Name() string {
	return f.name
}

func (f *auditlogFilter) Value() []interface{} {
	return f.value
}

func FilterType(values ...string) AuditLogFilter {
	var v = make([]interface{}, len(values))
	for i, value := range values {
		v[i] = value
	}
	return &auditlogFilter{
		name:  AuditLogFilterType,
		value: v,
	}
}

func FilterLevelEqual(values ...logger.LogLevel) AuditLogFilter {
	var v = make([]interface{}, len(values))
	for i, value := range values {
		v[i] = value
	}
	return &auditlogFilter{
		name:  AuditLogFilterLevel_EQ,
		value: v,
	}
}

func FilterLevelGreaterThan(value logger.LogLevel) AuditLogFilter {
	return &auditlogFilter{
		name:  AuditLogFilterLevel_GT,
		value: []interface{}{value},
	}
}

func FilterLevelLessThan(value logger.LogLevel) AuditLogFilter {
	return &auditlogFilter{
		name:  AuditLogFilterLevel_LT,
		value: []interface{}{value},
	}
}

func FilterTimestampEqual(values ...time.Time) AuditLogFilter {
	var v = make([]interface{}, len(values))
	for i, value := range values {
		v[i] = value
	}
	return &auditlogFilter{
		name:  AuditLogFilterTimestamp_EQ,
		value: v,
	}
}

func FilterUserID(values ...interface{}) AuditLogFilter {
	return &auditlogFilter{
		name:  AuditLogFilterUserID,
		value: values,
	}
}

func FilterObjectID(values ...interface{}) AuditLogFilter {
	return &auditlogFilter{
		name:  AuditLogFilterObjectID,
		value: values,
	}
}

func FilterContentType(values ...contenttypes.ContentType) AuditLogFilter {
	var v = make([]interface{}, len(values))
	for i, value := range values {
		v[i] = value.TypeName()
	}
	return &auditlogFilter{
		name:  AuditLogFilterContentType,
		value: v,
	}
}

func FilterData(values ...interface{}) AuditLogFilter {
	var v = make([]interface{}, len(values))
	for i, value := range values {
		var b = new(bytes.Buffer)
		if err := json.NewEncoder(b).Encode(value); err != nil {
			return nil
		}
		v[i] = b.Bytes()
	}
	return &auditlogFilter{
		name:  AuditLogFilterData,
		value: values,
	}
}
