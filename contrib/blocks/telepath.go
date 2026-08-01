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
		GetJSArgs: func(ctx context.Context, obj *FieldBlock) []interface{} {
			var field = obj.Field()
			if field == nil {
				panic("field is nil")
			}

			return []interface{}{
				obj.Name(),
				obj.FormField.Widget(),
				map[string]interface{}{
					"label":    obj.Label(ctx),
					"helpText": obj.HelpText(ctx),
					"required": field.Required(),
					"attrs":    field.Attrs(),
					"default":  field.ValueToForm(obj.GetDefault()),
				},
			}
		},
	}
}

func (m *StructBlock) Adapter(ctx context.Context) telepath.Adapter {
	return &telepath.ObjectAdapter[*StructBlock]{
		JSConstructor: "django.blocks.StructBlock",
		GetJSArgs: func(ctx context.Context, obj *StructBlock) []interface{} {

			var children = make([]map[string]interface{}, 0, obj.Fields.Len())
			for head := obj.Fields.Front(); head != nil; head = head.Next() {
				children = append(children, map[string]interface{}{
					"name":  head.Key,
					"block": head.Value,
				})
			}

			return []interface{}{
				obj.Name(),
				children,
				map[string]interface{}{
					"label":    obj.Label(ctx),
					"helpText": obj.HelpText(ctx),
					"required": obj.Field().Required(),
				},
			}
		},
	}
}

func (m *ListBlock) Adapter(ctx context.Context) telepath.Adapter {
	return &telepath.ObjectAdapter[*ListBlock]{
		JSConstructor: "django.blocks.ListBlock",
		GetJSArgs: func(ctx context.Context, obj *ListBlock) []interface{} {
			return []interface{}{
				obj.Name(),
				obj.Child,
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

func (m *StreamBlock) Adapter(ctx context.Context) telepath.Adapter {
	return &telepath.ObjectAdapter[*StreamBlock]{
		JSConstructor: "django.blocks.StreamBlock",
		GetJSArgs: func(ctx context.Context, obj *StreamBlock) []interface{} {

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
