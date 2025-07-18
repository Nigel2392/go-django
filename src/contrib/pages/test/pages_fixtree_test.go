package pages_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Nigel2392/go-django/src/contrib/pages"
)

var FixTreeTest_DATA_OK = []*pages.PageNode{
	{PK: 1, Title: "Test Page 1", Path: "001", Depth: 0, Numchild: 6},
	{PK: 2, Title: "Test Page 1-1", Path: "001001", Depth: 1, Numchild: 0},
	{PK: 3, Title: "Test Page 1-2", Path: "001002", Depth: 1, Numchild: 6},
	{PK: 4, Title: "Test Page 1-3", Path: "001003", Depth: 1, Numchild: 0},
	{PK: 5, Title: "Test Page 1-4", Path: "001004", Depth: 1, Numchild: 0},
	{PK: 6, Title: "Test Page 1-5", Path: "001005", Depth: 1, Numchild: 0},
	{PK: 7, Title: "Test Page 1-6", Path: "001006", Depth: 1, Numchild: 0},
	{PK: 8, Title: "Test Page 1-2-1", Path: "001002001", Depth: 2, Numchild: 0},
	{PK: 9, Title: "Test Page 1-2-2", Path: "001002002", Depth: 2, Numchild: 0},
	{PK: 10, Title: "Test Page 1-2-3", Path: "001002003", Depth: 2, Numchild: 0},
	{PK: 11, Title: "Test Page 1-2-4", Path: "001002004", Depth: 2, Numchild: 0},
	{PK: 12, Title: "Test Page 1-2-5", Path: "001002005", Depth: 2, Numchild: 5},
	{PK: 13, Title: "Test Page 1-2-6", Path: "001002006", Depth: 2, Numchild: 0},
	{PK: 14, Title: "Test Page 1-2-7", Path: "001002007", Depth: 2, Numchild: 0},
	{PK: 15, Title: "Test Page 1-2-5-1", Path: "001002005001", Depth: 3, Numchild: 0},
	{PK: 16, Title: "Test Page 1-2-5-2", Path: "001002005002", Depth: 3, Numchild: 0},
	{PK: 17, Title: "Test Page 1-2-5-3", Path: "001002005003", Depth: 3, Numchild: 0},
	{PK: 18, Title: "Test Page 1-2-5-4", Path: "001002005004", Depth: 3, Numchild: 0},
	{PK: 19, Title: "Test Page 1-2-5-5", Path: "001002005005", Depth: 3, Numchild: 0},
	{PK: 20, Title: "Test Page 2", Path: "002", Depth: 0, Numchild: 6},
	{PK: 21, Title: "Test Page 2-1", Path: "002001", Depth: 1, Numchild: 0},
	{PK: 22, Title: "Test Page 2-2", Path: "002002", Depth: 1, Numchild: 0},
	{PK: 23, Title: "Test Page 2-3", Path: "002003", Depth: 1, Numchild: 0},
	{PK: 24, Title: "Test Page 2-4", Path: "002004", Depth: 1, Numchild: 0},
	{PK: 25, Title: "Test Page 2-5", Path: "002005", Depth: 1, Numchild: 0},
	{PK: 26, Title: "Test Page 2-6", Path: "002006", Depth: 1, Numchild: 0},
}

