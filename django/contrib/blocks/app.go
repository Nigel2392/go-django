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
				assert.False(
					v.Type() == blockTyp,
					"field must be type of block and not interface{}",
				)

				if v.Type().Implements(blockTyp) {
					var (
						name        = f.Name()
						getBlockDef = fmt.Sprintf("Get%sDef", name)
						instance    = f.Instance()
						method, ok  = attrs.Method[func() Block](instance, getBlockDef)
					)
					if !ok {
						return nil, false
					}

					var blockDef = method()

					var vOf = reflect.ValueOf(blockDef)
					v.Set(vOf)

					return BlockField(blockDef, opts...), true
				}

				return nil, false
			}),
		)

		goldcrest.Register(
			attrs.DefaultForType, 0,
			attrs.DefaultGetter(func(f attrs.Field, t reflect.Type, v reflect.Value) (any, bool) {

				if v.Type().Implements(blockTyp) {
					var (
						name        = f.Name()
						getBlockDef = fmt.Sprintf("Get%sDef", name)
						instance    = f.Instance()
						method, ok  = attrs.Method[func() Block](instance, getBlockDef)
					)
					if !ok {
						return nil, false
					}

					var blockDef = method()

					attrs.Set(instance, name, blockDef)

					return blockDef, true
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

		tpl.Add(tpl.Config{
			AppName: "blocks",
			FS:      templateFS,
			Bases: []string{
				"blocks/base.tmpl",
			},
			Matches: tpl.MatchAnd(
				tpl.MatchPrefix("blocks"),
				tpl.MatchOr(
					tpl.MatchSuffix(".html"),
					tpl.MatchSuffix(".tmpl"),
				),
			),
		})

		return nil
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
