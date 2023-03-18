package modelutils

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/Nigel2392/go-django/core/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Default models to validate if a given struct is/implements a model.
var modelTypes []any = []any{
	gorm.Model{},
	models.Model[models.DefaultIDField]{},
	models.Model[uuid.UUID]{},
	models.Model[string]{},
	models.Model[int64]{},
	models.Model[int32]{},
	models.Model[int16]{},
	models.Model[int8]{},
	models.Model[int]{},
	models.Model[uint64]{},
	models.Model[uint32]{},
	models.Model[uint16]{},
	models.Model[uint8]{},
	models.Model[uint]{},
}

// Register a model type.
//
// This is used to check if a type is a model.
//
// By default, the following types are registered:
//
// - gorm.Model
func Register(model ...any) {
	modelTypes = append(modelTypes, model...)
}

// ID is a type that represents a model ID.
//
// It can be used to represent a model ID as a string, int or uint.
type ID string

// Return the ID as a string.
func (id ID) String() string {
	if id.IsUUID() {
		return id.UUID().String()
	}
	return string(id)
}

// Return the ID as an int.
func (id ID) Int() int {
	var i, _ = strconv.Atoi(string(id))
	return i
}

// Return the ID as a uint.
func (id ID) Uint() uint {
	return uint(id.Int())
}

// Return the ID as a uuid.
func (id ID) UUID() uuid.UUID {
	if id.IsZero() {
		return uuid.Nil
	}
	var uid, _ = uuid.Parse(string(id))
	return uid
}

// Returns true if ID is a uuid.
func (id ID) IsUUID() bool {
	_, err := uuid.Parse(string(id))
	return err == nil
}

// Switch on the database type and perform a WHERE query on the ID field with the appropriate value.
func (id ID) Switch(m any, IDField string, db *gorm.DB) (*gorm.DB, error) {
	var idField, err = GetField(m, "ID")
	if err != nil {
		return db, err
	}
	switch idField.(type) {
	case string:
		return db.Where("id = ?", id.String()), nil
	case int64, int32, int16, int8, int:
		return db.Where("id = ?", id.Int()), nil
	case uint64, uint32, uint16, uint8, uint:
		return db.Where("id = ?", id.Uint()), nil
	case uuid.UUID:
		return db.Where("id = ?", id.UUID()), nil
	case models.DefaultIDField:
		return db.Where("id = ?", models.DefaultIDField(id.UUID())), nil
	}
	return db, fmt.Errorf("unsupported ID type: %s", reflect.TypeOf(idField))
}

// Cast the ID to the given type.
func (id ID) Cast(to any) (any, error) {
	switch to := to.(type) {
	case reflect.Type:
		return id.castType(to)
	case reflect.Value:
		return id.castValue(to)
	case reflect.StructField:
		return id.castField(to)
	default:
		return id.castType(reflect.TypeOf(to))
	}
}

func (id *ID) Scan(value interface{}) error {
	var vID, ok = value.(string)
	if !ok {
		return fmt.Errorf("failed to scan ID: %s", reflect.TypeOf(value))
	}
	*id = ID(vID)
	return nil
}

func (id ID) Value() (driver.Value, error) {
	return string(id), nil
}

// Cast the ID to the given type.
func (id ID) castType(to reflect.Type) (any, error) {
	switch to.Kind() {
	case reflect.String:
		return id.String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return id.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return id.Uint(), nil
	default:
		if to == reflect.TypeOf(uuid.UUID{}) {
			return id.UUID(), nil
		} else if to == reflect.TypeOf(models.DefaultIDField{}) {
			return models.DefaultIDField(id.UUID()), nil
		}
	}
	return nil, fmt.Errorf("unsupported type: %s", to.Kind())
}

// Cast the ID to the given value.
func (id ID) castValue(to reflect.Value) (any, error) {
	return id.castType(to.Type())
}

// Cast the ID to the given field.
func (id ID) castField(to reflect.StructField) (any, error) {
	return id.castType(to.Type)
}

// Return true if the ID is a zero value.
//
// This is done by comparing the ID with the following values:
//
// - ""
// - "0"
func (id ID) IsZero() bool {
	return id == "" || id == "0" || id == "00000000-0000-0000-0000-000000000000"
}

