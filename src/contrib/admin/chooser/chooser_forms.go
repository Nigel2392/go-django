package chooser

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"reflect"

	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/Nigel2392/go-django/src/permissions"
	hut "github.com/Nigel2392/go-django/src/utils/httputils"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/goldcrest"

	_ "unsafe"
)

var (
	_ views.View         = (*ChooserFormPage[attrs.Definer])(nil)
	_ views.MethodsView  = (*ChooserFormPage[attrs.Definer])(nil)
	_ views.BindableView = (*ChooserFormPage[attrs.Definer])(nil)
	_ views.Renderer     = (*BoundChooserFormPage[attrs.Definer])(nil)
)

type ChooserFormPage[T attrs.Definer] struct {
	Template       string
	AllowedMethods []string
	Options        admin.FormViewOptions

	_Definition *ChooserDefinition[T]
}

func (v *ChooserFormPage[T]) ServeXXX(w http.ResponseWriter, req *http.Request) {
	// Placeholder method, will never get called.
}

func (v *ChooserFormPage[T]) Methods() []string {
	return v.AllowedMethods
}

func (v *ChooserFormPage[T]) GetTemplate(req *http.Request) string {
	if v.Template != "" {
		return v.Template
	}
	return "chooser/views/create.tmpl"
}

//go:linkname newInstanceView github.com/Nigel2392/go-django/src/contrib/admin.newInstanceView
func newInstanceView(tpl string, instance attrs.Definer, opts admin.FormViewOptions, app *admin.AppDefinition, model *admin.ModelDefinition, r *http.Request) *views.FormView[*admin.AdminModelForm[modelforms.ModelForm[attrs.Definer], attrs.Definer]]

func (v *ChooserFormPage[T]) Bind(w http.ResponseWriter, req *http.Request) (views.View, error) {
	var modelObj = attrs.NewObject[T](
		reflect.TypeOf(v._Definition.Model),
	)
	var base = &BoundChooserFormPage[T]{
		FormView: newInstanceView(
			"add",
			modelObj,
			v.Options,
			v._Definition.AdminApp,
			v._Definition.AdminModel,
			req,
		),
		View:           v,
		ResponseWriter: w,
		Request:        req,
		Model:          modelObj,
	}
	base.FormView.TemplateName = v.GetTemplate(req)
	base.FormView.BaseTemplateKey = ""
	return base, nil
}

func (v *ChooserFormPage[T]) GetContext(req *http.Request, bound *BoundChooserFormPage[T]) *ModalContext {
	var c = v._Definition.GetContext(req, v, bound)
	return c
}

type BoundChooserFormPage[T attrs.Definer] struct {
	*views.FormView[*admin.AdminModelForm[modelforms.ModelForm[attrs.Definer], attrs.Definer]]
	View           *ChooserFormPage[T]
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	Model          T

	isValid bool
}

func (v *BoundChooserFormPage[T]) ServeXXX(w http.ResponseWriter, req *http.Request) {
	// Placeholder method, will never get called.
}

func (v *BoundChooserFormPage[T]) Setup(w http.ResponseWriter, req *http.Request) (http.ResponseWriter, *http.Request) {
	if !permissions.HasObjectPermission(req, v.Model, "admin:add") {
		except.Fail(
			http.StatusForbidden,
			"User does not have permission to add this object",
		)
		return nil, nil
	}

	return w, req
}

func (v *BoundChooserFormPage[T]) Render(w http.ResponseWriter, req *http.Request, context ctx.Context) error {
	var writer = hut.NewFakeWriter(new(bytes.Buffer))

	v.FormView.SuccessFn = func(w http.ResponseWriter, req *http.Request, form *admin.AdminModelForm[modelforms.ModelForm[attrs.Definer], attrs.Definer]) {

		// Set isValidFlag to true
		// This function will run inside of v.FormView.Render if the form is
		// submitted successfully
		//
		// This is to ensure we don't encode a JSON response into [ChooserResponse.HTML]
		v.isValid = true

		var instance = form.Instance()
		for _, hook := range goldcrest.Get[admin.AdminModelHookFunc]("admin:model:add") {
			hook(req, admin.AdminSite, v.View._Definition.AdminModel, instance)
		}

		var pk = attrs.PrimaryKey(instance)
		var err = json.NewEncoder(w).Encode(ChooserResponse{
			Preview: v.View._Definition.GetPreviewString(
				req.Context(), instance,
			),
			PK: pk,
		})

		if err == nil {
			return
		}

		except.Fail(
			http.StatusInternalServerError,
			err,
		)
	}

	if err := v.FormView.Render(writer, req, v.GetTemplate(req), context); err != nil {
		return err
	}

	writer.CopyTo(
		w, hut.FlagCopyHeader|hut.FlagCopyStatus,
	)

	// If the form is valid, the response is already JSON
	// Copy the buffer and be done with it.
	if v.isValid {
		_, err := io.Copy(w, writer.WriteTo)
		return err
	}

	var response = ChooserResponse{
		HTML: writer.WriteTo.String(),
	}

	return json.NewEncoder(w).Encode(response)
}
