package contenttypes

import (
	"database/sql"
	"database/sql/driver"
	"reflect"

	"github.com/Nigel2392/django/core/errs"
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
	Model() string
	TypeName() string
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

func (c *BaseContentType[T]) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	var typeString, ok = src.(string)
	if !ok {
		return ErrInvalidScanType
	}

	var registryObj = DefinitionForType(typeString)
	if registryObj == nil {
		return ErrInvalidScanType
	}

	var newCtype = NewContentType[T](registryObj.ContentObject.(T))
	*c = *newCtype
	return nil
}

func (c BaseContentType[T]) Value() (driver.Value, error) {
	return c.TypeName(), nil
}
