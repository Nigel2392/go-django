package mysql

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

func registerKind(kinds []reflect.Kind, fn func(c *migrator.Column) string) {
	migrator.RegisterColumnKind(&drivers.DriverMySQL{}, kinds, fn)
	migrator.RegisterColumnKind(&drivers.DriverMariaDB{}, kinds, fn)
}

func registerType(t any, fn func(c *migrator.Column) string) {
	migrator.RegisterColumnType(&drivers.DriverMySQL{}, t, fn)
	migrator.RegisterColumnType(&drivers.DriverMariaDB{}, t, fn)
}

// MYSQL TYPES
func init() {
	// register kinds
	registerKind([]reflect.Kind{reflect.String}, Type__string)
	registerKind([]reflect.Kind{reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64}, Type__int)
	registerKind([]reflect.Kind{reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64}, Type__int)
	registerKind([]reflect.Kind{reflect.Float32, reflect.Float64}, Type__float)
	registerKind([]reflect.Kind{reflect.Bool}, Type__bool)
	registerKind([]reflect.Kind{reflect.Array, reflect.Slice, reflect.Map}, Type__string) // MySQL does not have a native array type, so we use string for JSON

	// register types
	registerType(drivers.Text(""), Type__string)
	registerType(drivers.Char(""), Type__char)
	registerType(drivers.String(""), Type__string)
	registerType(drivers.Int(0), Type__int)
	registerType(drivers.Bytes(nil), Type__blob)
	registerType(drivers.Bool(false), Type__bool)
	registerType(drivers.Float(0.0), Type__float)
	registerType(drivers.Timestamp{}, Type__timestamp)
	registerType(drivers.LocalTime{}, Type__timestamp)
	registerType(drivers.DateTime{}, Type__datetime)

	registerType((*contenttypes.ContentType)(nil), Type__string)
	registerType(contenttypes.BaseContentType[attrs.Definer]{}, Type__string)
	registerType(sql.NullString{}, Type__string)
	registerType(sql.NullFloat64{}, Type__int)
	registerType(sql.NullInt64{}, Type__int)
	registerType(sql.NullInt32{}, Type__int)
	registerType(sql.NullInt16{}, Type__int)
	registerType(sql.NullBool{}, Type__bool)
	registerType(sql.NullByte{}, Type__int)
	registerType(sql.NullTime{}, Type__datetime)
	registerType(time.Time{}, Type__datetime)
	registerType([]byte{}, Type__string)
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
	var max int64 = c.MaxLength

	if max == 0 {
		return "CHAR(1)"
	}

	if max < 1 || max > 255 {
		max = 255
	}

	var sb = new(strings.Builder)
	sb.WriteString("CHAR(")
	sb.WriteString(strconv.FormatInt(max, 10))
	sb.WriteString(")")
	return sb.String()
}

func Type__blob(c *migrator.Column) string {
	var max int64 = c.MaxLength
	if max == 0 {
		return "BLOB"
	}

	var sb = new(strings.Builder)
	sb.WriteString("VARBINARY(")
	sb.WriteString(strconv.FormatInt(max, 10))
	sb.WriteString(")")
	return sb.String()
}

func Type__float(c *migrator.Column) string {
	switch c.FieldType().Kind() {
	case reflect.Float32:
		return "FLOAT"
	case reflect.Float64:
		return "DOUBLE"
	}
	return "DOUBLE"
}

func Type__int(c *migrator.Column) string {
	var max float64 = c.MaxValue
	switch c.FieldType().Kind() {
	case reflect.Int8:
		return "SMALLINT"
	case reflect.Int16:
		return "INT"
	case reflect.Int32, reflect.Int:
		if max != 0 && max <= int32_max {
			return "INT"
		}
		return "BIGINT"
	case reflect.Int64:
		if max != 0 && max <= int32_max {
			return "INT"
		}
		return "BIGINT"
	}

	return "BIGINT"
}

func Type__bool(c *migrator.Column) string {
	return "BOOLEAN"
}

func Type__timestamp(c *migrator.Column) string {
	if c.MaxLength > 0 {
		if c.MaxLength <= 6 {
			return fmt.Sprintf("TIMESTAMP(%d)", c.MaxLength)
		}
		return "TIMESTAMP(6)"
	}

	return "TIMESTAMP"
}

func Type__datetime(c *migrator.Column) string {
	return "DATETIME"
}
