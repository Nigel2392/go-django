package contenttypes

import (
	"slices"
	"strings"
)

type ContentTypeDefinition struct {
	ContentObject  any
	GetLabel       func() string
	GetDescription func() string
	GetObject      func() any
	Aliases        []string
	_cType         ContentType
}

func (p *ContentTypeDefinition) ContentType() ContentType {
	if p._cType == nil {
		p._cType = NewContentType(p.ContentObject)
	}
	return p._cType
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
	registry   map[string]*ContentTypeDefinition
	aliases    map[string][]string
	aliasesRev map[string]string
}

func NewContentTypeRegistry() *ContentTypeRegistry {
	return &ContentTypeRegistry{}
}

func (p *ContentTypeRegistry) Aliases(typeName string) []string {
	return p.aliases[typeName]
}

func (p *ContentTypeRegistry) ReverseAlias(alias string) string {
	if p.aliasesRev == nil {
		return alias
	}
	if typeName, exists := p.aliasesRev[alias]; exists {
		return typeName
	}
	return alias
}

func (p *ContentTypeRegistry) RegisterAlias(alias string, typeName string) {
	if p.aliases == nil {
		p.aliases = make(map[string][]string)
		p.aliasesRev = make(map[string]string)
	}

	if _, exists := p.aliasesRev[alias]; exists {
		panic("pages: RegisterAlias called twice for alias " + alias)
	}

	p.aliasesRev[alias] = typeName

	if _, exists := p.aliases[typeName]; !exists {
		p.aliases[typeName] = make([]string, 0)
	}

	p.aliases[typeName] = append(p.aliases[typeName], alias)

	var aliasParts = strings.Split(alias, "/")
	if len(aliasParts) < 2 {
		return
	}

	var pkgParts = strings.Split(typeName, "/")
	if len(pkgParts) < 2 {
		return
	}

	var aliasPkg = aliasParts[len(aliasParts)-1]
	p.aliasesRev[aliasPkg] = typeName

	if _, exists := p.aliases[typeName]; !exists {
		p.aliases[typeName] = make([]string, 0)
	}
	p.aliases[typeName] = append(p.aliases[typeName], aliasPkg)
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

	if definition.Aliases != nil {
		for _, alias := range definition.Aliases {
			p.RegisterAlias(alias, typeName)
		}
	}

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
	if p.aliasesRev != nil {
		if alias, exists := p.aliasesRev[typeName]; exists {
			typeName = alias
		}
	}
	return p.registry[typeName]
}

func (p *ContentTypeRegistry) DefinitionForObject(page any) *ContentTypeDefinition {
	var typeName = NewContentType(page).TypeName()
	return p.registry[typeName]
}

func (p *ContentTypeRegistry) DefinitionForPackage(toplevelPkgName string, typeName string) *ContentTypeDefinition {
	if p.aliasesRev != nil {
		var togetherBuf = make([]byte, 0, len(toplevelPkgName)+len(typeName)+1)
		togetherBuf = append(togetherBuf, toplevelPkgName...)
		togetherBuf = append(togetherBuf, '.')
		togetherBuf = append(togetherBuf, typeName...)
		if alias, exists := p.aliasesRev[string(togetherBuf)]; exists {
			return p.registry[alias]
		}

	}

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

func Aliases(typeName string) []string {
	return contentTypeRegistryObject.Aliases(typeName)
}

func ReverseAlias(alias string) string {
	return contentTypeRegistryObject.ReverseAlias(alias)
}

func RegisterAlias(alias string, typeName string) {
	contentTypeRegistryObject.RegisterAlias(alias, typeName)
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
