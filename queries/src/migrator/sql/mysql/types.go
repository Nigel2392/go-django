package mysql

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
)

const (
	int16_max = 1 << 15
	int32_max = 1 << 31
)

func registerType(t drivers.Type, fn func(c *migrator.Column) string) {
	migrator.RegisterColumnType(&drivers.DriverMySQL{}, t, fn)
	migrator.RegisterColumnType(&drivers.DriverMariaDB{}, t, fn)
}

// MYSQL TYPES
func init() {
	registerType(drivers.TypeText, Type__string)
	registerType(drivers.TypeChar, Type__char)
	registerType(drivers.TypeString, Type__string)
	registerType(drivers.TypeInt, Type__int)
	registerType(drivers.TypeUint, Type__int)
	registerType(drivers.TypeBytes, Type__blob)
	registerType(drivers.TypeBool, Type__bool)
	registerType(drivers.TypeFloat, Type__float)
	registerType(drivers.TypeUUID, Type__string)
	registerType(drivers.TypeBLOB, Type__blob)
	registerType(drivers.TypeJSON, Type__string)
	registerType(drivers.TypeTimestamp, Type__timestamp)
	registerType(drivers.TypeLocalTime, Type__timestamp)
	registerType(drivers.TypeDateTime, Type__datetime)
}

func Type__string(c *migrator.Column) string {
	var max int64 = c.MaxLength

	var dbType = c.DBType()
	if dbType == drivers.TypeText {
		// If the field is of type drivers.Text, we use TEXT type
		return "TEXT"
	}

	if (dbType == drivers.TypeString) && (max > 0 && max <= 255) || c.FieldType() == reflect.TypeOf(drivers.String("")) {
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
