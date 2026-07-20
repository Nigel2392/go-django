package queries_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/Nigel2392/go-django/djester"
	"github.com/Nigel2392/go-django/djester/quest"
	"github.com/Nigel2392/go-django/djester/testdb"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/elliotchance/orderedmap/v2"
	"github.com/jmoiron/sqlx"
)

func init() {
	attrs.RegisterModel(&BenchmarkAuthorModel{})
	attrs.RegisterModel(&BenchmarkBookModel{})
}

const COUNT = 500

// Allows benchmarking reverse foreign key relations with
// Select("*", "Books.*")
// Preload("Books.*")
type BenchmarkAuthorModel struct {
	models.Model
	ID    uint64
	Name  string
	Books *queries.RelRevFK[attrs.Definer]
}

func (a *BenchmarkAuthorModel) Fields() []attrs.Field {
	return []attrs.Field{
		attrs.NewField(a, "ID", &attrs.FieldConfig{
			Primary: true,
		}),
		attrs.NewField(a, "Name", nil),
	}
}
func (a *BenchmarkAuthorModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return a.Model.Define(ctx, a, a.Fields).WithTableName("author_bench")
}

// Allows benchmarking forward foreign key relations with
// Select("*", "Author.*")
// SelectRelated("Author.*")
type BenchmarkBookModel struct {
	models.Model
	ID     uint64
	Title  string
	Author *BenchmarkAuthorModel
}

func (b *BenchmarkBookModel) Fields() []attrs.Field {
	return []attrs.Field{
		attrs.NewField(b, "ID", &attrs.FieldConfig{
			Primary: true,
		}),
		attrs.NewField(b, "Title", nil),
		attrs.NewField(b, "Author", &attrs.FieldConfig{
			Column:        "author_id",
			RelForeignKey: attrs.Relate(&BenchmarkAuthorModel{}, "", nil),
			Attributes: map[string]interface{}{
				attrs.AttrReverseAliasKey: "Books",
			},
		}),
	}
}

func (b *BenchmarkBookModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return b.Model.Define(ctx, b, b.Fields).WithTableName("book_bench")
}

type BenchmarkAuthor struct {
	ID   uint64
	Name string
}

func (a *BenchmarkAuthor) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, a,
		attrs.NewField(a, "ID", &attrs.FieldConfig{
			Primary: true,
		}),
		attrs.NewField(a, "Name", nil),
	).WithTableName("author_bench")
}

type BenchmarkBook struct {
	ID     uint64
	Title  string
	Author *BenchmarkAuthor
}

func (b *BenchmarkBook) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, b,
		attrs.NewField(b, "ID", &attrs.FieldConfig{
			Primary: true,
		}),
		attrs.NewField(b, "Title", nil),
		attrs.NewField(b, "Author", &attrs.FieldConfig{
			Column:        "author_id",
			RelForeignKey: attrs.Relate(&BenchmarkAuthor{}, "", nil),
			Attributes: map[string]interface{}{
				attrs.AttrReverseAliasKey: "Books",
			},
		}),
	).WithTableName("book_bench")
}

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

