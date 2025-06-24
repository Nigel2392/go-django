package queries_test

import (
	"reflect"
	"testing"
	_ "unsafe"

	"github.com/Nigel2392/go-django/queries/src/alias"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

//go:linkname newObjectFromIface github.com/Nigel2392/go-django/queries/internal.NewObjectFromIface
func newObjectFromIface(obj attrs.Definer) attrs.Definer

//go:linkname walkFields github.com/Nigel2392/go-django/queries/internal.WalkFields
func walkFields(
	m attrs.Definer,
	column string,
	aliasGen *alias.Generator,
) (
	definer attrs.Definer,
	parent attrs.Definer,
	f attrs.Field,
	chain []string,
	aliases []string,
	isRelated bool,
	err error,
)

//func TestNewObjectFromIface(t *testing.T) {
//	var obj = &Todo{
//		ID:          1,
//		Title:       "Test",
//		Description: "Test",
//		Done:        false,
//	}
//
//	var definer = newObjectFromIface(obj)
//	if definer == nil {
//		t.Fatal("newObjectFromIface returned nil")
//	}
//
//	if *(definer).(*Todo) != (Todo{}) {
//		t.Fatalf("newObjectFromIface returned wrong type: %T %+v", definer, definer)
//	}
//}

type walkFieldsExpected struct {
	column    string
	definer   attrs.Definer
	parent    attrs.Definer
	field     func() attrs.Field
	chain     []string
	aliases   []string
	isRelated bool
	err       error
}

type walkFieldsTest struct {
	name     string
	model    attrs.Definer
	expected []walkFieldsExpected
}

func getField(m attrs.Definer, field string) func() attrs.Field {
	return func() attrs.Field {
		defs := m.FieldDefs()
		f, _ := defs.Field(field)
		return f
	}
}

func fieldEquals(f1, f2 attrs.Field) bool {

	var (
		instance1 = f1.Instance()
		name1     = f1.Name()
		instance2 = f2.Instance()
		name2     = f2.Name()
	)

	return reflect.TypeOf(instance1) == reflect.TypeOf(instance2) && name1 == name2
}

var walkFieldsTests = []walkFieldsTest{
	{
		name:  "TestTodoID",
		model: &Todo{},
		expected: []walkFieldsExpected{{
			column:    "ID",
			definer:   &Todo{},
			parent:    nil,
			field:     getField(&Todo{}, "ID"),
			chain:     []string{},
			aliases:   []string{},
			isRelated: false,
			err:       nil,
		}},
	},
	{
		name:  "TestTodoUser",
		model: &Todo{},
		expected: []walkFieldsExpected{{
			column:    "User",
			definer:   &Todo{},
			parent:    nil,
			field:     getField(&Todo{}, "User"),
			chain:     []string{},
			aliases:   []string{},
			isRelated: false,
			err:       nil,
		}},
	},
	{
		name:  "TestTodoUserWithID",
		model: &Todo{},
		expected: []walkFieldsExpected{{
			column:    "User.ID",
			definer:   &User{},
			parent:    &Todo{},
			field:     getField(&User{}, "ID"),
			chain:     []string{"User"},
			aliases:   []string{"T_queries-users"},
			isRelated: true,
			err:       nil,
		}},
	},
	{
		name:  "TestObjectWithMultipleRelationsID1",
		model: &ObjectWithMultipleRelations{},
		expected: []walkFieldsExpected{
			{
				column:    "Obj1.ID",
				definer:   &User{},
				parent:    &ObjectWithMultipleRelations{},
				field:     getField(&User{}, "ID"),
				chain:     []string{"Obj1"},
				aliases:   []string{"T_queries-users"},
				isRelated: true,
				err:       nil,
			},
			{
				column:    "Obj2.ID",
				definer:   &User{},
				parent:    &ObjectWithMultipleRelations{},
				field:     getField(&User{}, "ID"),
				chain:     []string{"Obj2"},
				aliases:   []string{"T1_queries-users"},
				isRelated: true,
				err:       nil,
			},
		},
	},
	{
		name:  "TestNestedCategoriesParent",
		model: &Category{},
		expected: []walkFieldsExpected{{
			column:    "Parent.Parent",
			definer:   &Category{},
			parent:    &Category{},
			field:     getField(&Category{}, "Parent"),
			chain:     []string{"Parent"},
			aliases:   []string{"T_queries-categories"},
			isRelated: true,
			err:       nil,
		}},
	},
	{
		name:  "TestNestedCategoriesName",
		model: &Category{},
		expected: []walkFieldsExpected{{
			column:    "Parent.Parent.Name",
			definer:   &Category{},
			parent:    &Category{},
			field:     getField(&Category{}, "Name"),
			chain:     []string{"Parent", "Parent"},
			aliases:   []string{"T_queries-categories", "T1_queries-categories"},
			isRelated: true,
			err:       nil,
		}},
	},
}

func TestWalkFields(t *testing.T) {
	for _, test := range walkFieldsTests {
		t.Run(test.name, func(t *testing.T) {
			var aliasGen = alias.NewGenerator()

			for _, expected := range test.expected {
				var (
					definer, parent, field, chain, aliases, isRelated, err = walkFields(test.model, expected.column, aliasGen)
				)

				if reflect.TypeOf(definer) != reflect.TypeOf(expected.definer) {
					t.Errorf("expected definer %T, got %T", expected.definer, definer)
				}

				if expected.parent != nil {
					if reflect.TypeOf(parent) != reflect.TypeOf(expected.parent) {
						t.Errorf("expected parent %T, got %T", expected.parent, parent)
					}
				}

				if expected.parent == nil && parent != nil {
					t.Errorf("expected parent nil, got %T", parent)
				}

				var expectedField = expected.field()
				if !fieldEquals(field, expectedField) {
					t.Errorf("expected field %T.%s, got %T.%s", expectedField.Instance(), expectedField.Name(), field.Instance(), field.Name())
				}

				if len(chain) != len(expected.chain) {
					t.Errorf("expected chain length %d, got %d", len(expected.chain), len(chain))
				} else {
					for i := range chain {
						if chain[i] != expected.chain[i] {
							t.Errorf("expected chain %s, got %s", expected.chain[i], chain[i])
						}
					}
				}

				if len(aliases) != len(expected.aliases) {
					t.Errorf("expected aliases length %d, got %d", len(expected.aliases), len(aliases))
				} else {
					for i := range aliases {
						if aliases[i] != expected.aliases[i] {
							t.Errorf("expected alias %s, got %s", expected.aliases[i], aliases[i])
						}
					}
				}

				if isRelated != expected.isRelated {
					t.Errorf("expected isRelated %v, got %v", expected.isRelated, isRelated)
				}

				if err != nil && err.Error() != expected.err.Error() {
					t.Errorf("expected error %v, got %v", expected.err.Error(), err.Error())
				}
			}

		})
	}
}
