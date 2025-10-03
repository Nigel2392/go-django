package django_reflect_test

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"runtime"
	"testing"

	"github.com/Nigel2392/go-django/src/internal/django_reflect"
)

// Helper to check no error
func mustNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
}

// Helper to check error is (or wraps) expected sentinel
func mustIsErr(t *testing.T, err error, target error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %v, got nil", target)
	}
	if !errors.Is(err, target) {
		t.Fatalf("expected error %v, got %v", target, err)
	}
}

const stackSize = 8

// Helper to assert panic
func mustPanic(t *testing.T, fn func(), contains string) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic containing %q, got none", contains)
		}
		if contains != "" {
			msg := ""
			switch v := r.(type) {
			case string:
				msg = v
			case error:
				msg = v.Error()
			default:
				msg = reflect.TypeOf(r).String()
			}
			if !containsIn(msg, contains) {
				for i := 2; i < stackSize; i++ {
					var _, file, line, ok = runtime.Caller(i)
					if !ok {
						break
					}
					t.Logf("  at %s:%d", file, line)
				}

				t.Fatalf("expected panic containing %q, got %q", contains, msg)
			}
		}
	}()
	fn()
}

func containsIn(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) >= len(needle) && (func() bool {
		return bytes.Contains([]byte(haystack), []byte(needle))
	})())
}

// --- Success Cases ---

func TestCast_ExactMatch(t *testing.T) {
	src := func(a int, b string) error { return nil }
	out, err := django_reflect.CastFunc[func(int, string) error](src)
	mustNoErr(t, err)
	if out == nil {
		t.Fatalf("expected non-nil function")
	}
	if e := out(10, "ok"); e != nil {
		t.Fatalf("unexpected error from casted function: %v", e)
	}
}

func TestCast_InterfaceMatch(t *testing.T) {
	src := func(a int, b string) error { return nil }
	out, err := django_reflect.CastFunc[func(any, string) error](src)
	mustNoErr(t, err)
	if out == nil {
		t.Fatalf("expected non-nil function")
	}
	if e := out(10, "ok"); e != nil {
		t.Fatalf("unexpected error from casted function: %v", e)
	}
}

func TestCast_InterfaceMatchAll(t *testing.T) {
	src := func(a int, b string) error { return nil }
	out, err := django_reflect.CastFunc[func(any, any) any](src)
	mustNoErr(t, err)
	if out == nil {
		t.Fatalf("expected non-nil function")
	}
	if e := out(10, "ok"); e != nil {
		t.Fatalf("unexpected error from casted function: %v", e)
	}
}

func TestCast_ConvertibleNumericArgs(t *testing.T) {
	src := func(a, b float64) float64 { return a + b }
	out, err := django_reflect.CastFunc[func(int, int) float64](src)
	mustNoErr(t, err)
	sum := out(2, 5) // ints convertible to float64
	if sum != 7 {
		t.Fatalf("expected 7, got %v", sum)
	}
}

func TestCast_VariadicSource_ToFixedTarget(t *testing.T) {
	src := func(xs ...any) (bool, error) {
		if len(xs) == 2 {
			if ai, ok := xs[0].(int); ok {
				if bi, ok2 := xs[1].(int); ok2 {
					return ai == bi, nil
				}
			}
		}
		return false, nil
	}
	out, err := django_reflect.CastFunc[func(int, int) (bool, error)](src)
	mustNoErr(t, err)
	ok, e := out(3, 3)
	if e != nil {
		t.Fatalf("unexpected error: %v", e)
	}
	if !ok {
		t.Fatalf("expected equality result true")
	}
}

func TestCast_FixedSource_ToVariadicTarget_ExactArity1(t *testing.T) {
	src := func(a, b int) (bool, error) { return a == b, nil }
	out, err := django_reflect.CastFunc[func(...any) (bool, error)](src)
	mustNoErr(t, err)
	ok, e := out(4, 4)
	if e != nil {
		t.Fatalf("unexpected error: %v", e)
	}
	if !ok {
		t.Fatalf("expected equality true")
	}
	// Calling with 3 args should panic due to underlying fixed arity (behavior documented)
	mustPanic(t, func() {
		_, _ = out(1, 2, 3)
	}, "same number of arguments")
}

