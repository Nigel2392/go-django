package chooser

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"runtime/debug"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

var _ widgets.Widget = (*ChooserWidget)(nil)

type ChooserWidget struct {
	*widgets.BaseWidget
	TemplateKey string
	Templates   []string

	Definition  Chooser
	Model       attrs.Definer
	App         django.AppConfig
	ContentType *contenttypes.BaseContentType[attrs.Definer]
}

func NewChooserWidget(model attrs.Definer, widgetAttrs map[string]string) *ChooserWidget {

	if model == nil {
		panic("chooser widget requires a target model")
	}

	var app, ok = django.GetAppForModel(model)
	if !ok {
		panic(fmt.Sprintf(
			"chooser widget requires a target app, no app was found for %T",
			model,
		))
	}

	definition, ok := choosers.Get(reflect.TypeOf(model))
	if !ok {
		panic(fmt.Sprintf(
			"chooser widget requires a target chooser definition, no definition was found for %T",
			model,
		))
	}

	return &ChooserWidget{
		BaseWidget: widgets.NewBaseWidget(
			"file", "", widgetAttrs,
		),
		TemplateKey: "",
		Templates: []string{
			"chooser/widget.tmpl",
		},
		Model:      model,
		App:        app,
		Definition: definition,
		ContentType: contenttypes.NewContentType(
			model,
		),
	}
}

func (w *ChooserWidget) Media() media.Media {
	var m = media.NewMedia()
	m.AddCSS(media.CSS(django.Static("chooser/css/index.css")))
	m.AddJS(&media.JSAsset{
		URL: django.Static("chooser/js/index.js"),
	})
	return m
}

func (b *ChooserWidget) GetContextData(c context.Context, id, name string, value interface{}, attrs map[string]string) ctx.Context {
	var (
		ctx       = b.BaseWidget.GetContextData(c, id, name, value, attrs)
		appName   = b.App.Name()
		modelName = b.ContentType.Model()
	)

	var urlMap = map[string]string{
		"choose": django.Reverse("admin:apps:model:chooser:list", appName, modelName),
	}

	if b.Definition.CanCreate() {
		urlMap["create"] = django.Reverse("admin:apps:model:chooser:create", appName, modelName)
	}

	if b.Definition.CanUpdate() && !fields.IsZero(value) {
		urlMap["update"] = django.Reverse("admin:apps:model:chooser:update", appName, modelName, value)
	}

	ctx.Set("urls", urlMap)
	ctx.Set("title", b.Definition.GetTitle(c))
	return ctx
}

func (b *ChooserWidget) RenderWithErrors(ctx context.Context, w io.Writer, id, name string, value interface{}, errors []error, attrs map[string]string) error {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				logger.Error(fmt.Errorf("error rendering chooser widget: %w: %s", err, debug.Stack()))
				return
			}
			logger.Error(fmt.Errorf("error rendering chooser widget: %v: %s", r, debug.Stack()))
		}
	}()

	var context = b.GetContextData(ctx, id, name, value, attrs)
	if errors != nil {
		context.Set("errors", errors)
	}

	return tpl.FRender(w, context, b.TemplateKey, b.Templates...)
}

func (b *ChooserWidget) Render(ctx context.Context, w io.Writer, id, name string, value interface{}, attrs map[string]string) error {
	return b.RenderWithErrors(ctx, w, id, name, value, nil, attrs)
}
