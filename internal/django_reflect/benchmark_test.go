package django_reflect_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/Nigel2392/go-django/internal/django_reflect"
)

// =========================================================================
// 1. REGULAR TYPES BENCHMARKS (5 Benchmarks)
// =========================================================================

func BenchmarkCast_Regular_ExactMatch_Creation(b *testing.B) {
	src := func(a int, s string) int { return a + len(s) }
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = django_reflect.CastFunc[func(int, string) int](src)
	}
}

func BenchmarkCast_Regular_ExactMatch_Execution(b *testing.B) {
	src := func(a int, s string) int { return a + len(s) }
	out, _ := django_reflect.CastFunc[func(int, string) int](src)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = out(5, "hello")
	}
}

func BenchmarkCast_Regular_Convertible_Creation(b *testing.B) {
	src := func(a int32, b float32) float64 { return float64(a) + float64(b) }
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = django_reflect.CastFunc[func(int, float64) float64](src)
	}
}

func BenchmarkCast_Regular_Convertible_Execution(b *testing.B) {
	src := func(a int32, b float32) float64 { return float64(a) + float64(b) }
	out, _ := django_reflect.CastFunc[func(int, float64) float64](src)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = out(10, 5.5)
	}
}

func BenchmarkCast_Regular_Variadic_Execution(b *testing.B) {
	src := func(a int, c ...int) int {
		sum := a
		for _, v := range c {
			sum += v
		}
		return sum
	}
	out, _ := django_reflect.CastFunc[func(int, int, int) int](src)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = out(1, 2, 3)
	}
}

// =========================================================================
// 2. INTERFACES BENCHMARKS (5 Benchmarks)
// =========================================================================

func BenchmarkCast_Interface_AnyToAny_Execution(b *testing.B) {
	src := func(a any) any { return a }
	out, _ := django_reflect.CastFunc[func(any) any](src)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = out(42)
	}
}

func BenchmarkCast_Interface_SpecificToAny_Execution(b *testing.B) {
	src := func(a any) int {
		if v, ok := a.(int); ok {
			return v * 2
		}
		return 0
	}
	out, _ := django_reflect.CastFunc[func(int) int](src)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = out(21)
	}
}

func BenchmarkCast_Interface_AnyToSpecific_Execution(b *testing.B) {
	src := func(a int) int { return a * 2 }
	out, _ := django_reflect.CastFunc[func(any) int](src)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = out(21)
	}
}

func BenchmarkCast_Interface_StringerToBroad_Execution(b *testing.B) {
	src := func(s fmt.Stringer) string { return s.String() }
	out, _ := django_reflect.CastFunc[func(BroadStringer) string](src)

	// dummyBroadStringer is defined in func_cast_test.go
	d := dummyBroadStringer{val: "bench"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = out(d)
	}
}

type bDummyWriter struct{}

func (w bDummyWriter) Write(p []byte) (n int, err error) { return len(p), nil }

func BenchmarkCast_Interface_MethodUnwrap_Execution(b *testing.B) {
	type Writer interface {
		Write(p []byte) (n int, err error)
	}
	src := func(w Writer) int {
		n, _ := w.Write([]byte("bench"))
		return n
	}
	out, _ := django_reflect.CastFunc[func(any) int](src)
	dw := bDummyWriter{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = out(dw)
	}
}

func BenchmarkCast_Wrapper_Context_Creation(b *testing.B) {
	ctx := context.Background()
	src := func(ctx context.Context, a int) int { return a * 2 }
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = django_reflect.CastFunc[func(int) int](src, django_reflect.WrapWithContext(ctx))
	}
}

func BenchmarkCast_Wrapper_Context_Execution(b *testing.B) {
	ctx := context.Background()
	src := func(ctx context.Context, a int) int { return a * 2 }
	out, _ := django_reflect.CastFunc[func(int) int](src, django_reflect.WrapWithContext(ctx))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = out(5)
	}
}

