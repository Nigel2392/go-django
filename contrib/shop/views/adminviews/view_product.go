package adminviews

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/admin/components"
	"github.com/Nigel2392/go-django/contrib/filters"
	"github.com/Nigel2392/go-django/contrib/messages"
	"github.com/Nigel2392/go-django/contrib/shop/internal/app"
	"github.com/Nigel2392/go-django/contrib/shop/models"
	"github.com/Nigel2392/go-django/contrib/shop/util/forms"
	"github.com/Nigel2392/go-django/contrib/shop/util/signals"
	"github.com/Nigel2392/go-django/contrib/shop/views/adminviews/generic"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	djmodels "github.com/Nigel2392/go-django/src/models"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-django/src/views/list"
	"github.com/Nigel2392/mux"
	"github.com/shopspring/decimal"
)

var ViewProductList = generic.ListViewConfig[*models.Product]{
	ListTitle: trans.S("Product List"),
	Model:     &models.Product{},
	PerPage:   50,
	GetEditLink: func(r *http.Request, v *models.Product) string {
		return django.Reverse("admin:shop:products:edit", v.ID)
	},
	GetQuerySet: func(r *http.Request) *queries.QuerySet[*models.Product] {
		return queries.GetQuerySet(&models.Product{}).Select("*").Annotate(
			"LowestStock", queries.Subquery(queries.GetQuerySet(&models.ProductSku{}).
				Select(expr.MIN("Stock")).
				Filter("Product.ID", expr.OuterRef("ID")).
				OrderBy("Stock").
				Limit(1)),
		)
	},
	Filters: []filters.FilterSpec[*models.Product]{
		&filters.BaseFilterSpec[*queries.QuerySet[*models.Product]]{
			SpecName:  "search",
			FormField: fields.CharField(fields.HelpText(trans.S("Search by title or URL path"))),
			Apply: func(req *http.Request, value interface{}, object *queries.QuerySet[*models.Product]) (*queries.QuerySet[*models.Product], error) {
				if fields.IsZero(value) {
					return object, nil
				}

				return object.Filter(expr.Or(
					expr.Q("Title__icontains", value),
					expr.Q("Slug__icontains", value),
					expr.Q("Skus.Title__icontains", value),
				)), nil
			},
		},
	},
	GetHeaderActions: func(r *http.Request) []components.ShowableComponent {
		var actions = make([]components.ShowableComponent, 0)
		actions = append(actions, components.LinkConfig{
			Text: trans.S("Add new product"),
			Type: components.ClassTypeSecondary,
			URL: func(ctx context.Context) string {
				return django.Reverse("admin:shop:products:add")
			},
		})
		return actions
	},
	GetColumns: func(r *http.Request) ([]list.ListColumn[*models.Product], error) {
		var cols = []list.ListColumn[*models.Product]{
			list.FieldColumn[*models.Product](
				trans.S("Product ID"),
				"ID",
			),
			list.FieldColumn[*models.Product](
				trans.S("Title"),
				"Title",
			),
			list.FieldColumn[*models.Product](
				trans.S("Skus"),
				"SkuCount",
			),
			list.HTMLFieldColumn[*models.Product](
				trans.S("Lowest Stock"),
				"LowestStock",
				func(r *http.Request, defs attrs.Definitions, row *models.Product) template.HTML {
					return template.HTML(strconv.Itoa(row.LowestStock))
				},
			),
			list.HTMLFieldColumn(
				trans.S("Price (max)"),
				"PriceMax",
				func(r *http.Request, defs attrs.Definitions, row *models.Product) template.HTML {
					return template.HTML(fmt.Sprintf("€ %s", row.PriceMax))
				},
			),
		}

		return cols, nil
	},
}

type productSkuFormset interface {
	Forms() ([]modelforms.ModelForm[*models.ProductSku], error)
}

func ViewAddProduct(w http.ResponseWriter, r *http.Request, shop *app.ShopAppConfig) {
	if !permissions.HasObjectPermission(r, &models.Product{}, "products:add") {
		admin.ReLogin(w, r, r.URL.Path)
		return
	}

	var form = forms.NewAdminForm(r, &models.Product{}, func(p *models.Product) []admin.Panel {
		return p.AddPanels(r.Context())
	})

	form.Load()

	form.Form.SaveInstance = func(ctx context.Context, p *models.Product) error {

		flist, err := form.FormSet().Forms()
		if err != nil {
			return err
		}

		var (
			skuFormSet productSkuFormset
		)

		for _, form := range flist {
			switch f := form.(type) {
			case productSkuFormset:
				skuFormSet = f
			default:
				except.Fail(
					http.StatusInternalServerError,
					"unhandled form type",
				)
			}
		}

		skuForms, err := skuFormSet.Forms()
		if err != nil {
			return err
		}

		var priceMax decimal.Decimal
		var skus = make([]*models.ProductSku, 0, len(skuForms))
		for _, form := range skuForms {

			sku := form.Instance()
			skus = append(skus, sku)

			if sku.Price.GreaterThan(priceMax) {
				priceMax = sku.Price
			}

			form.CleanedData()["Product"] = p
		}

		p.SkuCount = len(skus)
		p.PriceMax = priceMax

		saved, err := djmodels.SaveModel(ctx, p)
		if err == nil && !saved {
			err = fmt.Errorf("model %T not saved", p)
		}

		if _, err := form.FormSet().Save(); err != nil {
			return err
		}

		return shop.SIGNALS.Products.Created.Send(&signals.ProductSignalData{
			BaseSignal: signals.BaseSignal{Context: r.Context()},
			Product:    p,
			Skus:       skus,
		})
	}

	var view = &views.FormView[*admin.AdminForm[*modelforms.BaseModelForm[*models.Product], *models.Product]]{
		BaseView: views.BaseView{
			AllowedMethods:  []string{http.MethodGet, http.MethodPost},
			BaseTemplateKey: "shop",
			TemplateName:    "shop/products/admin/add.tmpl",
			GetContextFn: func(req *http.Request) (ctx.Context, error) {
				var context = admin.NewContext(req, admin.AdminSite, nil)

				var backURL string
				if q := req.URL.Query().Get("next"); q != "" {
					backURL = q
				}

				context.Set("BackURL", backURL)
				context.Set("PostURL", django.Reverse("admin:shop:products:add"))
				context.SetPage(admin.PageOptions{
					TitleFn: trans.S("Add new product"),
				})

				return context, nil
			},
		},
		GetFormFn: func(req *http.Request) *admin.AdminForm[*modelforms.BaseModelForm[*models.Product], *models.Product] {
			return form
		},
		GetInitialFn: func(req *http.Request) map[string]interface{} {
			var initial = make(map[string]interface{})
			for _, field := range attrs.Define(r.Context(), &models.Product{}).Fields() {
				initial[field.Name()] = field.GetDefault()
			}
			return initial
		},
		SuccessFn: func(w http.ResponseWriter, req *http.Request, form *admin.AdminForm[*modelforms.BaseModelForm[*models.Product], *models.Product]) {
			var instance = form.Instance()
			except.Assert(
				instance != nil,
				http.StatusInternalServerError,
				"instance is nil after form submission",
			)

			messages.Success(r, "Product added")

			http.Redirect(w, r, django.Reverse("admin:shop:products"), http.StatusSeeOther)
		},
	}

	if err := views.Invoke(view, w, r); err != nil {
		except.Fail(500, err)
		return
	}
}

