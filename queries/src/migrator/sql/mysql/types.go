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

// MYSQL TYPES
func init() {
	// register kinds
	migrator.RegisterColumnKind(&drivers.DriverMySQL{}, []reflect.Kind{reflect.String}, Type__string)
	migrator.RegisterColumnKind(&drivers.DriverMySQL{}, []reflect.Kind{reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64}, Type__int)
	migrator.RegisterColumnKind(&drivers.DriverMySQL{}, []reflect.Kind{reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64}, Type__int)
	migrator.RegisterColumnKind(&drivers.DriverMySQL{}, []reflect.Kind{reflect.Float32, reflect.Float64}, Type__float)
	migrator.RegisterColumnKind(&drivers.DriverMySQL{}, []reflect.Kind{reflect.Bool}, Type__bool)
	migrator.RegisterColumnKind(&drivers.DriverMySQL{}, []reflect.Kind{reflect.Array, reflect.Slice, reflect.Map}, Type__string) // MySQL does not have a native array type, so we use string for JSON

	// register types
	migrator.RegisterColumnType(&drivers.DriverMySQL{}, drivers.Text(""), Type__string)
	migrator.RegisterColumnType(&drivers.DriverMySQL{}, drivers.String(""), Type__string)
	migrator.RegisterColumnType(&drivers.DriverMySQL{}, drivers.Int(0), Type__int)
	migrator.RegisterColumnType(&drivers.DriverMySQL{}, drivers.Bytes(nil), Type__blob)
	migrator.RegisterColumnType(&drivers.DriverMySQL{}, drivers.Bool(false), Type__bool)
	migrator.RegisterColumnType(&drivers.DriverMySQL{}, drivers.Float(0.0), Type__float)
	migrator.RegisterColumnType(&drivers.DriverMySQL{}, drivers.Time{}, Type__datetime)

	migrator.RegisterColumnType(&drivers.DriverMySQL{}, (*contenttypes.ContentType)(nil), Type__string)
	migrator.RegisterColumnType(&drivers.DriverMySQL{}, contenttypes.BaseContentType[attrs.Definer]{}, Type__string)
	migrator.RegisterColumnType(&drivers.DriverMySQL{}, sql.NullString{}, Type__string)
	migrator.RegisterColumnType(&drivers.DriverMySQL{}, sql.NullFloat64{}, Type__int)
	migrator.RegisterColumnType(&drivers.DriverMySQL{}, sql.NullInt64{}, Type__int)
	migrator.RegisterColumnType(&drivers.DriverMySQL{}, sql.NullInt32{}, Type__int)
	migrator.RegisterColumnType(&drivers.DriverMySQL{}, sql.NullInt16{}, Type__int)
	migrator.RegisterColumnType(&drivers.DriverMySQL{}, sql.NullBool{}, Type__bool)
	migrator.RegisterColumnType(&drivers.DriverMySQL{}, sql.NullByte{}, Type__int)
	migrator.RegisterColumnType(&drivers.DriverMySQL{}, sql.NullTime{}, Type__datetime)
	migrator.RegisterColumnType(&drivers.DriverMySQL{}, time.Time{}, Type__datetime)
	migrator.RegisterColumnType(&drivers.DriverMySQL{}, []byte{}, Type__string)

	// register kinds
	migrator.RegisterColumnKind(&drivers.DriverMariaDB{}, []reflect.Kind{reflect.String}, Type__string)
	migrator.RegisterColumnKind(&drivers.DriverMariaDB{}, []reflect.Kind{reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64}, Type__int)
	migrator.RegisterColumnKind(&drivers.DriverMariaDB{}, []reflect.Kind{reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64}, Type__int)
	migrator.RegisterColumnKind(&drivers.DriverMariaDB{}, []reflect.Kind{reflect.Float32, reflect.Float64}, Type__float)
	migrator.RegisterColumnKind(&drivers.DriverMariaDB{}, []reflect.Kind{reflect.Bool}, Type__bool)
	migrator.RegisterColumnKind(&drivers.DriverMariaDB{}, []reflect.Kind{reflect.Array, reflect.Slice, reflect.Map}, Type__string) // MySQL does not have a native array type, so we use string for JSON

	// register types
	migrator.RegisterColumnType(&drivers.DriverMariaDB{}, drivers.Text(""), Type__string)
	migrator.RegisterColumnType(&drivers.DriverMariaDB{}, drivers.String(""), Type__string)
	migrator.RegisterColumnType(&drivers.DriverMariaDB{}, drivers.Int(0), Type__int)
	migrator.RegisterColumnType(&drivers.DriverMariaDB{}, drivers.Bytes(nil), Type__blob)
	migrator.RegisterColumnType(&drivers.DriverMariaDB{}, drivers.Bool(false), Type__bool)
	migrator.RegisterColumnType(&drivers.DriverMariaDB{}, drivers.Float(0.0), Type__float)
	migrator.RegisterColumnType(&drivers.DriverMariaDB{}, drivers.Time{}, Type__datetime)

	migrator.RegisterColumnType(&drivers.DriverMariaDB{}, (*contenttypes.ContentType)(nil), Type__string)
	migrator.RegisterColumnType(&drivers.DriverMariaDB{}, contenttypes.BaseContentType[attrs.Definer]{}, Type__string)
	migrator.RegisterColumnType(&drivers.DriverMariaDB{}, sql.NullString{}, Type__string)
	migrator.RegisterColumnType(&drivers.DriverMariaDB{}, sql.NullFloat64{}, Type__int)
	migrator.RegisterColumnType(&drivers.DriverMariaDB{}, sql.NullInt64{}, Type__int)
	migrator.RegisterColumnType(&drivers.DriverMariaDB{}, sql.NullInt32{}, Type__int)
	migrator.RegisterColumnType(&drivers.DriverMariaDB{}, sql.NullInt16{}, Type__int)
	migrator.RegisterColumnType(&drivers.DriverMariaDB{}, sql.NullBool{}, Type__bool)
	migrator.RegisterColumnType(&drivers.DriverMariaDB{}, sql.NullByte{}, Type__int)
	migrator.RegisterColumnType(&drivers.DriverMariaDB{}, sql.NullTime{}, Type__datetime)
	migrator.RegisterColumnType(&drivers.DriverMariaDB{}, time.Time{}, Type__datetime)
	migrator.RegisterColumnType(&drivers.DriverMariaDB{}, []byte{}, Type__string)

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

func Type__datetime(c *migrator.Column) string {
	return "TIMESTAMP"
}
