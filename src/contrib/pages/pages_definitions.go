package pages

import (
	"context"
	"net/http"

	"github.com/Nigel2392/go-django/src/contrib/admin"
	models "github.com/Nigel2392/go-django/src/contrib/pages/page_models"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
)

type PageDefinition struct {
	*contenttypes.ContentTypeDefinition
	ServePage               func() PageView
	AddPanels               func(r *http.Request, page Page) []admin.Panel
	EditPanels              func(r *http.Request, page Page) []admin.Panel
	GetForID                func(ctx context.Context, ref models.PageNode, id int64) (Page, error)
	OnReferenceUpdate       func(ctx context.Context, ref models.PageNode, id int64) error
	OnReferenceBeforeDelete func(ctx context.Context, ref models.PageNode, id int64) error
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

func (p *PageDefinition) PageView() PageView {
	if p.ServePage != nil {
		return p.ServePage()
	}
	return nil
}
