package attrs_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/versatile/clone"
)

type structValueCaller[T any] struct {
	Ref T
}

func (t structValueCaller[T]) Value() reflect.Value {
	return reflect.ValueOf(t.Ref)
}

type testStructValue[T any, VAL any] struct {
	backref T
	val     VAL
}

func (tc testStructValue[T, VAL]) CCopy(ctx context.Context, s *clone.State, dst *testStructValue[T, VAL], caller clone.Caller) error {
	defer fmt.Println("tc.CCopy", dst)
	dst.backref = caller.(clone.StructCaller).Val.Addr().Interface().(T)
	return s.Copy(ctx, &dst.val, tc.val, structValueCaller[*testStructValue[T, VAL]]{Ref: dst})
}

type testStructValueRef[T any, VAL any] struct {
	backref T
	val     VAL
}

func (tc *testStructValueRef[T, VAL]) CCopy(ctx context.Context, s *clone.State, dst *testStructValueRef[T, VAL], caller clone.Caller) error {
	defer fmt.Println("tc.CCopy", dst)
	dst.backref = caller.(clone.StructCaller).Val.Addr().Interface().(T)
	return s.Copy(ctx, &dst.val, tc.val, structValueCaller[*testStructValueRef[T, VAL]]{Ref: dst})
}

type TestStructValueModelFields struct {
	TestModelFields
	Value testStructValue[*TestStructValueModelFields, int]
}

type TestStructValueModelFieldsRef struct {
	TestModelFields
	Value *testStructValueRef[*TestStructValueModelFieldsRef, int]
}

