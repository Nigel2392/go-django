package attrutils_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/src/core/attrs/attrutils"
)

type NilArgType struct{}

var (
	NIL_ARG = NilArgType{}
	ErrTest = errors.New("test error")
)

type BuilderObject struct {
	LastArg any
}

func (b BuilderObject) Method__StringArg(arg string) BuilderObject {
	b.LastArg = arg
	return b
}

func (b BuilderObject) Method__IntArg__Variadic(arg int, args ...int) (BuilderObject, []int) {
	b.LastArg = arg
	return b, append([]int{arg}, args...)
}

func (b BuilderObject) Method__NoArgs() (BuilderObject, error) {
	b.LastArg = NIL_ARG
	return b, ErrTest
}

func (b BuilderObject) Method__Error() (BuilderObject, error) {
	return BuilderObject{}, ErrTest
}

func (b BuilderObject) Method__MultipleReturns() (BuilderObject, int, string) {
	return b, 42, "test"
}

func (b BuilderObject) Method__MultipleReturnsWithError() (BuilderObject, int, error) {
	return BuilderObject{}, 0, ErrTest
}

func (b BuilderObject) Method__AnyArg__Variadic__MultipleReturnsWithError(arg any, args ...any) (BuilderObject, []any, error) {
	return b, append([]any{arg}, args...), ErrTest
}

// Wrong first return type -> should trigger TypeMismatch path in builderMethod,
// keeping the original builder and exposing all returns as retArgs.
func (b BuilderObject) Method__WrongFirstReturnType() (int, string) {
	return 7, "x"
}

// No returns -> builderMethod should report (builder unchanged, nil, nil).
func (b BuilderObject) Method__NoReturns() {}

// Pointer receiver, no returns: lets us exercise Chain success (no extra returns)
// while still mutating state via pointer receiver.
func (b *BuilderObject) Method__SetLastArgNoReturns(v any) { b.LastArg = v }

// --- Helper assertions ---

func mustNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func mustErrContains(t *testing.T, err error, sub string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", sub)
	}
	if !strings.Contains(err.Error(), sub) {
		t.Fatalf("expected error containing %q, got %q", sub, err.Error())
	}
}

// --- Tests for top-level BuilderMethod / builderMethod behavior ---
// A type that implements attrutils.ObjectBuilder[T] should take the fast path.
type MyBuilder struct {
	Called string
	Args   []any
}

// Implement the interface expected by BuilderMethod.
var _ attrutils.ObjectBuilder[MyBuilder] = (*MyBuilder)(nil)

func (m MyBuilder) CallBuilderMethod(methodName string, args ...any) (MyBuilder, []any, error) {
	m.Called = methodName
	m.Args = append([]any(nil), args...)
	return m, m.Args, nil
}

func TestBuilderMethod_InterfaceOverride(t *testing.T) {

	mb := MyBuilder{}
	out, ret, err := attrutils.BuilderMethod(mb, "Hello", 1, "x", true)
	mustNoErr(t, err)

	if out.Called != "Hello" {
		t.Fatalf("expected Called=Hello, got %q", out.Called)
	}
	wantArgs := []any{1, "x", true}
	if !reflect.DeepEqual(out.Args, wantArgs) || !reflect.DeepEqual(ret, wantArgs) {
		t.Fatalf("expected args %v round-tripped, got out.Args=%v ret=%v", wantArgs, out.Args, ret)
	}
}

func TestBuilderMethod_MethodNotFound(t *testing.T) {
	obj := BuilderObject{}
	_, _, err := attrutils.BuilderMethod(obj, "DoesNotExist")
	mustErrContains(t, err, "method DoesNotExist not found")
}

func TestBuilderMethod_ArgConversionSuccess_StringArg(t *testing.T) {
	obj := BuilderObject{}
	out, ret, err := attrutils.BuilderMethod(obj, "Method__StringArg", "hello")
	mustNoErr(t, err)
	if len(ret) != 0 {
		t.Fatalf("expected no extra return values, got %v", ret)
	}
	if out.LastArg != "hello" {
		t.Fatalf("expected LastArg=hello, got %#v", out.LastArg)
	}
}