func ViewEditProduct(w http.ResponseWriter, r *http.Request, shop *app.ShopAppConfig) {
	if !permissions.HasObjectPermission(r, &models.Product{}, "products:edit") {
		admin.ReLogin(w, r, r.URL.Path)
		return
	}

	var vars = mux.Vars(r)
	var productRow, err = queries.GetQuerySet(&models.Product{}).
		Select("*", "Skus.*").
		Filter("ID", vars.GetInt("product_id")).
		Get()

	if errors.Is(err, errors.NoRows) {
		except.RaiseNotFound("No product found.")
		return
	}

	if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"failed to retrieve product: %v", err,
		)
	}

	var form = forms.NewAdminForm(r, productRow.Object, func(p *models.Product) []admin.Panel {
		return p.EditPanels(r.Context())
	})

	form.Load()

	form.Form.SaveInstance = func(ctx context.Context, p *models.Product) error {

		flist, err := form.FormSet().Forms()
		if err != nil {
			return err
		}

		var (
			skuFormSet productSkuFormset
		)

		for _, form := range flist {
			switch f := form.(type) {
			case productSkuFormset:
				skuFormSet = f
			default:
				panic("unhandled form type")
			}
		}

		skuForms, err := skuFormSet.Forms()
		if err != nil {
			return err
		}

		var priceMax decimal.Decimal
		var skus = make([]*models.ProductSku, 0, len(skuForms))
		for _, form := range skuForms {

			sku := form.Instance()
			skus = append(skus, sku)

			if sku.Price.GreaterThan(priceMax) {
				priceMax = sku.Price
			}

			form.CleanedData()["Product"] = p
		}

		p.SkuCount = len(skus)
		p.PriceMax = priceMax

		saved, err := djmodels.SaveModel(ctx, p)
		if err == nil && !saved {
			err = fmt.Errorf("model %T not saved", p)
		}

		if _, err := form.FormSet().Save(); err != nil {
			return err
		}

		return shop.SIGNALS.Products.Updated.Send(&signals.ProductSignalData{
			BaseSignal: signals.BaseSignal{Context: r.Context()},
			Product:    p,
			Skus:       skus,
		})

	}

	var view = &views.FormView[*admin.AdminForm[*modelforms.BaseModelForm[*models.Product], *models.Product]]{
		BaseView: views.BaseView{
			AllowedMethods:  []string{http.MethodGet, http.MethodPost},
			BaseTemplateKey: "shop",
			TemplateName:    "shop/products/admin/edit.tmpl",
			GetContextFn: func(req *http.Request) (ctx.Context, error) {
				var context = admin.NewContext(req, admin.AdminSite, nil)

				var backURL string
				if q := req.URL.Query().Get("next"); q != "" {
					backURL = q
				}

				context.Set("BackURL", backURL)
				context.Set("PostURL", django.Reverse("admin:shop:products:edit", productRow.Object.ID))
				context.SetPage(admin.PageOptions{
					TitleFn: trans.S("Change product"),
				})

				return context, nil
			},
		},
		GetFormFn: func(req *http.Request) *admin.AdminForm[*modelforms.BaseModelForm[*models.Product], *models.Product] {
			return form
		},
		GetInitialFn: func(req *http.Request) map[string]interface{} {
			var initial = make(map[string]interface{})
			for _, field := range attrs.Define(r.Context(), productRow.Object).Fields() {
				initial[field.Name()] = field.GetValue()
			}
			return initial
		},
		SuccessFn: func(w http.ResponseWriter, req *http.Request, form *admin.AdminForm[*modelforms.BaseModelForm[*models.Product], *models.Product]) {
			var instance = form.Instance()
			except.Assert(
				instance != nil,
				http.StatusInternalServerError,
				"instance is nil after form submission",
			)

			messages.Success(r, "Product changed")

			http.Redirect(w, r, django.Reverse("admin:shop:products"), http.StatusSeeOther)
		},
	}

	if err := views.Invoke(view, w, r); err != nil {
		except.Fail(500, err)
		return
	}
}
