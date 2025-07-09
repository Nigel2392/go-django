package preload_test

import (
	"flag"
	"os"
	"testing"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/quest"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/djester/testdb"
)

var (
	preloadAuthors = []*PreloadAuthor{
		{Name: "Rowling"},
		{Name: "Martin"},
		{Name: "Tolkien"},
		{Name: "Shared Author"},
	}

	preloadAuthorProfiles = []*PreloadAuthorProfile{
		{Email: drivers.MustParseEmail("rowling@example.com"), FirstName: "J.K.", LastName: "Rowling", Author: preloadAuthors[0]},
		{Email: drivers.MustParseEmail("martin@example.com"), FirstName: "George R.R.", LastName: "Martin", Author: preloadAuthors[1]},
		{Email: drivers.MustParseEmail("tolkien@example.com"), FirstName: "J.R.R.", LastName: "Tolkien", Author: preloadAuthors[2]},
		{Email: drivers.MustParseEmail("shared@example.com"), FirstName: "Shared", LastName: "Author", Author: preloadAuthors[3]},
	}

	author0Books = []*PreloadBook{
		{Title: "Harry Potter and the Philosopher's Stone"},
		{Title: "Harry Potter and the Chamber of Secrets"},
		{Title: "Harry Potter and the Prisoner of Azkaban"},
	}
	author1Books = []*PreloadBook{
		{Title: "A Game of Thrones"},
		{Title: "A Clash of Kings"},
		{Title: "A Storm of Swords"},
	}
	author2Books = []*PreloadBook{
		{Title: "The Hobbit"},
		{Title: "The Lord of the Rings: The Fellowship of the Ring"},
		{Title: "The Lord of the Rings: The Two Towers"},
		{Title: "The Lord of the Rings: The Return of the King"},
	}

	authorBooksMap     = make(map[uint64][]*PreloadBook, 0)
	booksAuthorMap     = make(map[uint64][]*PreloadAuthor, 0)
	authorToProfileMap = make(map[uint64]*PreloadAuthorProfile, 0)
)