// Return true if the ID is a digit.
func (id ID) IsDigit() bool {
	for _, c := range id {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// Validate if a given type is a model.
//
// This is done by comparing the name and package path of the type.
//
// If the type implements a model, it will return true.
//
// Models can be specified by using the Register function.
func IsModel(val any) bool {
	switch t := val.(type) {
	case reflect.Type:
		return hasModelField(t) //|| isModel(t)
	case reflect.Value:
		return hasModelField(t.Type())
	case reflect.StructField:
		return IsModelField(t) || hasModelField(t.Type)
	}
	var typ = reflect.TypeOf(val)
	var has = hasModelField(typ) //|| isModel(typ)
	if !has {
		return false
	}
	var is = false
	for _, modelType := range modelTypes {
		// Validate if the type is a model.
		if typ == reflect.TypeOf(modelType) {
			is = true
			break
		}
	}
	return is
}

// Validate if a given type has a model field.
func hasModelField(typ reflect.Type) bool {
	typ = DePtrType(typ)
	if typ.Kind() != reflect.Struct {
		return false
	}
	for i := 0; i < typ.NumField(); i++ {
		var isField = IsModelField(typ.Field(i))
		if isField {
			return true
		}
	}
	return false
}

// Validate if a given type is a model field.
//
// This is done by comparing the name and package path of the type.
func IsModelField(typ any) bool {
	var compareName, comparePkgPath string
	switch typ := typ.(type) {
	case reflect.Type:
		compareName = typ.Name()
		comparePkgPath = typ.PkgPath()
	case reflect.Value:
		compareName = typ.Type().Name()
		comparePkgPath = typ.Type().PkgPath()
	case reflect.StructField:
		compareName = typ.Name
		comparePkgPath = typ.Type.PkgPath()
	}
	for _, t := range modelTypes {
		var name, pkgpath = models.GetMetaData(t)
		// fmt.Println(name, pkgpath, compareName, comparePkgPath)
		// If the model is a generic model, it will have a generic name.
		// we need to check if the package path is the same.
		if name == compareName && pkgpath == comparePkgPath ||
			strings.Contains(name, compareName) && pkgpath == comparePkgPath {
			return true
		}
	}
	return false
}

// Get the ID of any given struct.
//
// The ID field is determined by the IDField parameter.
func GetID(val any, IDField string) ID {
	var v = DePtr(val)
	if !v.IsValid() {
		return ""
	}
	var id = v.FieldByName(IDField)
	if !id.IsValid() {
		return ""
	}
	switch id := id.Interface().(type) {
	case gorm.Model:
		return ID(strconv.Itoa(int(id.ID)))
	case []byte:
		return ID(string(id))
	case string:
		return ID(id)
	case uuid.UUID:
		return ID(id.String())
	default:
		return ID(fmt.Sprint(id))
	}
}

// Get the name of any given struct.
func GetName(val any) string {
	var v = reflect.TypeOf(val)
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	return v.Name()
}

// Get a default display for the given model.
//
// If the model implements the DisplayableModel interface, the String() method will be called.
func GetModelDisplay(mdl any) string {
	switch mdlType := mdl.(type) {
	case fmt.Stringer:
		return mdlType.String()
	}

	var id = GetID(mdl, "ID")
	var name = GetName(mdl)
	if id.IsZero() {
		return name
	}

	return name + " " + id.String()
}

// Get fields for preloading and joining.
//
// This function will return a list of fields that should be preloaded and a list of fields that should be joined.
func GetPreloadFields(s any) (preload []string, joins []string) {
	var typeOf = reflect.TypeOf(s)
	if typeOf.Kind() == reflect.Ptr {
		typeOf = typeOf.Elem()
	}

	preload = make([]string, 0)
	joins = make([]string, 0)

	for i := 0; i < typeOf.NumField(); i++ {
		var field = typeOf.Field(i)
		var fieldType = field.Type

		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}

		// Validate wether the field is, or contains a gorm.Model
		if fieldType.Kind() == reflect.Struct &&
			!IsModelField(fieldType) &&
			fieldType.Name() != "Time" {
			// Append the field.
			joins = append(joins, field.Name)
		} else if fieldType.Kind() == reflect.Slice {
			var sliceType = fieldType.Elem()
			if sliceType.Kind() == reflect.Ptr {
				sliceType = sliceType.Elem()
			}
			// Validate wether the field is, or contains a gorm.Model
			if sliceType.Kind() == reflect.Struct &&
				!IsModelField(fieldType) &&
				sliceType.Name() != "Time" {
				// Append the field.
				preload = append(preload, field.Name)
			}
		}
	}

	return preload, joins
}

// Get a new model from the given model.
//
// If ptr is true, a pointer to the model will be returned.
func GetNewModel(m any, ptr bool) any {
	var typeOf = reflect.TypeOf(m)
	if typeOf.Kind() == reflect.Ptr {
		typeOf = typeOf.Elem()
	}
	if ptr {
		return reflect.New(typeOf).Interface()
	}
	return reflect.New(typeOf).Elem().Interface()
}

// Get a new slice of the given model.
func GetNewModelSlice(m any) any {
	var typeOf = reflect.TypeOf(m)
	if typeOf.Kind() == reflect.Ptr {
		typeOf = typeOf.Elem()
	}
	return reflect.MakeSlice(reflect.SliceOf(typeOf), 0, 0).Interface()
}
