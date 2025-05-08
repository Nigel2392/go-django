package pages

import (
	"context"

	models "github.com/Nigel2392/go-django/src/contrib/pages/page_models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	dj_models "github.com/Nigel2392/go-django/src/models"
)

type Page interface {
	ID() int64
	Reference() *models.PageNode
}

type SaveablePage interface {
	attrs.Definer
	Page
	dj_models.ContextSaver
}

type DeletablePage interface {
	attrs.Definer
	Page
	dj_models.ContextDeleter
}

var pageRegistryObject = &pageRegistry{}

// Register a page definition
//
// This is an extension of the contenttypes.ContentTypeDefinition
func Register(definition *PageDefinition) {
	pageRegistryObject.RegisterPageDefinition(definition)
}

// Return the custom page object belonging to the given node
func Specific(ctx context.Context, node models.PageNode) (Page, error) {
	return pageRegistryObject.SpecificInstance(ctx, node)
}

// Return the content type definition for the given page type
func DefinitionForType(typeName string) *PageDefinition {
	return pageRegistryObject.DefinitionForType(typeName)
}

// Return the content type definition for the given page object
func DefinitionForObject(page Page) *PageDefinition {
	return pageRegistryObject.DefinitionForObject(page)
}

// Returns a list of all registered page definitions
func ListDefinitions() []*PageDefinition {
	return pageRegistryObject.ListDefinitions()
}

// Returns a list of all registered root page definitions
func ListRootDefinitions() []*PageDefinition {
	return pageRegistryObject.ListRootDefinitions()
}

// Returns a list of all registered page definitions for the given type
func ListDefinitionsForType(typeName string) []*PageDefinition {
	return pageRegistryObject.ListDefinitionsForType(typeName)
}
