package models_test

import (
	"context"
	"os"
	"testing"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/fields"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/queries/src/quest"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/logger"
)

const (
	// sqlite3 memory database for testing
	TABLE_PAGE               = "CREATE TABLE IF NOT EXISTS page (id INTEGER PRIMARY KEY, title TEXT, description TEXT, page_id INTEGER, page_content_type TEXT);"
	TABLE_BLOG_PAGE          = "CREATE TABLE IF NOT EXISTS blog_page (page_id INTEGER PRIMARY KEY, author TEXT, tags TEXT, category TEXT, category_content_type TEXT);"
	TABLE_BLOG_PAGE_CATEGORY = "CREATE TABLE IF NOT EXISTS blog_page_category (category TEXT PRIMARY KEY, page_id INTEGER, category_content_type TEXT);"
)

func init() {
	attrs.RegisterModel(&Page{})
	attrs.RegisterModel(&BlogPage{})
	attrs.RegisterModel(&BlogPageCategory{})

	var db, err = drivers.Open(context.Background(), "sqlite3", "file:queries_memory?mode=memory&cache=shared")
	if err != nil {
		panic(err)
	}
	var settings = map[string]interface{}{
		django.APPVAR_DATABASE: db,
	}

	logger.Setup(&logger.Logger{
		Level:       logger.DBG,
		WrapPrefix:  logger.ColoredLogWrapper,
		OutputDebug: os.Stdout,
		OutputInfo:  os.Stdout,
		OutputWarn:  os.Stdout,
		OutputError: os.Stdout,
	})

	django.App(django.Configure(settings),
		django.Flag(django.FlagSkipDepsCheck),
	)

	quest.Table[*testing.T](nil, &Page{}, &BlogPage{}, &BlogPageCategory{}).Create()
}

type BasePage struct {
	models.Model
	ID int64
}

type Page struct {
	BasePage
	PageID          int64
	PageContentType *contenttypes.BaseContentType[attrs.Definer]
	Title           string
	Description     string
}

func (p *Page) TargetContentTypeField() attrs.FieldDefinition {
	var defs = p.FieldDefs()
	var f, _ = defs.Field("PageContentType")
	return f
}

func (p *Page) TargetPrimaryField() attrs.FieldDefinition {
	var defs = p.FieldDefs()
	var f, _ = defs.Field("PageID")
	return f
}

func (p *Page) FieldDefs() attrs.Definitions {
	return p.Model.Define(p,
		attrs.Unbound("ID", &attrs.FieldConfig{Primary: true}),
		attrs.Unbound("Title"),
		attrs.Unbound("Description"),
		attrs.Unbound("PageID"),
		attrs.Unbound("PageContentType"),
	)
}

type BlogPage struct {
	models.Model
	Proxy               *Page `proxy:"true"`
	PageID              int64
	Category            string
	CategoryContentType *contenttypes.BaseContentType[attrs.Definer]
	Author              string
	Tags                []string
}

func (b *BlogPage) TargetContentTypeField() attrs.FieldDefinition {
	var defs = b.FieldDefs()
	var f, _ = defs.Field("CategoryContentType")
	return f
}

func (b *BlogPage) TargetPrimaryField() attrs.FieldDefinition {
	var defs = b.FieldDefs()
	var f, _ = defs.Field("Category")
	return f
}

func (b *BlogPage) FieldDefs() attrs.Definitions {
	return b.Model.Define(b,
		fields.Embed("Proxy"),
		attrs.Unbound("PageID", &attrs.FieldConfig{Primary: true}),
		attrs.Unbound("Author"),
		attrs.Unbound("Tags"),
		attrs.Unbound("Category"),
		attrs.Unbound("CategoryContentType"),
	)
}

type BlogPageCategory struct {
	models.Model
	*BlogPage
	Category string
}

func (b *BlogPageCategory) FieldDefs() attrs.Definitions {
	return b.Model.Define(b,
		fields.Embed("BlogPage"),
		attrs.Unbound("Category", &attrs.FieldConfig{
			Primary: true,
		}),
	)
}

func TestProxyModelFieldDefs(t *testing.T) {
	var b = &BlogPage{}
	var defs = b.FieldDefs()
	if defs.Len() != 6 {
		t.Errorf("expected 6 fields, got %d: %v", defs.Len(), attrs.FieldNames(defs.Fields(), nil))
	}

	b.Proxy = &Page{}
	defs = b.FieldDefs()
	if defs.Len() != 10 {
		t.Errorf("expected 10 fields, got %d: %v", defs.Len(), attrs.FieldNames(defs.Fields(), nil))
	}

	defs.Set("ID", 1)
	defs.Set("Title", "New Title")
	defs.Set("Description", "New Description")
	defs.Set("Author", "John Doe")
	defs.Set("Tags", []string{"tag1", "tag2"})

	if b.Proxy.ID != 1 {
		t.Errorf("expected Proxy.ID to be 1, got %d", b.Proxy.ID)
	}

	if b.Proxy.Title != "New Title" {
		t.Errorf("expected Proxy.Title to be 'New Title', got '%s'", b.Proxy.Title)
	}

	if b.Proxy.Description != "New Description" {
		t.Errorf("expected Proxy.Description to be 'New Description', got '%s'", b.Proxy.Description)
	}

	if b.Author != "John Doe" {
		t.Errorf("expected Author to be 'John Doe', got '%s'", b.Author)
	}

	if len(b.Tags) != 2 || b.Tags[0] != "tag1" || b.Tags[1] != "tag2" {
		t.Errorf("expected Tags to be ['tag1', 'tag2'], got %v", b.Tags)
	}

	b.Proxy = nil

	defs = b.FieldDefs()

	if b.Proxy != nil {
		t.Errorf("expected Proxy to be nil, got %v", b.Proxy)
	}

	if defs.Len() != 6 {
		t.Errorf("expected 6 fields after setting Proxy to nil, got %d", defs.Len())
	}
}

func TestGetQuerySet(t *testing.T) {

	var b = &BlogPageCategory{}
	var err error

	var qs = queries.GetQuerySet(b)
	t.Logf("QuerySet: %T", qs)
	res, err := qs.All()
	if err != nil {
		t.Fatalf("failed to get all: %v", err)
	}

	if len(res) != 0 {
		t.Errorf("expected 0 results, got %d", len(res))
	}
}
