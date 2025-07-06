package sqlite

import (
	"fmt"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/Nigel2392/go-django/queries/src/migrator"
)

const (
	int16_max = 1 << 15
	int32_max = 1 << 31
)

// SQLITE TYPES
func init() {
	// register types
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, dbtype.Text, Type__string)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, dbtype.Char, Type__string)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, dbtype.String, Type__string)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, dbtype.Int, Type__int)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, dbtype.Uint, Type__int)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, dbtype.Bytes, Type__blob)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, dbtype.BLOB, Type__blob)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, dbtype.Bool, Type__bool)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, dbtype.Float, Type__float)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, dbtype.Decimal, Type__decimal)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, dbtype.UUID, Type__string)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, dbtype.ULID, Type__string)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, dbtype.JSON, Type__string)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, dbtype.Timestamp, Type__datetime)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, dbtype.LocalTime, Type__datetime)
	migrator.RegisterColumnType(&drivers.DriverSQLite{}, dbtype.DateTime, Type__datetime)
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
