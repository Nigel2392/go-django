package formsets_test

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	forms "github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/formsets"
)

// ─────────────────────────────────────────────────────────────────────────────
// Form factories
// ─────────────────────────────────────────────────────────────────────────────

// newTitleForm creates a single-field form. required controls whether the
// "title" CharField is required.
func newTitleForm(ctx context.Context, required bool) forms.Form {
	return forms.Initialize(
		forms.NewBaseForm(ctx),
		forms.WithFields(
			fields.CharField(
				fields.Name("title"),
				fields.Required(required),
			),
		),
	)
}

// newMultiFieldForm has a required "title" and an optional "body" field.
func newMultiFieldForm(ctx context.Context) forms.Form {
	return forms.Initialize(
		forms.NewBaseForm(ctx),
		forms.WithFields(
			fields.CharField(
				fields.Name("title"),
				fields.Required(true),
			),
			fields.CharField(
				fields.Name("body"),
				fields.Required(false),
			),
		),
	)
}

// ─────────────────────────────────────────────────────────────────────────────
// Formset factories
// ─────────────────────────────────────────────────────────────────────────────

// newThreeFormSet builds a three-form formset with no management form
// (MinNum == MaxNum == 3, CanAdd == false).  The caller may pre-populate
// opts with any additional flags (CanDelete, CanOrder, DeleteForms, …);
// this factory fills in everything else.
//
// Default initial values are: Alpha / Beta / Gamma.
func newThreeFormSet(ctx context.Context, opts formsets.FormsetOptions[forms.Form]) *formsets.BaseFormSet[forms.Form] {
	opts.NewForm = func(c context.Context) forms.Form { return newTitleForm(c, false) }
	opts.DefaultForms = func(c context.Context, _, _ int) ([]forms.Form, error) {
		return []forms.Form{
			newTitleForm(c, false),
			newTitleForm(c, false),
			newTitleForm(c, false),
		}, nil
	}
	opts.GetDefaults = func(_ context.Context, _ int) []map[string]interface{} {
		return []map[string]interface{}{
			{"title": "Alpha"},
			{"title": "Beta"},
			{"title": "Gamma"},
		}
	}
	opts.MinNum = 3
	opts.MaxNum = 3
	opts.Extra = 0
	opts.CanAdd = false
	return formsets.NewBaseFormSet[forms.Form](ctx, opts)
}

// newThreeRequiredFormSet is like newThreeFormSet but title is required on
// every child form.
func newThreeRequiredFormSet(ctx context.Context) *formsets.BaseFormSet[forms.Form] {
	return formsets.NewBaseFormSet[forms.Form](ctx, formsets.FormsetOptions[forms.Form]{
		NewForm: func(c context.Context) forms.Form { return newTitleForm(c, true) },
		DefaultForms: func(c context.Context, _, _ int) ([]forms.Form, error) {
			return []forms.Form{
				newTitleForm(c, true),
				newTitleForm(c, true),
				newTitleForm(c, true),
			}, nil
		},
		GetDefaults: func(_ context.Context, _ int) []map[string]interface{} {
			return []map[string]interface{}{
				{"title": "Alpha"},
				{"title": "Beta"},
				{"title": "Gamma"},
			}
		},
		MinNum: 3,
		MaxNum: 3,
		Extra:  0,
		CanAdd: false,
	})
}

// newThreeMultiFieldFormSet has three forms each with a required "title" and
// optional "body" field.
func newThreeMultiFieldFormSet(ctx context.Context) *formsets.BaseFormSet[forms.Form] {
	return formsets.NewBaseFormSet[forms.Form](ctx, formsets.FormsetOptions[forms.Form]{
		NewForm: func(c context.Context) forms.Form { return newMultiFieldForm(c) },
		DefaultForms: func(c context.Context, _, _ int) ([]forms.Form, error) {
			return []forms.Form{
				newMultiFieldForm(c),
				newMultiFieldForm(c),
				newMultiFieldForm(c),
			}, nil
		},
		GetDefaults: func(_ context.Context, _ int) []map[string]interface{} {
			return []map[string]interface{}{
				{"title": "Alpha", "body": "Alpha body"},
				{"title": "Beta", "body": "Beta body"},
				{"title": "Gamma", "body": "Gamma body"},
			}
		},
		MinNum: 3,
		MaxNum: 3,
		Extra:  0,
		CanAdd: false,
	})
}

