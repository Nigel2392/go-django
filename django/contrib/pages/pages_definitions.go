package pages

import (
	"context"
	"reflect"

	"github.com/Nigel2392/django/contrib/pages/models"
)

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

func (c *ContentType) New() SaveablePage {
	return reflect.New(c.rTypeElem).Interface().(SaveablePage)
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

type PageDefinition struct {
	PageObject SaveablePage
	GetForID   func(ctx context.Context, ref models.PageNode, id int64) (SaveablePage, error)
}

func (p *PageDefinition) ContentType() *ContentType {
	return NewContentType(p.PageObject)
}
