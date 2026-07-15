package objects

import (
	"errors"
	"fmt"
	"maps"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime/debug"
	"slices"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/djester"
	"github.com/Nigel2392/go-django/internal/django_reflect"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/PuerkitoBio/goquery"
)

const (
	_FORMFIELD_REQUEST_URL = "/djester/formfield-test/"
)

var (
	_ djester.Test = (*FieldTest[forms.Field])(nil)
)

type FieldTest[FORMFIELD forms.Field] struct {
	Label             string
	FieldNameOverride string // if this isn't set, the field WILL be provided a default name.
	ExpectsValid      bool
	ExpectedHTMLValue string
	ExpectedErrors    []error
	RequestForForm    func(d *djester.Tester, t *testing.T, ft *FieldTest[FORMFIELD], form forms.Form, fld FORMFIELD, urlValues url.Values, files map[string][]filesystem.FileHeader) *http.Request
	FormData          any // string | []string | []filesystem.FileHeader | func(d *djester.Tester, t *testing.T, ft *FieldTest[FORMFIELD], f FORMFIELD) (urlValues []string, files []filesystem.FileHeader)
	FormWithData      func(d *djester.Tester, t *testing.T, ft *FieldTest[FORMFIELD], f FORMFIELD, fieldName string, urlValues url.Values, files map[string][]filesystem.FileHeader, form forms.Form) forms.Form
	GetFormOverride   func(d *djester.Tester, t *testing.T, ft *FieldTest[FORMFIELD]) forms.Form
	FormField         any // FORMFIELD |  func(d *djester.Tester, t *testing.T, ft *FieldTest[FORMFIELD]) FORMFIELD
	Default           any // any | func(d *djester.Tester, t *testing.T, ft *FieldTest[FORMFIELD]) any
	ExpectedHTML      func(d *djester.Tester, t *testing.T, ft *FieldTest[FORMFIELD], fieldName string) []djester.HTMLAssertFunc
	Handle            []func(d *djester.Tester, t *testing.T, ft *FieldTest[FORMFIELD], field FORMFIELD, initial_data, cleaned_data any, errors []error)
}

func (f *FieldTest[FORMFIELD]) Name() string { return f.Label }

func (f *FieldTest[FORMFIELD]) Bench(dj *djester.Tester, b *testing.B) {
	b.Skipf("%T(%s) does not implenent benchmarks", f, f.Label)
}

func (f *FieldTest[FORMFIELD]) requestForForm(d *djester.Tester, t *testing.T, form forms.Form, fld FORMFIELD, urlValues url.Values, fileValues map[string][]filesystem.FileHeader) *http.Request {
	if f.RequestForForm != nil {
		return f.RequestForForm(d, t, f, form, fld, urlValues, fileValues)
	}
	return httptest.NewRequestWithContext(t.Context(), http.MethodPost, _FORMFIELD_REQUEST_URL, nil)
}

func (f *FieldTest[FORMFIELD]) getForm(d *djester.Tester, t *testing.T) forms.Form {
	if f.GetFormOverride != nil {
		return f.GetFormOverride(d, t, f)
	}
	return forms.NewBaseForm(t.Context())
}

func (f *FieldTest[FORMFIELD]) getFormField(d *djester.Tester, t *testing.T) (formField FORMFIELD, fieldName string) {
	switch fld := f.FormField.(type) {
	case FORMFIELD:
		formField = fld
	case func(d *djester.Tester, t *testing.T, ft *FieldTest[FORMFIELD]) FORMFIELD:
		formField = fld(d, t, f)
	default:
		t.Fatal("Formfield is not of type FORMFIELD or func(d *djester.Tester, t *testing.T, ft *FieldTest[FORMFIELD]) FORMFIELD")
	}

	if django_reflect.IsZero(formField) {
		t.Fatal("FormField is nil")
	}

	var defaultValue any
	switch _default := f.Default.(type) {
	case func(d *djester.Tester, t *testing.T, ft *FieldTest[FORMFIELD]) any:
		defaultValue = _default(d, t, f)
	default:
		defaultValue = _default
	}

	if f.FieldNameOverride != "" {
		fieldName = f.FieldNameOverride
	} else {
		fieldName = fmt.Sprintf("formField-%d", rand.Int())
	}

	formField.SetName(fieldName)
	formField.SetDefault(func() interface{} {
		return defaultValue
	})

	return formField, fieldName
}

