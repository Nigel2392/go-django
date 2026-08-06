package components

import (
	"context"
	"fmt"
	"io"

	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/a-h/templ"
)

type ClassType uint8

const (
	ClassTypePrimary ClassType = 1 << iota
	ClassTypeSecondary
	ClassTypeSuccess
	ClassTypeInfo
	ClassTypeWarning
	ClassTypeDanger
	ClassTypeHollow
)

type ButtonConfig struct {
	Text   func(context.Context) string
	Icon   templ.Component
	Type   ClassType
	Attrs  map[string]any
	Hidden bool
}

func (b ButtonConfig) IsShown() bool {
	return !b.Hidden
}

func (b ButtonConfig) Render(ctx context.Context, w io.Writer) error {
	return Button(b).Render(ctx, w)
}

type LinkConfig struct {
	Text   func(context.Context) string
	Icon   templ.Component
	Type   ClassType
	Attrs  map[string]any
	URL    func(context.Context) string
	Hidden bool
}

func (b LinkConfig) IsShown() bool {
	return !b.Hidden
}

func (b LinkConfig) Render(ctx context.Context, w io.Writer) error {
	return renderLinkConfig(b).Render(ctx, w)
}

func NewButton(text any, args ...interface{}) templ.Component {

	var (
		iconComponent templ.Component
		type_         ClassType = 0
		attrs         map[string]any
		hidden        bool
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
		case ClassType:
			type_ |= t
		case int:
			type_ |= ClassType(t)
		case uint:
			type_ |= ClassType(t)
		case map[string]any:
			attrs = t
		case bool:
			hidden = t
		case nil, any:
			continue loop
		default:
			panic(fmt.Sprintf("Unknown type: %T\n", t))
		}
	}

	var cfg = ButtonConfig{
		Text:   trans.GetTextFunc(text),
		Icon:   iconComponent,
		Type:   type_,
		Attrs:  attrs,
		Hidden: hidden,
	}

	return Button(cfg)
}

func ButtonPrimary(text string, icon any, hollow ...bool) templ.Component {
	var h = false
	if len(hollow) > 0 && hollow[0] {
		h = true
	}
	var typ = ClassTypePrimary
	if h {
		typ |= ClassTypeHollow
	}
	return NewButton(text, icon, typ)
}

func ButtonSecondary(text string, icon any, hollow ...bool) templ.Component {
	var h = false
	if len(hollow) > 0 && hollow[0] {
		h = true
	}
	var typ = ClassTypeSecondary
	if h {
		typ |= ClassTypeHollow
	}
	return NewButton(text, icon, typ)
}

func ButtonSuccess(text string, icon any, hollow ...bool) templ.Component {
	var h = false
	if len(hollow) > 0 && hollow[0] {
		h = true
	}
	var typ = ClassTypeSuccess
	if h {
		typ |= ClassTypeHollow
	}
	return NewButton(text, icon, typ)
}

func ButtonDanger(text string, icon any, hollow ...bool) templ.Component {
	var h = false
	if len(hollow) > 0 && hollow[0] {
		h = true
	}
	var typ = ClassTypeDanger
	if h {
		typ |= ClassTypeHollow
	}
	return NewButton(text, icon, typ)
}

func ButtonWarning(text string, icon any, hollow ...bool) templ.Component {
	var h = false
	if len(hollow) > 0 && hollow[0] {
		h = true
	}
	var typ = ClassTypeWarning
	if h {
		typ |= ClassTypeHollow
	}
	return NewButton(text, icon, typ)
}
