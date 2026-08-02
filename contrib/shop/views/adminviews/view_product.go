package adminviews

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/messages"
	"github.com/Nigel2392/go-django/contrib/shop/internal/app"
	"github.com/Nigel2392/go-django/contrib/shop/models"
	"github.com/Nigel2392/go-django/contrib/shop/util/forms"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	djmodels "github.com/Nigel2392/go-django/src/models"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/mux"
)

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

		saved, err := djmodels.SaveModel(ctx, p)
		if err == nil && !saved {
			err = fmt.Errorf("model %T not saved", p)
		}

		if err != nil {
			return err
		}

		flist, err := form.FormSet().Forms()
		if err != nil {
			return err
		}

		fs, ok := flist[0].(productSkuFormset)
		if !ok {
			return errors.TypeMismatch.Wrapf(
				"form %T does not implement productSkuFormset",
				form,
			)
		}

		skuForms, err := fs.Forms()
		if err != nil {
			return err
		}

		for _, form := range skuForms {
			fmt.Printf("%T %T %v\n", form, form.Instance(), form.Instance())

		}

		return nil
	}

	var view = &views.FormView[*admin.AdminForm[*modelforms.BaseModelForm[*models.Product], *models.Product]]{
		BaseView: views.BaseView{
			AllowedMethods:  []string{http.MethodGet, http.MethodPost},
			BaseTemplateKey: admin.BASE_KEY,
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

		saved, err := djmodels.SaveModel(ctx, p)
		if err == nil && !saved {
			err = fmt.Errorf("model %T not saved", p)
		}

		if err != nil {
			return err
		}

		flist, err := form.FormSet().Forms()
		if err != nil {
			return err
		}

		fs, ok := flist[0].(productSkuFormset)
		if !ok {
			return errors.TypeMismatch.Wrapf(
				"form %T does not implement productSkuFormset",
				form,
			)
		}

		skuForms, err := fs.Forms()
		if err != nil {
			return err
		}

		for _, form := range skuForms {
			if !form.HasChanged() {
				continue
			}

			var sku = form.Instance()
			sku.Product = p

			if _, err = form.Save(); err != nil {
				return err
			}
		}

		return nil
	}

	var view = &views.FormView[*admin.AdminForm[*modelforms.BaseModelForm[*models.Product], *models.Product]]{
		BaseView: views.BaseView{
			AllowedMethods:  []string{http.MethodGet, http.MethodPost},
			BaseTemplateKey: admin.BASE_KEY,
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

			messages.Success(r, "Product added")

			http.Redirect(w, r, django.Reverse("admin:shop:products"), http.StatusSeeOther)
		},
	}

	if err := views.Invoke(view, w, r); err != nil {
		except.Fail(500, err)
		return
	}
}
