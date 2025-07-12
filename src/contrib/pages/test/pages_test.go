package pages_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	_ "unsafe"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/quest"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/pages"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/djester/testdb"
	"github.com/Nigel2392/go-signals"
)

//go:linkname insertNode github.com/Nigel2392/go-django/src/contrib/pages.(*PageQuerySet).insertNode
func insertNode(qs *pages.PageQuerySet, node *pages.PageNode) (*pages.PageNode, error)

//go:linkname updateNodes github.com/Nigel2392/go-django/src/contrib/pages.(*PageQuerySet).updateNodes
func updateNodes(qs *pages.PageQuerySet, nodes []*pages.PageNode) error

//go:linkname incrementNumChild github.com/Nigel2392/go-django/src/contrib/pages.(*PageQuerySet).incrementNumChild
func incrementNumChild(qs *pages.PageQuerySet, pk int64) (*pages.PageNode, error)

//go:linkname decrementNumChild github.com/Nigel2392/go-django/src/contrib/pages.(*PageQuerySet).decrementNumChild
func decrementNumChild(qs *pages.PageQuerySet, pk int64) (*pages.PageNode, error)

func TestMain(m *testing.M) {

	var _, sqlDB = testdb.Open()

	attrs.RegisterModel(&pages.PageNode{})
	attrs.RegisterModel(&TestPage{})
	attrs.RegisterModel(&DBTestPage{})

	if django.Global == nil {
		django.App(django.Configure(map[string]interface{}{
			django.APPVAR_DATABASE: sqlDB,
		}),
			django.Flag(django.FlagSkipDepsCheck),
		)

		logger.Setup(&logger.Logger{
			Level:       logger.DBG,
			WrapPrefix:  logger.ColoredLogWrapper,
			OutputDebug: os.Stdout,
			OutputInfo:  os.Stdout,
			OutputWarn:  os.Stdout,
			OutputError: os.Stderr,
		})
	}

	var t = quest.Table[*testing.T](nil,
		&pages.PageNode{},
		&TestPage{},
	)

	t.Create()

	exitCode := m.Run()

	t.Drop()

	os.Exit(exitCode)

}

var _ pages.Page = &TestPage{}

type TestPage struct {
	Ref         *pages.PageNode
	Identifier  int64
	Description string
}

func (t *TestPage) FieldDefs() attrs.Definitions {
	return attrs.Define(t,
		attrs.Unbound("Identifier", &attrs.FieldConfig{
			Primary: true,
			Column:  "id",
		}),
		attrs.Unbound("Description"),
	).WithTableName("test_pages")
}

func nodesEqual(a, b *pages.PageNode) bool {
	pageARef := *a
	pageBRef := *b
	pageARef.CreatedAt = pageBRef.CreatedAt
	pageARef.UpdatedAt = pageBRef.UpdatedAt

	return a.PK == b.PK &&
		a.Title == b.Title &&
		a.Path == b.Path &&
		a.Depth == b.Depth &&
		a.Numchild == b.Numchild &&
		a.UrlPath == b.UrlPath &&
		a.Slug == b.Slug &&
		a.StatusFlags == b.StatusFlags &&
		a.PageID == b.PageID &&
		a.ContentType == b.ContentType
}

func nodeDiff(a, b *pages.PageNode) map[string]string { // map of name: v != v
	diff := make(map[string]string)
	if a.PK != b.PK {
		diff["PK"] = fmt.Sprintf("%d != %d", a.PK, b.PK)
	}
	if a.Title != b.Title {
		diff["Title"] = fmt.Sprintf("%s != %s", a.Title, b.Title)
	}
	if a.Path != b.Path {
		diff["Path"] = fmt.Sprintf("%s != %s", a.Path, b.Path)
	}
	if a.Depth != b.Depth {
		diff["Depth"] = fmt.Sprintf("%d != %d", a.Depth, b.Depth)
	}
	if a.Numchild != b.Numchild {
		diff["Numchild"] = fmt.Sprintf("%d != %d", a.Numchild, b.Numchild)
	}
	if a.UrlPath != b.UrlPath {
		diff["UrlPath"] = fmt.Sprintf("%s != %s", a.UrlPath, b.UrlPath)
	}
	if a.Slug != b.Slug {
		diff["Slug"] = fmt.Sprintf("%s != %s", a.Slug, b.Slug)
	}
	if a.StatusFlags != b.StatusFlags {
		diff["StatusFlags"] = fmt.Sprintf("%d != %d", a.StatusFlags, b.StatusFlags)
	}
	if a.PageID != b.PageID {
		diff["PageID"] = fmt.Sprintf("%d != %d", a.PageID, b.PageID)
	}
	if a.ContentType != b.ContentType {
		diff["ContentType"] = fmt.Sprintf("%s != %s", a.ContentType, b.ContentType)
	}
	if a.LatestRevisionID != b.LatestRevisionID {
		diff["LatestRevisionID"] = fmt.Sprintf("%d != %d", a.LatestRevisionID, b.LatestRevisionID)
	}
	if a.CreatedAt != b.CreatedAt {
		diff["CreatedAt"] = fmt.Sprintf("%s != %s", a.CreatedAt, b.CreatedAt)
	}
	if a.UpdatedAt != b.UpdatedAt {
		diff["UpdatedAt"] = fmt.Sprintf("%s != %s", a.UpdatedAt, b.UpdatedAt)
	}
	return diff
}

