package models_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	queries "github.com/Nigel2392/go-django/queries/src"
	queryerrors "github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
)

type ProxyOptionsModel struct {
	models.Model
	ID                int64
	PrimaryPage       *Page `proxy:"true" auto:"true"`
	ReviewPage        *Page `proxy:"true"`
	NamedPage         *Page
	*BlogPage         `proxy:"false"`
	*BlogPageCategory `proxy:"-"`
}

func (m *ProxyOptionsModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m, attrs.Unbound("ID", &attrs.Config{Primary: true}))
}

func TestProxyModelSetupOptions(t *testing.T) {
	ctx := t.Context()
	first := models.Setup(ctx, &ProxyOptionsModel{})
	second := models.Setup(ctx, &ProxyOptionsModel{})
	if first.PrimaryPage == nil || second.PrimaryPage == nil || first.PrimaryPage == second.PrimaryPage {
		t.Fatal("auto proxy must allocate a separate page for each model instance")
	}
	if first.PrimaryPage.Model.Object() != first.PrimaryPage || first.Model.Object() != first {
		t.Fatal("the embedder and proxy must each bind their own base model")
	}
	if first.ReviewPage != nil || first.NamedPage != nil || first.BlogPage != nil || first.BlogPageCategory != nil {
		t.Fatal("setup allocated a field without an enabled auto proxy")
	}

	tree := queries.ProxyFields(first)
	if tree.FieldsLen() != 2 {
		t.Fatalf("got %d proxy fields, want 2", tree.FieldsLen())
	}
	for _, name := range []string{"PrimaryPage", "ReviewPage"} {
		if !tree.IsProxyField(name) {
			t.Errorf("expected %q to be recognized as a proxy", name)
		}
	}
	for _, name := range []string{"NamedPage", "BlogPage", "BlogPageCategory"} {
		if tree.IsProxyField(name) {
			t.Errorf("unexpected proxy field %q", name)
		}
	}

	original := first.PrimaryPage
	if models.Setup(ctx, first).PrimaryPage != original {
		t.Fatal("repeated setup replaced a populated auto proxy")
	}
	replacement := &Page{Title: "Replacement"}
	first.PrimaryPage = replacement
	first.ReviewPage = &Page{Title: "Review"}
	models.Setup(ctx, first)
	if first.PrimaryPage != replacement || replacement.Model.Object() != replacement || first.ReviewPage.Model.Object() != first.ReviewPage {
		t.Fatal("setup failed to bind newly assigned proxies")
	}

	first.PrimaryPage = nil
	first.ReviewPage = nil
	models.Setup(ctx, first)
	if first.PrimaryPage == nil || first.PrimaryPage == replacement || first.ReviewPage != nil {
		t.Fatal("setup failed to recreate only the cleared auto proxy")
	}
}

func TestWalkNestedProxyFields(t *testing.T) {
	page := &Page{Title: "Leaf"}
	blog := &BlogPage{Proxy: page}
	for _, tc := range []struct {
		name        string
		blog        *BlogPage
		includeZero bool
		wantPaths   []string
	}{
		{name: "populated chain", blog: blog, wantPaths: []string{"BlogPage", "BlogPage.Proxy"}},
		{name: "missing leaf", blog: &BlogPage{}, wantPaths: []string{"BlogPage"}},
		{name: "include missing leaf", blog: &BlogPage{}, includeZero: true, wantPaths: []string{"BlogPage", "BlogPage.Proxy"}},
		{name: "missing chain"},
		{name: "include missing chain", includeZero: true, wantPaths: []string{"BlogPage", "BlogPage.Proxy"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := &BlogPageCategory{BlogPage: tc.blog}
			tree := queries.ProxyFields(root)
			var paths []string
			var walkedBlog *BlogPage
			for item, err := range tree.WalkObjectProxies(root, tc.includeZero) {
				if err != nil {
					t.Fatal(err)
				}
				paths = append(paths, strings.Join(item.Path, "."))
				if item.ParentField.Name() != item.Field.Name() {
					t.Errorf("bound field %q differs from proxy metadata %q", item.ParentField.Name(), item.Field.Name())
				}
				switch item.Field.Name() {
				case "BlogPage":
					var ok bool
					walkedBlog, ok = item.Value.(*BlogPage)
					if !ok || walkedBlog == nil || tc.blog != nil && walkedBlog != tc.blog {
						t.Fatalf("unexpected intermediate proxy %v", item.Value)
					}
				case "Proxy":
					walkedPage, ok := item.Value.(*Page)
					if !ok || walkedPage == nil || tc.blog == blog && walkedPage != page {
						t.Fatalf("unexpected leaf proxy %v", item.Value)
					}
					if item.Parent != walkedBlog {
						t.Errorf("leaf parent = %T %p, want %p", item.Parent, item.Parent, walkedBlog)
					}
				}
			}
			if !slices.Equal(paths, tc.wantPaths) {
				t.Errorf("walk paths = %v, want %v", paths, tc.wantPaths)
			}
			if root.BlogPage != tc.blog || tc.blog != nil && tc.blog != blog && tc.blog.Proxy != nil {
				t.Fatal("walk changed the supplied object")
			}
		})
	}

	t.Run("stop before visiting leaf", func(t *testing.T) {
		root := &BlogPageCategory{BlogPage: blog}
		for item, err := range queries.ProxyFields(root).WalkObjectProxies(root, false) {
			if err != nil || !slices.Equal(item.Path, []string{"BlogPage"}) {
				t.Fatalf("first item = %v, error = %v", item.Path, err)
			}
			return
		}
		t.Fatal("walk yielded no items")
	})
}

