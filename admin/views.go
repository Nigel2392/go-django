package admin

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/core/httputils"
	"github.com/Nigel2392/go-django/core/modelutils/namer"
	"github.com/Nigel2392/go-django/core/views"
	"github.com/Nigel2392/orderedmap"
	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/middleware/tracer"
	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/router/v3/request/response"
	"github.com/Nigel2392/tags"
)

var (
	template_index         = "admin/templates/index.tmpl"
	template_app_index     = "admin/templates/app_index.tmpl"
	template_create_object = "admin/templates/create_object.tmpl"
	template_update_object = "admin/templates/update_object.tmpl"
	template_delete_object = "admin/templates/delete_object.tmpl"
	template_list_objects  = "admin/templates/list_objects.tmpl"
	template_unauthorized  = "admin/errors/unauthorized.tmpl"
	template_errors        = "admin/errors/errors.tmpl"
)

func indexView(rq *request.Request) {
	var template, name, err = templateManager().Get(template_index)
	if err != nil {
		AdminSite_Logger.Critical(err)
		renderError(rq, "Error loading template", 500, err)
		return
	}

	err = response.Template(rq, template, name)
	if err != nil {
		AdminSite_Logger.Critical(err)
		renderError(rq, "Error rendering template", 500, err)
	}
}

func appIndex(app *application) router.HandleFunc {
	return router.HandleFunc(func(rq *request.Request) {
		var template, name, err = templateManager().Get(template_app_index)
		if err != nil {
			AdminSite_Logger.Critical(err)
			renderError(rq, "Error loading template", 500, err)
			return
		}

		rq.Data.Set("app", app)
		rq.Data.Set("title", app.Name)

		err = response.Template(rq, template, name)
		if err != nil {
			AdminSite_Logger.Critical(err)
			renderError(rq, "Error rendering template", 500, err)
		}
	})
}

func newCreateView[T ModelInterface[T]](m *viewOptions[T]) *views.CreateView[T] {
	return &views.CreateView[T]{
		BaseFormView: views.BaseFormView[T]{
			Template:    template_create_object,
			GetTemplate: templateManager().Get,
			BackURL:     goback,
			NeedsAuth:   true,
			NeedsAdmin:  true,
			GetInstance: func(r *request.Request) (v T, err error) {
				var typeOf = reflect.TypeOf(m.Options.Model)
				var isPtr = typeOf.Kind() == reflect.Ptr
				if isPtr {
					typeOf = typeOf.Elem()
				}

				var valueOf = reflect.New(typeOf)
				if isPtr {
					return valueOf.Interface().(T), nil
				} else {
					return valueOf.Elem().Interface().(T), nil
				}
			},
			FormTag: "admin-form",
			BeforeRender: func(r *request.Request, v T, fields *orderedmap.Map[string, *views.FormField]) {
				r.Data.Set("model", m.Model)
				setFieldClasses(r, fields)
			},
			PostRedirect: func(r *request.Request, v T) string {
				return string(m.Model.URL_List.Format())
			},
		},
		Fields: m.Options.FormFields,
	}
}

func newUpdateView[T ModelInterface[T]](m *viewOptions[T]) *views.UpdateView[T] {
	return &views.UpdateView[T]{
		BaseFormView: views.BaseFormView[T]{
			Template:    template_update_object,
			GetTemplate: templateManager().Get,
			BackURL:     goback,
			NeedsAuth:   true,
			NeedsAdmin:  true,
			GetInstance: getInstance(m.Options.Model),
			FormTag:     "admin-form",
			BeforeRender: func(r *request.Request, v T, fields *orderedmap.Map[string, *views.FormField]) {
				r.Data.Set("id", v.StringID())
				r.Data.Set("model", m.Model)
				setFieldClasses(r, fields)
			},
			PostRedirect: func(r *request.Request, v T) string {
				return string(m.Model.URL_List.Format())
			},
		},
		Fields: m.Options.FormFields,
	}
}

var classTextArea = "admin-form-textarea"
var classInput = "admin-form-input"

func setFieldClasses(r *request.Request, formFields *orderedmap.Map[string, *views.FormField]) {
	formFields.ForEach(func(k string, v *views.FormField) bool {
		var valueOf = reflect.ValueOf(v.Field)
		if valueOf.Kind() == reflect.Ptr {
			valueOf = valueOf.Elem()
		}
		switch valueOf.Kind() {
		case reflect.String:
			if v.Tags.Exists("textarea") {
				v.Tags["class"] = []string{classTextArea}
			}
		}
		v.Tags["class"] = append(v.Tags["class"], classInput)
		return true
	})
}