func TestCast_FixedSource_ToVariadicTarget_ExactArity2(t *testing.T) {
	src := func(a, b int) (bool, error) { return a == b, nil }
	out, err := django_reflect.CastFunc[func(int, ...any) (bool, error)](src)
	mustNoErr(t, err)
	ok, e := out(4, 4)
	if e != nil {
		t.Fatalf("unexpected error: %v", e)
	}
	if !ok {
		t.Fatalf("expected equality true")
	}
	// Calling with 3 args should panic due to underlying fixed arity (behavior documented)
	mustPanic(t, func() {
		_, _ = out(1, 2, 3)
	}, "same number of arguments")
}

func TestCast_InterfaceReturn(t *testing.T) {
	src := func() *bytes.Buffer {
		var b bytes.Buffer
		b.WriteString("hello")
		return &b
	}
	out, err := django_reflect.CastFunc[func() io.Writer](src)
	mustNoErr(t, err)
	w := out()
	if w == nil {
		t.Fatalf("expected non-nil writer")
	}
	if _, e := w.Write([]byte(" world")); e != nil {
		t.Fatalf("unexpected write error: %v", e)
	}
}

func TestCast_InterfaceParam(t *testing.T) {
	src := func(w io.Writer) error {
		_, err := w.Write([]byte("data"))
		return err
	}
	out, err := django_reflect.CastFunc[func(*bytes.Buffer) error](src)
	mustNoErr(t, err)
	var b bytes.Buffer
	if e := out(&b); e != nil {
		t.Fatalf("unexpected error: %v", e)
	}
	if got := b.String(); got != "data" {
		t.Fatalf("expected 'data', got %q", got)
	}
}

func TestCast_ConvertibleReturn(t *testing.T) {
	src := func() int32 { return 42 }
	out, err := django_reflect.CastFunc[func() int64](src)
	mustNoErr(t, err)
	if v := out(); v != 42 {
		t.Fatalf("expected 42, got %v", v)
	}
}

// ---django_reflect. Error Cases (creation time) ---

func TestCast_ArgCountMismatch(t *testing.T) {
	src := func(a int) error { return nil }
	_, err := django_reflect.CastFunc[func(int, int) error](src)
	mustIsErr(t, err, django_reflect.ErrArgCount)
}

func TestCast_ReturnCountMismatch(t *testing.T) {
	src := func(a int) (int, error) { return a, nil }
	_, err := django_reflect.CastFunc[func(int) int](src)
	mustIsErr(t, err, django_reflect.ErrReturnCount)
}

func TestCast_NotFunc_Source(t *testing.T) {
	var notFn = 123
	outType := reflect.TypeOf(func() {})
	_, err := django_reflect.RCastFunc(outType, notFn)
	mustIsErr(t, err, django_reflect.ErrNotFunc)
}

func TestCast_NotFunc_Target(t *testing.T) {
	src := func() {}
	_, err := django_reflect.RCastFunc(reflect.TypeOf(123), src)
	mustIsErr(t, err, django_reflect.ErrNotFunc)
}

func TestCast_NilFunction(t *testing.T) {
	var fn any = nil
	outType := reflect.TypeOf(func() {})
	_, err := django_reflect.RCastFunc(outType, fn)
	mustIsErr(t, err, django_reflect.ErrNotFunc)
}

// --- Runtime Panic Cases (inside wrapper) ---

func TestCast_RuntimePanic_OnUnconvertibleArg_IntToString(t *testing.T) {
	src := func(s string) int { return len(s) }
	// Target expects int param; wrapper created (sizes match) but conversion fails at call time.
	out, err := django_reflect.CastFunc[func(int) int](src)
	mustNoErr(t, err)
	mustPanic(t, func() {
		_ = out(5) // int cannot convert to string
	}, "could not convert")
}

func TestCast_RuntimePanic_OnUnconvertibleArg_StringToInt(t *testing.T) {
	src := func(i int) int { return i * 2 }
	out, err := django_reflect.CastFunc[func(string) int](src)
	mustNoErr(t, err)
	mustPanic(t, func() {
		_ = out("7") // string not convertible to int via reflect.ConvertibleTo
	}, "could not convert")
}