var FixTreeTest_DATA_WRONG = []*pages.PageNode{
	{PK: 1, Title: "Test Page 1", Path: "001", Depth: 0, Slug: "/test-page-1"},
	{PK: 2, Title: "Test Page 1-1", Path: "001001", Depth: 1, Slug: "test-page-1-1"},
	{PK: 3, Title: "Test Page 1-2", Path: "001002", Depth: 69, Slug: "test-page-1-2"},
	{PK: 4, Title: "Test Page 1-3", Path: "001003", Depth: 1, Slug: "test-page-1-3"},
	{PK: 5, Title: "Test Page 1-4", Path: "001004", Depth: 1, Slug: "test-page-1-4"},
	{PK: 6, Title: "Test Page 1-5", Path: "001005", Depth: 1, Slug: "test-page-1-5"},
	{PK: 7, Title: "Test Page 1-6", Path: "001006", Depth: 1, Slug: "test-page-1-6"},
	{PK: 8, Title: "Test Page 1-2-1", Path: "001002001", Depth: 2, Slug: "test-page-1-2-1"},
	{PK: 9, Title: "Test Page 1-2-2", Path: "001002009", Depth: 2, Slug: "test-page-1-2-2"},
	{PK: 10, Title: "Test Page 1-2-3", Path: "001002003", Depth: 2, Slug: "test-page-1-2-3"},
	{PK: 11, Title: "Test Page 1-2-4", Path: "001002002", Depth: 2, Slug: "test-page-1-2-4"},
	{PK: 12, Title: "Test Page 1-2-5", Path: "001002005", Depth: 2, Slug: "test-page-1-2-5"},
	{PK: 13, Title: "Test Page 1-2-6", Path: "001002007", Depth: 2, Slug: "test-page-1-2-6"},
	{PK: 14, Title: "Test Page 1-2-7", Path: "001002007", Depth: 2, Slug: "test-page-1-2-7"},
	{PK: 15, Title: "Test Page 1-2-5-1", Path: "001002005001", Depth: 3, Slug: "test-page-1-2-5-1"},
	{PK: 16, Title: "Test Page 1-2-5-2", Path: "001002005002", Depth: 3, Slug: "test-page-1-2-5-2"},
	{PK: 17, Title: "Test Page 1-2-5-3", Path: "001002005003", Depth: 3, Slug: "test-page-1-2-5-3"},
	{PK: 18, Title: "Test Page 1-2-5-4", Path: "001002005004", Depth: 3, Slug: "test-page-1-2-5-4"},
	{PK: 19, Title: "Test Page 1-2-5-5", Path: "001002005005", Depth: 3, Slug: "test-page-1-2-5-5"},
	{PK: 20, Title: "Test Page 2", Path: "002", Depth: 0, Slug: "test-page-2"},
	{PK: 21, Title: "Test Page 2-1", Path: "002001", Depth: 1, Slug: "test-page-2-1"},
	{PK: 22, Title: "Test Page 2-2", Path: "002002", Depth: 1, Slug: "test-page-2-2"},
	{PK: 23, Title: "Test Page 2-3", Path: "002003", Depth: 1, Slug: "test-page-2-3"},
	{PK: 24, Title: "Test Page 2-4", Path: "002004", Depth: 1, Slug: "test-page-2-4"},
	{PK: 25, Title: "Test Page 2-5", Path: "002005", Depth: 1, Slug: "test-page-2-5"},
	{PK: 26, Title: "Test Page 2-6", Path: "002006", Depth: 1, Slug: "test-page-2-6"},
}

type TreeTraversalTest struct {
	Path string
	Test func(t *testing.T, n *pages.PageNode) error
}

func testFuncBuilder(match ...func(t *testing.T, n *pages.PageNode) error) func(t *testing.T, n *pages.PageNode) error {
	return func(t *testing.T, n *pages.PageNode) error {
		for _, f := range match {
			if err := f(t, n); err != nil {
				return err
			}
		}
		return nil
	}
}

func testFuncPath(path string) func(t *testing.T, n *pages.PageNode) error {
	return func(t *testing.T, n *pages.PageNode) error {
		if n.Path != path {
			return fmt.Errorf("Expected path %s, got %s", path, n.Path)
		}
		return nil
	}
}

func testNumChild(num int64) func(t *testing.T, n *pages.PageNode) error {
	return func(t *testing.T, n *pages.PageNode) error {
		if n.Numchild != num {
			return fmt.Errorf("Expected numchild %d, got %d", num, n.Numchild)
		}
		return nil
	}
}

func testUrlPath(url string) func(t *testing.T, n *pages.PageNode) error {
	return func(t *testing.T, n *pages.PageNode) error {
		if n.UrlPath != url {
			return fmt.Errorf("Expected urlpath %s, got %s", url, n.UrlPath)
		}
		return nil
	}
}