func newDeleteView[T ModelInterface[T]](m *viewOptions[T]) *views.DeleteView[T] {
	return &views.DeleteView[T]{
		BaseView: views.BaseView[T]{
			Template:    template_delete_object,
			GetTemplate: templateManager().Get,
			BackURL:     func(r *request.Request, model T) string { return goback(r) },
			GetQuerySet: getInstance(m.Options.Model),
		},
	}
}

func getInstance[T ModelInterface[T]](m T) func(r *request.Request) (T, error) {
	return func(r *request.Request) (v T, err error) {
		var val any
		var ok bool
		var id = r.URLParams.Get("id")
		if id == "" {
			return v, errors.New("id not found")
		}
		val, err = m.GetFromStringID(id)
		if err != nil {
			return v, err
		}
		if v, ok = val.(T); !ok {
			return v, fmt.Errorf("invalid type %T, cannot cast to %T", val, v)
		}
		return v, nil
	}
}

type viewOptions[T ModelInterface[T]] struct {
	Model   *model
	Options *AdminOptions[T]
}

func listView[T ModelInterface[T]](m *viewOptions[T]) func(r *request.Request) {
	if len(m.Options.ListFields) == 0 {
		var v = reflect.ValueOf(m.Options.Model)
		m.Options.ListFields = make([]string, 0)
		var t = v.Type()
		for i := 0; i < t.NumField(); i++ {
			var field = t.Field(i)
			if field.Anonymous || !field.IsExported() {
				continue
			}
			var tag = t.Field(i).Tag
			var adminTag = tag.Get("admin-list")
			if adminTag == "-" || strings.EqualFold(field.Name, "id") {
				continue
			}
			m.Options.ListFields = append(m.Options.ListFields, adminTag)
		}
	}
	return func(r *request.Request) {
		if !r.User.IsAuthenticated() {
			Unauthorized(r, "You must be logged in to view this page")
			return
		}
		if !r.User.IsAdmin() {
			Unauthorized(r, "You must be an administrator to view this page")
			return
		}

		tpl, name, err := templateManager().Get(template_list_objects)
		if err != nil {
			r.Logger.Errorf("error getting template: %s", err.Error())
			r.Error(500, "Internal Server Error")
			return
		}

		var page_string = r.QueryParams.Get("page")
		var limit_string = r.QueryParams.Get("limit")
		var page int = httputils.MustInt(page_string, 1, 9999, 1)
		var limit int = httputils.MustInt(limit_string, 10, 9999, 25)

		items, totalCount, err := m.Options.Model.List(page, limit)
		if err != nil {
			r.Logger.Errorf("error listing %s: %s", namer.GetModelName(m.Options.Model), err.Error())
			r.Error(500, "Internal Server Error")
			return
		}
		var totalPages = int64(totalCount) / int64(limit)
		if int64(totalCount)%int64(limit) != 0 {
			totalPages++
		}
		var rows = make([][]any, len(items)+1)
		var rowNames = orderedmap.New[string, any]() // Map of field Name to display name
		for i, item := range items {
			var row = make([]any, len(m.Options.ListFields)+1)
			var valueOf = reflect.ValueOf(item)

			if valueOf.Kind() == reflect.Ptr {
				valueOf = valueOf.Elem()
			}

			var stringID = item.StringID()
			row[0] = template.HTML(fmt.Sprintf("<a href=\"%s\">%s</a>",
				m.Model.URL_Update.Format(stringID),
				stringID))

			rowNames.Set("ID", "ID")
		inner:
			for j, field := range m.Options.ListFields {
				var fieldIndex = j + 1
				var val = valueOf.FieldByName(field)
				var sField, ok = valueOf.Type().FieldByName(field)
				if !ok {
					panic(fmt.Sprintf("field %s not found in type %T", field, item))
				}

				var t = sField.Tag.Get("admin-form")
				if t == "-" {
					continue inner
				}

				var tagMap = tags.ParseWithDelimiter(t, ";", "=", ",")
				if val.Kind() == reflect.Ptr {
					val = val.Elem()
				}

				var rowName = tagMap.GetSingle("name")
				if rowName == "" {
					rowName = sField.Name
				}
				rowNames.Set(sField.Name, rowName)

				permissions, ok := tagMap.GetOK("permissions")
				if ok {
					if !r.User.HasPermissions(permissions...) {
						row[fieldIndex] = "**********"
						continue inner
					}
				}

				if valueOf.CanAddr() {
					var method = valueOf.MethodByName(fmt.Sprintf("Get%sDisplay", field))
					if method.IsValid() {
						var ret, ok = method.Interface().(func() string)
						if !ok {
							panic(fmt.Sprintf("invalid method signature for Get%sDisplay", field))
						}
						row[fieldIndex] = ret()
						continue inner
					}
				}

				switch iFace := val.Interface().(type) {
				case AdminDisplayer:
					row[fieldIndex] = iFace.AdminDisplay()
				case time.Time:
					row[fieldIndex] = iFace.Format("2006-01-02 15:04:05")
					continue inner
				case fmt.Stringer:
					row[fieldIndex] = iFace.String()
					continue inner
				case error:
					row[fieldIndex] = iFace.Error()
					continue inner
				}

				switch val.Kind() {
				case reflect.Struct, reflect.Slice, reflect.Map:
					panic(fmt.Errorf("non-primitive fields must implement AdminDisplayer interface: %s", field))
				}

				row[fieldIndex] = val.Interface()
			}
			rows[i+1] = row
		}

		rows[0] = rowNames.InOrder()

		var pastItemCount = int64(limit * page)
		if pastItemCount > totalCount {
			pastItemCount = totalCount
		}
		if pastItemCount < 0 {
			pastItemCount = 0
		}

		r.Data.Set("items", rows)
		r.Data.Set("page", page)
		r.Data.Set("limit", limit)
		r.Data.Set("totalPages", totalPages)
		r.Data.Set("items_in_past", pastItemCount)
		r.Data.Set("total_item_count", totalCount)
		r.Data.Set("model", m.Model)
		r.Data.Set("current_url", r.Request.URL.String())
		r.Data.Set("limit_choices", []int{10, 25, 50, 100})

		err = response.Template(r, tpl, name)
		if err != nil {
			r.Logger.Errorf("error rendering template: %s", err.Error())
			renderError(r, "Error rendnering template", 500, err)
			return
		}
	}
}

