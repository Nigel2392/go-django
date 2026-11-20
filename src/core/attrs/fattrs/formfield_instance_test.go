package fattrs_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/goldcrest"
)

type ffTestTarget struct {
	ID    uint64
	Title string
}

func (m *ffTestTarget) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*ffTestTarget, uint64] {
			return fattrs.PtrFieldConfig[*ffTestTarget, uint64]{
				Config: attrs.Config{Primary: true, ReadOnly: true},
			}
		}),
		fattrs.Field(m, "Title", &m.Title, func() fattrs.PtrFieldConfig[*ffTestTarget, string] {
			return fattrs.PtrFieldConfig[*ffTestTarget, string]{}
		}),
	)
}

type ffTestPtrThrough struct {
	Source *ffTestPtrSource
	Target *ffTestTarget
}

func (m *ffTestPtrThrough) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "Source", &m.Source, func() fattrs.PtrFieldConfig[*ffTestPtrThrough, *ffTestPtrSource] {
			return fattrs.PtrFieldConfig[*ffTestPtrThrough, *ffTestPtrSource]{
				Config: attrs.Config{Column: "source_id"},
			}
		}),
		fattrs.Field(m, "Target", &m.Target, func() fattrs.PtrFieldConfig[*ffTestPtrThrough, *ffTestTarget] {
			return fattrs.PtrFieldConfig[*ffTestPtrThrough, *ffTestTarget]{
				Config: attrs.Config{Column: "target_id"},
			}
		}),
	)
}

type ffTestPtrSource struct {
	ID       uint64
	Title    string
	Products any
}

func (m *ffTestPtrSource) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*ffTestPtrSource, uint64] {
			return fattrs.PtrFieldConfig[*ffTestPtrSource, uint64]{
				Config: attrs.Config{Primary: true, ReadOnly: true},
			}
		}),
		fattrs.Field(m, "Title", &m.Title, func() fattrs.PtrFieldConfig[*ffTestPtrSource, string] {
			return fattrs.PtrFieldConfig[*ffTestPtrSource, string]{}
		}),
		fattrs.Field(m, "Products", &m.Products, func() fattrs.PtrFieldConfig[*ffTestPtrSource, any] {
			return fattrs.PtrFieldConfig[*ffTestPtrSource, any]{
				Config: attrs.Config{
					RelManyToMany: attrs.Relate(
						&ffTestTarget{},
						"",
						&attrs.ThroughModel{
							This:   &ffTestPtrThrough{},
							Source: "Source",
							Target: "Target",
						},
					),
				},
			}
		}),
	)
}

type ffTestReflectlessThrough struct {
	Source *ffTestReflectlessSource
	Target *ffTestTarget
}

func (m *ffTestReflectlessThrough) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "Source", &m.Source, func() fattrs.PtrFieldConfig[*ffTestReflectlessThrough, *ffTestReflectlessSource] {
			return fattrs.PtrFieldConfig[*ffTestReflectlessThrough, *ffTestReflectlessSource]{
				Config: attrs.Config{Column: "source_id"},
			}
		}),
		fattrs.Field(m, "Target", &m.Target, func() fattrs.PtrFieldConfig[*ffTestReflectlessThrough, *ffTestTarget] {
			return fattrs.PtrFieldConfig[*ffTestReflectlessThrough, *ffTestTarget]{
				Config: attrs.Config{Column: "target_id"},
			}
		}),
	)
}

type ffTestReflectlessSource struct {
	ID       uint64
	Title    string
	Products any
}

