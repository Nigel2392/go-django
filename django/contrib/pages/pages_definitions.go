package pages

import (
	"context"

	"github.com/Nigel2392/django/contrib/pages/models"
)

type PageDefinition struct {
	PageObject SaveablePage
	GetForID   func(ctx context.Context, ref models.PageNode, id int64) (SaveablePage, error)
}

func (p *PageDefinition) ContentType() *ContentType {
	return NewContentType(p.PageObject)
}
