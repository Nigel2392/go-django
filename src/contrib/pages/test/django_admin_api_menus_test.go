package pages_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	_ "unsafe"

	"github.com/Nigel2392/go-django/src/contrib/pages"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware/authentication"
	"github.com/pkg/errors"
)

//go:linkname pageMenuHandler github.com/Nigel2392/go-django/src/contrib/pages.pageMenuHandler
func pageMenuHandler(w http.ResponseWriter, r *http.Request)

var nodes = []*pages.PageNode{
	{PK: 1, Title: "Home", Path: "001", StatusFlags: pages.StatusFlagPublished},
	{PK: 2, Title: "About", Path: "001001", StatusFlags: pages.StatusFlagPublished},
	{PK: 3, Title: "Contact", Path: "001002", StatusFlags: pages.StatusFlagPublished},
	{PK: 4, Title: "Services", Path: "001003", StatusFlags: pages.StatusFlagPublished},
	{PK: 5, Title: "Team", Path: "001004", StatusFlags: pages.StatusFlagPublished},
	{PK: 6, Title: "History", Path: "001001001", StatusFlags: pages.StatusFlagPublished},
	{PK: 7, Title: "Vision", Path: "001001002", StatusFlags: pages.StatusFlagPublished},
	{PK: 8, Title: "Mission", Path: "001001003", StatusFlags: pages.StatusFlagPublished},
	{PK: 9, Title: "Contact Us", Path: "001002001", StatusFlags: pages.StatusFlagPublished},
	{PK: 10, Title: "Services Offered", Path: "001003001", StatusFlags: pages.StatusFlagPublished},
	{PK: 11, Title: "Our Team", Path: "001004001", StatusFlags: pages.StatusFlagPublished},
	{PK: 12, Title: "Our History", Path: "001001001001", StatusFlags: pages.StatusFlagPublished},
	{PK: 13, Title: "Our Vision", Path: "001001002001", StatusFlags: pages.StatusFlagPublished},
	{PK: 14, Title: "Our Mission", Path: "001001003001", StatusFlags: pages.StatusFlagPublished},
}

type fakeHttpWriter struct {
	b bytes.Buffer
	h http.Header
	s int
}

func (f *fakeHttpWriter) Header() http.Header {
	return f.h
}

func (f *fakeHttpWriter) Write(b []byte) (int, error) {
	return f.b.Write(b)
}

func (f *fakeHttpWriter) WriteHeader(s int) {
	f.s = s
}

type pageMenuHandlerTest struct {
	mainItemID    string
	getParent     string
	expectedItems []pages.PageNode
}

type menuResponse struct {
	ParentItem *pages.PageNode  `json:"parent_item,omitempty"`
	Items      []pages.PageNode `json:"items"`
}

var pageMenuHandlerTests = []pageMenuHandlerTest{
	{
		mainItemID: "1",
		expectedItems: []pages.PageNode{
			{PK: 2, Title: "About", Path: "001001", StatusFlags: pages.StatusFlagPublished},
			{PK: 3, Title: "Contact", Path: "001002", StatusFlags: pages.StatusFlagPublished},
			{PK: 4, Title: "Services", Path: "001003", StatusFlags: pages.StatusFlagPublished},
			{PK: 5, Title: "Team", Path: "001004", StatusFlags: pages.StatusFlagPublished},
		},
	},
	{
		mainItemID: "2",
		getParent:  "true",
		expectedItems: []pages.PageNode{
			{PK: 2, Title: "About", Path: "001001", StatusFlags: pages.StatusFlagPublished},
			{PK: 3, Title: "Contact", Path: "001002", StatusFlags: pages.StatusFlagPublished},
			{PK: 4, Title: "Services", Path: "001003", StatusFlags: pages.StatusFlagPublished},
			{PK: 5, Title: "Team", Path: "001004", StatusFlags: pages.StatusFlagPublished},
		},
	},
	{
		mainItemID: "2",
		expectedItems: []pages.PageNode{
			{PK: 6, Title: "History", Path: "001001001", StatusFlags: pages.StatusFlagPublished},
			{PK: 7, Title: "Vision", Path: "001001002", StatusFlags: pages.StatusFlagPublished},
			{PK: 8, Title: "Mission", Path: "001001003", StatusFlags: pages.StatusFlagPublished},
		},
	},
	{
		mainItemID: "3",
		expectedItems: []pages.PageNode{
			{PK: 9, Title: "Contact Us", Path: "001002001", StatusFlags: pages.StatusFlagPublished},
		},
	},
	{
		mainItemID: "4",
		expectedItems: []pages.PageNode{
			{PK: 10, Title: "Services Offered", Path: "001003001", StatusFlags: pages.StatusFlagPublished},
		},
	},
	{
		mainItemID: "5",
		expectedItems: []pages.PageNode{
			{PK: 11, Title: "Our Team", Path: "001004001", StatusFlags: pages.StatusFlagPublished},
		},
	},
	{
		mainItemID: "6",
		expectedItems: []pages.PageNode{
			{PK: 12, Title: "Our History", Path: "001001001001", StatusFlags: pages.StatusFlagPublished},
		},
	},
	{
		mainItemID: "6",
		getParent:  "true",
		expectedItems: []pages.PageNode{
			{PK: 6, Title: "History", Path: "001001001", StatusFlags: pages.StatusFlagPublished},
			{PK: 7, Title: "Vision", Path: "001001002", StatusFlags: pages.StatusFlagPublished},
			{PK: 8, Title: "Mission", Path: "001001003", StatusFlags: pages.StatusFlagPublished},
		},
	},
}

