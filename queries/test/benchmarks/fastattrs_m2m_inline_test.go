package benchmarks_test

import (
	"context"
	"testing"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"

	// Assuming fattrs is importable here based on your structure
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
)

var (
	fast_inline_fldCnfPrimary = &attrs.FieldConfig{Primary: true}

	fast_inline_m2mRel = attrs.Relate(
		&FastAttrsInlineBenchmarkM2MTarget{},
		"", &attrs.ThroughModel{
			This:   &FastAttrsInlineBenchmarkM2MThrough{},
			Source: "SourceModel",
			Target: "TargetModel",
		},
	)
)

// -------------------------------------------------------------------------
// MANY-TO-MANY MODELS
// -------------------------------------------------------------------------

type FastAttrsInlineBenchmarkM2MSource struct {
	models.Model
	ID     uint64
	Title  string
	Target *queries.RelM2M[*FastAttrsInlineBenchmarkM2MTarget, *FastAttrsInlineBenchmarkM2MThrough]
}

func (m *FastAttrsInlineBenchmarkM2MSource) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*FastAttrsInlineBenchmarkM2MSource, uint64] {
			return fattrs.PtrFieldConfig[*FastAttrsInlineBenchmarkM2MSource, uint64]{
				Config:  *fast_inline_fldCnfPrimary,
				Default: uint64(0),
			}
		}),
		fattrs.Field(m, "Title", &m.Title, func() fattrs.PtrFieldConfig[*FastAttrsInlineBenchmarkM2MSource, string] {
			return fattrs.PtrFieldConfig[*FastAttrsInlineBenchmarkM2MSource, string]{}
		}),
		fattrs.Field(m, "Target", &m.Target, func() fattrs.PtrFieldConfig[*FastAttrsInlineBenchmarkM2MSource, *queries.RelM2M[*FastAttrsInlineBenchmarkM2MTarget, *FastAttrsInlineBenchmarkM2MThrough]] {
			return fattrs.PtrFieldConfig[*FastAttrsInlineBenchmarkM2MSource, *queries.RelM2M[*FastAttrsInlineBenchmarkM2MTarget, *FastAttrsInlineBenchmarkM2MThrough]]{
				Config: attrs.FieldConfig{
					RelManyToMany: fast_inline_m2mRel,
					Attributes: map[string]interface{}{
						attrs.AttrReverseAliasKey: "TargetReverse",
					},
				},
			}
		}),
	).WithTableName("fast_m2m_source_bench")
}

type FastAttrsInlineBenchmarkM2MTarget struct {
	models.Model
	ID            uint64
	Name          string
	TargetReverse *queries.RelM2M[attrs.Definer, attrs.Definer]
}

func (m *FastAttrsInlineBenchmarkM2MTarget) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*FastAttrsInlineBenchmarkM2MTarget, uint64] {
			return fattrs.PtrFieldConfig[*FastAttrsInlineBenchmarkM2MTarget, uint64]{
				Config: *fast_inline_fldCnfPrimary,
			}
		}),
		fattrs.Field(m, "Name", &m.Name, func() fattrs.PtrFieldConfig[*FastAttrsInlineBenchmarkM2MTarget, string] {
			return fattrs.PtrFieldConfig[*FastAttrsInlineBenchmarkM2MTarget, string]{}
		}),
	).WithTableName("fast_m2m_target_bench")
}

type FastAttrsInlineBenchmarkM2MThrough struct {
	ID          uint64
	SourceModel uint64
	TargetModel uint64
}

func (m *FastAttrsInlineBenchmarkM2MThrough) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*FastAttrsInlineBenchmarkM2MThrough, uint64] {
			return fattrs.PtrFieldConfig[*FastAttrsInlineBenchmarkM2MThrough, uint64]{
				Config: *fast_inline_fldCnfPrimary,
			}
		}),
		fattrs.Field(m, "SourceModel", &m.SourceModel, func() fattrs.PtrFieldConfig[*FastAttrsInlineBenchmarkM2MThrough, uint64] {
			return fattrs.PtrFieldConfig[*FastAttrsInlineBenchmarkM2MThrough, uint64]{
				Config: attrs.FieldConfig{
					Column: "source_id",
				},
			}
		}),
		fattrs.Field(m, "TargetModel", &m.TargetModel, func() fattrs.PtrFieldConfig[*FastAttrsInlineBenchmarkM2MThrough, uint64] {
			return fattrs.PtrFieldConfig[*FastAttrsInlineBenchmarkM2MThrough, uint64]{
				Config: attrs.FieldConfig{
					Column: "target_id",
				},
			}
		}),
	).WithTableName("fast_m2m_through_bench")
}

// -------------------------------------------------------------------------
// MANY-TO-MANY BENCHMARKS
// -------------------------------------------------------------------------

