package pages

import (
	"context"

	"github.com/Nigel2392/django/contrib/pages/models"
)

type PageDefinition struct {
	PageObject              Page
	GetForID                func(ctx context.Context, ref models.PageNode, id int64) (Page, error)
	OnReferenceUpdate       func(ctx context.Context, ref models.PageNode, id int64) error
	OnReferenceBeforeDelete func(ctx context.Context, ref models.PageNode, id int64) error
}

func (p *PageDefinition) ContentType() *ContentType {
	return NewContentType(p.PageObject)
}
