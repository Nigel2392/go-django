package django_reflect_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/Nigel2392/go-django/src/internal/django_reflect"
)

type String string

func TestNewFuncValid(t *testing.T) {
	fn := func() string { return "hello" }
	f := django_reflect.NewFunc(fn, reflect.TypeOf(""))
	if f == nil || f.Type.Kind() != reflect.Func {
		t.Error("Expected valid Func instance")
	}
}

func TestNewFuncInvalidNonFunc(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for non-func input")
		}
	}()
	django_reflect.NewFunc("not a func", reflect.TypeOf(""))
}

func TestNewFuncMismatchedReturnTypes(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for mismatched return types")
		}
	}()
	fn := func() string { return "hello" }
	django_reflect.NewFunc(fn, reflect.TypeOf(123)) // mismatch
}

func TestReturnsSuccess(t *testing.T) {
	fn := func() int { return 42 }
	f := django_reflect.NewFunc(fn, reflect.TypeOf(0))
	f.Returns(reflect.TypeOf(0))
}

func TestReturnsMismatch(t *testing.T) {
	fn := func() float64 { return 42 }
	f := django_reflect.NewFunc(fn, reflect.TypeOf(float64(0)))

	for _, typ := range f.ReturnTypes {
		t.Logf("(GOOD) Return type: %v", typ)
	}

	defer func() {
		if r := recover(); r == nil {
			for _, typ := range f.ReturnTypes {
				t.Logf("(BAD) Return type: %v", typ)
			}

			t.Error("Expected panic for invalid Returns types")
		}
	}()

	f = f.Returns(reflect.TypeOf("")) // mismatch

	// This should panic because the return type does not match the function's return type
	f.Call()
}

func TestValidateSuccess(t *testing.T) {
	fn := func(i int) int { return i + 1 }
	f := django_reflect.NewFunc(fn, reflect.TypeOf(0))
	out := f.Call(10)
	if out[0].(int) != 11 {
		t.Errorf("Expected 11, got %v", out[0])
	}
}

func TestBeforeExecFails(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic due to validation failure")
		}
	}()

	var fn = func(i int) int { return i }
	var funcObj = django_reflect.NewFunc(fn, reflect.TypeOf(0))
	funcObj.BeforeExec = func([]reflect.Value) error {
		return errors.New("invalid")
	}

	funcObj.Call(10) // should panic due to BeforeExec validation failure
}

func TestCallWithConvert(t *testing.T) {
	fn := func(i int) string { return string(rune(i)) }
	f := django_reflect.NewFunc(fn, reflect.TypeOf(String("")))
	out := f.Call(int8(65))
	if out[0] != String("A") {
		t.Errorf("Expected 'A', got %v (%T, %T)", out[0], out[0], String(""))
	}
}

func TestCallWithRequires(t *testing.T) {
	fn := func(a, b, c int) string { return fmt.Sprintf("%d + %d + %d = %d", a, b, c, a+b+c) }
	f := django_reflect.NewFunc(fn, reflect.TypeOf(String("")))
	f.Requires(0, reflect.TypeOf(int(0))) // requires int
	f.Requires(1, reflect.TypeOf(int(0))) // requires int
	f.Requires(2, reflect.TypeOf(int(0))) // requires int

	// This should succeed because we are passing an int
	out := f.Call(65, 66, 67)
	if out[0] != String("65 + 66 + 67 = 198") {
		t.Errorf("Expected '65 + 66 + 67 = 198', got %v", out[0])
	}
}

func TestCallWithRequiresMismatch(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic due to type mismatch in Requires")
		} else {
			t.Logf("Recovered from panic: %v", r)
		}
	}()

	fn := func(a, b String) uint64 { return uint64(len(a) + len(b)) }

	f := django_reflect.NewFunc(fn, reflect.TypeOf(String("")))
	f.Requires(0, reflect.TypeOf(float64(0)))
	f.Requires(1, reflect.TypeOf(float64(0)))

	// This should panic because we are passing a float64 instead of an int
	f.Call("aaaa", "bbbb") // should panic due to type mismatch
}

type myInt int

func TestCallAdheresTo(t *testing.T) {
	fn := func(a int, b ...myInt) myInt { return myInt(a + int(b[0])) }
	f := django_reflect.NewFunc(fn, reflect.TypeOf(int(0)))

	// This should succeed because the function adheres to the expected signature
	if !f.AdheresTo(func(x uint, y ...uint) uint { return 0 }) {
		t.Error("Expected function to adhere to the signature")
	}

	if !f.AdheresTo(func(x int, y []myInt) int { return 0 }) {
		t.Error("Expected function to adhere to the signature with myInt")
	}

	// This should fail because the function does not adhere to the expected signature
	if f.AdheresTo(func(x, y string) string { return x + y }) {
		t.Error("Expected function not to adhere to the signature")
	}
}

func TestCallVariadic(t *testing.T) {
	fn := func(nums ...int) int {
		sum := 0
		for _, n := range nums {
			sum += n
		}
		return sum
	}
	f := django_reflect.NewFunc(fn, reflect.TypeOf(0))
	out := f.Call(1, 2, 3)
	if out[0].(int) != 6 {
		t.Errorf("Expected 6, got %v", out[0])
	}
}

func TestCallVariadicWithConvert(t *testing.T) {
	fn := func(nums ...int64) int64 {
		sum := int64(0)
		for _, n := range nums {
			sum += n
		}
		return sum
	}
	f := django_reflect.NewFunc(fn, reflect.TypeOf(int(0)))
	out := f.Call(float32(1.0), float32(2.0), float32(3.0)) // should convert float32 to int64
	if out[0].(int) != 6 {
		t.Errorf("Expected 6, got %v", out[0])
	}
}

func TestCallFuncMismatchReturn(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic on return type mismatch")
		}
	}()
	fn := func() string { return "hello" }
	f := django_reflect.NewFunc(fn, reflect.TypeOf(int(0))) // mismatch
	f.Call()
}
