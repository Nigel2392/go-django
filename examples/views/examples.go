package views

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"github.com/Nigel2392/go-django/examples/todoapp/todos"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-django/src/views/list"
)

// 1. DetailView
var MyDetailView = &views.DetailView[todos.Todo]{
	BaseView: views.BaseView{
		AllowedMethods:  []string{http.MethodGet},
		TemplateName:    "todos/detail.html",
		BaseTemplateKey: "base",
	},
	ContextName: "todo",
	URLArgName:  "id",
	GetObjectFn: func(req *http.Request, urlArg string) (todos.Todo, error) {
		return todos.Todo{ID: 1, Title: "Test Detail", Description: "Desc", Done: true}, nil
	},
}

// 2. ListView
var MyListView = &list.View[*todos.Todo]{
	Model:           &todos.Todo{},
	AllowedMethods:  []string{http.MethodGet},
	TemplateName:    "todos/list.html",
	BaseTemplateKey: "base",
	AmountParam:     "amount",
	PageParam:       "page",
	DefaultAmount:   10,
	MaxAmount:       100,
	QuerySet: func(r *http.Request) *queries.QuerySet[*todos.Todo] {
		// Mock query set
		return queries.GetQuerySet(&todos.Todo{})
	},
	ListColumns: []list.ListColumn[*todos.Todo]{
		list.Column[*todos.Todo](trans.S("Title"), "Title"),
	},
}

// 3. ListViewColumns
var MyListViewWithColumns = &list.View[*todos.Todo]{
	Model:           &todos.Todo{},
	AllowedMethods:  []string{http.MethodGet},
	TemplateName:    "todos/list_columns.html",
	BaseTemplateKey: "base",
	AmountParam:     "limit",
	PageParam:       "page",
	DefaultAmount:   10,
	MaxAmount:       100,
	QuerySet: func(r *http.Request) *queries.QuerySet[*todos.Todo] {
		return queries.GetQuerySet(&todos.Todo{})
	},
	ListColumns: []list.ListColumn[*todos.Todo]{
		list.Column[*todos.Todo](trans.S("Title"), "Title"),
		list.BooleanFieldColumn[*todos.Todo](trans.S("Done"), "Done"),
		list.FuncColumn(trans.S("Description Prefix"), func(r *http.Request, defs attrs.Definitions, row *todos.Todo) interface{} {
			if len(row.Description) > 10 {
				return row.Description[:10] + "..."
			}
			return row.Description
		}),
		list.HTMLColumn(trans.S("Actions"), func(r *http.Request, defs attrs.Definitions, row *todos.Todo) template.HTML {
			return template.HTML(fmt.Sprintf(`<a href="/todos/%d/edit">Edit</a>`, row.ID))
		}),
	},
}

// 4. CreateView
type TodoForm struct {
	forms.BaseForm
	Title       string `form:"title" validate:"required"`
	Description string `form:"description"`
}

var TodoCreateView = &views.FormView[*TodoForm]{
	BaseView: views.BaseView{
		AllowedMethods:  []string{http.MethodGet, http.MethodPost},
		TemplateName:    "todos/create.html",
		BaseTemplateKey: "base",
	},
	GetFormFn: func(req *http.Request) *TodoForm {
		var form = &TodoForm{}
		form.WithContext(req.Context())
		return form
	},
	ValidFn: func(req *http.Request, form *TodoForm) error {
		return nil
	},
	SuccessFn: func(w http.ResponseWriter, req *http.Request, form *TodoForm) {
		http.Redirect(w, req, "/todos", http.StatusSeeOther)
	},
}

// 5. UpdateView
type TodoUpdateForm struct {
	forms.BaseForm
	Title string `form:"title" validate:"required"`
}

var TodoUpdateView = &views.FormView[*TodoUpdateForm]{
	BaseView: views.BaseView{
		AllowedMethods:  []string{http.MethodGet, http.MethodPost},
		TemplateName:    "todos/update.html",
		BaseTemplateKey: "base",
	},
	GetFormFn: func(req *http.Request) *TodoUpdateForm {
		var form = &TodoUpdateForm{}
		form.WithContext(req.Context())
		return form
	},
	GetInitialFn: func(req *http.Request) map[string]interface{} {
		return map[string]interface{}{
			"title": "Existing DB Title",
		}
	},
	ValidFn: func(req *http.Request, form *TodoUpdateForm) error {
		return nil
	},
	SuccessFn: func(w http.ResponseWriter, req *http.Request, form *TodoUpdateForm) {
		http.Redirect(w, req, "/todos", http.StatusSeeOther)
	},
}

