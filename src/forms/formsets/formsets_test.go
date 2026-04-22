package formsets

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
)

// makeTestForm creates a BaseForm with a single optional CharField and pre-set initial data,
// simulating what ModelFormPanel.GetForms() does (Load → SetInitial).
func makeTestForm(ctx context.Context, fieldName, initialValue string) *forms.BaseForm {
	f := forms.NewBaseForm(ctx)
	f.AddField(fieldName, fields.CharField(fields.Required(false)))
	f.SetInitial(map[string]interface{}{fieldName: initialValue})
	return f
}

// runFormsetIsValid simulates the CheckIsValid path: set POST data on the formset then validate.
// The formset assigns each sub-form the prefix "<index>" (e.g. "0"), so field keys must be
// prefixed accordingly (e.g. "0-title").
func runFormsetIsValid(fs *BaseFormSet[*forms.BaseForm], postData url.Values) bool {
	fs.WithData(postData, nil, &http.Request{})
	return forms.IsValid(context.Background(), fs)
}

// TestCheckIsValidPreservesInitialData verifies that initial data set on forms before
// CheckIsValid is not permanently lost when WithData resets it, so that HasChanged()
// returns the correct result afterwards.
func TestCheckIsValidPreservesInitialData(t *testing.T) {
	const fieldName = "title"
	const initialValue = "original"

	makeFS := func(f *forms.BaseForm) *BaseFormSet[*forms.BaseForm] {
		return NewBaseFormSet[*forms.BaseForm](context.Background(), FormsetOptions[*forms.BaseForm]{
			MinNum:    1,
			MaxNum:    1,
			CanAdd:    false,
			CanDelete: false,
			NewForm: func(ctx context.Context) *forms.BaseForm {
				return makeTestForm(ctx, fieldName, initialValue)
			},
			DefaultForms: func(ctx context.Context, max, min int) ([]*forms.BaseForm, error) {
				return []*forms.BaseForm{f}, nil
			},
		})
	}

	t.Run("unchanged submission reports HasChanged=false", func(t *testing.T) {
		f := makeTestForm(context.Background(), fieldName, initialValue)
		fs := makeFS(f)

		// The formset assigns prefix "0" to the first form, so the POST key is "0-title".
		runFormsetIsValid(fs, url.Values{"0-" + fieldName: {initialValue}})

		if fs.HasChanged() {
			t.Error("HasChanged() should be false when POST data matches initial data")
		}
	})

	t.Run("changed submission reports HasChanged=true", func(t *testing.T) {
		f := makeTestForm(context.Background(), fieldName, initialValue)
		fs := makeFS(f)

		runFormsetIsValid(fs, url.Values{"0-" + fieldName: {"modified"}})

		if !fs.HasChanged() {
			t.Error("HasChanged() should be true when POST data differs from initial data")
		}
	})
}
