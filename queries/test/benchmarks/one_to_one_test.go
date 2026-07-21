package benchmarks_test

import (
	"context"
	"testing"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/fields"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

func init() {
	attrs.RegisterModel(&BenchmarkO2OMain{})
	attrs.RegisterModel(&BenchmarkO2OTarget{})
	attrs.RegisterModel(&BenchmarkO2OThrough{})
}

type BenchmarkO2OMain struct {
	models.Model
	ID      uint64
	Title   string
	Through *queries.RelO2O[*BenchmarkO2OTarget, *BenchmarkO2OThrough]
}

func (t *BenchmarkO2OMain) Fields() []attrs.Field {
	return []attrs.Field{
		attrs.NewField(t, "ID", &attrs.FieldConfig{Primary: true}),
		attrs.NewField(t, "Title", nil),
		fields.NewOneToOneField[*queries.RelO2O[*BenchmarkO2OTarget, *BenchmarkO2OThrough]](t, "Through", &fields.FieldConfig{
			ScanTo:      &t.Through,
			ReverseName: "TargetReverse",
			Rel: attrs.Relate(
				&BenchmarkO2OTarget{},
				"",
				&attrs.ThroughModel{
					This:   &BenchmarkO2OThrough{},
					Source: "SourceModel",
					Target: "TargetModel",
				},
			),
		}),
	}
}

func (t *BenchmarkO2OMain) FieldDefs(ctx context.Context) attrs.Definitions {
	return t.Model.Define(ctx, t, t.Fields).WithTableName("o2o_main_bench")
}

type BenchmarkO2OThrough struct {
	ID          uint64
	SourceModel uint64
	TargetModel uint64
}

func (t *BenchmarkO2OThrough) Fields() []attrs.Field {
	return []attrs.Field{
		attrs.NewField(t, "ID", &attrs.FieldConfig{Primary: true}),
		attrs.NewField(t, "SourceModel", &attrs.FieldConfig{
			Column:     "source_id",
			Attributes: map[string]interface{}{attrs.AttrUniqueKey: true},
		}),
		attrs.NewField(t, "TargetModel", &attrs.FieldConfig{
			Column:     "target_id",
			Attributes: map[string]interface{}{attrs.AttrUniqueKey: true},
		}),
	}
}

func (t *BenchmarkO2OThrough) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, t, t.Fields).WithTableName("o2o_through_bench")
}

type BenchmarkO2OTarget struct {
	models.Model
	ID            uint64
	Name          string
	TargetReverse *queries.RelO2O[attrs.Definer, attrs.Definer]
}

func (t *BenchmarkO2OTarget) Fields() []attrs.Field {
	return []attrs.Field{
		attrs.NewField(t, "ID", &attrs.FieldConfig{Primary: true}),
		attrs.NewField(t, "Name", nil),
	}
}

func (t *BenchmarkO2OTarget) FieldDefs(ctx context.Context) attrs.Definitions {
	return t.Model.Define(ctx, t, t.Fields).WithTableName("o2o_target_bench")
}

func BenchmarkQuerySetOneToOne__Select(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&BenchmarkO2OMain{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*", "Through.*").
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

func BenchmarkQuerySetOneToOne__Preload(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&BenchmarkO2OMain{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*").
		SelectRelated("Through").
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

func BenchmarkQuerySetOneToOne__Select__Reverse(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&BenchmarkO2OTarget{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*", "TargetReverse.*").
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

func BenchmarkQuerySetOneToOne__Preload__Reverse(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&BenchmarkO2OTarget{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*").
		SelectRelated("TargetReverse").
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

func TestQuerySetOneToOne__Select(t *testing.T) {
	var qs = queries.
		GetQuerySet(&BenchmarkO2OMain{}).
		WithContext(drivers.SetLogSQLContext(t.Context(), false)).
		Select("*", "Through.*").
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

		if row.Object.Through == nil || row.Object.Through.Object == nil {
			t.Fatalf("query returned nil related row")
		}
	}
}

func TestQuerySetOneToOne__Preload(t *testing.T) {
	var qs = queries.
		GetQuerySet(&BenchmarkO2OMain{}).
		WithContext(drivers.SetLogSQLContext(t.Context(), false)).
		Select("*").
		SelectRelated("Through").
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

		if row.Object.Through == nil || row.Object.Through.Object == nil {
			t.Fatalf("query returned nil related row")
		}
	}
}

func TestQuerySetOneToOne__Select__Reverse(t *testing.T) {
	var qs = queries.
		GetQuerySet(&BenchmarkO2OTarget{}).
		WithContext(drivers.SetLogSQLContext(t.Context(), false)).
		Select("*", "TargetReverse.*").
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

		if row.Object.TargetReverse == nil || row.Object.TargetReverse.Model() == nil || row.Object.TargetReverse.Through() == nil {
			t.Fatalf("query returned nil or empty related row")
		}
	}
}

func TestQuerySetOneToOne__Preload__Reverse(t *testing.T) {
	var qs = queries.
		GetQuerySet(&BenchmarkO2OTarget{}).
		WithContext(drivers.SetLogSQLContext(t.Context(), false)).
		Select("*").
		SelectRelated("TargetReverse").
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

		if row.Object.TargetReverse == nil || row.Object.TargetReverse.Model() == nil || row.Object.TargetReverse.Through() == nil {
			t.Fatalf("query returned nil or empty related row")
		}
	}
}
