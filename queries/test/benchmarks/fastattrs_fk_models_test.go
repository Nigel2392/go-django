package benchmarks_test

import (
	"context"
	"fmt"
	"testing"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
)

var (
	fast_fldCnfRelAuthorModel = attrs.FieldConfig{
		Column:        "author_id",
		RelForeignKey: attrs.Relate(&FastAttrsBenchmarkAuthorModel{}, "", nil),
		Attributes: map[string]interface{}{
			attrs.AttrReverseAliasKey: "Books",
		},
	}
)

// Allows benchmarking reverse foreign key relations with
// Select("*", "Books.*")
// Preload("Books.*")
type FastAttrsBenchmarkAuthorModel struct {
	models.Model
	ID    int
	Name  string
	Books *queries.RelRevFK[attrs.Definer]
}

func (a *FastAttrsBenchmarkAuthorModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return a.Model.Define(ctx, a,
		fattrs.NewField(a, "ID", func() fattrs.FieldConfig[*FastAttrsBenchmarkAuthorModel] {
			return fattrs.FieldConfig[*FastAttrsBenchmarkAuthorModel]{
				Config:   *fast_fldCnfPrimary,
				GetValue: func(obj *FastAttrsBenchmarkAuthorModel) interface{} { return obj.ID },
				SetValue: func(obj *FastAttrsBenchmarkAuthorModel, value any) error {
					switch v := value.(type) {
					case uint64:
						obj.ID = int(v)
					case int64:
						obj.ID = int(v)
					case int:
						obj.ID = int(v)
					default:
						return fmt.Errorf("invalid ID type %T: %v", value, value)
					}
					return nil
				},
				Default: uint64(0),
			}
		}),
		fattrs.NewField(a, "Name", func() fattrs.FieldConfig[*FastAttrsBenchmarkAuthorModel] {
			return fattrs.FieldConfig[*FastAttrsBenchmarkAuthorModel]{
				Config:   attrs.FieldConfig{},
				GetValue: func(obj *FastAttrsBenchmarkAuthorModel) interface{} { return obj.Name },
				SetValue: func(obj *FastAttrsBenchmarkAuthorModel, value any) error {
					switch v := value.(type) {
					case string:
						obj.Name = v
					case []byte:
						obj.Name = string(v)
					default:
						return fmt.Errorf("invalid Name type %T: %v", value, value)
					}
					return nil
				},
				Default: "",
			}
		}),
	).WithTableName("author_bench")
}

// Allows benchmarking forward foreign key relations with
// Select("*", "Author.*")
// SelectRelated("Author.*")
type FastAttrsBenchmarkBookModel struct {
	models.Model
	ID     uint64
	Title  string
	Author *FastAttrsBenchmarkAuthorModel
}

func (b *FastAttrsBenchmarkBookModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return b.Model.Define(ctx, b,
		fattrs.NewField(b, "ID", func() fattrs.FieldConfig[*FastAttrsBenchmarkBookModel] {
			return fattrs.FieldConfig[*FastAttrsBenchmarkBookModel]{
				Config:   *fast_fldCnfPrimary,
				GetValue: func(obj *FastAttrsBenchmarkBookModel) interface{} { return obj.ID },
				SetValue: func(obj *FastAttrsBenchmarkBookModel, value any) error {
					switch v := value.(type) {
					case uint64:
						obj.ID = v
					case int64:
						obj.ID = uint64(v)
					case int:
						obj.ID = uint64(v)
					default:
						return fmt.Errorf("invalid ID type %T: %v", value, value)
					}
					return nil
				},
				Default: uint64(0),
			}
		}),
		fattrs.NewField(b, "Title", func() fattrs.FieldConfig[*FastAttrsBenchmarkBookModel] {
			return fattrs.FieldConfig[*FastAttrsBenchmarkBookModel]{
				Config:   attrs.FieldConfig{},
				GetValue: func(obj *FastAttrsBenchmarkBookModel) interface{} { return obj.Title },
				SetValue: func(obj *FastAttrsBenchmarkBookModel, value any) error {
					switch v := value.(type) {
					case string:
						obj.Title = v
					case []byte:
						obj.Title = string(v)
					default:
						return fmt.Errorf("invalid Title type %T: %v", value, value)
					}
					return nil
				},
				Default: "",
			}
		}),
		fattrs.NewField(b, "Author", func() fattrs.FieldConfig[*FastAttrsBenchmarkBookModel] {
			return fattrs.FieldConfig[*FastAttrsBenchmarkBookModel]{
				Config: fast_fldCnfRelAuthorModel,
				GetValue: func(obj *FastAttrsBenchmarkBookModel) interface{} {
					return obj.Author
				},
				SetValue: func(obj *FastAttrsBenchmarkBookModel, value any) error {
					switch v := value.(type) {
					case int64:
						obj.Author = &FastAttrsBenchmarkAuthorModel{
							ID: int(v),
						}
					case *FastAttrsBenchmarkAuthorModel:
						obj.Author = v
					default:
						return errors.TypeMismatch.Wrapf(
							"%T does not match expected author type",
							value,
						)
					}
					return nil
				},
				Default: fattrs.NULL{},
			}
		}),
	).WithTableName("book_bench")
}

func BenchmarkFastAttrsModelQuerySetForeignKeys__NoPreload__Authors(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkAuthorModel{}).
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

func BenchmarkFastAttrsModelQuerySetForeignKeys__NoPreload__Books(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkBookModel{}).
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

func BenchmarkFastAttrsModelQuerySetForeignKeys__Select__OneToX(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkAuthorModel{}).
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

		var chk *FastAttrsBenchmarkAuthorModel
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

func BenchmarkFastAttrsModelQuerySetForeignKeys__Select__OneToOne(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkBookModel{}).
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

func BenchmarkFastAttrsModelQuerySetForeignKeys__Select__OneToX__InitInLoop(b *testing.B) {

	for b.Loop() {
		var qs = queries.
			GetQuerySet(&FastAttrsBenchmarkAuthorModel{}).
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

		var chk *FastAttrsBenchmarkAuthorModel
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

func BenchmarkFastAttrsModelQuerySetForeignKeys__Select__OneToOne__InitInLoop(b *testing.B) {
	for b.Loop() {
		var qs = queries.
			GetQuerySet(&FastAttrsBenchmarkBookModel{}).
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

func BenchmarkFastAttrsModelQuerySetForeignKeys__Preload__OneToX(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkAuthorModel{}).
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

		var chk *FastAttrsBenchmarkAuthorModel
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

func BenchmarkFastAttrsModelQuerySetForeignKeys__Preload__OneToOne(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkBookModel{}).
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

func TestFastAttrsModelQuerySetForeignKeys__NoPreload__Authors(t *testing.T) {
	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkAuthorModel{}).
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

func TestFastAttrsModelQuerySetForeignKeys__NoPreload__Books(t *testing.T) {
	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkBookModel{}).
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

func TestFastAttrsModelQuerySetForeignKeys__Select__OneToX(t *testing.T) {
	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkAuthorModel{}).
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

func TestFastAttrsModelQuerySetForeignKeys__Select__OneToOne(t *testing.T) {
	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkBookModel{}).
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

func TestFastAttrsModelQuerySetForeignKeys__Preload__OneToX(t *testing.T) {
	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkAuthorModel{}).
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

func TestFastAttrsModelQuerySetForeignKeys__Preload__OneToOne(t *testing.T) {
	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkBookModel{}).
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
