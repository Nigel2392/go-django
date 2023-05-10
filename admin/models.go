package admin

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/Nigel2392/go-django/core/httputils"
	"github.com/Nigel2392/go-django/core/modelutils/namer"
	"github.com/Nigel2392/go-django/core/views/interfaces"
	"github.com/Nigel2392/orderedmap"
	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/routevars"
	"github.com/gosimple/slug"
)

type ModelInterface[T any] interface {
	interfaces.Saver
	interfaces.Deleter
	interfaces.StringGetter[T]
	interfaces.Lister[T]
}

type Application struct {
	Name   string
	Models *orderedmap.Map[string, *Model]
	Index  router.Registrar
	URL    string
}

type Model struct {
	Name string
	Pkg  string

	URL_List   routevars.URLFormatter
	URL_Create routevars.URLFormatter
	URL_Update routevars.URLFormatter
	URL_Delete routevars.URLFormatter

	Model any

	Permissions Permissions
}

type Permissions struct {
	Create string // Name of the Create permission
	Update string // Name of the Update permission
	Delete string // Name of the Delete permission
	List   string // Name of the List permission
}

type AdminOptions[T ModelInterface[T]] struct {
	FormFields []string
	ListFields []string
	Model      T
}

type AdminDisplayer interface {
	AdminDisplay() string
}

func newPermission(typ string, s any) string {
	typeOf := reflect.TypeOf(s)
	if typeOf.Kind() == reflect.Ptr {
		typeOf = typeOf.Elem()
	}

	if typeOf.PkgPath() != "" {
		var pkgPath = httputils.GetPkgPath(s)
		return fmt.Sprintf("%s_%s_%s", typ, pkgPath, strings.ToLower(typeOf.Name()))
	}

	return fmt.Sprintf("%s_%s", typ, strings.ToLower(typeOf.Name()))
}

func Register[T ModelInterface[T]](m AdminOptions[T]) {
	if len(m.ListFields) == 0 {
		panic("ListFields must be set")
	}
	if m.FormFields == nil {
		m.FormFields = []string{"*"}
	}
	var (
		name       = namer.GetModelName(m.Model)
		pkgName    = namer.GetAppName(m.Model)
		pkgURL     = fmt.Sprintf("/%s", slug.Make(pkgName))
		urlList    = fmt.Sprintf("/%s", slug.Make(name))
		url_Create = "/create"
		url_Update = "/update/<<id:any>>"
		url_Delete = "/delete/<<id:any>>"

		base_url = filepath.Join(AdminSite_URL, pkgURL, urlList)
	)

	var model_to_register = &Model{
		Name: name,
		Pkg:  pkgName,

		URL_List:   routevars.URLFormatter(base_url),
		URL_Create: routevars.URLFormatter(filepath.Join(base_url, url_Create)),
		URL_Update: routevars.URLFormatter(filepath.Join(base_url, url_Update)),
		URL_Delete: routevars.URLFormatter(filepath.Join(base_url, url_Delete)),

		Model: m.Model,

		Permissions: Permissions{
			Create: newPermission("create", m.Model),
			Update: newPermission("update", m.Model),
			Delete: newPermission("delete", m.Model),
			List:   newPermission("list", m.Model),
		},
	}

	var viewOpts = &viewOptions[T]{
		Model:   model_to_register,
		Options: &m,
	}

	var (
		viewList   = router.HandleFunc(listView(viewOpts))
		viewCreate = newCreateView(viewOpts)
		viewUpdate = newUpdateView(viewOpts)
		viewDelete = newDeleteView(viewOpts)
	)

	var app = adminSite_Apps.Get(model_to_register.Pkg)
	if app == nil {
		app = &Application{
			Name:   model_to_register.Pkg,
			Models: orderedmap.New[string, *Model](),
			Index: adminSite_Route.HandleFunc(
				router.GET, pkgURL,
				appIndex(app), model_to_register.Pkg,
			),
		}
	}
	if app.Models.Exists(name) {
		panic(fmt.Errorf("Model %s already registered", name))
	}

	app.Models.Set(name, model_to_register)

	// Listview
	var listView = app.Index.HandleFunc(router.GET,
		urlList, viewList, name)

	// Createview
	listView.HandleFunc(router.GET,
		url_Create, viewCreate, "create")
	listView.HandleFunc(router.POST,
		url_Create, viewCreate, "create")

	// Updateview
	listView.HandleFunc(router.GET,
		url_Update, viewUpdate, "update")
	listView.HandleFunc(router.POST,
		url_Update, viewUpdate, "update")

	// Deleteview
	listView.HandleFunc(router.GET,
		url_Delete, viewDelete, "delete")
	listView.HandleFunc(router.POST,
		url_Delete, viewDelete, "delete")

	adminSite_Apps.Set(model_to_register.Pkg, app)
}
