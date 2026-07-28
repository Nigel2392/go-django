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
	"github.com/Nigel2392/go-django/src/core/attrs/fastattrs"
)

func init() {
	fastattrs.RegisterModel(func(field func(string, fastattrs.FieldConfig[*FastAttrsBenchmarkO2ONoThroughMain])) {
		field("ID", fastattrs.FieldConfig[*FastAttrsBenchmarkO2ONoThroughMain]{
			Config:   *fast_fldCnfPrimary,
			GetValue: func(obj *FastAttrsBenchmarkO2ONoThroughMain) interface{} { return obj.ID },
			SetValue: func(obj *FastAttrsBenchmarkO2ONoThroughMain, value any) error {
				switch v := value.(type) {
				case uint64:
					obj.ID = int32(v)
				case int64:
					obj.ID = int32(v)
				case int:
					obj.ID = int32(v)
				default:
					return fmt.Errorf("invalid ID type %T: %v", value, value)
				}
				return nil
			},
			Default: uint64(0),
		})
		field("Title", fastattrs.FieldConfig[*FastAttrsBenchmarkO2ONoThroughMain]{
			Config:   attrs.FieldConfig{},
			GetValue: func(obj *FastAttrsBenchmarkO2ONoThroughMain) interface{} { return obj.Title },
			SetValue: func(obj *FastAttrsBenchmarkO2ONoThroughMain, value any) error {
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
		field("Target", fastattrs.FieldConfig[*FastAttrsBenchmarkO2ONoThroughMain]{
			Config: attrs.FieldConfig{
				Column:      "target_id",
				RelOneToOne: attrs.Relate(&FastAttrsBenchmarkO2ONoThroughTarget{}, "", nil),
				Attributes: map[string]interface{}{
					attrs.AttrReverseAliasKey: "MainReverse",
				},
			},
			GetValue: func(obj *FastAttrsBenchmarkO2ONoThroughMain) interface{} {
				return obj.Target
			},
			SetValue: func(obj *FastAttrsBenchmarkO2ONoThroughMain, value any) error {
				switch v := value.(type) {
				case int64:
					obj.Target = &FastAttrsBenchmarkO2ONoThroughTarget{
						ID: uint64(v),
					}
				case *FastAttrsBenchmarkO2ONoThroughTarget:
					obj.Target = v
				default:
					return errors.TypeMismatch.Wrapf(
						"%T does not match expected target type",
						value,
					)
				}
				return nil
			},
			Default: fastattrs.NULL{},
		})
	})
	fastattrs.RegisterModel(func(field func(string, fastattrs.FieldConfig[*FastAttrsBenchmarkO2ONoThroughTarget])) {
		field("ID", fastattrs.FieldConfig[*FastAttrsBenchmarkO2ONoThroughTarget]{
			Config:   *fast_fldCnfPrimary,
			GetValue: func(obj *FastAttrsBenchmarkO2ONoThroughTarget) interface{} { return obj.ID },
			SetValue: func(obj *FastAttrsBenchmarkO2ONoThroughTarget, value any) error {
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
		field("Name", fastattrs.FieldConfig[*FastAttrsBenchmarkO2ONoThroughTarget]{
			Config:   attrs.FieldConfig{},
			GetValue: func(obj *FastAttrsBenchmarkO2ONoThroughTarget) interface{} { return obj.Name },
			SetValue: func(obj *FastAttrsBenchmarkO2ONoThroughTarget, value any) error {
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
}

type FastAttrsBenchmarkO2ONoThroughTarget struct {
	models.Model
	ID          uint64
	Name        string
	MainReverse *FastAttrsBenchmarkO2ONoThroughMain
}

func (t *FastAttrsBenchmarkO2ONoThroughTarget) FieldDefs(ctx context.Context) attrs.Definitions {
	return t.Model.Define(ctx, t,
		fastattrs.NewField(t, "ID"),
		fastattrs.NewField(t, "Name"),
	).WithTableName("o2o_nt_target_bench")
}

type FastAttrsBenchmarkO2ONoThroughMain struct {
	models.Model
	ID     int32
	Title  string
	Target *FastAttrsBenchmarkO2ONoThroughTarget
}

func (t *FastAttrsBenchmarkO2ONoThroughMain) FieldDefs(ctx context.Context) attrs.Definitions {
	return t.Model.Define(ctx, t,
		fastattrs.NewField(t, "ID"),
		fastattrs.NewField(t, "Title"),
		fastattrs.NewField(t, "Target"),
	).WithTableName("o2o_nt_main_bench")
}

// BENCHMARKS
func BenchmarkFastAttrsQuerySetO2ONoThrough__Select(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkO2ONoThroughMain{}).
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

func BenchmarkFastAttrsQuerySetO2ONoThrough__Preload(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkO2ONoThroughMain{}).
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

func BenchmarkFastAttrsQuerySetO2ONoThrough__Select__Reverse(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkO2ONoThroughTarget{}).
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

func BenchmarkFastAttrsQuerySetO2ONoThrough__Preload__Reverse(b *testing.B) {
	b.StopTimer()

	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkO2ONoThroughTarget{}).
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

var fast_o2oTestContext = drivers.SetLogSQLContext(context.Background(), false)

func TestFastAttrsQuerySetO2ONoThrough__Select(t *testing.T) {
	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkO2ONoThroughMain{}).
		WithContext(fast_o2oTestContext).
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

func TestFastAttrsQuerySetO2ONoThrough__Preload(t *testing.T) {
	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkO2ONoThroughMain{}).
		WithContext(fast_o2oTestContext).
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

func TestFastAttrsQuerySetO2ONoThrough__Select__Reverse(t *testing.T) {
	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkO2ONoThroughTarget{}).
		WithContext(fast_o2oTestContext).
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

func TestFastAttrsQuerySetO2ONoThrough__Preload__Reverse(t *testing.T) {
	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkO2ONoThroughTarget{}).
		WithContext(fast_o2oTestContext).
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

func TestFastAttrsQuerySetO2ONoThrough__Preload__Reverse__Deep(t *testing.T) {
	var qs = queries.
		GetQuerySet(&FastAttrsBenchmarkO2ONoThroughTarget{}).
		WithContext(fast_o2oTestContext).
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
