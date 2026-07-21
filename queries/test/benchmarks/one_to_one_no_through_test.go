package benchmarks_test

import (
	"context"
	"testing"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

func init() {
	attrs.RegisterModel(&BenchmarkO2ONoThroughMain{})
	attrs.RegisterModel(&BenchmarkO2ONoThroughTarget{})
}

type BenchmarkO2ONoThroughTarget struct {
	models.Model
	ID          uint64
	Name        string
	MainReverse *BenchmarkO2ONoThroughMain
}

func (t *BenchmarkO2ONoThroughTarget) Fields() []attrs.Field {
	return []attrs.Field{
		attrs.NewField(t, "ID", &attrs.FieldConfig{Primary: true}),
		attrs.NewField(t, "Name", nil),
	}
}

func (t *BenchmarkO2ONoThroughTarget) FieldDefs(ctx context.Context) attrs.Definitions {
	return t.Model.Define(ctx, t, t.Fields).WithTableName("o2o_nt_target_bench")
}

type BenchmarkO2ONoThroughMain struct {
	models.Model
	ID     uint64
	Title  string
	Target *BenchmarkO2ONoThroughTarget
}

func (t *BenchmarkO2ONoThroughMain) Fields() []attrs.Field {
	return []attrs.Field{
		attrs.NewField(t, "ID", &attrs.FieldConfig{Primary: true}),
		attrs.NewField(t, "Title", nil),
		attrs.NewField(t, "Target", &attrs.FieldConfig{
			Column:      "target_id",
			RelOneToOne: attrs.Relate(&BenchmarkO2ONoThroughTarget{}, "", nil),
			Attributes: map[string]interface{}{
				attrs.AttrReverseAliasKey: "MainReverse",
			},
		}),
	}
}

func (t *BenchmarkO2ONoThroughMain) FieldDefs(ctx context.Context) attrs.Definitions {
	return t.Model.Define(ctx, t, t.Fields).WithTableName("o2o_nt_main_bench")
}

// BENCHMARKS
func BenchmarkQuerySetO2ONoThrough__Select(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&BenchmarkO2ONoThroughMain{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*", "Target.*").
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

func BenchmarkQuerySetO2ONoThrough__Preload(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&BenchmarkO2ONoThroughMain{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*").
		SelectRelated("Target").
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

func BenchmarkQuerySetO2ONoThrough__Select__Reverse(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&BenchmarkO2ONoThroughTarget{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*", "MainReverse.*").
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

func BenchmarkQuerySetO2ONoThrough__Preload__Reverse(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&BenchmarkO2ONoThroughTarget{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*").
		SelectRelated("MainReverse").
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

// TESTS

var o2oTestContext = drivers.SetLogSQLContext(context.Background(), false)

func TestQuerySetO2ONoThrough__Select(t *testing.T) {
	var qs = queries.
		GetQuerySet(&BenchmarkO2ONoThroughMain{}).
		WithContext(o2oTestContext).
		Select("*", "Target.*").
		Limit(COUNT * 2)

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

		if row.Object.Target == nil {
			t.Fatalf("query returned nil related row")
		}
	}
}

func TestQuerySetO2ONoThrough__Preload(t *testing.T) {
	var qs = queries.
		GetQuerySet(&BenchmarkO2ONoThroughMain{}).
		WithContext(o2oTestContext).
		Select("*").
		SelectRelated("Target").
		Limit(COUNT * 2)

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

		if row.Object.Target == nil {
			t.Fatalf("query returned nil related row")
		}
	}
}

func TestQuerySetO2ONoThrough__Select__Reverse(t *testing.T) {
	var qs = queries.
		GetQuerySet(&BenchmarkO2ONoThroughTarget{}).
		WithContext(o2oTestContext).
		Select("*", "MainReverse.*").
		Limit(COUNT * 2)

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
		if row.Object.MainReverse == nil {
			t.Fatalf("query returned nil or empty related row")
		}
	}
}

func TestQuerySetO2ONoThrough__Preload__Reverse(t *testing.T) {
	var qs = queries.
		GetQuerySet(&BenchmarkO2ONoThroughTarget{}).
		WithContext(o2oTestContext).
		Select("*").
		SelectRelated("MainReverse").
		Limit(COUNT * 2)

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
		if row.Object.MainReverse == nil {
			t.Fatalf("query returned nil or empty related row")
		}
	}
}

func TestQuerySetO2ONoThrough__Preload__Reverse__Deep(t *testing.T) {
	var qs = queries.
		GetQuerySet(&BenchmarkO2ONoThroughTarget{}).
		WithContext(o2oTestContext).
		Select("*").
		SelectRelated("MainReverse", "MainReverse.Target").
		Limit(COUNT * 2)

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
		if row.Object.MainReverse == nil {
			t.Fatalf("query returned nil or empty related row")
		}
	}
}