func testFuncDepth(depth int64) func(t *testing.T, n *pages.PageNode) error {
	return func(t *testing.T, n *pages.PageNode) error {
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
			testUrlPath("/test-page-1"),
		),
	},
	{Path: "001", Test: testFuncBuilder(
		testFuncPath("001"),
		testFuncDepth(0),
		testNumChild(6),
		testUrlPath("/test-page-1"),
	)},
	{Path: "001001", Test: testFuncBuilder(
		testFuncPath("001001"),
		testFuncDepth(1),
		testNumChild(0),
		testUrlPath("/test-page-1/test-page-1-1"),
	)},
	{Path: "001002", Test: testFuncBuilder(
		testFuncPath("001002"),
		testFuncDepth(1),
		testNumChild(6),
		testUrlPath("/test-page-1/test-page-1-2"),
	)},
	{Path: "001002001", Test: testFuncBuilder(
		testFuncPath("001002001"),
		testFuncDepth(2),
		testNumChild(0),
		testUrlPath("/test-page-1/test-page-1-2/test-page-1-2-1"),
	)},
	{Path: "001002002", Test: testFuncBuilder(
		testFuncPath("001002002"),
		testFuncDepth(2),
		testNumChild(0),
		testUrlPath("/test-page-1/test-page-1-2/test-page-1-2-4"),
	)},
	{Path: "001002003", Test: testFuncBuilder(
		testFuncPath("001002003"),
		testFuncDepth(2),
		testNumChild(0),
		testUrlPath("/test-page-1/test-page-1-2/test-page-1-2-3"),
	)},
	{Path: "001002004", Test: testFuncBuilder(
		testFuncPath("001002004"),
		testFuncDepth(2),
		testNumChild(5),
		testUrlPath("/test-page-1/test-page-1-2/test-page-1-2-5"),
	)},
	{Path: "001002004001", Test: testFuncBuilder(
		testFuncPath("001002004001"),
		testFuncDepth(3),
		testNumChild(0),
		testUrlPath("/test-page-1/test-page-1-2/test-page-1-2-5/test-page-1-2-5-1"),
	)},
	{Path: "001002004002", Test: testFuncBuilder(
		testFuncPath("001002004002"),
		testFuncDepth(3),
		testNumChild(0),
		testUrlPath("/test-page-1/test-page-1-2/test-page-1-2-5/test-page-1-2-5-2"),
	)},
	{Path: "001002004003", Test: testFuncBuilder(
		testFuncPath("001002004003"),
		testFuncDepth(3),
		testNumChild(0),
		testUrlPath("/test-page-1/test-page-1-2/test-page-1-2-5/test-page-1-2-5-3"),
	)},
	{Path: "001002004004", Test: testFuncBuilder(
		testFuncPath("001002004004"),
		testFuncDepth(3),
		testNumChild(0),
		testUrlPath("/test-page-1/test-page-1-2/test-page-1-2-5/test-page-1-2-5-4"),
	)},
	{Path: "001002004005", Test: testFuncBuilder(
		testFuncPath("001002004005"),
		testFuncDepth(3),
		testNumChild(0),
		testUrlPath("/test-page-1/test-page-1-2/test-page-1-2-5/test-page-1-2-5-5"),
	)},
	{Path: "001002005", Test: testFuncBuilder(
		testFuncPath("001002005"),
		testFuncDepth(2),
		testNumChild(0),
		testUrlPath("/test-page-1/test-page-1-2/test-page-1-2-7"),
	)},
	{Path: "001002006", Test: testFuncBuilder(
		testFuncPath("001002006"),
		testFuncDepth(2),
		testNumChild(0),
		testUrlPath("/test-page-1/test-page-1-2/test-page-1-2-2"),
	)},
	{Path: "001003", Test: testFuncBuilder(
		testFuncPath("001003"),
		testFuncDepth(1),
		testNumChild(0),
		testUrlPath("/test-page-1/test-page-1-3"),
	)},
	{Path: "001004", Test: testFuncBuilder(
		testFuncPath("001004"),
		testFuncDepth(1),
		testNumChild(0),
		testUrlPath("/test-page-1/test-page-1-4"),
	)},
	{Path: "001005", Test: testFuncBuilder(
		testFuncPath("001005"),
		testFuncDepth(1),
		testNumChild(0),
		testUrlPath("/test-page-1/test-page-1-5"),
	)},
	{Path: "001006", Test: testFuncBuilder(
		testFuncPath("001006"),
		testFuncDepth(1),
		testNumChild(0),
		testUrlPath("/test-page-1/test-page-1-6"),
	)},
	{Path: "002", Test: testFuncBuilder(
		testFuncPath("002"),
		testFuncDepth(0),
		testNumChild(6),
		testUrlPath("/test-page-2"),
	)},
	{Path: "002001", Test: testFuncBuilder(
		testFuncPath("002001"),
		testFuncDepth(1),
		testNumChild(0),
		testUrlPath("/test-page-2/test-page-2-1"),
	)},
	{Path: "002002", Test: testFuncBuilder(
		testFuncPath("002002"),
		testFuncDepth(1),
		testNumChild(0),
		testUrlPath("/test-page-2/test-page-2-2"),
	)},
	{Path: "002003", Test: testFuncBuilder(
		testFuncPath("002003"),
		testFuncDepth(1),
		testNumChild(0),
		testUrlPath("/test-page-2/test-page-2-3"),
	)},
	{Path: "002004", Test: testFuncBuilder(
		testFuncPath("002004"),
		testFuncDepth(1),
		testNumChild(0),
		testUrlPath("/test-page-2/test-page-2-4"),
	)},
	{Path: "002005", Test: testFuncBuilder(
		testFuncPath("002005"),
		testFuncDepth(1),
		testNumChild(0),
		testUrlPath("/test-page-2/test-page-2-5"),
	)},
	{Path: "002006", Test: testFuncBuilder(
		testFuncPath("002006"),
		testFuncDepth(1),
		testNumChild(0),
		testUrlPath("/test-page-2/test-page-2-6"),
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

	var d, _ = json.MarshalIndent(node, "", "  ")

	t.Logf("%s", d)
}
