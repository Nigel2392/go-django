package pages

import (
	"database/sql"
	"database/sql/driver"
	"reflect"
)

var _ sql.Scanner = (*ContentType)(nil)
var _ driver.Valuer = (*ContentType)(nil)

type ContentType struct {
	rType     reflect.Type
	rTypeElem reflect.Type
	pkgPath   string
	modelName string
}

func NewContentType(p Page) *ContentType {
	var rType = reflect.TypeOf(p)
	var rTypeElem = rType
	if rType.Kind() == reflect.Ptr {
		rTypeElem = rType.Elem()
	}
	return &ContentType{
		rType:     rType,
		rTypeElem: rTypeElem,
		pkgPath:   rTypeElem.PkgPath(),
		modelName: rTypeElem.Name(),
	}
}

func (c *ContentType) PkgPath() string {
	return c.pkgPath
}

func (c *ContentType) Model() string {
	return c.modelName
}

func (c *ContentType) New() Page {
	return reflect.New(c.rTypeElem).Interface().(Page)
}

func (c *ContentType) TypeName() string {
	var typeName = c.pkgPath
	var modelName = c.modelName
	var b = make([]byte, 0, len(typeName)+len(modelName)+1)
	b = append(b, typeName...)
	b = append(b, '.')
	b = append(b, modelName...)
	return string(b)
}

func (c *ContentType) Scan(src interface{}) error {
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

	var newCtype = NewContentType(registryObj.PageObject)
	*c = *newCtype
	return nil
}

func (c ContentType) Value() (driver.Value, error) {
	return c.TypeName(), nil
}
