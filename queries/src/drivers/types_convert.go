package drivers

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/shopspring/decimal"
)

var (
	registerGoDBTypeConversions = &sync.Once{}
	registeredTypeConversions   = make(map[string]reflect.Type)
)

// RegisterGoType registers a Go type with a string identifier.
//
// It stores the full package path and type name of the Go type,
// including pointer levels, in the registeredTypeConversions map.
//
// After registering, the type can be retrieved using the full type name.
// This is useful, for example in engine_table.go where we need to convert
// default values from JSON to Go types.
//
// This function is exposed in case you want to register custom Go types
// OR
// for types which have been migrated to another package,
// but the old type name is still used in migration files.
func RegisterGoType(typeName string, typ any) {
	var t reflect.Type
	switch v := typ.(type) {
	case reflect.Type:
		t = v
	default:
		t = reflect.TypeOf(v)
	}

	if t == nil {
		panic("cannot register nil type")
	}

	registeredTypeConversions[typeName] = t
}

func StringForType(i any) string {
	var t reflect.Type
	switch v := i.(type) {
	case reflect.Type:
		t = v
	default:
		t = reflect.TypeOf(v)
	}

	if t == nil {
		return "nil"
	}

	var ptrCtr = 0
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
		ptrCtr++
	}

	var pkg = t.PkgPath()
	if pkg != "" {
		return fmt.Sprintf(
			"%s%s.%s",
			strings.Repeat("*", ptrCtr),
			pkg, t.Name(),
		)
	}

	return fmt.Sprintf(
		"%s%s", strings.Repeat("*", ptrCtr), t.Name(),
	)
}

func TypeFromString(typeString string) (reflect.Type, bool) {
	registerTypeConversions()
	typ, ok := registeredTypeConversions[typeString]
	return typ, ok
}

// DBToDefaultGoType converts a database type to its default Go representation.
//
// This should only be used as a last resort, as it does not take into account the actual type that
// the value was first defined as.
//
// I.E, if a field is defined as [uuid.UUID], it will return a type of [dbtype.UUID].
// This means that the value will be interpreted as [UUID], and not as [uuid.UUID].
//
// Whenever possible, use [TypeFromString] to convert a type string to a Go type.
func DBToDefaultGoType(dbType dbtype.Type) reflect.Type {
	var scanTo any
	switch dbType {
	case dbtype.Invalid, dbtype.JSON:
		scanTo = new(interface{})
	case dbtype.Text:
		scanTo = new(Text)
	case dbtype.String:
		scanTo = new(String)
	case dbtype.Char:
		scanTo = new(Char)
	case dbtype.Int:
		scanTo = new(Int)
	case dbtype.Uint:
		scanTo = new(Uint)
	case dbtype.Float:
		scanTo = new(Float)
	case dbtype.Bool:
		scanTo = new(Bool)
	case dbtype.UUID:
		scanTo = new(UUID)
	case dbtype.Bytes:
		scanTo = new(Bytes)
	case dbtype.BLOB:
		scanTo = new(BLOB)
	case dbtype.Timestamp:
		scanTo = new(Timestamp)
	case dbtype.LocalTime:
		scanTo = new(LocalTime)
	case dbtype.DateTime:
		scanTo = new(DateTime)
	case dbtype.Decimal:
		scanTo = new(decimal.Decimal)
	default:
		panic(fmt.Errorf(
			"unknown db type %s, cannot convert to Go type",
			dbType.String(),
		))
	}

	return reflect.TypeOf(scanTo).Elem()
}

func registerTypeConversions() {
	registerGoDBTypeConversions.Do(func() {

		if !dbtype.IsLocked() {
			panic("dbtype registry is not locked, call dbtype.Lock() before registering all db type conversions")
		}

		for rTyp, _ := range dbtype.Types() {
			registeredTypeConversions[StringForType(rTyp)] = rTyp
		}
	})
}

//type Default struct {
//	DBType dbtype.Type `json:"db_type"`
//	GOType string      `json:"go_type"`
//	Value  any         `json:"value"`
//}
//