func TestPolymorphicProxyJoinDefinition(t *testing.T) {
	model := &BlogPage{}
	field, ok := queries.ProxyFields(model).Get("Proxy")
	if !ok {
		t.Fatal("missing Proxy metadata")
	}
	lhs := queries.ClauseTarget{Model: model, Table: queries.Table{Name: "blog_page", Alias: "blog"}}
	rhs := queries.ClauseTarget{Model: &Page{}, Table: queries.Table{Name: "page", Alias: "base"}}
	join := field.GenerateTargetClause(nil, lhs, rhs)
	if join.Table != rhs.Table || join.TypeJoin != expr.TypeJoinInner {
		t.Errorf("join = %v on %v, want inner join on %v", join.TypeJoin, join.Table, rhs.Table)
	}
	key := join.JoinDefCondition
	if key == nil || key.Next == nil || key.Next.Next != nil {
		t.Fatalf("expected two join conditions, got %v", key)
	}
	if key.Operator != expr.EQ || key.LHS().TableOrAlias != lhs.Table.Alias || key.LHS().FieldColumn.Name() != "PageID" ||
		key.RHS().TableOrAlias != rhs.Table.Alias || key.RHS().FieldColumn.Name() != "PageID" {
		t.Errorf("unexpected primary-key join condition %v", key)
	}
	ctype := key.Next
	wantType := contenttypes.NewContentType[attrs.Definer](model).TypeName()
	if ctype.Operator != expr.EQ || ctype.LHS().TableOrAlias != rhs.Table.Alias ||
		ctype.LHS().FieldColumn.Name() != "PageContentType" || ctype.RHS().RawSQL != "?" ||
		!slices.Equal(ctype.RHS().Values, []any{wantType}) {
		t.Errorf("unexpected content-type join condition %v", ctype)
	}
}

type ControlledProxyPage struct {
	Page
}

func (*ControlledProxyPage) ControlsEmbedderSaving() bool { return true }

type ControlledProxyEmbedder struct {
	models.Model
	ID   int64
	Base *ControlledProxyPage `proxy:"true"`
}

func (m *ControlledProxyEmbedder) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m, attrs.Unbound("ID", &attrs.Config{Primary: true}))
}

func TestProxyWithCustomSaveControlIsReadOnly(t *testing.T) {
	ctx := t.Context()
	obj := &ControlledProxyEmbedder{
		ID:   27,
		Base: &ControlledProxyPage{Page: Page{PageID: 11}},
	}
	field, ok := attrs.Define(ctx, obj).Field("Base")
	if !ok {
		t.Fatal("missing controlled proxy field")
	}
	proxy, ok := field.(interface {
		queries.ProxyField
		queries.SaveableDependantField
		AllowDBEdit() bool
		AllowReverseRelation() bool
		CanMigrate() bool
	})
	if !ok {
		t.Fatalf("unexpected controlled proxy field type %T", field)
	}
	if !proxy.IsProxy() || proxy.AllowDBEdit() || proxy.AllowReverseRelation() || proxy.CanMigrate() || proxy.ColumnName() != "" {
		t.Errorf("unexpected metadata for controlled proxy field %q", proxy.Name())
	}
	if err := proxy.Save(ctx, obj); !queryerrors.Is(err, queryerrors.NotImplemented) {
		t.Fatalf("save error = %v, want NotImplemented", err)
	}
	if obj.Base.PageID != 11 || obj.Base.PageContentType != nil {
		t.Fatal("saving a controlled proxy changed its target")
	}
}

