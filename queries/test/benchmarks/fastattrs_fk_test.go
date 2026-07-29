package benchmarks_test

import (
	"context"
	"testing"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fastattrs"
)

var (
	fast_fldCnfRelAuthor = attrs.FieldConfig{
		Column:        "author_id",
		RelForeignKey: attrs.Relate(&FastAttrsBenchmarkAuthor{}, "", nil),
		Attributes: map[string]interface{}{
			attrs.AttrReverseAliasKey: "Books",
		},
	}
)

type FastAttrsBenchmarkAuthor struct {
	ID    uint64
	Name  string
	Books *queries.RelRevFK[attrs.Definer]
}

func (a *FastAttrsBenchmarkAuthor) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make[*FastAttrsBenchmarkAuthor, attrs.Field](ctx, a,
		fastattrs.Field(a, "ID", &a.ID, func() fastattrs.PtrFieldConfig[*FastAttrsBenchmarkAuthor, uint64] {
			return fastattrs.PtrFieldConfig[*FastAttrsBenchmarkAuthor, uint64]{
				Config: *fast_fldCnfPrimary,
			}
		}),
		fastattrs.Field(a, "Name", &a.Name, func() fastattrs.PtrFieldConfig[*FastAttrsBenchmarkAuthor, string] {
			return fastattrs.PtrFieldConfig[*FastAttrsBenchmarkAuthor, string]{
				Config: attrs.FieldConfig{},
			}
		}),
		fastattrs.Field(a, "Books", &a.Books, func() fastattrs.PtrFieldConfig[*FastAttrsBenchmarkAuthor, *queries.RelRevFK[attrs.Definer]] {
			return fastattrs.PtrFieldConfig[*FastAttrsBenchmarkAuthor, *queries.RelRevFK[attrs.Definer]]{
				//	Config: attrs.FieldConfig{
				//		RelForeignKeyReverse: attrs.Relate(
				//			&FastAttrsBenchmarkBook{},
				//			"Author",
				//			nil,
				//		),
				//	},
			}
		}),
		//		fields.NewManyToManyField[*queries.RelRevFK[attrs.Definer]](a, "Books", &fields.FieldConfig{
		//			ScanTo:    &a.Books,
		//			IsReverse: true,
		//		}),
	).WithTableName("author_bench")
}

type FastAttrsBenchmarkBook struct {
	ID     uint64
	Title  string
	Author *FastAttrsBenchmarkAuthor
}

func (b *FastAttrsBenchmarkBook) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make[*FastAttrsBenchmarkBook, attrs.Field](ctx, b,
		fastattrs.Field(b, "ID", &b.ID, func() fastattrs.PtrFieldConfig[*FastAttrsBenchmarkBook, uint64] {
			return fastattrs.PtrFieldConfig[*FastAttrsBenchmarkBook, uint64]{
				Config: *fast_fldCnfPrimary,
			}
		}),
		fastattrs.Field(b, "Title", &b.Title, func() fastattrs.PtrFieldConfig[*FastAttrsBenchmarkBook, string] {
			return fastattrs.PtrFieldConfig[*FastAttrsBenchmarkBook, string]{
				Config: attrs.FieldConfig{},
			}
		}),
		fastattrs.Field(b, "Author", &b.Author, func() fastattrs.PtrFieldConfig[*FastAttrsBenchmarkBook, *FastAttrsBenchmarkAuthor] {
			return fastattrs.PtrFieldConfig[*FastAttrsBenchmarkBook, *FastAttrsBenchmarkAuthor]{
				Config:  fast_fldCnfRelAuthor,
				Default: fastattrs.NULL{},
				New: func() *FastAttrsBenchmarkAuthor {
					return &FastAttrsBenchmarkAuthor{}
				},
			}
		}),
	).WithTableName("book_bench")
}

func BenchmarkFastAttrsQuerySetForeignKeys__NoPreload__Authors(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkAuthor{}).
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

func BenchmarkFastAttrsQuerySetForeignKeys__NoPreload__Books(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkBook{}).
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

func BenchmarkFastAttrsQuerySetForeignKeys__Select__OneToX(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkAuthor{}).
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

		var chk *FastAttrsBenchmarkAuthor
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

func BenchmarkFastAttrsQuerySetForeignKeys__Select__OneToOne(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkBook{}).
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

func BenchmarkFastAttrsQuerySetForeignKeys__Select__OneToX__InitInLoop(b *testing.B) {

	for b.Loop() {
		var qs = queries.
			GetQuerySet(&FastAttrsBenchmarkAuthor{}).
			WithContext(drivers.SetLogSQLContext(b.Context(), false)).
			Select("*", "Books.*").
			Limit(TOTAL_COUNT * 2)

		var rowLen, rows, err = qs.IterAll()

		if err != nil {
			b.Fatalf("error while querying objects: %v", err)
		}

		if rowLen != COUNT {
			b.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", COUNT, rowLen)
		}

		var chk *FastAttrsBenchmarkAuthor
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

func BenchmarkFastAttrsQuerySetForeignKeys__Select__OneToOne__InitInLoop(b *testing.B) {
	for b.Loop() {
		var qs = queries.
			GetQuerySet(&FastAttrsBenchmarkBook{}).
			WithContext(drivers.SetLogSQLContext(b.Context(), false)).
			Select("*", "Author.*").
			Limit(TOTAL_BOOKS * 2)

		var rows, err = qs.All()

		if err != nil {
			b.Fatalf("error while querying objects: %v", err)
		}

		if len(rows) != TOTAL_BOOKS {
			b.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", TOTAL_BOOKS, len(rows))
		}
	}
}

func BenchmarkFastAttrsQuerySetForeignKeys__Preload__OneToX(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkAuthor{}).
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

		var chk *FastAttrsBenchmarkAuthor
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

func BenchmarkFastAttrsQuerySetForeignKeys__Preload__OneToOne(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkBook{}).
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

func TestFastAttrsQuerySetForeignKeys__NoPreload__Authors(t *testing.T) {
	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkAuthor{}).
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

func TestFastAttrsQuerySetForeignKeys__NoPreload__Books(t *testing.T) {
	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkBook{}).
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

func TestFastAttrsQuerySetForeignKeys__Select__OneToX(t *testing.T) {
	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkAuthor{}).
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

func TestFastAttrsQuerySetForeignKeys__Select__OneToOne(t *testing.T) {
	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkBook{}).
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

func TestFastAttrsQuerySetForeignKeys__Preload__OneToX(t *testing.T) {
	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkAuthor{}).
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

func TestFastAttrsQuerySetForeignKeys__Preload__OneToOne(t *testing.T) {
	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkBook{}).
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
