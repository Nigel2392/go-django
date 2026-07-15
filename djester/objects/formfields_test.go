package objects_test

import (
	"testing"

	"github.com/Nigel2392/go-django/djester"
	"github.com/Nigel2392/go-django/djester/objects"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/forms/fields"
)

func TestDjester(t *testing.T) {
	d := &djester.Tester{
		Settings: map[string]any{},
		Flags: []django.AppFlag{
			django.FlagSkipCmds,
			django.FlagSkipChecks,
		},
		Apps: []djester.AppInitFuncOrAppConfig{},
		Tests: []djester.Test{
			&objects.FieldTest[*fields.BaseField]{
				Label:   "TestObjectFields1",
				Default: "test_object_fields_1_default_value",
				FormField: fields.CharField(
					fields.MinLength(8),
					fields.MaxLength(255),
				),
				ExpectedHTML: func(d *djester.Tester, t *testing.T, ft *objects.FieldTest[*fields.BaseField], fieldName string) []djester.HTMLAssertFunc {
					return []djester.HTMLAssertFunc{
						objects.AssertInputValueMatches(fieldName, ft.Default.(string)),
						objects.AssertInputAttributes(fieldName,
							objects.AttributeValueEQ("name", fieldName),
							objects.AttributeValueEQ("type", "text"),
							objects.AttributeValueEQ("maxlength", "255"),
							objects.AttributeValueEQ("minlength", "8"),
						),
					}
				},
			},
			&objects.FieldTest[*fields.BaseField]{
				Label:   "TestObjectFields2",
				Default: "test_object_fields_1_default_value",
				FormField: fields.CharField(
					fields.MinLength(8),
					fields.MaxLength(255),
				),
				FormData:     "this is really valid", // len == 7 but min = 8
				ExpectsValid: true,
				ExpectedHTML: func(d *djester.Tester, t *testing.T, ft *objects.FieldTest[*fields.BaseField], fieldName string) []djester.HTMLAssertFunc {
					return []djester.HTMLAssertFunc{
						objects.AssertInputValueMatches(fieldName, ft.FormData.(string)),
						objects.AssertInputAttributes(fieldName,
							objects.AttributeValueEQ("name", fieldName),
							objects.AttributeValueEQ("type", "text"),
							objects.AttributeValueEQ("maxlength", "255"),
							objects.AttributeValueEQ("minlength", "8"),
						),
					}
				},
				Handle: []func(d *djester.Tester, t *testing.T, ft *objects.FieldTest[*fields.BaseField], field *fields.BaseField, initial_data, cleaned_data any, errors []error){
					func(d *djester.Tester, t *testing.T, ft *objects.FieldTest[*fields.BaseField], field *fields.BaseField, initial_data, cleaned_data any, errors []error) {
						if cleaned_data != ft.FormData {
							t.Errorf("expected %q, got %q", ft.FormData, cleaned_data)
						}
					},
				},
			},
			&objects.FieldTest[*fields.BaseField]{
				Label:   "TestObjectFields3",
				Default: "test_object_fields_1_default_value",
				FormField: fields.CharField(
					fields.MinLength(8),
					fields.MaxLength(255),
				),
				ExpectsValid: false,
				FormData:     "invalid", // len == 7 but min = 8
				ExpectedErrors: []error{
					errs.ErrLengthMin,
				},
				ExpectedHTML: func(d *djester.Tester, t *testing.T, ft *objects.FieldTest[*fields.BaseField], fieldName string) []djester.HTMLAssertFunc {
					return []djester.HTMLAssertFunc{
						objects.AssertInputAttributes(fieldName,
							objects.AttributeValueEQ("name", fieldName),
							objects.AttributeValueEQ("type", "text"),
							objects.AttributeValueEQ("maxlength", "255"),
							objects.AttributeValueEQ("minlength", "8"),
						),
					}
				},
			},
			&objects.FieldTest[*fields.BaseField]{
				Label: "TestObjectFields4",
				FormField: fields.CharField(
					fields.Required(true),
				),
				ExpectsValid: false,
				FormData:     "",
				ExpectedErrors: []error{
					errs.ErrFieldRequired,
				},
			},
			&objects.FieldTest[*fields.BaseField]{
				Label:        "TestObjectFields5",
				FormField:    fields.NumberField[int](),
				ExpectsValid: true,
				FormData:     "1023",
			},
			&objects.FieldTest[*fields.BaseField]{
				Label:        "TestObjectFields6",
				FormField:    fields.NumberField[float32](),
				ExpectsValid: true,
				FormData:     "1023.24",
				Handle: []func(d *djester.Tester, t *testing.T, ft *objects.FieldTest[*fields.BaseField], field *fields.BaseField, initial_data, cleaned_data any, errors []error){
					func(d *djester.Tester, t *testing.T, ft *objects.FieldTest[*fields.BaseField], field *fields.BaseField, initial_data, cleaned_data any, errors []error) {
						if cleaned_data != float32(1023.24) {
							t.Errorf("expected 1023.24, got %v %T", cleaned_data, cleaned_data)
						}
					},
				},
			},
			&objects.FieldTest[*fields.BaseField]{
				Label:        "TestObjectFields7",
				FormField:    fields.NumberField[int](),
				ExpectsValid: false,
				FormData:     "abc",
				ExpectedErrors: []error{
					errs.ErrInvalidSyntax,
				},
			},
		},
	}

	d.Test(t)
}
