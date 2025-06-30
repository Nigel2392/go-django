package postgres

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

// POSTGRES TYPES
func init() {
	// register types
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, drivers.TypeText, Type__string)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, drivers.TypeChar, Type__char)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, drivers.TypeString, Type__string)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, drivers.TypeInt, Type__int)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, drivers.TypeUint, Type__int)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, drivers.TypeBytes, Type__blob)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, drivers.TypeBool, Type__bool)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, drivers.TypeFloat, Type__float)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, drivers.TypeDecimal, Type__decimal)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, drivers.TypeUUID, Type__uuid)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, drivers.TypeBLOB, Type__blob)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, drivers.TypeJSON, Type__string)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, drivers.TypeTimestamp, Type__datetime)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, drivers.TypeLocalTime, Type__localtime)
	migrator.RegisterColumnType(&drivers.DriverPostgres{}, drivers.TypeDateTime, Type__datetime)
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

func Type__decimal(c *migrator.Column) string {
	var precision, scale int64 = c.Precision, c.Scale
	if precision <= 0 {
		precision = 10
	}

	if scale == 0 {
		scale = 5
	}

	if scale < 0 || scale > precision {
		scale = precision
	}

	return fmt.Sprintf("DECIMAL(%d, %d)", precision, scale)
}

func Type__uuid(c *migrator.Column) string {
	if c.DBType() == drivers.TypeUUID {
		return "UUID"
	}

	return "VARCHAR(36)"
}

func Type__char(c *migrator.Column) string {
	var max int64 = c.MaxLength
	if max <= 0 {
		max = 1 // Default to CHAR(1) if no length is specified
	}

	return fmt.Sprintf("CHAR(%d)", max)
}

func Type__blob(c *migrator.Column) string {
	var dbType = c.DBType()
	if dbType == drivers.TypeBytes {
		return "BYTEA"
	}

	if dbType == drivers.TypeBLOB {
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
