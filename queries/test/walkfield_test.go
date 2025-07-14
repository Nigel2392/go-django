package queries_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

type WalkFieldTestNode struct {
	Path   string
	Part   string
	Model  attrs.Definer
	Fields []string
}

type WalkQuerySetFieldsTest struct {
	Model           attrs.Definer
	IncludeFinalRel bool
	FieldPath       string
	Aliases         []string
	Nodes           []WalkFieldTestNode
}

var walk_qs_fields_test = []WalkQuerySetFieldsTest{
	{
		Model:           &Todo{},
		FieldPath:       "User.Profile.Image",
		IncludeFinalRel: true,
		Nodes: []WalkFieldTestNode{
			{Model: &Todo{}, Fields: []string{"ID", "Title", "Description", "Done", "User"}, Path: "", Part: ""},
			{Model: &User{}, Fields: []string{"ID", "Name", "Profile", "ManyToManySet"}, Path: "User", Part: "User"},
			{Model: &Profile{}, Fields: []string{"ID", "Name", "Email", "Image"}, Path: "User.Profile", Part: "Profile"},
			{Model: &Image{}, Fields: []string{"ID", "Path"}, Path: "User.Profile.Image", Part: "Image"},
		},
	},
	{
		Model:           &Todo{},
		FieldPath:       "User",
		IncludeFinalRel: true,
		Nodes: []WalkFieldTestNode{
			{Model: &Todo{}, Fields: []string{"ID", "Title", "Description", "Done", "User"}, Path: "", Part: ""},
			{Model: &User{}, Fields: []string{"ID", "Name", "Profile", "ManyToManySet"}, Path: "User", Part: "User"},
		},
	},
	{
		Model:     &Todo{},
		FieldPath: "User",
		Nodes: []WalkFieldTestNode{
			{Model: &Todo{}, Fields: []string{"User"}, Path: "User", Part: "User"},
		},
	},
}

func TestAttrsWalkFields(t *testing.T) {
	t.Logf("Running %d WalkField tests", len(walk_qs_fields_test))

	for _, test := range walk_qs_fields_test {
		t.Run(fmt.Sprintf("%T.%s", test.Model, test.FieldPath), func(t *testing.T) {
			var chain, err = attrs.WalkRelationChain(test.Model, test.IncludeFinalRel, strings.Split(test.FieldPath, "."))
			if err != nil {
				t.Fatalf("WalkField(%s) failed: %v", test.FieldPath, err)
			}

			if len(chain.Chain) != len(test.Nodes)-1 {
				t.Fatalf("Expected %d nodes in chain, got %d (%v)", len(test.Nodes)-1, len(chain.Chain), chain.Chain)
			}

			var idx = 0
			var curr = chain.Root
			for curr != nil {

				var expectedNode = test.Nodes[idx]

				t.Logf("Checking node %d: %s (%T == %T)", idx, expectedNode.Path, curr.Model, expectedNode.Model)

				if reflect.TypeOf(curr.Model) != reflect.TypeOf(expectedNode.Model) {
					t.Errorf("Expected model %T, got %T", expectedNode.Model, curr.Model)
				}

				if idx > 0 {
					if reflect.TypeOf(curr.Prev.FieldRel.Model()) != reflect.TypeOf(expectedNode.Model) {
						t.Errorf("Expected field model %T, got %T", expectedNode.Model, curr.Prev.FieldRel.Model())
					}
				}

				curr = curr.Next
				idx++
			}

			if idx != len(test.Nodes) {
				t.Errorf("Expected %d nodes, got %d", len(test.Nodes), idx)
			}
		})
	}
}

func TestQuerySetWalkFields(t *testing.T) {
	t.Logf("Running %d WalkField tests", len(walk_qs_fields_test))

	for _, test := range walk_qs_fields_test {
		t.Run(fmt.Sprintf("%T.%s", test.Model, test.FieldPath), func(t *testing.T) {
			var querySet = queries.GetQuerySet(test.Model)
			var chain, aliasses, err = querySet.WalkField(test.FieldPath, test.IncludeFinalRel, true)
			if err != nil {
				t.Fatalf("WalkField(%s) failed: %v", test.FieldPath, err)
			}

			if len(test.Aliases) > 0 {
				if len(aliasses) != len(test.Aliases) {
					t.Fatalf("Expected %d aliases, got %d", len(test.Aliases), len(aliasses))
				}

				for i, alias := range aliasses {
					if alias != test.Aliases[i] {
						t.Errorf("Expected alias %s, got %s", test.Aliases[i], alias)
					}
				}
			}

			if len(chain.Chain) != len(test.Nodes)-1 {
				t.Errorf("Expected %d nodes in chain, got %d (%v)", len(test.Nodes)-1, len(chain.Chain), chain.Chain)
			}

			var idx = 0
			var curr = chain.Root
			for curr != nil {

				t.Logf("Checking node %d: %s (%T)", idx, curr.ChainPart, curr.Model)

				var expectedNode = test.Nodes[idx]
				t.Logf("Checking node %d: %s (%T == %T)", idx, expectedNode.Path, curr.Model, expectedNode.Model)

				if reflect.TypeOf(curr.Model) != reflect.TypeOf(expectedNode.Model) {
					t.Errorf("Expected model %T, got %T", expectedNode.Model, curr.Model)
				}

				if idx > 0 { // the root has no previous field
					if reflect.TypeOf(curr.Prev.FieldRel.Model()) != reflect.TypeOf(expectedNode.Model) {
						t.Errorf("Expected field model %T, got %T", expectedNode.Model, curr.Prev.FieldRel.Model())
					}
				}

				curr = curr.Next
				idx++
			}

			if idx != len(test.Nodes) {
				t.Errorf("Expected %d nodes, got %d", len(test.Nodes), idx)
			}
		})
	}

}