func TestMain(m *testing.M) {
	testing.Init()

	flag.Parse()

	var which, db = testdb.Open()
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

	django.App(django.Configure(settings))

	logger.Debugf("Using %s database for queries tests", which)

	//if !testing.Verbose() {
	//	logger.SetLevel(logger.WRN)
	//}

	var tables = quest.Table[*testing.T](nil,
		&PreloadBook{},
		&PreloadAuthorBook{},
		&PreloadAuthor{},
		&PreloadAuthorProfile{},
	)

	attrs.ResetDefinitions.Send(nil)

	tables.Create()
	defer tables.Drop()

	logger.Infof("Creating preload authors, profiles, and books")

	var err error
	preloadAuthors, err = queries.GetQuerySet(&PreloadAuthor{}).BulkCreate(preloadAuthors)
	if err != nil {
		logger.Errorf("Failed to create preload authors: %v", err)
		os.Exit(1)
	}

	preloadAuthorProfiles[0].Author = preloadAuthors[0]
	preloadAuthorProfiles[1].Author = preloadAuthors[1]
	preloadAuthorProfiles[2].Author = preloadAuthors[2]
	preloadAuthorProfiles[3].Author = preloadAuthors[3]

	logger.Infof("Creating profiles for %d authors", len(preloadAuthors))

	preloadAuthorProfiles, err = queries.GetQuerySet(&PreloadAuthorProfile{}).BulkCreate(preloadAuthorProfiles)
	if err != nil {
		logger.Errorf("Failed to create preload author profiles: %v", err)
		os.Exit(1)
	}

	logger.Infof("Adding books to author %q", preloadAuthors[0].Name)

	_, err = preloadAuthors[0].Books.Objects().AddTargets(author0Books...)
	if err != nil {
		logger.Errorf("Failed to add books to author %s: %v", preloadAuthors[0].Name, err)
		os.Exit(1)
	}

	logger.Infof("Adding books to author %q", preloadAuthors[1].Name)

	_, err = preloadAuthors[1].Books.Objects().AddTargets(author1Books...)
	if err != nil {
		logger.Errorf("Failed to add books to author %s: %v", preloadAuthors[1].Name, err)
		os.Exit(1)
	}

	logger.Infof("Adding books to author %q", preloadAuthors[2].Name)

	_, err = preloadAuthors[2].Books.Objects().AddTargets(author2Books...)
	if err != nil {
		logger.Errorf("Failed to add books to author %s: %v", preloadAuthors[2].Name, err)
		os.Exit(1)
	}

	logger.Infof("Adding books to shared author %q", preloadAuthors[3].Name)

	_, err = preloadAuthors[3].Books.Objects().AddTargets(author0Books...)
	if err != nil {
		logger.Errorf("Failed to add books to author %s: %v", preloadAuthors[3].Name, err)
		os.Exit(1)
	}

	_, err = preloadAuthors[3].Books.Objects().AddTargets(author1Books...)
	if err != nil {
		logger.Errorf("Failed to add books to author %s: %v", preloadAuthors[3].Name, err)
		os.Exit(1)
	}

	_, err = preloadAuthors[3].Books.Objects().AddTargets(author2Books...)
	if err != nil {
		logger.Errorf("Failed to add books to author %s: %v", preloadAuthors[3].Name, err)
		os.Exit(1)
	}

	authorBooksMap = mapAuthors(authorBooksMap, preloadAuthors[0], author0Books)
	authorBooksMap = mapAuthors(authorBooksMap, preloadAuthors[1], author1Books)
	authorBooksMap = mapAuthors(authorBooksMap, preloadAuthors[2], author2Books)
	authorBooksMap = mapAuthors(authorBooksMap, preloadAuthors[3], author0Books, author1Books, author2Books)

	booksAuthorMap = mapBooks(booksAuthorMap, author0Books[0], []*PreloadAuthor{preloadAuthors[0], preloadAuthors[3]})
	booksAuthorMap = mapBooks(booksAuthorMap, author0Books[1], []*PreloadAuthor{preloadAuthors[0], preloadAuthors[3]})
	booksAuthorMap = mapBooks(booksAuthorMap, author0Books[2], []*PreloadAuthor{preloadAuthors[0], preloadAuthors[3]})

	booksAuthorMap = mapBooks(booksAuthorMap, author1Books[0], []*PreloadAuthor{preloadAuthors[1], preloadAuthors[3]})
	booksAuthorMap = mapBooks(booksAuthorMap, author1Books[1], []*PreloadAuthor{preloadAuthors[1], preloadAuthors[3]})
	booksAuthorMap = mapBooks(booksAuthorMap, author1Books[2], []*PreloadAuthor{preloadAuthors[1], preloadAuthors[3]})

	booksAuthorMap = mapBooks(booksAuthorMap, author2Books[0], []*PreloadAuthor{preloadAuthors[2], preloadAuthors[3]})
	booksAuthorMap = mapBooks(booksAuthorMap, author2Books[1], []*PreloadAuthor{preloadAuthors[2], preloadAuthors[3]})
	booksAuthorMap = mapBooks(booksAuthorMap, author2Books[2], []*PreloadAuthor{preloadAuthors[2], preloadAuthors[3]})
	booksAuthorMap = mapBooks(booksAuthorMap, author2Books[3], []*PreloadAuthor{preloadAuthors[2], preloadAuthors[3]})

	authorToProfileMap[preloadAuthors[0].ID] = preloadAuthorProfiles[0]
	authorToProfileMap[preloadAuthors[1].ID] = preloadAuthorProfiles[1]
	authorToProfileMap[preloadAuthors[2].ID] = preloadAuthorProfiles[2]
	authorToProfileMap[preloadAuthors[3].ID] = preloadAuthorProfiles[3]

	var exitCode = m.Run()
	if err := db.Close(); err != nil {
		logger.Errorf("Failed to close test database: %v", err)
		exitCode = 1
	}
	os.Exit(exitCode)
}

func mapAuthors(m map[uint64][]*PreloadBook, author *PreloadAuthor, books ...[]*PreloadBook) map[uint64][]*PreloadBook {
	if m[author.ID] == nil {
		m[author.ID] = make([]*PreloadBook, 0, len(books))
	}

	for _, bookList := range books {
		m[author.ID] = append(m[author.ID], bookList...)
	}
	return m
}

func mapBooks(m map[uint64][]*PreloadAuthor, book *PreloadBook, authors ...[]*PreloadAuthor) map[uint64][]*PreloadAuthor {
	if m[book.ID] == nil {
		m[book.ID] = make([]*PreloadAuthor, 0, len(authors))
	}

	for _, authorList := range authors {
		m[book.ID] = append(m[book.ID], authorList...)
	}
	return m
}