func BenchmarkFastAttrsInlineQuerySetManyToMany__NoPreload(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&FastAttrsInlineBenchmarkM2MSource{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*").
		Limit(M2M_SOURCES_COUNT * 2)

	b.StartTimer()
	b.ResetTimer()

	for b.Loop() {
		var rowLen, _, err = qs.IterAll()

		if err != nil {
			b.Fatalf("error while querying objects: %v", err)
		}

		if rowLen != M2M_SOURCES_COUNT {
			b.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", M2M_SOURCES_COUNT, rowLen)
		}
	}
}

func BenchmarkFastAttrsInlineQuerySetManyToMany__Select(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&FastAttrsInlineBenchmarkM2MSource{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*", "Target.*").
		Limit(TOTAL_M2M_THROUGHS)

	b.StartTimer()
	b.ResetTimer()

	for b.Loop() {
		var rowLen, rows, err = qs.IterAll()

		if err != nil {
			b.Fatalf("error while querying objects: %v", err)
		}

		if rowLen != M2M_SOURCES_COUNT {
			b.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", M2M_SOURCES_COUNT, rowLen)
		}

		var chk *FastAttrsInlineBenchmarkM2MSource
		for row, err := range rows {
			if err != nil {
				b.Fatalf("error while querying objects: %v", err)
			}
			chk = row.Object
			break
		}

		relLen := len(chk.Target.AsList())
		if relLen != M2M_TARGETS_PER_SOURCE {
			b.Fatalf("query returned incorrect number of related rows, wanted: %d, got: %d", M2M_TARGETS_PER_SOURCE, relLen)
		}
	}
}

func BenchmarkFastAttrsInlineQuerySetManyToMany__Select__Deep(b *testing.B) {
	b.StopTimer()

	// Forces the ORM to JOIN through the target and back out to the reverse relation
	var qs = queries.
		GetQuerySet(&FastAttrsInlineBenchmarkM2MSource{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*", "Target.*", "Target.TargetReverse.*").
		Limit(TOTAL_M2M_THROUGHS * TOTAL_M2M_THROUGHS)

	b.StartTimer()
	b.ResetTimer()

	for b.Loop() {
		var rowLen, _, err = qs.IterAll()

		if err != nil {
			b.Fatalf("error while querying objects: %v", err)
		}

		if rowLen != M2M_SOURCES_COUNT {
			b.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", M2M_SOURCES_COUNT, rowLen)
		}
	}
}

func BenchmarkFastAttrsInlineQuerySetManyToMany__Preload(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&FastAttrsInlineBenchmarkM2MSource{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*").
		Preload("Target").
		Limit(M2M_SOURCES_COUNT * 2)

	b.StartTimer()
	b.ResetTimer()

	for b.Loop() {
		var rowLen, rows, err = qs.IterAll()

		if err != nil {
			b.Fatalf("error while querying objects: %v", err)
		}

		if rowLen != M2M_SOURCES_COUNT {
			b.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", M2M_SOURCES_COUNT, rowLen)
		}

		var chk *FastAttrsInlineBenchmarkM2MSource
		for row, err := range rows {
			if err != nil {
				b.Fatalf("error while querying objects: %v", err)
			}
			chk = row.Object
			break
		}

		relLen := len(chk.Target.AsList())
		if relLen != M2M_TARGETS_PER_SOURCE {
			b.Fatalf("query returned incorrect number of related rows, wanted: %d, got: %d", M2M_TARGETS_PER_SOURCE, relLen)
		}
	}
}

func BenchmarkFastAttrsInlineQuerySetManyToMany__Preload__Deep(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&FastAttrsInlineBenchmarkM2MSource{}).
		WithContext(drivers.SetLogSQLContext(b.Context(), false)).
		Select("*").
		Preload("Target", "Target.TargetReverse").
		Limit(M2M_SOURCES_COUNT * 2)

	b.StartTimer()
	b.ResetTimer()

	for b.Loop() {
		var rowLen, rows, err = qs.IterAll()

		if err != nil {
			b.Fatalf("error while querying objects: %v", err)
		}

		if rowLen != M2M_SOURCES_COUNT {
			b.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", M2M_SOURCES_COUNT, rowLen)
		}

		var chk *FastAttrsInlineBenchmarkM2MSource
		for row, err := range rows {
			if err != nil {
				b.Fatalf("error while querying objects: %v", err)
			}
			chk = row.Object
			break
		}

		lst := chk.Target.AsList()
		relLen := len(lst)
		if relLen != M2M_TARGETS_PER_SOURCE {
			b.Fatalf("query returned incorrect number of related rows, wanted: %d, got: %d", M2M_TARGETS_PER_SOURCE, relLen)
		}

		if M2M_TARGETS_COUNT > 0 && M2M_TARGETS_PER_SOURCE > 0 {
			var chkDst *FastAttrsInlineBenchmarkM2MTarget
			for _, target := range lst {
				chkDst = target.Object
				break
			}

			relLen = len(chkDst.TargetReverse.AsList())
			if relLen != M2M_TARGETS_PER_SOURCE {
				b.Fatalf("query returned incorrect number of related rows, wanted: %d, got: %d", M2M_TARGETS_PER_SOURCE, relLen)
			}
		}
	}
}

func TestFastAttrsInlineManyToMany__Preload(t *testing.T) {

	var qs = queries.
		GetQuerySet(&FastAttrsInlineBenchmarkM2MSource{}).
		Select("*").
		Preload(queries.NoJoins("Target")).
		Limit(M2M_SOURCES_COUNT * 2)

	var rowLen, rows, err = qs.IterAll()

	if err != nil {
		t.Fatalf("error while querying objects: %v", err)
	}

	if rowLen != M2M_SOURCES_COUNT {
		t.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", M2M_SOURCES_COUNT, rowLen)
	}

	for row, err := range rows {
		if err != nil {
			t.Fatalf("error while querying objects: %v", err)
		}
		relLen := len(row.Object.Target.AsList())
		if relLen != M2M_TARGETS_PER_SOURCE {
			t.Fatalf("query returned incorrect number of related rows, wanted: %d, got: %d", M2M_TARGETS_PER_SOURCE, relLen)
		}
	}
}

func TestFastAttrsInlineQuerySetManyToMany__NoPreload(t *testing.T) {
	var qs = queries.
		GetQuerySet(&FastAttrsInlineBenchmarkM2MSource{}).
		Select("*").
		Limit(M2M_SOURCES_COUNT * 2)

	var rowLen, _, err = qs.IterAll()

	if err != nil {
		t.Fatalf("error while querying objects: %v", err)
	}

	if rowLen != M2M_SOURCES_COUNT {
		t.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", M2M_SOURCES_COUNT, rowLen)
	}
}

func TestFastAttrsInlineQuerySetManyToMany__Select(t *testing.T) {
	var qs = queries.
		GetQuerySet(&FastAttrsInlineBenchmarkM2MSource{}).
		Select("*", "Target.*").
		Limit(TOTAL_M2M_THROUGHS)

	var rowLen, rows, err = qs.IterAll()

	if err != nil {
		t.Fatalf("error while querying objects: %v", err)
	}

	if rowLen != M2M_SOURCES_COUNT {
		t.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", M2M_SOURCES_COUNT, rowLen)
	}

	for row, err := range rows {
		if err != nil {
			t.Fatalf("error while querying objects: %v", err)
		}

		relLen := len(row.Object.Target.AsList())
		if relLen != M2M_TARGETS_PER_SOURCE {
			t.Fatalf("query returned incorrect number of related rows, wanted: %d, got: %d", M2M_TARGETS_PER_SOURCE, relLen)
		}
	}
}

func TestFastAttrsInlineQuerySetManyToMany__Select__Deep(t *testing.T) {
	var qs = queries.
		GetQuerySet(&FastAttrsInlineBenchmarkM2MSource{}).
		Select("*", "Target.*", "Target.TargetReverse.*").
		Limit(TOTAL_M2M_THROUGHS * TOTAL_M2M_THROUGHS)

	var rowLen, rows, err = qs.IterAll()

	if err != nil {
		t.Fatalf("error while querying objects: %v", err)
	}

	if rowLen != M2M_SOURCES_COUNT {
		t.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", M2M_SOURCES_COUNT, rowLen)
	}

	for row, err := range rows {
		if err != nil {
			t.Fatalf("error while querying objects: %v", err)
		}

		lst := row.Object.Target.AsList()
		relLen := len(lst)
		if relLen != M2M_TARGETS_PER_SOURCE {
			t.Fatalf("query returned incorrect number of related rows, wanted: %d, got: %d", M2M_TARGETS_PER_SOURCE, relLen)
		}

		for _, target := range lst {
			relLen = len(target.Object.TargetReverse.AsList())
			if relLen != M2M_TARGETS_PER_SOURCE {
				t.Fatalf("query returned incorrect number of related rows, wanted: %d, got: %d", M2M_TARGETS_PER_SOURCE, relLen)
			}
		}
	}
}

func TestFastAttrsInlineQuerySetManyToMany__Preload__Deep(t *testing.T) {
	var qs = queries.
		GetQuerySet(&FastAttrsInlineBenchmarkM2MSource{}).
		Select("*").
		Preload(queries.NoJoins("Target")).
		Preload(queries.NoJoins("Target.TargetReverse")).
		Limit(TOTAL_M2M_THROUGHS * 2)

	var rowLen, rows, err = qs.IterAll()

	if err != nil {
		t.Fatalf("error while querying objects: %v", err)
	}

	if rowLen != M2M_SOURCES_COUNT {
		t.Fatalf("query returned incorrect number of rows, wanted: %d, got: %d", M2M_SOURCES_COUNT, rowLen)
	}

	for row, err := range rows {
		if err != nil {
			t.Fatalf("error while querying objects: %v", err)
		}

		lst := row.Object.Target.AsList()
		relLen := len(lst)
		if relLen != M2M_TARGETS_PER_SOURCE {
			t.Fatalf("query returned incorrect number of related rows, wanted: %d, got: %d", M2M_TARGETS_PER_SOURCE, relLen)
		}

		for _, target := range lst {
			relLen = len(target.Object.TargetReverse.AsList())
			if relLen != M2M_TARGETS_PER_SOURCE {
				t.Fatalf("query returned incorrect number of related rows, wanted: %d, got: %d", M2M_TARGETS_PER_SOURCE, relLen)
			}
		}
	}
}
