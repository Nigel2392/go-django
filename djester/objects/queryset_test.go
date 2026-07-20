package objects_test

import (
	"context"
	"testing"

	"github.com/Nigel2392/go-django/djester"
	"github.com/Nigel2392/go-django/djester/objects"
	"github.com/Nigel2392/go-django/djester/quest"
	"github.com/Nigel2392/go-django/djester/testdb"
	queries "github.com/Nigel2392/go-django/queries/src"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

var (
	_ attrs.Definer = (*Author)(nil)
	_ attrs.Definer = (*Book)(nil)
)

type Author struct {
	ID   int64
	Name string
}

func (a *Author) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, a,
		attrs.NewField(a, "ID", &attrs.FieldConfig{
			Primary: true,
		}),
		attrs.NewField(a, "Name", nil),
	).WithTableName("djstr_qs_author")
}

type Book struct {
	ID     int64
	Title  string
	Author *Author
}

func (b *Book) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, b,
		attrs.NewField(b, "ID", &attrs.FieldConfig{
			Primary: true,
		}),
		attrs.NewField(b, "Title", nil),
		attrs.NewField(b, "Author", &attrs.FieldConfig{
			Column:        "author_id",
			RelForeignKey: attrs.Relate(&Author{}, "", nil),
		}),
	).WithTableName("djstr_qs_book")
}

func TestQuerySetTests(t *testing.T) {
	var _, db = testdb.Open()
	_ = django.App(django.Configure(map[string]any{
		django.APPVAR_DATABASE: db,
	}))

	// t.Fatal("TODO: IMPLEMENT drivers.ContextWithQuerier IN QUERYSET COMPILER OR QUERYSET ITSELF!!!!!!!!!!!!")

	// create tables
	var tables = quest.Table[*testing.T](nil,
		&Author{},
		&Book{},
	)

	// Reset the definitions to ensure all models are registered
	// before reverse fields are fully setup.
	attrs.ResetDefinitions.Send(nil)

	tables.Create()

	t.Run("TestCreateList", func(t *testing.T) {

		var test = objects.QuerySetTest{
			Create: []*Author{
				{Name: "Author 1"},
				{Name: "Author 2"},
				{Name: "Author 3"},
				{Name: "Author 4"},
				{Name: "Author 5"},
			},
			Execute: func(_ *djester.Tester, _ *testing.T, ctx context.Context) {
				count, err := queries.GetQuerySetWithContext(ctx, &Author{}).Count()
				if err != nil {
					t.Fatalf("error while counting newly created objects: %v", err)
				}

				if count != 5 {
					t.Fatalf("unpexpected count result, wanted %d, got %d", 5, count)
				}
			},
		}

		test.Test(nil, t)

		t.Run("TestTeardown", func(t *testing.T) {
			count, err := queries.GetQuerySet(&Author{}).Count()
			if err != nil {
				t.Fatalf("error while counting newly created objects: %v", err)
			}

			if count != 0 {
				t.Fatalf("unpexpected count result, wanted %d, got %d", 0, count)
			}
		})
	})

	t.Run("TestCreateNestedList", func(t *testing.T) {

		var test = objects.QuerySetTest{
			Create: []any{
				&Author{Name: "Author 1"},
				&Author{Name: "Author 2"},
				&Author{Name: "Author 3"},
				&Author{Name: "Author 4"},
				&Author{Name: "Author 5"},
				[]any{
					&Author{Name: "Author 6"},
					&Author{Name: "Author 7"},
					[]any{
						&Author{Name: "Author 8"},
						&Author{Name: "Author 9"},
						&Author{Name: "Author 10"},
					},
				},
			},
			Execute: func(_ *djester.Tester, _ *testing.T, ctx context.Context) {
				count, err := queries.GetQuerySetWithContext(ctx, &Author{}).Count()
				if err != nil {
					t.Fatalf("error while counting newly created objects: %v", err)
				}

				if count != 10 {
					t.Fatalf("unpexpected count result, wanted %d, got %d", 10, count)
				}
			},
		}

		test.Test(nil, t)

		t.Run("TestTeardown", func(t *testing.T) {
			count, err := queries.GetQuerySet(&Author{}).Count()
			if err != nil {
				t.Fatalf("error while counting newly created objects: %v", err)
			}

			if count != 0 {
				t.Fatalf("unpexpected count result, wanted %d, got %d", 0, count)
			}
		})
	})

	t.Run("TestCreateMultipleObjectsNestedList", func(t *testing.T) {

		var firstAuthor = &Author{Name: "Author 1"}
		var test = objects.QuerySetTest{
			Create: []any{
				firstAuthor,
				&Author{Name: "Author 2"},
				&Author{Name: "Author 3"},
				&Author{Name: "Author 4"},
				&Author{Name: "Author 5"},
				[]any{
					&Author{Name: "Author 6"},
					&Author{Name: "Author 7"},
					[]attrs.Definer{
						&Author{Name: "Author 8"},
						&Author{Name: "Author 9"},
						&Author{Name: "Author 10"},
					},
				},

				&Book{Title: "Book 1", Author: firstAuthor},
				&Book{Title: "Book 2", Author: firstAuthor},
				&Book{Title: "Book 3", Author: firstAuthor},
				&Book{Title: "Book 4", Author: firstAuthor},
				&Book{Title: "Book 5", Author: firstAuthor},

				[][]*Book{
					{
						&Book{Title: "Book 6", Author: firstAuthor},
						&Book{Title: "Book 7", Author: firstAuthor},
					},
					{
						&Book{Title: "Book 8", Author: firstAuthor},
						&Book{Title: "Book 9", Author: firstAuthor},
						&Book{Title: "Book 10", Author: firstAuthor},
					},
				},
			},
			Execute: func(_ *djester.Tester, _ *testing.T, ctx context.Context) {
				count, err := queries.GetQuerySetWithContext(ctx, &Author{}).Count()
				if err != nil {
					t.Fatalf("error while counting newly created objects author: %v", err)
				}

				if count != 10 {
					t.Fatalf("unpexpected count result, wanted %d auhors, got %d", 10, count)
				}

				count, err = queries.GetQuerySetWithContext(ctx, &Book{}).Count()
				if err != nil {
					t.Fatalf("error while counting newly created objects books: %v", err)
				}

				if count != 10 {
					t.Fatalf("unpexpected count result, wanted %d books, got %d", 10, count)
				}
			},
		}

		test.Test(nil, t)

		t.Run("TestTeardown", func(t *testing.T) {
			count, err := queries.GetQuerySet(&Author{}).Count()
			if err != nil {
				t.Fatalf("error while counting newly created objects: %v", err)
			}

			if count != 0 {
				t.Fatalf("unpexpected count result, wanted %d, got %d", 0, count)
			}
		})
	})
}