func TestBuilderMethod_ArgConversionFailure_WrongType(t *testing.T) {
	obj := BuilderObject{}
	_, _, err := attrutils.BuilderMethod(obj, "Method__StringArg", func() {}) // not a string
	if err == nil {
		t.Fatalf("expected conversion error, got nil")
	}
}

func TestBuilderMethod_Variadic_Ints(t *testing.T) {
	obj := BuilderObject{}
	out, ret, err := attrutils.BuilderMethod(obj, "Method__IntArg__Variadic", 1, 2, 3)
	mustNoErr(t, err)

	if out.LastArg != 1 {
		t.Fatalf("expected LastArg=1, got %#v", out.LastArg)
	}
	if len(ret) != 1 {
		t.Fatalf("expected one extra return value (the []int), got %v", ret)
	}
	ints, ok := ret[0].([]int)
	if !ok {
		t.Fatalf("expected ret[0] to be []int, got %T: %#v", ret[0], ret[0])
	}
	if !reflect.DeepEqual(ints, []int{1, 2, 3}) {
		t.Fatalf("expected []int{1,2,3}, got %v", ints)
	}
}

func TestBuilderMethod_NoArgs_WithErrorAtEnd(t *testing.T) {
	obj := BuilderObject{}
	out, ret, err := attrutils.BuilderMethod(obj, "Method__NoArgs")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if len(ret) != 0 {
		t.Fatalf("expected no extra return values, got %v", ret)
	}
	if out.LastArg != NIL_ARG {
		t.Fatalf("expected LastArg=NIL_ARG, got %#v", out.LastArg)
	}
}

func TestBuilderMethod_ErrorOnlyZeroBuilder(t *testing.T) {
	obj := BuilderObject{}
	out, ret, err := attrutils.BuilderMethod(obj, "Method__Error")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if len(ret) != 0 {
		t.Fatalf("expected no extra return values, got %v", ret)
	}
	// Should be zero value
	if out.LastArg != nil {
		t.Fatalf("expected zero-value BuilderObject (LastArg=nil), got %#v", out.LastArg)
	}
}

func TestBuilderMethod_MultipleReturns_NoError(t *testing.T) {
	obj := BuilderObject{}
	out, ret, err := attrutils.BuilderMethod(obj, "Method__MultipleReturns")
	mustNoErr(t, err)

	if out.LastArg != nil {
		t.Fatalf("expected LastArg unchanged (nil), got %#v", out.LastArg)
	}
	if len(ret) != 2 {
		t.Fatalf("expected 2 extra return values, got %v", ret)
	}
	if ret[0] != 42 || ret[1] != "test" {
		t.Fatalf("expected (42, \"test\"), got (%#v, %#v)", ret[0], ret[1])
	}
}

func TestBuilderMethod_MultipleReturns_WithError(t *testing.T) {
	obj := BuilderObject{}
	out, ret, err := attrutils.BuilderMethod(obj, "Method__MultipleReturnsWithError")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	// first return is the builder (zero value per implementation)
	if out.LastArg != nil {
		t.Fatalf("expected zero-value BuilderObject, got %#v", out.LastArg)
	}
	if len(ret) != 1 || ret[0] != 0 {
		t.Fatalf("expected single extra return value 0, got %v", ret)
	}
}

func TestBuilderMethod_VariadicAny_WithError(t *testing.T) {
	obj := BuilderObject{}
	out, ret, err := attrutils.BuilderMethod(obj, "Method__AnyArg__Variadic__MultipleReturnsWithError", "a", 1, true)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	// builder (first return) is unchanged here (value receiver returning b)
	if out.LastArg != nil {
		t.Fatalf("expected LastArg unchanged (nil), got %#v", out.LastArg)
	}
	if len(ret) != 1 {
		t.Fatalf("expected a single []any return value, got %v", ret)
	}
	gotSlice, ok := ret[0].([]any)
	if !ok {
		t.Fatalf("expected ret[0] to be []any, got %T", ret[0])
	}
	wantSlice := []any{"a", 1, true}
	if !reflect.DeepEqual(gotSlice, wantSlice) {
		t.Fatalf("expected %v, got %v", wantSlice, gotSlice)
	}
}