func TestCast_RuntimePanic_TooManyArgsForFixedSource(t *testing.T) {
	src := func(a int, b int) int { return a + b }
	_, err := django_reflect.CastFunc[func(int, int, int) int](src) // Allowed because destination considered variadic? No; both non-variadic -> should error
	if err == nil {
		t.Fatalf("expected argument count error, got nil")
	}
	mustIsErr(t, err, django_reflect.ErrArgCount)
}

func TestCast_VariadicSource_ExtraArgs(t *testing.T) {
	src := func(xs ...int) int {
		sum := 0
		for _, v := range xs {
			sum += v
		}
		return sum
	}
	out, err := django_reflect.CastFunc[func(int, int, int) int](src)
	mustNoErr(t, err)
	if got := out(1, 2, 3); got != 6 {
		t.Fatalf("expected 6, got %v", got)
	}
	t.Logf("underlying type: %T", out)
}

// --- Direct Function Type Conversion Path ---

type myAdder func(int, int) int

func TestCast_DirectConvertibleFunctionType(t *testing.T) {
	src := func(a, b int) int { return a + b }
	out, err := django_reflect.CastFunc[myAdder](src)
	mustNoErr(t, err)
	if out(2, 3) != 5 {
		t.Fatalf("expected 5")
	}
	t.Logf("underlying type: %T", out)
}

// --- Multiple Return Conversion ---

func TestCast_MultipleReturnConvertible(t *testing.T) {
	src := func() (int32, float32) { return 10, 2.5 }
	out, err := django_reflect.CastFunc[func() (int64, float64)](src)
	mustNoErr(t, err)
	i, f := out()
	if i != 10 || f != 2.5 {
		t.Fatalf("expected (10,2.5) got (%v,%v)", i, f)
	}
	t.Logf("underlying types: %T, %T", i, f)
}

// --- Ensure Wrapper Uses Out Return Types ---

func TestCast_ReturnInterfaceWidening(t *testing.T) {
	type stringer interface {
		String() string
	}
	src := func() *bytes.Buffer {
		var b bytes.Buffer
		b.WriteString("abc")
		return &b
	}
	out, err := django_reflect.CastFunc[func() stringer](src)
	mustNoErr(t, err)
	s := out()
	if s.String() != "abc" {
		t.Fatalf("expected 'abc', got %q", s.String())
	}
	t.Logf("underlying type: %T", s)
}

// --- Edge: Zero Return Functions ---

func TestCast_NoReturn(t *testing.T) {
	src := func(a int) { t.Logf("got %d", a) }
	out, err := django_reflect.CastFunc[func(int)](src)
	mustNoErr(t, err)
	out(5)
}

func TestCast_NoReturn_VariadicForward(t *testing.T) {
	src := func(xs ...any) { t.Logf("got %d args: %v", len(xs), xs) }
	out, err := django_reflect.CastFunc[func(int, string)](src)
	mustNoErr(t, err)
	out(10, "x")
}

// --- Edge: Variadic Destination Only (fake variadic) ---

func TestCast_DestinationVariadicOnly(t *testing.T) {
	src := func(a int) int { return a * 2 }
	// Destination is variadic, source is not; wrapper will not truly accept >1 arg.
	out, err := django_reflect.CastFunc[func(...int) int](src)
	mustNoErr(t, err)
	if v := out(3); v != 6 {
		t.Fatalf("expected 6, got %v", v)
	}
	mustPanic(t, func() {
		_ = out(1, 2) // should panic due to assertion (not real variadic bridging)
	}, "same number of arguments")
}

func TestCast_FixedSource_ToVariadicTarget_UnwrapInterfaceArgs(t *testing.T) {
	src := func(a, b int) int { return a + b }
	out, err := django_reflect.CastFunc[func(...any) int](src)
	mustNoErr(t, err)
	if sum := out(2, 5); sum != 7 {
		t.Fatalf("expected 7, got %v", sum)
	}
	mustPanic(t, func() { _ = out(1, 2, 3) }, "same number of arguments")
}

