package images

import (
	"net/http"
	"strconv"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/pagination"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/views"
)

var _ = contenttypes.Register(&contenttypes.ContentTypeDefinition{
	GetLabel:       trans.S("Image"),
	GetPluralLabel: trans.S("Images"),
	GetDescription: trans.S("An image file"),
	GetInstanceLabel: func(a any) string {
		var image = a.(*Image)
		if image.Title != "" {
			return image.Title
		}
		return ""
	},
	ContentObject: &Image{},
})

func AdminImageModelOptions() admin.ModelOptions {
	return admin.ModelOptions{
		RegisterToAdminMenu: true,
		Model:               &Image{},
		Name:                "Image",
		AddView: admin.FormViewOptions{
			Panels: []admin.Panel{
				admin.FieldPanel("ID"),
				admin.FieldPanel("Title"),
				admin.FieldPanel("Path"),
				admin.FieldPanel("CreatedAt"),
				admin.FieldPanel("FileSize"),
				admin.FieldPanel("FileHash"),
			},
		},
		EditView: admin.FormViewOptions{
			Panels: []admin.Panel{
				admin.FieldPanel("ID"),
				admin.FieldPanel("Title"),
				admin.FieldPanel("Path"),
				admin.FieldPanel("CreatedAt"),
				admin.FieldPanel("FileSize"),
				admin.FieldPanel("FileHash"),
			},
		},
		ListView: admin.ListViewOptions{
			GetHandler: func(adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition) views.View {

				const (
					amountParam = "amount"
					pageParam   = "page"
					maxAmount   = 25
				)

				return &views.BaseView{
					BaseTemplateKey: "admin",
					TemplateName:    "images/image_list.html",
					GetContextFn: func(req *http.Request) (ctx.Context, error) {
						var (
							context = admin.NewContext(
								req,
								adminSite,
								ctx.RequestContext(req),
							)

							amountValue = req.URL.Query().Get(amountParam)
							pageValue   = req.URL.Query().Get(pageParam)
							amount      uint64
							page        uint64
							err         error
						)

						if amountValue == "" {
							amount = maxAmount
						} else {
							amount, err = strconv.ParseUint(amountValue, 10, 64)
						}
						if err != nil {
							return context, err
						}

						if pageValue == "" {
							page = 1
						} else {
							page, err = strconv.ParseUint(pageValue, 10, 64)
						}
						if err != nil {
							return context, err
						}

						if amount > maxAmount {
							amount = maxAmount
						}

						var paginator = &pagination.QueryPaginator[*Image]{
							Context: req.Context(),
							Amount:  int(amount),
							BaseQuerySet: func() *queries.QuerySet[*Image] {
								return queries.GetQuerySetWithContext(req.Context(), &Image{}).
									OrderBy("-CreatedAt")
							},
						}

						pageObject, err := paginator.Page(int(page))
						if err != nil && !errors.Is(err, errors.NoRows) {
							return context, err
						}

						context.Set("view_list", pageObject.Results())
						context.Set("view_amount", amount)
						context.Set("view_page", page)
						context.Set("view_paginator", paginator)
						context.Set("view_paginator_object", pageObject)

						// set the constants
						context.Set("view_max_amount", maxAmount)
						context.Set("view_amount_param", amountParam)
						context.Set("view_page_param", pageParam)
						return context, nil
					},
				}
			},
		},
	}
}
