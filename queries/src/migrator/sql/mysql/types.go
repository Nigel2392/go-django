package mysql

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/Nigel2392/go-django/queries/src/migrator"
)

const (
	int16_max = 1 << 15
	int32_max = 1 << 31
)

func registerType(t dbtype.Type, fn func(c *migrator.Column) string) {
	migrator.RegisterColumnType(&drivers.DriverMySQL{}, t, fn)
	migrator.RegisterColumnType(&drivers.DriverMariaDB{}, t, fn)
}

// MYSQL TYPES
func init() {
	registerType(dbtype.Text, Type__string)
	registerType(dbtype.Char, Type__char)
	registerType(dbtype.String, Type__string)
	registerType(dbtype.Int, Type__int)
	registerType(dbtype.Uint, Type__int)
	registerType(dbtype.Bytes, Type__blob)
	registerType(dbtype.Bool, Type__bool)
	registerType(dbtype.Float, Type__float)
	registerType(dbtype.Decimal, Type__decimal)
	registerType(dbtype.UUID, Type__uuid)
	registerType(dbtype.ULID, Type__ulid)
	registerType(dbtype.BLOB, Type__blob)
	registerType(dbtype.JSON, Type__string)
	registerType(dbtype.Timestamp, Type__timestamp)
	registerType(dbtype.LocalTime, Type__datetime)
	registerType(dbtype.DateTime, Type__datetime)
}

func Type__string(c *migrator.Column) string {
	var max int64 = c.MaxLength

	var dbType = c.DBType()
	if dbType == dbtype.Text {
		// If the field is of type drivers.Text, we use TEXT type
		return "TEXT"
	}

	if (dbType == dbtype.String) && (max > 0 && max <= 255) || c.FieldType() == reflect.TypeOf(drivers.String("")) {
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

func Type__ulid(c *migrator.Column) string {
	// ULID is stored as a string of 26 characters
	// in MySQL, we can use VARCHAR(26) for this purpose
	return "VARCHAR(26)"
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

func Type__decimal(c *migrator.Column) string {
	var precision, scale int64 = c.Precision, c.Scale

	if precision == 0 {
		precision = 10
	}

	if scale == 0 {
		scale = 5
	}

	return fmt.Sprintf("DECIMAL(%d, %d)", precision, scale)
}

func Type__uuid(c *migrator.Column) string {
	return "CHAR(36)"
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