func TestNonVariadicToVariadicArgs(t *testing.T) {
	var fn = func(a, b string, c ...int) (string, int, error) {
		return a + b, len(c), nil
	}
	f, err := django_reflect.CastFunc[func(string, []byte, uint, uint, uint) (string, int, error)](fn)
	mustNoErr(t, err)
	s, l, e := f("hello ", []byte("world"), 1, 2, 3)
	mustNoErr(t, e)

	if s != "hello world" {
		t.Fatalf("expected 'hello world', got %q", s)
	}

	if l != 3 {
		t.Fatalf("expected length 3, got %d", l)
	}

	t.Logf("got %q and length %d", s, l)
}

func TestStringToByteReturn(t *testing.T) {
	var fn = func(a, b string, c ...int) (string, int, error) {
		return a + b, len(c), nil
	}

	f, err := django_reflect.CastFunc[func(string, string, int, int, int) ([]byte, int, error)](fn)
	mustNoErr(t, err)
	s, l, e := f("hello ", "world", 1, 2, 3)
	mustNoErr(t, e)
	if string(s) != "hello world" {
		t.Fatalf("expected 'hello world', got %q", s)
	}
	if l != 3 {
		t.Fatalf("expected length 3, got %d", l)
	}
	t.Logf("got %q and length %d", s, l)
}

func TestCast_DiscardExtraReturns_ToErrorOnly(t *testing.T) {
	src := func(s string, n int) (string, int, error) {
		return s + "!", n * 2, nil
	}

	out, err := django_reflect.CastFunc[func(string, int) error](src)
	mustNoErr(t, err)

	e := out("hello", 3)
	if e != nil {
		t.Fatalf("unexpected error: %v", e)
	}
}

func TestCast_DiscardExtraReturns_ToNoReturn(t *testing.T) {
	src := func(s string, n int) (string, int, error) {
		return s + "!", n * 2, nil
	}

	out, err := django_reflect.CastFunc[func(string, int)](src)
	mustNoErr(t, err)

	// Just call it; no return values expected
	out("hi", 5)
}

func TestCast_DiscardPartialReturn_ToErrorOnlyFromTwoReturns(t *testing.T) {
	src := func(s string, n int) (string, int) {
		return s + "!", n * 2
	}

	out, err := django_reflect.CastFunc[func(string, int) error](src)
	mustIsErr(t, err, django_reflect.ErrReturnCount)

	_ = out
}

func TestCast_AnyReturns(t *testing.T) {
	src := func(s string) (string, error) {
		return s + "!", nil
	}

	out, err := django_reflect.CastFunc[func(string) any](src)
	mustNoErr(t, err)

	// Just call it; no return values expected
	v := out("hi")
	if lst, ok := v.([]any); !ok {
		t.Fatalf("expected []any return, got %T", v)
	} else if len(lst) != 2 {
		t.Fatalf("expected 2 values in []any, got %d", len(lst))
	}
}

func TestCast_AnyErrorReturns(t *testing.T) {
	src := func(s string) (string, error) {
		return s + "!", nil
	}

	out, err := django_reflect.CastFunc[func(string) (any, error)](src)
	mustNoErr(t, err)

	// Just call it; no return values expected
	v, err := out("hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res, ok := v.(string); !ok {
		t.Fatalf("expected string return, got %T", v)
	} else if res != "hi!" {
		t.Fatalf("expected 'hi!', got %q", res)
	}
}

func TestCast_AnyListReturns(t *testing.T) {
	src := func(s string, n int) (string, int, error) {
		return s + "!", n * 2, nil
	}

	out, err := django_reflect.CastFunc[func(string, int) any](src)
	mustNoErr(t, err)

	// Just call it; no return values expected
	v := out("hi", 5)
	if lst, ok := v.([]any); !ok {
		t.Fatalf("expected []any return, got %T", v)
	} else if len(lst) != 3 {
		t.Fatalf("expected 3 values in []any, got %d", len(lst))
	}
}

func TestCast_AnyListErrorReturns(t *testing.T) {
	src := func(s string, n int) (string, int, error) {
		return s + "!", n * 2, nil
	}

	out, err := django_reflect.CastFunc[func(string, int) (any, error)](src)
	mustNoErr(t, err)

	// Just call it; no return values expected
	v, err := out("hi", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if lst, ok := v.([]any); !ok {
		t.Fatalf("expected []any return, got %T", v)
	} else if len(lst) != 2 {
		t.Fatalf("expected 2 values in []any, got %d", len(lst))
	}
}
