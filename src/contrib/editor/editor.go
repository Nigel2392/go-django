package editor

import (
	"embed"
	"fmt"
	"io/fs"
	"reflect"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/mux"
)

//go:embed static/**
//go:embed static/**/**
var _editorJS_FS embed.FS
var editorJS_FS fs.FS

const (
	ROUTE_PATH = "/__go-editorjs__"

	APPVAR_WRAP_RENDERED_BLOCKS = "APPVAR_WRAP_RENDERED_BLOCKS"
)

var (
	RENDER_ERRORS = true
)

func NewAppConfig() django.AppConfig {
	var err error
	editorJS_FS, err = fs.Sub(_editorJS_FS, "static")
	if err != nil {
		panic(err)
	}

	// Assign the editorjs field to the editorjs block data
	//
	// This will then automatically assign the appropriate form widget for the field when used in a form.
	//
	// If the struct which the field belongs to defines a `Get<FieldName>Features() []string` method,
	// then these features will be used to build the editorjs widget.
	attrs.RegisterFormFieldGetter(&EditorJSBlockData{}, func(f attrs.Field, new_field_t_indirected reflect.Type, field_v reflect.Value, opts ...func(fields.Field)) (fields.Field, bool) {
		var instance = f.Instance()
		var featuresFunc, ok = attrs.Method[func() []string](
			instance, fmt.Sprintf("Get%sFeatures", f.Name()),
		)
		var features []string = nil
		if ok {
			features = featuresFunc()
		}
		return EditorJSField(features, opts...), true
	})

	staticfiles.AddFS(
		editorJS_FS,
		filesystem.MatchAnd(
			filesystem.MatchPrefix("editorjs"),
			filesystem.MatchOr(
				filesystem.MatchExt(".js"),
				filesystem.MatchExt(".css"),
			),
		),
	)

	var appconfig = apps.NewAppConfig("editor")
	appconfig.Ready = func() error {
		var features = Features()
		var m = django.Global.Mux
		var editorNs = mux.NewRoute(mux.ANY, ROUTE_PATH, nil, "editor")
		for _, feature := range features {
			feature.OnRegister(editorNs)
		}
		if len(editorNs.Children) > 0 {
			m.AddRoute(editorNs)
		}
		return nil
	}
	return appconfig
}