// newMgmtFormSet builds a formset that includes a management form
// (MinNum=0, MaxNum=5, CanAdd=true).  n controls how many default forms
// DefaultForms returns.
func newMgmtFormSet(ctx context.Context, n int) *formsets.BaseFormSet[forms.Form] {
	var defaults []map[string]interface{}
	for i := 0; i < n; i++ {
		defaults = append(defaults, map[string]interface{}{
			"title": fmt.Sprintf("Form%d", i),
		})
	}
	return formsets.NewBaseFormSet[forms.Form](ctx, formsets.FormsetOptions[forms.Form]{
		NewForm: func(c context.Context) forms.Form { return newTitleForm(c, false) },
		DefaultForms: func(c context.Context, _, _ int) ([]forms.Form, error) {
			out := make([]forms.Form, n)
			for i := 0; i < n; i++ {
				out[i] = newTitleForm(c, false)
			}
			return out, nil
		},
		GetDefaults: func(_ context.Context, _ int) []map[string]interface{} {
			return defaults
		},
		MinNum: 0,
		MaxNum: 5,
		Extra:  0,
		CanAdd: true,
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Package-level test-data fixtures
// ─────────────────────────────────────────────────────────────────────────────

var (
	// ── unchanged / changed ─────────────────────────────────────────────────

	// Each title matches the corresponding GetDefaults entry (Alpha/Beta/Gamma)
	threeFormUnchangedData = url.Values{
		"0-title": {"Alpha"},
		"1-title": {"Beta"},
		"2-title": {"Gamma"},
	}

	// All three titles differ from the GetDefaults initial values
	threeFormAllChangedData = url.Values{
		"0-title": {"A"},
		"1-title": {"B"},
		"2-title": {"C"},
	}

	// Only the first form's title differs
	threeFormFirstChangedData = url.Values{
		"0-title": {"CHANGED"},
		"1-title": {"Beta"},
		"2-title": {"Gamma"},
	}

	// Only the middle form's title differs
	threeFormMiddleChangedData = url.Values{
		"0-title": {"Alpha"},
		"1-title": {"CHANGED"},
		"2-title": {"Gamma"},
	}

	// Only the last form's title differs
	threeFormLastChangedData = url.Values{
		"0-title": {"Alpha"},
		"1-title": {"Beta"},
		"2-title": {"CHANGED"},
	}

	// ── required-field variants ──────────────────────────────────────────────

	threeFormRequiredValidData = url.Values{
		"0-title": {"Alpha"},
		"1-title": {"Beta"},
		"2-title": {"Gamma"},
	}

	// Middle form submits an empty required title
	threeFormRequiredMidEmptyData = url.Values{
		"0-title": {"Alpha"},
		"1-title": {""},
		"2-title": {"Gamma"},
	}

	// All required titles are empty
	threeFormRequiredAllEmptyData = url.Values{
		"0-title": {""},
		"1-title": {""},
		"2-title": {""},
	}

	// ── deletion fixtures ────────────────────────────────────────────────────

	threeFormDeleteNoneData = url.Values{
		"0-title": {"Alpha"},
		"1-title": {"Beta"},
		"2-title": {"Gamma"},
	}

	threeFormDeleteFirstData = url.Values{
		"0-" + formsets.DELETION_FIELD_NAME: {"on"},
		"1-title":                           {"Beta"},
		"2-title":                           {"Gamma"},
	}

	threeFormDeleteMiddleData = url.Values{
		"0-title":                           {"Alpha"},
		"1-" + formsets.DELETION_FIELD_NAME: {"on"},
		"2-title":                           {"Gamma"},
	}

	threeFormDeleteLastData = url.Values{
		"0-title":                           {"Alpha"},
		"1-title":                           {"Beta"},
		"2-" + formsets.DELETION_FIELD_NAME: {"on"},
	}

	threeFormDeleteAllData = url.Values{
		"0-" + formsets.DELETION_FIELD_NAME: {"on"},
		"1-" + formsets.DELETION_FIELD_NAME: {"on"},
		"2-" + formsets.DELETION_FIELD_NAME: {"on"},
	}

	// ── ordering fixtures ────────────────────────────────────────────────────

	// Reverse the natural order: slot 0 gets order 2, slot 1 gets order 1,
	// slot 2 gets order 0.  Expected final sequence: Gamma, Beta, Alpha.
	threeFormReverseOrderData = url.Values{
		"0-" + formsets.ORDERING_FIELD_NAME: {"2"},
		"0-title":                           {"Alpha"},
		"1-" + formsets.ORDERING_FIELD_NAME: {"1"},
		"1-title":                           {"Beta"},
		"2-" + formsets.ORDERING_FIELD_NAME: {"0"},
		"2-title":                           {"Gamma"},
	}

	// Forward order – already sorted, so the output equals the input.
	threeFormForwardOrderData = url.Values{
		"0-" + formsets.ORDERING_FIELD_NAME: {"0"},
		"0-title":                           {"Alpha"},
		"1-" + formsets.ORDERING_FIELD_NAME: {"1"},
		"1-title":                           {"Beta"},
		"2-" + formsets.ORDERING_FIELD_NAME: {"2"},
		"2-title":                           {"Gamma"},
	}

	// ── multi-field fixtures ─────────────────────────────────────────────────

	threeFormMultiAllValidData = url.Values{
		"0-title": {"Alpha"},
		"0-body":  {"Alpha body"},
		"1-title": {"Beta"},
		"1-body":  {"Beta body"},
		"2-title": {"Gamma"},
		"2-body":  {"Gamma body"},
	}

	// Middle form leaves the required title empty
	threeFormMultiMidTitleEmptyData = url.Values{
		"0-title": {"Alpha"},
		"0-body":  {"Alpha body"},
		"1-title": {""},
		"1-body":  {"Beta body"},
		"2-title": {"Gamma"},
		"2-body":  {"Gamma body"},
	}
)

// ─────────────────────────────────────────────────────────────────────────────
// IsValid
// ─────────────────────────────────────────────────────────────────────────────

func TestFormSet_IsValid(t *testing.T) {
	t.Run("AllFormsValid", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeRequiredFormSet(ctx)
		fs.WithData(threeFormRequiredValidData, nil, nil)
		if !forms.IsValid(ctx, fs) {
			t.Fatalf("expected formset to be valid, errors: %v", fs.ErrorList())
		}
	})

	t.Run("MiddleFormInvalid", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeRequiredFormSet(ctx)
		fs.WithData(threeFormRequiredMidEmptyData, nil, nil)
		if forms.IsValid(ctx, fs) {
			t.Fatal("expected formset invalid when middle required field is empty")
		}
	})

	t.Run("AllFormsInvalid", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeRequiredFormSet(ctx)
		fs.WithData(threeFormRequiredAllEmptyData, nil, nil)
		if forms.IsValid(ctx, fs) {
			t.Fatal("expected formset invalid when all required fields are empty")
		}
	})

	t.Run("FirstFormInvalid", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeRequiredFormSet(ctx)
		fs.WithData(url.Values{
			"0-title": {""},
			"1-title": {"Beta"},
			"2-title": {"Gamma"},
		}, nil, nil)
		if forms.IsValid(ctx, fs) {
			t.Fatal("expected formset invalid when first required field is empty")
		}
	})

	t.Run("LastFormInvalid", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeRequiredFormSet(ctx)
		fs.WithData(url.Values{
			"0-title": {"Alpha"},
			"1-title": {"Beta"},
			"2-title": {""},
		}, nil, nil)
		if forms.IsValid(ctx, fs) {
			t.Fatal("expected formset invalid when last required field is empty")
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// FormCount
// ─────────────────────────────────────────────────────────────────────────────

func TestFormSet_FormCount(t *testing.T) {
	t.Run("ThreeFormsAfterValidation", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeRequiredFormSet(ctx)
		fs.WithData(threeFormRequiredValidData, nil, nil)
		forms.IsValid(ctx, fs)
		flist, err := fs.Forms()
		if err != nil {
			t.Fatalf("Forms() error: %v", err)
		}
		if len(flist) != 3 {
			t.Fatalf("expected 3 forms, got %d", len(flist))
		}
	})

	t.Run("ZeroFormsWithNoDefaults", func(t *testing.T) {
		ctx := context.Background()
		// A formset that has 0 required forms and no defaults
		fs := formsets.NewBaseFormSet[forms.Form](ctx, formsets.FormsetOptions[forms.Form]{
			NewForm: func(c context.Context) forms.Form { return newTitleForm(c, false) },
			MinNum:  0,
			MaxNum:  0,
			Extra:   0,
			CanAdd:  false,
		})
		fs.WithData(url.Values{}, nil, nil)
		forms.IsValid(ctx, fs)
		flist, _ := fs.Forms()
		if len(flist) != 0 {
			t.Fatalf("expected 0 forms, got %d", len(flist))
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// HasChanged
// ─────────────────────────────────────────────────────────────────────────────

func TestFormSet_HasChanged(t *testing.T) {
	type tc struct {
		name        string
		data        url.Values
		canDelete   bool
		wantChanged bool
	}

	cases := []tc{
		{
			name:        "AllUnchanged",
			data:        threeFormUnchangedData,
			wantChanged: false,
		},
		{
			name:        "AllChanged",
			data:        threeFormAllChangedData,
			wantChanged: true,
		},
		{
			name:        "FirstChanged",
			data:        threeFormFirstChangedData,
			wantChanged: true,
		},
		{
			name:        "MiddleChanged",
			data:        threeFormMiddleChangedData,
			wantChanged: true,
		},
		{
			name:        "LastChanged",
			data:        threeFormLastChangedData,
			wantChanged: true,
		},
		// Even if the remaining titles match initial, a deletion is a change.
		{
			name:        "DeleteFirst",
			data:        threeFormDeleteFirstData,
			canDelete:   true,
			wantChanged: true,
		},
		{
			name:        "DeleteMiddle",
			data:        threeFormDeleteMiddleData,
			canDelete:   true,
			wantChanged: true,
		},
		{
			name:        "DeleteLast",
			data:        threeFormDeleteLastData,
			canDelete:   true,
			wantChanged: true,
		},
		{
			name:        "DeleteAll",
			data:        threeFormDeleteAllData,
			canDelete:   true,
			wantChanged: true,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()
			fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{
				CanDelete: c.canDelete,
			})
			fs.WithData(c.data, nil, nil)
			forms.IsValid(ctx, fs)

			if got := fs.HasChanged(); got != c.wantChanged {
				t.Fatalf("HasChanged() = %v, want %v", got, c.wantChanged)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Deletion
// ─────────────────────────────────────────────────────────────────────────────

func TestFormSet_Deletion(t *testing.T) {
	type tc struct {
		name        string
		data        url.Values
		wantFinal   int
		wantDeleted int
	}

	cases := []tc{
		{"DeleteNone", threeFormDeleteNoneData, 3, 0},
		{"DeleteFirst", threeFormDeleteFirstData, 2, 1},
		{"DeleteMiddle", threeFormDeleteMiddleData, 2, 1},
		{"DeleteLast", threeFormDeleteLastData, 2, 1},
		{"DeleteAll", threeFormDeleteAllData, 0, 3},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()
			fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{
				CanDelete: true,
			})
			fs.WithData(c.data, nil, nil)
			forms.IsValid(ctx, fs)

			flist, _ := fs.Forms()
			if len(flist) != c.wantFinal {
				t.Errorf("Forms() len = %d, want %d", len(flist), c.wantFinal)
			}
			del := fs.DeletedForms()
			if len(del) != c.wantDeleted {
				t.Errorf("DeletedForms() len = %d, want %d", len(del), c.wantDeleted)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// CleanedDataList
// ─────────────────────────────────────────────────────────────────────────────

func TestFormSet_CleanedDataList(t *testing.T) {
	t.Run("AllFormsValid_TitlesPresent", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{})
		fs.WithData(threeFormAllChangedData, nil, nil)
		if !forms.IsValid(ctx, fs) {
			t.Fatalf("expected valid: %v", fs.ErrorList())
		}

		list := fs.CleanedDataList()
		if len(list) != 3 {
			t.Fatalf("expected 3 cleaned-data maps, got %d", len(list))
		}
		wantTitles := []string{"A", "B", "C"}
		for i, m := range list {
			v, ok := m["title"]
			if !ok {
				t.Errorf("form %d: 'title' missing from cleaned data", i)
				continue
			}
			if v != wantTitles[i] {
				t.Errorf("form %d: title = %q, want %q", i, v, wantTitles[i])
			}
		}
	})

	t.Run("UnchangedTitlesPreserved", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{})
		fs.WithData(threeFormUnchangedData, nil, nil)
		if !forms.IsValid(ctx, fs) {
			t.Fatalf("expected valid: %v", fs.ErrorList())
		}

		list := fs.CleanedDataList()
		wantTitles := []string{"Alpha", "Beta", "Gamma"}
		for i, m := range list {
			if m["title"] != wantTitles[i] {
				t.Errorf("form %d: title = %q, want %q", i, m["title"], wantTitles[i])
			}
		}
	})

	t.Run("AfterDeletion_CorrectCount", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{
			CanDelete: true,
		})
		fs.WithData(threeFormDeleteMiddleData, nil, nil)
		forms.IsValid(ctx, fs)

		list := fs.CleanedDataList()
		if len(list) != 2 {
			t.Fatalf("expected 2 entries after one deletion, got %d", len(list))
		}
	})

	t.Run("AfterAllDeleted_EmptyList", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{
			CanDelete: true,
		})
		fs.WithData(threeFormDeleteAllData, nil, nil)
		forms.IsValid(ctx, fs)

		list := fs.CleanedDataList()
		if len(list) != 0 {
			t.Fatalf("expected empty list after all deleted, got %d", len(list))
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Error aggregation
// ─────────────────────────────────────────────────────────────────────────────

// TestFormSet_ErrorLists tests form-level (non-field) errors that originate
// from per-form validators.  Required-field validation failures are field-level
// errors and therefore appear in BoundErrorsList, not ErrorLists.
func TestFormSet_ErrorLists(t *testing.T) {
	// A validator that injects a form-level error for the "error" title.
	markerValidator := func(f forms.Form, cleaned map[string]any) []error {
		if v, _ := cleaned["title"].(string); v == "error" {
			return []error{fmt.Errorf("title %q triggers a form-level error", v)}
		}
		return nil
	}

	t.Run("MiddleFormHasValidatorError", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{})
		fs.AddValidator(markerValidator)
		fs.WithData(url.Values{
			"0-title": {"Alpha"},
			"1-title": {"error"},
			"2-title": {"Gamma"},
		}, nil, nil)
		forms.IsValid(ctx, fs)

		lists := fs.ErrorLists()
		if len(lists) != 3 {
			t.Fatalf("expected 3 per-form error slices, got %d", len(lists))
		}
		if len(lists[0]) != 0 {
			t.Errorf("form 0 unexpectedly has errors: %v", lists[0])
		}
		if len(lists[1]) == 0 {
			t.Error("form 1 expected a form-level validator error")
		}
		if len(lists[2]) != 0 {
			t.Errorf("form 2 unexpectedly has errors: %v", lists[2])
		}
	})

	t.Run("FirstFormHasValidatorError", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{})
		fs.AddValidator(markerValidator)
		fs.WithData(url.Values{
			"0-title": {"error"},
			"1-title": {"Beta"},
			"2-title": {"Gamma"},
		}, nil, nil)
		forms.IsValid(ctx, fs)

		lists := fs.ErrorLists()
		if len(lists[0]) == 0 {
			t.Error("form 0 expected a form-level validator error")
		}
		// forms 1 and 2 should be fine
		if len(lists[1]) != 0 {
			t.Errorf("form 1 unexpectedly has errors: %v", lists[1])
		}
		if len(lists[2]) != 0 {
			t.Errorf("form 2 unexpectedly has errors: %v", lists[2])
		}
	})

	t.Run("NoValidatorErrors", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{})
		fs.AddValidator(markerValidator)
		fs.WithData(threeFormAllChangedData, nil, nil)
		forms.IsValid(ctx, fs)

		for i, errs := range fs.ErrorLists() {
			if len(errs) != 0 {
				t.Errorf("form %d unexpectedly has errors: %v", i, errs)
			}
		}
	})
}

