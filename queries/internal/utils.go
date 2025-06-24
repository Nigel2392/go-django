package internal

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/queries/src/alias"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/query_errors"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/jmoiron/sqlx"

	_ "unsafe"
)

const (
	CACHE_TRAVERSAL_RESULTS = false
)

//go:linkname getRelatedName github.com/Nigel2392/go-django/src/core/attrs.getRelatedName
func getRelatedName(f attrs.Field, default_ string) string

func GetRelatedName(f attrs.Field, default_ string) string {
	if isReverser, ok := f.(interface{ IsReverse() bool }); ok && isReverser.IsReverse() {
		return getRelatedName(f, default_)
	}

	return f.Name()
}

func DefinerListToList[T attrs.Definer](list []attrs.Definer) []T {
	var result = make([]T, len(list))
	for i, obj := range list {
		result[i] = obj.(T)
	}
	return result
}

func NewDefiner[T attrs.Definer]() T {
	return NewObjectFromIface(*new(T)).(T)
}

func GetFromAttrs[T any](attrMap map[string]any, key string) (T, bool) {
	var n T
	if v, ok := attrMap[key]; ok {
		if t, ok := v.(T); ok {
			return t, true
		}
		var (
			rT = reflect.TypeOf((*T)(nil)).Elem()
			vT = reflect.TypeOf(v)
			vV = reflect.ValueOf(v)
		)

		if vT.AssignableTo(rT) {
			return vV.Interface().(T), true
		}

		if vT.ConvertibleTo(rT) {
			return vV.Convert(rT).Interface().(T), true
		}

		return n, false
	}
	return n, false
}

func NewObjectFromIface(obj attrs.Definer) attrs.Definer {
	var objTyp = reflect.TypeOf(obj)
	if objTyp.Kind() != reflect.Ptr {
		panic("newObjectFromIface: objTyp is not a pointer")
	}
	var newObj = reflect.New(objTyp.Elem()).Interface()
	return attrs.NewObject[attrs.Definer](newObj)
}

func ListUnpack(list ...any) []any {
	var result = make([]any, 0, len(list))
	for _, item := range list {
		var rVal = reflect.ValueOf(item)
		switch rVal.Kind() {
		case reflect.Slice, reflect.Array:
			for j := 0; j < rVal.Len(); j++ {
				result = append(result, ListUnpack(rVal.Index(j).Interface())...)
			}
		default:
			result = append(result, item)
		}
	}
	return result
}

type walkFieldsResult struct {
	definer   attrs.Definer
	parent    attrs.Definer
	field     attrs.Field
	chain     []string
	aliases   []string
	isRelated bool
}

var walkFieldsCache = make(map[string]walkFieldsResult)

func WalkFields(
	m attrs.Definer,
	column string,
	aliasGen *alias.Generator,
) (
	definer attrs.Definer,
	parent attrs.Definer,
	f attrs.Field,
	chain []string,
	aliases []string,
	isRelated bool,
	err error,
) {

	var cacheKey = fmt.Sprintf("%T.%s", m, column)
	if CACHE_TRAVERSAL_RESULTS {
		if result, ok := walkFieldsCache[cacheKey]; ok {
			return result.definer, result.parent, result.field, result.chain, result.aliases, result.isRelated, nil
		}
	}

	var parts = strings.Split(column, ".")
	var current = m
	var field attrs.Field

	chain = make([]string, 0, len(parts)-1)
	aliases = make([]string, 0, len(parts)-1)

	var defs = current.FieldDefs()
	for i, part := range parts {
		f, ok := defs.Field(part)
		if !ok {
			return nil, nil, nil, nil, nil, false, fmt.Errorf("internal.WalkFields: field %q not found in %T", part, current)
		}
		field = f

		if i == len(parts)-1 {
			break
		}

		var rel = f.Rel()
		if rel == nil {
			return nil, nil, nil, nil, nil, false, fmt.Errorf("internal.WalkFields: field %q is not a relation", part)
		}

		parent = current
		current = rel.Model()
		defs = current.FieldDefs()
		chain = append(chain, part)
		aliases = append(aliases, aliasGen.GetTableAlias(
			defs.TableName(), strings.Join(chain, "."),
		))

		if current == nil {
			return nil, nil, nil, nil, nil, false, fmt.Errorf("internal.WalkFields: field %q has no related model", part)
		}

		isRelated = true
	}

	if CACHE_TRAVERSAL_RESULTS {
		walkFieldsCache[cacheKey] = walkFieldsResult{
			definer:   current,
			parent:    parent,
			field:     field,
			chain:     chain,
			aliases:   aliases,
			isRelated: isRelated,
		}
	}

	return current, parent, field, chain, aliases, isRelated, nil
}

type QueryInfo struct {
	DatabaseName string // The name of the database connection
	DB           drivers.Database
	DBX          func(string) string
	SqlxDriver   string
}

func SqlxDriverName(db drivers.Database) string {
	var driver = reflect.TypeOf(db.Driver())
	if driver == nil {
		return ""
	}
	if data, ok := drivers.Retrieve(driver); ok {
		return data.Name
	}
	return ""
}

// driverValue prepares the value for the driver to be used in a query.
// it makes sure that the value adheres to the [driver.Value] interface.
func DriverValue(arg any) (driver.Value, error) {
	switch v := arg.(type) {
	case driver.Valuer:
		var err error
		arg, err = v.Value()
		if err != nil {
			return nil, fmt.Errorf(
				"failed to get value from driver.Valuer: %w", err,
			)
		}
	case attrs.Definer:
		var defs = v.FieldDefs()
		var prim = defs.Primary()
		return DriverValue(prim.GetValue())

	case nil, time.Time, string, []byte, int64, uint64, float64, bool:
		// these types are already compatible with driver.Value
		return v, nil
	}

	var rVal = reflect.ValueOf(arg)
	if !rVal.IsValid() || rVal.Kind() == reflect.Ptr && rVal.IsNil() {
		return nil, nil
	}

	switch rVal.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		arg = rVal.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		arg = rVal.Uint()
	case reflect.Float32, reflect.Float64:
		arg = rVal.Float()
	case reflect.String:
		arg = rVal.String()
	case reflect.Bool:
		arg = rVal.Bool()
	case reflect.Slice, reflect.Array:
		if rVal.Type().Elem().Kind() == reflect.Uint8 {
			//  byte slice, e.g. for binary data
			arg = rVal.Bytes()
		} else {
			return nil, fmt.Errorf(
				"unsupported slice type for driver.Value: %s (%T): %w",
				rVal.Type().Elem().Kind(), arg, query_errors.ErrTypeMismatch,
			)
		}
	default:
		return nil, fmt.Errorf(
			"unsupported type for driver.Value: %s (%T): %w",
			rVal.Kind(), arg, query_errors.ErrTypeMismatch,
		)
	}

	return arg, nil
}

func GetQueryInfo(dbKey string) (*QueryInfo, error) {
	var db, ok = django.ConfigGetOK[drivers.Database](
		django.Global.Settings,
		dbKey,
	)
	if !ok {
		return nil, query_errors.ErrNoDatabase
	}

	var sqlxDriver = SqlxDriverName(db)
	if sqlxDriver == "" {
		return nil, query_errors.ErrUnknownDriver
	}

	var queryInfo = &QueryInfo{
		DatabaseName: dbKey,
		DB:           db,
		DBX: func(s string) string {
			return sqlx.Rebind(sqlx.BindType(sqlxDriver), s)
		},
		SqlxDriver: sqlxDriver,
	}

	return queryInfo, nil
}
