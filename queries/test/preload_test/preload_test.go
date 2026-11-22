package preload_test

import (
	"database/sql/driver"
	"slices"
	"testing"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

func TestPreloadBookReferencePrimaryKeyDDL(t *testing.T) {
	model := &PreloadBookReference{}
	table := migrator.NewModelTable(model)
	column := migrator.NewTableColumn(table, attrs.Define(t.Context(), model).Primary())
	if !column.Primary {
		t.Fatal("book reference ID must be a primary key")
	}
	for _, drv := range []driver.Driver{&drivers.DriverMySQL{}, &drivers.DriverMariaDB{}} {
		if got := migrator.GetFieldType(drv, &column); got != "VARCHAR(48)" {
			t.Errorf("%T: book reference ID type is %s, want VARCHAR(48)", drv, got)
		}
	}
}

func TestPreloadForwardForeignKey(t *testing.T) {
	sharedBook, otherBook := author0Books[0], author1Books[0]
	refs := []*PreloadBookReference{
		{ID: "first-reference", Book: sharedBook},
		{ID: "second-reference", Book: sharedBook},
		{ID: "third-reference", Book: otherBook},
	}
	ctx, tx, err := queries.StartTransaction(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	qs := queries.GetQuerySet(&PreloadBookReference{}).WithContext(ctx)
	if _, err := qs.BulkCreate(refs); err != nil {
		t.Fatal(err)
	}

	expected := make(map[string]*PreloadBook, len(refs))
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		expected[ref.ID] = ref.Book
		ids = append(ids, ref.ID)
	}

	for _, tc := range []struct {
		name string
		path string
		join bool
	}{
		{name: "direct relation", path: "Book"},
		{name: "nested relation", path: "Book.Authors"},
		{name: "selected relation with nested preload", path: "Book.Authors", join: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			query := qs.Filter("ID__in", ids)
			if tc.join {
				query = query.Select("*", "Book.*")
			}
			rows, err := query.Preload(tc.path).All()
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != len(refs) {
				t.Fatalf("got %d references, want %d", len(rows), len(refs))
			}

			for _, row := range rows {
				ref := row.Object
				want := expected[ref.ID]
				if want == nil || ref.Book == nil || ref.Book.ID != want.ID || ref.Book.Title != want.Title {
					t.Fatalf("reference %s has book %+v, want %+v", ref.ID, ref.Book, want)
				}
				if tc.path == "Book" {
					continue
				}
				var gotIDs, wantIDs []uint64
				for _, author := range ref.Book.Authors {
					gotIDs = append(gotIDs, author.Model().(*PreloadAuthor).ID)
				}
				for _, author := range booksAuthorMap[want.ID] {
					wantIDs = append(wantIDs, author.ID)
				}
				slices.Sort(gotIDs)
				slices.Sort(wantIDs)
				if !slices.Equal(gotIDs, wantIDs) {
					t.Fatalf("reference %s has book authors %v, want %v", ref.ID, gotIDs, wantIDs)
				}
			}
		})
	}
}

