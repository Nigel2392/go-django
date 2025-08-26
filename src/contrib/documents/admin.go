package documents

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/admin/components/columns"
	"github.com/Nigel2392/go-django/src/contrib/filters"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-django/src/views/list"
)

var _ = contenttypes.Register(&contenttypes.ContentTypeDefinition{
	GetLabel:       trans.S("Document"),
	GetPluralLabel: trans.S("Documents"),
	GetDescription: trans.S("A document file"),
	GetInstanceLabel: func(a any) string {
		var document = a.(*Document)
		if document.Title != "" {
			return document.Title
		}
		return ""
	},
	ContentObject: &Document{},
})

func AdminDocumentModelOptions(app *AppConfig) admin.ModelOptions {
	var initAdminForm = func(updating bool) func(instance attrs.Definer, form modelforms.ModelForm[attrs.Definer]) {
		return func(instance attrs.Definer, form modelforms.ModelForm[attrs.Definer]) {
			var fileField = &fields.FileStorageField{
				BaseField: fields.NewField(
					fields.Label(trans.S("Document File")),
					fields.HelpText(trans.S("Upload an document file")),
					fields.Required(false),
				),
				Extensions: app.Options.AllowedFileExts,
				UploadTo: func(fileObject *widgets.FileObject) string {
					return filepath.Join(app.MediaDir(), fileObject.Name)
				},
				StorageBackend: app.MediaBackend(),
			}

			form.AddField("DocumentFile", fileField)

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
			form.AddWidget("Path", pathWidget)
			form.AddWidget("FileSize", sizeWidget)

			// Validator for the DocumentFile field
			form.SetValidators(func(f forms.Form, m map[string]interface{}) []error {
				var fileFace, ok = m["DocumentFile"]
				if !ok {
					if !updating && fields.IsZero(m["Path"]) {
						// If the path is empty, we require the file to be uploaded
						return []error{errs.NewValidationError[string](
							"DocumentFile", "This field is required",
						)}
					}
					return nil
				}

				fileObj, ok := fileFace.(*widgets.FileObject)
				if !ok {
					if !updating && fields.IsZero(m["Path"]) {
						// If the path is empty, we require the file to be uploaded
						return []error{errs.NewValidationError[string](
							"DocumentFile", "This field is required",
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
						"DocumentFile", fmt.Sprintf("Failed to save file: %v", err),
					)}
				}

				file, ok := fileFace.(mediafiles.StoredObject)
				if !ok {
					return []error{errs.NewValidationError[string](
						"DocumentFile", fmt.Sprintf("Invalid file type: %T", fileFace),
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

				// Delete the DocumentFile field from the model's cleaned data
				// This is to prevent the file from being saved again
				delete(m, "DocumentFile")

				return nil
			})
		}
	}

	return admin.ModelOptions{
		RegisterToAdminMenu: true,
		Model:               &Document{},
		Name:                "document",
		MenuOrder:           15,
		MenuLabel:           trans.S("Documents"),
		MenuIcon: func(ctx context.Context) string {
			return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-folder2-open" viewBox="0 0 16 16">
  <path d="M1 3.5A1.5 1.5 0 0 1 2.5 2h2.764c.958 0 1.76.56 2.311 1.184C7.985 3.648 8.48 4 9 4h4.5A1.5 1.5 0 0 1 15 5.5v.64c.57.265.94.876.856 1.546l-.64 5.124A2.5 2.5 0 0 1 12.733 15H3.266a2.5 2.5 0 0 1-2.481-2.19l-.64-5.124A1.5 1.5 0 0 1 1 6.14zM2 6h12v-.5a.5.5 0 0 0-.5-.5H9c-.964 0-1.71-.629-2.174-1.154C6.374 3.334 5.82 3 5.264 3H2.5a.5.5 0 0 0-.5.5zm-.367 1a.5.5 0 0 0-.496.562l.64 5.124A1.5 1.5 0 0 0 3.266 14h9.468a1.5 1.5 0 0 0 1.489-1.314l.64-5.124A.5.5 0 0 0 14.367 7z"/>
</svg>`
		},
		AddView: admin.FormViewOptions{
			FormInit: initAdminForm(false),
			Panels: []admin.Panel{
				admin.FieldPanel("Title"),
				admin.FieldPanel("Path"),
				admin.FieldPanel("FileSize"),
				admin.FieldPanel("DocumentFile"),
			},
		},
		EditView: admin.FormViewOptions{
			FormInit: initAdminForm(true),
			Panels: []admin.Panel{
				admin.FieldPanel("ID"),
				admin.FieldPanel("Title"),
				admin.FieldPanel("Path"),
				admin.FieldPanel("DocumentFile"),
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
						TemplateName:    "documents/documents_delete.tmpl",
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
			PerPage: 25,
			ViewOptions: admin.ViewOptions{
				Fields: []string{
					"Title", "Path", "Size", "CreatedAt", "FileHash",
				},
			},
			BulkActions: []admin.BulkAction{
				admin.BulkActionDelete,
			},
			GetQuerySet: func(adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition) *queries.QuerySet[attrs.Definer] {
				return queries.GetQuerySet[attrs.Definer](&Document{}).OrderBy("-CreatedAt")
			},
			Columns: map[string]list.ListColumn[attrs.Definer]{
				"Path": list.EditableColumn[attrs.Definer](
					trans.S("Path"),
					list.EditableColumnConfig{
						FieldName: "Path",
					},
				),
				"FileName": list.ProcessableFieldColumn(
					trans.S("File Name"),
					"Path",
					func(r *http.Request, defs attrs.Definitions, row attrs.Definer, path string) interface{} {
						return filepath.Base(path)
					},
				),
				"CreatedAt": columns.TimeSinceColumn[attrs.Definer](
					trans.S("Created"),
					"CreatedAt",
				),
				"Size": list.ProcessableFieldColumn(
					trans.S("Size"),
					"FileSize",
					func(r *http.Request, defs attrs.Definitions, row attrs.Definer, size int64) interface{} {
						switch {
						case size == 0:
							return trans.T(r.Context(), "Unknown")
						case size < 1024:
							return trans.T(r.Context(), "%dB", size)
						case size < 1024*1024:
							return trans.T(r.Context(), "%dKB", size/1024)
						case size < 1024*1024*1024:
							return trans.T(r.Context(), "%dMB", size/1024/1024)
						default:
							return trans.T(r.Context(), "%dGB", size/1024/1024/1024)
						}
					},
				),
			},
			Filters: []filters.FilterSpec[attrs.Definer]{
				&filters.BaseFilterSpec[*queries.QuerySet[attrs.Definer]]{
					SpecName:  "search",
					FormField: fields.CharField(fields.HelpText(trans.S("Search by title or URL path"))),
					Apply: func(req *http.Request, value interface{}, object *queries.QuerySet[attrs.Definer]) (*queries.QuerySet[attrs.Definer], error) {
						if fields.IsZero(value) {
							return object, nil
						}

						return object.Filter(expr.Or(
							expr.Q("Title__icontains", value),
							expr.Q("Path__icontains", value),
						)), nil
					},
				},
			},
			Search: &admin.SearchOptions{
				Fields: []admin.SearchField{
					{Name: "Title", Lookup: expr.LOOKUP_ICONTANS},
					{Name: "Path", Lookup: expr.LOOKUP_ICONTANS},
				},
			},
		},
	}
}
