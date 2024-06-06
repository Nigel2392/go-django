package blocks

import (
	"io"
	"net/url"

	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/core/tpl"
	"github.com/Nigel2392/django/forms"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/forms/media"
	"github.com/Nigel2392/django/forms/widgets"
	"github.com/Nigel2392/go-telepath/telepath"
)

type Block interface {
	Name() string
	SetName(name string)
	Label() string
	HelpText() string
	Field() fields.Field
	SetField(field fields.Field)
	RenderForm(w io.Writer, id, name string, value interface{}, errors []error, context ctx.Context) error
	Render(w io.Writer, value interface{}, context ctx.Context) error
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
	Validators []func(interface{}) error
	Default    func() interface{}

	LabelFunc func() string
	HelpFunc  func() string
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

func (b *BaseBlock) RenderForm(w io.Writer, id, name string, value interface{}, errors []error, context ctx.Context) error {
	var blockCtx = b.FormContext(name, value, context)
	blockCtx.Errors = errors
	return b.Field().Widget().RenderWithErrors(w, id, name, value, errors, blockCtx.Attrs)
}

func (b *BaseBlock) Render(w io.Writer, value interface{}, context ctx.Context) error {
	var blockCtx = NewBlockContext(b, context)
	blockCtx.Value = value
	return tpl.FRender(w, blockCtx, b.Template)
}

func (b *BaseBlock) Label() string {
	if b.LabelFunc != nil {
		return b.LabelFunc()
	}
	return b.Field().Label()
}

func (b *BaseBlock) HelpText() string {
	if b.HelpFunc != nil {
		return b.HelpFunc()
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

func (b *BaseBlock) ValueOmittedFromData(data url.Values, files map[string][]io.ReadCloser, name string) bool {
	return b.Field().Widget().ValueOmittedFromData(data, files, name)
}

func (b *BaseBlock) ValueFromDataDict(data url.Values, files map[string][]io.ReadCloser, name string) (interface{}, []error) {
	return b.Field().Widget().ValueFromDataDict(data, files, name)
}

func (b *BaseBlock) Validate(value interface{}) []error {

	for _, validator := range b.Validators {
		if err := validator(value); err != nil {
			return []error{err}
		}
	}

	return b.Field().Validate(value)
}

func (b *BaseBlock) Clean(value interface{}) (interface{}, error) {
	return b.Field().Clean(value)
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
