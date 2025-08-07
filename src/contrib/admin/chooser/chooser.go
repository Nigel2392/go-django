package chooser

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/views"
)

type Chooser interface {
	Setup() error
	GetTitle(ctx context.Context) string
	GetModel() attrs.Definer

	CanCreate() bool
	CanUpdate() bool

	ListView(adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition) views.View
	CreateView(adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition) views.View
	UpdateView(adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition, instance attrs.Definer) views.View
}

type ChooserDefinition[T attrs.Definer] struct {
	Model  T
	Title  any // string or func(ctx context.Context) string
	Labels map[string]func(ctx context.Context) string

	ListPage   *ChooserListPage[T]
	CreatePage *ChooserFormPage[T]
	UpdatePage *ChooserFormPage[T]
}

func (c *ChooserDefinition[T]) Setup() error {
	if c.Title == nil {
		return errors.ValueError.Wrap("ChooserDefinition.Title cannot be nil")
	}

	if reflect.ValueOf(c.Model).IsNil() {
		return errors.TypeMismatch.Wrap("ChooserDefinition.Model cannot be nil")
	}

	if c.ListPage != nil {
		c.ListPage._Definition = c
	}

	if c.CreatePage != nil {
		c.CreatePage._Definition = c
	}

	if c.UpdatePage != nil {
		c.UpdatePage._Definition = c
	}

	return nil
}

func (c *ChooserDefinition[T]) GetTitle(ctx context.Context) string {
	switch v := c.Title.(type) {
	case string:
		return v
	case func(ctx context.Context) string:
		return v(ctx)
	}
	assert.Fail("ChooserDefinition.Title must be a string or a function that returns a string")
	return ""
}

func (o *ChooserDefinition[T]) GetLabel(labels map[string]func(context.Context) string, field string, default_ string) func(ctx context.Context) string {
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
	return func(ctx context.Context) string {
		return default_
	}
}

func (c *ChooserDefinition[T]) GetModel() attrs.Definer {
	return c.Model
}

func (c *ChooserDefinition[T]) CanCreate() bool {
	return c.CreatePage != nil
}

func (c *ChooserDefinition[T]) CanUpdate() bool {
	return c.UpdatePage != nil
}

func (c *ChooserDefinition[T]) ListView(adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition) views.View {
	if c.ListPage != nil {
		c.ListPage._Definition = c
	}
	return c.ListPage
}

func (c *ChooserDefinition[T]) CreateView(adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition) views.View {
	if c.CreatePage != nil {
		c.CreatePage._Definition = c
	}
	return c.CreatePage
}

func (c *ChooserDefinition[T]) UpdateView(adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition, instance attrs.Definer) views.View {
	if c.UpdatePage != nil {
		c.UpdatePage._Definition = c
	}
	return c.UpdatePage
}

func (c *ChooserDefinition[T]) GetContext(req *http.Request, page, bound views.View) *ModalContext {
	var ctx = ctx.RequestContext(req)
	ctx.Set("chooser", c)
	ctx.Set("chooser_page", page)
	ctx.Set("chooser_view", bound)

	return &ModalContext{
		ContextWithRequest: ctx,
		Definition:         c,
		View:               bound,
	}
}

type ChooserResponse struct {
	HTML        string `json:"html"`
	PreviewHTML string `json:"preview_html,omitempty"`
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
