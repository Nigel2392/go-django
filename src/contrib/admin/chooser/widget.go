package chooser

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"runtime/debug"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/internal/django_reflect"
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
			"text", "", widgetAttrs,
		),
		TemplateKey: "",
		Templates: []string{
			"chooser/widget/widget.tmpl",
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

func (w *ChooserWidget) ValueToForm(value interface{}) interface{} {
	return value
}

func (w *ChooserWidget) ValueToGo(value interface{}) (interface{}, error) {
	if _, ok := value.(attrs.Definer); ok {
		return value, nil
	}

	var newObj = attrs.NewObject[attrs.Definer](w.Model)
	var defs = newObj.FieldDefs()
	var prim = defs.Primary()
	if err := prim.Scan(value); err != nil {
		return nil, errors.Wrapf(
			err, "failed to scan value into primary key field %q",
			prim.Name(),
		)
	}

	return newObj, nil
}

func (b *ChooserWidget) GetContextData(c context.Context, id, name string, value interface{}, widgetAttrs map[string]string) ctx.Context {
	var (
		ctx       = b.BaseWidget.GetContextData(c, id, name, value, widgetAttrs)
		appName   = b.App.Name()
		modelName = b.ContentType.Model()
	)

	var urlMap = map[string]string{
		"choose": django.Reverse("admin:apps:model:chooser:list", appName, modelName),
	}

	if b.Definition.CanCreate() {
		urlMap["create"] = django.Reverse("admin:apps:model:chooser:create", appName, modelName)
	}

	if b.Definition.CanUpdate() && !django_reflect.IsZero(value) {
		urlMap["update"] = django.Reverse("admin:apps:model:chooser:update", appName, modelName, value)
	}

	ctx.Set("urls", urlMap)
	ctx.Set("title", b.Definition.GetTitle(c))

	if django_reflect.IsZero(value) {
		ctx.Set("value", nil)
		ctx.Set("preview", nil)
	} else {

		var meta = attrs.GetModelMeta(b.Model)
		var defs = meta.Definitions()
		var primDef = defs.Primary()

		var modelRow, err = queries.GetQuerySet(b.Model).
			Filter(primDef.Name(), value).
			Get()

		if err != nil {
			except.Fail(
				http.StatusInternalServerError,
				err,
			)
			return ctx
		}

		ctx.Set("value", attrs.PrimaryKey(modelRow.Object))
		ctx.Set("preview", b.Definition.GetPreviewString(c, modelRow.Object))
	}

	return ctx
}

func (b *ChooserWidget) RenderWithErrors(ctx context.Context, w io.Writer, id, name string, value interface{}, errors []error, attrs map[string]string, context ctx.Context) error {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				logger.Error(fmt.Errorf("error rendering chooser widget: %w: %s", err, debug.Stack()))
				return
			}
			logger.Error(fmt.Errorf("error rendering chooser widget: %v: %s", r, debug.Stack()))
		}
	}()

	return tpl.FRender(w, context, b.TemplateKey, b.Templates...)
}

func (b *ChooserWidget) Render(ctx context.Context, w io.Writer, id, name string, value interface{}, attrs map[string]string) error {
	return b.RenderWithErrors(ctx, w, id, name, value, nil, attrs, b.GetContextData(ctx, id, name, value, attrs))
}
