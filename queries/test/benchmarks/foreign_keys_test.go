package benchmarks_test

import (
	"context"
	"testing"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

var (
	fldCnfRelAuthor = &attrs.FieldConfig{
		Column:        "author_id",
		RelForeignKey: attrs.Relate(&BenchmarkAuthor{}, "", nil),
		Attributes: map[string]interface{}{
			attrs.AttrReverseAliasKey: "Books",
		},
	}
	fldCnfRelAuthorModel = &attrs.FieldConfig{
		Column:        "author_id",
		RelForeignKey: attrs.Relate(&BenchmarkAuthorModel{}, "", nil),
		Attributes: map[string]interface{}{
			attrs.AttrReverseAliasKey: "Books",
		},
	}
)

// Allows benchmarking reverse foreign key relations with
// Select("*", "Books.*")
// Preload("Books.*")
type BenchmarkAuthorModel struct {
	models.Model
	ID    int
	Name  string
	Books *queries.RelRevFK[attrs.Definer]
}

func (a *BenchmarkAuthorModel) Fields() []attrs.Field {
	return []attrs.Field{
		attrs.NewField(a, "ID", fldCnfPrimary),
		attrs.NewField(a, "Name"),
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
		attrs.NewField(b, "ID", fldCnfPrimary),
		attrs.NewField(b, "Title"),
		attrs.NewField(b, "Author", fldCnfRelAuthorModel),
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
		attrs.NewField(a, "ID", fldCnfPrimary),
		attrs.NewField(a, "Name"),
	).WithTableName("author_bench")
}

type BenchmarkBook struct {
	ID     uint64
	Title  string
	Author *BenchmarkAuthor
}

func (b *BenchmarkBook) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, b,
		attrs.NewField(b, "ID", fldCnfPrimary),
		attrs.NewField(b, "Title"),
		attrs.NewField(b, "Author", fldCnfRelAuthor),
	).WithTableName("book_bench")
}