// Unauthorized redirects the user to the unauthorized page.
// It will also log a stacktrace of the code that called this function.
func Unauthorized(r *request.Request, msg ...string) {
	// Runtime.Caller
	var err = errors.New("Unauthorized access")
	var callInfo = tracer.TraceSafe(err, 16, 1)

	AdminSite_Logger.Debugf("Unauthorized access by: %s\n", r.User)
	for _, c := range callInfo.Trace() {
		AdminSite_Logger.Debugf("\tUnauthorized access from: %s:%d\n", c.File, c.Line)
	}

	if len(msg) > 0 {
		for _, m := range msg {
			r.Data.AddMessage("error", m)
		}
	}
	r.Redirect(
		adminSite_Route.URL(router.GET, "admin:unauthorized").Format(),
		http.StatusFound,
		r.Request.URL.String())
}

func unauthorizedView(rq *request.Request) {
	var template, name, err = templateManager().Get(template_unauthorized)
	if err != nil {
		AdminSite_Logger.Critical(err)
		renderError(rq, "Error loading template", 500, err)
		return
	}

	rq.Data.Set("title", "Unauthorized")

	rq.ReSetNext()

	err = response.Template(rq, template, name)
	if err != nil {
		AdminSite_Logger.Critical(err)
		renderError(rq, "Error rendering template", 500, err)
	}
}

func renderError(rq *request.Request, err string, code int, errDetail error) {
	rq.Response.Buffer().Reset()
	var template, name, tErr = templateManager().Get(template_errors)
	if tErr != nil {
		AdminSite_Logger.Critical(tErr)
		rq.Error(500, http.StatusText(500))
		return
	}

	rq.Data.Set("title", "Error")
	rq.Data.Set("error", err)
	rq.Data.Set("error_code", code)
	rq.Data.Set("detail", errDetail.Error())

	tErr = response.Template(rq, template, name)
	if tErr != nil {
		AdminSite_Logger.Critical(tErr)
		rq.Error(500, http.StatusText(500))
	}
}