func TestNestedProxyQuery(t *testing.T) {
	ctx := t.Context()
	root := &BlogPageCategory{Category: "test-nested-proxy"}
	leaf := &Page{BasePage: BasePage{ID: 3201}, Title: "Nested leaf", Description: "Loaded through two proxy joins"}
	branch := &BlogPage{PageID: 3202, Author: "Nested author"}
	t.Cleanup(func() {
		for _, fixture := range []struct {
			model attrs.Definer
			field string
			value any
		}{
			{&Page{}, "ID", leaf.ID},
			{&BlogPage{}, "PageID", branch.PageID},
			{&BlogPageCategory{}, "Category", root.Category},
		} {
			if _, err := queries.GetQuerySet(fixture.model).Filter(fixture.field, fixture.value).Delete(); err != nil {
				t.Errorf("delete %T fixture: %v", fixture.model, err)
			}
		}
	})
	if _, err := queries.GetQuerySetWithContext(ctx, root).Exec(
		"INSERT INTO TABLE(SELF) (category) VALUES (?[1])", root.Category,
	); err != nil {
		t.Fatalf("insert category: %v", err)
	}
	if _, err := queries.GetQuerySetWithContext(ctx, branch).Exec(
		"INSERT INTO TABLE(SELF) (page_id, author, tags, category, category_content_type) VALUES (?[1], ?[2], ?[3], ?[4], ?[5])",
		branch.PageID, branch.Author, "[]", root.Category, contenttypes.NewContentType[attrs.Definer](root).TypeName(),
	); err != nil {
		t.Fatalf("insert intermediate proxy: %v", err)
	}
	if _, err := queries.GetQuerySetWithContext(ctx, leaf).Exec(
		"INSERT INTO TABLE(SELF) (id, page_id, page_content_type, title, description) VALUES (?[1], ?[2], ?[3], ?[4], ?[5])",
		leaf.ID, branch.PageID, contenttypes.NewContentType[attrs.Definer](branch).TypeName(), leaf.Title, leaf.Description,
	); err != nil {
		t.Fatalf("insert leaf proxy: %v", err)
	}

	// Project scalar fields from all three tables to check nested scanning.
	row, err := queries.GetQuerySetWithContext(ctx, &BlogPageCategory{}).
		Select("Category", "BlogPage.PageID", "BlogPage.Author", "BlogPage.Proxy.ID",
			"BlogPage.Proxy.PageID", "BlogPage.Proxy.Title", "BlogPage.Proxy.Description").
		Filter("BlogPage.Proxy.Title", leaf.Title).Get()
	if err != nil {
		t.Fatalf("load nested proxy chain: %v", err)
	}
	got := row.Object
	if got.Category != root.Category || got.BlogPage == nil || got.BlogPage.Proxy == nil {
		t.Fatalf("incomplete nested result: category=%q intermediate=%+v", got.Category, got.BlogPage)
	}
	if got.BlogPage.PageID != branch.PageID || got.Author != branch.Author {
		t.Errorf("intermediate proxy fields = (%d, %q), want (%d, %q)",
			got.BlogPage.PageID, got.Author, branch.PageID, branch.Author)
	}
	if got.Proxy.ID != leaf.ID || got.Proxy.PageID != branch.PageID || got.Proxy.Title != leaf.Title || got.Proxy.Description != leaf.Description {
		t.Errorf("leaf proxy fields = (%d, %d, %q, %q), want (%d, %d, %q, %q)",
			got.Proxy.ID, got.Proxy.PageID, got.Proxy.Title, got.Proxy.Description, leaf.ID, branch.PageID, leaf.Title, leaf.Description)
	}
}
