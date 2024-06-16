package pages

import (
	"context"

	"github.com/Nigel2392/django/contrib/pages/models"
)

type pageRegistry struct {
	registry map[string]*PageDefinition
}

func (p *pageRegistry) RegisterPageDefinition(definition *PageDefinition) {

	if p.registry == nil {
		p.registry = make(map[string]*PageDefinition)
	}

	if definition == nil {
		panic("pages: RegisterPageDefinition definition is nil")
	}

	if definition.PageObject == nil || definition.GetForID == nil {
		panic("pages: RegisterPageDefinition definition is missing PageObject or GetForID")
	}
	var contentType = definition.ContentType()
	var typeName = contentType.TypeName()
	if _, exists := p.registry[typeName]; exists {
		panic("pages: RegisterPageDefinition called twice for " + typeName)
	}

	p.registry[typeName] = definition
}

func (p *pageRegistry) DefinitionForType(typeName string) *PageDefinition {
	return p.registry[typeName]
}

func (p *pageRegistry) DefinitionForObject(page Page) *PageDefinition {
	var typeName = NewContentType(page).TypeName()
	return p.registry[typeName]
}

func (p *pageRegistry) SpecificInstance(ctx context.Context, node models.PageNode) (Page, error) {
	var typeName = node.Typehash
	var definition, exists = p.registry[typeName]
	if !exists {
		return nil, nil
	}

	return definition.GetForID(ctx, node, node.PageID)
}

var pageRegistryObject = &pageRegistry{}

func RegisterPageDefinition(definition *PageDefinition) {
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