func (f *FieldTest[FORMFIELD]) getFormData(d *djester.Tester, t *testing.T, fld FORMFIELD, fieldName string) (submitData url.Values, submitFiles map[string][]filesystem.FileHeader, useData bool) {
	if f.FormData == nil {
		return nil, nil, false
	}

	var (
		data       []string
		files      []filesystem.FileHeader
		urlValues  = make(url.Values, 1)
		fileValues = make(map[string][]filesystem.FileHeader, 1)
	)

	switch getData := f.FormData.(type) {
	case string:
		data = []string{getData}
	case []string:
		data = getData
	case func(d *djester.Tester, t *testing.T, ft *FieldTest[FORMFIELD], f FORMFIELD) (urlValues []string, files []filesystem.FileHeader):
		data, files = getData(d, t, f, fld)
	}

	if len(data) > 0 {
		urlValues[fieldName] = data
	}

	if len(files) > 0 {
		fileValues[fieldName] = files
	}

	return urlValues, fileValues, true
}

func (f *FieldTest[FORMFIELD]) renderFormBoundField(t *testing.T, fld forms.BoundField) *goquery.Document {
	defer func() {
		if err := recover(); err != nil {
			t.Fatalf("panic while rendering widget:\n%s", debug.Stack())
		}
	}()
	var doc, err = goquery.NewDocumentFromReader(strings.NewReader(string(
		fld.Field(),
	)))
	if err != nil {
		t.Fatalf("Error while initializing goquery document: %v", err)
	}
	return doc
}

func (f *FieldTest[FORMFIELD]) Test(d *djester.Tester, t *testing.T) {
	if f.FormField == nil {
		t.Fatal("FormField cannot be nil.")
	}

	if f.FormData == nil && f.FormWithData != nil {
		t.Fatal("You must provide both GetFormData and FormWithData when FormWithData is set.")
	}

	var (
		form                 = f.getForm(d, t)
		formField, fieldName = f.getFormField(d, t)
		data, files, submit  = f.getFormData(d, t, formField, fieldName)
		fldDefault           = formField.Default()
	)

	form.AddField(fieldName, formField)

	if submit {
		if f.FormWithData != nil {
			form = f.FormWithData(d, t, f, formField, fieldName, data, files, form)
		} else {
			form = forms.Initialize(form,
				forms.WithData[forms.Form](data, files, f.requestForForm(d, t, form, formField, data, files)),
			)
		}

		isValid := forms.IsValid(t.Context(), form)
		if isValid != f.ExpectsValid {
			if f.ExpectsValid {
				t.Fatal("valid form expected, but form was not valid")
			} else {
				t.Fatal("invalid form expected, but form was valid")
			}
		}

		var formErrors = make([]error, 0)
		formErrors = append(formErrors, form.ErrorList()...)

		var boundErrors = form.BoundErrors()
		if boundErrors != nil && boundErrors.Len() > 0 {
			var boundList, ok = boundErrors.Get(fieldName)
			if ok && len(boundList) > 0 {
				formErrors = append(formErrors, boundList...)
			}
		}

		var (
			cleanedData any
			ok          bool
		)
		if isValid {
			clndMap := form.CleanedData()
			cleanedData, ok = clndMap[fieldName]
			if !ok {
				t.Fatalf("Could not find fieldname %q in %v", fieldName, cleanedData)
			}
		}

		for _, expected := range f.ExpectedErrors {
			var found bool
			for _, err := range formErrors {
				if errors.Is(err, expected) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected error %q not found", expected)
			}
		}

		for _, h := range f.Handle {
			h(d, t, f, formField, fldDefault, cleanedData, formErrors)
		}
	}

	if f.ExpectedHTML != nil {
		assertHtml := f.ExpectedHTML(d, t, f, fieldName)

		var (
			boundForm      = form.BoundForm()
			boundFields    = boundForm.FieldMap()
			boundField, ok = boundFields[fieldName]
		)

		if !ok {
			t.Fatalf(
				"field %q not found in bound form fields: %v",
				fieldName, slices.Collect(maps.Keys(boundFields)),
			)
		}

		document := f.renderFormBoundField(t, boundField)
		asserter := d.Assert(t, true)
		asserter.AssertHTMLDoc(document, assertHtml...)
	}
}
