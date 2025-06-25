package postgres

import (
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
)

const (
	int16_max = 1 << 15
	int32_max = 1 << 31
)

// POSTGRES TYPES
func init() {
	// register kinds
	migrator.RegisterColumnKind(&drivers.DriverPostgres{}, []reflect.Kind{reflect.String}, Type__string)
	migrator.RegisterColumnKind(&drivers.DriverPostgres{}, []reflect.Kind{reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64}, Type__int)
	migrator.RegisterColumnKind(&drivers.DriverPostgres{}, []reflect.Kind{reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64}, Type__int)
	migrator.RegisterColumnKind(&drivers.DriverPostgres{}, []reflect.Kind{reflect.Float32, reflect.Float64}, Type__float)
	migrator.RegisterColumnKind(&drivers.DriverPostgres{}, []reflect.Kind{reflect.Bool}, Type__bool)
	migrator.RegisterColumnKind(&drivers.DriverPostgres{}, []reflect.Kind{reflect.Array, reflect.Slice, reflect.Map}, Type__string)

	// register types
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, drivers.Text(""), Type__string)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, drivers.Char(""), Type__char)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, drivers.String(""), Type__string)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, drivers.Int(0), Type__int)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, drivers.Bytes(nil), Type__blob)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, drivers.Bool(false), Type__bool)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, drivers.Float(0.0), Type__float)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, drivers.Timestamp{}, Type__datetime)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, drivers.LocalTime{}, Type__localtime)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, drivers.DateTime{}, Type__datetime)

	migrator.RegisterColumnType(&drivers.DriverPostgres{}, (*contenttypes.ContentType)(nil), Type__string)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, contenttypes.BaseContentType[attrs.Definer]{}, Type__string)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, sql.NullString{}, Type__string)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, sql.NullFloat64{}, Type__int)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, sql.NullInt64{}, Type__int)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, sql.NullInt32{}, Type__int)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, sql.NullInt16{}, Type__int)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, sql.NullBool{}, Type__bool)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, sql.NullByte{}, Type__int)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, sql.NullTime{}, Type__datetime)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, time.Time{}, Type__datetime)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, []byte{}, Type__string)
}

func Type__string(c *migrator.Column) string {
	var max int64 = c.MaxLength

	if c.FieldType() == reflect.TypeOf(drivers.Text("")) {
		// If the field is of type drivers.Text, we use TEXT type
		return "TEXT"
	}

	if c.FieldType() == reflect.TypeOf(drivers.String("")) && (max == 0 || max <= 255) {
		if max > 0 && max <= 255 {
			return fmt.Sprintf("VARCHAR(%d)", max)
		}
		return "VARCHAR(255)"
	}

	if max == 0 {
		return "TEXT"
	}

	var sb = new(strings.Builder)
	sb.WriteString("VARCHAR(")
	sb.WriteString(strconv.FormatInt(max, 10))
	sb.WriteString(")")
	return sb.String()
}

func Type__char(c *migrator.Column) string {
	if c.FieldType() == reflect.TypeOf(drivers.Char("")) {
		// If the field is of type drivers.Char, we use CHAR type
		return "CHAR"
	}

	var max int64 = c.MaxLength
	if max <= 0 {
		max = 1 // Default to CHAR(1) if no length is specified
	}

	return fmt.Sprintf("CHAR(%d)", max)
}

func Type__blob(c *migrator.Column) string {
	if c.FieldType() == reflect.TypeOf(drivers.Bytes(nil)) || c.FieldType() == reflect.TypeOf([]byte{}) {
		return "BYTEA"
	}

	// For other types, we can use TEXT as a fallback
	return "TEXT"
}

func Type__float(c *migrator.Column) string {
	switch c.FieldType().Kind() {
	case reflect.Float32:
		return "REAL"
	case reflect.Float64:
		return "DOUBLE PRECISION"
	}
	return "DOUBLE PRECISION"
}

func Type__int(c *migrator.Column) string {
	if c.Primary && c.Auto {
		// If the column is a primary key and auto-incrementing, use SERIAL
		return "BIGSERIAL"
	}
	var max float64 = c.MaxValue
	switch c.FieldType().Kind() {
	case reflect.Int8:
		return "SMALLINT"
	case reflect.Int16:
		return "INTEGER"
	case reflect.Int32, reflect.Int:
		if max != 0 && max <= int32_max {
			return "INTEGER"
		}
		return "BIGINT"
	case reflect.Int64:
		if max != 0 && max <= int32_max {
			return "INTEGER"
		}
		return "BIGINT"
	}

	return "BIGINT"
}

func Type__bool(c *migrator.Column) string {
	return "BOOLEAN"
}

func Type__timestamp(c *migrator.Column) string {
	return "TIMESTAMP"
}

func Type__localtime(c *migrator.Column) string {
	return "TIMESTAMPTZ"
}

func Type__datetime(c *migrator.Column) string {
	return "TIMESTAMP"
}
