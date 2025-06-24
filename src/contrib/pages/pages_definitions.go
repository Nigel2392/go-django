package pages

import (
	"context"
	"net/http"

	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
)

type PageDefinition struct {
	// The underlying content type definition for the blog page model
	*contenttypes.ContentTypeDefinition

	// Serve the page with this view instead
	ServePage func(page Page) PageView

	// Panels for the model when creating a new page
	//
	// This contains fields from the custom model, as well as the underlying page node model.
	AddPanels func(r *http.Request, page Page) []admin.Panel

	// Panels for the model when editing a page
	//
	// This contains fields from the custom model, as well as the underlying page node model.
	EditPanels func(r *http.Request, page Page) []admin.Panel

	// Query for an instance of this model by its ID
	GetForID func(ctx context.Context, ref *PageNode, id int64) (Page, error)

	// Callback function to be called when a reference node is updated
	OnReferenceUpdate func(ctx context.Context, ref *PageNode, id int64) error

	// Callback function to be called before a reference node is deleted
	OnReferenceBeforeDelete func(ctx context.Context, ref *PageNode, id int64) error

	// Maximum number of pages allowed for this model
	// MaxNum          int

	// Disallow creation of this model through the admin interface
	DissallowCreate bool

	// Disallow this page type to be a root page (i.e. it must have a parent)
	DisallowRoot bool

	// Allowed parent page types for this model
	//
	// This is a list of content type strings that this model can be a child of.
	ParentPageTypes []string

	// Allowed child page types for this model
	//
	// This is a list of content type strings that this model can be a parent of.
	ChildPageTypes []string

	_parentPageTypes map[string]struct{}
	_childPageTypes  map[string]struct{}
}

func (p *PageDefinition) Label() string {
	if p.GetLabel != nil {
		return p.GetLabel()
	}
	return ""
}

func (p *PageDefinition) Description() string {
	if p.GetDescription != nil {
		return p.GetDescription()
	}
	return ""
}

func (p *PageDefinition) ContentType() contenttypes.ContentType {
	if p.ContentTypeDefinition == nil {
		return nil
	}
	return p.ContentTypeDefinition.ContentType()
}

func (p *PageDefinition) AppLabel() string {
	return p.ContentType().AppLabel()
}

func (p *PageDefinition) Model() string {
	return p.ContentType().Model()
}

func (p *PageDefinition) PageView(page Page) PageView {
	if p.ServePage != nil {
		return p.ServePage(page)
	}
	return nil
}
