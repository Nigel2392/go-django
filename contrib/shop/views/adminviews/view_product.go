package adminviews

import (
	"cmp"
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/admin/components"
	"github.com/Nigel2392/go-django/contrib/admin/icons"
	"github.com/Nigel2392/go-django/contrib/filters"
	"github.com/Nigel2392/go-django/contrib/messages"
	"github.com/Nigel2392/go-django/contrib/shop/internal/app"
	"github.com/Nigel2392/go-django/contrib/shop/models"
	"github.com/Nigel2392/go-django/contrib/shop/supplier"
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
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	djmodels "github.com/Nigel2392/go-django/src/models"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-django/src/views/list"
	gosigs "github.com/Nigel2392/go-signals"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/versatile/clone"
)

var ViewProductList = generic.ListViewConfig[*models.Product]{
	ListTitle: trans.S("Product List"),
	Model:     &models.Product{},
	Permissions: []string{
		"products:list",
	},
	GetEditLink: func(r *http.Request, v *models.Product) string {
		return django.Reverse("admin:apps:model:edit", "shop", "products", v.ID)
	},
	GetQuerySet: func(r *http.Request) *queries.QuerySet[*models.Product] {
		return queries.GetQuerySet(&models.Product{}).SelectRelated("Category")
	},
	Filters: []filters.FilterSpec[*models.Product]{
		&filters.BaseFilterSpec[*queries.QuerySet[*models.Product]]{
			SpecName: "search",
			FormField: fields.CharField(
				fields.Placeholder("Search..."),
			),
			Apply: func(req *http.Request, value interface{}, object *queries.QuerySet[*models.Product]) (*queries.QuerySet[*models.Product], error) {
				if fields.IsZero(value) {
					return object, nil
				}

				return object.Distinct().Filter(expr.Or(
					expr.Q("Title__icontains", value),
					expr.Q("Slug__icontains", value),
					expr.Q("Supplier__icontains", value),
					expr.Q("Category.Name__icontains", value),
				)), nil
			},
		},
		//	generic.FilterSpecAmount[*models.Product]("amount", AMOUNT_OPTIONS,
		//		generic.OptFilterSpecApplyAmount(false),
		//	),
	},
	GetHeaderActions: func(r *http.Request) []components.ShowableComponent {

		var amount, _ = strconv.Atoi(r.URL.Query().Get("amount"))
		if amount < 1 {
			amount = 25
		}

		var actions = make([]components.ShowableComponent, 0)
		actions = append(actions, generic.AmountHeaderAction(
			r, amount, "admin:apps:model", nil, "shop", "products",
		))

		actions = append(actions, components.LinkConfig{
			Text: trans.S("Add"),
			Icon: icons.Icon("icon-plus", map[string]string{}),
			Type: components.ClassTypeSecondary,
			URL: func(ctx context.Context) string {
				return admin.ModelURL(&models.Product{}, []string{"add"}) // django.Reverse("admin:apps:model:add", "shop", "products")
			},
		})

		return actions
	},
	GetColumns: func(r *http.Request) ([]list.ListColumn[*models.Product], error) {
		var cols = []list.ListColumn[*models.Product]{
			list.FieldColumn[*models.Product](
				trans.S("Title"),
				"Title",
			),
			list.FieldColumn[*models.Product](
				trans.S("Product ID"),
				"ID",
			),
			list.FieldColumn[*models.Product](
				trans.S("Price"),
				"Price",
			),
			list.FieldColumn[*models.Product](
				trans.S("Discount"),
				"DiscountPrice",
			),
			list.FieldColumn[*models.Product](
				trans.S("Category"),
				"Category",
			),
			list.FieldColumn[*models.Product](
				trans.S("Supplier"),
				"Supplier",
			),
		}

		return cols, nil
	},
}

func ViewAddNewProductSupplier(w http.ResponseWriter, r *http.Request, shop *app.ShopAppConfig) {
	if !permissions.HasObjectPermission(r, &models.Product{}, "products:add") {
		admin.ReLogin(w, r, r.URL.Path)
		return
	}

	qs := supplier.Suppliers().
		WithContext(r.Context()).
		OrderBy("Enabled")

	rows, err := qs.All()
	if errors.Is(err, errors.NoRows) {
		except.RaiseNotFound("No suppliers found.")
		return
	}
	if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"failed to retrieve suppliers: %v", err,
		)
	}

	bound := make([]supplier.BoundBackend, 0, len(rows))
	for _, row := range rows {
		backend, ok := supplier.Retrieve(row.Object.InternalName)
		if !ok {
			backend = &supplier.Supplier{
				Name:      row.Object.InternalName,
				TextLabel: trans.S(row.Object.InternalName),
			}
		}

		bound = append(bound, supplier.BoundBackend{
			Context:  r.Context(),
			Instance: row.Object,
			Backend:  backend,
		})
	}

	slices.SortStableFunc(bound, func(a, b supplier.BoundBackend) int {
		if a.Instance.Enabled != b.Instance.Enabled {
			if a.Instance.Enabled {
				return -1
			}
			return 1
		}

		return cmp.Or(
			cmp.Compare(a.Ordering(), b.Ordering()),
			cmp.Compare(a.InternalName(), b.InternalName()),
		)
	})

	var (
		backURL = django.Reverse("admin:apps:model", "shop", "products")
		data    = admin.NewContext(r, admin.AdminSite, nil)
	)

	if q := r.URL.Query().Get("next"); q != "" {
		backURL = q
	}

	data.Set("definitions", bound)
	data.Set("BackURL", backURL)
	data.SetPage(admin.PageOptions{
		TitleFn:    trans.S("Choose product supplier"),
		SubtitleFn: trans.S("Sets up any extra fields that the supplier might require on a per-product basis"),
		BreadCrumbs: []admin.BreadCrumb{
			{
				Title: trans.T(r.Context(), "Products"),
				URL:   django.Reverse("admin:apps:model", "shop", "products"),
			},
		},
	})

	if err := tpl.FRender(w, data, "shop", "shop/products/admin/suppliers.tmpl"); err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"failed to render suppliers: %v", err,
		)
	}
}

