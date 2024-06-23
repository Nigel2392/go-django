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

type BaseContentType[T any] struct {
	rType     reflect.Type
	rTypeElem reflect.Type
	pkgPath   string
	modelName string
}

type ContentType interface {
	PkgPath() string
	AppLabel() string
	TypeName() string
	Model() string
	New() interface{}
}

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

func (c *BaseContentType[T]) PkgPath() string {
	return c.pkgPath
}

func (c *BaseContentType[T]) AppLabel() string {
	var lastSlash = strings.LastIndex(c.pkgPath, "/")
	if lastSlash == -1 {
		panic(fmt.Sprintf("invalid package path: %s", c.pkgPath))
	}
	return c.pkgPath[lastSlash+1:]
}

func (c *BaseContentType[T]) Model() string {
	return c.modelName
}

func (c *BaseContentType[T]) New() T {
	return reflect.New(c.rTypeElem).Interface().(T)
}

var ErrInvalidScanType errs.Error = "invalid scan type"

func (c *BaseContentType[T]) TypeName() string {
	var typeName = c.pkgPath
	var modelName = c.modelName
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
	var typeString, ok = src.(string)
	if !ok {
		return errors.Wrapf(
			ErrInvalidScanType,
			"expected string, got %T",
			src,
		)
	}

	var registryObj = DefinitionForType(typeString)
	if registryObj == nil {
		return errors.Errorf("invalid content type: %s", typeString)
	}

	var newCtype = NewContentType(registryObj.ContentObject.(T))
	*c = *newCtype
	return nil
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