func TestFormSet_BoundErrorsList(t *testing.T) {
	t.Run("MiddleFormTitleError", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeRequiredFormSet(ctx)
		fs.WithData(threeFormRequiredMidEmptyData, nil, nil)
		forms.IsValid(ctx, fs)

		lists := fs.BoundErrorsList()
		if len(lists) != 3 {
			t.Fatalf("expected 3 BoundErrors maps, got %d", len(lists))
		}
		if lists[0] != nil && lists[0].Len() != 0 {
			t.Errorf("form 0 unexpected bound errors")
		}
		if lists[1] == nil || lists[1].Len() == 0 {
			t.Error("form 1 expected bound errors, got none")
		} else {
			if _, ok := lists[1].Get("title"); !ok {
				t.Errorf("form 1 expected error on 'title', keys: %v", lists[1].Keys())
			}
		}
		if lists[2] != nil && lists[2].Len() != 0 {
			t.Errorf("form 2 unexpected bound errors")
		}
	})

	t.Run("AllFormsHaveBoundErrors", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeRequiredFormSet(ctx)
		fs.WithData(threeFormRequiredAllEmptyData, nil, nil)
		forms.IsValid(ctx, fs)

		lists := fs.BoundErrorsList()
		for i, be := range lists {
			if be == nil || be.Len() == 0 {
				t.Errorf("form %d expected bound errors", i)
			}
		}
	})
}

