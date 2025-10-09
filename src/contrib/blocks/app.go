package blocks

import (
	"database/sql/driver"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"reflect"

	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/goldcrest"
)

type SubBlockData struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type StreamBlockData struct {
	Blocks []SubBlockData `json:"blocks"`
}

func (s StreamBlockData) Value() (driver.Value, error) {
	jsonData, err := json.Marshal(s)
	return jsonData, err
}

func (s *StreamBlockData) Scan(value interface{}) error {
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, s)
	case string:
		return json.Unmarshal([]byte(v), s)
	case nil:
		*s = StreamBlockData{}
		return nil
	default:
		return fmt.Errorf("cannot scan %T into StreamBlockData", value)
	}
}

type ListBlockData []*ListBlockValue

func (l ListBlockData) Value() (driver.Value, error) {
	jsonData, err := json.Marshal(l)
	return jsonData, err
}

func (l *ListBlockData) Scan(value interface{}) error {
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, l)
	case string:
		return json.Unmarshal([]byte(v), l)
	case nil:
		*l = ListBlockData{}
		return nil
	default:
		return fmt.Errorf("cannot scan %T into ListBlockData", value)
	}
}

func init() {
	dbtype.Add(&StreamBlockData{}, dbtype.JSON)
	dbtype.Add(&ListBlockData{}, dbtype.JSON)

	// Assign the BlockField form field to any struct field which is of type <X>BlockData.
	//
	// This will then automatically assign the appropriate form widget for the field when used in a form.
	//
	// The struct which the field belongs to should define a `Get<FieldName>Block() Block` method,
	// which will return the block definition for the field.
	// The form field will not be set up if this method is not found.
	var getter = func(f attrs.Field, new_field_t_indirected reflect.Type, field_v reflect.Value, opts ...func(fields.Field)) (fields.Field, bool) {
		var instance = f.Instance()
		var featureFunc, ok = attrs.Method[func() Block](
			instance, fmt.Sprintf("Get%sBlock", f.Name()),
		)
		if !ok {
			logger.Errorf("No Get%sBlock() method found on %T, cannot set up BlockField", f.Name(), instance)
			return nil, false
		}

		return BlockField(featureFunc(), opts...), true
	}

	attrs.RegisterFormFieldGetter(&StreamBlockData{}, getter)
	attrs.RegisterFormFieldGetter(ListBlockData{}, getter)
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
