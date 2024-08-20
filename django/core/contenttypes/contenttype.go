package contenttypes

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"

	"github.com/Nigel2392/django/core/errs"
	"github.com/pkg/errors"
)

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
	Model() string
	New() interface{}
}

// ShortcutContentType is an interface that defines the methods that a content type must implement.
// It allows for a shorter type name to be used.
//
// This short type name can be used as an alias and allows for easier reference to the model in code.
//
// I.E: "github.com/Nigel2392/auth.User" can be shortened to "auth.User".
type ShortcutContentType[T any] interface {
	ContentType
	ShortTypeName() string
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

// PkgPath returns the package path of the model.
// It is the full import path of the package that the model is defined in.
//
// I.E: "github.com/Nigel2392/auth"
func (c *BaseContentType[T]) PkgPath() string {
	return c.pkgPath
}

// AppLabel returns the app label of the model.
// It is the last part of the package path, and is safe to use for short names.
func (c *BaseContentType[T]) AppLabel() string {
	var lastSlash = strings.LastIndex(c.pkgPath, "/")
	if lastSlash == -1 {
		panic(fmt.Sprintf("invalid package path: %s", c.pkgPath))
	}
	return c.pkgPath[lastSlash+1:]
}

// Model returns the model name.
func (c *BaseContentType[T]) Model() string {
	return c.modelName
}

// New returns a new instance of the model.
func (c *BaseContentType[T]) New() T {
	return reflect.New(c.rTypeElem).Interface().(T)
}

var ErrInvalidScanType errs.Error = "invalid scan type"

// TypeName returns the full type name of the model.
// It is the package path and the model name separated by a dot.
// It is used to uniquely identify the model.
// I.E: "github.com/Nigel2392/auth.User"
func (c *BaseContentType[T]) TypeName() string {
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
	if src == nil {
		return nil
	}
	switch src := src.(type) {
	case string:
		var registryObj = DefinitionForType(src)
		if registryObj == nil {
			return errors.Errorf("invalid content type: %s", src)
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
	var typeString = string(data)
	var registryObj = DefinitionForType(typeString)
	if registryObj == nil {
		return errors.Errorf("invalid content type: %s", typeString)
	}

	var newCtype = NewContentType(registryObj.ContentObject.(T))
	*c = *newCtype
	return nil
}
