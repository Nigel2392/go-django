package pages_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	_ "unsafe"

	"github.com/Nigel2392/django/contrib/pages"
	"github.com/Nigel2392/django/contrib/pages/models"
)

//go:linkname pageMenuHandler github.com/Nigel2392/django/contrib/pages.pageMenuHandler
func pageMenuHandler(w http.ResponseWriter, r *http.Request)

var nodes = []models.PageNode{
	{ID: 1, Title: "Home", Path: "001", StatusFlags: models.StatusFlagPublished},
	{ID: 2, Title: "About", Path: "001001", StatusFlags: models.StatusFlagPublished},
	{ID: 3, Title: "Contact", Path: "001002", StatusFlags: models.StatusFlagPublished},
	{ID: 4, Title: "Services", Path: "001003", StatusFlags: models.StatusFlagPublished},
	{ID: 5, Title: "Team", Path: "001004", StatusFlags: models.StatusFlagPublished},
	{ID: 6, Title: "History", Path: "001001001", StatusFlags: models.StatusFlagPublished},
	{ID: 7, Title: "Vision", Path: "001001002", StatusFlags: models.StatusFlagPublished},
	{ID: 8, Title: "Mission", Path: "001001003", StatusFlags: models.StatusFlagPublished},
	{ID: 9, Title: "Contact Us", Path: "001002001", StatusFlags: models.StatusFlagPublished},
	{ID: 10, Title: "Services Offered", Path: "001003001", StatusFlags: models.StatusFlagPublished},
	{ID: 11, Title: "Our Team", Path: "001004001", StatusFlags: models.StatusFlagPublished},
	{ID: 12, Title: "Our History", Path: "001001001001", StatusFlags: models.StatusFlagPublished},
	{ID: 13, Title: "Our Vision", Path: "001001002001", StatusFlags: models.StatusFlagPublished},
	{ID: 14, Title: "Our Mission", Path: "001001003001", StatusFlags: models.StatusFlagPublished},
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
	expectedItems []models.PageNode
}

type menuResponse struct {
	ParentItem *models.PageNode  `json:"parent_item,omitempty"`
	Items      []models.PageNode `json:"items"`
}

var pageMenuHandlerTests = []pageMenuHandlerTest{
	{
		mainItemID: "1",
		expectedItems: []models.PageNode{
			{ID: 2, Title: "About", Path: "001001", StatusFlags: models.StatusFlagPublished},
			{ID: 3, Title: "Contact", Path: "001002", StatusFlags: models.StatusFlagPublished},
			{ID: 4, Title: "Services", Path: "001003", StatusFlags: models.StatusFlagPublished},
			{ID: 5, Title: "Team", Path: "001004", StatusFlags: models.StatusFlagPublished},
		},
	},
	{
		mainItemID: "2",
		getParent:  "true",
		expectedItems: []models.PageNode{
			{ID: 2, Title: "About", Path: "001001", StatusFlags: models.StatusFlagPublished},
			{ID: 3, Title: "Contact", Path: "001002", StatusFlags: models.StatusFlagPublished},
			{ID: 4, Title: "Services", Path: "001003", StatusFlags: models.StatusFlagPublished},
			{ID: 5, Title: "Team", Path: "001004", StatusFlags: models.StatusFlagPublished},
		},
	},
	{
		mainItemID: "2",
		expectedItems: []models.PageNode{
			{ID: 6, Title: "History", Path: "001001001", StatusFlags: models.StatusFlagPublished},
			{ID: 7, Title: "Vision", Path: "001001002", StatusFlags: models.StatusFlagPublished},
			{ID: 8, Title: "Mission", Path: "001001003", StatusFlags: models.StatusFlagPublished},
		},
	},
	{
		mainItemID: "3",
		expectedItems: []models.PageNode{
			{ID: 9, Title: "Contact Us", Path: "001002001", StatusFlags: models.StatusFlagPublished},
		},
	},
	{
		mainItemID: "4",
		expectedItems: []models.PageNode{
			{ID: 10, Title: "Services Offered", Path: "001003001", StatusFlags: models.StatusFlagPublished},
		},
	},
	{
		mainItemID: "5",
		expectedItems: []models.PageNode{
			{ID: 11, Title: "Our Team", Path: "001004001", StatusFlags: models.StatusFlagPublished},
		},
	},
	{
		mainItemID: "6",
		expectedItems: []models.PageNode{
			{ID: 12, Title: "Our History", Path: "001001001001", StatusFlags: models.StatusFlagPublished},
		},
	},
	{
		mainItemID: "6",
		getParent:  "true",
		expectedItems: []models.PageNode{
			{ID: 6, Title: "History", Path: "001001001", StatusFlags: models.StatusFlagPublished},
			{ID: 7, Title: "Vision", Path: "001001002", StatusFlags: models.StatusFlagPublished},
			{ID: 8, Title: "Mission", Path: "001001003", StatusFlags: models.StatusFlagPublished},
		},
	},
}

func TestPageMenuHandler(t *testing.T) {
	var err error
	sqlDB, err = sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		panic(err)
	}

	// Create test_pages table
	if err := sqlDB.Ping(); err != nil {
		panic(err)
	}

	var driverType = sqlDB.Driver()
	var backend, ok = models.GetBackend(driverType)
	if !ok {
		panic(fmt.Sprintf("no backend configured for %T", driverType))
	}

	if err := backend.CreateTable(sqlDB); err != nil {
		panic(err)
	}

	qs, err := backend.NewQuerySet(sqlDB)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize queryset for backend %T", backend))
	}

	pages.QuerySet = func() models.DBQuerier {
		return &pages.Querier{
			Querier: qs,
			Db:      sqlDB,
		}
	}

	// Insert test data
	var ctx = context.Background()
	for _, node := range nodes {
		if _, err := qs.InsertNode(ctx, node.Title, node.Path, node.Depth, node.Numchild, node.UrlPath, int64(node.StatusFlags), node.PageID, node.ContentType); err != nil {
			panic(err)
		}
	}

	allNodes, err := qs.AllNodes(ctx, 1000, 0)
	if err != nil {
		panic(err)
	}

	var nodeRefs = make([]*models.PageNode, len(allNodes))
	for i := 0; i < len(allNodes); i++ {
		nodeRefs[i] = &allNodes[i]
	}

	var tree = pages.NewNodeTree(nodeRefs)

	tree.FixTree()

	err = qs.UpdateNodes(ctx, nodeRefs)
	if err != nil {
		panic(err)
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
			r = r.WithContext(ctx)
			pageMenuHandler(w, r)

			//if w.s != http.StatusOK {
			//	t.Errorf("expected status %d; got %d", http.StatusOK, w.s)
			//	return
			//}

			var gotItems menuResponse
			if err := json.Unmarshal(w.b.Bytes(), &gotItems); err != nil {
				t.Errorf("failed to unmarshal response: %v (%s)", err, w.b.String())
				return
			}

			if len(gotItems.Items) != len(test.expectedItems) {
				t.Errorf("expected %d items; got %d", len(test.expectedItems), len(gotItems.Items))
			}

			for i, item := range gotItems.Items {
				if item.ID != test.expectedItems[i].ID {
					t.Errorf("expected item ID %d; got %d", test.expectedItems[i].ID, item.ID)
				}
			}
		})
	}

	sqlDB.Exec("DROP TABLE PageNode")
}