func TestClone(t *testing.T) {
	t.Run("Typed", func(t *testing.T) {
		var (
			dst *TestModelFields
			src = TestModelFields{
				ID:   1,
				Name: "TestModelFields",
				Objects: []int64{
					1, 2, 3, 4, 5, 6,
				},
			}
		)

		err := clone.Copy(t.Context(), &dst, &src)
		if err != nil {
			t.Fatalf("error during interface test: %v", err)
		}

		if dst == nil {
			t.Fatal("dst is nil")
		}

		if dst.ID != src.ID || src.ID != 1 {
			t.Fatalf("dst ID mismatch src ID: %d != %d", dst.ID, src.ID)
		}

		if dst.Name != src.Name || src.Name != "TestModelFields" {
			t.Fatalf("dst Name mismatch src Name: %s != %s", dst.Name, src.Name)
		}

		t.Logf("%+v", dst)
	})

	t.Run("Any", func(t *testing.T) {
		var (
			_dst any
			src  = TestModelFields{
				ID:   1,
				Name: "TestModelFields",
				Objects: []int64{
					1, 2, 3, 4, 5, 6,
				},
			}
		)

		err := clone.Copy(t.Context(), &_dst, &src)
		if err != nil {
			t.Fatalf("error during interface test: %v", err)
		}

		dst, _ := _dst.(*TestModelFields)
		if dst == nil {
			t.Fatal("dst is nil")
		}

		if dst.ID != src.ID || src.ID != 1 {
			t.Fatalf("dst ID mismatch src ID: %d != %d", dst.ID, src.ID)
		}

		if dst.Name != src.Name || src.Name != "TestModelFields" {
			t.Fatalf("dst Name mismatch src Name: %s != %s", dst.Name, src.Name)
		}

		t.Logf("%+v", dst)
	})

	t.Run("Definer", func(t *testing.T) {
		var (
			_dst attrs.Definer
			src  = TestModelFields{
				ID:   1,
				Name: "TestModelFields",
				Objects: []int64{
					1, 2, 3, 4, 5, 6,
				},
			}
		)

		err := clone.Copy(t.Context(), &_dst, &src)
		if err != nil {
			t.Fatalf("error during interface test: %v", err)
		}

		dst, _ := _dst.(*TestModelFields)
		if dst == nil {
			t.Fatal("dst is nil")
		}

		if dst.ID != src.ID || src.ID != 1 {
			t.Fatalf("dst ID mismatch src ID: %d != %d", dst.ID, src.ID)
		}

		if dst.Name != src.Name || src.Name != "TestModelFields" {
			t.Fatalf("dst Name mismatch src Name: %s != %s", dst.Name, src.Name)
		}

		t.Logf("%+v", dst)
	})

	t.Run("SliceRefs_REGULAR", func(t *testing.T) {
		var (
			src1 = &TestModelFields{
				ID:   1,
				Name: "TestSliceRefs1",
				Objects: []int64{
					1, 2, 3,
				},
			}
			src2 = &TestModelFields{
				ID:   2,
				Name: "TestSliceRefs2",
				Objects: []int64{
					4, 5, 6,
				},
			}
			src3 = &TestModelFields{
				ID:   3,
				Name: "TestSliceRefs3",
				Objects: []int64{
					7, 8, 9,
				},
			}
			dstSlice [][]*TestModelFields
			srcSlice = [][]*TestModelFields{
				{src1, src2, src3},
				{src3, src2, src1},
				{src1, src3, src2},
			}
		)

		err := clone.Copy(t.Context(), &dstSlice, srcSlice)
		if err != nil {
			t.Fatalf("error during interface test: %v", err)
		}

		if len(dstSlice) != 3 {
			t.Fatalf("dst slice length: %d", len(dstSlice))
		}

		if len(dstSlice[0]) != 3 ||
			len(dstSlice[1]) != 3 ||
			len(dstSlice[2]) != 3 {
			t.Fatalf("dst slice length: %d/%d/%d", len(dstSlice[0]), len(dstSlice[1]), len(dstSlice[2]))
		}

		t.Logf("dstSlice:\n\tSlice 1: %v\n\tSlice 2: %v\n\tSlice 3: %v", dstSlice[0], dstSlice[1], dstSlice[2])
		t.Logf("srcSlice:\n\tSlice 1: %v\n\tSlice 2: %v\n\tSlice 3: %v", srcSlice[0], srcSlice[1], srcSlice[2])

		if !pmatches(dstSlice[0][0], dstSlice[1][2], dstSlice[2][0]) ||
			!pmatches(dstSlice[0][1], dstSlice[1][1], dstSlice[2][2]) ||
			!pmatches(dstSlice[0][2], dstSlice[1][0], dstSlice[2][1]) {
			t.Fatalf("pointer mismatch in dst slice:\n\tSlice 1: %v\n\tSlice 2: %v\n\tSlice 3: %v", dstSlice[0], dstSlice[1], dstSlice[2])
		}

		if pmatches(dstSlice[0][0], srcSlice[1][2]) ||
			pmatches(dstSlice[0][1], srcSlice[1][1]) ||
			pmatches(dstSlice[0][2], srcSlice[1][0]) {
			t.Fatalf("pointer mismatch in dst slice, reference to src slice:\n\tSlice 1: %v\n\tSlice 2: %v\n\tSlice 3: %v", dstSlice[0], dstSlice[1], dstSlice[2])
		}

	})

	t.Run("SliceRefs_VAL_TYP", func(t *testing.T) {
		var (
			src1 = &TestStructValueModelFields{
				ID:   1,
				Name: "TestSliceRefs1",
				Objects: []int64{
					1, 2, 3,
				},
			}
			src2 = &TestStructValueModelFields{
				ID:   2,
				Name: "TestSliceRefs2",
				Objects: []int64{
					4, 5, 6,
				},
			}
			src3 = &TestStructValueModelFields{
				ID:   3,
				Name: "TestSliceRefs3",
				Objects: []int64{
					7, 8, 9,
				},
			}
		)

		src1.Value.backref = src1
		src1.Value.val = 1

		src2.Value.backref = src2
		src2.Value.val = 2

		src3.Value.backref = src3
		src3.Value.val = 3

		var (
			dstSlice [][]*TestStructValueModelFields
			srcSlice = [][]*TestStructValueModelFields{
				{src1, src2, src3},
				{src3, src2, src1},
				{src1, src3, src2},
			}
		)

		err := clone.Copy(t.Context(), &dstSlice, srcSlice)
		if err != nil {
			t.Fatalf("error during interface test: %v", err)
		}

		if len(dstSlice) != 3 {
			t.Fatalf("dst slice length: %d", len(dstSlice))
		}

		if len(dstSlice[0]) != 3 ||
			len(dstSlice[1]) != 3 ||
			len(dstSlice[2]) != 3 {
			t.Fatalf("dst slice length: %d/%d/%d", len(dstSlice[0]), len(dstSlice[1]), len(dstSlice[2]))
		}

		t.Logf("dstSlice:\n\tSlice 1: %v\n\tSlice 2: %v\n\tSlice 3: %v", dstSlice[0], dstSlice[1], dstSlice[2])
		t.Logf("srcSlice:\n\tSlice 1: %v\n\tSlice 2: %v\n\tSlice 3: %v", srcSlice[0], srcSlice[1], srcSlice[2])

		if !pmatches(dstSlice[0][0], dstSlice[1][2], dstSlice[2][0]) ||
			!pmatches(dstSlice[0][1], dstSlice[1][1], dstSlice[2][2]) ||
			!pmatches(dstSlice[0][2], dstSlice[1][0], dstSlice[2][1]) {
			t.Fatalf("pointer mismatch in dst slice:\n\tSlice 1: %v\n\tSlice 2: %v\n\tSlice 3: %v", dstSlice[0], dstSlice[1], dstSlice[2])
		}

		if !pmatches(dstSlice[0][0], dstSlice[1][2].Value.backref, dstSlice[2][0].Value.backref) ||
			!pmatches(dstSlice[0][1], dstSlice[1][1].Value.backref, dstSlice[2][2].Value.backref) ||
			!pmatches(dstSlice[0][2], dstSlice[1][0].Value.backref, dstSlice[2][1].Value.backref) {
			t.Fatalf("pointer mismatch in dst slice:\n\tSlice 1: %v\n\tSlice 2: %v\n\tSlice 3: %v", dstSlice[0], dstSlice[1], dstSlice[2])
		}

		if pmatches(dstSlice[0][0], srcSlice[1][2]) ||
			pmatches(dstSlice[0][1], srcSlice[1][1]) ||
			pmatches(dstSlice[0][2], srcSlice[1][0]) {
			t.Fatalf("pointer mismatch in dst slice, reference to src slice:\n\tSlice 1: %v\n\tSlice 2: %v\n\tSlice 3: %v", dstSlice[0], dstSlice[1], dstSlice[2])
		}

	})

	t.Run("SliceRefs_REF_TYP", func(t *testing.T) {
		var (
			src1 = &TestStructValueModelFieldsRef{
				ID:   1,
				Name: "TestSliceRefs1",
				Objects: []int64{
					1, 2, 3,
				},
				Value: &testStructValueRef[*TestStructValueModelFieldsRef, int]{},
			}
			src2 = &TestStructValueModelFieldsRef{
				ID:   2,
				Name: "TestSliceRefs2",
				Objects: []int64{
					4, 5, 6,
				},
				Value: &testStructValueRef[*TestStructValueModelFieldsRef, int]{},
			}
			src3 = &TestStructValueModelFieldsRef{
				ID:   3,
				Name: "TestSliceRefs3",
				Objects: []int64{
					7, 8, 9,
				},
				Value: &testStructValueRef[*TestStructValueModelFieldsRef, int]{},
			}
		)

		src1.Value.backref = src1
		src1.Value.val = 1

		src2.Value.backref = src2
		src2.Value.val = 2

		src3.Value.backref = src3
		src3.Value.val = 3

		var (
			dstSlice [][]*TestStructValueModelFieldsRef
			srcSlice = [][]*TestStructValueModelFieldsRef{
				{src1, src2, src3},
				{src3, src2, src1},
				{src1, src3, src2},
			}
		)

		err := clone.Copy(t.Context(), &dstSlice, srcSlice)
		if err != nil {
			t.Fatalf("error during interface test: %v", err)
		}

		if len(dstSlice) != 3 {
			t.Fatalf("dst slice length: %d", len(dstSlice))
		}

		if len(dstSlice[0]) != 3 ||
			len(dstSlice[1]) != 3 ||
			len(dstSlice[2]) != 3 {
			t.Fatalf("dst slice length: %d/%d/%d", len(dstSlice[0]), len(dstSlice[1]), len(dstSlice[2]))
		}

		t.Logf("dstSlice:\n\tSlice 1: %v\n\tSlice 2: %v\n\tSlice 3: %v", dstSlice[0], dstSlice[1], dstSlice[2])
		t.Logf("srcSlice:\n\tSlice 1: %v\n\tSlice 2: %v\n\tSlice 3: %v", srcSlice[0], srcSlice[1], srcSlice[2])

		if !pmatches(dstSlice[0][0], dstSlice[1][2], dstSlice[2][0]) ||
			!pmatches(dstSlice[0][1], dstSlice[1][1], dstSlice[2][2]) ||
			!pmatches(dstSlice[0][2], dstSlice[1][0], dstSlice[2][1]) {
			t.Fatalf("pointer mismatch in dst slice:\n\tSlice 1: %v\n\tSlice 2: %v\n\tSlice 3: %v", dstSlice[0], dstSlice[1], dstSlice[2])
		}

		if !pmatches(dstSlice[0][0], dstSlice[1][2].Value.backref, dstSlice[2][0].Value.backref) ||
			!pmatches(dstSlice[0][1], dstSlice[1][1].Value.backref, dstSlice[2][2].Value.backref) ||
			!pmatches(dstSlice[0][2], dstSlice[1][0].Value.backref, dstSlice[2][1].Value.backref) {
			t.Fatalf("pointer mismatch in dst slice:\n\tSlice 1: %v\n\tSlice 2: %v\n\tSlice 3: %v", dstSlice[0], dstSlice[1], dstSlice[2])
		}

		if pmatches(dstSlice[0][0], srcSlice[1][2]) ||
			pmatches(dstSlice[0][1], srcSlice[1][1]) ||
			pmatches(dstSlice[0][2], srcSlice[1][0]) {
			t.Fatalf("pointer mismatch in dst slice, reference to src slice:\n\tSlice 1: %v\n\tSlice 2: %v\n\tSlice 3: %v", dstSlice[0], dstSlice[1], dstSlice[2])
		}

	})

	t.Run("SliceRefsAny", func(t *testing.T) {
		var (
			src1 = &TestModelFields{
				ID:   1,
				Name: "TestSliceRefs1",
				Objects: []int64{
					1, 2, 3,
				},
			}
			src2 = &TestModelFields{
				ID:   2,
				Name: "TestSliceRefs2",
				Objects: []int64{
					4, 5, 6,
				},
			}
			src3 = &TestModelFields{
				ID:   3,
				Name: "TestSliceRefs3",
				Objects: []int64{
					7, 8, 9,
				},
			}
			dstSlice [][]any
			srcSlice = [][]any{
				{src1, src2, src3},
				{src3, src2, src1},
				{src1, src3, src2},
			}
		)

		err := clone.Copy(t.Context(), &dstSlice, srcSlice)
		if err != nil {
			t.Fatalf("error during interface test: %v", err)
		}

		if len(dstSlice) != 3 {
			t.Fatalf("dst slice length: %d", len(dstSlice))
		}

		if len(dstSlice[0]) != 3 ||
			len(dstSlice[1]) != 3 ||
			len(dstSlice[2]) != 3 {
			t.Fatalf("dst slice length: %d/%d/%d", len(dstSlice[0]), len(dstSlice[1]), len(dstSlice[2]))
		}

		t.Logf("dstSlice:\n\tSlice 1: %v\n\tSlice 2: %v\n\tSlice 3: %v", dstSlice[0], dstSlice[1], dstSlice[2])
		t.Logf("srcSlice:\n\tSlice 1: %v\n\tSlice 2: %v\n\tSlice 3: %v", srcSlice[0], srcSlice[1], srcSlice[2])

		if !pmatches(dstSlice[0][0], dstSlice[1][2], dstSlice[2][0]) ||
			!pmatches(dstSlice[0][1], dstSlice[1][1], dstSlice[2][2]) ||
			!pmatches(dstSlice[0][2], dstSlice[1][0], dstSlice[2][1]) {
			t.Fatalf("pointer mismatch in dst slice:\n\tSlice 1: %v\n\tSlice 2: %v\n\tSlice 3: %v", dstSlice[0], dstSlice[1], dstSlice[2])
		}

		if pmatches(dstSlice[0][0], srcSlice[1][2]) ||
			pmatches(dstSlice[0][1], srcSlice[1][1]) ||
			pmatches(dstSlice[0][2], srcSlice[1][0]) {
			t.Fatalf("pointer mismatch in dst slice, reference to src slice:\n\tSlice 1: %v\n\tSlice 2: %v\n\tSlice 3: %v", dstSlice[0], dstSlice[1], dstSlice[2])
		}

	})

}

func pmatches(v ...any) bool {
	var l any
	for i, c := range v {
		if i == 0 {
			l = c
			continue
		}

		if c != l {
			return false
		}
	}

	return true
}