func BenchmarkCast_Wrapper_Context_NoReturns_Execution(b *testing.B) {
	ctx := context.Background()
	src := func(ctx context.Context) {}
	out, _ := django_reflect.CastFunc[func()](src, django_reflect.WrapWithContext(ctx))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out()
	}
}

type bModel struct{}

func (m *bModel) Save(ctx context.Context) error { return nil }

func BenchmarkCast_Wrapper_Method_Execution(b *testing.B) {
	ctx := context.Background()
	m := &bModel{}
	saveFn, _ := django_reflect.Method[func() error](m, "Save", django_reflect.WrapWithContext(ctx))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = saveFn()
	}
}

func BenchmarkCast_Wrapper_Context_MultiReturn_Execution(b *testing.B) {
	ctx := context.Background()
	src := func(ctx context.Context, s string) (string, int, error) { return s, len(s), nil }

	// Cast down to error (dropping the string and int)
	out, _ := django_reflect.CastFunc[func(string) error](src, django_reflect.WrapWithContext(ctx))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = out("test")
	}
}

func BenchmarkCast_Wrapper_Context_DestTakesContext1_Execution(b *testing.B) {
	ctx := context.Background()
	src := func(ctx context.Context, a int) int { return a * 2 }
	out, _ := django_reflect.CastFunc[func(context.Context, int) int](src, django_reflect.WrapWithContext(ctx))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = out(context.Background(), 5)
	}
}

func BenchmarkCast_Wrapper_Context_DestTakesContext2_Execution(b *testing.B) {
	ctx := context.Background()
	src := func(ctx context.Context, s string) (string, error) { return s, nil }
	out, _ := django_reflect.CastFunc[func(context.Context, any) any](src, django_reflect.WrapWithContext(ctx))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = out(context.Background(), "test")
	}
}

func BenchmarkCast_10Args_ExactMatch_Execution(b *testing.B) {
	src := func(a, b_, c, d, e, f, g, h, i, j int) int { return a + j }
	out, _ := django_reflect.CastFunc[func(int, int, int, int, int, int, int, int, int, int) int](src)
	b.ResetTimer()
	for x := 0; x < b.N; x++ {
		_ = out(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	}
}

func BenchmarkCast_10Args_Convertible_Execution(b *testing.B) {
	src := func(a, b_, c, d, e, f, g, h, i, j int32) int64 { return int64(a + j) }
	out, _ := django_reflect.CastFunc[func(int8, int8, int8, int8, int8, int8, int8, int8, int8, int8) int32](src)
	b.ResetTimer()
	for x := 0; x < b.N; x++ {
		_ = out(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	}
}

func BenchmarkCast_10Args_AnyToSpecific_Execution(b *testing.B) {
	src := func(a, b_, c, d, e, f, g, h, i, j int) int { return a + j }
	out, _ := django_reflect.CastFunc[func(any, any, any, any, any, any, any, any, any, any) float64](src)
	b.ResetTimer()
	for x := 0; x < b.N; x++ {
		_ = out(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	}
}

func BenchmarkCast_10Args_Variadic_Execution(b *testing.B) {
	src := func(a ...int) int { return a[0] + a[len(a)-1] }
	out, _ := django_reflect.CastFunc[func(any, any, any, any, any, any, any, any, any, any) float64](src)
	b.ResetTimer()
	for x := 0; x < b.N; x++ {
		_ = out(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	}
}

func BenchmarkCast_10Args_Wrapper_Context_Execution(b *testing.B) {
	ctx := context.Background()
	src := func(ctx context.Context, a, b_, c, d, e, f, g, h, i int64) float32 { return float32(a + i) }
	out, _ := django_reflect.CastFunc[func(int8, int8, int8, int8, int8, int8, int8, int8, int8) int64](src, django_reflect.WrapWithContext(ctx))
	b.ResetTimer()
	for x := 0; x < b.N; x++ {
		_ = out(1, 2, 3, 4, 5, 6, 7, 8, 9)
	}
}
