package admin

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"reflect"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/core/httputils"
	"github.com/Nigel2392/go-django/core/models/modelutils"
	"github.com/Nigel2392/orderedmap"
	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/router/v3/templates"
	"github.com/Nigel2392/router/v3/templates/extensions"
)

func init() {
	var TemplateFS, err = fs.Sub(templateFileSystem, "assets/templates")
	if err != nil {
		panic(err)
	}
	templateFS = TemplateFS
	adminSite_Route.Use(
		adminSiteMiddleware,
	)
	adminSite_Route.Get("", router.HandleFunc(indexView), "admin:index")

	var static = adminSite_Route.Get("/static/<<any>>",
		router.NewFSHandler(fmt.Sprintf("%s/static/", AdminSite_URL), staticFileSystem),
		"static")
	static.(*router.Route).DisableMiddleware()
}

var (
	AdminSite_Name                  = "Admin"
	AdminSite_URL                   = "/admin"
	AdminSite_Logger request.Logger = &request.NopLogger{}

	AdminSite_ExtensionTemplateManager *templates.Manager
	AdminSite_ExtensionOptions         *extensions.Options

	adminSite_TemplateMgr *templates.Manager
	adminSite_Apps        = orderedmap.New[string, *Application]()
	adminSite_Route       = router.Group(AdminSite_URL, "admin")

	adminSite_Extensions      []extensions.Extension
	adminSite_ExtensionsRoute = adminSite_Route.Group("/admin-extensions", "admin-extensions")
)

func Route() router.Registrar {
	if adminSite_Route == nil {
		panic("Admin site routes are nil!")
	}
	return adminSite_Route
}

func goback(r *request.Request) string {
	var prev = r.Request.Referer()
	if prev == "" {
		return r.URL(router.GET, "admin:index").Format()
	}
	return prev
}

var staticFileSystem, _ = fs.Sub(sfs, "assets/static")

var templateFS fs.FS

// The template file system for the admin site.
//
// This is where the admin site templates are stored.
//
//go:embed assets/templates/*
var templateFileSystem embed.FS

//go:embed assets/static/*
var sfs embed.FS

func templateManager() *templates.Manager {
	if adminSite_TemplateMgr != nil {
		return adminSite_TemplateMgr
	}
	var mgr = &templates.Manager{
		TEMPLATEFS:             templateFS,
		USE_TEMPLATE_CACHE:     false,
		BASE_TEMPLATE_SUFFIXES: []string{".html", ".tmpl"},
		BASE_TEMPLATE_DIRS:     []string{"base"},
		TEMPLATE_DIRS:          []string{"admin"},
		DEFAULT_FUNCS: template.FuncMap{
			"title": func(s any) string {
				if s == nil {
					return AdminSite_Name
				}
				switch s := s.(type) {
				case string:
					return httputils.TitleCaser.String(s)
				}
				return formatFunc(s)
			},
			"lower":  strings.ToLower,
			"upper":  strings.ToUpper,
			"div":    divideFunc,
			"max":    maxStrLenFunc,
			"format": formatFunc,
			"join":   joinFunc,
			"has_permissions": func(r request.User, perms ...string) bool {
				if len(perms) == 0 {
					return true
				}
				if r == nil {
					return false
				}
				return r.HasPermissions(perms...)
			},
		},
	}
	adminSite_TemplateMgr = mgr
	return mgr
}

// Join a list of types into a string.
func joinFunc(args ...any) string {
	var s = make([]string, len(args))
	for i, arg := range args {
		s[i] = formatFunc(arg)
	}
	return strings.Join(s, "")
}

// Format a type.
// If the type can not be formatted, fmt.Sprint.
func formatFunc(a any) string {
	if a == nil {
		return ""
	}
	if modelutils.IsModel(a) {
		return modelutils.GetModelDisplay(a, false)
	}
	switch a := a.(type) {
	case time.Time:
		return a.Format("15:04:05 02-01-2006")
	}
	var t = reflect.TypeOf(a)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		var s = reflect.ValueOf(a)
		var l = s.Len()
		var r = make([]string, l)
		for i := 0; i < l; i++ {
			r[i] = fmt.Sprint(s.Index(i).Interface())
		}
		return strings.Join(r, ", ")
	case reflect.Map:
		var s = reflect.ValueOf(a)
		var l = s.Len()
		var r = make([]string, l)
		var i = 0
		for _, k := range s.MapKeys() {
			r[i] = k.String() + ": " + fmt.Sprint(s.MapIndex(k).Interface())
			i++
		}
		return strings.Join(r, ", ")
	}
	return fmt.Sprint(a)
}

// Format a time.Time or *time.Time to a string.
// This is used to format strings accordingly.
func formatTime(t any) string {
	switch t := t.(type) {
	case time.Time:
		return t.Format("2006-01-02 15:04:05")
	case *time.Time:
		return t.Format("2006-01-02 15:04:05")
	default:
		return ""
	}
}

// Cut a string or []byte to a maximum length.
// If the string is longer than max, it will be cut and "..." will be appended.
func maxStrLenFunc(s any, max int) any {
	switch v := s.(type) {
	case string:
		if len(v) > max {
			return v[:max] + "..."
		}
		return v
	case []byte:
		if len(v) > max {
			return append(v[:max], []byte("...")...)
		}
		return v
	default:
		return s
	}
}

// Divide two numbers.
func divideFunc(a, b int) int {
	if b == 0 || a == 0 {
		return 0
	}
	return a / b
}
