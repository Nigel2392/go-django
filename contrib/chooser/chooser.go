package chooser

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/utils/djangoapps"
	"github.com/Nigel2392/go-django/src/views"
)

type chooser interface {
	Setup(chooserKey string) error

	GetTitle(ctx context.Context) string
	GetPreviewString(ctx context.Context, instance attrs.Definer) string
	GetExtraData(ctx context.Context, instance attrs.Definer) map[string]any

	GetModel() attrs.Definer

	ListView() views.View
	Media() media.Media
}

type ChooserDefinition[T attrs.Definer] struct {
	ChooserKey    string
	Model         T
	Title         any // string or func(ctx context.Context) string
	Labels        map[string]func(ctx context.Context) string
	PreviewString func(ctx context.Context, instance T) string
	ExtraData     func(ctx context.Context, instance T) map[string]any

	ListPage    *ChooserListPage[T]
	DjangoApp   djangoapps.BaseAppConfig
	ContentType contenttypes.ContentType
	MediaFn     func() media.Media
}

type WrappedModel[T attrs.Definer] struct {
	Model      T
	Definition *ChooserDefinition[T]
	Context    context.Context
}

func WrapModels[T attrs.Definer](ctx context.Context, def *ChooserDefinition[T], models []T) []*WrappedModel[T] {
	var wrappedModels []*WrappedModel[T]
	for _, model := range models {
		var wrappedModel = WrapModel(ctx, def, model)
		if wrappedModel == nil {
			continue
		}

		wrappedModels = append(
			wrappedModels,
			wrappedModel,
		)
	}
	return wrappedModels
}

func WrapModel[T attrs.Definer](ctx context.Context, def *ChooserDefinition[T], model T) *WrappedModel[T] {
	if reflect.ValueOf(model).IsNil() {
		return nil
	}

	return &WrappedModel[T]{
		Model:      model,
		Definition: def,
		Context:    ctx,
	}
}

func (w *WrappedModel[T]) PreviewHTML() string {
	return w.Definition.GetPreviewString(w.Context, w.Model)
}

func (w *WrappedModel[T]) ExtraData() map[string]any {
	return w.Definition.GetExtraData(w.Context, w.Model)
}

func (c *ChooserDefinition[T]) Setup(chooserKey string) error {
	if c.Title == nil {
		return errors.ValueError.Wrap("ChooserDefinition.Title cannot be nil")
	}

	if reflect.ValueOf(c.Model).IsNil() {
		return errors.TypeMismatch.Wrap("ChooserDefinition.Model cannot be nil")
	}

	var djangoApp, ok = djangoapps.GetAppForModel(c.Model)
	if !ok {
		return errors.TypeMismatch.Wrapf(
			"ChooserDefinition.Model is not a valid Django model, no app found for %T",
			c.Model,
		)
	}

	c.DjangoApp = djangoApp
	c.ChooserKey = chooserKey
	c.ContentType = contenttypes.NewContentType[any](c.Model)

	if c.ContentType == nil {
		return errors.ValueError.Wrapf(
			"ChooserDefinition.Model is not a valid Django model, no app found for %T",
			c.Model,
		)
	}

	c.setupListPage()

	return nil
}

func (c *ChooserDefinition[T]) setupListPage() {
	if c.ListPage == nil {
		c.ListPage = &ChooserListPage[T]{}
	}

	if len(c.ListPage.AllowedMethods) == 0 {
		c.ListPage.AllowedMethods = []string{"GET"}
	}

	if c.ListPage.PerPage == 0 {
		c.ListPage.PerPage = 20
	}

	if len(c.ListPage.Fields) == 0 {
		var meta = attrs.GetModelMeta(c.Model)
		var flds = meta.Definitions().Fields()
		c.ListPage.Fields = make([]string, 0, len(flds))
		for _, fld := range flds {
			c.ListPage.Fields = append(c.ListPage.Fields, fld.Name())
		}
	}

	if len(c.ListPage.Labels) == 0 {
		var meta = attrs.GetModelMeta(c.Model)
		var flds = meta.Definitions().Fields()
		c.ListPage.Labels = make(map[string]func(ctx context.Context) string, len(flds))
		for _, fld := range flds {
			c.ListPage.Labels[fld.Name()] = fld.Label
		}
	}

	c.ListPage._Definition = c
}

func (c *ChooserDefinition[T]) GetTitle(ctx context.Context) string {
	var title, ok = trans.GetText(ctx, c.Title)
	if ok {
		return title
	}
	assert.Fail("ChooserDefinition.Title must be a string or a function that returns a string")
	return ""
}

func (o *ChooserDefinition[T]) GetLabel(labels map[string]func(context.Context) string, field string, default_ any) func(ctx context.Context) string {
	if labels != nil {
		var label, ok = labels[field]
		if ok {
			return label
		}
	}
	if o.Labels != nil {
		var label, ok = o.Labels[field]
		if ok {
			return label
		}
	}
	if fn := trans.GetTextFunc(default_); fn != nil {
		return fn
	}
	assert.Fail("ChooserDefinition.GetLabel: default_ must be a string or a function that returns a string")
	return nil
}

func (c *ChooserDefinition[T]) GetPreviewString(ctx context.Context, instance attrs.Definer) (previewString string) {
	if c.PreviewString != nil {
		previewString = c.PreviewString(ctx, instance.(T))
	}

	if previewString == "" {
		previewString = attrs.ToString(instance)
	}

	if previewString == "" {
		previewString = fmt.Sprintf(
			"%T(%v)",
			instance, attrs.PrimaryKey(ctx, instance),
		)
	}

	return previewString
}

func (c *ChooserDefinition[T]) GetExtraData(ctx context.Context, instance attrs.Definer) map[string]any {
	if c.ExtraData != nil {
		return c.ExtraData(ctx, instance.(T))
	}
	return map[string]any{}
}

func (c *ChooserDefinition[T]) GetModel() attrs.Definer {
	return attrs.NewObject[attrs.Definer](context.Background(), c.Model)
}

func (c *ChooserDefinition[T]) ListView() views.View {
	if c.ListPage != nil {
		c.ListPage._Definition = c
	}
	return c.ListPage
}

func (c *ChooserDefinition[T]) Media() media.Media {
	if c.MediaFn != nil {
		return c.MediaFn()
	}
	return media.NewMedia()
}

func (c *ChooserDefinition[T]) GetContext(req *http.Request, page, bound views.View) *ModalContext {
	var modelName = c.ContentType.ShortTypeName()
	var ctx = ctx.RequestContext(req)
	ctx.Set("chooser", c)
	ctx.Set("chooser_page", page)
	ctx.Set("chooser_view", bound)
	ctx.Set("chooser_key", c.ChooserKey)
	ctx.Set("model_name", modelName)

	var urlMap = map[string]string{
		"choose": django.Reverse("chooser:list", modelName, c.ChooserKey),
	}

	ctx.Set("urls", urlMap)

	return &ModalContext{
		ContextWithRequest: ctx,
		Definition:         c,
		View:               bound,
	}
}

type ChooserResponse struct {
	HTML      string         `json:"html"`
	Preview   string         `json:"preview,omitempty"`
	ExtraData map[string]any `json:"data,omitempty"`
	PK        any            `json:"pk,omitempty"`
}

func (c *ChooserDefinition[T]) Render(w http.ResponseWriter, req *http.Request, context ctx.Context, base, template string) error {
	var buf = new(bytes.Buffer)
	if err := tpl.FRender(buf, context, base, template); err != nil {
		return err
	}

	var response = ChooserResponse{
		HTML: buf.String(),
	}

	return json.NewEncoder(w).Encode(response)
}
