package sqlite

import (
	"database/sql"
	"encoding/json"
	"reflect"
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

// SQLITE TYPES
func init() {
	// register kinds
	migrator.RegisterColumnKind(&drivers.DriverSQLite{}, []reflect.Kind{reflect.String}, Type__string)
	migrator.RegisterColumnKind(&drivers.DriverSQLite{}, []reflect.Kind{reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64}, Type__int)
	migrator.RegisterColumnKind(&drivers.DriverSQLite{}, []reflect.Kind{reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64}, Type__int)
	migrator.RegisterColumnKind(&drivers.DriverSQLite{}, []reflect.Kind{reflect.Float32, reflect.Float64}, Type__float)
	migrator.RegisterColumnKind(&drivers.DriverSQLite{}, []reflect.Kind{reflect.Bool}, Type__bool)
	migrator.RegisterColumnKind(&drivers.DriverSQLite{}, []reflect.Kind{reflect.Array, reflect.Slice, reflect.Map}, Type__string) // SQLite does not have a native array type, so we use string for JSON

	// register types
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, drivers.Text(""), Type__string)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, drivers.Char(""), Type__string)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, drivers.String(""), Type__string)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, drivers.Int(0), Type__int)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, drivers.Bytes(nil), Type__blob)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, drivers.Bool(false), Type__bool)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, drivers.Float(0.0), Type__float)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, drivers.Timestamp{}, Type__datetime)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, drivers.LocalTime{}, Type__datetime)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, drivers.DateTime{}, Type__datetime)

	migrator.RegisterColumnType(&drivers.DriverSQLite{}, (*contenttypes.ContentType)(nil), Type__string)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, contenttypes.BaseContentType[attrs.Definer]{}, Type__string)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, sql.NullString{}, Type__string)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, sql.NullFloat64{}, Type__int)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, sql.NullInt64{}, Type__int)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, sql.NullInt32{}, Type__int)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, sql.NullInt16{}, Type__int)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, sql.NullBool{}, Type__bool)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, sql.NullByte{}, Type__int)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, sql.NullTime{}, Type__datetime)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, time.Time{}, Type__datetime)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, []byte{}, Type__blob)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, json.RawMessage{}, Type__string)
}

func Type__string(c *migrator.Column) string {
	return "TEXT"
}

func Type__blob(c *migrator.Column) string {
	return "BLOB"
}

func Type__float(c *migrator.Column) string {
	return "REAL"
}

func Type__int(c *migrator.Column) string {
	return "INTEGER"
}

func Type__bool(c *migrator.Column) string {
	return "BOOLEAN"
}

func Type__datetime(c *migrator.Column) string {
	return "TIMESTAMP"
}
