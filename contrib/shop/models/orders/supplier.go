package orders

import (
	"context"
	"html/template"
	"reflect"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/blocks"
	"github.com/Nigel2392/go-django/contrib/shop/models/products"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/forms/widgets/options"
)

func init() {
	attrs.RegisterFormFieldType(reflect.TypeFor[products.SupplierValue](), func(opts ...func(fields.Field)) fields.Field {
		var widget = options.NewSelectInput(nil, func(ctx context.Context) []widgets.Option {
			var wo = make([]widgets.Option, 0)
			for _, backend := range IterSupplierBackends(true) {
				wo = append(wo, widgets.NewOption(backend.InternalName(), backend.Label(ctx), backend.InternalName()))
			}
			return wo
		})

		return fields.CharField(append(opts, fields.Widget(widget))...)
	})

}

type Supplier struct {
	Order           int
	Name            string
	HTMLIcon        func(ctx context.Context) template.HTML
	TextLabel       func(ctx context.Context) string
	TextDescription func(ctx context.Context) string
	ProductModel    func(ctx context.Context) products.ProductModel
	AddPanel        func(ctx context.Context) []admin.Panel
	EditPanel       func(ctx context.Context) []admin.Panel
	OrderBlockType  func(ctx context.Context, order *Order, orderItem *OrderItem) (blocks.Block, error)
	OrderIsAccepted func(ctx context.Context, order *Order, orderItem *OrderItem) (*blocks.StructBlockValue, error)
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

func (s *Supplier) ProductType(ctx context.Context) products.ProductModel {
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

func (s *Supplier) OrderBlock(ctx context.Context, order *Order, orderItem *OrderItem) (blocks.Block, error) {
	if s.OrderBlockType == nil {
		return nil, nil
	}
	return s.OrderBlockType(ctx, order, orderItem)
}

func (s *Supplier) CompleteOrderItem(ctx context.Context, order *Order, orderItem *OrderItem) (*blocks.StructBlockValue, error) {
	if s.OrderIsAccepted == nil {
		return nil, nil
	}
	return s.OrderIsAccepted(ctx, order, orderItem)
}