func (t *TestPage) ID() int64 {
	return int64(t.Identifier)
}

func (t *TestPage) Reference() *pages.PageNode {
	return t.Ref
}

func (t *TestPage) Save(ctx context.Context) error {
	return nil
}

type DBTestPage struct {
	TestPage
}

func (t *DBTestPage) Save(ctx context.Context) error {
	var err error
	if t.Identifier == 0 {
		_, err = queries.GetQuerySet(&DBTestPage{}).ExplicitSave().Create(
			t,
		)
	} else {
		_, err = queries.GetQuerySet(&DBTestPage{}).ExplicitSave().Update(
			t,
		)
	}
	return err
}

func TestContentType(t *testing.T) {
	var cType = contenttypes.NewContentType(&TestPage{})

	t.Run("TypeName", func(t *testing.T) {
		if cType.Model() != "TestPage" {
			t.Fatalf("expected TestPage as Model, got %s", cType.Model())
		}
	})

}

func TestPageRegistry(t *testing.T) {
	t.Run("Definition", func(t *testing.T) {

		var definition = &pages.PageDefinition{
			ContentTypeDefinition: &contenttypes.ContentTypeDefinition{
				GetLabel: trans.S(""),
				ContentObject: &TestPage{
					Ref: &pages.PageNode{
						PK:     1,
						Title:  "Title",
						Path:   "001",
						PageID: 69,
						Depth:  0,
					},
					Identifier:  69,
					Description: "Description",
				},
			},
			GetForID: func(ctx context.Context, node *pages.PageNode, id int64) (pages.Page, error) {
				return &TestPage{
					Ref:         node,
					Identifier:  id,
					Description: fmt.Sprintf("Description for %s (%d)", node.Title, id),
				}, nil
			},
		}

		pages.Register(definition)
		var cType = definition.ContentType()

		t.Run("SpecificInstance", func(t *testing.T) {
			var node = &pages.PageNode{
				Title:       "Test Page",
				PageID:      69,
				ContentType: cType.TypeName(),
			}
			var instance, err = node.Specific(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			var testPage, ok = instance.(*TestPage)
			if !ok {
				t.Fatalf("expected *TestPage, got %T", instance)
			}
			var descString = fmt.Sprintf("Description for %s (%d)", node.Title, node.PageID)
			if testPage.Description != descString {
				t.Fatalf("expected %q, got %q", descString, testPage.Description)
			}
		})

		t.Run("ForObject", func(t *testing.T) {
			var testDef = pages.DefinitionForObject(&TestPage{})
			if testDef == nil {
				t.Fatal("expected definition, got nil")
				return
			}

			if definition.ContentObject != testDef.ContentObject {
				t.Fatalf("expected %+v, got %+v", definition.ContentObject, testDef.ContentObject)
			}

			var page = testDef.ContentObject.(*TestPage)
			if page.Description != "Description" {
				t.Fatalf("expected Description, got %s", page.Description)
			}

			if page.Identifier != 69 {
				t.Fatalf("expected Identifier, got %d", page.Identifier)
			}

			if page.Ref.Title != "Title" {
				t.Fatalf("expected Title, got %s", page.Ref.Title)
			}
		})
	})
}

func TestPageNode(t *testing.T) {
	var _, err = queries.GetQuerySet(&pages.PageNode{}).Delete()
	if err != nil {
		t.Fatalf("failed to delete existing PageNode records: %v", err)
	}

	var (
		rootNode = &pages.PageNode{
			Title: "Root",
			Slug:  "root",
		}
		childNode = &pages.PageNode{
			Title: "Child",
			Slug:  "child",
		}
		childSiblingNode = pages.PageNode{
			Title: "ChildSibling",
			Slug:  "childsibling",
		}
		subChildNode = pages.PageNode{
			Title: "SubChild",
			Slug:  "subchild",
		}
		subChildNode2 = pages.PageNode{
			Title: "SubChild2",
			Slug:  "subchild2",
		}
		childSiblingSubChildNode = pages.PageNode{
			Title: "ChildSiblingSubChild",
			Slug:  "childsiblingsubchild",
		}
		queryCtx = context.Background()
	)

	var (
		rootCreateCounter  = 0
		childCreateCounter = 0
		nodeUpdateCounter  = 0
		nodeDeleteCounter  = 0
	)

	pages.SignalNodeUpdated.Listen(func(s signals.Signal[*pages.PageNodeSignal], ps *pages.PageNodeSignal) error {
		nodeUpdateCounter++
		return nil
	})

	pages.SignalNodeBeforeDelete.Listen(func(s signals.Signal[*pages.PageNodeSignal], ps *pages.PageNodeSignal) error {
		nodeDeleteCounter++
		return nil
	})

	pages.SignalRootCreated.Listen(func(s signals.Signal[*pages.PageNodeSignal], ps *pages.PageNodeSignal) error {
		rootCreateCounter++
		return nil
	})

	pages.SignalChildCreated.Listen(func(s signals.Signal[*pages.PageNodeSignal], ps *pages.PageNodeSignal) error {
		childCreateCounter++
		return nil
	})

	var lastId = int64(0)

	var qs = pages.NewPageQuerySet().WithContext(queryCtx)

	t.Run("Root", func(t *testing.T) {
		var err = qs.AddRoot(rootNode)
		if err != nil {
			t.Fatal(err)
			return
		}

		if rootNode.PK == 0 {
			t.Fatal("expected ID not 0, got 0")
		}

		lastId = rootNode.PK

		if rootNode.Path != "001" {
			t.Fatalf("expected Path 1, got %s", rootNode.Path)
		}

		if rootNode.Depth != 0 {
			t.Fatalf("expected Depth 0, got %d", rootNode.Depth)
		}

		if rootNode.Numchild != 0 {
			t.Fatalf("expected Numchild 0, got %d", rootNode.Numchild)
		}

		if rootNode.UrlPath != "/root" {
			t.Fatalf("expected UrlPath /root, got %s", rootNode.UrlPath)
		}

		if rootNode.Slug != "root" {
			t.Fatalf("expected Slug root, got %s", rootNode.Slug)
		}

		if rootNode.StatusFlags != 0 {
			t.Fatalf("expected StatusFlagPublished, got %d", rootNode.StatusFlags)
		}

		if rootNode.PageID != 0 {
			t.Fatalf("expected PageID 0, got %d", rootNode.PageID)
		}

		if rootNode.ContentType != "" {
			t.Fatalf("expected ContentType empty, got %s", rootNode.ContentType)
		}

		t.Run("AddChild", func(t *testing.T) {

			t.Logf("Adding child node to root: %+v", childNode)

			var err = qs.CreateChildNode(rootNode, childNode)
			if err != nil {
				t.Fatal(err)
				return
			}

			if rootNode.Path != "001" {
				t.Fatalf("expected Path 001, got %s", rootNode.Path)
			}

			if childNode.PK != lastId+1 {
				t.Fatalf("expected ID %d, got %d", lastId+1, childNode.PK)
			}

			childNode, err = qs.GetNodeByID(childNode.PK)
			if err != nil {
				t.Fatal(err)
				return
			}

			if childNode.Path != "001001" {
				t.Fatalf("expected Path 001001, got %s", childNode.Path)
			}

			if childNode.Depth != 1 {
				t.Fatalf("expected Depth 1, got %d", childNode.Depth)
			}

			if childNode.Numchild != 0 {
				t.Fatalf("expected Numchild 0, got %d", childNode.Numchild)
			}

			if childNode.UrlPath != "/root/child" {
				t.Fatalf("expected UrlPath /root/child, got %s", childNode.UrlPath)
			}

			if childNode.Slug != "child" {
				t.Fatalf("expected Slug child, got %s", childNode.Slug)
			}

			if childNode.StatusFlags != 0 {
				t.Fatalf("expected StatusFlagPublished, got %d", childNode.StatusFlags)
			}

			if childNode.PageID != 0 {
				t.Fatalf("expected PageID 0, got %d", childNode.PageID)
			}

			if childNode.ContentType != "" {
				t.Fatalf("expected ContentType empty, got %s", childNode.ContentType)
			}

			if rootNode.Numchild != 1 {
				t.Fatalf("expected Numchild 1 for rootNode, got %d", rootNode.Numchild)
			}

			t.Run("GetChildren", func(t *testing.T) {

				t.Logf("Getting children of root node: %+v", rootNode)

				var children, err = qs.GetChildNodes(rootNode, pages.StatusFlagNone, 0, 1000)
				if err != nil {
					t.Fatal(err)
					return
				}

				if len(children) != 1 {
					t.Fatalf("expected 1 child for GetChildren, got %d", len(children))
					return
				}

				t.Logf("Found child node: %+v", children[0])

				if !nodesEqual(children[0], childNode) {
					t.Fatalf("expected %+v, got %+v (%+v)", childNode, children[0], nodeDiff(childNode, children[0]))
					return
				}
			})

			if t.Failed() {
				return
			}

			t.Run("AddSubChild", func(t *testing.T) {

				t.Logf("Adding sub-child node to child: %+v", subChildNode)

				var err = qs.CreateChildNode(childNode, &subChildNode)
				if err != nil {
					t.Fatal(err)
					return
				}

				if subChildNode.PK != lastId+2 {
					t.Fatalf("expected ID %d, got %d", lastId+2, subChildNode.PK)
				}

				if subChildNode.Path != "001001001" {
					t.Fatalf("expected Path 001001001, got %s", subChildNode.Path)
				}

				if subChildNode.Depth != 2 {
					t.Fatalf("expected Depth 2, got %d", subChildNode.Depth)
				}

				if subChildNode.Numchild != 0 {
					t.Fatalf("expected Numchild 0, got %d", subChildNode.Numchild)
				}

				if subChildNode.UrlPath != "/root/child/subchild" {
					t.Fatalf("expected UrlPath /root/child/subchild, got %s", subChildNode.UrlPath)
				}

				if subChildNode.Slug != "subchild" {
					t.Fatalf("expected Slug subchild, got %s", subChildNode.Slug)
				}

				if childNode.Numchild != 1 {
					t.Fatalf("expected Numchild 1, got %d", childNode.Numchild)
				}

				t.Run("GetAncestors", func(t *testing.T) {
					var ancestors, err = qs.AncestorNodes(subChildNode.Path, int(subChildNode.Depth)+1)
					if err != nil {
						t.Fatal(err)
						return
					}

					if len(ancestors) != 2 {
						t.Fatalf("expected 2 ancestors for GetAncestors, got %d", len(ancestors))
						return
					}

					if !nodesEqual(ancestors[0], rootNode) {
						t.Fatalf("expected %+v, got %+v", rootNode, ancestors[0])
						return
					}

					if !nodesEqual(ancestors[1], childNode) {
						t.Fatalf("expected %+v, got %+v", childNode, ancestors[1])
						return
					}
				})

				t.Run("GetRootDescendants", func(t *testing.T) {
					var descendants, err = qs.GetDescendants(rootNode.Path, 0, pages.StatusFlagNone, 0, 1000)
					if err != nil {
						t.Fatal(err)
						return
					}

					if len(descendants) != 2 {
						t.Fatalf("expected 2 descendants, got %d", len(descendants))
						return
					}

					if !nodesEqual(descendants[0], childNode) {
						t.Fatalf("expected %+v, got %+v", childNode, descendants[0])
						return
					}

					if !nodesEqual(descendants[1], &subChildNode) {
						t.Fatalf("expected %+v, got %+v", subChildNode, descendants[1])
						return
					}
				})

				t.Run("ParentNode", func(t *testing.T) {
					var parent, err = qs.ParentNode(subChildNode.Path, int(subChildNode.Depth))
					if err != nil {
						t.Fatal(err)
						return
					}

					if !nodesEqual(parent, childNode) {
						t.Fatalf("expected %+v, got %+v", childNode, parent)
					}
				})

				t.Run("DeleteNode", func(t *testing.T) {
					var err = qs.DeleteNode(&subChildNode)
					if err != nil {
						t.Fatal(err)
						return
					}

					descendants, err := qs.GetDescendants(rootNode.Path, 0, pages.StatusFlagNone, 0, 1000)
					if err != nil {
						t.Fatal(err)
						return
					}

					if len(descendants) != 1 {
						t.Fatalf("expected 1 descendant, got %d", len(descendants))
						return
					}

					childNode, err = qs.GetNodeByID(childNode.PK)
					if err != nil {
						t.Fatal(err)
						return
					}

					if !nodesEqual(descendants[0], childNode) {
						t.Fatalf("expected %+v, got %+v", childNode, descendants[0])
						return
					}

					if childNode.Numchild != 0 {
						t.Fatalf("expected Numchild 0, got %d", childNode.Numchild)
						return
					}
				})
			})
		})

		if t.Failed() {
			return
		}

		t.Run("AddSibling", func(t *testing.T) {
			var err = qs.CreateChildNode(rootNode, &childSiblingNode)
			if err != nil {
				t.Fatal(err)
				return
			}

			if childSiblingNode.PK != lastId+3 {
				t.Fatalf("expected ID %d, got %d", lastId+3, childSiblingNode.PK)
			}

			if childSiblingNode.Path != "001002" {
				t.Fatalf("expected Path 001002, got %s", childSiblingNode.Path)
			}

			if childSiblingNode.Depth != 1 {
				t.Fatalf("expected Depth 1, got %d", childSiblingNode.Depth)
			}

			if childSiblingNode.Numchild != 0 {
				t.Fatalf("expected Numchild 0, got %d", childSiblingNode.Numchild)
			}

			if childSiblingNode.StatusFlags != 0 {
				t.Fatalf("expected StatusFlagPublished, got %d", childSiblingNode.StatusFlags)
			}

			if childSiblingNode.PageID != 0 {
				t.Fatalf("expected PageID 0, got %d", childSiblingNode.PageID)
			}

			if childSiblingNode.ContentType != "" {
				t.Fatalf("expected ContentType empty, got %s", childSiblingNode.ContentType)
			}

			if childSiblingNode.UrlPath != "/root/childsibling" {
				t.Fatalf("expected UrlPath /root/childsibling, got %s", childSiblingNode.UrlPath)
			}

			if rootNode.Numchild != 2 {
				t.Fatalf("expected Numchild 2, got %d", rootNode.Numchild)
			}

			t.Run("GetChildren", func(t *testing.T) {
				var children, err = qs.GetChildNodes(rootNode, pages.StatusFlagNone, 0, 1000)
				if err != nil {
					t.Fatal(err)
					return
				}

				if len(children) != 2 {
					t.Fatalf("expected 2 children, got %d", len(children))
					return
				}

				if !nodesEqual(children[1], &childSiblingNode) {
					t.Fatalf("expected %+v, got %+v", childSiblingNode, children[1])
					return
				}

				if childSiblingNode.Path != "001002" {
					t.Fatalf("expected Path 001002, got %s", childSiblingNode.Path)
					return
				}
			})

			t.Run("AddSubChild", func(t *testing.T) {
				var err = qs.CreateChildNode(&childSiblingNode, &childSiblingSubChildNode)
				if err != nil {
					t.Fatal(err)
					return
				}

				if childSiblingNode.Path != "001002" {
					t.Fatalf("expected Path 001002, got %s", childSiblingNode.Path)
					return
				}

				if childSiblingSubChildNode.PK != lastId+4 {
					t.Fatalf("expected ID %d, got %d", lastId+4, childSiblingSubChildNode.PK)
				}

				if childSiblingSubChildNode.Path != "001002001" {
					t.Fatalf("expected Path 001002001, got %s", childSiblingSubChildNode.Path)
				}

				if childSiblingSubChildNode.Depth != 2 {
					t.Fatalf("expected Depth 2, got %d", childSiblingSubChildNode.Depth)
				}

				if childSiblingSubChildNode.Numchild != 0 {
					t.Fatalf("expected Numchild 0, got %d", childSiblingSubChildNode.Numchild)
				}

				if childSiblingNode.Numchild != 1 {
					t.Fatalf("expected Numchild 1, got %d", childSiblingNode.Numchild)
				}

				t.Run("GetAncestors", func(t *testing.T) {
					var ancestors, err = qs.AncestorNodes(childSiblingSubChildNode.Path, int(childSiblingSubChildNode.Depth)+1)
					if err != nil {
						t.Fatal(err)
						return
					}

					if len(ancestors) != 2 {
						t.Fatalf("expected 2 ancestors, got %d", len(ancestors))
						return
					}

					if !nodesEqual(ancestors[0], rootNode) {
						t.Fatalf("expected %+v, got %+v", rootNode, ancestors[0])
						return
					}

					if !nodesEqual(ancestors[1], &childSiblingNode) {
						t.Fatalf("expected %+v, got %+v", childSiblingNode, ancestors[1])
						return
					}

					if childSiblingNode.Path != "001002" {
						t.Fatalf("expected Path 001002, got %s", childSiblingNode.Path)
						return
					}
				})

				if childSiblingNode.Path != "001002" {
					t.Fatalf("expected Path 001002, got %s", childSiblingNode.Path)
				}

				t.Run("GetRootDescendants", func(t *testing.T) {
					var descendants, err = qs.GetDescendants(rootNode.Path, 0, pages.StatusFlagNone, 0, 1000)
					if err != nil {
						t.Fatal(err)
						return
					}

					if len(descendants) != 3 {
						t.Fatalf("expected 3 descendants, got %d", len(descendants))
						return
					}

					if !nodesEqual(descendants[1], &childSiblingNode) {
						t.Fatalf("expected %+v, got %+v", childSiblingNode, descendants[1])
						return
					}

					if !nodesEqual(descendants[2], &childSiblingSubChildNode) {
						t.Fatalf("expected %+v, got %+v", childSiblingSubChildNode, descendants[2])
						return
					}

					if childSiblingNode.Path != "001002" {
						t.Fatalf("expected Path 001002, got %s", childSiblingNode.Path)
						return
					}
				})

				if childSiblingNode.Path != "001002" {
					t.Fatalf("expected Path 001002, got %s", childSiblingNode.Path)
				}

				t.Run("CheckSignals", func(t *testing.T) {
					if rootCreateCounter != 1 {
						t.Fatalf("expected 1, got %d", rootCreateCounter)
					}

					if childCreateCounter != 4 {
						t.Fatalf("expected 4, got %d", childCreateCounter)
					}

					if nodeUpdateCounter != 0 {
						t.Fatalf("expected 0, got %d", nodeUpdateCounter)
					}

					if nodeDeleteCounter != 1 {
						t.Fatalf("expected 1, got %d", nodeDeleteCounter)
					}

					if childSiblingNode.Path != "001002" {
						t.Fatalf("expected Path 001002, got %s", childSiblingNode.Path)
					}
				})

				if childSiblingNode.Path != "001002" {
					t.Fatalf("expected Path 001002, got %s", childSiblingNode.Path)
				}

				t.Run("IncNumChild", func(t *testing.T) {
					var node, err = incrementNumChild(qs, childSiblingNode.PK)
					if err != nil {
						t.Fatal(err)
						return
					}

					if node.Numchild != 2 {
						t.Fatalf("expected Numchild 2, got %d", node.Numchild)
					}

					if node.Title != "ChildSibling" {
						t.Fatalf("expected Title ChildSibling, got %s", node.Title)
					}

					if node.Path != "001002" {
						t.Fatalf("expected Path 001002, got %s", node.Path)
						return
					}

					if childSiblingNode.Path != "001002" {
						t.Fatalf("expected Path 001002, got %s", childSiblingNode.Path)
					}
				})

				if childSiblingNode.Path != "001002" {
					t.Fatalf("expected Path 001002, got %s", childSiblingNode.Path)
				}

				t.Run("DecNumChild", func(t *testing.T) {
					var node, err = decrementNumChild(qs, childSiblingNode.PK)
					if err != nil {
						t.Fatal(err)
						return
					}

					if node.Numchild != 1 {
						t.Fatalf("expected Numchild 1, got %d", node.Numchild)
					}

					if node.Title != "ChildSibling" {
						t.Fatalf("expected Title ChildSibling, got %s", node.Title)
					}

					if node.Path != "001002" {
						t.Fatalf("expected Path 001002, got %s", node.Path)
						return
					}

					if childSiblingNode.Path != "001002" {
						t.Fatalf("expected Path 001002, got %s", childSiblingNode.Path)
					}
				})

				if childSiblingNode.Path != "001002" {
					t.Fatalf("expected Path 001002, got %s", childSiblingNode.Path)
					return
				}

			})

			if childSiblingNode.Path != "001002" {
				t.Fatalf("expected Path 001002, got %s", childSiblingNode.Path)
				return
			}

		})

		if childSiblingNode.Path != "001002" {
			t.Fatalf("expected Path 001002, got %s", childSiblingNode.Path)
			return
		}
	})

	switch {
	case rootNode.Path == "":
		t.Fatalf("expected Path not empty, got %q for Page(%d): %s", rootNode.Path, rootNode.ID(), rootNode.Title)
		return
	case childNode.Path == "":
		t.Fatalf("expected Path not empty, got %q for Page(%d): %s", childNode.Path, childNode.ID(), childNode.Title)
		return
	case childSiblingNode.Path == "":
		t.Fatalf("expected Path not empty, got %q for Page(%d): %s", childSiblingNode.Path, childSiblingNode.ID(), childSiblingNode.Title)
		return
	case subChildNode.Path == "":
		t.Fatalf("expected Path not empty, got %q for Page(%d): %s", subChildNode.Path, subChildNode.ID(), subChildNode.Title)
		return
	}

	if childSiblingNode.Path != "001002" {
		t.Fatalf("expected Path 001002, got %s", childSiblingNode.Path)
		return
	}

	t.Run("TraverseSlugs", func(t *testing.T) {
		type slugsTest struct {
			slug  string
			path  string
			depth int64

			expectedPath string
			shouldEqual  *pages.PageNode
		}

		var slugsTests = []slugsTest{
			{"root", "", 0, "001", rootNode},
			{"childsibling", "001", 1, "001002", &childSiblingNode},
			{"childsiblingsubchild", "001002", 2, "001002001", &childSiblingSubChildNode},
		}

		for _, test := range slugsTests {
			t.Run(fmt.Sprintf("Traverse-Slug-%s", test.slug), func(t *testing.T) {
				var node, err = qs.GetNodeBySlug(test.slug, test.depth, test.path)
				if err != nil {
					t.Fatalf("expected no error, got %v (%s %s)", err, test.slug, test.path)
					return
				}

				if node.Path != test.expectedPath {
					t.Fatalf("expected Path %s, got %s", test.expectedPath, node.Path)
				}

				if node.Depth != int64(test.depth) {
					t.Fatalf("expected Depth %d, got %d", test.depth, node.Depth)
				}

				if !nodesEqual(node, test.shouldEqual) {
					t.Fatalf("expected %+v, got %+v", test.shouldEqual, node)
				}
			})
		}
	})

	var nodesToUpdate = []*pages.PageNode{
		{Title: "Root 1"},
		{Title: "Root 2"},
		{Title: "Root 3"},
	}

	for _, node := range nodesToUpdate {
		if err := qs.AddRoot(node); err != nil {
			t.Fatal(err)
		}
		if len(node.Path) != pages.STEP_LEN {
			t.Fatalf("expected Path of length %d, got %d", pages.STEP_LEN, len(node.Path))
		}
	}

	for i, node := range nodesToUpdate {
		node.Title = fmt.Sprintf("Root %d Updated", i+1)
	}

	t.Run("UpdateNodes", func(t *testing.T) {
		var err = updateNodes(qs, nodesToUpdate)
		if err != nil {
			t.Fatal(err)
			return
		}

		for _, node := range nodesToUpdate {
			var updatedNode, err = qs.GetNodeByID(node.PK)
			if err != nil {
				t.Fatal(err)
				return
			}

			if updatedNode.Title != node.Title {
				t.Fatalf("expected %s, got %s", node.Title, updatedNode.Title)
			}
		}
	})

	t.Run("CheckNodesUpdated", func(t *testing.T) {
		for i, node := range nodesToUpdate {
			if node.Title != fmt.Sprintf("Root %d Updated", i+1) {
				t.Fatalf("expected Root %d Updated, got %s", i+1, node.Title)
			}

			if node.UrlPath != fmt.Sprintf("/root-%d", i+1) {
				t.Fatalf("expected /root-%d, got %s", i+1, node.UrlPath)
			}
		}
	})

	t.Run("MoveNode", func(t *testing.T) {

		if err := qs.CreateChildNode(childNode, &subChildNode2); err != nil {
			t.Fatal(err)
			return
		}

		var err = qs.MoveNode(childNode, nodesToUpdate[0])
		if err != nil {
			t.Fatal(err)
			return
		}

		sub, err := qs.GetNodeByID(childNode.PK)
		if err != nil {
			t.Fatal(err)
			return
		}

		if sub.Path != "002001" {
			t.Fatalf("expected Path 002001, got %s", sub.Path)
		}

		if sub.Depth != 1 {
			t.Fatalf("expected Depth 1, got %d", sub.Depth)
		}

		if sub.UrlPath != "/root-1/child" {
			t.Fatalf("expected UrlPath /root-1/child, got %s", sub.UrlPath)
		}

		if sub.Numchild != 1 {
			t.Fatalf("expected Numchild 1, got %d", sub.Numchild)
		}

		childNode = sub

		t.Run("CheckMovedNodeChild", func(t *testing.T) {
			subSub, err := qs.GetNodeByID(subChildNode2.PK)
			if err != nil {
				t.Fatal(err)
				return
			}

			if subSub.Path != "002001001" {
				t.Fatalf("expected Path 002001001, got %s", subSub.Path)
			}

			if subSub.Depth != 2 {
				t.Fatalf("expected Depth 2, got %d", subSub.Depth)
			}

			if subSub.UrlPath != "/root-1/child/subchild2" {
				t.Fatalf("expected UrlPath /root-1/child/subchild2, got %s", subSub.UrlPath)
			}

			if sub.Numchild != 1 {
				t.Fatalf("expected Numchild 1, got %d", sub.Numchild)
			}

			parentNode, err := qs.ParentNode(subSub.Path, int(subSub.Depth))
			if err != nil {
				t.Fatal(err)
				return
			}

			if !nodesEqual(parentNode, sub) {
				t.Fatalf("expected %+v, got %+v", sub, parentNode)
			}
		})
	})

	pages.Register(&pages.PageDefinition{
		ContentTypeDefinition: &contenttypes.ContentTypeDefinition{
			ContentObject: &DBTestPage{},
			GetLabel:      trans.S(""),
		},
		GetForID: func(ctx context.Context, ref *pages.PageNode, id int64) (pages.Page, error) {
			var page = &DBTestPage{}
			page.Ref = ref
			var row, err = queries.GetQuerySet(&DBTestPage{}).Filter("Identifier", id).First()
			return row.Object, err
		},
	})

	var createAndBind = func(t *testing.T, node *pages.PageNode, page *DBTestPage) *DBTestPage {

		if node.Path == "" {
			t.Fatalf("node.Path is empty, cannot create page: %+v", node)
		}

		page.Ref = node
		if err := page.Save(queryCtx); err != nil {
			t.Fatalf("failed to save page: %v", err)
		}
		node.PageID = page.ID()
		node.ContentType = pages.DefinitionForObject(page).ContentType().TypeName()

		t.Logf("Created page %s for %s (%d)", node.Title, page.Description, page.Reference().PK)
		t.Logf("REFERENCE: %+v", page.Reference())
		t.Logf("NODE: %+v", node)

		if err := qs.UpdateNode(page.Reference()); err != nil {
			t.Fatalf("failed to update node: %v", err)
		}

		return page
	}

	var (
		dbTestPage_Root = createAndBind(t, rootNode, &DBTestPage{
			TestPage: TestPage{
				Description: "Root Description",
			},
		})
		dbTestPage_Child = createAndBind(t, childNode, &DBTestPage{
			TestPage: TestPage{
				Description: "Child Description",
			},
		})
		dbTestPage_ChildSibling = createAndBind(t, &childSiblingNode, &DBTestPage{
			TestPage: TestPage{
				Description: "ChildSibling Description",
			},
		})
		dbTestPage_ChildSiblingSubChild = createAndBind(t, &childSiblingSubChildNode, &DBTestPage{
			TestPage: TestPage{
				Description: "ChildSiblingSubChild Description",
			},
		})
	)

	var pageList = []pages.Page{
		dbTestPage_Root,
		dbTestPage_Child,
		dbTestPage_ChildSibling,
		dbTestPage_ChildSiblingSubChild,
	}

	for _, page := range pageList {
		t.Run(fmt.Sprintf("Specific_Page_%s", page.Reference().Title), func(t *testing.T) {
			var instance, err = page.Reference().Specific(queryCtx)
			if err != nil {
				t.Fatal(err)
				return
			}

			var dbPage, ok = instance.(*DBTestPage)
			if !ok {
				t.Fatalf("expected *DBTestPage, got %T", instance)
				return
			}

			if dbPage.Description != page.(*DBTestPage).Description {
				t.Fatalf("expected %s, got %s", page.(*DBTestPage).Description, dbPage.Description)
			}
		})
	}
}