func TestPreload(t *testing.T) {
	t.Run("TestAuthorHasBooks", func(t *testing.T) {
		authorRows, err := queries.GetQuerySet(&PreloadAuthor{}).
			Prefix("authors_books_qs").
			Preload("Books").
			All()
		if err != nil {
			t.Fatalf("Failed to get author: %v", err)
		}

		if len(authorRows) != 4 {
			t.Fatalf("Expected 4 authors, got %d", len(authorRows))
		}

		for _, authorRow := range authorRows {
			var author = authorRow.Object
			if author.Books.Len() == 0 || author.Books.Len() != len(authorBooksMap[author.ID]) {
				t.Fatalf("Expected author %s to have %d books, got %d", author.Name, len(authorBooksMap[author.ID]), author.Books.Len())
			}

			t.Logf("Author %s has %d books", author.Name, author.Books.Len())

			for _, book := range author.Books.AsList() {
				var bookAuthors = booksAuthorMap[book.Object.ID]
				if len(bookAuthors) == 0 {
					t.Fatalf("Expected book %s to have authors, got none", book.Object.Title)
				}

				var found bool
				for _, a := range bookAuthors {
					if a.ID == author.ID {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("Expected book %s to have author %s, got none", book.Object.Title, author.Name)
				}

				// t.Logf("Book %q belongs to author %q", book.Object.Title, author.Name)
			}
		}
	})

	t.Run("TestBookHasAuthors", func(t *testing.T) {
		bookRows, err := queries.GetQuerySet(&PreloadBook{}).
			Preload("Authors.Books").
			All()
		if err != nil {
			t.Fatalf("Failed to get book: %v", err)
		}

		if len(bookRows) != 10 {
			t.Fatalf("Expected 10 books, got %d", len(bookRows))
		}

		for _, bookRow := range bookRows {
			var book = bookRow.Object
			if len(book.Authors) == 0 || len(book.Authors) != len(booksAuthorMap[book.ID]) {
				t.Fatalf("Expected book %s to have %d authors, got %d", book.Title, len(booksAuthorMap[book.ID]), len(book.Authors))
			}

			t.Logf("Book %s has %d authors", book.Title, len(book.Authors))

			if len(book.Authors) == 0 {
				t.Fatalf("Expected book %s to have authors, got none", book.Title)
			}

			for _, authorRow := range book.Authors {
				var author = authorRow.Model().(*PreloadAuthor)
				var authorBooks = authorBooksMap[author.ID]
				if len(authorBooks) == 0 {
					t.Fatalf("Expected author %s to have books, got none", author.Name)
				}

				var found bool
				for _, b := range authorBooks {
					if b.ID == book.ID {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("Expected author %s to have book %s, got none", author.Name, book.Title)
				}

				t.Logf("Nested Author %q wrote book %q", author.Name, book.Title)
			}
		}
	})

	t.Run("TestAuthorOnlyProfile", func(t *testing.T) {
		var qs = queries.GetQuerySet(&PreloadAuthor{}).SelectRelated("Profile")
		authorRows, err := qs.All()
		if err != nil {
			t.Fatalf("Failed to get author with profile: %v", err)
		}

		if len(authorRows) != 4 {
			t.Fatalf("Expected 4 authors, got %d", len(authorRows))
		}

		for _, authorRow := range authorRows {
			var author = authorRow.Object
			if author.Profile == nil {
				t.Fatalf("Expected author %s to have a profile, got none:\n\t%s", author.Name, qs.LatestQuery().SQL())
			}

			var profile = authorToProfileMap[author.ID]
			if profile == nil {
				t.Fatalf("Expected author %s to have a profile, got none:\n\t%s", author.Name, qs.LatestQuery().SQL())
			}

			if author.Profile.ID != profile.ID {
				t.Fatalf("Expected author %s to have profile ID %d, got %d", author.Name, profile.ID, author.Profile.ID)
			}

			if author.Profile.Email.Address != profile.Email.Address {
				t.Fatalf("Expected author %s profile email %s to match profile email %s", author.Name, author.Profile.Email.Address, profile.Email.Address)
			}
			if author.Profile.FirstName != profile.FirstName {
				t.Fatalf("Expected author %s profile first name %s to match profile first name %s", author.Name, author.Profile.FirstName, profile.FirstName)
			}
			if author.Profile.LastName != profile.LastName {
				t.Fatalf("Expected author %s profile last name %s to match profile last name %s", author.Name, author.Profile.LastName, profile.LastName)
			}

		}
	})

	t.Run("TestBooksHasAuthorsWithProfile", func(t *testing.T) {
		var qs = queries.GetQuerySet(&PreloadBook{}).
			Preload(queries.Preload{
				Path: "Authors",
				QuerySet: queries.GetQuerySet[attrs.Definer](&PreloadAuthor{}).
					SelectRelated("Profile"),
			})

		bookRows, err := qs.All()
		if err != nil {
			t.Fatalf("Failed to get book with authors and profile: %v", err)
		}

		if len(bookRows) != 10 {
			t.Fatalf("Expected 10 books, got %d", len(bookRows))
		}

		for _, bookRow := range bookRows {
			var book = bookRow.Object
			if len(book.Authors) == 0 || len(book.Authors) != len(booksAuthorMap[book.ID]) {
				t.Fatalf("Expected book %s to have %d authors, got %d", book.Title, len(booksAuthorMap[book.ID]), len(book.Authors))
			}

			t.Logf("Book %s has %d authors with profiles", book.Title, len(book.Authors))

			for _, authorRow := range book.Authors {
				var author = authorRow.Model().(*PreloadAuthor)
				var profile = authorToProfileMap[author.ID]
				if profile == nil {
					t.Fatalf("Expected author %s to have a profile, got none:\n\t%s", author.Name, qs.LatestQuery().SQL())
				}

				if author.Profile == nil {
					t.Fatalf("Expected author %s to have a profile, but it is nil:\n\t%s", author.Name, qs.LatestQuery().SQL())
				}

				if author.Profile.ID != profile.ID {
					t.Fatalf("Expected author %s to have profile ID %d, got %d", author.Name, profile.ID, author.Profile.ID)
				}

				if profile.Author.Name != author.Name {
					t.Fatalf("Expected profile author name %s to match author name %s", profile.Author.Name, author.Name)
				}

				if author.Profile.Email.Address != profile.Email.Address {
					t.Fatalf("Expected author %s profile email %s to match profile email %s", author.Name, author.Profile.Email.Address, profile.Email.Address)
				}

				t.Logf("Author %s has profile with ID %d", author.Name, profile.ID)
			}
		}
	})

	t.Run("TestAuthorHasBooksNoPreload", func(t *testing.T) {
		authorRows, err := queries.GetQuerySet(&PreloadAuthor{}).Select("*", "Books.*").All()
		if err != nil {
			t.Fatalf("Failed to get author: %v", err)
		}

		if len(authorRows) != 4 {
			t.Fatalf("Expected 4 authors, got %d", len(authorRows))
		}

		for _, authorRow := range authorRows {
			var author = authorRow.Object
			if author.Books.Len() == 0 || author.Books.Len() != len(authorBooksMap[author.ID]) {
				t.Fatalf("Expected author %s to have %d books, got %d", author.Name, len(authorBooksMap[author.ID]), author.Books.Len())
			}

			t.Logf("Author %s has %d books", author.Name, author.Books.Len())

			for _, book := range author.Books.AsList() {
				var bookAuthors = booksAuthorMap[book.Object.ID]
				if len(bookAuthors) == 0 {
					t.Fatalf("Expected book %s to have authors, got none", book.Object.Title)
				}

				var found bool
				for _, a := range bookAuthors {
					if a.ID == author.ID {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("Expected book %s to have author %s, got none", book.Object.Title, author.Name)
				}

				// t.Logf("Book %q belongs to author %q", book.Object.Title, author.Name)
			}
		}
	})

	t.Run("TestBookHasAuthorsNoPreload", func(t *testing.T) {
		bookRows, err := queries.GetQuerySet(&PreloadBook{}).Select("*", "Authors.*").All()
		if err != nil {
			t.Fatalf("Failed to get book: %v", err)
		}

		if len(bookRows) != 10 {
			t.Fatalf("Expected 10 books, got %d", len(bookRows))
		}

		for _, bookRow := range bookRows {
			var book = bookRow.Object
			if len(book.Authors) == 0 || len(book.Authors) != len(booksAuthorMap[book.ID]) {
				t.Fatalf("Expected book %s to have %d authors, got %d", book.Title, len(booksAuthorMap[book.ID]), len(book.Authors))
			}

			t.Logf("Book %s has %d authors", book.Title, len(book.Authors))

			for _, authorRow := range book.Authors {
				var author = authorRow.Model().(*PreloadAuthor)
				var authorBooks = authorBooksMap[author.ID]
				if len(authorBooks) == 0 {
					t.Fatalf("Expected author %s to have books, got none", author.Name)
				}

				var found bool
				for _, b := range authorBooks {
					if b.ID == book.ID {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("Expected author %s to have book %s, got none", author.Name, book.Title)
				}

				// t.Logf("Author %q wrote book %q", author.Name, book.Title)
			}
		}
	})
}
