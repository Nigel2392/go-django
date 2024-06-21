package contenttypes_test

import (
	"fmt"
	"testing"

	"github.com/Nigel2392/django/core/contenttypes"
)

type TestStructOne struct {
	ID   int
	Name string
}

type TestStructTwo struct {
	ID   int
	Name string
}

type TestStructThree struct {
	ID   int
	Name string
}

func TestContentType(t *testing.T) {
	var (
		aliasOne = []string{
			"contenttypes.TestStructOne",
			"test.TestStructOne",
		}
		aliasTwo = []string{
			"contenttypes.TestStructTwo",
			"test.TestStructTwo",
		}
		aliasThree = []string{
			"contenttypes.TestStructThree",
			"test.TestStructThree",
		}
		defOne = &contenttypes.ContentTypeDefinition{
			ContentObject: &TestStructOne{},
			GetLabel: func() string {
				return "test struct one"
			},
			Aliases: aliasOne,
			GetObject: func() interface{} {
				return &TestStructOne{
					ID:   1,
					Name: "name",
				}
			},
		}
		defTwo = &contenttypes.ContentTypeDefinition{
			ContentObject: TestStructTwo{},
			GetLabel: func() string {
				return "test struct two"
			},
			Aliases: aliasTwo,
			GetObject: func() interface{} {
				return TestStructTwo{
					ID:   2,
					Name: "name",
				}
			},
		}
		defThree = &contenttypes.ContentTypeDefinition{
			ContentObject: (*TestStructThree)(nil),
			GetLabel: func() string {
				return "test struct three"
			},
			Aliases: aliasThree,
			GetObject: func() interface{} {
				return &TestStructThree{
					ID:   3,
					Name: "name",
				}
			},
		}
	)

	var (
		ctOne   = contenttypes.NewContentType(&TestStructOne{})
		ctTwo   = contenttypes.NewContentType(TestStructTwo{})
		ctThree = contenttypes.NewContentType((*TestStructThree)(nil))
	)

	t.Run("TestAppLabel", func(t *testing.T) {
		if ctOne.AppLabel() != "contenttypes_test" {
			t.Errorf("expected %q, got %q", "contenttypes_test", ctOne.PkgPath())
		}

		if ctTwo.AppLabel() != "contenttypes_test" {
			t.Errorf("expected %q, got %q", "contenttypes_test", ctTwo.PkgPath())
		}

		if ctThree.AppLabel() != "contenttypes_test" {
			t.Errorf("expected %q, got %q", "contenttypes_test", ctThree.PkgPath())
		}
	})

	t.Run("TestTypeName", func(t *testing.T) {
		if ctOne.Model() != "TestStructOne" {
			t.Errorf("expected %q, got %q", "TestStructOne", ctOne.Model())
		}

		if ctTwo.Model() != "TestStructTwo" {
			t.Errorf("expected %q, got %q", "TestStructTwo", ctTwo.Model())
		}

		if ctThree.Model() != "TestStructThree" {
			t.Errorf("expected %q, got %q", "TestStructThree", ctThree.Model())
		}
	})

	contenttypes.Register(defOne)
	contenttypes.Register(defTwo)
	contenttypes.Register(defThree)

	t.Run("TestScan", func(t *testing.T) {

		var contentTypeNames [][3]string = [][3]string{
			{
				"github.com/Nigel2392/django/core/contenttypes_test.TestStructOne",
				"github.com/Nigel2392/django/core/contenttypes_test.TestStructTwo",
				"github.com/Nigel2392/django/core/contenttypes_test.TestStructThree",
			},
			{"contenttypes_test.TestStructOne", "contenttypes_test.TestStructTwo", "contenttypes_test.TestStructThree"},
			{"contenttypes.TestStructOne", "contenttypes.TestStructTwo", "contenttypes.TestStructThree"},
			{"test.TestStructOne", "test.TestStructTwo", "test.TestStructThree"},
		}

		for _, typNames := range contentTypeNames {

			t.Run(fmt.Sprintf("TestScan_%s", typNames[0]), func(t *testing.T) {
				var (
					typnameOne   = typNames[0]
					typnameTwo   = typNames[1]
					typnameThree = typNames[2]
				)

				var ct = contenttypes.BaseContentType[any]{}
				err := ct.Scan(typnameOne)
				if err != nil {
					t.Errorf("expected nil error, got %v", err)
				}

				if ct.PkgPath() != "github.com/Nigel2392/django/core/contenttypes_test" {
					t.Errorf("expected %q, got %q", "github.com/Nigel2392/django/core/contenttypes_test", ct.PkgPath())
				}

				if ct.Model() != "TestStructOne" {
					t.Errorf("expected %q, got %q", "TestStructOne", ct.Model())
				}

				err = ct.Scan(typnameTwo)
				if err != nil {
					t.Errorf("expected nil error, got %v", err)
				}

				if ct.PkgPath() != "github.com/Nigel2392/django/core/contenttypes_test" {
					t.Errorf("expected %q, got %q", "github.com/Nigel2392/django/core/contenttypes_test", ct.PkgPath())
				}

				if ct.Model() != "TestStructTwo" {
					t.Errorf("expected %q, got %q", "TestStructTwo", ct.Model())
				}

				err = ct.Scan(typnameThree)
				if err != nil {
					t.Errorf("expected nil error, got %v", err)
				}

				if ct.PkgPath() != "github.com/Nigel2392/django/core/contenttypes_test" {
					t.Errorf("expected %q, got %q", "github.com/Nigel2392/django/core/contenttypes_test", ct.PkgPath())
				}

				if ct.Model() != "TestStructThree" {
					t.Errorf("expected %q, got %q", "TestStructThree", ct.Model())
				}
			})
		}
	})

	t.Run("TestRegister", func(t *testing.T) {

		t.Run("TestGetContentTypeDefinitionForObject", func(t *testing.T) {
			def := contenttypes.DefinitionForObject(&TestStructOne{})
			if def != defOne {
				t.Errorf("expected %v, got %v", defOne, def)
			}

			def = contenttypes.DefinitionForObject(TestStructTwo{})
			if def != defTwo {
				t.Errorf("expected %v, got %v", defTwo, def)
			}

			def = contenttypes.DefinitionForObject((*TestStructThree)(nil))
			if def != defThree {
				t.Errorf("expected %v, got %v", defThree, def)
			}
		})

		t.Run("TestGetContentTypeDefinition", func(t *testing.T) {
			def := contenttypes.DefinitionForPackage("contenttypes_test", "TestStructOne")
			if def != defOne {
				t.Errorf("expected %v, got %v", defOne, def)
			}

			def = contenttypes.DefinitionForPackage("contenttypes_test", "TestStructTwo")
			if def != defTwo {
				t.Errorf("expected %v, got %v", defTwo, def)
			}

			def = contenttypes.DefinitionForPackage("contenttypes_test", "TestStructThree")
			if def != defThree {
				t.Errorf("expected %v, got %v", defThree, def)
			}
		})

		t.Run("TestGetContentTypeDefinitionForAlias", func(t *testing.T) {
			def := contenttypes.DefinitionForPackage("contenttypes", "TestStructOne")
			if def != defOne {
				t.Errorf("expected %v, got %v", defOne, def)
			}

			def = contenttypes.DefinitionForPackage("contenttypes", "TestStructTwo")
			if def != defTwo {
				t.Errorf("expected %v, got %v", defTwo, def)
			}

			def = contenttypes.DefinitionForPackage("contenttypes", "TestStructThree")
			if def != defThree {
				t.Errorf("expected %v, got %v", defThree, def)
			}

			def = contenttypes.DefinitionForPackage("test", "TestStructOne")
			if def != defOne {
				t.Errorf("expected %v, got %v", defOne, def)
			}

			def = contenttypes.DefinitionForPackage("test", "TestStructTwo")
			if def != defTwo {
				t.Errorf("expected %v, got %v", defTwo, def)
			}

			def = contenttypes.DefinitionForPackage("test", "TestStructThree")
			if def != defThree {
				t.Errorf("expected %v, got %v", defThree, def)
			}
		})

		t.Run("TestGetContentTypeDefinitionGetObjects", func(t *testing.T) {
			var obj any
			obj = defOne.Object()
			if obj == nil {
				t.Errorf("expected non-nil object")
			}

			if obj.(*TestStructOne).ID != 1 {
				t.Errorf("expected %d, got %d", 1, obj.(*TestStructOne).ID)
			}

			if obj.(*TestStructOne).Name != "name" {
				t.Errorf("expected %q, got %q", "name", obj.(*TestStructOne).Name)
			}

			obj = defTwo.Object()
			if obj == nil {
				t.Errorf("expected non-nil object")
			}

			if obj.(TestStructTwo).ID != 2 {
				t.Errorf("expected %d, got %d", 2, obj.(TestStructTwo).ID)
			}

			if obj.(TestStructTwo).Name != "name" {
				t.Errorf("expected %q, got %q", "name", obj.(TestStructTwo).Name)
			}

			obj = defThree.Object()
			if obj == nil {
				t.Errorf("expected non-nil object")
			}

			if obj.(*TestStructThree).ID != 3 {
				t.Errorf("expected %d, got %d", 3, obj.(*TestStructThree).ID)
			}

			if obj.(*TestStructThree).Name != "name" {
				t.Errorf("expected %q, got %q", "name", obj.(*TestStructThree).Name)
			}
		})
	})
}
