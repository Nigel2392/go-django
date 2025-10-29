package blocks

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"reflect"
	"strings"

	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/goldcrest"
)

func methodGetBlock(model any, field attrs.FieldDefinition) (Block, error) {
	var methodName = fmt.Sprintf("Get%sBlock", field.Name())
	method, ok := attrs.Method[func() Block](model, methodName)
	if !ok {
		return nil, errors.FieldNotFound.Wrapf(
			"no %s method found on %T", methodName, model,
		)
	}
	blockDef := method()
	if blockDef == nil {
		return nil, errors.ValueError.Wrapf(
			"%s method on %T returned nil", methodName, model,
		)
	}
	return blockDef, nil
}

func init() {
	dbtype.Add(&StreamBlockValue{}, dbtype.JSON)
	dbtype.Add(&ListBlockValue{}, dbtype.JSON)

	// Assign the BlockField form field to any struct field which is of type <X>BlockData.
	//
	// This will then automatically assign the appropriate form widget for the field when used in a form.
	//
	// The struct which the field belongs to should define a `Get<FieldName>Block() Block` method,
	// which will return the block definition for the field.
	// The form field will not be set up if this method is not found.
	var getter = func(f attrs.Field, new_field_t_indirected reflect.Type, field_v reflect.Value, opts ...func(fields.Field)) (fields.Field, bool) {
		var instance = f.Instance()
		block, err := methodGetBlock(instance, f)
		if err != nil {
			logger.Error("blocks: failed to get block for field %s on %T: %v", f.Name(), instance, err)
			return nil, false
		}

		return BlockField(block, opts...), true
	}

	attrs.RegisterFormFieldGetter(&StreamBlockValue{}, getter)
	attrs.RegisterFormFieldGetter(&ListBlockValue{}, getter)
}

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
			filesystem.MatchAnd(
				filesystem.MatchPrefix("blocks"),
				filesystem.MatchOr(
					filesystem.MatchSuffix(".css"),
					filesystem.MatchSuffix(".js"),
					filesystem.MatchSuffix(".png"),
					filesystem.MatchSuffix(".jpg"),
					filesystem.MatchSuffix(".jpeg"),
					filesystem.MatchSuffix(".svg"),
				),
			),
		)

		tpl.Add(tpl.Config{
			AppName: "blocks",
			FS:      templateFS,
			Bases: []string{
				"blocks/base.tmpl",
			},
			Matches: filesystem.MatchAnd(
				filesystem.MatchPrefix("blocks"),
				filesystem.MatchOr(
					filesystem.MatchSuffix(".html"),
					filesystem.MatchSuffix(".tmpl"),
				),
			),
		})

		return nil
	}

	tpl.Funcs(template.FuncMap{
		"render_block": func(rc ctx.Context, value RenderableValue) (template.HTML, error) {
			rctx, ok := rc.(*ctx.HTTPRequestContext)
			var c context.Context
			if ok {
				c = rctx.Request().Context()
			} else {
				c = context.Background()
			}
			var buf = new(bytes.Buffer)
			var err = value.Render(c, buf, rc)
			return template.HTML(buf.String()), err
		},
		"block_value": func(value any, path string) (any, error) {
			var bound BoundBlockValue
			switch v := value.(type) {
			case BoundBlockValue:
				bound = v
			case *BlockContext:
				bound, _ = v.Value.(BoundBlockValue)
			}

			if bound == nil {
				panic("block_value: value is not a BoundBlockValue")
			}

			var parts = strings.Split(path, ".")
			var v, err = bound.Block().ValueAtPath(bound, parts)
			if err != nil {
				return nil, err
			}

			if v, ok := v.(BoundBlockValue); ok {
				return v.Data(), nil
			}

			return v, nil
		},
	})

	tpl.RequestFuncs(func(r *http.Request) template.FuncMap {
		return template.FuncMap{
			"render_block": func(rc ctx.Context, value RenderableValue) (template.HTML, error) {
				var c = r.Context()
				var buf = new(bytes.Buffer)
				var err = value.Render(c, buf, rc)
				return template.HTML(buf.String()), err
			},
		}
	})

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
