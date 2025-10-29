package blocks

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/url"
	"reflect"
	"strings"

	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/internal/django_reflect"
	"github.com/Nigel2392/go-telepath/telepath"
)

type Block interface {
	Name() string
	SetName(name string)
	SetLabel(label any)
	SetHelpText(helpText any)
	SetDefault(def interface{})
	Label(ctx context.Context) string
	HelpText(ctx context.Context) string
	Field() fields.Field
	SetField(field fields.Field)
	ValueAtPath(bound BoundBlockValue, parts []string) (interface{}, error)
	ValueFromDB(value json.RawMessage) (interface{}, error)
	RenderForm(ctx context.Context, w io.Writer, id, name string, value interface{}, errors []error, context ctx.Context) error
	Render(ctx context.Context, w io.Writer, value interface{}, context ctx.Context) error
	GetDefault() interface{}
	telepath.AdapterGetter
	media.MediaDefiner
	widgets.FormValuer
	forms.Validator
	forms.Cleaner
}

var _ Block = (*BaseBlock)(nil)

type BaseBlock struct {
	Name_      string                                     `json:"-"`
	Template   string                                     `json:"-"`
	FormField  fields.Field                               `json:"-"`
	Validators []func(context.Context, interface{}) error `json:"-"`
	Default    func() interface{}                         `json:"-"`

	DataType        any
	ValueFromDBFunc func(b *BaseBlock, j json.RawMessage) (interface{}, error) `json:"-"`
	LabelFunc       func(ctx context.Context) string                           `json:"-"`
	HelpFunc        func(ctx context.Context) string                           `json:"-"`
}

func (b *BaseBlock) Name() string {
	return b.Name_
}

func (b *BaseBlock) SetName(name string) {
	b.Name_ = name
	b.Field().SetName(name)
}

func (b *BaseBlock) SetField(field fields.Field) {
	b.FormField = field
	field.SetName(b.Name_)
}

func (b *BaseBlock) ValueFromDB(value json.RawMessage) (interface{}, error) {
	if b.ValueFromDBFunc != nil {
		var v, err = b.ValueFromDBFunc(b, value)
		return newFieldBlockValue(b, v), err
		// return v, err
	}

	if b.DataType != nil {
		var rT = reflect.TypeOf(b.DataType)
		var isPtr = rT.Kind() == reflect.Ptr
		if isPtr {
			rT = rT.Elem()
		}
		var v = reflect.New(rT)
		if err := json.Unmarshal(value, v.Interface()); err != nil {
			return nil, err
		}
		var val any
		if isPtr {
			val = v.Interface()
		} else {
			val = v.Elem().Interface()
		}
		// return val, nil
		return newFieldBlockValue(b, val), nil
	}

	if len(value) > 0 {
		var v interface{}
		if err := json.Unmarshal(value, &v); err != nil {
			return nil, err
		}
		return v, nil
	}
	return nil, nil
}

func (b *BaseBlock) ValueAtPath(bound BoundBlockValue, parts []string) (interface{}, error) {
	return bound.Data(), nil
}

func (b *BaseBlock) SetLabel(label any) {
	b.LabelFunc = trans.GetTextFunc(label)
}

func (b *BaseBlock) SetHelpText(helpText any) {
	b.HelpFunc = trans.GetTextFunc(helpText)
}

func (b *BaseBlock) Field() fields.Field {
	if b.FormField == nil {
		b.SetField(fields.CharField())
	}
	return b.FormField
}

func (b *BaseBlock) FormContext(name string, value interface{}, context ctx.Context) *BlockContext {
	var blockCtx = NewBlockContext(b, context)
	blockCtx.Name = name
	blockCtx.Value = value
	return blockCtx
}

