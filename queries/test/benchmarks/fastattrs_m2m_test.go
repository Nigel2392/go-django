package benchmarks_test

import (
	"context"
	"fmt"
	"testing"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/fields"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"

	// Assuming fastattrs is importable here based on your structure
	"github.com/Nigel2392/go-django/src/core/attrs/fastattrs"
)

var (
	fast_fldCnfPrimary = &attrs.FieldConfig{Primary: true}

	fast_fldCnfSrcId = &attrs.FieldConfig{
		Column:        "source_id",
		RelForeignKey: attrs.Relate(&FastAttrsBenchmarkM2MSource{}, "", nil),
	}
	fast_fldCnfTargetId = &attrs.FieldConfig{
		Column:        "target_id",
		RelForeignKey: attrs.Relate(&FastAttrsBenchmarkM2MTarget{}, "", nil),
	}

	fast_m2mRel = attrs.Relate(
		&FastAttrsBenchmarkM2MTarget{},
		"", &attrs.ThroughModel{
			This:   &FastAttrsBenchmarkM2MThrough{},
			Source: "SourceModel",
			Target: "TargetModel",
		},
	)
)

func init() {
	// Register Source Model with fastattrs
	fastattrs.RegisterModel(func(addField func(string, fastattrs.FieldConfig[*FastAttrsBenchmarkM2MSource])) {
		addField("ID", fastattrs.FieldConfig[*FastAttrsBenchmarkM2MSource]{
			Config:   *fast_fldCnfPrimary,
			GetValue: func(obj *FastAttrsBenchmarkM2MSource) interface{} { return obj.ID },
			SetValue: func(obj *FastAttrsBenchmarkM2MSource, value any) error {
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
		})
		addField("Title", fastattrs.FieldConfig[*FastAttrsBenchmarkM2MSource]{
			GetValue: func(obj *FastAttrsBenchmarkM2MSource) interface{} { return obj.Title },
			SetValue: func(obj *FastAttrsBenchmarkM2MSource, value any) error {
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
		})
	})

	// Register Target Model with fastattrs
	fastattrs.RegisterModel(func(addField func(string, fastattrs.FieldConfig[*FastAttrsBenchmarkM2MTarget])) {
		addField("ID", fastattrs.FieldConfig[*FastAttrsBenchmarkM2MTarget]{
			Config:   *fast_fldCnfPrimary,
			GetValue: func(obj *FastAttrsBenchmarkM2MTarget) interface{} { return obj.ID },
			SetValue: func(obj *FastAttrsBenchmarkM2MTarget, value any) error {
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
		})
		addField("Name", fastattrs.FieldConfig[*FastAttrsBenchmarkM2MTarget]{
			GetValue: func(obj *FastAttrsBenchmarkM2MTarget) interface{} { return obj.Name },
			SetValue: func(obj *FastAttrsBenchmarkM2MTarget, value any) error {
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
		})
	})

	// Register Through Model with fastattrs
	fastattrs.RegisterModel(func(addField func(string, fastattrs.FieldConfig[*FastAttrsBenchmarkM2MThrough])) {
		addField("ID", fastattrs.FieldConfig[*FastAttrsBenchmarkM2MThrough]{
			Config:   *fast_fldCnfPrimary,
			GetValue: func(obj *FastAttrsBenchmarkM2MThrough) interface{} { return obj.ID },
			SetValue: func(obj *FastAttrsBenchmarkM2MThrough, value any) error {
				switch v := value.(type) {
				case uint64:
					obj.ID = v
				case int64:
					obj.ID = uint64(v)
				case int:
					obj.ID = uint64(v)
				}
				return nil
			},
			Default: uint64(0),
		})
		addField("SourceModel", fastattrs.FieldConfig[*FastAttrsBenchmarkM2MThrough]{
			Config:   *fast_fldCnfSrcId,
			GetValue: func(obj *FastAttrsBenchmarkM2MThrough) interface{} { return obj.SourceModel },
			SetValue: func(obj *FastAttrsBenchmarkM2MThrough, value any) error {
				switch v := value.(type) {
				case uint64:
					obj.SourceModel = v
				case int64:
					obj.SourceModel = uint64(v)
				case int:
					obj.SourceModel = uint64(v)
				}
				return nil
			},
			Default: uint64(0),
		})
		addField("TargetModel", fastattrs.FieldConfig[*FastAttrsBenchmarkM2MThrough]{
			Config:   *fast_fldCnfTargetId,
			GetValue: func(obj *FastAttrsBenchmarkM2MThrough) interface{} { return obj.TargetModel },
			SetValue: func(obj *FastAttrsBenchmarkM2MThrough, value any) error {
				switch v := value.(type) {
				case uint64:
					obj.TargetModel = v
				case int64:
					obj.TargetModel = uint64(v)
				case int:
					obj.TargetModel = uint64(v)
				}
				return nil
			},
			Default: uint64(0),
		})
	})
}

// -------------------------------------------------------------------------
// MANY-TO-MANY MODELS
// -------------------------------------------------------------------------

type FastAttrsBenchmarkM2MSource struct {
	models.Model
	ID     uint64
	Title  string
	Target *queries.RelM2M[*FastAttrsBenchmarkM2MTarget, *FastAttrsBenchmarkM2MThrough]
}

func (m *FastAttrsBenchmarkM2MSource) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m, fastattrs.NewField(m, "ID"),
		fastattrs.NewField(m, "Title"),
		fields.NewManyToManyField[*queries.RelM2M[*FastAttrsBenchmarkM2MTarget, *FastAttrsBenchmarkM2MThrough]](m, "Target", &fields.FieldConfig{
			ScanTo:      &m.Target,
			ReverseName: "TargetReverse",
			Rel:         fast_m2mRel,
		})).WithTableName("fast_m2m_source_bench")
}

type FastAttrsBenchmarkM2MTarget struct {
	models.Model
	ID            uint64
	Name          string
	TargetReverse *queries.RelM2M[attrs.Definer, attrs.Definer]
}

func (m *FastAttrsBenchmarkM2MTarget) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m, fastattrs.NewField(m, "ID"),
		fastattrs.NewField(m, "Name")).WithTableName("fast_m2m_target_bench")
}