func BenchmarkQuerySetForeignKeys__NoPreload__Authors(b *testing.B) {
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

func BenchmarkQuerySetForeignKeys__NoPreload__Books(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&BenchmarkBookModel{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*").
		Limit(TOTAL_COUNT * 4)

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

func BenchmarkQuerySetForeignKeys__Select__OneToX(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&BenchmarkAuthorModel{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*", "Books.*").
		Limit(TOTAL_COUNT * 2)

	b.StartTimer()
	b.ResetTimer()

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
		if relLen != BOOKS_PER_AUTHOR {
			b.Fatalf("query returned incorrect number of related rows, wanted: %d, got: %d", BOOKS_PER_AUTHOR, relLen)
		}
	}
}

func BenchmarkQuerySetForeignKeys__Select__OneToOne(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&BenchmarkBookModel{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*", "Author.*").
		Limit(TOTAL_BOOKS * 2)

	b.StartTimer()
	b.ResetTimer()

	for b.Loop() {
		var rows, err = qs.All()

		if err != nil {
			b.Fatalf("error while querying objects: %v", err)
		}

		if len(rows) != TOTAL_BOOKS {
			b.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", TOTAL_BOOKS, len(rows))
		}
	}
}

func BenchmarkQuerySetForeignKeys__Preload__OneToX(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&BenchmarkAuthorModel{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*").
		Preload("Books").
		Limit(TOTAL_COUNT * 2)

	b.StartTimer()
	b.ResetTimer()

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
		if relLen != BOOKS_PER_AUTHOR {
			b.Fatalf("query returned incorrect number of related rows, wanted: %d, got: %d", BOOKS_PER_AUTHOR, relLen)
		}
	}
}

func BenchmarkQuerySetForeignKeys__Preload__OneToOne(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&BenchmarkBookModel{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*").
		SelectRelated("Author").
		Limit(TOTAL_BOOKS * 2)

	b.StartTimer()
	b.ResetTimer()

	for b.Loop() {
		var rows, err = qs.All()

		if err != nil {
			b.Fatalf("error while querying objects: %v", err)
		}

		if len(rows) != TOTAL_BOOKS {
			b.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", TOTAL_BOOKS, len(rows))
		}
	}
}

func TestQuerySetForeignKeys__NoPreload__Authors(t *testing.T) {
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

func TestQuerySetForeignKeys__NoPreload__Books(t *testing.T) {
	var qs = queries.
		GetQuerySet(&BenchmarkBookModel{}).
		WithContext(drivers.SetLogSQLContext(t.Context(), false)).
		Select("*").
		Limit(TOTAL_COUNT * 4)

	var count, _, err = qs.IterAll()
	if err != nil {
		t.Fatalf("error while querying objects: %v", err)
	}

	if count != TOTAL_BOOKS {
		t.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", TOTAL_BOOKS, count)
	}
}

func TestQuerySetForeignKeys__Select__OneToX(t *testing.T) {
	var qs = queries.
		GetQuerySet(&BenchmarkAuthorModel{}).
		WithContext(drivers.SetLogSQLContext(t.Context(), false)).
		Select("*", "Books.*").
		Limit(TOTAL_COUNT * 2)

	var rowLen, rows, err = qs.IterAll()
	if err != nil {
		t.Fatalf("error while querying objects: %v", err)
	}

	if rowLen != COUNT {
		t.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", COUNT, rowLen)
	}

	for row, err := range rows {
		if err != nil {
			t.Fatalf("error while querying objects: %v", err)
		}

		relLen := len(row.Object.Books.AsList())
		if relLen != BOOKS_PER_AUTHOR {
			t.Fatalf("query returned incorrect number of related rows, wanted: %d, got: %d", BOOKS_PER_AUTHOR, relLen)
		}
	}
}

func TestQuerySetForeignKeys__Select__OneToOne(t *testing.T) {
	var qs = queries.
		GetQuerySet(&BenchmarkBookModel{}).
		WithContext(drivers.SetLogSQLContext(t.Context(), false)).
		Select("*", "Author.*").
		Limit(TOTAL_BOOKS * 2)

	var rows, err = qs.All()
	if err != nil {
		t.Fatalf("error while querying objects: %v", err)
	}

	if len(rows) != TOTAL_BOOKS {
		t.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", TOTAL_BOOKS, len(rows))
	}

	for _, row := range rows {
		if row.Object.Author == nil {
			t.Fatalf("query returned nil related row")
		}
	}
}

func TestQuerySetForeignKeys__Preload__OneToX(t *testing.T) {
	var qs = queries.
		GetQuerySet(&BenchmarkAuthorModel{}).
		WithContext(drivers.SetLogSQLContext(t.Context(), false)).
		Select("*").
		Preload("Books").
		Limit(TOTAL_COUNT * 2)

	var rowLen, rows, err = qs.IterAll()
	if err != nil {
		t.Fatalf("error while querying objects: %v", err)
	}

	if rowLen != COUNT {
		t.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", COUNT, rowLen)
	}

	for row, err := range rows {
		if err != nil {
			t.Fatalf("error while querying objects: %v", err)
		}

		relLen := len(row.Object.Books.AsList())
		if relLen != BOOKS_PER_AUTHOR {
			t.Fatalf("query returned incorrect number of related rows, wanted: %d, got: %d", BOOKS_PER_AUTHOR, relLen)
		}
	}
}

func TestQuerySetForeignKeys__Preload__OneToOne(t *testing.T) {
	var qs = queries.
		GetQuerySet(&BenchmarkBookModel{}).
		WithContext(drivers.SetLogSQLContext(t.Context(), false)).
		Select("*").
		SelectRelated("Author").
		Limit(TOTAL_BOOKS * 2)

	var rows, err = qs.All()
	if err != nil {
		t.Fatalf("error while querying objects: %v", err)
	}

	if len(rows) != TOTAL_BOOKS {
		t.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", TOTAL_BOOKS, len(rows))
	}

	for _, row := range rows {
		if row.Object.Author == nil {
			t.Fatalf("query returned nil related row")
		}
	}
}