func setupBenchmark(b *testing.B, extraRelatedBooks bool) func() error {
	attrs.ALLOW_METHOD_CHECKS = false

	var bookCount = COUNT
	if extraRelatedBooks {
		bookCount *= 2
	}

	var (
		authors = make([]*BenchmarkAuthor, COUNT)
		books   = make([]*BenchmarkBook, bookCount)
	)

	for i := range COUNT {
		authors[i] = &BenchmarkAuthor{
			Name: fmt.Sprintf("BenchmarkAuthor #%d", i),
		}

		books[i] = &BenchmarkBook{
			Title:  fmt.Sprintf("Benchmark Book #%d", i),
			Author: authors[i],
		}
	}

	if extraRelatedBooks {
		for i := range COUNT {
			books[i+COUNT] = &BenchmarkBook{
				Title:  fmt.Sprintf("Special Edition #%d", i),
				Author: authors[0],
			}
		}
	}

	authors, authorDelete := quest.CreateObjects(djester.BW(b), authors...)
	books, bookDelete := quest.CreateObjects(djester.BW(b), books...)

	return func() error {
		return errors.Join(bookDelete(0), authorDelete(0))
	}
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

	delete := setupBenchmark(b, true)
	defer delete()

	which, db := testdb.Open()
	ctx := drivers.SetLogSQLContext(b.Context(), false)

	rawSQL := sqlx.Rebind(sqlx.BindType(which), `
		SELECT a.id, a.name, b.id, b.title, b.author_id 
		FROM author_bench a 
		LEFT JOIN book_bench b ON a.id = b.author_id 
		LIMIT ?
	`)

	b.StartTimer()

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

func BenchmarkQuerySet__Scan__Authors(b *testing.B) {
	b.StopTimer()

	delete := setupBenchmark(b, false)
	defer delete()

	var qs = queries.
		GetQuerySet(&BenchmarkAuthor{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*").
		Limit(COUNT * 2)

	b.StartTimer()

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

func BenchmarkBaseline_QuerySet_OrderedMap_Dedupe(b *testing.B) {
	b.StopTimer()

	delete := setupBenchmark(b, true)
	defer delete()

	var qs = queries.
		GetQuerySet(&BenchmarkAuthorModel{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*", "Books.*").
		Limit(COUNT * 2)

	b.StartTimer()

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

func BenchmarkQuerySet__WithoutRelation__Authors(b *testing.B) {
	b.StopTimer()

	delete := setupBenchmark(b, false)
	defer delete()

	var qs = queries.
		GetQuerySet(&BenchmarkAuthor{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*").
		Limit(COUNT * 2)

	b.StartTimer()

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

func BenchmarkQuerySet__WithoutRelation__Books(b *testing.B) {
	b.StopTimer()

	delete := setupBenchmark(b, false)
	defer delete()

	var qs = queries.
		GetQuerySet(&BenchmarkBook{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*").
		Limit(COUNT * 2)

	b.StartTimer()

	for b.Loop() {
		var count, _, err = qs.IterAll()

		if err != nil {
			b.Fatalf("error while querying objects: %v", err)
		}

		if count != COUNT {
			b.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", COUNT, count)
		}
	}
}

func BenchmarkQuerySetWithModel__WithoutRelation__Authors(b *testing.B) {
	b.StopTimer()

	delete := setupBenchmark(b, false)
	defer delete()

	var qs = queries.
		GetQuerySet(&BenchmarkAuthorModel{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*").
		Limit(COUNT * 2)

	b.StartTimer()

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

func BenchmarkQuerySetWithModel__WithoutRelation__Books(b *testing.B) {
	b.StopTimer()

	delete := setupBenchmark(b, false)
	defer delete()

	var qs = queries.
		GetQuerySet(&BenchmarkBookModel{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*").
		Limit(COUNT * 2)

	b.StartTimer()

	for b.Loop() {
		var count, _, err = qs.IterAll()

		if err != nil {
			b.Fatalf("error while querying objects: %v", err)
		}

		if count != COUNT {
			b.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", COUNT, count)
		}
	}
}

func BenchmarkQuerySetForeignKeys__NoPreload__Authors(b *testing.B) {
	b.StopTimer()

	delete := setupBenchmark(b, true)
	defer delete()

	var qs = queries.
		GetQuerySet(&BenchmarkAuthorModel{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*").
		Limit(COUNT * 2)

	b.StartTimer()

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

func BenchmarkQuerySetForeignKeys__NoPreload__Books(b *testing.B) {
	b.StopTimer()

	delete := setupBenchmark(b, true)
	defer delete()

	var qs = queries.
		GetQuerySet(&BenchmarkBookModel{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*").
		Limit(COUNT * 4)

	b.StartTimer()

	for b.Loop() {
		var count, _, err = qs.IterAll()

		if err != nil {
			b.Fatalf("error while querying objects: %v", err)
		}

		if count != COUNT*2 {
			b.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", COUNT*2, count)
		}
	}
}

func BenchmarkQuerySetForeignKeys__Select__OneToX(b *testing.B) {
	b.StopTimer()

	delete := setupBenchmark(b, true)
	defer delete()

	var qs = queries.
		GetQuerySet(&BenchmarkAuthorModel{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*", "Books.*").
		Limit(COUNT * 2)

	b.StartTimer()

	for b.Loop() {
		var rowLen, rows, err = qs.IterAll()

		if err != nil {
			b.Fatalf("error while querying objects: %v", err)
		}

		if rowLen != COUNT {
			b.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", COUNT, rowLen)
		}

		var chk *BenchmarkAuthorModel
		for row, err := range rows {
			if err != nil {
				b.Fatalf("error while querying objects: %v", err)
			}
			chk = row.Object
			break
		}
		relLen := len(chk.Books.AsList())
		if relLen != COUNT+1 {
			b.Fatalf("query returned incorrect number of related rows, wanted: %d, got: %d", COUNT+1, relLen)
		}
	}
}

func BenchmarkQuerySetForeignKeys__Select__OneToOne(b *testing.B) {
	b.StopTimer()

	delete := setupBenchmark(b, true)
	defer delete()

	var qs = queries.
		GetQuerySet(&BenchmarkBookModel{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*", "Author.*").
		Limit(COUNT * 2)

	b.StartTimer()

	for b.Loop() {
		var rows, err = qs.All()

		if err != nil {
			b.Fatalf("error while querying objects: %v", err)
		}

		if len(rows) != COUNT*2 {
			b.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", COUNT*2, len(rows))
		}
	}
}

func BenchmarkQuerySetForeignKeys__Preload__OneToX(b *testing.B) {
	b.StopTimer()

	delete := setupBenchmark(b, true)
	defer delete()

	var qs = queries.
		GetQuerySet(&BenchmarkAuthorModel{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*").
		Preload("Books").
		Limit(COUNT * 2)

	b.StartTimer()

	for b.Loop() {
		var rowLen, rows, err = qs.IterAll()

		if err != nil {
			b.Fatalf("error while querying objects: %v", err)
		}

		if rowLen != COUNT {
			b.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", COUNT, rowLen)
		}

		var chk *BenchmarkAuthorModel
		for row, err := range rows {
			if err != nil {
				b.Fatalf("error while querying objects: %v", err)
			}
			chk = row.Object
			break
		}
		relLen := len(chk.Books.AsList())
		if relLen != COUNT+1 {
			b.Fatalf("query returned incorrect number of related rows, wanted: %d, got: %d", COUNT+1, relLen)
		}
	}
}

func BenchmarkQuerySetForeignKeys__Preload__OneToOne(b *testing.B) {
	b.StopTimer()

	delete := setupBenchmark(b, true)
	defer delete()

	var qs = queries.
		GetQuerySet(&BenchmarkBookModel{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*").
		SelectRelated("Author").
		Limit(COUNT * 2)

	b.StartTimer()
	for b.Loop() {
		var rows, err = qs.All()

		if err != nil {
			b.Fatalf("error while querying objects: %v", err)
		}

		if len(rows) != COUNT*2 {
			b.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", COUNT*2, len(rows))
		}
	}
}
