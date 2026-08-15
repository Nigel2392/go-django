package benchmarks_test

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/Nigel2392/go-django/djester"
	"github.com/Nigel2392/go-django/djester/objects"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/fields/formfields"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/forms/fields"
)

func TestManyToManyFormField(t *testing.T) {

	var firstM2m, err = queries.GetQuerySet(&BenchmarkM2MSource{}).First()
	if err != nil {
		t.Fatalf("Error while fetching row: %v", err)
	}

	var defs = attrs.Define(t.Context(), firstM2m.Object)
	var fld, _ = defs.Field("Target")

	selectedTargets, err := firstM2m.Object.Target.Objects().All()
	if err != nil {
		t.Fatalf("Error while fetching related rows: %v", err)
	}

	exclSelectedTargets, err := queries.GetQuerySet(&BenchmarkM2MTarget{}).
		Filter(expr.Q("ID__in", firstM2m.Object.Target.Objects().Select("ID")).Not(true)).
		All()
	if err != nil {
		t.Fatalf("Error while fetching non-related rows: %v", err)
	}

	var exclMap = make(map[uint64]struct{})
	var selectedTargetIds = make([]string, 0, len(selectedTargets))
	var exclSelectedTargetIds = make([]string, 0, len(exclSelectedTargets))
	for _, t := range selectedTargets {
		selectedTargetIds = append(selectedTargetIds, strconv.FormatUint(uint64(t.Object.ID), 10))
	}
	for _, t := range exclSelectedTargets {
		exclSelectedTargetIds = append(exclSelectedTargetIds, strconv.FormatUint(uint64(t.Object.ID), 10))
		exclMap[uint64(t.Object.ID)] = struct{}{}
	}

	var FIELD_TEST_1 = &objects.FieldTest[*formfields.ManyToManyFormField]{
		Label: "TestManyToManyFieldHTML",
		FormField: &formfields.ManyToManyFormField{
			BaseRelationField: formfields.BaseRelationField{
				BaseField: fields.NewField(),
				Field:     fld,
				Relation:  fld.Rel(),
			},
		},
		ExpectedHTML: func(d *djester.Tester, t *testing.T, ft *objects.FieldTest[*formfields.ManyToManyFormField], fieldName string) []djester.HTMLAssertFunc {
			return []djester.HTMLAssertFunc{
				objects.AssertInputValueMatches(fieldName, selectedTargetIds...),
			}
		},
	}
	var FIELD_TEST_2 = &objects.FieldTest[*formfields.ManyToManyFormField]{
		Label:             "TestManyToManyFieldValue",
		FieldNameOverride: "TestManyToManyFieldValue",
		FormField: &formfields.ManyToManyFormField{
			BaseRelationField: formfields.BaseRelationField{
				BaseField: fields.NewField(),
				Field:     fld,
				Relation:  fld.Rel(),
			},
		},
		FormData: url.Values{
			"TestManyToManyFieldValue": exclSelectedTargetIds,
		},
		ExpectsValid: true,
		Handle: []func(d *djester.Tester, t *testing.T, ft *objects.FieldTest[*formfields.ManyToManyFormField], field *formfields.ManyToManyFormField, initial_data, cleaned_data any, errors []error){
			func(d *djester.Tester, t *testing.T, ft *objects.FieldTest[*formfields.ManyToManyFormField], field *formfields.ManyToManyFormField, initial_data, cleaned_data any, errors []error) {
				c := cleaned_data.([]attrs.Definer)
				for _, d := range c {
					p := attrs.PrimaryKey(t.Context(), d)
					if _, ok := exclMap[p.(uint64)]; !ok {
						t.Errorf("Expected to find id %T %v in %v", p, p, exclMap)
					}
				}
			},
		},
	}
	var FIELD_TEST_3 = &objects.FieldTest[*formfields.ManyToManyFormField]{
		Label:             "TestManyToManyFieldClearValue",
		FieldNameOverride: "TestManyToManyFieldClearValue",
		FormField: &formfields.ManyToManyFormField{
			BaseRelationField: formfields.BaseRelationField{
				BaseField: fields.NewField(),
				Field:     fld,
				Relation:  fld.Rel(),
			},
		},
		FormData: url.Values{
			"TestManyToManyFieldClearValue": []string{},
		},
		ExpectsValid: true,
		Handle: []func(d *djester.Tester, t *testing.T, ft *objects.FieldTest[*formfields.ManyToManyFormField], field *formfields.ManyToManyFormField, initial_data, cleaned_data any, errors []error){
			func(d *djester.Tester, t *testing.T, ft *objects.FieldTest[*formfields.ManyToManyFormField], field *formfields.ManyToManyFormField, initial_data, cleaned_data any, errors []error) {
				if cleaned_data == nil {
					return // all good
				}
				c := cleaned_data.([]attrs.Definer)
				if len(c) > 0 {
					t.Errorf("Expected relations to be cleared, found %v", len(c))
				}
			},
		},
	}

	t.Run(FIELD_TEST_1.Label, func(t *testing.T) {
		FIELD_TEST_1.Test(&djester.Tester{}, t)
	})
	t.Run(FIELD_TEST_2.Label, func(t *testing.T) {
		FIELD_TEST_2.Test(&djester.Tester{}, t)
	})
	t.Run(FIELD_TEST_3.Label, func(t *testing.T) {
		FIELD_TEST_3.Test(&djester.Tester{}, t)
	})

}

