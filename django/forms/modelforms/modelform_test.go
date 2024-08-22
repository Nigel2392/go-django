package modelforms_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/forms/modelforms"
)

var _ (attrs.Definer) = (*TestModel)(nil)

type TestModel struct {
	ID   int
	Name string
	Age  int
}

func (m *TestModel) FieldDefs() attrs.Definitions {
	return attrs.Define(
		m,
		attrs.NewField(m, "ID", &attrs.FieldConfig{Null: true, Blank: true, ReadOnly: false}),
		attrs.NewField(m, "Name", &attrs.FieldConfig{Null: true, Blank: true, ReadOnly: false}),
		attrs.NewField(m, "Age", &attrs.FieldConfig{Null: true, Blank: true, ReadOnly: false, Validators: []func(interface{}) error{
			func(v interface{}) error {
				if age, ok := v.(int); ok {
					if age <= 0 {
						return errors.New("Age must be greater than 0")
					}
					return nil
				}
				return errors.New("Invalid type")
			},
		}}),
	)
}

func TestModelForm(t *testing.T) {
	var m = &TestModel{
		ID:   1,
		Name: "name",
		Age:  0,
	}

	var f = modelforms.NewBaseModelForm(m)

	t.Run("InitNilModel", func(t *testing.T) {
		var f = modelforms.NewBaseModelForm[*TestModel](nil)

		if f.Model == nil {
			t.Errorf("expected %v, got %v", &TestModel{}, f.Model)
		}

		t.Run("LoadForm", func(t *testing.T) {
			f.Load()

			if f.FormFields.Len() != 3 {
				t.Errorf("expected %d, got %d", 3, f.FormFields.Len())
			}

			if !reflect.DeepEqual(f.BaseForm.Initial, map[string]interface{}{
				"ID":   0,
				"Name": "",
				"Age":  0,
			}) {
				t.Errorf("expected %v, got %v", nil, f.BaseForm.Initial)
			}
		})
	})

	t.Run("AutoFields", func(t *testing.T) {
		if len(f.ModelFields) != 3 {
			t.Errorf("expected %d, got %d", 3, len(f.ModelFields))
		}

		if len(f.ModelExclude) != 0 {
			t.Errorf("expected %d, got %d", 0, len(f.ModelExclude))
		}

		if len(f.InstanceFields) != 3 {
			t.Errorf("expected %d, got %d", 2, len(f.InstanceFields))
		}

		if f.InstanceFields[0].Name() != "ID" {
			t.Errorf("expected %q, got %q", "ID", f.InstanceFields[0].Name())
		}

		if f.InstanceFields[1].Name() != "Name" {
			t.Errorf("expected %q, got %q", "Name", f.InstanceFields[1].Name())
		}

		if f.InstanceFields[2].Name() != "Age" {
			t.Errorf("expected %q, got %q", "Age", f.InstanceFields[2].Name())
		}

		f = modelforms.NewBaseModelForm(m)

		t.Run("LoadForm", func(t *testing.T) {
			f.Load()

			if f.FormFields.Len() != 3 {
				t.Errorf("expected %d, got %d", 3, f.FormFields.Len())
			}

			if f, ok := f.FormFields.Get("ID"); !ok {
				t.Errorf("expected %t, got %t", true, ok)
			} else {
				if f.Name() != "ID" {
					t.Errorf("expected %q, got %q", "ID", f.Name())
				}
			}

			if f, ok := f.FormFields.Get("Name"); !ok {
				t.Errorf("expected %t, got %t", true, ok)
			} else {
				if f.Name() != "Name" {
					t.Errorf("expected %q, got %q", "Name", f.Name())
				}
			}

			if f, ok := f.FormFields.Get("Age"); !ok {
				t.Errorf("expected %t, got %t", true, ok)
			} else {
				if f.Name() != "Age" {
					t.Errorf("expected %q, got %q", "Age", f.Name())
				}
			}

			if f.BaseForm.Initial["ID"] != 1 {
				t.Errorf("expected %d, got %d", 1, f.BaseForm.Initial["ID"])
			}

			if f.BaseForm.Initial["Name"] != "name" {
				t.Errorf("expected %q, got %q", "name", f.BaseForm.Initial["Name"])
			}

			if f.BaseForm.Initial["Age"] != 0 {
				t.Errorf("expected %d, got %d", 0, f.BaseForm.Initial["Age"])
			}
		})

		f = modelforms.NewBaseModelForm(m)

		t.Run("SaveForm", func(t *testing.T) {
			f.Load()

			f.BaseForm.Cleaned = map[string]interface{}{
				"ID":   2,
				"Name": "new name",
				"Age":  1,
			}

			if err := f.Save(); err != nil {
				t.Errorf("expected %v, got %v", nil, err)
			}

			if m.ID != 2 {
				t.Errorf("expected %d, got %d", 2, m.ID)
			}

			if m.Name != "new name" {
				t.Errorf("expected %q, got %q", "new name", m.Name)
			}

			if m.Age != 1 {
				t.Errorf("expected %d, got %d", 1, m.Age)
			}
		})

		m.ID = 1
		m.Name = "name"

		f = modelforms.NewBaseModelForm(m)

		t.Run("ExcludeFields", func(t *testing.T) {
			f.SetExclude("ID")

			f.Load()

			if f.FormFields.Len() != 2 {
				t.Errorf("expected length %d, got %d", 1, f.FormFields.Len())
				for head := f.FormFields.Front(); head != nil; head = head.Next() {
					t.Logf("field: %q", head.Value.Name())
				}
			}

			if f, ok := f.FormFields.Get("Name"); !ok {
				t.Errorf("expected %t, got %t", true, ok)
			} else {
				if f.Name() != "Name" {
					t.Errorf("expected %q, got %q", "Name", f.Name())
				}
			}

			if f, ok := f.FormFields.Get("Age"); !ok {
				t.Errorf("expected %t, got %t", true, ok)
			} else {
				if f.Name() != "Age" {
					t.Errorf("expected %q, got %q", "Age", f.Name())
				}
			}

			if f.BaseForm.Initial["ID"] != nil {
				t.Errorf("expected %v, got %v", nil, f.BaseForm.Initial["ID"])
			}

			if f.BaseForm.Initial["Name"] != "name" {
				t.Errorf("expected %q, got %q", "name", f.BaseForm.Initial["Name"])
			}

			if f.BaseForm.Initial["Age"] != 1 {
				t.Errorf("expected %d, got %d", 1, f.BaseForm.Initial["Age"])
			}

			t.Run("SaveForm", func(t *testing.T) {
				f.BaseForm.Cleaned = map[string]interface{}{
					"ID":   2,
					"Name": "new name",
					"Age":  1,
				}

				if err := f.Save(); err != nil {
					t.Errorf("expected (err) %v, got %v", nil, err)
				}

				if m.ID != 1 {
					t.Errorf("expected (ID) %v, got %v", 1, m.ID)
				}

				if m.Name != "new name" {
					t.Errorf("expected (Name) %q, got %q", "new name", m.Name)
				}

				if m.Age != 1 {
					t.Errorf("expected (Age) %v, got %v", 1, m.Age)
				}
			})

			t.Run("SaveFormFail", func(t *testing.T) {

				f.BaseForm.Cleaned = map[string]interface{}{
					"ID":   2,
					"Name": "new name",
					"Age":  -1,
				}

				if err := f.Save(); err == nil {
					t.Errorf("expected (err) %v, got %v", errors.New("Age must be greater than 0"), err)
				}

				if m.ID != 1 {
					t.Errorf("expected (ID) %v, got %v", 1, m.ID)
				}

				if m.Name != "new name" {
					t.Errorf("expected (Name) %q, got %q", "new name", m.Name)
				}

				if m.Age != -1 {
					t.Errorf("expected (Age) %v, got %v", -1, m.Age)
				}
			})
		})
	})
}