func TestBuilderMethod_WrongFirstReturnType_TypeMismatchPath(t *testing.T) {
	obj := BuilderObject{}
	out, ret, err := attrutils.BuilderMethod(obj, "Method__WrongFirstReturnType")
	mustNoErr(t, err)

	// Since first return isn't convertible to BuilderObject, original builder should be kept.
	if out.LastArg != nil {
		t.Fatalf("expected original builder (LastArg nil), got %#v", out.LastArg)
	}
	if len(ret) != 2 || ret[0] != 7 || ret[1] != "x" {
		t.Fatalf("expected ret=[7, \"x\"], got %v", ret)
	}
}

func TestBuilderMethod_NoReturns(t *testing.T) {
	obj := BuilderObject{}
	out, ret, err := attrutils.BuilderMethod(obj, "Method__NoReturns")
	mustNoErr(t, err)
	if len(ret) != 0 {
		t.Fatalf("expected no return values, got %v", ret)
	}
	// unchanged
	if out != obj {
		t.Fatalf("expected builder unchanged")
	}
}

// --- Tests for the Builder[T] wrapper ---

func TestBuilder_New_Orig_Ref(t *testing.T) {
	obj := BuilderObject{}
	b := attrutils.NewBuilder(obj)
	if b.Orig() != obj || b.Ref() != obj {
		t.Fatalf("expected Orig and Ref to equal initial object")
	}
}

func TestBuilder_Call_Success_UpdatesRef(t *testing.T) {
	obj := BuilderObject{}
	b := attrutils.NewBuilder(obj)
	ref, ret, err := b.Call("Method__StringArg", "hi")
	mustNoErr(t, err)
	if len(ret) != 0 {
		t.Fatalf("expected no ret values, got %v", ret)
	}
	if ref.LastArg != "hi" {
		t.Fatalf("expected Ref.LastArg to be updated to 'hi', got %#v", ref.LastArg)
	}
	if b.Ref().LastArg != "hi" {
		t.Fatalf("builder's internal ref not updated")
	}
}

func TestBuilder_Call_Failure_DoesNotUpdateRef(t *testing.T) {
	obj := BuilderObject{}
	b := attrutils.NewBuilder(obj)
	_, _, err := b.Call("Method__StringArg", 999) // wrong type
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	// Ref should remain unchanged
	if b.Ref().LastArg != nil {
		t.Fatalf("expected Ref unchanged on error, got %#v", b.Ref().LastArg)
	}
}

func TestBuilder_Chain_Success_NoReturns_PointerMethod(t *testing.T) {
	ptr := &BuilderObject{}
	b := attrutils.NewBuilder(ptr)
	// This method mutates via pointer receiver and returns nothing (valid for Chain).
	b = b.Chain("Method__SetLastArgNoReturns", "chained")
	if b.Error() != nil {
		t.Fatalf("unexpected chain error: %v", b.Error())
	}
	if b.Ref().LastArg != "chained" {
		t.Fatalf("expected mutation via chain, got %#v", b.Ref().LastArg)
	}
}

func TestBuilder_Chain_ReturnsValues_SetsError_AndStops(t *testing.T) {
	obj := BuilderObject{}
	b := attrutils.NewBuilder(obj)

	// Using a method that returns values should mark an error in Chain.
	b = b.Chain("Method__AnyArg__Variadic__MultipleReturnsWithError", "oops")
	if b.Error() == nil {
		t.Fatalf("expected chain error due to unexpected return values")
	}

	// Further chains should be no-ops.
	prev := b.Ref()
	b = b.Chain("Method__AnyArg__Variadic__MultipleReturnsWithError", "ignored")
	if b.Ref() != prev {
		t.Fatalf("expected no change after chain error")
	}

	// Call should also surface the existing error without attempting invocation.
	_, _, err := b.Call("Method__AnyArg__Variadic__MultipleReturnsWithError", "still ignored")
	if err == nil {
		t.Fatalf("expected previously stored error from chain")
	}
}
