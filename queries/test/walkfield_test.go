package queries_test

import (
	"fmt"
	"reflect"
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

type WalkQuerySetFieldTest struct {
	Model           attrs.Definer
	IncludeFinalRel bool
	FieldPath       string
	Aliases         []string
	Nodes           []WalkFieldTestNode
}

var walk_qs_fields_test = []WalkQuerySetFieldTest{
	{
		Model:     &Todo{},
		FieldPath: "User.Profile.Image",
		Nodes: []WalkFieldTestNode{
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

func WalkQuerySetFieldsTest(t *testing.T) {
	for _, test := range walk_qs_fields_test {
		t.Run(fmt.Sprintf("%T.%s", test.Model, test.FieldPath), func(t *testing.T) {
			var querySet = queries.GetQuerySet(test.Model)
			var chain, aliasses, err = querySet.WalkField(test.FieldPath, true, true)
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

			if len(chain.Chain) != len(test.Nodes) {
				t.Fatalf("Expected %d nodes in chain, got %d", len(test.Nodes), len(chain.Chain))
			}

			var idx = 0
			var curr = chain.Root
			for curr != nil {
				if idx >= len(test.Nodes) {
					t.Fatalf("More nodes in chain than expected: %d > %d", idx, len(test.Nodes))
				}

				var expectedNode = test.Nodes[idx]
				if reflect.TypeOf(curr.Model) != reflect.TypeOf(expectedNode.Model) {
					t.Errorf("Expected model %T, got %T", expectedNode.Model, curr.Model)
				}

				curr = curr.Next
				idx++
			}

		})
	}
}