// 6. DeleteView
type MyDeleteView struct {
	views.DetailView[todos.Todo]
}

func (v *MyDeleteView) ServePOST(w http.ResponseWriter, req *http.Request) {
	_, err := v.GetURLArg(req)
	if err != nil {
		except.Fail(http.StatusBadRequest, err)
		return
	}
	http.Redirect(w, req, "/todos", http.StatusSeeOther)
}

var TodoDeleteView = &MyDeleteView{
	DetailView: views.DetailView[todos.Todo]{
		BaseView: views.BaseView{
			AllowedMethods:  []string{http.MethodGet, http.MethodPost},
			TemplateName:    "todos/delete_confirm.html",
			BaseTemplateKey: "base",
		},
		ContextName: "todo",
		URLArgName:  "id",
		GetObjectFn: func(req *http.Request, urlArg string) (todos.Todo, error) {
			return todos.Todo{ID: 1, Title: "Test Delete"}, nil
		},
	},
}

// 7. FormView
type ContactForm struct {
	forms.BaseForm
	Email   string `form:"email" validate:"required,email"`
	Message string `form:"message" validate:"required"`
}

var ContactFormView = &views.FormView[*ContactForm]{
	BaseView: views.BaseView{
		AllowedMethods:  []string{http.MethodGet, http.MethodPost},
		TemplateName:    "contact/form.html",
		BaseTemplateKey: "base",
	},
	GetFormFn: func(req *http.Request) *ContactForm {
		var form = &ContactForm{}
		form.WithContext(req.Context())
		return form
	},
	ValidFn: func(req *http.Request, form *ContactForm) error {
		return nil
	},
	SuccessFn: func(w http.ResponseWriter, req *http.Request, form *ContactForm) {
		http.Redirect(w, req, "/contact/success", http.StatusSeeOther)
	},
}

// 8. TemplateView
var AboutTemplateView = &views.BaseView{
	AllowedMethods:  []string{http.MethodGet},
	TemplateName:    "pages/about.html",
	BaseTemplateKey: "base",
	GetContextFn: func(req *http.Request) (ctx.Context, error) {
		c := ctx.RequestContext(req)
		c.Set("Title", "About Us")
		c.Set("Version", "1.0.0")
		return c, nil
	},
}

// 9. ErrorView
type CustomErrorView struct {
	views.BaseView
}

func (v *CustomErrorView) HandleError(w http.ResponseWriter, req *http.Request, err error, code int) {
	c, _ := v.GetContext(req)
	c.Set("ErrorCode", code)
	c.Set("ErrorMessage", err.Error())
	w.WriteHeader(code)
	_ = v.Render(w, req, v.GetTemplates(req), c)
}

var GlobalErrorView = &CustomErrorView{
	BaseView: views.BaseView{
		TemplateName:    "errors/custom_error.html",
		BaseTemplateKey: "base",
	},
}

// 10. JSONView
type JSONView struct {
	views.BaseView
	DataFn func(req *http.Request) (interface{}, error)
}

func (v *JSONView) ServeGET(w http.ResponseWriter, req *http.Request) {
	data, err := v.DataFn(req)
	if err != nil {
		except.Fail(http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

var APIStatusView = &JSONView{
	BaseView: views.BaseView{
		AllowedMethods: []string{http.MethodGet},
	},
	DataFn: func(req *http.Request) (interface{}, error) {
		return map[string]interface{}{
			"status":  "ok",
			"version": "1.2.3",
		}, nil
	},
}

// 11. RedirectView
type RedirectView struct {
	views.BaseView
	RedirectURL string
}

func (v *RedirectView) ServeGET(w http.ResponseWriter, req *http.Request) {
	http.Redirect(w, req, v.RedirectURL, http.StatusSeeOther)
}

var OldRouteView = &RedirectView{
	BaseView: views.BaseView{
		AllowedMethods: []string{http.MethodGet},
	},
	RedirectURL: "/new-route",
}
