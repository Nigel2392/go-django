package blocks

import (
	"embed"
	"fmt"
	"io/fs"
	"reflect"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/apps"
	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/staticfiles"
	"github.com/Nigel2392/django/core/tpl"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/goldcrest"
)

//go:embed assets/static/**
var _staticFS embed.FS

//go:embed assets/templates/**
var _templateFS embed.FS

var (
	staticFS   fs.FS
	templateFS fs.FS
	AppConfig  *apps.AppConfig
)

func NewAppConfig() *apps.AppConfig {
	var cfg = apps.NewAppConfig(
		"blocks",
	)

	cfg.Init = func(settings django.Settings) error {

		var blockTyp = reflect.TypeOf((*Block)(nil)).Elem()

		goldcrest.Register(
			attrs.HookFormFieldForType, 0,
			attrs.FormFieldGetter(func(f attrs.Field, t reflect.Type, v reflect.Value, opts ...func(fields.Field)) (fields.Field, bool) {
				if v.Type() == blockTyp {
					panic("Not implemented, field must be type of block and not interface{}")
				}

				fmt.Println("HookFormFieldForType", v.Type(), blockTyp, v.Type().Implements(blockTyp))
				if v.Type().Implements(blockTyp) {
					var name = f.Name()
					fmt.Println("Name", name)
					var getBlockDef = fmt.Sprintf("Get%sDef", name)
					fmt.Println("GetBlockDef", getBlockDef)
					var instance = f.Instance()
					var method, ok = attrs.Method[func() Block](instance, getBlockDef)
					fmt.Printf("Method %v, %v %T %v %s\n", method, ok, instance, instance, getBlockDef)
					if !ok {
						return nil, false
					}

					var blockDef = method()

					return BlockField(blockDef, opts...), true
				}

				return nil, false
			}),
		)

		staticfiles.AddFS(
			staticFS,
			tpl.MatchAnd(
				tpl.MatchPrefix("blocks"),
				tpl.MatchOr(
					tpl.MatchSuffix(".css"),
					tpl.MatchSuffix(".js"),
					tpl.MatchSuffix(".png"),
					tpl.MatchSuffix(".jpg"),
					tpl.MatchSuffix(".jpeg"),
					tpl.MatchSuffix(".svg"),
				),
			),
		)

		tpl.AddFS(
			templateFS,
			tpl.MatchAnd(
				tpl.MatchPrefix("blocks"),
				tpl.MatchOr(
					tpl.MatchSuffix(".html"),
					tpl.MatchSuffix(".tmpl"),
				),
			),
		)

		return tpl.Bases("blocks",
			"blocks/base.tmpl",
		)
	}

	AppConfig = cfg

	return cfg
}

func init() {
	var err error
	staticFS, err = fs.Sub(_staticFS, "assets/static")
	assert.Err(err)

	templateFS, err = fs.Sub(_templateFS, "assets/templates")
	assert.Err(err)
}
