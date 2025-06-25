package drivers

import (
	"context"
	"database/sql"
	"database/sql/driver"

	"github.com/Nigel2392/go-django/queries/src/query_errors"
	"github.com/go-sql-driver/mysql"
	pg_stdlib "github.com/jackc/pgx/v5/stdlib"
	"github.com/mattn/go-sqlite3"

	"reflect"
)

const (
	SQLITE3_DRIVER_NAME  = "sqlite3"
	MYSQL_DRIVER_NAME    = "mysql"
	MARIADB_DRIVER_NAME  = "mariadb"
	POSTGRES_DRIVER_NAME = "postgres"
)

type Driver struct {
	Name              string
	SupportsReturning SupportsReturningType
	Driver            driver.Driver
	Open              func(ctx context.Context, dsn string, opts ...OpenOption) (Database, error)
}

type driverRegistry struct {
	byName map[string]*Driver
	byType map[reflect.Type]*Driver
}

var drivers = &driverRegistry{
	byName: make(map[string]*Driver),
	byType: make(map[reflect.Type]*Driver),
}

type OpenOption func(driverName string, db any) error

func init() {
	sql.Register(MARIADB_DRIVER_NAME, DriverMariaDB{})

	Register(SQLITE3_DRIVER_NAME, Driver{
		SupportsReturning: SupportsReturningColumns,
		Driver:            &DriverSQLite{},
		Open: func(ctx context.Context, dsn string, opts ...OpenOption) (Database, error) {
			return OpenSQL(SQLITE3_DRIVER_NAME, dsn, opts...)
		},
	})
	Register(MYSQL_DRIVER_NAME, Driver{
		SupportsReturning: SupportsReturningLastInsertId,
		Driver:            &DriverMySQL{},
		Open: func(ctx context.Context, dsn string, opts ...OpenOption) (Database, error) {
			return OpenSQL(MYSQL_DRIVER_NAME, dsn, opts...)
		},
	})
	Register(MARIADB_DRIVER_NAME, Driver{
		SupportsReturning: SupportsReturningColumns,
		Driver:            &DriverMariaDB{},
		Open: func(ctx context.Context, dsn string, opts ...OpenOption) (Database, error) {
			return OpenSQL(MARIADB_DRIVER_NAME, dsn, opts...)
		},
	})
	Register(POSTGRES_DRIVER_NAME, Driver{
		SupportsReturning: SupportsReturningColumns,
		Driver:            &DriverPostgres{},
		Open: func(ctx context.Context, dsn string, opts ...OpenOption) (Database, error) {
			return OpenPGX(ctx, dsn, opts...)
		},
	})
}

/*
Package drivers provides a shortcut to access the registered drivers
and their capabilities. It allows you to check if a driver supports
returning values, and to get the name of the driver for a given SQL database.
*/

type SupportsReturningType string

const (
	SupportsReturningNone         SupportsReturningType = ""
	SupportsReturningLastInsertId SupportsReturningType = "last_insert_id"
	SupportsReturningColumns      SupportsReturningType = "columns"
)

type (
	DriverPostgres = pg_stdlib.Driver
	DriverMySQL    = mysql.MySQLDriver
	DriverSQLite   = sqlite3.SQLiteDriver
	DriverMariaDB  struct {
		mysql.MySQLDriver
	}
	connectorMariaDB struct {
		driver.Connector
	}
)

func (d DriverMariaDB) Open(dsn string) (driver.Conn, error) {
	return d.MySQLDriver.Open(dsn)
}

func (d DriverMariaDB) OpenConnector(dsn string) (driver.Connector, error) {
	connector, err := d.MySQLDriver.OpenConnector(dsn)
	if err != nil {
		return nil, err
	}
	return &connectorMariaDB{
		Connector: connector,
	}, nil
}

func (c *connectorMariaDB) Driver() driver.Driver {
	return &DriverMariaDB{}
}

// SupportsReturning returns the type of returning supported by the database.
// It can be one of the following:
//
// - SupportsReturningNone: no returning supported
// - SupportsReturningLastInsertId: last insert id supported
// - SupportsReturningColumns: returning columns supported
func SupportsReturning(db interface{ Driver() driver.Driver }) SupportsReturningType {
	var d, ok = Retrieve(db.Driver())
	if !ok {
		return SupportsReturningNone
	}
	return d.SupportsReturning
}

// Change allows you to change a driver's properties within a function.
func Change(name string, fn func(driver *Driver)) {
	if driver, exists := drivers.byName[name]; exists {
		fn(driver)
	} else {
		panic("driver not found: " + name)
	}
}

// Register registers a driver with the given database name.
//
// This is used to:
// - determine the proper returning support for the driver
// - open a database connection using the registered opener
//
// If your driver is not one of:
// - github.com/go-django-queries/src/drivers.DriverMariaDB
// - github.com/go-sql-driver/mysql.MySQLDriver
// - github.com/mattn/go-sqlite3.SQLiteDriver
// - github.com/jackc/pgx/v5/stdlib.Driver
//
// Then it explicitly needs to be registered here.
func Register(name string, driver Driver) {
	switch {
	case (name == "" && driver.Name == ""):
		panic("name or driver cannot be empty")
	case name != "" && driver.Name != "" && name != driver.Name:
		panic("name and driver.Name must match")
	case driver.Driver == nil:
		panic("driver.Driver cannot be nil")
	case driver.Open == nil:
		panic("driver.Open cannot be nil")
	case name != "" || driver.Name != "":
		if name == "" {
			name = driver.Name
		}
		driver.Name = name
	}

	drivers.byName[name] = &driver
	drivers.byType[reflect.TypeOf(driver.Driver)] = &driver
}

// Retrieve retrieves the driver by name or type.
func Retrieve(nameOrType any) (*Driver, bool) {
	switch v := nameOrType.(type) {
	case string:
		driver, exists := drivers.byName[v]
		return driver, exists
	case reflect.Type:
		driver, exists := drivers.byType[v]
		return driver, exists
	case driver.Driver:
		return Retrieve(reflect.TypeOf(v))
	case interface{ Driver() driver.Driver }:
		return Retrieve(reflect.TypeOf(v.Driver()))
	}
	panic("nameOrType must be a string, reflect.Type, or driver.Driver")
}

// Open opens a database connection using the registered opener for the given driver name.
//
// This should always be used instead of directly using sql.Open or pgx.Connect.
func Open(ctx context.Context, driverName, dsn string, opts ...OpenOption) (Database, error) {
	opener, exists := drivers.byName[driverName]
	if !exists {
		return nil, query_errors.ErrUnknownDriver
	}
	return opener.Open(ctx, dsn, opts...)
}
