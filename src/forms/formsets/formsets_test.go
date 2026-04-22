package formsets

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/elliotchance/orderedmap/v2"
)

type hasChangedStubForm struct {
	changed bool
}

func (f *hasChangedStubForm) AddFormError(errorList ...error) {}

func (f *hasChangedStubForm) WithData(data url.Values, files map[string][]filesystem.FileHeader, r *http.Request) {
}

func (f *hasChangedStubForm) Data() (url.Values, map[string][]filesystem.FileHeader) {
	return nil, nil
}

func (f *hasChangedStubForm) SetPrefix(prefix string) {}

func (f *hasChangedStubForm) Prefix() string { return "" }

func (f *hasChangedStubForm) Field(name string) (fields.Field, bool) { return nil, false }

func (f *hasChangedStubForm) Widget(name string) (widgets.Widget, bool) { return nil, false }

func (f *hasChangedStubForm) ErrorList() []error { return nil }

func (f *hasChangedStubForm) BoundErrors() *orderedmap.OrderedMap[string, []error] {
	return orderedmap.NewOrderedMap[string, []error]()
}

func (f *hasChangedStubForm) WithContext(ctx context.Context) {}

func (f *hasChangedStubForm) CleanedData() map[string]any { return nil }

func (f *hasChangedStubForm) PrefixName(fieldName string) string { return fieldName }

func (f *hasChangedStubForm) HasChanged() bool { return f.changed }

func TestBaseFormSetHasChangedInitializesForms(t *testing.T) {
	fs := NewBaseFormSet[*hasChangedStubForm](context.Background(), FormsetOptions[*hasChangedStubForm]{
		MinNum:    1,
		MaxNum:    1,
		CanAdd:    false,
		CanDelete: false,
		NewForm: func(ctx context.Context) *hasChangedStubForm {
			return &hasChangedStubForm{}
		},
		DefaultForms: func(ctx context.Context, max, min int) ([]*hasChangedStubForm, error) {
			return []*hasChangedStubForm{{changed: true}}, nil
		},
	})

	if !fs.HasChanged() {
		t.Fatalf("expected formset HasChanged to initialize forms and report changes")
	}
}
