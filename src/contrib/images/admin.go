package images

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
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
	"github.com/Nigel2392/go-django/src/views/list"
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
		if image.Path != "" {
			return image.Path
		}
		return fmt.Sprintf("Image %d", image.ID)
	},
	ContentObject: &Image{},
})

func AdminImageModelOptions(app *AppConfig) admin.ModelOptions {
	var initAdminForm = func(updating bool) func(instance attrs.Definer, form modelforms.ModelForm[attrs.Definer]) {
		return func(instance attrs.Definer, form modelforms.ModelForm[attrs.Definer]) {
			var fileField = &fields.FileStorageField{
				BaseField: fields.NewField(
					fields.Label(trans.S("Image File")),
					fields.HelpText(trans.S("Upload an image file")),
					fields.Required(false),
				),
				Extensions: app.Options.AllowedFileExts,
				UploadTo: func(fileObject *widgets.FileObject) string {
					return filepath.Join(app.MediaDir(), fileObject.Name)
				},
				StorageBackend: app.MediaBackend(),
			}

			form.AddField("ImageFile", fileField)

			var pathWidget, ok = form.Widget("Path")
			if !ok {
				assert.Fail("Path field not found in form")
				return
			}
			if !updating {
				pathWidget.Hide(true)
			} else {
				pathWidget.SetAttrs(map[string]string{
					"readonly": "readonly",
				})
			}

			sizeWidget, ok := form.Widget("FileSize")
			if !ok {
				assert.Fail("FileSize field not found in form")
				return
			}
			if !updating {
				sizeWidget.Hide(true)
			} else {
				sizeWidget.SetAttrs(map[string]string{
					"readonly": "readonly",
				})
			}

			//	hashWidget, ok := form.Widget("FileHash")
			//	if !ok {
			//		assert.Fail("FileHash field not found in form")
			//		return
			//	}
			//	if !updating {
			//		hashWidget.Hide(true)
			//	} else {
			//		hashWidget.SetAttrs(map[string]string{
			//			"readonly": "readonly",
			//			"disabled": "disabled",
			//		})
			//	}

			form.AddWidget("Path", pathWidget)
			form.AddWidget("FileSize", sizeWidget)
			// form.AddWidget("FileHash", hashWidget)

			// Validator for the ImageFile field
			form.SetValidators(func(f forms.Form, m map[string]interface{}) []error {
				var fileFace, ok = m["ImageFile"]
				if !ok {
					if !updating && fields.IsZero(m["Path"]) {
						// If the path is empty, we require the file to be uploaded
						return []error{errs.NewValidationError[string](
							"ImageFile", trans.T(form.Context(), "This field is required"),
						)}
					}
					return nil
				}

				fileObj, ok := fileFace.(*widgets.FileObject)
				if !ok {
					if !updating && fields.IsZero(m["Path"]) {
						// If the path is empty, we require the file to be uploaded
						return []error{errs.NewValidationError[string](
							"ImageFile", trans.T(form.Context(), "This field is required"),
						)}
					}
					return nil
				}

				if fileObj.File == nil || fileObj.File.Len() == 0 {
					return []error{errs.NewValidationError[string](
						"ImageFile", trans.T(form.Context(), "File is empty, please try to upload it again"),
					)}
				}

				var err error
				m["FileSize"] = fileObj.File.Len()
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

				// Base the title on the file name if not set
				title, ok := m["Title"].(string)
				if !ok || title == "" {
					var fName = fileObj.Name
					var baseName = filepath.Base(fName)
					var ext = filepath.Ext(baseName)
					if ext != "" {
						baseName = baseName[:len(baseName)-len(ext)]
					}
					m["Title"] = baseName
				}

				// Delete the ImageFile field from the model's cleaned data
				// This is to prevent the file from being saved again
				delete(m, "ImageFile")

				return nil
			})
		}
	}

	return admin.ModelOptions{
		RegisterToAdminMenu: true,
		Model:               &Image{},
		Name:                "image",
		MenuOrder:           10,
		MenuLabel:           trans.S("Images"),
		MenuIcon: func(ctx context.Context) string {
			return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-image" viewBox="0 0 16 16">
 	<path d="M6.002 5.5a1.5 1.5 0 1 1-3 0 1.5 1.5 0 0 1 3 0"/>
 	<path d="M2.002 1a2 2 0 0 0-2 2v10a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V3a2 2 0 0 0-2-2zm12 1a1 1 0 0 1 1 1v6.5l-3.777-1.947a.5.5 0 0 0-.577.093l-3.71 3.71-2.66-1.772a.5.5 0 0 0-.63.062L1.002 12V3a1 1 0 0 1 1-1z"/>
</svg>`
		},
		AddView: admin.FormViewOptions{
			FormInit: initAdminForm(false),
			Panels: []admin.Panel{
				admin.FieldPanel("Title"),
				admin.FieldPanel("Path"),
				admin.FieldPanel("ImageFile"),
				admin.FieldPanel("FileSize"),
				// admin.FieldPanel("FileHash"),
			},
		},
		EditView: admin.FormViewOptions{
			FormInit: initAdminForm(true),
			Panels: []admin.Panel{
				admin.FieldPanel("ID"),
				admin.FieldPanel("Title"),
				admin.FieldPanel("Path"),
				admin.FieldPanel("ImageFile"),
				admin.PanelClass("collapsed", admin.LabeledPanelGroup(
					trans.S("File Metadata"),
					trans.S("Metadata about the file"),
					admin.FieldPanel("CreatedAt"),
					admin.FieldPanel("FileSize"),
					admin.FieldPanel("FileHash"),
				)),
			},
		},
		DeleteView: admin.DeleteViewOptions{
			GetHandler: func(adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition, instances []attrs.Definer) views.View {
				return &admin.AdminDeleteView{
					BaseView: views.BaseView{
						BaseTemplateKey: "admin",
						TemplateName:    "images/images_delete.tmpl",
						AllowedMethods:  []string{"GET", "POST"},
					},
					Permissions: []string{"admin:delete"},
					AdminSite:   adminSite,
					App:         app,
					Model:       model,
					Instances:   instances,
				}
			},
		},
		ListView: admin.ListViewOptions{
			Search: &admin.SearchOptions{
				Fields: []admin.SearchField{
					{Name: "Title", Lookup: expr.LOOKUP_ICONTANS},
					{Name: "Path", Lookup: expr.LOOKUP_ICONTANS},
				},
				GetList: func(b *admin.BoundSearchView, list []attrs.Definer, _ []list.ListColumn[attrs.Definer]) (admin.StringRenderer, error) {
					return &SearchComponent{View: b, Objects: list}, nil
				},
			},
			//Ordering: []string{"-CreatedAt"},
			//GetList: func(r *http.Request, adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition, results []attrs.Definer) (list.StringRenderer, error) {
			//	return &ImageListComponent{AdminSite: adminSite, App: app, Model: model, Results: results, R: r}, nil
			//},
			GetHandler: func(adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition) views.View {
				const (
					amountParam = "amount"
					pageParam   = "page"
					maxAmount   = 25
				)

				return &views.BaseView{
					BaseTemplateKey: "admin",
					TemplateName:    "images/image_list.tmpl",
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
