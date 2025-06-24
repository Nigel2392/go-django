package migrator

import (
	"database/sql/driver"
	"fmt"
	"reflect"
)

var (
	drivers_to_kinds = make(map[reflect.Type]map[reflect.Kind]func(c *Column) string)
	drivers_to_types = make(map[reflect.Type]map[reflect.Type]func(c *Column) string)
)

// RegisterColumnKind registers a function to convert a field to a database type for a specific driver and kind.
//
// The function will be called with the field as an argument and should return the database type as a string.
func RegisterColumnKind(driver driver.Driver, typ []reflect.Kind, fn func(c *Column) string) {
	t := reflect.TypeOf(driver)
	m, ok := drivers_to_kinds[t]
	if !ok || m == nil {
		m = make(map[reflect.Kind]func(c *Column) string)
		drivers_to_kinds[t] = m
	}

	for _, k := range typ {
		m[k] = fn
	}
}

// RegisterColumnType registers a function to convert a field to a database type for a specific driver and type.
//
// The function will be called with the field as an argument and should return the database type as a string.
func RegisterColumnType(driver driver.Driver, typ interface{}, fn func(c *Column) string) {
	t := reflect.TypeOf(driver)
	m, ok := drivers_to_types[t]
	if !ok || m == nil {
		m = make(map[reflect.Type]func(c *Column) string)
		drivers_to_types[t] = m
	}

	var typType = reflect.TypeOf(typ)

	if typType.Kind() == reflect.Ptr {
		typType = typType.Elem()
	}

	m[typType] = fn
}

// GetFieldType returns the database type for a field based on the driver and field type.
//
// It first checks if the field has a custom database type defined in its attributes referenced by [AttrDBTypeKey],
// and if not, it uses the registered functions to determine the type.
func GetFieldType(driver driver.Driver, c *Column) string {
	var typ = c.FieldType()
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	var fn = getType(driver, typ)
	if fn == nil {
		panic(fmt.Sprintf(
			"no type registered for %s", typ.String(),
		))
	}

	return fn(c)
}

func getType(driver driver.Driver, typ reflect.Type) func(c *Column) string {
	t := reflect.TypeOf(driver)

	// First: absolute type match
	if v, ok := drivers_to_types[t]; ok && v != nil {
		if fn, ok := v[typ]; ok {
			return checkDBType(fn)
		}
	}

	// Fallback: kind-based match
	if m, ok := drivers_to_kinds[t]; ok && m != nil {
		if fn, ok := m[typ.Kind()]; ok {
			return checkDBType(fn)
		}
	}

	return nil
}

func checkDBType(fn func(c *Column) string) func(c *Column) string {
	if fn == nil {
		return nil
	}

	return func(c *Column) string {
		var atts = c.Field.Attrs()
		var dbType = atts[AttrDBTypeKey]
		if dbType != nil {
			return dbType.(string)
		}

		return fn(c)
	}
}
