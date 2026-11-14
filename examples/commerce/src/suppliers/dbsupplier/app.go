package dbsupplier

import (
	"context"
	"encoding/json"
	"html/template"
	"reflect"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/settings"
	"github.com/Nigel2392/go-django/contrib/shop/supplier"
	"github.com/Nigel2392/go-django/internal/forms"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/forms/widgets/options"
)

type BaseSetting struct {
	Type               dbtype.Type
	Default            any
	Permission         []string
	DisplayIcon        func(ctx context.Context) template.HTML
	DisplayLabel       func(ctx context.Context) string
	DisplayDescription func(ctx context.Context) string
	Field              func(ctx context.Context) forms.Field
}

func (b BaseSetting) DBType() dbtype.Type {
	return b.Type
}

func (b BaseSetting) Permissions() []string {
	return b.Permission
}

func (b BaseSetting) DefaultValue() any {
	return b.Default
}

func (b BaseSetting) Icon(ctx context.Context) template.HTML {
	if b.DisplayIcon != nil {
		return b.DisplayIcon(ctx)
	}
	return ""
}

func (b BaseSetting) Label(ctx context.Context) string {
	if b.DisplayLabel != nil {
		return b.DisplayLabel(ctx)
	}
	return ""
}

func (b BaseSetting) Description(ctx context.Context) string {
	if b.DisplayDescription != nil {
		return b.DisplayDescription(ctx)
	}
	return ""
}

func (b BaseSetting) FormField(ctx context.Context) forms.Field {
	if b.Field == nil {
		return nil
	}
	return b.Field(ctx)
}

func (b BaseSetting) ValueFromDB(ctx context.Context, val json.RawMessage) (any, error) {
	var (
		typ  = drivers.DBToDefaultGoType(b.Type)
		nval = reflect.New(typ)
	)

	err := json.Unmarshal(val, nval.Interface())
	return nval.Elem().Interface(), err
}

type DBSupplierAppConfig struct {
	*apps.DBRequiredAppConfig
}

var CONFIG *DBSupplierAppConfig

func NewAppConfig() *DBSupplierAppConfig {
	CONFIG = &DBSupplierAppConfig{
		DBRequiredAppConfig: apps.NewDBAppConfig("dbsupplier"),
	}

	CONFIG.ModelObjects = []attrs.Definer{
		&Product{},
	}

	supplier.Register(CONFIG)
	settings.RegisterGroup(0, "suppliers", settings.SettingsGroup{
		Label:       trans.S("Suppliers"),
		Description: trans.S("Supplier specific settings"),
	})
	settings.RegisterSetting(0, "suppliers", "commerce.dbsupplier.backend", BaseSetting{
		Type:               dbtype.Bool,
		DisplayLabel:       trans.S("Datebase Supplier Enabled"),
		DisplayDescription: trans.S("Wether to enable or disable the database supplier backend."),
		Field: func(ctx context.Context) forms.Field {
			return fields.BooleanField()
		},
	})
	settings.RegisterSetting(0, "suppliers", "commerce.dbsupplier.backend.storage", BaseSetting{
		Type:               dbtype.String,
		DisplayLabel:       trans.S("Datebase Supplier Storage"),
		DisplayDescription: trans.S("Which storage backend to use for the database supplier."),
		Field: func(ctx context.Context) forms.Field {
			return fields.CharField(fields.Widget(options.NewSelectInput(nil, func(ctx context.Context) []widgets.Option {
				return []widgets.Option{
					widgets.NewOption("local", trans.T(ctx, "Local"), "local"),
					widgets.NewOption("cloud", trans.T(ctx, "Cloud"), "cloud"),
				}
			})))
		},
	})
	return CONFIG
}

func (s *DBSupplierAppConfig) Ordering() int {
	return 0
}
func (s *DBSupplierAppConfig) InternalName() string {
	return s.AppName
}
func (s *DBSupplierAppConfig) Icon(ctx context.Context) template.HTML {
	return `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" fill="currentColor" class="bi bi-device-hdd" viewBox="0 0 16 16">
 	<path d="M12 2.5a.5.5 0 1 1-1 0 .5.5 0 0 1 1 0m0 11a.5.5 0 1 1-1 0 .5.5 0 0 1 1 0m-7.5.5a.5.5 0 1 0 0-1 .5.5 0 0 0 0 1M5 2.5a.5.5 0 1 1-1 0 .5.5 0 0 1 1 0M8 8a1 1 0 1 0 0-2 1 1 0 0 0 0 2"/>
 	<path d="M12 7a4 4 0 0 1-3.937 4c-.537.813-1.02 1.515-1.181 1.677a1.102 1.102 0 0 1-1.56-1.559c.1-.098.396-.314.795-.588A4 4 0 0 1 8 3a4 4 0 0 1 4 4m-1 0a3 3 0 1 0-3.891 2.865c.667-.44 1.396-.91 1.955-1.268.224-.144.483.115.34.34l-.62.96A3 3 0 0 0 11 7"/>
 	<path d="M2 2a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2zm2-1a1 1 0 0 0-1 1v12a1 1 0 0 0 1 1h8a1 1 0 0 0 1-1V2a1 1 0 0 0-1-1z"/>
</svg>`
}

func (s *DBSupplierAppConfig) Label(ctx context.Context) string {
	return trans.T(ctx, "Database Supplier")
}

func (s *DBSupplierAppConfig) Description(ctx context.Context) string {
	return trans.T(ctx, "Product storage from a database")
}

func (s *DBSupplierAppConfig) ProductType(ctx context.Context) supplier.ProductModel {
	return &Product{}
}

func (s *DBSupplierAppConfig) ProductAddPanel(ctx context.Context) []admin.Panel {
	return []admin.Panel{
		admin.TabbedPanel(
			admin.PanelTab(
				trans.S("Content"),
				admin.TitlePanel(admin.FieldPanel("Title")).
					WithOutputFields("Slug"),
				admin.FieldPanel("Slug"),
				admin.FieldPanel("Image"),
				admin.FieldPanel("Editor").Class("fullsize"),
			),
			admin.PanelTab(
				trans.S("Metadata"),
				admin.FieldPanel("Ranking"),
				admin.FieldPanel("Category"),
				admin.RowPanel(
					admin.FieldPanel("Price"),
					admin.FieldPanel("TaxRate"),
					admin.FieldPanel("DiscountPrice"),
				),
				admin.FieldPanel("Supplier"),
				admin.FieldPanel("SupplierID"),
			),
		),
	}
}

func (s *DBSupplierAppConfig) ProductEditPanel(ctx context.Context) []admin.Panel {
	return []admin.Panel{
		admin.TabbedPanel(
			admin.PanelTab(
				trans.S("Content"),
				admin.TitlePanel(admin.FieldPanel("Title")).
					WithOutputFields("Slug"),
				admin.FieldPanel("Slug"),
				admin.FieldPanel("Image"),
				admin.FieldPanel("Editor").Class("fullsize"),
			),
			admin.PanelTab(
				trans.S("Metadata"),
				admin.FieldPanel("Ranking"),
				admin.FieldPanel("Category"),
				admin.RowPanel(
					admin.FieldPanel("Price"),
					admin.FieldPanel("TaxRate"),
					admin.FieldPanel("DiscountPrice"),
				),
				admin.FieldPanel("Supplier"),
				admin.FieldPanel("SupplierID"),
			),
		),
	}
}
