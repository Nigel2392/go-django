package formsets

import (
	"context"
	"maps"
	"net/http"
	"net/url"
	"testing"

	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
)

// persistentInitialForm wraps BaseForm and stores initial data in a separate field
// that survives WithData/Reset.  This mirrors BaseModelForm.InitialData(), which
// reads from the model instance rather than from BaseForm.Initial, so the data
// remains accessible even after WithData clears BaseForm.Initial.
type persistentInitialForm struct {
	*forms.BaseForm
	stableInitial map[string]interface{}
}

func (f *persistentInitialForm) InitialData() map[string]interface{} {
	// Merge: stableInitial provides the DB-sourced values; BaseForm.InitialData()
	// may carry overrides set via SetInitial (same pattern as BaseModelForm).
	data := make(map[string]interface{})
	maps.Copy(data, f.stableInitial)
	maps.Copy(data, f.BaseForm.InitialData())
	return data
}

// makeTestForm creates a persistentInitialForm with a single optional CharField.
// stableInitial is populated with the given value, simulating what
// ModelFormPanel.GetForms() does (Load → SetInitial from model instance).
func makeTestForm(ctx context.Context, fieldName, initialValue string) *persistentInitialForm {
	base := forms.NewBaseForm(ctx)
	base.AddField(fieldName, fields.CharField(fields.Required(false)))
	return &persistentInitialForm{
		BaseForm:      base,
		stableInitial: map[string]interface{}{fieldName: initialValue},
	}
}

// runFormsetIsValid simulates the CheckIsValid path: set POST data on the formset then validate.
// The formset assigns each sub-form the prefix "<index>" (e.g. "0"), so field keys must be
// prefixed accordingly (e.g. "0-title").
func runFormsetIsValid(fs *BaseFormSet[*persistentInitialForm], postData url.Values) bool {
	fs.WithData(postData, nil, &http.Request{})
	return forms.IsValid(fs.Context(), fs)
}

// TestCheckIsValidPreservesInitialData verifies that HasChanged() is accurate after
// CheckIsValid when sub-forms carry per-instance initial data (e.g. loaded from DB).
// The key property: InitialData() must survive WithData/Reset so the formset can
// call SetInitial(getter.InitialData()) and give HasChanged() something to compare.
func TestCheckIsValidPreservesInitialData(t *testing.T) {
	const fieldName = "title"
	const initialValue = "original"

	makeFS := func(f *persistentInitialForm) *BaseFormSet[*persistentInitialForm] {
		return NewBaseFormSet[*persistentInitialForm](context.Background(), FormsetOptions[*persistentInitialForm]{
			MinNum:    1,
			MaxNum:    1,
			CanAdd:    false,
			CanDelete: false,
			NewForm: func(ctx context.Context) *persistentInitialForm {
				return makeTestForm(ctx, fieldName, initialValue)
			},
			DefaultForms: func(ctx context.Context, max, min int) ([]*persistentInitialForm, error) {
				return []*persistentInitialForm{f}, nil
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
