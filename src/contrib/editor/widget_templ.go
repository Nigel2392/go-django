// Code generated by templ - DO NOT EDIT.

// templ: version: v0.3.819
package editor

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import templruntime "github.com/a-h/templ/runtime"

import (
	"encoding/json"
	"fmt"

	"context"
	"io"

	"github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

func getSafe[T any](m map[string]interface{}, key string) (value T) {
	if v, ok := m[key]; ok {
		return v.(T)
	}
	return value
}

var _ widgets.Widget = (*EditorJSWidget)(nil)

type EditorJSWidget struct {
	widgets.BaseWidget
	Features []string
}

func NewEditorJSWidget(features ...string) *EditorJSWidget {
	if len(features) == 0 {
		features = EditorRegistry.features.Keys()
	}

	return &EditorJSWidget{
		BaseWidget: widgets.BaseWidget{
			Type: "editorjs",
		},
		Features: features,
	}
}

func (b *EditorJSWidget) ValueToForm(value interface{}) interface{} {
	var editorJsData EditorJSData
	if fields.IsZero(value) {
		return editorJsData
	}

	var featuresMap = make(map[string]struct{})
	for _, feature := range b.Features {
		featuresMap[feature] = struct{}{}
	}

	switch value := value.(type) {
	case *EditorJSBlockData:
		var blocks = make([]BlockData, 0)

		for _, block := range value.Blocks {
			if _, ok := featuresMap[block.Type()]; !ok {
				continue
			}
			var data = block.Data()
			blocks = append(blocks, data)
		}

		var d = EditorJSData{
			Time:    value.Time,
			Version: value.Version,
			Blocks:  blocks,
		}

		editorJsData = d
	case EditorJSData:
		editorJsData = value
	default:
		panic(fmt.Sprintf("Invalid value type: %T", value))
	}

	return editorJsData
}

func (b *EditorJSWidget) ValueToGo(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	var editorJsData EditorJSData
	switch value := value.(type) {
	case *EditorJSBlockData:
		return value, nil
	case EditorJSData:
		editorJsData = value
	case string:
		var d EditorJSData
		if err := json.Unmarshal([]byte(value), &d); err != nil {
			return nil, err
		}
		editorJsData = d
	default:
		panic(fmt.Sprintf("Invalid value type: %T (%v)", value, value))
	}

	return ValueToGo(b.Features, editorJsData)
}

func (b *EditorJSWidget) GetContextData(id, name string, value interface{}, attrs map[string]string) ctx.Context {
	var context = b.BaseWidget.GetContextData(id, name, value, attrs)
	// ...
	return context
}

func (b *EditorJSWidget) Component(id, name, value string, errors []error, attrs map[string]string, config map[string]interface{}) templ.Component {
	return templruntime.GeneratedTemplate(func(templ_7745c5c3_Input templruntime.GeneratedComponentInput) (templ_7745c5c3_Err error) {
		templ_7745c5c3_W, ctx := templ_7745c5c3_Input.Writer, templ_7745c5c3_Input.Context
		if templ_7745c5c3_CtxErr := ctx.Err(); templ_7745c5c3_CtxErr != nil {
			return templ_7745c5c3_CtxErr
		}
		templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templruntime.GetBuffer(templ_7745c5c3_W)
		if !templ_7745c5c3_IsBuffer {
			defer func() {
				templ_7745c5c3_BufErr := templruntime.ReleaseBuffer(templ_7745c5c3_Buffer)
				if templ_7745c5c3_Err == nil {
					templ_7745c5c3_Err = templ_7745c5c3_BufErr
				}
			}()
		}
		ctx = templ.InitializeContext(ctx)
		templ_7745c5c3_Var1 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var1 == nil {
			templ_7745c5c3_Var1 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 1, "<div id=\"")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		var templ_7745c5c3_Var2 string
		templ_7745c5c3_Var2, templ_7745c5c3_Err = templ.JoinStringErrs(fmt.Sprintf("%s-container", id))
		if templ_7745c5c3_Err != nil {
			return templ.Error{Err: templ_7745c5c3_Err, FileName: `src/contrib/editor/widget.templ`, Line: 114, Col: 45}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var2))
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 2, "\" class=\"editorjs-widget-container\" data-controller=\"editorjs-widget\" data-editorjs-widget-config-value=\"")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		var templ_7745c5c3_Var3 string
		templ_7745c5c3_Var3, templ_7745c5c3_Err = templ.JoinStringErrs(templ.JSONString(config))
		if templ_7745c5c3_Err != nil {
			return templ.Error{Err: templ_7745c5c3_Err, FileName: `src/contrib/editor/widget.templ`, Line: 114, Col: 176}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var3))
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 3, "\"><input type=\"hidden\" id=\"")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		var templ_7745c5c3_Var4 string
		templ_7745c5c3_Var4, templ_7745c5c3_Err = templ.JoinStringErrs(id)
		if templ_7745c5c3_Err != nil {
			return templ.Error{Err: templ_7745c5c3_Err, FileName: `src/contrib/editor/widget.templ`, Line: 117, Col: 19}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var4))
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 4, "\" name=\"")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		var templ_7745c5c3_Var5 string
		templ_7745c5c3_Var5, templ_7745c5c3_Err = templ.JoinStringErrs(name)
		if templ_7745c5c3_Err != nil {
			return templ.Error{Err: templ_7745c5c3_Err, FileName: `src/contrib/editor/widget.templ`, Line: 118, Col: 23}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var5))
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 5, "\" value=\"")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		var templ_7745c5c3_Var6 string
		templ_7745c5c3_Var6, templ_7745c5c3_Err = templ.JoinStringErrs(value)
		if templ_7745c5c3_Err != nil {
			return templ.Error{Err: templ_7745c5c3_Err, FileName: `src/contrib/editor/widget.templ`, Line: 119, Col: 25}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var6))
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 6, "\" data-editorjs-widget-target=\"input\"><div data-editorjs-widget-target=\"editor\" id=\"")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		var templ_7745c5c3_Var7 string
		templ_7745c5c3_Var7, templ_7745c5c3_Err = templ.JoinStringErrs(fmt.Sprintf("%s-editor", id))
		if templ_7745c5c3_Err != nil {
			return templ.Error{Err: templ_7745c5c3_Err, FileName: `src/contrib/editor/widget.templ`, Line: 122, Col: 83}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var7))
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 7, "\"></div></div>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return nil
	})
}

