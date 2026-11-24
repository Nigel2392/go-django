package adminviews

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/admin/components"
	"github.com/Nigel2392/go-django/contrib/filters"
	"github.com/Nigel2392/go-django/contrib/messages"
	"github.com/Nigel2392/go-django/contrib/shop/orders"
	"github.com/Nigel2392/go-django/contrib/shop/payments"
	"github.com/Nigel2392/go-django/contrib/shop/util/forms"
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
	"github.com/Nigel2392/go-django/src/forms/formsets"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	djmodels "github.com/Nigel2392/go-django/src/models"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-django/src/views/list"
	"github.com/Nigel2392/mux/v2"
)

var ViewOrderList = generic.ListViewConfig[*orders.Order]{
	ListTitle: trans.S("Order List"),
	Model:     &orders.Order{},
	Permissions: []string{
		"orders:list",
	},
	GetEditLink: func(r *http.Request, v *orders.Order) string {
		return django.Reverse("admin:apps:model:edit", "shop", "orders", v.ID)
	},
	GetQuerySet: func(r *http.Request) *queries.QuerySet[*orders.Order] {
		return queries.GetQuerySet(&orders.Order{}).Preload(queries.NoJoins("Payments"))
	},
	Filters: []filters.FilterSpec[*orders.Order]{
		&filters.BaseFilterSpec[*queries.QuerySet[*orders.Order]]{
			SpecName: "search",
			FormField: fields.CharField(
				fields.Placeholder("Search..."),
			),
			Apply: func(req *http.Request, value interface{}, object *queries.QuerySet[*orders.Order]) (*queries.QuerySet[*orders.Order], error) {
				if fields.IsZero(value) {
					return object, nil
				}

				paymentOrders := queries.GetQuerySet(&payments.Payment{}).
					WithContext(req.Context()).
					Filter(expr.Or(expr.Q("ID__icontains", value), expr.Q("Provider__icontains", value))).
					Select("Order")
				return object.Filter(expr.Or(
					expr.Q("ID__icontains", value),
					expr.Q("ID__in", queries.Subquery(paymentOrders)),
				)), nil
			},
		},
	},
	GetHeaderActions: func(r *http.Request) []components.ShowableComponent {

		var amount, _ = strconv.Atoi(r.URL.Query().Get("amount"))
		if amount < 1 {
			amount = 25
		}

		var actions = make([]components.ShowableComponent, 0)
		actions = append(actions, generic.AmountHeaderAction(
			r, amount, "admin:apps:model", nil, "shop", "orders",
		))

		return actions
	},
	GetColumns: func(r *http.Request) ([]list.ListColumn[*orders.Order], error) {
		var cols = []list.ListColumn[*orders.Order]{
			list.FieldColumn[*orders.Order](
				trans.S("Order ID"),
				"ID",
			),
			list.FuncColumn(trans.S("Payment IDs"), func(_ *http.Request, _ attrs.Definitions, order *orders.Order) any {
				attempts := order.Payments.AsList()
				ids := make([]string, 0, len(attempts))
				for _, attempt := range attempts {
					ids = append(ids, attempt.ID.String())
				}
				return strings.Join(ids, ", ")
			}),
		}

		return cols, nil
	},
}

type orderItemFormset interface {
	Forms() ([]modelforms.ModelForm[*orders.OrderItem], error)
}

func ViewAddOrder(w http.ResponseWriter, r *http.Request) {
	if !permissions.HasObjectPermission(r, &orders.Order{}, "orders:add") {
		admin.ReLogin(w, r, r.URL.Path)
		return
	}

	var form = forms.NewAdminForm(r, &orders.Order{}, func(p *orders.Order) []admin.Panel {
		return p.AddPanels(r.Context())
	})

	form.Load()

	form.Form.SaveInstance = func(ctx context.Context, p *orders.Order) error {

		flist, err := form.FormSet().Forms()
		if err != nil {
			return err
		}

		var (
			skuFormSet orderItemFormset
		)

		for i, form := range flist {
			if c, ok := form.(admin.ClusterOrderableFormT[formsets.BaseFormSetForm]); ok {
				flist[i] = c.Form()
			}
		}

		for _, form := range flist {
			switch f := form.(type) {
			case orderItemFormset:
				skuFormSet = f
			default:
				panic("unhandled form type")
			}
		}

		skuForms, err := skuFormSet.Forms()
		if err != nil {
			return err
		}

		// var priceMax decimal.Decimal
		var skus = make([]*orders.OrderItem, 0, len(skuForms))
		for _, form := range skuForms {

			sku := form.Instance()
			skus = append(skus, sku)

			//	if sku.Price.GreaterThan(priceMax) {
			//		priceMax = sku.Price
			//	}

			form.CleanedData()["Order"] = p
		}

		//	p.SkuCount = len(skus)
		//	p.PriceMax = priceMax

		saved, err := djmodels.SaveModel(ctx, p)
		if err == nil && !saved {
			err = fmt.Errorf("model %T not saved", p)
		}

		return orders.Signals.Created.Send(r.Context(), orders.OrderCreatedSignal{
			Request: r,
			Order:   p,
			Lines:   skus,
		})
	}

	var view = &views.FormView[*admin.AdminForm[*modelforms.BaseModelForm[*orders.Order], *orders.Order]]{
		AllowedMethods:  []string{http.MethodGet, http.MethodPost},
		BaseTemplateKey: "shop",
		TemplateName:    "shop/orders/admin/add.tmpl",
		GetContextFn: func(req *http.Request) (ctx.Context, error) {
			var context = admin.NewContext(req, admin.AdminSite, nil)

			var backURL string
			if q := req.URL.Query().Get("next"); q != "" {
				backURL = q
			}

			context.Set("BackURL", backURL)
			context.Set("PostURL", django.Reverse("admin:apps:model:add", "shop", "orders"))
			context.SetPage(admin.PageOptions{
				TitleFn: trans.S("Change order"),
			})

			return context, nil
		},
		GetFormFn: func(req *http.Request) *admin.AdminForm[*modelforms.BaseModelForm[*orders.Order], *orders.Order] {
			return form
		},
		GetInitialFn: func(req *http.Request) map[string]interface{} {
			var initial = make(map[string]interface{})
			for _, field := range attrs.Define(r.Context(), &orders.Order{}).Fields() {
				initial[field.Name()] = field.GetValue()
			}
			return initial
		},
		SuccessFn: func(w http.ResponseWriter, req *http.Request, form *admin.AdminForm[*modelforms.BaseModelForm[*orders.Order], *orders.Order]) {
			var instance = form.Instance()
			except.Assert(
				instance != nil,
				http.StatusInternalServerError,
				"instance is nil after form submission",
			)

			messages.Success(r, "Order changed")

			http.Redirect(w, r, django.Reverse("admin:apps:model", "shop", "orders"), http.StatusSeeOther)
		},
	}

	if err := views.Invoke(view, w, r); err != nil {
		except.Fail(500, err)
		return
	}
}

