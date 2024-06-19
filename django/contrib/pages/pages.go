package pages

import (
	"context"

	"github.com/Nigel2392/django/contrib/pages/models"
)

type Page interface {
	ID() int64
	Reference() *models.PageNode
}

type SaveablePage interface {
	Page
	Save(ctx context.Context) error
}

type DeletablePage interface {
	Page
	Delete(ctx context.Context) error
}

var pageRegistryObject = &pageRegistry{}

func Register(definition *PageDefinition) {
	pageRegistryObject.RegisterPageDefinition(definition)
}

func Specific(ctx context.Context, node models.PageNode) (Page, error) {
	return pageRegistryObject.SpecificInstance(ctx, node)
}

func DefinitionForType(typeName string) *PageDefinition {
	return pageRegistryObject.DefinitionForType(typeName)
}

func DefinitionForObject(page Page) *PageDefinition {
	return pageRegistryObject.DefinitionForObject(page)
}

func ListDefinitions() []*PageDefinition {
	return pageRegistryObject.ListDefinitions()
}
