package contenttypes_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/Nigel2392/go-django/src/core/contenttypes"
)

type identifiable interface {
	identifier() int
}

type TestStructOne struct {
	ID   int
	Name string
}

func (t *TestStructOne) identifier() int {
	return t.ID
}

type TestStructTwo struct {
	ID   int
	Name string
}

func (t *TestStructTwo) identifier() int {
	return t.ID
}

type TestStructThree struct {
	ID   int
	Name string
}

func (t *TestStructThree) identifier() int {
	return t.ID
}

var (
	instanceStorage = map[string]map[int]interface{}{
		"TestStructOne": {
			1: &TestStructOne{ID: 1, Name: "instance one"},
			2: &TestStructOne{ID: 2, Name: "instance two"},
		},
		"TestStructTwo": {
			1: TestStructTwo{ID: 1, Name: "instance one"},
		},
		"TestStructThree": {
			1: &TestStructThree{ID: 1, Name: "instance three"},
		},
	}
)

func getInstanceFunc(modelName string) func(interface{}) (interface{}, error) {
	return func(id interface{}) (interface{}, error) {
		idInt, ok := id.(int)
		if !ok {
			return nil, fmt.Errorf("invalid ID type")
		}
		if instance, exists := instanceStorage[modelName][idInt]; exists {
			return instance, nil
		}
		return nil, fmt.Errorf("instance not found")
	}
}

func getInstancesFunc(modelName string) func(uint, uint) ([]interface{}, error) {
	return func(amount, offset uint) ([]interface{}, error) {
		instances := instanceStorage[modelName]
		var results []interface{}
		var count uint
		for _, instance := range instances {
			if count >= offset && count < offset+amount {
				results = append(results, instance)
			}
			count++
		}
		slices.SortFunc(results, func(a, b any) int {
			idA := a.(identifiable).identifier()
			idB := b.(identifiable).identifier()
			if idA < idB {
				return -1
			}
			if idA > idB {
				return 1
			}
			return 0
		})
		return results, nil
	}
}