func (m *ffTestReflectlessSource) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*ffTestReflectlessSource, uint64] {
			return fattrs.PtrFieldConfig[*ffTestReflectlessSource, uint64]{
				Primary:  true,
				ReadOnly: true,
			}
		}),
		fattrs.Field(m, "Title", &m.Title, func() fattrs.PtrFieldConfig[*ffTestReflectlessSource, string] {
			return fattrs.PtrFieldConfig[*ffTestReflectlessSource, string]{}
		}),
		fattrs.NewField(m, "Products", func() fattrs.FieldConfig[*ffTestReflectlessSource] {
			return fattrs.FieldConfig[*ffTestReflectlessSource]{
				RelManyToMany: attrs.Relate(
					&ffTestTarget{},
					"",
					&attrs.ThroughModel{
						This:   &ffTestReflectlessThrough{},
						Source: "Source",
						Target: "Target",
					},
				),
				GetValue: func(obj *ffTestReflectlessSource) interface{} { return obj.Products },
				SetValue: func(obj *ffTestReflectlessSource, value any) error {
					obj.Products = value
					return nil
				},
				Default: fattrs.NULL{},
			}
		}),
	)
}

func init() {
	attrs.RegisterModel(&ffTestTarget{})
	attrs.RegisterModel(&ffTestPtrThrough{})
	attrs.RegisterModel(&ffTestPtrSource{})
	attrs.RegisterModel(&ffTestReflectlessThrough{})
	attrs.RegisterModel(&ffTestReflectlessSource{})
	_ = attrs.ResetDefinitions.Send(context.Background(), nil)
}

func registerSpyFormFieldHook(t *testing.T) *[]attrs.Field {
	t.Helper()
	prevHooks := goldcrest.Get[attrs.FormFieldGetter](attrs.HookFormFieldForType)
	var captured []attrs.Field

	goldcrest.Register(
		attrs.HookFormFieldForType,
		0,
		attrs.FormFieldGetter(func(f attrs.Field, newFieldType reflect.Type, fieldVal reflect.Value, opts ...func(fields.Field)) (fields.Field, bool) {
			rel := f.Rel()
			if rel != nil && rel.Type() == attrs.RelManyToMany {
				captured = append(captured, f)
				return fields.NewField(opts...), true
			}
			return nil, false
		}),
	)

	t.Cleanup(func() {
		goldcrest.Unregister(attrs.HookFormFieldForType)
		for _, h := range prevHooks {
			goldcrest.Register(attrs.HookFormFieldForType, 50, h)
		}
	})

	return &captured
}

func TestPtrField_FormField_PassesBoundInstanceAndField(t *testing.T) {
	captured := registerSpyFormFieldHook(t)

	t.Run("BoundInstance", func(t *testing.T) {
		*captured = nil
		src := &ffTestPtrSource{ID: 42, Title: "Coupon 42"}
		prodField, ok := attrs.Define(t.Context(), src).Field("Products")
		if !ok {
			t.Fatalf("expected Products field on ffTestPtrSource")
		}

		ff := prodField.FormField()
		if ff == nil {
			t.Fatalf("expected non-nil FormField")
		}
		if len(*captured) == 0 {
			t.Fatalf("expected FormFieldGetter hook to be called")
		}

		gotField := (*captured)[len(*captured)-1]
		if gotField.Instance() != src {
			t.Errorf(
				"hook received field bound to instance %+v (ptr=%p), want %+v (ptr=%p)",
				gotField.Instance(), gotField.Instance(), src, src,
			)
		}
		if pk := attrs.PrimaryKey(t.Context(), gotField.Instance()); pk != uint64(42) {
			t.Errorf("gotField.Instance() PK = %v, want %d", pk, uint64(42))
		}
	})

	t.Run("FieldColumnName", func(t *testing.T) {
		src := &ffTestPtrSource{ID: 43, Title: "Coupon 43"}
		prodField, ok := attrs.Define(t.Context(), src).Field("Products")
		if !ok {
			t.Fatalf("expected Products field on ffTestPtrSource")
		}

		ff := prodField.FormField()
		if ff == nil {
			t.Fatalf("expected non-nil FormField")
		}
		if len(*captured) == 0 {
			t.Fatalf("expected FormFieldGetter hook to have recorded a field")
		}

		gotField := (*captured)[len(*captured)-1]
		if gotField.ColumnName() != prodField.ColumnName() {
			t.Errorf("hook received field ColumnName() = %q, want %q", gotField.ColumnName(), prodField.ColumnName())
		}
	})

	t.Run("DistinctPerInstance", func(t *testing.T) {
		*captured = nil
		s1 := &ffTestPtrSource{ID: 1, Title: "Coupon 1"}
		s2 := &ffTestPtrSource{ID: 2, Title: "Coupon 2"}

		f1, ok := attrs.Define(t.Context(), s1).Field("Products")
		if !ok {
			t.Fatalf("expected Products field on s1")
		}
		f2, ok := attrs.Define(t.Context(), s2).Field("Products")
		if !ok {
			t.Fatalf("expected Products field on s2")
		}

		ff1 := f1.FormField()
		ff2 := f2.FormField()

		if ff1 == ff2 {
			t.Errorf("expected distinct FormField instances for s1 and s2, got same pointer %p", ff1)
		}
		if len(*captured) != 2 {
			t.Fatalf("expected hook to be called twice (once per instance), got %d calls", len(*captured))
		}
		if (*captured)[0].Instance() != s1 {
			t.Errorf("first hook call Instance() = %+v (ptr=%p), want s1 %+v (ptr=%p)", (*captured)[0].Instance(), (*captured)[0].Instance(), s1, s1)
		}
		if (*captured)[1].Instance() != s2 {
			t.Errorf("second hook call Instance() = %+v (ptr=%p), want s2 %+v (ptr=%p)", (*captured)[1].Instance(), (*captured)[1].Instance(), s2, s2)
		}
	})
}