func ViewAddProduct(w http.ResponseWriter, r *http.Request, shop *app.ShopAppConfig) {
	if !permissions.HasObjectPermission(r, &models.Product{}, "products:add") {
		admin.ReLogin(w, r, r.URL.Path)
		return
	}

	var supplierVal = supplier.SupplierValue(mux.Vars(r).Get("supplier"))
	var model = supplierVal.ProductType(r.Context())
	var form = forms.NewAdminForm(r, model, func(p supplier.ProductModel) []admin.Panel {
		return supplierVal.ProductAddPanel(r.Context())
	})

	form.Load()

	transactionCtx, tx, err := queries.StartTransaction(r.Context())
	if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"failed to start transaction",
		)
	}

	defer tx.Rollback(transactionCtx)
	r = r.WithContext(transactionCtx)

	form.Form.SaveInstance = func(ctx context.Context, p supplier.ProductModel) error {
		saved, err := djmodels.SaveModel(ctx, p)
		if err == nil && !saved {
			err = fmt.Errorf("model %T not saved", p)
		}

		if err != nil {
			return err
		}

		product := p.SpecificObject().Embedder().(*models.Product)
		product.Supplier = supplierVal

		_, err = queries.GetQuerySet[attrs.Definer](&models.Product{}).
			ExplicitSave().
			Create(product)

		if err != nil {
			return err
		}

		return nil
	}

	var view = &views.FormView[*admin.AdminForm[*modelforms.BaseModelForm[supplier.ProductModel], supplier.ProductModel]]{
		AllowedMethods:  []string{http.MethodGet, http.MethodPost},
		BaseTemplateKey: "shop",
		TemplateName:    "shop/products/admin/add.tmpl",
		GetContextFn: func(req *http.Request) (ctx.Context, error) {
			var context = admin.NewContext(req, admin.AdminSite, nil)
			var backURL string
			if q := req.URL.Query().Get("next"); q != "" {
				backURL = q
			}

			context.Set("supplier", supplierVal)
			context.Set("BackURL", backURL)
			context.Set("PostURL", django.Reverse("admin:commerce:new_with_supplier", supplierVal))
			context.SetPage(admin.PageOptions{
				TitleFn: trans.S("Add new product"),
			})

			return context, nil
		},
		GetFormFn: func(req *http.Request) *admin.AdminForm[*modelforms.BaseModelForm[supplier.ProductModel], supplier.ProductModel] {
			return form
		},
		GetInitialFn: func(req *http.Request) map[string]interface{} {
			var initial = make(map[string]interface{})
			for _, field := range attrs.Define(r.Context(), &models.Product{}).Fields() {
				initial[field.Name()] = field.GetDefault()
			}
			initial["Supplier"] = supplierVal
			return initial
		},
		SuccessFn: func(w http.ResponseWriter, req *http.Request, form *admin.AdminForm[*modelforms.BaseModelForm[supplier.ProductModel], supplier.ProductModel]) {
			var instance = form.Instance()
			except.Assert(
				instance != nil,
				http.StatusInternalServerError,
				"instance is nil after form submission",
			)

			err := shop.SIGNALS.Products.Created.Send(r.Context(), &signals.ProductSignalData{
				BaseSignal: signals.NewBaseSignal(req.Context(), r, nil, nil),
				Product: signals.Changed[*models.Product]{
					New: instance.SpecificObject().Embedder().(*models.Product),
				},
				Specific: signals.Changed[supplier.ProductModel]{
					New: instance,
				},
			})
			if err != nil {
				e, ok := gosigs.SignalError(err)
				if !ok || len(e.Related) != 1 {
					except.Fail(
						http.StatusInternalServerError,
						"error while sending shop product created signal: %v", err,
					)
				}

				except.Fail(
					http.StatusInternalServerError,
					"error while sending shop product created signal: %v", e.Related[0],
				)
			}

			if err = tx.Commit(transactionCtx); err != nil {
				except.Fail(
					http.StatusInternalServerError,
					"failed to commit transaction: %v", err,
				)
			}

			messages.Success(r, "Product added")

			http.Redirect(w, r, django.Reverse("admin:apps:model", "shop", "products"), http.StatusSeeOther)

		},
	}

	if err := views.Invoke(view, w, r); err != nil {
		except.Fail(500, err)
		return
	}
}