func (b *BaseBlock) RenderForm(ctx context.Context, w io.Writer, id, name string, value interface{}, errors []error, context ctx.Context) error {
	var blockCtx = b.FormContext(name, value, context)
	blockCtx.Errors = errors
	return b.Field().Widget().RenderWithErrors(
		ctx, w, id, name, value, errors, blockCtx.Attrs,
		b.Field().Widget().GetContextData(ctx, id, name, value, blockCtx.Attrs),
	)
}

var htmlEscaper = strings.NewReplacer(
	`&`, "&amp;",
	`'`, "&#39;", // "&#39;" is shorter than "&apos;" and apos was not in HTML until HTML5.
	`<`, "&lt;",
	`>`, "&gt;",
	`"`, "&#34;", // "&#34;" is shorter than "&quot;".
)

func (b *BaseBlock) Render(ctx context.Context, w io.Writer, value interface{}, context ctx.Context) error {
	var blockCtx = NewBlockContext(b, context)
	if b.Template != "" {
		blockCtx.Value = value
		return tpl.FRender(w, blockCtx, b.Template)
	}

	v, ok := value.(*FieldBlockValue)
	if !ok {
		return fmt.Errorf("value must be a *FieldBlockValue")
	}

	switch val := v.V.(type) {
	case RenderableValue:
		return val.Render(ctx, w, context)
	case interface{ HTML() template.HTML }:
		_, err := io.WriteString(w, string(val.HTML()))
		return err
	default:
		var str = fmt.Sprintf("%v", val)
		_, err := io.WriteString(w, htmlEscaper.Replace(str))
		return err
	}
}

func (b *BaseBlock) Label(ctx context.Context) string {
	if b.LabelFunc != nil {
		return b.LabelFunc(ctx)
	}
	var label = b.Field().Label(ctx)
	if label == "" {
		label = b.Name()
	}
	return label
}

func (b *BaseBlock) HelpText(ctx context.Context) string {
	if b.HelpFunc != nil {
		return b.HelpFunc(ctx)
	}
	return ""
}

func (b *BaseBlock) SetDefault(def interface{}) {
	var rv = reflect.ValueOf(def)
	if rv.Kind() == reflect.Func {
		var err error
		b.Default, err = django_reflect.CastFunc[func() interface{}](rv)
		if err != nil {
			panic(err)
		}
	} else {
		b.Default = func() interface{} {
			return def
		}
	}
}

func (b *BaseBlock) GetDefault() interface{} {
	if b.Default != nil {
		return b.Default()
	}
	return b.Field().Default()
}

func (b *BaseBlock) ValueToGo(value interface{}) (interface{}, error) {
	return b.Field().ValueToGo(value)
}

func (b *BaseBlock) ValueToForm(value interface{}) interface{} {
	if b, ok := value.(BoundBlockValue); ok {
		value = b.Data()
	}
	return b.Field().ValueToForm(value)
}

func (b *BaseBlock) ValueOmittedFromData(ctx context.Context, data url.Values, files map[string][]filesystem.FileHeader, name string) bool {
	return b.Field().Widget().ValueOmittedFromData(ctx, data, files, name)
}

func (b *BaseBlock) ValueFromDataDict(ctx context.Context, data url.Values, files map[string][]filesystem.FileHeader, name string) (interface{}, []error) {
	return b.Field().Widget().ValueFromDataDict(ctx, data, files, name)
}

func (b *BaseBlock) Validate(ctx context.Context, value interface{}) []error {

	for _, validator := range b.Validators {
		if err := validator(ctx, value); err != nil {
			return []error{err}
		}
	}

	return b.Field().Validate(ctx, value)
}

func (b *BaseBlock) Clean(ctx context.Context, value interface{}) (interface{}, error) {
	return b.Field().Clean(ctx, value)
}

func (b *BaseBlock) Media() media.Media {
	return b.Field().Widget().Media()
}

func (b *BaseBlock) Adapter(ctx context.Context) telepath.Adapter {
	return nil
}

func NewBaseBlock(opts ...OptFunc[*BaseBlock]) *BaseBlock {
	var b = &BaseBlock{}
	runOpts(opts, b)
	return b
}
