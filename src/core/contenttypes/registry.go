package contenttypes

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Nigel2392/go-signals"
)

// ContentTypeRegistry is a struct that holds information about all registered models.
//
// It allows for easy management of different models and their aliases.
type ContentTypeRegistry struct {
	registry   map[string]*ContentTypeDefinition
	aliases    map[string][]string
	aliasesRev map[string]string

	onRegister signals.Signal[*ContentTypeDefinition]
}

// NewContentTypeRegistry creates a new ContentTypeRegistry instance.
//
// Generally, the package-level functions should be used instead of creating a new instance
func NewContentTypeRegistry() *ContentTypeRegistry {
	return &ContentTypeRegistry{
		registry:   make(map[string]*ContentTypeDefinition),
		aliases:    make(map[string][]string),
		aliasesRev: make(map[string]string),
		onRegister: signals.New[*ContentTypeDefinition]("contenttypes.OnRegister"),
	}
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
		return
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
		panic("ContentTypeRegistry: Register definition is nil")
	}

	if definition.ContentObject == nil {
		panic("ContentTypeRegistry: Register definition is missing ContentObject")
	}

	var contentType = definition.ContentType()
	var typeName = contentType.TypeName()
	if _, exists := p.registry[typeName]; exists {
		p.EditDefinition(definition)
		return
	} else {
		p.registry[typeName] = definition
	}

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

	p.onRegister.Send(definition)
}

// EditDefinition edits the definition for the given model.
//
// This allows for easily changing certain properties of a content type definition later on.
func (p *ContentTypeRegistry) EditDefinition(def *ContentTypeDefinition) {
	var typeName = def.ContentType().TypeName()
	var oldDef = p.registry[typeName]
	if oldDef == nil {
		panic("ContentTypeRegistry: EditDefinition called for unknown type " + typeName)
	}
	if def.GetLabel != nil {
		oldDef.GetLabel = def.GetLabel
	}
	if def.GetPluralLabel != nil {
		oldDef.GetPluralLabel = def.GetPluralLabel
	}
	if def.GetDescription != nil {
		oldDef.GetDescription = def.GetDescription
	}
	if def.GetInstanceLabel != nil {
		oldDef.GetInstanceLabel = def.GetInstanceLabel
	}
	if def.GetObject != nil {
		oldDef.GetObject = def.GetObject
	}
	if def.GetInstance != nil {
		oldDef.GetInstance = def.GetInstance
	}
	if def.GetInstances != nil {
		oldDef.GetInstances = def.GetInstances
	}
	if def.GetInstancesByIDs != nil {
		oldDef.GetInstancesByIDs = def.GetInstancesByIDs
	}

	if p.aliasesRev == nil {
		p.aliasesRev = make(map[string]string)
	}

	if def.Aliases != nil {
		for _, alias := range def.Aliases {
			p.RegisterAlias(alias, typeName)
		}
	}

	p.registry[typeName] = oldDef
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
			return strings.Compare(a.Description(context.Background()), b.Description(context.Background()))
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
func (p *ContentTypeRegistry) DefinitionForObject(object any) *ContentTypeDefinition {
	var typeName = NewContentType(object).TypeName()
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

func (p *ContentTypeRegistry) GetInstance(ctx context.Context, typeName string, id interface{}) (interface{}, error) {
	var definition = p.DefinitionForType(typeName)
	if definition == nil {
		return nil, fmt.Errorf("ContentTypeRegistry: GetInstance called for unknown type %s", typeName)
	}
	return definition.Instance(ctx, id)
}

func (p *ContentTypeRegistry) GetInstances(ctx context.Context, typeName string, amount, offset uint) ([]interface{}, error) {
	var definition = p.DefinitionForType(typeName)
	if definition == nil {
		return nil, fmt.Errorf("ContentTypeRegistry: GetInstances called for unknown type %s", typeName)
	}
	return definition.Instances(ctx, amount, offset)
}

func (p *ContentTypeRegistry) GetInstancesByIDs(ctx context.Context, typeName string, ids []interface{}) ([]interface{}, error) {
	var definition = p.DefinitionForType(typeName)
	if definition == nil {
		return nil, fmt.Errorf("ContentTypeRegistry: GetInstancesByIDs called for unknown type %s", typeName)
	}
	return definition.InstancesByIDs(ctx, ids)
}

var (
	contentTypeRegistryObject = NewContentTypeRegistry()
)

// Register registers a model with the registry.
func Register(definition *ContentTypeDefinition) *ContentTypeDefinition {
	contentTypeRegistryObject.Register(definition)
	return definition
}

// EditDefinition edits the definition for the given model.
func EditDefinition(def *ContentTypeDefinition) {
	contentTypeRegistryObject.EditDefinition(def)
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

// GetInstance returns an instance of the model by its ID.
func GetInstance(ctx context.Context, typeName string, id interface{}) (interface{}, error) {
	return contentTypeRegistryObject.GetInstance(ctx, typeName, id)
}

// GetInstances returns a list of instances of the model.
func GetInstances(ctx context.Context, typeName string, amount, offset uint) ([]interface{}, error) {
	return contentTypeRegistryObject.GetInstances(ctx, typeName, amount, offset)
}

// GetInstancesByIDs returns a list of instances of the model by a list of IDs.
//
// If the model does not implement GetInstancesByID, it will fall back to calling GetInstance for each ID.
func GetInstancesByIDs(ctx context.Context, typeName string, ids []interface{}) ([]interface{}, error) {
	return contentTypeRegistryObject.GetInstancesByIDs(ctx, typeName, ids)
}

// OnRegister allows a function to be called when a new content type is registered.
func OnRegister(fn func(def *ContentTypeDefinition)) (signals.Receiver[*ContentTypeDefinition], error) {
	return contentTypeRegistryObject.onRegister.Listen(func(s signals.Signal[*ContentTypeDefinition], ctd *ContentTypeDefinition) error {
		fn(ctd)
		return nil
	})
}
