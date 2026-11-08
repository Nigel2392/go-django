package supplier

import (
	"context"
	"fmt"
	"io"

	"github.com/Nigel2392/go-django/contrib/blocks"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-telepath/telepath"
)

type SupplierBlock struct {
	*blocks.StreamBlock
}

func (l *SupplierBlock) Media() media.Media {
	m := media.NewMedia()
	m.AddJS(media.JS(django.Static("shop/admin/js/supplier-block.js")))
	return m
}

func (l *SupplierBlock) RenderForm(ctx context.Context, w io.Writer, id, name string, value interface{}, errors []error, tplCtx ctx.Context) error {
	var (
		ctxData = blocks.NewBlockContext(l, tplCtx)
		val     *blocks.StreamBlockValue
		ok      bool
	)
	ctxData.ID = id
	ctxData.Name = name
	ctxData.Value = value

	if value == nil || value == "" {
		value = l.GetDefault()
	}

	if value == nil {
		value = blocks.NewBlockValue(l, []*blocks.StreamBlockData{})
	}

	if val, ok = value.(*blocks.StreamBlockValue); !ok {
		return fmt.Errorf("value must be a *StreamBlockValue, got %T", value)
	}

	val.BlockObject = l

	var blockErrs = blocks.NewBlockErrors[int](errors...)
	var bt, err = telepath.PackJSON(ctx, blocks.JSContext, l)
	if err != nil {
		return err
	}

	return l.RenderTempl(id, name, val, string(bt), blockErrs, ctxData).Render(ctx, w)
}

func (m *SupplierBlock) Adapter(ctx context.Context) telepath.Adapter {
	return &telepath.ObjectAdapter[*SupplierBlock]{
		JSConstructor: "django.blocks.SupplierBlock",
		GetJSArgs: func(ctx context.Context, obj *SupplierBlock) []interface{} {

			var children = make([]map[string]interface{}, 0, obj.Children.Len())
			var defaults = make(map[string]interface{})
			for head := obj.Children.Front(); head != nil; head = head.Next() {
				children = append(children, map[string]interface{}{
					"name":  head.Key,
					"block": head.Value,
				})

				var defaultVal = head.Value.GetDefault()
				if defaultVal != nil {
					defaults[head.Key] = head.Value.ValueToForm(defaultVal)
				}
			}

			return []interface{}{
				obj.Name(),
				children,
				defaults,
				map[string]interface{}{
					"label":    obj.Label(ctx),
					"helpText": obj.HelpText(ctx),
					"required": obj.Field().Required(),
					"minNum":   obj.MinNum(),
					"maxNum":   obj.MaxNum(),
				},
			}
		},
	}
}
