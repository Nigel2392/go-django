package blocks

import (
	"context"

	"github.com/Nigel2392/go-telepath/telepath"
)

var (
	JSContext = telepath.NewContext()
)

func (b *FieldBlock) Adapter(ctx context.Context) telepath.Adapter {
	return &telepath.ObjectAdapter[*FieldBlock]{
		JSConstructor: "django.blocks.FieldBlock",
		GetJSArgs: func(obj *FieldBlock) []interface{} {
			return []interface{}{map[string]interface{}{
				"name":     obj.Name(),
				"label":    obj.Label(ctx),
				"helpText": obj.HelpText(ctx),
				"required": obj.Field().Required(),
				"html":     obj.RenderHTML(ctx),
			}}
		},
	}
}

func (m *StructBlock) Adapter(ctx context.Context) telepath.Adapter {
	return &telepath.ObjectAdapter[*StructBlock]{
		JSConstructor: "django.blocks.StructBlock",
		GetJSArgs: func(obj *StructBlock) []interface{} {

			var fields = make(map[string]interface{})
			for head := obj.Fields.Front(); head != nil; head = head.Next() {
				fields[head.Key] = head.Value
			}

			return []interface{}{map[string]interface{}{
				"name":     obj.Name(),
				"label":    obj.Label(ctx),
				"helpText": obj.HelpText(ctx),
				"required": obj.Field().Required(),
				"fields":   fields,
			}}
		},
	}
}

func (m *ListBlockValue) Adapter() telepath.Adapter {
	return &telepath.ObjectAdapter[*ListBlockValue]{
		JSConstructor: "django.blocks.ListBlockValue",
		GetJSArgs: func(obj *ListBlockValue) []interface{} {
			return []interface{}{
				obj.ID,
				obj.Order,
				obj.Data,
			}
		},
	}
}

func (m *ListBlock) Adapter(ctx context.Context) telepath.Adapter {
	return &telepath.ObjectAdapter[*ListBlock]{
		JSConstructor: "django.blocks.ListBlock",
		GetJSArgs: func(obj *ListBlock) []interface{} {
			return []interface{}{map[string]interface{}{
				"name":     obj.Name(),
				"label":    obj.Label(ctx),
				"helpText": obj.HelpText(ctx),
				"required": obj.Field().Required(),
				"minNum":   obj.MinNum(),
				"maxNum":   obj.MaxNum(),
			}}
		},
	}
}
