package benchmarks_test

import (
	"database/sql"
	"testing"

	"github.com/Nigel2392/go-django/djester/testdb"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/elliotchance/orderedmap/v2"
	"github.com/jmoiron/sqlx"
)

type DedupeAuthor struct {
	ID    uint64
	Name  string
	Books []*DedupeBook
}

type DedupeBook struct {
	ID       uint64
	Title    string
	AuthorID uint64
}

func BenchmarkBaseline_RawDriver(b *testing.B) {
	which, db := testdb.Open()
	ctx := drivers.SetLogSQLContext(b.Context(), false)
	b.ResetTimer()

	for b.Loop() {
		rows, err := db.QueryContext(ctx, sqlx.Rebind(sqlx.BindType(which), "SELECT id, name FROM author_bench LIMIT ?"), COUNT*2)
		if err != nil {
			b.Fatal(err)
		}

		var count int
		for rows.Next() {
			var id uint64
			var name string
			if err := rows.Scan(&id, &name); err != nil {
				b.Fatal(err)
			}
			count++
		}
		rows.Close()
	}
}

func BenchmarkBaseline_RawSQL_OrderedMap_Dedupe(b *testing.B) {
	b.StopTimer()

	which, db := testdb.Open()
	ctx := drivers.SetLogSQLContext(b.Context(), false)

	rawSQL := sqlx.Rebind(sqlx.BindType(which), `
		SELECT a.id, a.name, b.id, b.title, b.author_id 
		FROM author_bench a 
		LEFT JOIN book_bench b ON a.id = b.author_id 
		LIMIT ?
	`)

	b.StartTimer()
	b.ResetTimer()

	for b.Loop() {
		rows, err := db.QueryContext(ctx, rawSQL, COUNT*2)
		if err != nil {
			b.Fatal(err)
		}

		om := orderedmap.NewOrderedMap[uint64, *DedupeAuthor]()

		for rows.Next() {
			var (
				aID       uint64
				aName     string
				bID       sql.NullInt64
				bTitle    sql.NullString
				bAuthorID sql.NullInt64
			)

			if err := rows.Scan(&aID, &aName, &bID, &bTitle, &bAuthorID); err != nil {
				b.Fatal(err)
			}

			author, exists := om.Get(aID)
			if !exists {
				author = &DedupeAuthor{ID: aID, Name: aName}
				om.Set(aID, author)
			}

			if bID.Valid {
				author.Books = append(author.Books, &DedupeBook{
					ID:       uint64(bID.Int64),
					Title:    bTitle.String,
					AuthorID: uint64(bAuthorID.Int64),
				})
			}
		}
		rows.Close()
	}
}

