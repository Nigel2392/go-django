package pages

import (
	"context"
	"slices"
	"strings"

	"github.com/Nigel2392/django/contrib/pages/models"
	"github.com/Nigel2392/django/core/contenttypes"
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

	if definition.ContentObject == nil || definition.GetForID == nil {
		panic("pages: RegisterPageDefinition definition is missing PageObject or GetForID")
	}

	if definition.ContentTypeDefinition == nil {
		var cType = definition.ContentType()
		definition.ContentTypeDefinition = contenttypes.DefinitionForType(cType.TypeName())
	}

	if definition.GetLabel == nil {
		panic("pages: RegisterPageDefinition definition is missing GetLabel")
	}

	var contentType = definition.ContentType()
	var typeName = contentType.TypeName()
	if _, exists := p.registry[typeName]; exists {
		panic("pages: RegisterPageDefinition called twice for " + typeName)
	}

	if definition.OnReferenceUpdate != nil {
		SignalNodeUpdated.Listen(func(s signals.Signal[*PageSignal], ps *PageSignal) error {
			if ps.Node.ContentType == typeName {
				return definition.OnReferenceUpdate(ps.Ctx, *ps.Node, ps.PageID)
			}
			return nil
		})
	}

	if definition.OnReferenceBeforeDelete != nil {
		SignalNodeBeforeDelete.Listen(func(s signals.Signal[*PageSignal], ps *PageSignal) error {
			if ps.Node.ContentType == typeName {
				return definition.OnReferenceBeforeDelete(ps.Ctx, *ps.Node, ps.PageID)
			}
			return nil
		})
	}

	p.registry[typeName] = definition
	contenttypes.Register(definition.ContentTypeDefinition)
}

func (p *pageRegistry) ListDefinitions() []*PageDefinition {
	var definitions = make([]*PageDefinition, 0, len(p.registry))
	for _, definition := range p.registry {
		definitions = append(definitions, definition)
	}
	slices.SortStableFunc(definitions, func(a, b *PageDefinition) int {
		var result = strings.Compare(a.ContentType().Model(), b.ContentType().Model())
		if result == 0 {
			return strings.Compare(a.Description(), b.Description())
		}
		return result
	})
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

func (p *pageRegistry) SpecificInstance(ctx context.Context, node models.PageNode) (Page, error) {
	var typeName = node.ContentType
	var definition = p.DefinitionForType(
		typeName,
	)
	if definition == nil {
		return &node, nil
		// return nil, errors.Wrapf(
		// ErrContentTypeInvalid, "Page type %s not found", typeName,
		// )
	}

	return definition.GetForID(ctx, node, node.PageID)
}
