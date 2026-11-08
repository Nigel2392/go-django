package supplier

import (
	"context"
	"html/template"

	"github.com/Nigel2392/go-django/contrib/blocks"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

type Supplier struct {
	Order           int
	Name            string
	HTMLIcon        func(ctx context.Context) template.HTML
	TextLabel       func(ctx context.Context) string
	TextDescription func(ctx context.Context) string
	ProductBlock    func() *blocks.StructBlock
}

// lowest first, defaults to InternalName otherwise
func (s *Supplier) Ordering() int {
	return s.Order
}

func (s *Supplier) InternalName() string {
	return s.Name
}

func (s *Supplier) Icon(ctx context.Context) template.HTML {
	if s.HTMLIcon == nil {
		return ""
	}
	return s.HTMLIcon(ctx)
}

func (s *Supplier) Label(ctx context.Context) string {
	if s.TextLabel == nil {
		return attrs.NiceName(s.Name)
	}
	return s.TextLabel(ctx)
}

func (s *Supplier) Description(ctx context.Context) string {
	if s.TextDescription == nil {
		return ""
	}
	return s.TextDescription(ctx)
}

func (s *Supplier) ProductWidget() *blocks.StructBlock {
	if s.ProductBlock == nil {
		return nil
	}
	return s.ProductBlock()
}