func (b *EditorJSWidget) RenderWithErrors(w io.Writer, id, name string, value interface{}, errors []error, attrs map[string]string) error {
	var valueStr string
	if value == nil || value == "" {
		value = "{}"
		goto getContext
	}

	switch value := value.(type) {
	case []byte:
		valueStr = string(value)
	case string:
		valueStr = value
	default:
		var d, err = json.Marshal(value)
		if err != nil {
			return err
		}
		valueStr = string(d)
	}

getContext:
	var widgetCtx = b.GetContextData(id, name, value, attrs)
	if errors != nil {
		widgetCtx.Set("errors", errors)
	}

	var (
		config    = buildConfig(widgetCtx, b.Features...)
		renderCtx = context.Background()
	)

	config["holder"] = fmt.Sprintf("%s-editor", id)
	var component = b.Component(
		id, name, valueStr, errors, attrs, config,
	)

	return component.Render(renderCtx, w)
}

func (b *EditorJSWidget) Render(w io.Writer, id, name string, value interface{}, attrs map[string]string) error {
	return b.RenderWithErrors(w, id, name, value, nil, attrs)
}

func (b *EditorJSWidget) Media() media.Media {
	var (
		m           = b.BaseWidget.Media()
		featureList = Features(b.Features...)
	)

	for _, feature := range featureList {
		m = m.Merge(feature.Media())
	}

	m.AddCSS(
		media.CSS(django.Static("editorjs/css/index.css")),
	)

	m.AddJS(
		&media.JSAsset{URL: django.Static("editorjs/js/index.js")},
	)

	return m
}

var _ = templruntime.GeneratedTemplate