func TestReflectlessField_FormField_PassesBoundInstance(t *testing.T) {
	captured := registerSpyFormFieldHook(t)

	t.Run("BoundInstance", func(t *testing.T) {
		*captured = nil
		src := &ffTestReflectlessSource{ID: 77, Title: "Ref 77"}
		prodField, ok := attrs.Define(t.Context(), src).Field("Products")
		if !ok {
			t.Fatalf("expected Products field on ffTestReflectlessSource")
		}

		ff := prodField.FormField()
		if ff == nil {
			t.Fatalf("expected non-nil FormField")
		}
		if len(*captured) == 0 {
			t.Fatalf("expected FormFieldGetter hook to be called")
		}

		gotField := (*captured)[len(*captured)-1]
		if gotField.Instance() != src {
			t.Errorf(
				"hook received field bound to instance %+v (ptr=%p), want %+v (ptr=%p)",
				gotField.Instance(), gotField.Instance(), src, src,
			)
		}
		if pk := attrs.PrimaryKey(t.Context(), gotField.Instance()); pk != uint64(77) {
			t.Errorf("gotField.Instance() PK = %v, want %d", pk, uint64(77))
		}
	})

	t.Run("DistinctPerInstance", func(t *testing.T) {
		*captured = nil
		s1 := &ffTestReflectlessSource{ID: 1, Title: "Ref 1"}
		s2 := &ffTestReflectlessSource{ID: 2, Title: "Ref 2"}

		f1, ok := attrs.Define(t.Context(), s1).Field("Products")
		if !ok {
			t.Fatalf("expected Products field on s1")
		}
		f2, ok := attrs.Define(t.Context(), s2).Field("Products")
		if !ok {
			t.Fatalf("expected Products field on s2")
		}

		ff1 := f1.FormField()
		ff2 := f2.FormField()

		if ff1 == ff2 {
			t.Errorf("expected distinct FormField instances for s1 and s2, got same pointer %p", ff1)
		}
		if len(*captured) != 2 {
			t.Fatalf("expected hook to be called twice (once per instance), got %d calls", len(*captured))
		}
		if (*captured)[0].Instance() != s1 {
			t.Errorf("first hook call Instance() = %+v (ptr=%p), want s1 %+v (ptr=%p)", (*captured)[0].Instance(), (*captured)[0].Instance(), s1, s1)
		}
		if (*captured)[1].Instance() != s2 {
			t.Errorf("second hook call Instance() = %+v (ptr=%p), want s2 %+v (ptr=%p)", (*captured)[1].Instance(), (*captured)[1].Instance(), s2, s2)
		}
	})
}
