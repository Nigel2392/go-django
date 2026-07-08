package views_test

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Nigel2392/go-django/djester"
	"github.com/Nigel2392/go-django/djester/quest"
	"github.com/Nigel2392/go-django/examples/todoapp/todos"
	"github.com/Nigel2392/go-django/examples/views"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	dj_views "github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/mux"
)

func TestExampleViews(t *testing.T) {
	mapFS := fstest.MapFS{
		"todos/detail.html":         &fstest.MapFile{Data: []byte(`{{template "base" .}}{{define "content"}}DETAIL_VIEW_HTML{{end}}`)},
		"todos/list.html":           &fstest.MapFile{Data: []byte(`{{template "base" .}}{{define "content"}}LIST_VIEW_HTML{{end}}`)},
		"todos/list_columns.html":   &fstest.MapFile{Data: []byte(`{{template "base" .}}{{define "content"}}LIST_COLUMNS_VIEW_HTML{{end}}`)},
		"todos/create.html":         &fstest.MapFile{Data: []byte(`{{template "base" .}}{{define "content"}}CREATE_VIEW_HTML{{end}}`)},
		"todos/update.html":         &fstest.MapFile{Data: []byte(`{{template "base" .}}{{define "content"}}UPDATE_VIEW_HTML{{end}}`)},
		"todos/delete_confirm.html": &fstest.MapFile{Data: []byte(`{{template "base" .}}{{define "content"}}DELETE_VIEW_HTML{{end}}`)},
		"contact/form.html":         &fstest.MapFile{Data: []byte(`{{template "base" .}}{{define "content"}}FORM_VIEW_HTML{{end}}`)},
		"pages/about.html":          &fstest.MapFile{Data: []byte(`{{template "base" .}}{{define "content"}}ABOUT_VIEW_HTML{{end}}`)},
		"errors/custom_error.html":  &fstest.MapFile{Data: []byte(`{{template "base" .}}{{define "content"}}ERROR_VIEW_HTML{{end}}`)},
		"base.tmpl":                 &fstest.MapFile{Data: []byte(`{{define "base"}}{{block "content" .}}{{end}}{{end}}`)},
	}

	app := apps.NewAppConfig("examples")
	app.TemplateConfig = &tpl.Config{
		AppName: "examples",
		FS:      mapFS,
		Bases: []string{
			"base.tmpl",
		},
		Matches: func(path string) bool { return true },
	}

	var tables *quest.DBTables[*testing.T]

	app.Init = func(settings django.Settings) error {
		tables = quest.Table(t,
			&todos.Todo{},
		)

		tables.Create()
		return nil
	}

	app.Routing = func(m mux.Multiplexer) {
		m.Handle("GET", "/todos", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
		m.Handle("GET", "/contact/success", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
		m.Handle("GET", "/new-route", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))

		m.Handle("GET", "/detail/<<id>>", dj_views.Serve(views.MyDetailView))
		m.Handle("GET", "/list", dj_views.Serve(views.MyListView))
		m.Handle("GET", "/list_cols", dj_views.Serve(views.MyListViewWithColumns))
		m.Handle("GET", "/create", dj_views.Serve(views.TodoCreateView))
		m.Handle("POST", "/create", dj_views.Serve(views.TodoCreateView))
		m.Handle("GET", "/update/<<id>>", dj_views.Serve(views.TodoUpdateView))
		m.Handle("POST", "/update/<<id>>", dj_views.Serve(views.TodoUpdateView))
		m.Handle("GET", "/delete/<<id>>", dj_views.Serve(views.TodoDeleteView))
		m.Handle("POST", "/delete/<<id>>", dj_views.Serve(views.TodoDeleteView))
		m.Handle("GET", "/form", dj_views.Serve(views.ContactFormView))
		m.Handle("POST", "/form", dj_views.Serve(views.ContactFormView))
		m.Handle("GET", "/about", dj_views.Serve(views.AboutTemplateView))
		m.Handle("GET", "/json", dj_views.Serve(views.APIStatusView))
		m.Handle("GET", "/redirect", dj_views.Serve(views.OldRouteView))
		m.Handle("GET", "/trigger_error", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			views.GlobalErrorView.HandleError(w, r, fmt.Errorf("triggered manually"), 500)
		}))
	}

	tester := &djester.Tester{
		Settings: map[string]any{
			django.APPVAR_ALLOWED_HOSTS:  []string{"*"},
			django.APPVAR_DISABLE_NOSURF: true,
			django.APPVAR_DEBUG:          true,
			django.APPVAR_RECOVERER:      false,
		},
		Apps: []djester.AppInitFuncOrAppConfig{
			app,
		},
		Database: djester.Database{
			Engine:           "sqlite3",
			ConnectionString: ":memory:",
		},
		BeforeSetup: func(d *djester.Tester) error {
			attrs.RegisterModel(&todos.Todo{})
			return nil
		},
		Tests: []djester.Test{
			&djester.BasicTest{
				Label: "DetailView",
				Function: func(d *djester.Tester, t *testing.T) {
					resp, err := d.Get("/detail/1", nil, nil)
					assert := d.Assert(t, true)
					assert.AssertNil(err)
					assert.AssertEqual(http.StatusOK, resp.StatusCode)
					b, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					assert.Assert(strings.Contains(string(b), "DETAIL_VIEW_HTML"), "body missing template text")
				},
			},
			&djester.BasicTest{
				Label: "ListView",
				Function: func(d *djester.Tester, t *testing.T) {
					resp, err := d.Get("/list", nil, nil)
					assert := d.Assert(t, true)
					assert.AssertNil(err)
					assert.AssertEqual(http.StatusOK, resp.StatusCode)
					b, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					assert.Assert(strings.Contains(string(b), "LIST_VIEW_HTML"), "body missing template text")
				},
			},
			&djester.BasicTest{
				Label: "ListViewColumns",
				Function: func(d *djester.Tester, t *testing.T) {
					resp, err := d.Get("/list_cols", nil, nil)
					assert := d.Assert(t, true)
					assert.AssertNil(err)
					assert.AssertEqual(http.StatusOK, resp.StatusCode)
					b, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					assert.Assert(strings.Contains(string(b), "LIST_COLUMNS_VIEW_HTML"), "body missing template text")
				},
			},
			&djester.BasicTest{
				Label: "CreateView GET",
				Function: func(d *djester.Tester, t *testing.T) {
					resp, err := d.Get("/create", nil, nil)
					assert := d.Assert(t, true)
					assert.AssertNil(err)
					assert.AssertEqual(http.StatusOK, resp.StatusCode)
					b, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					assert.Assert(strings.Contains(string(b), "CREATE_VIEW_HTML"), "body missing template text")
				},
			},
			&djester.BasicTest{
				Label: "CreateView POST",
				Function: func(d *djester.Tester, t *testing.T) {
					resp, err := d.PostForm("/create", nil, nil, map[string]interface{}{"title": "New Todo"})
					assert := d.Assert(t, true)
					assert.AssertNil(err)
					assert.AssertEqual(http.StatusOK, resp.StatusCode)
					assert.Assert(strings.HasSuffix(resp.Request.URL.Path, "/todos"), "expected redirect to /todos, got %s", resp.Request.URL.Path)
				},
			},
			&djester.BasicTest{
				Label: "UpdateView GET",
				Function: func(d *djester.Tester, t *testing.T) {
					resp, err := d.Get("/update/1", nil, nil)
					assert := d.Assert(t, true)
					assert.AssertNil(err)
					assert.AssertEqual(http.StatusOK, resp.StatusCode)
					b, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					assert.Assert(strings.Contains(string(b), "UPDATE_VIEW_HTML"), "body missing template text")
				},
			},
			&djester.BasicTest{
				Label: "UpdateView POST",
				Function: func(d *djester.Tester, t *testing.T) {
					resp, err := d.PostForm("/update/1", nil, nil, map[string]interface{}{"title": "Updated Todo"})
					assert := d.Assert(t, true)
					assert.AssertNil(err)
					assert.AssertEqual(http.StatusOK, resp.StatusCode)
					assert.Assert(strings.HasSuffix(resp.Request.URL.Path, "/todos"), "expected redirect to /todos, got %s", resp.Request.URL.Path)
				},
			},
			&djester.BasicTest{
				Label: "DeleteView GET",
				Function: func(d *djester.Tester, t *testing.T) {
					resp, err := d.Get("/delete/1", nil, nil)
					assert := d.Assert(t, true)
					assert.AssertNil(err)
					assert.AssertEqual(http.StatusOK, resp.StatusCode)
					b, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					assert.Assert(strings.Contains(string(b), "DELETE_VIEW_HTML"), "body missing template text")
				},
			},
			&djester.BasicTest{
				Label: "DeleteView POST",
				Function: func(d *djester.Tester, t *testing.T) {
					resp, err := d.Post("/delete/1", nil, nil, strings.NewReader(""))
					assert := d.Assert(t, true)
					assert.AssertNil(err)
					assert.AssertEqual(http.StatusOK, resp.StatusCode)
					assert.Assert(strings.HasSuffix(resp.Request.URL.Path, "/todos"), "expected redirect to /todos, got %s", resp.Request.URL.Path)
				},
			},
			&djester.BasicTest{
				Label: "FormView GET",
				Function: func(d *djester.Tester, t *testing.T) {
					resp, err := d.Get("/form", nil, nil)
					assert := d.Assert(t, true)
					assert.AssertNil(err)
					assert.AssertEqual(http.StatusOK, resp.StatusCode)
					b, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					assert.Assert(strings.Contains(string(b), "FORM_VIEW_HTML"), "body missing template text")
				},
			},
			&djester.BasicTest{
				Label: "FormView POST",
				Function: func(d *djester.Tester, t *testing.T) {
					resp, err := d.PostForm("/form", nil, nil, map[string]interface{}{"email": "test@test.com", "message": "hello"})
					assert := d.Assert(t, true)
					assert.AssertNil(err)
					assert.AssertEqual(http.StatusOK, resp.StatusCode)
					assert.Assert(strings.HasSuffix(resp.Request.URL.Path, "/contact/success"), "expected redirect to /contact/success, got %s", resp.Request.URL.Path)
				},
			},
			&djester.BasicTest{
				Label: "TemplateView",
				Function: func(d *djester.Tester, t *testing.T) {
					resp, err := d.Get("/about", nil, nil)
					assert := d.Assert(t, true)
					assert.AssertNil(err)
					assert.AssertEqual(http.StatusOK, resp.StatusCode)
					b, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					assert.Assert(strings.Contains(string(b), "ABOUT_VIEW_HTML"), "body missing template text")
				},
			},
			&djester.BasicTest{
				Label: "ErrorView",
				Function: func(d *djester.Tester, t *testing.T) {
					resp, err := d.Get("/trigger_error", nil, nil)
					assert := d.Assert(t, true)
					assert.AssertNil(err)
					assert.AssertEqual(500, resp.StatusCode)
					b, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					assert.Assert(strings.Contains(string(b), "ERROR_VIEW_HTML"), "body missing template text")
				},
			},
			&djester.BasicTest{
				Label: "JSONView",
				Function: func(d *djester.Tester, t *testing.T) {
					resp, err := d.Get("/json", nil, nil)
					assert := d.Assert(t, true)
					assert.AssertNil(err)
					assert.AssertEqual(http.StatusOK, resp.StatusCode)
					b, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					assert.Assert(strings.Contains(string(b), `"status":"ok"`), "missing json key")
				},
			},
			&djester.BasicTest{
				Label: "RedirectView",
				Function: func(d *djester.Tester, t *testing.T) {
					resp, err := d.Get("/redirect", nil, nil)
					assert := d.Assert(t, true)
					assert.AssertNil(err)
					assert.AssertEqual(http.StatusOK, resp.StatusCode)
					assert.Assert(strings.HasSuffix(resp.Request.URL.Path, "/new-route"), "expected redirect to /new-route, got %s", resp.Request.URL.Path)
				},
			},
		},
	}

	tester.Test(t)

	tables.Drop()
}
