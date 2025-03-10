// Code generated by templ - DO NOT EDIT.

// templ: version: v0.3.833
package components

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import templruntime "github.com/a-h/templ/runtime"

import (
	"fmt"
)

type ButtonType uint8

const (
	ButtonTypePrimary ButtonType = 1 << iota
	ButtonTypeSecondary
	ButtonTypeSuccess
	ButtonTypeWarning
	ButtonTypeDanger
	ButtonTypeHollow
)

type ButtonConfig struct {
	Text string
	Icon templ.Component
	Type ButtonType
}

func NewButton(text string, args ...interface{}) templ.Component {

	var (
		iconComponent templ.Component
		type_         ButtonType = 0
	)
loop:
	for _, arg := range args {
		switch t := arg.(type) {
		case templ.Component:
			iconComponent = t
		case string:
			if t == "" {
				continue loop
			}
			iconComponent = templ.Raw(t)
		case ButtonType:
			type_ |= t
		case int:
			type_ |= ButtonType(t)
		case uint:
			type_ |= ButtonType(t)
		case nil, any:
			continue loop
		default:
			panic(fmt.Sprintf("Unknown type: %T\n", t))
		}
	}

	var cfg = ButtonConfig{
		Text: text,
		Icon: iconComponent,
		Type: type_,
	}

	return Button(cfg)
}

func ButtonPrimary(text string, icon any, hollow ...bool) templ.Component {
	var h = false
	if len(hollow) > 0 && hollow[0] {
		h = true
	}
	var typ = ButtonTypePrimary
	if h {
		typ |= ButtonTypeHollow
	}
	return NewButton(text, icon, typ)
}

func ButtonSecondary(text string, icon any, hollow ...bool) templ.Component {
	var h = false
	if len(hollow) > 0 && hollow[0] {
		h = true
	}
	var typ = ButtonTypeSecondary
	if h {
		typ |= ButtonTypeHollow
	}
	return NewButton(text, icon, typ)
}

func ButtonSuccess(text string, icon any, hollow ...bool) templ.Component {
	var h = false
	if len(hollow) > 0 && hollow[0] {
		h = true
	}
	var typ = ButtonTypeSuccess
	if h {
		typ |= ButtonTypeHollow
	}
	return NewButton(text, icon, typ)
}

func ButtonDanger(text string, icon any, hollow ...bool) templ.Component {
	var h = false
	if len(hollow) > 0 && hollow[0] {
		h = true
	}
	var typ = ButtonTypeDanger
	if h {
		typ |= ButtonTypeHollow
	}
	return NewButton(text, icon, typ)
}

func ButtonWarning(text string, icon any, hollow ...bool) templ.Component {
	var h = false
	if len(hollow) > 0 && hollow[0] {
		h = true
	}
	var typ = ButtonTypeWarning
	if h {
		typ |= ButtonTypeHollow
	}
	return NewButton(text, icon, typ)
}

func Button(config ButtonConfig) templ.Component {
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
		var templ_7745c5c3_Var2 = []any{
			"button",
			templ.KV("primary", config.Type&ButtonTypePrimary != 0),
			templ.KV("secondary", config.Type&ButtonTypeSecondary != 0),
			templ.KV("success", config.Type&ButtonTypeSuccess != 0),
			templ.KV("danger", config.Type&ButtonTypeDanger != 0),
			templ.KV("warning", config.Type&ButtonTypeWarning != 0),
			templ.KV("hollow", config.Type&ButtonTypeHollow != 0),
		}
		templ_7745c5c3_Err = templ.RenderCSSItems(ctx, templ_7745c5c3_Buffer, templ_7745c5c3_Var2...)
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 1, "<button class=\"")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		var templ_7745c5c3_Var3 string
		templ_7745c5c3_Var3, templ_7745c5c3_Err = templ.JoinStringErrs(templ.CSSClasses(templ_7745c5c3_Var2).String())
		if templ_7745c5c3_Err != nil {
			return templ.Error{Err: templ_7745c5c3_Err, FileName: `src/contrib/admin/components/button.templ`, Line: 1, Col: 0}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var3))
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 2, "\">")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		if config.Icon != nil {
			templ_7745c5c3_Err = config.Icon.Render(ctx, templ_7745c5c3_Buffer)
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
		}
		var templ_7745c5c3_Var4 string
		templ_7745c5c3_Var4, templ_7745c5c3_Err = templ.JoinStringErrs(config.Text)
		if templ_7745c5c3_Err != nil {
			return templ.Error{Err: templ_7745c5c3_Err, FileName: `src/contrib/admin/components/button.templ`, Line: 138, Col: 21}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var4))
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 3, "</button>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return nil
	})
}

var _ = templruntime.GeneratedTemplate
