package contenttypes

import (
	"fmt"
	"slices"
	"strings"
)

type ContentTypeDefinition struct {
	ContentObject  any
	GetLabel       func() string
	GetDescription func() string
	GetObject      func() any
}

func (p *ContentTypeDefinition) ContentType() ContentType {
	return NewContentType(p.ContentObject)
}

func (p *ContentTypeDefinition) Label() string {
	if p.GetLabel != nil {
		return p.GetLabel()
	}
	return ""
}

func (p *ContentTypeDefinition) Description() string {
	if p.GetDescription != nil {
		return p.GetDescription()
	}
	return ""
}

func (p *ContentTypeDefinition) Object() any {
	if p.GetObject != nil {
		return p.GetObject()
	}
	return p.ContentType().New()
}

type ContentTypeRegistry struct {
	registry map[string]*ContentTypeDefinition
}

func NewContentTypeRegistry() *ContentTypeRegistry {
	return &ContentTypeRegistry{}
}

func (p *ContentTypeRegistry) Register(definition *ContentTypeDefinition) {

	if p.registry == nil {
		p.registry = make(map[string]*ContentTypeDefinition)
	}

	if definition == nil {
		panic("pages: RegisterPageDefinition definition is nil")
	}

	if definition.ContentObject == nil {
		panic("pages: RegisterPageDefinition definition is missing PageObject or GetForID")
	}

	if definition.GetLabel == nil {
		panic("pages: RegisterPageDefinition definition is missing GetLabel")
	}

	var contentType = definition.ContentType()
	var typeName = contentType.TypeName()
	if _, exists := p.registry[typeName]; exists {
		panic("pages: RegisterPageDefinition called twice for " + typeName)
	}

	p.registry[typeName] = definition
}

func (p *ContentTypeRegistry) ListDefinitions() []*ContentTypeDefinition {
	var definitions = make([]*ContentTypeDefinition, 0, len(p.registry))
	for _, definition := range p.registry {
		definitions = append(definitions, definition)
	}
	slices.SortStableFunc(definitions, func(a, b *ContentTypeDefinition) int {
		var result = strings.Compare(a.ContentType().Model(), b.ContentType().Model())
		if result == 0 {
			return strings.Compare(a.Description(), b.Description())
		}
		return result
	})
	return definitions
}

func (p *ContentTypeRegistry) DefinitionForType(typeName string) *ContentTypeDefinition {
	return p.registry[typeName]
}

func (p *ContentTypeRegistry) DefinitionForObject(page any) *ContentTypeDefinition {
	var typeName = NewContentType(page).TypeName()
	return p.registry[typeName]
}

func (p *ContentTypeRegistry) DefinitionForPackage(toplevelPkgName string, typeName string) *ContentTypeDefinition {
	fmt.Printf("Looking for %s.%s in %v\n", toplevelPkgName, typeName, p.registry)
	for fullPkgPath, definition := range p.registry {
		var parts = strings.Split(fullPkgPath, "/")
		if len(parts) < 2 {
			continue
		}
		var pkgInfo = parts[len(parts)-1]
		var infoParts = strings.Split(pkgInfo, ".")
		if len(infoParts) < 2 {
			continue
		}
		var pkg = infoParts[0]
		var typ = infoParts[1]
		if pkg == toplevelPkgName && typ == typeName {
			return definition
		}
	}
	return nil
}

var contentTypeRegistryObject = &ContentTypeRegistry{}

func Register(definition *ContentTypeDefinition) {
	contentTypeRegistryObject.Register(definition)
}

func DefinitionForType(typeName string) *ContentTypeDefinition {
	return contentTypeRegistryObject.DefinitionForType(typeName)
}

func DefinitionForObject(obj any) *ContentTypeDefinition {
	return contentTypeRegistryObject.DefinitionForObject(obj)
}

func ListDefinitions() []*ContentTypeDefinition {
	return contentTypeRegistryObject.ListDefinitions()
}

func DefinitionForPackage(toplevelPkgName string, typeName string) *ContentTypeDefinition {
	return contentTypeRegistryObject.DefinitionForPackage(toplevelPkgName, typeName)
}
