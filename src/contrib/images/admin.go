package images

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	"github.com/Nigel2392/go-django/src/core/pagination"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/Nigel2392/go-django/src/forms/widgets"
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
			FormInit: func(instance attrs.Definer, form modelforms.ModelForm[attrs.Definer]) {
				var fileField = &fields.FileStorageField{
					BaseField: fields.NewField(
						fields.Label(trans.S("Image File")),
						fields.HelpText(trans.S("Upload an image file")),
						fields.Required(true),
					),
					UploadTo: func(fileObject *widgets.FileObject) string {
						return filepath.Join(app.MediaDir(), fileObject.Name)
					},
					StorageBackend: app.MediaBackend(),
				}

				form.AddField("ImageFile", fileField)
				form.SetFields("Title", "ImageFile", "Path")

				var pathField, ok = form.Field("Path")
				if !ok {
					assert.Fail("Path field not found in form")
					return
				}

				var widget = pathField.Widget()
				widget.Hide(true)

				form.AddWidget("Path", widget)

				form.SetValidators(func(f forms.Form, m map[string]interface{}) []error {
					var fileFace, ok = m["ImageFile"]
					if !ok {
						return []error{errs.NewValidationError[string](
							"ImageFile", "This field is required",
						)}
					}

					fileObj, ok := fileFace.(*widgets.FileObject)
					if !ok {
						return []error{errs.NewValidationError[string](
							"ImageFile", fmt.Sprintf("Invalid file type: %T", fileFace),
						)}
					}

					var err error
					fileFace, err = fileField.Save(fileObj)
					if err != nil {
						return []error{errs.NewValidationError[string](
							"ImageFile", fmt.Sprintf("Failed to save file: %v", err),
						)}
					}

					file, ok := fileFace.(mediafiles.StoredObject)
					if !ok {
						return []error{errs.NewValidationError[string](
							"ImageFile", fmt.Sprintf("Invalid file type: %T", fileFace),
						)}
					}

					m["Path"] = file.Path()

					return nil
				})
			},
			Panels: []admin.Panel{
				admin.FieldPanel("ID"),
				admin.FieldPanel("Title"),
				admin.FieldPanel("Path"),
				admin.FieldPanel("ImageFile"),
			},
		},
		EditView: admin.FormViewOptions{
			FormInit: func(instance attrs.Definer, form modelforms.ModelForm[attrs.Definer]) {
				var fileField = &fields.FileStorageField{
					BaseField: fields.NewField(
						fields.Label(trans.S("Image File")),
						fields.HelpText(trans.S("Upload an image file")),
						fields.Required(false),
					),
					UploadTo: func(fileObject *widgets.FileObject) string {
						return filepath.Join(app.MediaDir(), fileObject.Name)
					},
					StorageBackend: app.MediaBackend(),
				}

				form.AddField("ImageFile", fileField)
				form.SetFields("Title", "ImageFile", "Path")

				var pathField, ok = form.Field("Path")
				if !ok {
					assert.Fail("Path field not found in form")
					return
				}

				var widget = pathField.Widget()
				widget.Hide(true)

				form.AddWidget("Path", widget)

				form.SetValidators(func(f forms.Form, m map[string]interface{}) []error {
					var fileFace, ok = m["ImageFile"]
					if !ok {
						if fields.IsZero(m["Path"]) {
							// If the path is empty, we require the file to be uploaded
							return []error{errs.NewValidationError[string](
								"ImageFile", "This field is required",
							)}
						}
						return nil
					}

					fileObj, ok := fileFace.(*widgets.FileObject)
					if !ok {
						if fields.IsZero(m["Path"]) {
							// If the path is empty, we require the file to be uploaded
							return []error{errs.NewValidationError[string](
								"ImageFile", "This field is required",
							)}
						}
						return nil
					}

					var err error
					fileFace, err = fileField.Save(fileObj)
					if err != nil {
						return []error{errs.NewValidationError[string](
							"ImageFile", fmt.Sprintf("Failed to save file: %v", err),
						)}
					}

					file, ok := fileFace.(mediafiles.StoredObject)
					if !ok {
						return []error{errs.NewValidationError[string](
							"ImageFile", fmt.Sprintf("Invalid file type: %T", fileFace),
						)}
					}

					m["Path"] = file.Path()

					return nil
				})
			},
			Panels: []admin.Panel{
				admin.FieldPanel("ID"),
				admin.FieldPanel("Title"),
				admin.FieldPanel("Path"),
				admin.FieldPanel("ImageFile"),
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

						// set model information
						context.Set("app", app)
						context.Set("model", model)
						return context, nil
					},
				}
			},
		},
	}
}
