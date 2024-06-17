package pages_test

import (
	"fmt"
	"testing"

	"github.com/Nigel2392/django/contrib/pages"
	"github.com/Nigel2392/django/contrib/pages/models"
)

var FixTreeTest_DATA_OK = []*models.PageNode{
	{ID: 1, Title: "Test Page 1", Path: "001", Depth: 0, Numchild: 6},
	{ID: 2, Title: "Test Page 1-1", Path: "001001", Depth: 1, Numchild: 0},
	{ID: 3, Title: "Test Page 1-2", Path: "001002", Depth: 1, Numchild: 6},
	{ID: 4, Title: "Test Page 1-3", Path: "001003", Depth: 1, Numchild: 0},
	{ID: 5, Title: "Test Page 1-4", Path: "001004", Depth: 1, Numchild: 0},
	{ID: 6, Title: "Test Page 1-5", Path: "001005", Depth: 1, Numchild: 0},
	{ID: 7, Title: "Test Page 1-6", Path: "001006", Depth: 1, Numchild: 0},
	{ID: 8, Title: "Test Page 1-2-1", Path: "001002001", Depth: 2, Numchild: 0},
	{ID: 9, Title: "Test Page 1-2-2", Path: "001002002", Depth: 2, Numchild: 0},
	{ID: 10, Title: "Test Page 1-2-3", Path: "001002003", Depth: 2, Numchild: 0},
	{ID: 11, Title: "Test Page 1-2-4", Path: "001002004", Depth: 2, Numchild: 0},
	{ID: 12, Title: "Test Page 1-2-5", Path: "001002005", Depth: 2, Numchild: 5},
	{ID: 13, Title: "Test Page 1-2-6", Path: "001002006", Depth: 2, Numchild: 0},
	{ID: 14, Title: "Test Page 1-2-7", Path: "001002007", Depth: 2, Numchild: 0},
	{ID: 15, Title: "Test Page 1-2-5-1", Path: "001002005001", Depth: 3},
	{ID: 16, Title: "Test Page 1-2-5-2", Path: "001002005002", Depth: 3},
	{ID: 17, Title: "Test Page 1-2-5-3", Path: "001002005003", Depth: 3},
	{ID: 18, Title: "Test Page 1-2-5-4", Path: "001002005004", Depth: 3},
	{ID: 19, Title: "Test Page 1-2-5-5", Path: "001002005005", Depth: 3},
	{ID: 20, Title: "Test Page 2", Path: "002", Depth: 0, Numchild: 6},
	{ID: 21, Title: "Test Page 2-1", Path: "002001", Depth: 1},
	{ID: 22, Title: "Test Page 2-2", Path: "002002", Depth: 1},
	{ID: 23, Title: "Test Page 2-3", Path: "002003", Depth: 1},
	{ID: 24, Title: "Test Page 2-4", Path: "002004", Depth: 1},
	{ID: 25, Title: "Test Page 2-5", Path: "002005", Depth: 1},
	{ID: 26, Title: "Test Page 2-6", Path: "002006", Depth: 1},
}

