package supplier

import (
	"context"
	"html/template"

	"github.com/Nigel2392/go-django/contrib/admin"
)

type SupplierValue string

func (s SupplierValue) Icon(ctx context.Context) template.HTML {
	backend, ok := Retrieve(string(s))
	if !ok {
		return ""
	}
	return backend.Icon(ctx)
}

func (s SupplierValue) Label(ctx context.Context) string {
	backend, ok := Retrieve(string(s))
	if !ok {
		return string(s)
	}
	return backend.Label(ctx)
}

func (s SupplierValue) Description(ctx context.Context) string {
	backend, ok := Retrieve(string(s))
	if !ok {
		return ""
	}
	return backend.Description(ctx)
}

func (s SupplierValue) ProductType(ctx context.Context) ProductModel {
	backend, ok := Retrieve(string(s))
	if !ok {
		return nil
	}
	return backend.ProductType(ctx)
}

func (s SupplierValue) ProductAddPanel(ctx context.Context) []admin.Panel {
	backend, ok := Retrieve(string(s))
	if !ok {
		return nil
	}
	return backend.ProductAddPanel(ctx)
}

func (s SupplierValue) ProductEditPanel(ctx context.Context) []admin.Panel {
	backend, ok := Retrieve(string(s))
	if !ok {
		return nil
	}
	return backend.ProductEditPanel(ctx)
}
