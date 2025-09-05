package contenttypes

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/errs"
)

func init() {
	dbtype.Add((ContentType)(nil), dbtype.Text)
}

var _ sql.Scanner = (*BaseContentType[any])(nil)
var _ driver.Valuer = (*BaseContentType[any])(nil)

// BaseContentType is a base type for ContentType.
// It implements the ContentType interface.
// It uses generics to better let developers work with their content types.
type BaseContentType[T any] struct {
	rType     reflect.Type
	rTypeElem reflect.Type
	pkgPath   string
	modelName string
}

func (c *BaseContentType[T]) DBType() dbtype.Type {
	return dbtype.Text
}

// ContentType is an interface that defines the methods that a content type must implement.
//
// A content type is a type that represents a model in the database.
//
// It does not represent individual instances of the model.
//
// It is used to store information about a model, such as its human-readable name,
// description, and aliases.
type ContentType interface {
	PkgPath() string
	AppLabel() string
	TypeName() string
	ShortTypeName() string
	Model() string
	New() interface{}
}

// NewContentType returns a new BaseContentType.
// It takes the model object or a pointer to the model object as an argument.
func NewContentType[T any](p T) *BaseContentType[T] {
	var rType = reflect.TypeOf(p)
	var rTypeElem = rType
	if rType.Kind() == reflect.Ptr {
		rTypeElem = rType.Elem()
	}
	return &BaseContentType[T]{
		rType:     rType,
		rTypeElem: rTypeElem,
		pkgPath:   rTypeElem.PkgPath(),
		modelName: rTypeElem.Name(),
	}
}

// ChangeBaseType changes the BaseContentType to a new type.
func ChangeBaseType[OldT, NewT any](c *BaseContentType[OldT]) *BaseContentType[NewT] {
	return &BaseContentType[NewT]{
		rType:     c.rType,
		rTypeElem: c.rTypeElem,
		pkgPath:   c.pkgPath,
		modelName: c.modelName,
	}
}

func (c *BaseContentType[T]) String() string {
	if c == nil {
		return ""
	}
	return fmt.Sprintf("%s.%s", c.pkgPath, c.modelName)
}

// PkgPath returns the package path of the model.
// It is the full import path of the package that the model is defined in.
//
// I.E: "github.com/Nigel2392/auth"
func (c *BaseContentType[T]) PkgPath() string {
	if c == nil {
		return ""
	}
	return c.pkgPath
}

// AppLabel returns the app label of the model.
// It is the last part of the package path, and is safe to use for short names.
func (c *BaseContentType[T]) AppLabel() string {
	if c == nil {
		return ""
	}
	var lastSlash = strings.LastIndex(c.pkgPath, "/")
	if lastSlash == -1 {
		// panic(fmt.Sprintf("invalid package path: %s", c.pkgPath))
		return c.pkgPath
	}
	return c.pkgPath[lastSlash+1:]
}

// Model returns the model name.
func (c *BaseContentType[T]) Model() string {
	if c == nil {
		return ""
	}
	return c.modelName
}

// New returns a new instance of the model.
func (c *BaseContentType[T]) New() T {
	if c == nil {
		var zero T
		var newOf = reflect.TypeOf(zero)
		if newOf == nil || newOf.Kind() == reflect.Invalid {
			panic("BaseContentType is nil")
		}
		if newOf.Kind() == reflect.Interface {
			return reflect.Zero(newOf).Interface().(T)
		}
		if newOf.Kind() == reflect.Ptr {
			return reflect.New(newOf.Elem()).Interface().(T)
		}
		var newVal = reflect.New(newOf).Elem()
		return newVal.Interface().(T)
	}
	return reflect.New(c.rTypeElem).Interface().(T)
}

var ErrInvalidScanType errs.Error = "invalid scan type"

// TypeName returns the full type name of the model.
// It is the package path and the model name separated by a dot.
// It is used to uniquely identify the model.
// I.E: "github.com/Nigel2392/auth.User"
func (c *BaseContentType[T]) TypeName() string {
	if c == nil {
		return ""
	}

	var typeName = c.pkgPath
	var modelName = c.modelName
	var b = make([]byte, 0, len(typeName)+len(modelName)+1)
	b = append(b, typeName...)
	b = append(b, '.')
	b = append(b, modelName...)
	return string(b)
}

// ShortTypeName returns the short type name of the model.
// It is the app label and the model name separated by a dot.
// It is used as an alias for the model.
// I.E: "github.com/Nigel2392/auth.User" -> "auth.User"
func (c *BaseContentType[T]) ShortTypeName() string {
	if c == nil {
		return ""
	}

	var (
		typeName  = c.AppLabel()
		modelName = c.modelName
	)
	var b = make([]byte, 0, len(typeName)+len(modelName)+1)
	b = append(b, typeName...)
	b = append(b, '.')
	b = append(b, modelName...)
	return string(b)
}

// Scan implements the sql.Scanner interface.
// It supports scanning a string into a BaseContentType.
// The string must be a valid type name.
func (c *BaseContentType[T]) Scan(src interface{}) error {
	if c == nil {
		return errors.ValueError.Wrap("BaseContentType is nil")
	}

	if src == nil {
		return nil
	}
	switch src := src.(type) {
	case string:
		var registryObj = DefinitionForType(src)
		if registryObj == nil {
			return errors.InvalidContentType.Wrapf(
				"invalid content type: %s, are you sure it is registered?", src,
			)
		}
		var newCtype = NewContentType(registryObj.ContentObject.(T))
		*c = *newCtype
		return nil
	case []byte:
		return c.Scan(string(src))
	default:
		return errors.Wrapf(
			ErrInvalidScanType,
			"expected string, got %T",
			src,
		)
	}
}

// Value implements the driver.Valuer interface.
// It returns the type name of the BaseContentType.
func (c BaseContentType[T]) Value() (driver.Value, error) {
	return c.TypeName(), nil
}

// MarshalJSON implements the json.Marshaler interface.
// It marshals the type name of the BaseContentType.
func (c BaseContentType[T]) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, c.TypeName())), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// It unmarshals the type name of the BaseContentType.
func (c *BaseContentType[T]) UnmarshalJSON(data []byte) error {
	var typeString string
	if err := json.Unmarshal(data, &typeString); err != nil {
		return errors.Wrap(err, "error unmarshaling content type")
	}

	var registryObj = DefinitionForType(typeString)
	if registryObj == nil {
		return errors.NotImplemented.Wrapf(
			"invalid content type: %s, are you sure it is registered?", typeString,
		)
	}

	var newCtype = NewContentType(registryObj.ContentObject.(T))
	*c = *newCtype
	return nil
}
