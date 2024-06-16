package pages

import (
	"context"

	"github.com/Nigel2392/django/contrib/pages/models"
)

type Page interface {
	ID() int64
	Reference() models.PageNode
	// Specific() (Page, error)
	//Parent(update bool) (Page, error)
	//Children(update bool) ([]Page, error)
	//Ancestors() ([]Page, error)
	//Descendants() ([]Page, error)
}

type SaveablePage interface {
	Page
	Save(ctx context.Context) error
}