func sliceEquals(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		found := false
		for j := range b {
			if a[i] == b[j] {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
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
			GetInstance:  getInstanceFunc("TestStructOne"),
			GetInstances: getInstancesFunc("TestStructOne"),
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
			GetInstance:  getInstanceFunc("TestStructTwo"),
			GetInstances: getInstancesFunc("TestStructTwo"),
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
			GetInstance:  getInstanceFunc("TestStructThree"),
			GetInstances: getInstancesFunc("TestStructThree"),
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
				"github.com/Nigel2392/go-django/src/core/contenttypes_test.TestStructOne",
				"github.com/Nigel2392/go-django/src/core/contenttypes_test.TestStructTwo",
				"github.com/Nigel2392/go-django/src/core/contenttypes_test.TestStructThree",
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

				if ct.PkgPath() != "github.com/Nigel2392/go-django/src/core/contenttypes_test" {
					t.Errorf("expected %q, got %q", "github.com/Nigel2392/go-django/src/core/contenttypes_test", ct.PkgPath())
				}

				if ct.Model() != "TestStructOne" {
					t.Errorf("expected %q, got %q", "TestStructOne", ct.Model())
				}

				err = ct.Scan(typnameTwo)
				if err != nil {
					t.Errorf("expected nil error, got %v", err)
				}

				if ct.PkgPath() != "github.com/Nigel2392/go-django/src/core/contenttypes_test" {
					t.Errorf("expected %q, got %q", "github.com/Nigel2392/go-django/src/core/contenttypes_test", ct.PkgPath())
				}

				if ct.Model() != "TestStructTwo" {
					t.Errorf("expected %q, got %q", "TestStructTwo", ct.Model())
				}

				err = ct.Scan(typnameThree)
				if err != nil {
					t.Errorf("expected nil error, got %v", err)
				}

				if ct.PkgPath() != "github.com/Nigel2392/go-django/src/core/contenttypes_test" {
					t.Errorf("expected %q, got %q", "github.com/Nigel2392/go-django/src/core/contenttypes_test", ct.PkgPath())
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

		t.Run("TestAliasesAndReverseAliases", func(t *testing.T) {
			var cTyp = contenttypes.NewContentType(&TestStructOne{})
			var aliasesWithShortTypeName = append(aliasOne, cTyp.ShortTypeName())
			var aliases = contenttypes.Aliases(cTyp.TypeName())

			if !sliceEquals(aliases, aliasesWithShortTypeName) {
				t.Errorf("expected aliases %v, got %v", aliasesWithShortTypeName, aliases)
			}

			reverseAlias := contenttypes.ReverseAlias("test.TestStructOne")
			if reverseAlias != cTyp.TypeName() {
				t.Errorf("expected reverse alias %q, got %q", cTyp.TypeName(), reverseAlias)
			}
		})
		t.Run("TestGetInstance", func(t *testing.T) {
			instance, err := contenttypes.GetInstance("contenttypes_test.TestStructOne", 1)
			if err != nil || instance.(*TestStructOne).ID != 1 {
				t.Errorf("expected instance with ID 1, got %v, error: %v", instance, err)
			}
			instance, err = contenttypes.GetInstance("contenttypes_test.TestStructTwo", 1)
			if err != nil || instance.(TestStructTwo).ID != 1 {
				t.Errorf("expected instance with ID 1, got %v, error: %v", instance, err)
			}
			instance, err = contenttypes.GetInstance("contenttypes_test.TestStructThree", 1)
			if err != nil || instance.(*TestStructThree).ID != 1 {
				t.Errorf("expected instance with ID 1, got %v, error: %v", instance, err)
			}
		})

		t.Run("TestGetInstances", func(t *testing.T) {
			instances, err := contenttypes.GetInstances("contenttypes_test.TestStructOne", 2, 0)
			if err != nil || len(instances) != 2 {
				t.Errorf("expected 2 instances, got %v, error: %v", instances, err)
			}

			if instances[0].(*TestStructOne).ID != 1 {
				t.Errorf("expected instance with ID 1, got %v", instances[0])
			}

			instances, err = contenttypes.GetInstances("contenttypes_test.TestStructTwo", 1, 0)
			if err != nil || len(instances) != 1 {
				t.Errorf("expected 1 instance, got %v, error: %v", instances, err)
			}

			if instances[0].(TestStructTwo).ID != 1 {
				t.Errorf("expected instance with ID 1, got %v", instances[0])
			}
		})

		t.Run("TestGetInstancesByID", func(t *testing.T) {
			// Mock IDs for testing
			ids := []interface{}{1, 2}

			// Case 1: Test when `GetInstancesByID` is defined
			defOne.GetInstancesByIDs = func(ids []interface{}) ([]interface{}, error) {
				var instances []interface{}
				for _, id := range ids {
					if instance, exists := instanceStorage["TestStructOne"][id.(int)]; exists {
						instances = append(instances, instance)
					} else {
						return nil, fmt.Errorf("instance with ID %v not found", id)
					}
				}
				return instances, nil
			}

			t.Run("With GetInstancesByID defined", func(t *testing.T) {
				instances, err := defOne.InstancesByIDs(ids)
				if err != nil {
					t.Errorf("expected nil error, got %v", err)
				}
				if len(instances) != len(ids) {
					t.Errorf("expected %d instances, got %d", len(ids), len(instances))
				}
			})

			// Case 2: Test when `GetInstancesByID` is not defined, expecting a fallback to `GetInstance`
			defOne.GetInstancesByIDs = nil // Remove custom `GetInstancesByID`

			t.Run("Without GetInstancesByID, fallback to GetInstance", func(t *testing.T) {
				instances, err := defOne.InstancesByIDs(ids)
				if err != nil {
					t.Errorf("expected nil error, got %v", err)
				}
				if len(instances) != len(ids) {
					t.Errorf("expected %d instances, got %d", len(ids), len(instances))
				}

				slices.SortFunc(instances, func(a, b any) int {
					idA := a.(identifiable).identifier()
					idB := b.(identifiable).identifier()
					if idA < idB {
						return -1
					}
					if idA > idB {
						return 1
					}
					return 0
				})

				for i, instance := range instances {
					if instance.(identifiable).identifier() != ids[i] {
						t.Errorf("expected instance with ID %v, got %v", ids[i], instance)
					}
				}
			})
		})

		t.Run("TestRegisterDuplicateContentType", func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic on duplicate content type registration")
				}
			}()
			contenttypes.Register(defOne)
		})

		t.Run("TestPluralLabelAndDescription", func(t *testing.T) {
			if defOne.PluralLabel() != "test struct ones" {
				t.Errorf("expected plural label 'test struct ones', got %s", defOne.PluralLabel())
			}
			if defOne.Description() != "" {
				t.Errorf("expected empty description, got %s", defOne.Description())
			}
			defOne.GetDescription = func() string {
				return "A test struct"
			}
			if defOne.Description() != "A test struct" {
				t.Errorf("expected description 'A test struct', got %s", defOne.Description())
			}
		})

		t.Run("TestEditDefinition", func(t *testing.T) {
			newLabelFunc := func() string { return "modified label" }
			newDef := &contenttypes.ContentTypeDefinition{
				ContentObject: &TestStructOne{},
				GetLabel:      newLabelFunc,
			}
			contenttypes.EditDefinition(newDef)
			updatedDef := contenttypes.DefinitionForObject(&TestStructOne{})
			if updatedDef.GetLabel() != "modified label" {
				t.Errorf("expected updated label modified label, got %s", updatedDef.GetLabel())
			}
		})
	})
}
