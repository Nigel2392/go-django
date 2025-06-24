package links_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/editor"
	_ "github.com/Nigel2392/go-django/src/contrib/editor/features/links"
	"github.com/Nigel2392/go-django/src/contrib/pages"
	"github.com/Nigel2392/go-django/src/djester/testdb"
	"github.com/PuerkitoBio/goquery"
	_ "github.com/mattn/go-sqlite3"
)

func TestViews(t *testing.T) {

	var _, db = testdb.Open()
	var app = django.App(
		django.Configure(map[string]interface{}{
			django.APPVAR_ALLOWED_HOSTS: []string{"*"},
			django.APPVAR_DEBUG:         false,
			django.APPVAR_DATABASE:      db,
		}),
		django.Apps(
			pages.NewAppConfig,
			editor.NewAppConfig,
		),
		django.Flag(
			django.FlagSkipCmds,
			django.FlagSkipDepsCheck,
		),
	)

	pages.SetRoutePrefix("/pages")

	var err = app.Initialize()
	if err != nil {
		t.Fatal(err)
	}

	var server = httptest.NewServer(app.Mux)
	defer server.Close()

	var page = &pages.PageNode{
		Title: "Google",
	}
	err = pages.CreateRootNode(
		context.Background(), page,
	)
	if err != nil {
		t.Fatal(err)
	}

	if page.ID() == 0 {
		t.Fatal("Page ID is 0")
	}

	var editorData = fmt.Sprintf(
		`{"time":1600000000,"blocks":[{"type":"paragraph","data":{"text":"Hello <a class=\"page-link\" data-page-id=\"%v\">Google</a>"}}],"version":"2.19.0"}`,
		page.ID(),
	)

	editorJSBlockData, err := editor.JSONUnmarshalEditorData(
		[]string{"pagelink", "paragraph"},
		[]byte(editorData),
	)
	if err != nil {
		t.Fatal(err)
	}

	rendered, err := editorJSBlockData.Render()
	if err != nil {
		t.Fatal(err)
	}

	htmlNode, err := goquery.NewDocumentFromReader(
		strings.NewReader(string(rendered)),
	)
	if err != nil {
		t.Fatal(err)
	}

	var links = htmlNode.Find("a.page-link")
	if links.Length() != 1 {
		t.Fatalf("Expected 1 link, got %d", links.Length())
	}

	var href, ok = links.Attr("href")
	if !ok {
		t.Fatal("Link has no href attribute")
	}

	if strings.Trim(href, "/") != fmt.Sprintf("pages/%v", page.Slug) {
		t.Fatalf("Expected href to be pages/%s, got %s", page.Slug, href)
	}
}