func ViewEditProduct(w http.ResponseWriter, r *http.Request, shop *app.ShopAppConfig, instance *models.Product) {
	if !permissions.HasObjectPermission(r, &models.Product{}, "products:edit") {
		admin.ReLogin(w, r, r.URL.Path)
		return
	}

	specific, err := instance.Specific.Specific(r.Context(), true)
	if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"error retrieving specific instance of product: %v", err,
		)
	}

	var originalProduct *models.Product
	if r.Method == http.MethodPost {
		err = clone.Copy(
			r.Context(),
			&originalProduct,
			instance,
			clone.Flag(clone.CF_NOVALIDATE),
		)
		if err != nil {
			except.Fail(
				http.StatusInternalServerError,
				"error retrieving cloning instance of product: %v", err,
			)
		}
	}

	transactionCtx, tx, err := queries.StartTransaction(r.Context())
	if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"failed to start transaction",
		)
	}

	defer tx.Rollback(transactionCtx)
	r = r.WithContext(transactionCtx)

	var form = forms.NewAdminForm(r, specific.(supplier.ProductModel), func(p supplier.ProductModel) []admin.Panel {
		return instance.Supplier.ProductEditPanel(r.Context())
	})

	form.Load()

	form.Form.SaveInstance = func(ctx context.Context, p supplier.ProductModel) error {

		saved, err := djmodels.SaveModel(ctx, p)
		if err == nil && !saved {
			err = fmt.Errorf("model %T not saved", p)
		}

		if err != nil {
			return err
		}

		product := p.SpecificObject().Embedder().(*models.Product)

		_, err = queries.GetQuerySet[attrs.Definer](&models.Product{}).
			WithContext(ctx).
			ExplicitSave().
			Update(product)

		return err
	}

	var view = &views.FormView[*admin.AdminForm[*modelforms.BaseModelForm[supplier.ProductModel], supplier.ProductModel]]{
		AllowedMethods:  []string{http.MethodGet, http.MethodPost},
		BaseTemplateKey: "shop",
		TemplateName:    "shop/products/admin/edit.tmpl",
		GetContextFn: func(req *http.Request) (ctx.Context, error) {
			var context = admin.NewContext(req, admin.AdminSite, nil)

			var backURL string
			if q := req.URL.Query().Get("next"); q != "" {
				backURL = q
			}

			context.Set("supplier", instance.Supplier)
			context.Set("BackURL", backURL)
			context.Set("PostURL", django.Reverse("admin:apps:model:edit", "shop", "products", instance.ID))
			context.SetPage(admin.PageOptions{
				TitleFn: trans.S("Change product"),
			})

			return context, nil
		},
		GetFormFn: func(req *http.Request) *admin.AdminForm[*modelforms.BaseModelForm[supplier.ProductModel], supplier.ProductModel] {
			return form
		},
		GetInitialFn: func(req *http.Request) map[string]interface{} {
			var initial = make(map[string]interface{})
			for _, field := range attrs.Define(r.Context(), specific).Fields() {
				initial[field.Name()] = field.GetValue()
			}
			for _, field := range attrs.Define(r.Context(), instance).Fields() {
				initial[field.Name()] = field.GetValue()
			}
			return initial
		},
		SuccessFn: func(w http.ResponseWriter, req *http.Request, form *admin.AdminForm[*modelforms.BaseModelForm[supplier.ProductModel], supplier.ProductModel]) {
			var instance = form.Instance()
			except.Assert(
				instance != nil,
				http.StatusInternalServerError,
				"instance is nil after form submission",
			)

			err := shop.SIGNALS.Products.Updated.Send(r.Context(), &signals.ProductSignalData{
				BaseSignal: signals.NewBaseSignal(r.Context(), r, nil, nil),
				Product: signals.Changed[*models.Product]{
					Old: originalProduct,
					New: instance.SpecificObject().Embedder().(*models.Product),
				},
				Specific: signals.Changed[supplier.ProductModel]{
					New: instance,
				},
			})
			if err != nil {
				e, ok := gosigs.SignalError(err)
				if !ok || len(e.Related) != 1 {
					except.Fail(
						http.StatusInternalServerError,
						"error while sending shop product updated signal: %v", err,
					)
				}

				except.Fail(
					http.StatusInternalServerError,
					"error while sending shop product updated signal: %v", e.Related[0],
				)
			}

			if err = tx.Commit(transactionCtx); err != nil {
				except.Fail(
					http.StatusInternalServerError,
					"failed to commit transaction: %v", err,
				)
			}

			messages.Success(r, "Product changed")

			http.Redirect(w, r, django.Reverse("admin:apps:model", "shop", "products"), http.StatusSeeOther)
		},
	}

	if err := views.Invoke(view, w, r); err != nil {
		except.Fail(500, err)
		return
	}
}