type FastAttrsBenchmarkM2MThrough struct {
	ID          uint64
	SourceModel uint64
	TargetModel uint64
}

func (m *FastAttrsBenchmarkM2MThrough) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m, fastattrs.NewField(m, "ID"),
		fastattrs.NewField(m, "SourceModel"),
		fastattrs.NewField(m, "TargetModel"),
	).WithTableName("fast_m2m_through_bench")
}

// -------------------------------------------------------------------------
// MANY-TO-MANY BENCHMARKS
// -------------------------------------------------------------------------

func BenchmarkFastAttrsQuerySetManyToMany__NoPreload(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkM2MSource{}).
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

func BenchmarkFastAttrsQuerySetManyToMany__Select(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkM2MSource{}).
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

		var chk *FastAttrsBenchmarkM2MSource
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

func BenchmarkFastAttrsQuerySetManyToMany__Select__Deep(b *testing.B) {
	b.StopTimer()

	// Forces the ORM to JOIN through the target and back out to the reverse relation
	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkM2MSource{}).
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

func BenchmarkFastAttrsQuerySetManyToMany__Preload(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkM2MSource{}).
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

		var chk *FastAttrsBenchmarkM2MSource
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

func BenchmarkFastAttrsQuerySetManyToMany__Preload__Deep(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkM2MSource{}).
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

		var chk *FastAttrsBenchmarkM2MSource
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

		var chkDst *FastAttrsBenchmarkM2MTarget
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

func TestFastAttrsManyToMany__Preload(t *testing.T) {

	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkM2MSource{}).
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

func TestFastAttrsQuerySetManyToMany__NoPreload(t *testing.T) {
	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkM2MSource{}).
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

func TestFastAttrsQuerySetManyToMany__Select(t *testing.T) {
	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkM2MSource{}).
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

func TestFastAttrsQuerySetManyToMany__Select__Deep(t *testing.T) {
	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkM2MSource{}).
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

func TestFastAttrsQuerySetManyToMany__Preload__Deep(t *testing.T) {
	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkM2MSource{}).
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