var FixTreeTest_DATA_WRONG = []*models.PageNode{
	{ID: 1, Title: "Test Page 1", Path: "001", Depth: 0},
	{ID: 2, Title: "Test Page 1-1", Path: "001001", Depth: 1},
	{ID: 3, Title: "Test Page 1-2", Path: "001002", Depth: 69},
	{ID: 4, Title: "Test Page 1-3", Path: "001003", Depth: 1},
	{ID: 5, Title: "Test Page 1-4", Path: "001004", Depth: 1},
	{ID: 6, Title: "Test Page 1-5", Path: "001005", Depth: 1},
	{ID: 7, Title: "Test Page 1-6", Path: "001006", Depth: 1},
	{ID: 8, Title: "Test Page 1-2-1", Path: "001002001", Depth: 2},
	{ID: 9, Title: "Test Page 1-2-2", Path: "001002009", Depth: 2},
	{ID: 10, Title: "Test Page 1-2-3", Path: "001002003", Depth: 2},
	{ID: 11, Title: "Test Page 1-2-4", Path: "001002002", Depth: 2},
	{ID: 12, Title: "Test Page 1-2-5", Path: "001002005", Depth: 2},
	{ID: 13, Title: "Test Page 1-2-6", Path: "001002007", Depth: 2},
	{ID: 14, Title: "Test Page 1-2-7", Path: "001002007", Depth: 2},
	{ID: 15, Title: "Test Page 1-2-5-1", Path: "001002005001", Depth: 3},
	{ID: 16, Title: "Test Page 1-2-5-2", Path: "001002005002", Depth: 3},
	{ID: 17, Title: "Test Page 1-2-5-3", Path: "001002005003", Depth: 3},
	{ID: 18, Title: "Test Page 1-2-5-4", Path: "001002005004", Depth: 3},
	{ID: 19, Title: "Test Page 1-2-5-5", Path: "001002005005", Depth: 3},
	{ID: 20, Title: "Test Page 2", Path: "002", Depth: 0},
	{ID: 21, Title: "Test Page 2-1", Path: "002001", Depth: 1},
	{ID: 22, Title: "Test Page 2-2", Path: "002002", Depth: 1},
	{ID: 23, Title: "Test Page 2-3", Path: "002003", Depth: 1},
	{ID: 24, Title: "Test Page 2-4", Path: "002004", Depth: 1},
	{ID: 25, Title: "Test Page 2-5", Path: "002005", Depth: 1},
	{ID: 26, Title: "Test Page 2-6", Path: "002006", Depth: 1},
}

type TreeTraversalTest struct {
	Path string
	Test func(t *testing.T, n *models.PageNode) error
}

func testFuncBuilder(match ...func(t *testing.T, n *models.PageNode) error) func(t *testing.T, n *models.PageNode) error {
	return func(t *testing.T, n *models.PageNode) error {
		for _, f := range match {
			if err := f(t, n); err != nil {
				return err
			}
		}
		return nil
	}
}

func testFuncPath(path string) func(t *testing.T, n *models.PageNode) error {
	return func(t *testing.T, n *models.PageNode) error {
		if n.Path != path {
			return fmt.Errorf("Expected path %s, got %s", path, n.Path)
		}
		return nil
	}
}

func testNumChild(num int64) func(t *testing.T, n *models.PageNode) error {
	return func(t *testing.T, n *models.PageNode) error {
		if n.Numchild != num {
			return fmt.Errorf("Expected numchild %d, got %d", num, n.Numchild)
		}
		return nil
	}
}

func testFuncDepth(depth int64) func(t *testing.T, n *models.PageNode) error {
	return func(t *testing.T, n *models.PageNode) error {
		if n.Depth != depth {
			return fmt.Errorf("Expected	 depth %d, got %d", depth, n.Depth)
		}
		return nil
	}
}

var (
	BuildTreeTest = []TreeTraversalTest{
		{
			Path: "001",
			Test: testFuncBuilder(
				testFuncPath("001"),
				testFuncDepth(0),
			),
		},
		{
			Path: "001001",
			Test: testFuncBuilder(
				testFuncPath("001001"),
				testFuncDepth(1),
			),
		}, {
			Path: "001002005",
			Test: testFuncBuilder(
				testFuncPath("001002005"),
				testFuncDepth(2),
				testNumChild(5),
			),
		},
		{
			Path: "001002005001",
			Test: testFuncBuilder(
				testFuncPath("001002005001"),
				testFuncDepth(3),
			),
		},
	}
)

func TestBuildTree(t *testing.T) {
	var node = pages.NewNodeTree(FixTreeTest_DATA_OK)

	for _, test := range BuildTreeTest {
		t.Run(fmt.Sprintf("TestNode-%s", test.Path), func(t *testing.T) {
			var n = node.FindNode(test.Path)

			if n == nil {
				t.Errorf("Node not found for path %s", test.Path)
				return
			}

			if err := test.Test(t, n.Ref); err != nil {
				t.Errorf("Test failed for path %s (%s)", test.Path, err)
			}
		})
	}
}