func ViewEditOrder(w http.ResponseWriter, r *http.Request) {
	if !permissions.HasObjectPermission(r, &orders.Order{}, "orders:edit") {
		admin.ReLogin(w, r, r.URL.Path)
		return
	}

	var vars = mux.Vars(r)
	var orderRow, err = queries.GetQuerySet(&orders.Order{}).
		Preload(queries.NoJoins("Payments"), queries.NoJoins("OrderItems")).
		Filter("ID", vars.Get("order_id")).
		Get()

	if errors.Is(err, errors.NoRows) {
		except.RaiseNotFound("No order found.")
		return
	}

	if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"failed to retrieve order: %v", err,
		)
	}

	var form = forms.NewAdminForm(r, orderRow.Object, func(p *orders.Order) []admin.Panel {
		return p.EditPanels(r.Context())
	})

	form.Load()

	form.Form.SaveInstance = func(ctx context.Context, p *orders.Order) error {

		flist, err := form.FormSet().Forms()
		if err != nil {
			return err
		}

		var (
			skuFormSet orderItemFormset
		)

		for i, form := range flist {
			if c, ok := form.(admin.ClusterOrderableFormT[formsets.BaseFormSetForm]); ok {
				flist[i] = c.Form()
			}
		}

		for _, form := range flist {
			switch f := form.(type) {
			case orderItemFormset:
				skuFormSet = f
			default:
				panic("unhandled form type")
			}
		}

		skuForms, err := skuFormSet.Forms()
		if err != nil {
			return err
		}

		// var priceMax decimal.Decimal
		var skus = make([]*orders.OrderItem, 0, len(skuForms))
		for _, form := range skuForms {

			sku := form.Instance()
			skus = append(skus, sku)

			//	if sku.Price.GreaterThan(priceMax) {
			//		priceMax = sku.Price
			//	}

			form.CleanedData()["Order"] = p
		}

		//	p.SkuCount = len(skus)
		//	p.PriceMax = priceMax

		saved, err := djmodels.SaveModel(ctx, p)
		if err == nil && !saved {
			err = fmt.Errorf("model %T not saved", p)
		}

		return orders.Signals.Updated.Send(r.Context(), orders.OrderUpdatedSignal{
			Request: r,
			Order:   p,
			Lines:   skus,
		})
	}

	var view = &views.FormView[*admin.AdminForm[*modelforms.BaseModelForm[*orders.Order], *orders.Order]]{
		AllowedMethods:  []string{http.MethodGet, http.MethodPost},
		BaseTemplateKey: "shop",
		TemplateName:    "shop/orders/admin/edit.tmpl",
		GetContextFn: func(req *http.Request) (ctx.Context, error) {
			var context = admin.NewContext(req, admin.AdminSite, nil)

			var backURL string
			if q := req.URL.Query().Get("next"); q != "" {
				backURL = q
			}

			context.Set("BackURL", backURL)
			context.Set("PostURL", django.Reverse("admin:apps:model:edit", "shop", "orders", orderRow.Object.ID))
			context.SetPage(admin.PageOptions{
				TitleFn: trans.S("Change order"),
			})

			return context, nil
		},
		GetFormFn: func(req *http.Request) *admin.AdminForm[*modelforms.BaseModelForm[*orders.Order], *orders.Order] {
			return form
		},
		GetInitialFn: func(req *http.Request) map[string]interface{} {
			var initial = make(map[string]interface{})
			for _, field := range attrs.Define(r.Context(), orderRow.Object).Fields() {
				initial[field.Name()] = field.GetValue()
			}
			return initial
		},
		SuccessFn: func(w http.ResponseWriter, req *http.Request, form *admin.AdminForm[*modelforms.BaseModelForm[*orders.Order], *orders.Order]) {
			var instance = form.Instance()
			except.Assert(
				instance != nil,
				http.StatusInternalServerError,
				"instance is nil after form submission",
			)

			messages.Success(r, "Order changed")

			http.Redirect(w, r, django.Reverse("admin:apps:model", "shop", "orders"), http.StatusSeeOther)
		},
	}

	if err := views.Invoke(view, w, r); err != nil {
		except.Fail(500, err)
		return
	}
}
