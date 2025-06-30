package sqlite

import (
	"fmt"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
)

const (
	int16_max = 1 << 15
	int32_max = 1 << 31
)

// SQLITE TYPES
func init() {
	// register types
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, drivers.TypeText, Type__string)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, drivers.TypeChar, Type__string)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, drivers.TypeString, Type__string)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, drivers.TypeInt, Type__int)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, drivers.TypeUint, Type__int)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, drivers.TypeBytes, Type__blob)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, drivers.TypeBLOB, Type__blob)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, drivers.TypeBool, Type__bool)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, drivers.TypeFloat, Type__float)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, drivers.TypeDecimal, Type__decimal)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, drivers.TypeUUID, Type__string)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, drivers.TypeJSON, Type__string)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, drivers.TypeTimestamp, Type__datetime)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, drivers.TypeLocalTime, Type__datetime)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, drivers.TypeDateTime, Type__datetime)
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

func Type__int(c *migrator.Column) string {
	return "INTEGER"
}

func Type__bool(c *migrator.Column) string {
	return "BOOLEAN"
}

func Type__datetime(c *migrator.Column) string {
	return "TIMESTAMP"
}