var FixTreeTest = []TreeTraversalTest{
	{
		Path: "001",
		Test: testFuncBuilder(
			testFuncPath("001"),
			testFuncDepth(0),
			testNumChild(6),
		),
	},
	{Path: "001", Test: testFuncBuilder(
		testFuncPath("001"),
		testFuncDepth(0),
		testNumChild(6),
	)},
	{Path: "001001", Test: testFuncBuilder(
		testFuncPath("001001"),
		testFuncDepth(1),
		testNumChild(0),
	)},
	{Path: "001002", Test: testFuncBuilder(
		testFuncPath("001002"),
		testFuncDepth(1),
		testNumChild(6),
	)},
	{Path: "001002001", Test: testFuncBuilder(
		testFuncPath("001002001"),
		testFuncDepth(2),
		testNumChild(0),
	)},
	{Path: "001002002", Test: testFuncBuilder(
		testFuncPath("001002002"),
		testFuncDepth(2),
		testNumChild(0),
	)},
	{Path: "001002003", Test: testFuncBuilder(
		testFuncPath("001002003"),
		testFuncDepth(2),
		testNumChild(0),
	)},
	{Path: "001002004", Test: testFuncBuilder(
		testFuncPath("001002004"),
		testFuncDepth(2),
		testNumChild(5),
	)},
	{Path: "001002004001", Test: testFuncBuilder(
		testFuncPath("001002004001"),
		testFuncDepth(3),
		testNumChild(0),
	)},
	{Path: "001002004002", Test: testFuncBuilder(
		testFuncPath("001002004002"),
		testFuncDepth(3),
		testNumChild(0),
	)},
	{Path: "001002004003", Test: testFuncBuilder(
		testFuncPath("001002004003"),
		testFuncDepth(3),
		testNumChild(0),
	)},
	{Path: "001002004004", Test: testFuncBuilder(
		testFuncPath("001002004004"),
		testFuncDepth(3),
		testNumChild(0),
	)},
	{Path: "001002004005", Test: testFuncBuilder(
		testFuncPath("001002004005"),
		testFuncDepth(3),
		testNumChild(0),
	)},
	{Path: "001002005", Test: testFuncBuilder(
		testFuncPath("001002005"),
		testFuncDepth(2),
		testNumChild(0),
	)},
	{Path: "001002006", Test: testFuncBuilder(
		testFuncPath("001002006"),
		testFuncDepth(2),
		testNumChild(0),
	)},
	{Path: "001003", Test: testFuncBuilder(
		testFuncPath("001003"),
		testFuncDepth(1),
		testNumChild(0),
	)},
	{Path: "001004", Test: testFuncBuilder(
		testFuncPath("001004"),
		testFuncDepth(1),
		testNumChild(0),
	)},
	{Path: "001005", Test: testFuncBuilder(
		testFuncPath("001005"),
		testFuncDepth(1),
		testNumChild(0),
	)},
	{Path: "001006", Test: testFuncBuilder(
		testFuncPath("001006"),
		testFuncDepth(1),
		testNumChild(0),
	)},
	{Path: "002", Test: testFuncBuilder(
		testFuncPath("002"),
		testFuncDepth(0),
		testNumChild(6),
	)},
	{Path: "002001", Test: testFuncBuilder(
		testFuncPath("002001"),
		testFuncDepth(1),
		testNumChild(0),
	)},
	{Path: "002002", Test: testFuncBuilder(
		testFuncPath("002002"),
		testFuncDepth(1),
		testNumChild(0),
	)},
	{Path: "002003", Test: testFuncBuilder(
		testFuncPath("002003"),
		testFuncDepth(1),
		testNumChild(0),
	)},
	{Path: "002004", Test: testFuncBuilder(
		testFuncPath("002004"),
		testFuncDepth(1),
		testNumChild(0),
	)},
	{Path: "002005", Test: testFuncBuilder(
		testFuncPath("002005"),
		testFuncDepth(1),
		testNumChild(0),
	)},
	{Path: "002006", Test: testFuncBuilder(
		testFuncPath("002006"),
		testFuncDepth(1),
		testNumChild(0),
	)},
}

func TestFixTree(t *testing.T) {
	var node = pages.NewNodeTree(FixTreeTest_DATA_WRONG)

	node.FixTree()

	for _, test := range FixTreeTest {
		t.Run(fmt.Sprintf("TestFixTreeNode-%s", test.Path), func(t *testing.T) {
			var n = node.FindNode(test.Path)

			if n == nil {
				t.Errorf("Node not found for path %s", test.Path)
				return
			}

			if err := test.Test(t, n.Ref); err != nil {
				t.Errorf("Test failed for path %s (%s)", test.Path, err)
			}
		})
	}
}
