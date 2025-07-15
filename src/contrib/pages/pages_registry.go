package pages

import (
	"context"
	"slices"
	"strings"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-signals"
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

	if definition.ContentObject == nil {
		panic("pages: RegisterPageDefinition definition is missing PageObject")
	}

	if definition.ContentTypeDefinition == nil {
		var cType = definition.ContentType()
		definition.ContentTypeDefinition = contenttypes.DefinitionForType(cType.TypeName())
	}

	var contentType = definition.ContentType()
	var typeName = contentType.TypeName()
	if _, exists := p.registry[typeName]; exists {
		panic("pages: RegisterPageDefinition called twice for " + typeName)
	}

	if definition._childPageTypes == nil {
		definition._childPageTypes = make(map[string]struct{})
		for _, childType := range definition.ChildPageTypes {
			definition._childPageTypes[childType] = struct{}{}
		}
	}

	if definition.OnReferenceUpdate != nil {
		SignalNodeUpdated.Listen(func(s signals.Signal[*PageNodeSignal], ps *PageNodeSignal) error {
			if ps.Node.ContentType == typeName {
				return definition.OnReferenceUpdate(ps.Ctx, ps.Node, ps.PageID)
			}
			return nil
		})
	}

	if definition.OnReferenceBeforeDelete != nil {
		SignalNodeBeforeDelete.Listen(func(s signals.Signal[*PageNodeSignal], ps *PageNodeSignal) error {
			if ps.Node.ContentType == typeName {
				return definition.OnReferenceBeforeDelete(ps.Ctx, ps.Node, ps.PageID)
			}
			return nil
		})
	}

	p.registry[typeName] = definition
	contenttypes.Register(definition.ContentTypeDefinition)
}

func sortDefinitions(definitions []*PageDefinition) {
	slices.SortStableFunc(definitions, func(a, b *PageDefinition) int {
		var result = strings.Compare(a.ContentType().Model(), b.ContentType().Model())
		if result == 0 {
			return strings.Compare(a.Description(), b.Description())
		}
		return result
	})
}

func FilterCreatableDefinitions(definitions []*PageDefinition) []*PageDefinition {
	var creatable = make([]*PageDefinition, 0, len(definitions))
	for _, definition := range definitions {
		if !definition.DissallowCreate {
			creatable = append(creatable, definition)
		}
	}
	return creatable
}

func (p *pageRegistry) ListDefinitions() []*PageDefinition {
	var definitions = make([]*PageDefinition, 0, len(p.registry))
	for _, definition := range p.registry {
		definitions = append(definitions, definition)
	}
	sortDefinitions(definitions)
	return definitions
}

func (p *pageRegistry) ListRootDefinitions() []*PageDefinition {
	var definitions = make([]*PageDefinition, 0, len(p.registry))
	for _, definition := range p.registry {
		if !definition.DisallowRoot {
			definitions = append(definitions, definition)
		}
	}
	sortDefinitions(definitions)
	return definitions
}

func (p *pageRegistry) ListDefinitionsForType(typeName string) []*PageDefinition {
	var definitions = make([]*PageDefinition, 0, len(p.registry))
	var definition = p.registry[typeName]
	if definition == nil {
		return definitions
	}

	if len(definition._childPageTypes) > 0 {
		for _, def := range p.registry {
			if _, exists := definition._childPageTypes[def.ContentType().TypeName()]; exists {
				if len(def._parentPageTypes) > 0 {
					if _, exists := def._parentPageTypes[typeName]; exists {
						definitions = append(definitions, def)
					}
				} else {
					definitions = append(definitions, def)
				}
			}
		}
	} else {
		for _, def := range p.registry {
			definitions = append(definitions, def)
		}
	}

	sortDefinitions(definitions)

	return definitions
}

func (p *pageRegistry) DefinitionForType(typeName string) *PageDefinition {
	typeName = contenttypes.ReverseAlias(typeName)
	return p.registry[typeName]
}

func (p *pageRegistry) DefinitionForObject(page Page) *PageDefinition {
	var typeName = contenttypes.NewContentType(page).TypeName()
	return p.registry[typeName]
}

func (p *pageRegistry) SpecificInstance(ctx context.Context, node *PageNode) (Page, error) {
	var typeName = node.ContentType
	var definition = p.DefinitionForType(
		typeName,
	)
	if definition == nil {
		return node, nil
		// return nil, errors.Wrapf(
		// ErrContentTypeInvalid, "Page type %s not found", typeName,
		// )
	}

	if node.PageID == 0 {
		return node, ErrNoPageID.Wrapf(
			"pages: SpecificInstance called with node %s that has no PageID (%q)",
			typeName, node.Title,
		)
		//panic(fmt.Sprintf(
		//	"pages: SpecificInstance called with node %s that has no PageID (%+v)",
		//	typeName, node,
		//))
	}

	if definition.GetForID == nil {
		var newObject = attrs.NewObject[Page](definition.ContentObject)
		var meta = attrs.GetModelMeta(newObject)
		var defs = meta.Definitions()
		var querySet = queries.GetQuerySet(newObject).
			WithContext(ctx).
			Filter(defs.Primary().Name(), node.PageID)

		var row, err = querySet.Get()
		if err != nil {
			return nil, ErrNoPage.Wrapf(
				"pages: SpecificInstance failed to get page for node %s with PageID %d: %v",
				typeName, node.PageID, err,
			)
		}
		return row.Object, nil
	}

	return definition.GetForID(ctx, node, node.PageID)
}