func BenchmarkBaseline_QuerySet__Scan__Authors(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&BenchmarkAuthor{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*").
		Limit(COUNT * 2)

	b.StartTimer()
	b.ResetTimer()

	for b.Loop() {
		query := qs.QueryAll()

		var count int
		for query.Next() {
			var id uint64
			var name string
			if err := query.Scan(&id, &name); err != nil {
				b.Fatal(err)
			}
			count++
		}

		query.Close()
	}
}

// allows us to see if the bottleneck is more centered around the attrs package
func BenchmarkBaseline_QuerySet__Scan__Authors__Fields(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&BenchmarkAuthor{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*").
		Limit(COUNT * 2)

	b.StartTimer()
	b.ResetTimer()

	for b.Loop() {
		query := qs.QueryAll()

		for query.Next() {
			var author = &BenchmarkAuthor{}
			var defs = attrs.Define(b.Context(), author)
			var id, _ = defs.Field("ID")
			var name, _ = defs.Field("Name")
			if err := query.Scan(id, name); err != nil {
				b.Fatal(err)
			}
		}

		query.Close()
	}
}

func BenchmarkBaseline_QuerySet__Precompiled__Scan__Authors(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&BenchmarkAuthor{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*").
		Limit(TOTAL_COUNT * 2)
	query := qs.QueryAll()

	b.StartTimer()
	b.ResetTimer()

	for b.Loop() {

		query.Reset()

		var count int
		for query.Next() {
			var id uint64
			var name string
			if err := query.Scan(&id, &name); err != nil {
				b.Fatal(err)
			}
			count++
		}

		query.Close()

		if count != COUNT {
			b.Fatalf("expected %d, got %d", COUNT, count)
		}
	}
}

func BenchmarkBaseline_QuerySet__OrderedMap_Dedupe(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&BenchmarkAuthorModel{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*", "Books.*").
		Limit(TOTAL_COUNT * 2)

	b.StartTimer()
	b.ResetTimer()

	for b.Loop() {
		query := qs.QueryAll()
		om := orderedmap.NewOrderedMap[uint64, *DedupeAuthor]()

		for query.Next() {
			var (
				aID       uint64
				aName     string
				bID       sql.NullInt64
				bTitle    sql.NullString
				bAuthorID sql.NullInt64
			)

			if err := query.Scan(&aID, &aName, &bID, &bTitle, &bAuthorID); err != nil {
				b.Fatal(err)
			}

			author, exists := om.Get(aID)
			if !exists {
				author = &DedupeAuthor{ID: aID, Name: aName}
				om.Set(aID, author)
			}

			if bID.Valid {
				author.Books = append(author.Books, &DedupeBook{
					ID:       uint64(bID.Int64),
					Title:    bTitle.String,
					AuthorID: uint64(bAuthorID.Int64),
				})
			}
		}
		query.Close()
	}
}

func BenchmarkBaseline_QuerySet__Authors(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&BenchmarkAuthor{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*").
		Limit(COUNT * 2)

	b.StartTimer()
	b.ResetTimer()

	for b.Loop() {
		var rowLen, _, err = qs.IterAll()

		if err != nil {
			b.Fatalf("error while querying objects: %v", err)
		}

		if rowLen != COUNT {
			b.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", COUNT, rowLen)
		}
	}
}

func BenchmarkBaseline_QuerySet__Books(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&BenchmarkBook{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*").
		Limit(TOTAL_COUNT * 2)

	b.StartTimer()
	b.ResetTimer()

	for b.Loop() {
		var count, _, err = qs.IterAll()

		if err != nil {
			b.Fatalf("error while querying objects: %v", err)
		}

		if count != TOTAL_BOOKS {
			b.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", TOTAL_BOOKS, count)
		}
	}
}

func TestBaseline_RawDriver(t *testing.T) {
	which, db := testdb.Open()
	ctx := drivers.SetLogSQLContext(t.Context(), false)

	rows, err := db.QueryContext(ctx, sqlx.Rebind(sqlx.BindType(which), "SELECT id, name FROM author_bench LIMIT ?"), COUNT*2)
	if err != nil {
		t.Fatal(err)
	}

	var count int
	for rows.Next() {
		var id uint64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			t.Fatal(err)
		}
		count++
	}
	rows.Close()

	if count != COUNT {
		t.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", COUNT, count)
	}
}

func TestBaseline_RawSQL_OrderedMap_Dedupe(t *testing.T) {
	which, db := testdb.Open()
	ctx := drivers.SetLogSQLContext(t.Context(), false)

	rawSQL := sqlx.Rebind(sqlx.BindType(which), `
		SELECT a.id, a.name, b.id, b.title, b.author_id 
		FROM author_bench a 
		LEFT JOIN book_bench b ON a.id = b.author_id 
		LIMIT ?
	`)

	rows, err := db.QueryContext(ctx, rawSQL, COUNT*2)
	if err != nil {
		t.Fatal(err)
	}

	om := orderedmap.NewOrderedMap[uint64, *DedupeAuthor]()

	var rawCount int
	for rows.Next() {
		rawCount++
		var (
			aID       uint64
			aName     string
			bID       sql.NullInt64
			bTitle    sql.NullString
			bAuthorID sql.NullInt64
		)

		if err := rows.Scan(&aID, &aName, &bID, &bTitle, &bAuthorID); err != nil {
			t.Fatal(err)
		}

		author, exists := om.Get(aID)
		if !exists {
			author = &DedupeAuthor{ID: aID, Name: aName}
			om.Set(aID, author)
		}

		if bID.Valid {
			author.Books = append(author.Books, &DedupeBook{
				ID:       uint64(bID.Int64),
				Title:    bTitle.String,
				AuthorID: uint64(bAuthorID.Int64),
			})
		}
	}
	rows.Close()

	if om.Len() != COUNT {
		t.Fatalf("query returned incorrect number of authors, wanted: %d, got: %d", COUNT, om.Len())
	}

	for el := om.Front(); el != nil; el = el.Next() {
		if len(el.Value.Books) != BOOKS_PER_AUTHOR {
			t.Fatalf("query returned incorrect number of books per author, wanted: %d, got: %d", BOOKS_PER_AUTHOR, len(el.Value.Books))
		}
	}
}

func TestBaseline_QuerySet__Scan__Authors(t *testing.T) {
	var qs = queries.
		GetQuerySet(&BenchmarkAuthor{}).
		WithContext(drivers.SetLogSQLContext(t.Context(), false)).
		Select("*").
		Limit(COUNT * 2)

	query := qs.QueryAll()
	var count int
	for query.Next() {
		var id uint64
		var name string
		if err := query.Scan(&id, &name); err != nil {
			t.Fatal(err)
		}
		count++
	}
	query.Close()
	if count != COUNT {
		t.Fatalf("expected %d, got %d", COUNT, count)
	}
}

func TestBaseline_QuerySet__Precompiled__Scan__Authors(t *testing.T) {
	var qs = queries.
		GetQuerySet(&BenchmarkAuthor{}).
		WithContext(drivers.SetLogSQLContext(t.Context(), false)).
		Select("*").
		Limit(TOTAL_COUNT * 2)
	query := qs.QueryAll()

	var count int
	for query.Next() {
		var id uint64
		var name string
		if err := query.Scan(&id, &name); err != nil {
			t.Fatal(err)
		}
		count++
	}
	query.Close()
	if count != COUNT {
		t.Fatalf("expected %d, got %d", COUNT, count)
	}
}

func TestBaseline_QuerySet__OrderedMap_Dedupe(t *testing.T) {
	var qs = queries.
		GetQuerySet(&BenchmarkAuthorModel{}).
		WithContext(drivers.SetLogSQLContext(t.Context(), false)).
		Select("*", "Books.*").
		Limit(TOTAL_COUNT * 2)

	query := qs.QueryAll()
	om := orderedmap.NewOrderedMap[uint64, *DedupeAuthor]()

	for query.Next() {
		var (
			aID       uint64
			aName     string
			bID       sql.NullInt64
			bTitle    sql.NullString
			bAuthorID sql.NullInt64
		)

		if err := query.Scan(&aID, &aName, &bID, &bTitle, &bAuthorID); err != nil {
			t.Fatal(err)
		}

		author, exists := om.Get(aID)
		if !exists {
			author = &DedupeAuthor{ID: aID, Name: aName}
			om.Set(aID, author)
		}

		if bID.Valid {
			author.Books = append(author.Books, &DedupeBook{
				ID:       uint64(bID.Int64),
				Title:    bTitle.String,
				AuthorID: uint64(bAuthorID.Int64),
			})
		}
	}
	query.Close()
	if om.Len() != COUNT {
		t.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", COUNT, om.Len())
	}
}

func TestBaseline_QuerySet__Authors(t *testing.T) {
	var qs = queries.
		GetQuerySet(&BenchmarkAuthor{}).
		WithContext(drivers.SetLogSQLContext(t.Context(), false)).
		Select("*").
		Limit(COUNT * 2)

	var rowLen, _, err = qs.IterAll()

	if err != nil {
		t.Fatalf("error while querying objects: %v", err)
	}

	if rowLen != COUNT {
		t.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", COUNT, rowLen)
	}
}

func TestBaseline_QuerySet__Books(t *testing.T) {
	var qs = queries.
		GetQuerySet(&BenchmarkBook{}).
		WithContext(drivers.SetLogSQLContext(t.Context(), false)).
		Select("*").
		Limit(TOTAL_COUNT * 2)

	var count, _, err = qs.IterAll()

	if err != nil {
		t.Fatalf("error while querying objects: %v", err)
	}

	if count != TOTAL_BOOKS {
		t.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", TOTAL_BOOKS, count)
	}
}

func TestBaseline_QuerySet__WithModel__Authors(t *testing.T) {
	var qs = queries.
		GetQuerySet(&BenchmarkAuthorModel{}).
		WithContext(drivers.SetLogSQLContext(t.Context(), false)).
		Select("*").
		Limit(COUNT * 2)

	var rowLen, _, err = qs.IterAll()

	if err != nil {
		t.Fatalf("error while querying objects: %v", err)
	}

	if rowLen != COUNT {
		t.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", COUNT, rowLen)
	}
}

func TestBaseline_QuerySet__WithModel__Books(t *testing.T) {
	var qs = queries.
		GetQuerySet(&BenchmarkBookModel{}).
		WithContext(drivers.SetLogSQLContext(t.Context(), false)).
		Select("*").
		Limit(TOTAL_COUNT * 2)

	var count, _, err = qs.IterAll()

	if err != nil {
		t.Fatalf("error while querying objects: %v", err)
	}

	if count != TOTAL_BOOKS {
		t.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", TOTAL_BOOKS, count)
	}
}

func BenchmarkBaseline_QuerySet__WithModel__Authors(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&BenchmarkAuthorModel{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*").
		Limit(COUNT * 2)

	b.StartTimer()
	b.ResetTimer()

	for b.Loop() {
		var rowLen, _, err = qs.IterAll()

		if err != nil {
			b.Fatalf("error while querying objects: %v", err)
		}

		if rowLen != COUNT {
			b.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", COUNT, rowLen)
		}
	}
}

func BenchmarkBaseline_QuerySet__WithModel__Books(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&BenchmarkBookModel{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*").
		Limit(TOTAL_COUNT * 2)

	b.StartTimer()
	b.ResetTimer()

	for b.Loop() {
		var count, _, err = qs.IterAll()

		if err != nil {
			b.Fatalf("error while querying objects: %v", err)
		}

		if count != TOTAL_BOOKS {
			b.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", COUNT, count)
		}
	}
}
