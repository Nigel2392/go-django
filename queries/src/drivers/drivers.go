package drivers

import (
	"context"
	"database/sql/driver"
	"fmt"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"

	"reflect"
)

type Driver struct {
	Name               string
	SupportsReturning  SupportsReturningType
	Driver             driver.Driver
	Open               func(ctx context.Context, drv *Driver, dsn string, opts ...OpenOption) (Database, error)
	BuildDatabaseError func(err error) errors.DatabaseError
	ExplainQuery       func(ctx context.Context, q DB, query string, args []any) (string, error)
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
//
// This is only really useful for testing purposes, where you might want to
// change the behavior of a driver during tests.
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
	case *Driver:
		return v, true
	case interface{ Driver() driver.Driver }:
		return Retrieve(reflect.TypeOf(v.Driver()))
	}
	panic("nameOrType must be a string, reflect.Type, or driver.Driver")
}

// Open opens a database connection using the registered opener for the given driver name.
//
// This should always be used instead of directly using sql.Open or pgx.Connect.
func Open(ctx context.Context, driverName, dsn string, opts ...OpenOption) (Database, error) {
	driver, exists := drivers.byName[driverName]
	if !exists {
		return nil, errors.UnknownDriver.WithCause(fmt.Errorf(
			"driver not found: %s", driverName,
		))
	}

	var db, err = driver.Open(ctx, driver, dsn, opts...)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// DatabaseError converts a driver error to a [errors.DatabaseError] and a boolean indicating
// whether the driver was able to convert the error.
func DatabaseError(driverOrName any, err error) (errors.DatabaseError, error, bool) {
	var d, ok = Retrieve(driverOrName)
	if !ok || d == nil || d.BuildDatabaseError == nil {
		return nil, err, false
	}

	var dbErr = d.BuildDatabaseError(err)
	return dbErr, dbErr, true
}

// databaseError is used internally to convert a driver error to a
// [errors.DatabaseError]
func databaseError(d *Driver, err error) error {
	if d == nil || d.BuildDatabaseError == nil || err == nil {
		return err
	}
	return d.BuildDatabaseError(err)
}