func TestForeignKeyFormField(t *testing.T) {

	var firstBook, err = queries.GetQuerySet(&BenchmarkBook{}).First()
	if err != nil {
		t.Fatalf("Error while fetching row: %v", err)
	}

	var defs = attrs.Define(t.Context(), firstBook.Object)
	var fld, _ = defs.Field("Author")

	exclSelectedTarget, err := queries.GetQuerySet(&BenchmarkAuthor{}).
		Filter(expr.Q("ID", firstBook.Object.Author.ID).Not(true)).
		First()
	if err != nil {
		t.Fatalf("Error while fetching non-related rows: %v", err)
	}

	var FIELD_TEST_1 = &objects.FieldTest[*formfields.ForeignKeyFormField]{
		Label: "TestForeignKeyFieldHTML",
		FormField: &formfields.ForeignKeyFormField{
			BaseRelationField: formfields.BaseRelationField{
				BaseField: fields.NewField(),
				Field:     fld,
				Relation:  fld.Rel(),
			},
		},
		Default: firstBook.Object.Author,
		ExpectedHTML: func(d *djester.Tester, t *testing.T, ft *objects.FieldTest[*formfields.ForeignKeyFormField], fieldName string) []djester.HTMLAssertFunc {
			return []djester.HTMLAssertFunc{
				objects.AssertInputValueMatches(fieldName, strconv.FormatUint(firstBook.Object.Author.ID, 10)),
			}
		},
	}
	var FIELD_TEST_2 = &objects.FieldTest[*formfields.ForeignKeyFormField]{
		Label:             "TestForeignKeyFieldValue",
		FieldNameOverride: "TestForeignKeyFieldValue",
		FormField: &formfields.ForeignKeyFormField{
			BaseRelationField: formfields.BaseRelationField{
				BaseField: fields.NewField(),
				Field:     fld,
				Relation:  fld.Rel(),
			},
		},
		FormData: url.Values{
			"TestForeignKeyFieldValue": []string{strconv.FormatUint(uint64(exclSelectedTarget.Object.ID), 10)},
		},
		ExpectsValid: true,
		Handle: []func(d *djester.Tester, t *testing.T, ft *objects.FieldTest[*formfields.ForeignKeyFormField], field *formfields.ForeignKeyFormField, initial_data, cleaned_data any, errors []error){
			func(d *djester.Tester, t *testing.T, ft *objects.FieldTest[*formfields.ForeignKeyFormField], field *formfields.ForeignKeyFormField, initial_data, cleaned_data any, errors []error) {
				p := attrs.PrimaryKey(t.Context(), cleaned_data)
				if p.(uint64) != uint64(exclSelectedTarget.Object.ID) {
					t.Errorf("Expected to find id %T %v != %v", p, p, uint64(exclSelectedTarget.Object.ID))
				}
			},
		},
	}
	var FIELD_TEST_3 = &objects.FieldTest[*formfields.ForeignKeyFormField]{
		Label:             "TestForeignKeyFieldClearValue",
		FieldNameOverride: "TestForeignKeyFieldClearValue",
		FormField: &formfields.ForeignKeyFormField{
			BaseRelationField: formfields.BaseRelationField{
				BaseField: fields.NewField(),
				Field:     fld,
				Relation:  fld.Rel(),
			},
		},
		FormData: url.Values{
			"TestForeignKeyFieldClearValue": []string{},
		},
		ExpectsValid: true,
		Handle: []func(d *djester.Tester, t *testing.T, ft *objects.FieldTest[*formfields.ForeignKeyFormField], field *formfields.ForeignKeyFormField, initial_data, cleaned_data any, errors []error){
			func(d *djester.Tester, t *testing.T, ft *objects.FieldTest[*formfields.ForeignKeyFormField], field *formfields.ForeignKeyFormField, initial_data, cleaned_data any, errors []error) {
				if len(errors) > 0 {
					t.Fatal(errors)
				}
				if cleaned_data == nil {
					return // all good
				}
				c := cleaned_data.(attrs.Definer)
				if c != nil {
					t.Errorf("Expected relations to be cleared, found %v", c)
				}
			},
		},
	}

	t.Run(FIELD_TEST_1.Label, func(t *testing.T) {
		FIELD_TEST_1.Test(&djester.Tester{}, t)
	})
	t.Run(FIELD_TEST_2.Label, func(t *testing.T) {
		FIELD_TEST_2.Test(&djester.Tester{}, t)
	})
	t.Run(FIELD_TEST_3.Label, func(t *testing.T) {
		FIELD_TEST_3.Test(&djester.Tester{}, t)
	})

}
