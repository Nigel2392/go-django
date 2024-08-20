package contenttypes

import (
	"fmt"
	"slices"
	"strings"
)

// ContentTypeDefinition is a struct that holds information about a model.
//
// This struct is used to register models with the ContentTypeRegistry.
//
// It is used to store information about a model, such as its human-readable name,
// description, and aliases.
//
// It allows for more flexibility to work with models that are not directly
// created by the framework, such as models from third-party apps.
type ContentTypeDefinition struct {
	// The model itself.
	//
	// This must be either a struct, or a pointer to a struct.
	ContentObject any

	// A function that returns the human-readable name of the model.
	//
	// This can be used to provide a custom name for the model.
	GetLabel func() string

	// A function that returns a description of the model.
	//
	// This should return an accurate description of the model and what it represents.
	GetDescription func() string

	// A function that returns a new instance of the model.
	//
	// This should return a new instance of the model that can be safely typecast to the
	// correct model type.
	GetObject func() any

	// A list of aliases for the model.
	//
	// This can be used to provide additional names for the model and make it easier to
	// reference the model in code from the registry.
	//
	// For example, after a big refactor or renaming of a model, you can add the old name
	// as an alias to make it easier to reference the model in code.
	//
	// This should be the full type name of the model, including the package path.
	Aliases []string

	// The ContentType instance for the model.
	//
	// This is automatically generated from the ContentObject field.
	_cType ContentType
}

// Returns the ContentType instance for this model.
func (p *ContentTypeDefinition) ContentType() ContentType {
	if p._cType == nil {
		p._cType = NewContentType(p.ContentObject)
	}
	return p._cType
}

// Returns the model's human-readable name.
func (p *ContentTypeDefinition) Label() string {
	if p.GetLabel != nil {
		return p.GetLabel()
	}
	return ""
}

// Returns a description of the model and what it represents.
func (p *ContentTypeDefinition) Description() string {
	if p.GetDescription != nil {
		return p.GetDescription()
	}
	return ""
}

// Returns a new instance of the model.
//
// This method returns an object of type Any which can safely be typecast to the
// correct model type.
func (p *ContentTypeDefinition) Object() any {
	if p.GetObject != nil {
		return p.GetObject()
	}
	return p.ContentType().New()
}

// ContentTypeRegistry is a struct that holds information about all registered models.
//
// It allows for easy management of different models and their aliases.
type ContentTypeRegistry struct {
	registry   map[string]*ContentTypeDefinition
	aliases    map[string][]string
	aliasesRev map[string]string
}

// NewContentTypeRegistry creates a new ContentTypeRegistry instance.
//
// Generally, the package-level functions should be used instead of creating a new instance
func NewContentTypeRegistry() *ContentTypeRegistry {
	return &ContentTypeRegistry{}
}

// Aliases returns a list of aliases for the given model's type name.
func (p *ContentTypeRegistry) Aliases(typeName string) []string {
	return p.aliases[typeName]
}

// ReverseAlias returns the type name for the given alias.
func (p *ContentTypeRegistry) ReverseAlias(alias string) string {
	if p.aliasesRev == nil {
		return alias
	}
	if typeName, exists := p.aliasesRev[alias]; exists {
		return typeName
	}
	return alias
}

// RegisterAlias registers an alias for a given type name.
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

// Register registers a model with the registry.
//
// It will automatically generate the type name for the model and add it to the registry.
//
// If any aliases are provided, they will also be added to the registry.
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

	if p.aliasesRev == nil {
		p.aliasesRev = make(map[string]string)
	}

	var alias = fmt.Sprintf(
		"%s.%s",
		contentType.AppLabel(),
		contentType.Model(),
	)
	p.RegisterAlias(alias, typeName)

	if definition.Aliases != nil {
		for _, alias := range definition.Aliases {
			p.RegisterAlias(alias, typeName)
		}
	}

}

// ListDefinitions returns a list of all registered models.
//
// The list is sorted by model name and description.
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

// DefinitionForType returns the ContentTypeDefinition for the given type name.
//
// If the type name is an alias, the definition for the actual type name will be returned.
func (p *ContentTypeRegistry) DefinitionForType(typeName string) *ContentTypeDefinition {
	if p.aliasesRev != nil {
		if alias, exists := p.aliasesRev[typeName]; exists {
			typeName = alias
		}
	}
	return p.registry[typeName]
}

// DefinitionForObject returns the ContentTypeDefinition for the given object.
func (p *ContentTypeRegistry) DefinitionForObject(page any) *ContentTypeDefinition {
	var typeName = NewContentType(page).TypeName()
	return p.registry[typeName]
}

// DefinitionForPackage returns the ContentTypeDefinition for the given package and type name.
//
// If the type name is an alias, the definition for the actual type name will be returned.
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

// Register registers a model with the registry.
func Register(definition *ContentTypeDefinition) {
	contentTypeRegistryObject.Register(definition)
}

// Aliases returns a list of aliases for the given model's type name.
func Aliases(typeName string) []string {
	return contentTypeRegistryObject.Aliases(typeName)
}

// ReverseAlias returns the type name for the given alias.
func ReverseAlias(alias string) string {
	return contentTypeRegistryObject.ReverseAlias(alias)
}

// RegisterAlias registers an alias for a given type name.
func RegisterAlias(alias string, typeName string) {
	contentTypeRegistryObject.RegisterAlias(alias, typeName)
}

// DefinitionForType returns the ContentTypeDefinition for the given type name.
func DefinitionForType(typeName string) *ContentTypeDefinition {
	return contentTypeRegistryObject.DefinitionForType(typeName)
}

// DefinitionForObject returns the ContentTypeDefinition for the given object.
func DefinitionForObject(obj any) *ContentTypeDefinition {
	return contentTypeRegistryObject.DefinitionForObject(obj)
}

// ListDefinitions returns a list of all registered models.
func ListDefinitions() []*ContentTypeDefinition {
	return contentTypeRegistryObject.ListDefinitions()
}

// DefinitionForPackage returns the ContentTypeDefinition for the given package and type name.
func DefinitionForPackage(toplevelPkgName string, typeName string) *ContentTypeDefinition {
	return contentTypeRegistryObject.DefinitionForPackage(toplevelPkgName, typeName)
}
