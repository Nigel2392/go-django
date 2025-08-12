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
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-django/src/views/list"
	"github.com/Nigel2392/mux"
)

type Chooser interface {
	Setup(chooserKey string) error
	GetTitle(ctx context.Context) string
	GetPreviewString(ctx context.Context, instance attrs.Definer) string
	GetModel() attrs.Definer

	CanCreate() bool

	ListView(adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition) views.View
	CreateView(adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition) views.View
}

type ChooserDefinition[T attrs.Definer] struct {
	ChooserKey    string
	Model         T
	Title         any // string or func(ctx context.Context) string
	Labels        map[string]func(ctx context.Context) string
	PreviewString func(ctx context.Context, instance T) string

	ListPage   *ChooserListPage[T]
	CreatePage *ChooserFormPage[T]

	DjangoApp  django.AppConfig
	AdminApp   *admin.AppDefinition
	AdminModel *admin.ModelDefinition
}

func (c *ChooserDefinition[T]) Setup(chooserKey string) error {
	if c.Title == nil {
		return errors.ValueError.Wrap("ChooserDefinition.Title cannot be nil")
	}

	if reflect.ValueOf(c.Model).IsNil() {
		return errors.TypeMismatch.Wrap("ChooserDefinition.Model cannot be nil")
	}

	var djangoApp, ok = django.GetAppForModel(c.Model)
	if !ok {
		return errors.TypeMismatch.Wrapf(
			"ChooserDefinition.Model is not a valid Django model, no app found for %T",
			c.Model,
		)
	}

	adminApp, ok := admin.AdminSite.Apps.Get(djangoApp.Name())
	if !ok {
		return errors.TypeMismatch.Wrapf(
			"ChooserDefinition.Model is not a valid Django model, no admin app found for %s",
			djangoApp.Name(),
		)
	}

	var cType = contenttypes.NewContentType(c.Model)
	adminModel, ok := adminApp.Models.Get(cType.Model())
	if !ok {
		return errors.TypeMismatch.Wrapf(
			"ChooserDefinition.Model is not a valid Django model, no admin model found for %s",
			cType.Model(),
		)
	}

	c.DjangoApp = djangoApp
	c.AdminApp = adminApp
	c.AdminModel = adminModel
	c.ChooserKey = chooserKey

	c.setupListPage()
	c.setupCreatePage()

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
		c.ListPage.Fields = c.AdminModel.ListView.Fields
	}

	if len(c.ListPage.Labels) == 0 {
		c.ListPage.Labels = c.AdminModel.ListView.Labels
	}

	if len(c.ListPage.Format) == 0 {
		c.ListPage.Format = c.AdminModel.ListView.Format
	}

	if len(c.ListPage.Columns) == 0 {
		var newCols = make(map[string]list.ListColumn[T], len(c.AdminModel.ListView.Columns))
		for name, col := range c.AdminModel.ListView.Columns {
			newCols[name] = list.ChangeColumnType[T](col)
		}
		c.ListPage.Columns = newCols
	}

	c.ListPage._Definition = c
}

func (c *ChooserDefinition[T]) setupCreatePage() {
	if c.CreatePage == nil {
		return
	}

	c.CreatePage._Definition = c

	if len(c.CreatePage.AllowedMethods) == 0 {
		c.CreatePage.AllowedMethods = []string{"POST"}
	}

	if len(c.CreatePage.Options.Panels) == 0 &&
		len(c.CreatePage.Options.Fields) == 0 &&
		len(c.CreatePage.Options.Exclude) == 0 &&
		len(c.CreatePage.Options.Widgets) == 0 &&
		c.CreatePage.Options.FormInit != nil &&
		c.CreatePage.Options.GetForm != nil &&
		c.CreatePage.Options.GetHandler != nil {
		c.CreatePage.Options = c.AdminModel.AddView
	}

	c.CreatePage.Options.SetupDefaults(
		c.Model, "GetAddPanels",
	)
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
	switch v := default_.(type) {
	case string:
		return func(ctx context.Context) string {
			return v
		}
	case func(context.Context) string:
		return v
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
			instance, attrs.PrimaryKey(instance),
		)
	}

	return previewString
}

func (c *ChooserDefinition[T]) GetModel() attrs.Definer {
	return attrs.NewObject[attrs.Definer](c.Model)
}

func (c *ChooserDefinition[T]) CanCreate() bool {
	return c.CreatePage != nil
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

func (c *ChooserDefinition[T]) GetContext(req *http.Request, page, bound views.View) *ModalContext {
	var ctx = ctx.RequestContext(req)
	ctx.Set("chooser", c)
	ctx.Set("chooser_page", page)
	ctx.Set("chooser_view", bound)

	var urlVars = mux.Vars(req)
	var appName = urlVars.Get("app_name")
	var modelName = urlVars.Get("model_name")

	var urlMap = map[string]string{
		"choose": django.Reverse("admin:apps:model:chooser:list", appName, modelName, c.ChooserKey),
	}

	if c.CanCreate() {
		urlMap["create"] = django.Reverse("admin:apps:model:chooser:create", appName, modelName, c.ChooserKey)
	}

	ctx.Set("urls", urlMap)

	return &ModalContext{
		ContextWithRequest: ctx,
		Definition:         c,
		View:               bound,
	}
}

type ChooserResponse struct {
	HTML    string `json:"html"`
	Preview string `json:"preview,omitempty"`
	PK      any    `json:"pk,omitempty"`
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
