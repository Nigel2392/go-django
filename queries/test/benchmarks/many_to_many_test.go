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
	attrs.RegisterModel(&BenchmarkM2MSource{})
	attrs.RegisterModel(&BenchmarkM2MTarget{})
	attrs.RegisterModel(&BenchmarkM2MThrough{})
}

// -------------------------------------------------------------------------
// MANY-TO-MANY MODELS
// -------------------------------------------------------------------------

type BenchmarkM2MSource struct {
	models.Model
	ID     uint64
	Title  string
	Target *queries.RelM2M[*BenchmarkM2MTarget, *BenchmarkM2MThrough]
}

func (m *BenchmarkM2MSource) Fields() []attrs.Field {
	return []attrs.Field{
		attrs.NewField(m, "ID", fldCnfPrimary),
		attrs.NewField(m, "Title"),
		fields.NewManyToManyField[*queries.RelM2M[*BenchmarkM2MTarget, *BenchmarkM2MThrough]](m, "Target", &fields.FieldConfig{
			ScanTo:      &m.Target,
			ReverseName: "TargetReverse",
			Rel:         m2mRel,
		}),
	}
}

func (m *BenchmarkM2MSource) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m, m.Fields).WithTableName("m2m_source_bench")
}

type BenchmarkM2MTarget struct {
	models.Model
	ID            uint64
	Name          string
	TargetReverse *queries.RelM2M[attrs.Definer, attrs.Definer]
}

var (
	fldCnfSrcId = &attrs.FieldConfig{
		Column:        "source_id",
		RelForeignKey: attrs.Relate(&BenchmarkM2MSource{}, "", nil),
	}
	fldCnfTargetId = &attrs.FieldConfig{
		Column:        "target_id",
		RelForeignKey: attrs.Relate(&BenchmarkM2MTarget{}, "", nil),
	}

	m2mRel = attrs.Relate(
		&BenchmarkM2MTarget{},
		"", &attrs.ThroughModel{
			This:   &BenchmarkM2MThrough{},
			Source: "SourceModel",
			Target: "TargetModel",
		},
	)
)

func (m *BenchmarkM2MTarget) Fields() []attrs.Field {
	return []attrs.Field{
		attrs.NewField(m, "ID", fldCnfPrimary),
		attrs.NewField(m, "Name"),
	}
}

func (m *BenchmarkM2MTarget) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m, m.Fields).WithTableName("m2m_target_bench")
}

type BenchmarkM2MThrough struct {
	ID          uint64
	SourceModel uint64
	TargetModel uint64
}

func (m *BenchmarkM2MThrough) Fields() []attrs.Field {
	return []attrs.Field{
		attrs.NewField(m, "ID", fldCnfPrimary),
		attrs.NewField(m, "SourceModel", fldCnfSrcId),
		attrs.NewField(m, "TargetModel", fldCnfTargetId),
	}
}

func (m *BenchmarkM2MThrough) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m, m.Fields).WithTableName("m2m_through_bench")
}

func TestManyToMany__Preload(t *testing.T) {

	var qs = queries.
		GetQuerySet(&BenchmarkM2MSource{}).
		// WithContext(drivers.SetLogSQLContext(t.Context(), false)).
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

// -------------------------------------------------------------------------
// MANY-TO-MANY BENCHMARKS
// -------------------------------------------------------------------------

func BenchmarkQuerySetManyToMany__NoPreload(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&BenchmarkM2MSource{}).
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

func BenchmarkQuerySetManyToMany__Select(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&BenchmarkM2MSource{}).
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

		var chk *BenchmarkM2MSource
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

func BenchmarkQuerySetManyToMany__Select__Deep(b *testing.B) {
	b.StopTimer()

	// Forces the ORM to JOIN through the target and back out to the reverse relation
	var qs = queries.
		GetQuerySet(&BenchmarkM2MSource{}).
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

func BenchmarkQuerySetManyToMany__Preload(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&BenchmarkM2MSource{}).
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

		var chk *BenchmarkM2MSource
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

func BenchmarkQuerySetManyToMany__Preload__Deep(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&BenchmarkM2MSource{}).
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

		var chk *BenchmarkM2MSource
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

		var chkDst *BenchmarkM2MTarget
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

func TestQuerySetManyToMany__NoPreload(t *testing.T) {
	var qs = queries.
		GetQuerySet(&BenchmarkM2MSource{}).
		WithContext(drivers.SetLogSQLContext(t.Context(), false)).
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

func TestQuerySetManyToMany__Select(t *testing.T) {
	var qs = queries.
		GetQuerySet(&BenchmarkM2MSource{}).
		WithContext(drivers.SetLogSQLContext(t.Context(), false)).
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

func TestQuerySetManyToMany__Select__Deep(t *testing.T) {
	var qs = queries.
		GetQuerySet(&BenchmarkM2MSource{}).
		WithContext(drivers.SetLogSQLContext(t.Context(), false)).
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

func TestQuerySetManyToMany__Preload__Deep(t *testing.T) {
	var qs = queries.
		GetQuerySet(&BenchmarkM2MSource{}).
		// WithContext(drivers.SetLogSQLContext(t.Context(), false)).
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
