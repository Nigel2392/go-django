package formsets_test

import (
	"context"
	"net/url"
	"testing"

	forms "github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/formsets"
)

func newTestForm(ctx context.Context) forms.Form {
	return forms.Initialize(
		forms.NewBaseForm(ctx),
		forms.WithFields(
			fields.CharField(
				fields.Name("title"),
				fields.Required(false),
			),
		),
	)
}

func newTestFormSet(ctx context.Context, canDelete bool) *formsets.BaseFormSet[forms.Form] {
	return formsets.NewBaseFormSet[forms.Form](ctx, formsets.FormsetOptions[forms.Form]{
		NewForm: func(c context.Context) forms.Form {
			return newTestForm(c)
		},
		DefaultForms: func(c context.Context, _, _ int) ([]forms.Form, error) {
			return []forms.Form{newTestForm(c)}, nil
		},
		GetDefaults: func(context.Context, int) []map[string]interface{} {
			return []map[string]interface{}{{"title": "same"}}
		},
		MinNum:     1,
		Extra:      0,
		MaxNum:     1,
		CanAdd:     false,
		CanDelete:  canDelete,
		SkipPrefix: true,
	})
}

func TestFormSetHasChanged(t *testing.T) {
	t.Run("UnchangedData", func(t *testing.T) {
		ctx := context.Background()
		fs := newTestFormSet(ctx, false)
		fs.WithData(url.Values{
			"title": {"same"},
		}, nil, nil)

		if !forms.IsValid(ctx, fs) {
			t.Fatalf("expected formset to be valid")
		}

		if fs.HasChanged() {
			t.Fatalf("expected HasChanged() to be false for unchanged data")
		}
	})

	t.Run("ChangedData", func(t *testing.T) {
		ctx := context.Background()
		fs := newTestFormSet(ctx, false)
		fs.WithData(url.Values{
			"title": {"updated"},
		}, nil, nil)

		if !forms.IsValid(ctx, fs) {
			t.Fatalf("expected formset to be valid")
		}

		if !fs.HasChanged() {
			t.Fatalf("expected HasChanged() to be true for changed data")
		}
	})

	t.Run("DeletedForm", func(t *testing.T) {
		ctx := context.Background()
		fs := newTestFormSet(ctx, true)
		fs.WithData(url.Values{
			"title":    {"same"},
			"__DELETE__": {"on"},
		}, nil, nil)

		if !forms.IsValid(ctx, fs) {
			t.Fatalf("expected formset to be valid")
		}

		if !fs.HasChanged() {
			t.Fatalf("expected HasChanged() to be true when a form is marked for deletion")
		}
	})
}
