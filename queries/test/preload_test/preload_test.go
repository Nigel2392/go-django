package preload_test

import (
	"testing"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

func TestPreload(t *testing.T) {
	t.Run("TestAuthorHasBooks", func(t *testing.T) {
		authorRows, err := queries.GetQuerySet(&PreloadAuthor{}).
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
			if book.Authors.Len() == 0 || book.Authors.Len() != len(booksAuthorMap[book.ID]) {
				t.Fatalf("Expected book %s to have %d authors, got %d", book.Title, len(booksAuthorMap[book.ID]), book.Authors.Len())
			}

			t.Logf("Book %s has %d authors", book.Title, book.Authors.Len())

			if len(book.Authors.AsList()) == 0 {
				t.Fatalf("Expected book %s to have authors, got none", book.Title)
			}

			for _, authorRow := range book.Authors.AsList() {
				var author = authorRow.Object.(*PreloadAuthor)
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
			if book.Authors.Len() == 0 || book.Authors.Len() != len(booksAuthorMap[book.ID]) {
				t.Fatalf("Expected book %s to have %d authors, got %d", book.Title, len(booksAuthorMap[book.ID]), book.Authors.Len())
			}

			t.Logf("Book %s has %d authors with profiles", book.Title, book.Authors.Len())

			for _, authorRow := range book.Authors.AsList() {
				var author = authorRow.Object.(*PreloadAuthor)
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
			if book.Authors.Len() == 0 || book.Authors.Len() != len(booksAuthorMap[book.ID]) {
				t.Fatalf("Expected book %s to have %d authors, got %d", book.Title, len(booksAuthorMap[book.ID]), book.Authors.Len())
			}

			t.Logf("Book %s has %d authors", book.Title, book.Authors.Len())

			for _, authorRow := range book.Authors.AsList() {
				var author = authorRow.Object.(*PreloadAuthor)
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
