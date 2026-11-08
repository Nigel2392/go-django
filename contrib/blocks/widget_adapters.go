package blocks

import (
	"context"
	"strings"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-telepath/telepath"
)

func RenderTelepathAdapterHTML(ctx context.Context, obj widgets.Widget) []interface{} {
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

	var args = []interface{}{sb.String()}

	switch obj.FormType() {
	case "file":
		args = append(args, django.Media("__MEDIA_PATH__"))
	default:
	}

	return args
}

func init() {

	telepath.RegisterFunc(func(ctx context.Context, obj widgets.Widget) (telepath.Adapter, bool) {
		var adapter = &telepath.ObjectAdapter[widgets.Widget]{
			GetJSArgs: RenderTelepathAdapterHTML,
		}
		switch obj.FormType() {
		case "checkbox":
			adapter.JSConstructor = "django.widgets.CheckboxInput"
		case "radio":
			adapter.JSConstructor = "django.widgets.RadioSelect"
		case "select":
			adapter.JSConstructor = "django.widgets.SelectWidget"
		case "file":
			adapter.JSConstructor = "django.widgets.FileWidget"
		default:
			adapter.JSConstructor = "django.widgets.Widget"
		}
		return adapter, true
	})
}
