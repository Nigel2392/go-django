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

type hasChangedStub struct {
	changed bool
}

func (f *hasChangedStub) AddFormError(errorList ...error) {}

func (f *hasChangedStub) WithData(data url.Values, files map[string][]filesystem.FileHeader, r *http.Request) {
}

func (f *hasChangedStub) Data() (url.Values, map[string][]filesystem.FileHeader) {
	return nil, nil
}

func (f *hasChangedStub) SetPrefix(prefix string) {}

func (f *hasChangedStub) Prefix() string { return "" }

func (f *hasChangedStub) Field(name string) (fields.Field, bool) { return nil, false }

func (f *hasChangedStub) Widget(name string) (widgets.Widget, bool) { return nil, false }

func (f *hasChangedStub) ErrorList() []error { return nil }

func (f *hasChangedStub) BoundErrors() *orderedmap.OrderedMap[string, []error] {
	return orderedmap.NewOrderedMap[string, []error]()
}

func (f *hasChangedStub) WithContext(ctx context.Context) {}

func (f *hasChangedStub) CleanedData() map[string]any { return nil }

func (f *hasChangedStub) PrefixName(fieldName string) string { return fieldName }

func (f *hasChangedStub) HasChanged() bool { return f.changed }

func TestBaseFormSetHasChangedInitializesForms(t *testing.T) {
	fs := NewBaseFormSet[*hasChangedStub](context.Background(), FormsetOptions[*hasChangedStub]{
		MinNum:    1,
		MaxNum:    1,
		CanAdd:    false,
		CanDelete: false,
		NewForm: func(ctx context.Context) *hasChangedStub {
			return &hasChangedStub{}
		},
		DefaultForms: func(ctx context.Context, max, min int) ([]*hasChangedStub, error) {
			return []*hasChangedStub{{changed: true}}, nil
		},
	})

	if !fs.HasChanged() {
		t.Fatalf("expected formset HasChanged to initialize forms and report changes")
	}
}