func TestFormSet_AggregatedErrorList(t *testing.T) {
	// The formset-level ErrorList() collects form-level (non-field) errors from
	// each child form.  Field-level errors (e.g., required field missing) live in
	// BoundErrors and do not appear here; those make the formset invalid via
	// forms.IsValid returning false.
	markerValidator := func(f forms.Form, cleaned map[string]any) []error {
		if v, _ := cleaned["title"].(string); v == "error" {
			return []error{fmt.Errorf("form-level error on %q", v)}
		}
		return nil
	}

	t.Run("ValidatorError_AggregatedInErrorList", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{})
		fs.AddValidator(markerValidator)
		fs.WithData(url.Values{
			"0-title": {"Alpha"},
			"1-title": {"error"},
			"2-title": {"Gamma"},
		}, nil, nil)
		forms.IsValid(ctx, fs)

		errs := fs.ErrorList()
		if len(errs) == 0 {
			t.Fatal("expected aggregated form-level errors, got none")
		}
	})

	t.Run("FieldErrors_MakeFormsetInvalid_NotInErrorList", func(t *testing.T) {
		// Required-field failures are field-level; they make IsValid return false
		// but do NOT populate the formset-level ErrorList().
		ctx := context.Background()
		fs := newThreeRequiredFormSet(ctx)
		fs.WithData(threeFormRequiredAllEmptyData, nil, nil)
		if forms.IsValid(ctx, fs) {
			t.Fatal("expected formset invalid")
		}
		// BoundErrors contain the field-level errors
		boundLists := fs.BoundErrorsList()
		for i, be := range boundLists {
			if be == nil || be.Len() == 0 {
				t.Errorf("form %d expected bound field errors, got none", i)
			}
		}
	})

	t.Run("AllFormsValid_NoAggregatedErrors", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeRequiredFormSet(ctx)
		fs.WithData(threeFormRequiredValidData, nil, nil)
		forms.IsValid(ctx, fs)

		if errs := fs.ErrorList(); len(errs) != 0 {
			t.Errorf("expected no aggregated errors, got %d", len(errs))
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Custom per-form validators
// ─────────────────────────────────────────────────────────────────────────────

func TestFormSet_CustomValidator(t *testing.T) {
	bannedValidator := func(f forms.Form, cleaned map[string]any) []error {
		if v, _ := cleaned["title"].(string); v == "banned" {
			return []error{fmt.Errorf("title %q is not allowed", v)}
		}
		return nil
	}

	t.Run("ValidatorRejectsBannedValue", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{})
		fs.AddValidator(bannedValidator)
		fs.WithData(url.Values{
			"0-title": {"Alpha"},
			"1-title": {"banned"},
			"2-title": {"Gamma"},
		}, nil, nil)

		if forms.IsValid(ctx, fs) {
			t.Fatal("expected formset invalid due to banned value")
		}
		lists := fs.ErrorLists()
		if len(lists) != 3 {
			t.Fatalf("expected 3 error lists, got %d", len(lists))
		}
		if len(lists[0]) != 0 {
			t.Errorf("form 0 should have no errors, got %v", lists[0])
		}
		if len(lists[1]) == 0 {
			t.Error("form 1 should have a validator error for the banned value")
		}
		if len(lists[2]) != 0 {
			t.Errorf("form 2 should have no errors, got %v", lists[2])
		}
	})

	t.Run("ValidatorRejectsAllForms_OnlyFirstFormGetsError", func(t *testing.T) {
		// Validators run only while isValid is still true.  Once form 0's validator
		// adds an error and isValid becomes false, forms 1 and 2 skip the validator
		// entirely—this is the formset's designed short-circuit behaviour.
		// The test explicitly asserts both that form 0 has an error AND that forms
		// 1 and 2 do not, so that a future change to the short-circuit logic would
		// be caught immediately.
		ctx := context.Background()
		fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{})
		fs.AddValidator(bannedValidator)
		fs.WithData(url.Values{
			"0-title": {"banned"},
			"1-title": {"banned"},
			"2-title": {"banned"},
		}, nil, nil)

		if forms.IsValid(ctx, fs) {
			t.Fatal("expected formset invalid when all titles are banned")
		}
		lists := fs.ErrorLists()
		if len(lists[0]) == 0 {
			t.Error("form 0 should have the validator error")
		}
		// forms 1 and 2 never reach the validator because isValid was already false
		if len(lists[1]) != 0 {
			t.Errorf("form 1 should not have validator errors (short-circuit): %v", lists[1])
		}
		if len(lists[2]) != 0 {
			t.Errorf("form 2 should not have validator errors (short-circuit): %v", lists[2])
		}
	})

	t.Run("ValidatorAllowsAllValues", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{})
		fs.AddValidator(bannedValidator)
		fs.WithData(threeFormAllChangedData, nil, nil)

		if !forms.IsValid(ctx, fs) {
			t.Fatalf("expected formset valid, errors: %v", fs.ErrorList())
		}
	})

	t.Run("MultipleValidatorsRunInOrder", func(t *testing.T) {
		ctx := context.Background()
		var callOrder []int
		fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{})
		fs.AddValidator(func(f forms.Form, cleaned map[string]any) []error {
			callOrder = append(callOrder, 1)
			return nil
		})
		fs.AddValidator(func(f forms.Form, cleaned map[string]any) []error {
			callOrder = append(callOrder, 2)
			return nil
		})
		fs.WithData(threeFormAllChangedData, nil, nil)
		forms.IsValid(ctx, fs)

		// 2 validators × 3 forms = 6 calls, alternating 1, 2 per form
		if len(callOrder) != 6 {
			t.Fatalf("expected 6 validator calls (2×3), got %d: %v", len(callOrder), callOrder)
		}
		for i := 0; i < 6; i += 2 {
			if callOrder[i] != 1 || callOrder[i+1] != 2 {
				t.Errorf("unexpected call order at [%d,%d]: %v", i, i+1, callOrder)
			}
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Ordering
// ─────────────────────────────────────────────────────────────────────────────

func TestFormSet_Ordering(t *testing.T) {
	t.Run("ReverseOrder_GammaBetaAlpha", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{
			CanOrder: true,
		})
		fs.WithData(threeFormReverseOrderData, nil, nil)
		if !forms.IsValid(ctx, fs) {
			t.Fatalf("expected valid, errors: %v", fs.ErrorList())
		}

		list := fs.CleanedDataList()
		if len(list) != 3 {
			t.Fatalf("expected 3 forms, got %d", len(list))
		}
		wantTitles := []string{"Gamma", "Beta", "Alpha"}
		for i, m := range list {
			if m["title"] != wantTitles[i] {
				t.Errorf("form[%d] title = %q, want %q", i, m["title"], wantTitles[i])
			}
		}
	})

	t.Run("ForwardOrder_AlphaBetaGamma", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{
			CanOrder: true,
		})
		fs.WithData(threeFormForwardOrderData, nil, nil)
		if !forms.IsValid(ctx, fs) {
			t.Fatalf("expected valid, errors: %v", fs.ErrorList())
		}

		list := fs.CleanedDataList()
		wantTitles := []string{"Alpha", "Beta", "Gamma"}
		for i, m := range list {
			if m["title"] != wantTitles[i] {
				t.Errorf("form[%d] title = %q, want %q", i, m["title"], wantTitles[i])
			}
		}
	})

	t.Run("OrderingPreservesFormCount", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{
			CanOrder: true,
		})
		fs.WithData(threeFormReverseOrderData, nil, nil)
		forms.IsValid(ctx, fs)

		flist, _ := fs.Forms()
		if len(flist) != 3 {
			t.Errorf("expected 3 forms after ordering, got %d", len(flist))
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Management form
// ─────────────────────────────────────────────────────────────────────────────

func TestFormSet_ManagementForm(t *testing.T) {
	t.Run("TotalFormsEqualsDefaultCount", func(t *testing.T) {
		ctx := context.Background()
		fs := newMgmtFormSet(ctx, 3)
		fs.WithData(url.Values{
			formsets.TOTAL_FORM_COUNT: {"3"},
			"0-title":                 {"Form0"},
			"1-title":                 {"Form1"},
			"2-title":                 {"Form2"},
		}, nil, nil)
		if !forms.IsValid(ctx, fs) {
			t.Fatalf("expected valid: %v", fs.ErrorList())
		}
		flist, _ := fs.Forms()
		if len(flist) != 3 {
			t.Errorf("expected 3 forms, got %d", len(flist))
		}
	})

	t.Run("TotalFormsLessThanDefaultCount", func(t *testing.T) {
		ctx := context.Background()
		fs := newMgmtFormSet(ctx, 3)
		fs.WithData(url.Values{
			formsets.TOTAL_FORM_COUNT: {"2"},
			"0-title":                 {"Form0"},
			"1-title":                 {"Form1"},
		}, nil, nil)
		if !forms.IsValid(ctx, fs) {
			t.Fatalf("expected valid: %v", fs.ErrorList())
		}
		flist, _ := fs.Forms()
		if len(flist) != 2 {
			t.Errorf("expected 2 forms (truncated), got %d", len(flist))
		}
	})

	t.Run("TotalFormsGreaterThanDefaults_CanAdd", func(t *testing.T) {
		ctx := context.Background()
		fs := newMgmtFormSet(ctx, 2)
		fs.WithData(url.Values{
			formsets.TOTAL_FORM_COUNT: {"3"},
			"0-title":                 {"Form0"},
			"1-title":                 {"Form1"},
			"2-title":                 {"ExtraForm"},
		}, nil, nil)
		if !forms.IsValid(ctx, fs) {
			t.Fatalf("expected valid: %v", fs.ErrorList())
		}
		flist, _ := fs.Forms()
		if len(flist) != 3 {
			t.Errorf("expected 3 forms (2 default + 1 new), got %d", len(flist))
		}
	})

	t.Run("MissingTotalForms_Invalid", func(t *testing.T) {
		ctx := context.Background()
		fs := newMgmtFormSet(ctx, 2)
		// No TOTAL_FORMS key → management form fails
		fs.WithData(url.Values{
			"0-title": {"Form0"},
			"1-title": {"Form1"},
		}, nil, nil)
		if forms.IsValid(ctx, fs) {
			t.Fatal("expected invalid when TOTAL_FORMS is absent")
		}
	})

	t.Run("TotalFormsExceedsMaxNum_Invalid", func(t *testing.T) {
		ctx := context.Background()
		fs := newMgmtFormSet(ctx, 3)
		// MaxNum is 5, so 6 should fail management-form validation
		fs.WithData(url.Values{
			formsets.TOTAL_FORM_COUNT: {"6"},
			"0-title":                 {"Form0"},
		}, nil, nil)
		if forms.IsValid(ctx, fs) {
			t.Fatal("expected invalid when TOTAL_FORMS > MaxNum")
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Multiple fields per form
// ─────────────────────────────────────────────────────────────────────────────

func TestFormSet_MultipleFields(t *testing.T) {
	t.Run("AllFormsValid_TitleAndBodyPresent", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeMultiFieldFormSet(ctx)
		fs.WithData(threeFormMultiAllValidData, nil, nil)
		if !forms.IsValid(ctx, fs) {
			t.Fatalf("expected valid: %v", fs.ErrorList())
		}

		list := fs.CleanedDataList()
		if len(list) != 3 {
			t.Fatalf("expected 3 maps, got %d", len(list))
		}
		wantTitles := []string{"Alpha", "Beta", "Gamma"}
		wantBodies := []string{"Alpha body", "Beta body", "Gamma body"}
		for i, m := range list {
			if m["title"] != wantTitles[i] {
				t.Errorf("form %d title = %q, want %q", i, m["title"], wantTitles[i])
			}
			if m["body"] != wantBodies[i] {
				t.Errorf("form %d body = %q, want %q", i, m["body"], wantBodies[i])
			}
		}
	})

	t.Run("MiddleFormRequiredTitleEmpty_Invalid", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeMultiFieldFormSet(ctx)
		fs.WithData(threeFormMultiMidTitleEmptyData, nil, nil)
		if forms.IsValid(ctx, fs) {
			t.Fatal("expected invalid when middle form's required title is empty")
		}
		lists := fs.BoundErrorsList()
		if len(lists) != 3 {
			t.Fatalf("expected 3 bound-error maps, got %d", len(lists))
		}
		if lists[0] != nil && lists[0].Len() != 0 {
			t.Errorf("form 0 unexpected bound errors")
		}
		if lists[1] == nil || lists[1].Len() == 0 {
			t.Error("form 1 expected bound errors on 'title'")
		}
		if lists[2] != nil && lists[2].Len() != 0 {
			t.Errorf("form 2 unexpected bound errors")
		}
	})

	t.Run("OptionalBodyMayBeEmpty", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeMultiFieldFormSet(ctx)
		fs.WithData(url.Values{
			"0-title": {"Alpha"},
			"0-body":  {""},
			"1-title": {"Beta"},
			"1-body":  {""},
			"2-title": {"Gamma"},
			"2-body":  {""},
		}, nil, nil)
		if !forms.IsValid(ctx, fs) {
			t.Fatalf("expected valid when optional body is empty: %v", fs.ErrorList())
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Form(i) accessor
// ─────────────────────────────────────────────────────────────────────────────

func TestFormSet_FormAccess(t *testing.T) {
	ctx := context.Background()
	fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{})
	fs.WithData(threeFormAllChangedData, nil, nil)
	forms.IsValid(ctx, fs)

	for _, idx := range []int{0, 1, 2} {
		t.Run(fmt.Sprintf("ValidIndex_%d", idx), func(t *testing.T) {
			_, ok := fs.Form(idx)
			if !ok {
				t.Errorf("Form(%d) should succeed", idx)
			}
		})
	}

	for _, idx := range []int{-1, 3, 100} {
		idx := idx
		t.Run(fmt.Sprintf("InvalidIndex_%d", idx), func(t *testing.T) {
			_, ok := fs.Form(idx)
			if ok {
				t.Errorf("Form(%d) should fail for a 3-form set", idx)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Prefix propagation
// ─────────────────────────────────────────────────────────────────────────────

func TestFormSet_Prefix(t *testing.T) {
	t.Run("PrefixForm_ReturnsCorrectKey", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{})
		fs.SetPrefix("items")

		cases := []struct {
			idx  int
			want string
		}{
			{0, "items-0"},
			{1, "items-1"},
			{2, "items-2"},
		}
		for _, c := range cases {
			if got := fs.PrefixForm(c.idx); got != c.want {
				t.Errorf("PrefixForm(%d) = %q, want %q", c.idx, got, c.want)
			}
		}
	})

	t.Run("PrefixedDataKeys_Validated", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{})
		fs.SetPrefix("items")
		fs.WithData(url.Values{
			"items-0-title": {"Alpha"},
			"items-1-title": {"Beta"},
			"items-2-title": {"Gamma"},
		}, nil, nil)
		if !forms.IsValid(ctx, fs) {
			t.Fatalf("expected valid with prefixed keys: %v", fs.ErrorList())
		}

		list := fs.CleanedDataList()
		wantTitles := []string{"Alpha", "Beta", "Gamma"}
		for i, m := range list {
			if m["title"] != wantTitles[i] {
				t.Errorf("form %d title = %q, want %q", i, m["title"], wantTitles[i])
			}
		}
	})

	t.Run("NoPrefixSet_NoFormsetPrefix", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{})
		if got := fs.Prefix(); got != "" {
			t.Errorf("Prefix() = %q, want empty string before SetPrefix", got)
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// DeleteForms callback
// ─────────────────────────────────────────────────────────────────────────────

func TestFormSet_DeleteFormsCallback(t *testing.T) {
	t.Run("CallbackReceivesDeletedForms", func(t *testing.T) {
		ctx := context.Background()
		var capturedDeleted []forms.Form
		fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{
			CanDelete: true,
			DeleteForms: func(c context.Context, deleted []forms.Form) error {
				capturedDeleted = deleted
				return nil
			},
		})
		// Delete all forms so Save() has no non-deleted forms to iterate over
		// (BaseForm.Save would panic without a Save method on each sub-form).
		fs.WithData(threeFormDeleteAllData, nil, nil)
		forms.IsValid(ctx, fs)

		if _, err := fs.Save(); err != nil {
			t.Fatalf("Save() error: %v", err)
		}
		if len(capturedDeleted) != 3 {
			t.Errorf("DeleteForms callback received %d forms, want 3", len(capturedDeleted))
		}
	})

	t.Run("CallbackNotCalledWhenNoneDeleted", func(t *testing.T) {
		ctx := context.Background()
		callbackCalled := false
		fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{
			CanDelete: true,
			DeleteForms: func(c context.Context, deleted []forms.Form) error {
				callbackCalled = true
				return nil
			},
		})
		// Submit data with no deletion flags.  We verify via DeletedForms() rather
		// than Save() because BaseForm lacks a Save() method and calling
		// formset.Save() with non-deleted forms in FormList would panic.
		fs.WithData(threeFormDeleteNoneData, nil, nil)
		forms.IsValid(ctx, fs)

		if len(fs.DeletedForms()) != 0 {
			t.Errorf("expected 0 deleted forms, got %d", len(fs.DeletedForms()))
		}
		if callbackCalled {
			t.Error("DeleteForms callback should not have been called when no form is deleted")
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Load
// ─────────────────────────────────────────────────────────────────────────────

func TestFormSet_Load(t *testing.T) {
	t.Run("PopulatesFormList", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{})
		fs.Load()
		flist, err := fs.Forms()
		if err != nil {
			t.Fatalf("Forms() error: %v", err)
		}
		if len(flist) != 3 {
			t.Errorf("expected 3 forms after Load(), got %d", len(flist))
		}
	})

	t.Run("IdempotentWithData", func(t *testing.T) {
		// If Load() is called after WithData + request set, it should be a no-op.
		// We verify that calling Load() after IsValid doesn't reset forms.
		ctx := context.Background()
		fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{})
		fs.WithData(threeFormAllChangedData, nil, nil)
		forms.IsValid(ctx, fs)
		lenBefore := func() int { l, _ := fs.Forms(); return len(l) }()
		// Load should not double-populate or reset
		// (it skips when formData is set but req is nil, as per the guard condition)
		fs.Load()
		lenAfter := func() int { l, _ := fs.Forms(); return len(l) }()
		if lenBefore != lenAfter {
			t.Errorf("Load() after IsValid changed form count: before=%d after=%d", lenBefore, lenAfter)
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// ForEach
// ─────────────────────────────────────────────────────────────────────────────

func TestFormSet_ForEach(t *testing.T) {
	t.Run("IteratesAllThreeForms", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{})
		fs.WithData(threeFormAllChangedData, nil, nil)
		forms.IsValid(ctx, fs)

		var indices []int
		if err := fs.ForEach(func(_ forms.Form, i int) error {
			indices = append(indices, i)
			return nil
		}); err != nil {
			t.Fatalf("ForEach error: %v", err)
		}
		if len(indices) != 3 {
			t.Fatalf("expected 3 iterations, got %d", len(indices))
		}
		for i, idx := range indices {
			if idx != i {
				t.Errorf("iteration %d got index %d, want %d", i, idx, i)
			}
		}
	})

	t.Run("StopsOnError", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{})
		fs.WithData(threeFormAllChangedData, nil, nil)
		forms.IsValid(ctx, fs)

		callCount := 0
		sentinel := fmt.Errorf("stop")
		err := fs.ForEach(func(_ forms.Form, _ int) error {
			callCount++
			return sentinel
		})
		if err != sentinel {
			t.Errorf("expected sentinel error, got %v", err)
		}
		if callCount != 1 {
			t.Errorf("ForEach should have stopped after first error, but ran %d times", callCount)
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Combined: deletion + ordering
// ─────────────────────────────────────────────────────────────────────────────

func TestFormSet_DeleteThenOrder(t *testing.T) {
	// Delete the form at slot 1, keep slots 0 and 2 with reversed ordering.
	//
	// Slot 0: order=1, title="Alpha"
	// Slot 1: __DELETE__=on, order=2 (deleted, still needs __ORDER__ to avoid tie)
	// Slot 2: order=0, title="Gamma"
	//
	// Expected after sorting non-deleted forms: [Gamma (order 0), Alpha (order 1)]
	t.Run("DeleteMiddle_ReverseRemainder", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{
			CanDelete: true,
			CanOrder:  true,
		})
		fs.WithData(url.Values{
			"0-" + formsets.ORDERING_FIELD_NAME: {"1"},
			"0-title":                           {"Alpha"},
			"1-" + formsets.DELETION_FIELD_NAME: {"on"},
			"1-" + formsets.ORDERING_FIELD_NAME: {"2"},
			"2-" + formsets.ORDERING_FIELD_NAME: {"0"},
			"2-title":                           {"Gamma"},
		}, nil, nil)
		if !forms.IsValid(ctx, fs) {
			t.Fatalf("expected valid: %v", fs.ErrorList())
		}
		if del := fs.DeletedForms(); len(del) != 1 {
			t.Errorf("expected 1 deleted form, got %d", len(del))
		}
		list := fs.CleanedDataList()
		if len(list) != 2 {
			t.Fatalf("expected 2 remaining forms, got %d", len(list))
		}
		if list[0]["title"] != "Gamma" {
			t.Errorf("form[0] title = %q, want Gamma", list[0]["title"])
		}
		if list[1]["title"] != "Alpha" {
			t.Errorf("form[1] title = %q, want Alpha", list[1]["title"])
		}
	})

	t.Run("DeleteFirst_OrderRemainder", func(t *testing.T) {
		ctx := context.Background()
		fs := newThreeFormSet(ctx, formsets.FormsetOptions[forms.Form]{
			CanDelete: true,
			CanOrder:  true,
		})
		// Delete slot 0; keep slot 1 (order 1) and slot 2 (order 0) → [Gamma, Beta]
		fs.WithData(url.Values{
			"0-" + formsets.DELETION_FIELD_NAME: {"on"},
			"0-" + formsets.ORDERING_FIELD_NAME: {"2"},
			"1-" + formsets.ORDERING_FIELD_NAME: {"1"},
			"1-title":                           {"Beta"},
			"2-" + formsets.ORDERING_FIELD_NAME: {"0"},
			"2-title":                           {"Gamma"},
		}, nil, nil)
		if !forms.IsValid(ctx, fs) {
			t.Fatalf("expected valid: %v", fs.ErrorList())
		}
		if del := fs.DeletedForms(); len(del) != 1 {
			t.Errorf("expected 1 deleted form, got %d", len(del))
		}
		list := fs.CleanedDataList()
		if len(list) != 2 {
			t.Fatalf("expected 2 remaining forms, got %d", len(list))
		}
		if list[0]["title"] != "Gamma" {
			t.Errorf("form[0] title = %q, want Gamma", list[0]["title"])
		}
		if list[1]["title"] != "Beta" {
			t.Errorf("form[1] title = %q, want Beta", list[1]["title"])
		}
	})
}
