package blocks

import (
	"context"
	"strings"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-telepath/telepath"
)

type widgetValueContextKey struct{}

func init() {
	var adapterFunc = func(ctx context.Context, obj widgets.Widget) []interface{} {
		var (
			sb   strings.Builder
			id   = "__ID__"
			name = "__NAME__"
			fld  = obj.Field()
		)

		var attrs map[string]string
		if fld != nil {
			attrs = fld.Attrs()
		}

		var widgetCtx = obj.GetContextData(ctx, id, name, nil, attrs)
		var err = obj.RenderWithErrors(
			ctx, &sb, id, name, nil, nil, attrs, widgetCtx,
		)
		assert.Assert(
			err == nil,
			"error rendering widget for telepath: %v", err,
		)
		return []interface{}{sb.String()}
	}

	telepath.RegisterFunc(func(ctx context.Context, obj widgets.Widget) (telepath.Adapter, bool) {
		var adapter = &telepath.ObjectAdapter[widgets.Widget]{
			GetJSArgs: adapterFunc,
		}
		switch obj.FormType() {
		case "checkbox":
			adapter.JSConstructor = "django.widgets.CheckboxInput"
		case "radio":
			adapter.JSConstructor = "django.widgets.RadioSelect"
		case "select":
			adapter.JSConstructor = "django.widgets.SelectWidget"
		default:
			adapter.JSConstructor = "django.widgets.Widget"
		}
		return adapter, true
	})
}