type DummyUser struct {
	IsAdministrator bool
}

func (u *DummyUser) IsAuthenticated() bool {
	return true
}

func (u *DummyUser) IsAdmin() bool {
	return u.IsAdministrator
}

func TestPageMenuHandler(t *testing.T) {

	// Insert test data
	var qs = pages.NewPageQuerySet()
	for i, node := range nodes {

		if node.Depth > 0 {
			var parentNode, err = qs.ParentNode(node.Path, int(node.Depth))
			if err != nil {
				panic(err)
			}

			node.SetUrlPath(parentNode)
		} else {
			node.SetUrlPath(nil)
		}

		if _, err := insertNode(qs, node); err != nil {
			panic(errors.Wrapf(
				err, "failed to insert node [%d/%d] %s",
				i, len(nodes), node.Title,
			))
		}
	}

	allNodes, err := qs.AllNodes(pages.StatusFlagNone, 0, 1000)
	if err != nil {
		t.Fatalf("failed to retrieve all nodes: %v", err)
	}

	var nodeRefs = make([]*pages.PageNode, len(allNodes))
	copy(nodeRefs, allNodes)

	var tree = pages.NewNodeTree(nodeRefs)

	tree.FixTree()

	err = updateNodes(qs, nodeRefs)
	if err != nil {
		t.Fatalf("failed to update nodes: %v", err)
	}

	for _, test := range pageMenuHandlerTests {
		test := test
		t.Run(test.mainItemID, func(t *testing.T) {
			var w = &fakeHttpWriter{h: make(http.Header)}
			var r, _ = http.NewRequest(http.MethodGet, "/admin/api/pages/menu", nil)
			var q = r.URL.Query()
			q.Set("page_id", test.mainItemID)
			if test.getParent != "" {
				q.Set("get_parent", test.getParent)
			}
			r.URL.RawQuery = q.Encode()
			r = r.WithContext(qs.Context())
			var m = authentication.AddUserMiddleware(func(r *http.Request) authentication.User {
				return &DummyUser{IsAdministrator: true}
			})
			var handler = m(mux.NewHandler(pageMenuHandler))
			handler.ServeHTTP(w, r)

			// if w.s != http.StatusOK {
			// t.Errorf("expected status %d; got %d", http.StatusOK, w.s)
			// return
			// }

			var gotItems menuResponse
			if err := json.Unmarshal(w.b.Bytes(), &gotItems); err != nil {
				t.Errorf("failed to unmarshal response: %v (%s)", err, w.b.String())
				return
			}

			if len(gotItems.Items) != len(test.expectedItems) {
				t.Errorf("expected %d items; got %d", len(test.expectedItems), len(gotItems.Items))
			}

			for i, item := range gotItems.Items {
				if item.PK != test.expectedItems[i].PK {
					t.Errorf("expected item ID %d; got %d", test.expectedItems[i].PK, item.PK)
				}
			}
		})
	}
}
