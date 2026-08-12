package adminviews

import (
	"context"
	"fmt"
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
)

var ViewOrderList = generic.ListViewConfig[*models.Order]{
	ListTitle: trans.S("Order List"),
	Model:     &models.Order{},
	GetEditLink: func(r *http.Request, v *models.Order) string {
		return django.Reverse("admin:shop:orders:edit", v.ID)
	},
	GetQuerySet: func(r *http.Request) *queries.QuerySet[*models.Order] {
		return queries.GetQuerySet(&models.Order{}).Select("*", "Payment.*")
	},
	Filters: []filters.FilterSpec[*models.Order]{
		&filters.BaseFilterSpec[*queries.QuerySet[*models.Order]]{
			SpecName: "search",
			FormField: fields.CharField(
				fields.Placeholder("Search..."),
			),
			Apply: func(req *http.Request, value interface{}, object *queries.QuerySet[*models.Order]) (*queries.QuerySet[*models.Order], error) {
				if fields.IsZero(value) {
					return object, nil
				}

				return object.Filter(expr.Or(
					expr.Q("ID__icontains", value),
					expr.Q("Payment.ID__icontains", value),
					expr.Q("Payment.Provider__icontains", value),
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
			r, amount, "admin:shop:orders", nil,
		))

		return actions
	},
	GetColumns: func(r *http.Request) ([]list.ListColumn[*models.Order], error) {
		var cols = []list.ListColumn[*models.Order]{
			list.FieldColumn[*models.Order](
				trans.S("Order ID"),
				"ID",
			),
			list.FieldColumn[*models.Order](
				trans.S("Payment ID"),
				"Payment.ID",
			),
		}

		return cols, nil
	},
}

type orderItemFormset interface {
	Forms() ([]modelforms.ModelForm[*models.OrderItem], error)
}

func ViewEditOrder(w http.ResponseWriter, r *http.Request, shop *app.ShopAppConfig) {
	if !permissions.HasObjectPermission(r, &models.Order{}, "orders:edit") {
		admin.ReLogin(w, r, r.URL.Path)
		return
	}

	var vars = mux.Vars(r)
	var orderRow, err = queries.GetQuerySet(&models.Order{}).
		Select("*", "Payment.*", "OrderItem.*").
		Filter("ID", vars.GetInt("order_id")).
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

	var form = forms.NewAdminForm(r, orderRow.Object, func(p *models.Order) []admin.Panel {
		return p.EditPanels(r.Context())
	})

	form.Load()

	form.Form.SaveInstance = func(ctx context.Context, p *models.Order) error {

		flist, err := form.FormSet().Forms()
		if err != nil {
			return err
		}

		var (
			skuFormSet orderItemFormset
		)

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
		var skus = make([]*models.OrderItem, 0, len(skuForms))
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

		if _, err := form.FormSet().Save(); err != nil {
			return err
		}

		return shop.SIGNALS.Orders.Updated.Send(&signals.OrderSignalData{
			BaseSignal: signals.NewBaseSignal(ctx, r, nil, nil),
			Current:    p,
			OrderItems: skus,
			Reason:     "admin_update",
		})

	}

	var view = &views.FormView[*admin.AdminForm[*modelforms.BaseModelForm[*models.Order], *models.Order]]{
		BaseView: views.BaseView{
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
				context.Set("PostURL", django.Reverse("admin:shop:orders:edit", orderRow.Object.ID))
				context.SetPage(admin.PageOptions{
					TitleFn: trans.S("Change order"),
				})

				return context, nil
			},
		},
		GetFormFn: func(req *http.Request) *admin.AdminForm[*modelforms.BaseModelForm[*models.Order], *models.Order] {
			return form
		},
		GetInitialFn: func(req *http.Request) map[string]interface{} {
			var initial = make(map[string]interface{})
			for _, field := range attrs.Define(r.Context(), orderRow.Object).Fields() {
				initial[field.Name()] = field.GetValue()
			}
			return initial
		},
		SuccessFn: func(w http.ResponseWriter, req *http.Request, form *admin.AdminForm[*modelforms.BaseModelForm[*models.Order], *models.Order]) {
			var instance = form.Instance()
			except.Assert(
				instance != nil,
				http.StatusInternalServerError,
				"instance is nil after form submission",
			)

			messages.Success(r, "Order changed")

			http.Redirect(w, r, django.Reverse("admin:shop:orders"), http.StatusSeeOther)
		},
	}

	if err := views.Invoke(view, w, r); err != nil {
		except.Fail(500, err)
		return
	}
}
