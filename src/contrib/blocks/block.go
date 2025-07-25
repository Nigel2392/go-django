package blocks

import (
	"context"
	"io"
	"net/url"

	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-telepath/telepath"
)

type Block interface {
	Name() string
	SetName(name string)
	Label(ctx context.Context) string
	HelpText(ctx context.Context) string
	Field() fields.Field
	SetField(field fields.Field)
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
	Name_      string
	Template   string
	FormField  fields.Field
	Validators []func(context.Context, interface{}) error
	Default    func() interface{}

	LabelFunc func(ctx context.Context) string
	HelpFunc  func(ctx context.Context) string
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
	return b.Field().Widget().RenderWithErrors(ctx, w, id, name, value, errors, blockCtx.Attrs)
}

func (b *BaseBlock) Render(ctx context.Context, w io.Writer, value interface{}, context ctx.Context) error {
	var blockCtx = NewBlockContext(b, context)
	blockCtx.Value = value
	return tpl.FRender(w, blockCtx, b.Template)
}

func (b *BaseBlock) Label(ctx context.Context) string {
	if b.LabelFunc != nil {
		return b.LabelFunc(ctx)
	}
	return b.Field().Label(ctx)
}

func (b *BaseBlock) HelpText(ctx context.Context) string {
	if b.HelpFunc != nil {
		return b.HelpFunc(ctx)
	}
	return ""
}

func (b *BaseBlock) GetDefault() interface{} {
	if b.Default != nil {
		return b.Default()
	}
	return nil
}

func (b *BaseBlock) ValueToGo(value interface{}) (interface{}, error) {
	return b.Field().ValueToGo(value)
}

func (b *BaseBlock) ValueToForm(value interface{}) interface{} {
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

func (b *BaseBlock) Adapter() telepath.Adapter {
	return nil
}

func NewBaseBlock(opts ...OptFunc[*BaseBlock]) *BaseBlock {
	var b = &BaseBlock{}
	runOpts(opts, b)
	return b
}
