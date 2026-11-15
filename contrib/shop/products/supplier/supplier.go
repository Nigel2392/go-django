package supplier

import (
	"context"
	"html/template"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/queries/src/specific"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/shopspring/decimal"
	"golang.org/x/text/currency"
)

type Price struct {
	Unit   currency.Unit
	Amount decimal.Decimal
}

type ProductModel interface {
	specific.Model
	ProductPrice(ctx context.Context) Price
}

type Supplier struct {
	Order           int
	Name            string
	HTMLIcon        func(ctx context.Context) template.HTML
	TextLabel       func(ctx context.Context) string
	TextDescription func(ctx context.Context) string
	ProductModel    func(ctx context.Context) ProductModel
	AddPanel        func(ctx context.Context) []admin.Panel
	EditPanel       func(ctx context.Context) []admin.Panel
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

func (s *Supplier) ProductType(ctx context.Context) ProductModel {
	if s.ProductModel == nil {
		return nil
	}
	return s.ProductModel(ctx)
}

func (s *Supplier) ProductAddPanel(ctx context.Context) []admin.Panel {
	if s.AddPanel == nil {
		return nil
	}
	return s.AddPanel(ctx)
}

func (s *Supplier) ProductEditPanel(ctx context.Context) []admin.Panel {
	if s.EditPanel == nil {
		return nil
	}
	return s.EditPanel(ctx)
}
