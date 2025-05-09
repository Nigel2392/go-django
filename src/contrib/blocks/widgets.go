package blocks

import (
	"io"
	"net/url"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

var _ widgets.Widget = (*BlockWidget)(nil)

type BlockWidget struct {
	BlockDef Block
}

func (bw *BlockWidget) Hide(hidden bool)                 {}
func (bw *BlockWidget) SetAttrs(attrs map[string]string) {}

func NewBlockWidget(blockDef Block) *BlockWidget {
	var bw = &BlockWidget{
		BlockDef: blockDef,
	}

	return bw
}

func (bw *BlockWidget) IsHidden() bool {
	return false
}

func (bw *BlockWidget) GetContextData(id, name string, value interface{}, attrs map[string]string) ctx.Context {
	var (
		blockCtx = NewBlockContext(
			bw.BlockDef,
			ctx.NewContext(nil),
		)
	)

	blockCtx.ID = id
	blockCtx.Name = name
	blockCtx.Value = value
	blockCtx.Attrs = attrs

	return blockCtx
}

func (bw *BlockWidget) RenderWithErrors(w io.Writer, id, name string, value interface{}, errors []error, attrs map[string]string) error {
	var ctxData = bw.GetContextData(id, name, value, attrs)

	for i, err := range errors {
		switch e := err.(type) {
		case *errs.ValidationError[string]:
			errors[i] = e.Err
		case errs.ValidationError[string]:
			errors[i] = e.Err
		}
	}

	return renderBlockWidget(w, bw, ctxData.(*BlockContext), errors)
}

func (bw *BlockWidget) FormType() string {
	return "block"
}

func (bw *BlockWidget) IdForLabel(name string) string {
	return name
}

func (bw *BlockWidget) Render(w io.Writer, id, name string, value interface{}, attrs map[string]string) error {
	return bw.RenderWithErrors(w, id, name, value, nil, attrs)
}

func (bw *BlockWidget) ValueToGo(value interface{}) (interface{}, error) {
	return bw.BlockDef.ValueToGo(value)
}

func (bw *BlockWidget) ValueToForm(value interface{}) interface{} {
	return bw.BlockDef.ValueToForm(value)
}

func (bw *BlockWidget) ValueOmittedFromData(data url.Values, files map[string][]filesystem.FileHeader, name string) bool {
	return bw.BlockDef.ValueOmittedFromData(data, files, name)
}

func (bw *BlockWidget) ValueFromDataDict(data url.Values, files map[string][]filesystem.FileHeader, name string) (interface{}, []error) {
	return bw.BlockDef.ValueFromDataDict(data, files, name)
}

func (bw *BlockWidget) Validate(value interface{}) []error {
	return bw.BlockDef.Validate(value)
}

func (bw *BlockWidget) Clean(value interface{}) (interface{}, error) {
	return bw.BlockDef.Clean(value)
}

func (bw *BlockWidget) Media() media.Media {
	var m = bw.BlockDef.Media()
	if m == nil {
		m = media.NewMedia()
	}
	m.AddJS(
		media.JS(
			django.Static("blocks/js/index.js"),
		),
	)
	return m
}
