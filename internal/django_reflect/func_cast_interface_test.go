package django_reflect_test

import (
	"fmt"
	"testing"

	"github.com/Nigel2392/go-django/internal/django_reflect"
)

// 1. Speaker / LoudSpeaker
type Speaker interface{ Speak() string }
type LoudSpeaker interface {
	Speak() string
	Yell() string
}
type myLoudSpeaker struct{}

func (m myLoudSpeaker) Speak() string { return "speak" }
func (m myLoudSpeaker) Yell() string  { return "YELL" }

type simpleSpeaker struct{}

func (s simpleSpeaker) Speak() string { return "speak" }

// 2. Getter / Setter / GetSetter
type Getter interface{ Get() int }
type Setter interface{ Set(int) }
type GetSetter interface {
	Getter
	Setter
}
type myGetSetter struct{ val int }

func (m *myGetSetter) Get() int  { return m.val }
func (m *myGetSetter) Set(v int) { m.val = v }

// 3. Animal / Dog
type Animal interface{ Eat() }
type Dog interface {
	Animal
	Bark()
}
type myDog struct{}

func (d myDog) Eat()  {}
func (d myDog) Bark() {}

type basicAnimal struct{}

func (a basicAnimal) Eat() {}

// 4. Runner
type Runner interface{ Run() }
type myRunner struct{}

func (r myRunner) Run() {}

// 5. Flyer
type Flyer interface{ Fly() }
type myFlyer struct{}

func (f myFlyer) Fly() {}

// 6. Printer
type Printer interface{ Print(...any) }
type myPrinter struct{}

func (p myPrinter) Print(a ...any) {}

// 7. Closer
type Closer interface{ Close() error }
type myCloser struct{}

func (c myCloser) Close() error { return nil }

// 8. Calculator
type Calculator interface{ Add(int, int) int }
type myCalc struct{}

func (c myCalc) Add(a, b int) int { return a + b }

// 9. Transformer
type Transformer interface{ Transform(string) string }
type myTransformer struct{}

func (t myTransformer) Transform(s string) string { return s + "!" }

// 10. Validator
type Validator interface{ Validate() bool }
type myValidator struct{}

func (v myValidator) Validate() bool { return true }

type simpleStringer struct{ val string }

func (s simpleStringer) String() string { return s.val }

func TestCast_FuncBroadToFmtStringer(t *testing.T) {
	// The user explicitly asked to try if func(BroadStringer) is convertible to func(fmt.Stringer)
	src := func(b BroadStringer) string {
		if b == nil {
			return "<nil>"
		}
		return b.String()
	}

	// This conversion succeeds at creation time because the argument check is done at runtime.
	out, err := django_reflect.CastFunc[func(fmt.Stringer) string](src)
	mustNoErr(t, err)

	// Calling with a concrete type that implements BroadStringer should work,
	// even if passed as an interface of fmt.Stringer, because the unwrapping mechanism
	// extracts the concrete type and checks if it implements BroadStringer.
	d := dummyBroadStringer{val: "hello"}
	var fs fmt.Stringer = d
	res := out(fs)
	if res != "hello" {
		t.Fatalf("expected 'hello', got %q", res)
	}

	// Calling with a concrete type that ONLY implements fmt.Stringer should panic at runtime
	// because it cannot satisfy the BroadStringer requirements of the original function.
	mustPanic(t, func() {
		out(simpleStringer{val: "fail"})
	}, "could not convert")
}

func TestCast_Interface1_LoudSpeaker(t *testing.T) {
	src := func(s Speaker) string { return s.Speak() }

	// Contravariant cast: from Speaker to LoudSpeaker
	out, err := django_reflect.CastFunc[func(LoudSpeaker) string](src)
	mustNoErr(t, err)

	res := out(myLoudSpeaker{})
	if res != "speak" {
		t.Fatalf("expected 'speak'")
	}
}

func TestCast_Interface2_LoudSpeaker_RuntimePanic(t *testing.T) {
	src := func(s LoudSpeaker) string { return s.Yell() }

	// Cast from LoudSpeaker to Speaker
	out, err := django_reflect.CastFunc[func(Speaker) string](src)
	mustNoErr(t, err)

	// Supplying actual LoudSpeaker works
	res := out(myLoudSpeaker{})
	if res != "YELL" {
		t.Fatalf("expected 'YELL'")
	}

	// Supplying simple Speaker panics
	mustPanic(t, func() {
		out(simpleSpeaker{})
	}, "could not convert")
}

func TestCast_Interface3_GetSetter(t *testing.T) {
	src := func(g Getter) int { return g.Get() }
	out, err := django_reflect.CastFunc[func(GetSetter) int](src)
	mustNoErr(t, err)

	gs := &myGetSetter{val: 42}
	if v := out(gs); v != 42 {
		t.Fatalf("expected 42, got %d", v)
	}
}

func TestCast_Interface4_DogToAnimal(t *testing.T) {
	src := func(d Dog) { d.Bark() }
	out, err := django_reflect.CastFunc[func(Animal)](src)
	mustNoErr(t, err)

	// Panics when animal doesn't bark
	mustPanic(t, func() {
		out(basicAnimal{})
	}, "could not convert")
}

func TestCast_Interface5_AnyToRunner(t *testing.T) {
	src := func(a any) {}
	out, err := django_reflect.CastFunc[func(Runner)](src)
	mustNoErr(t, err)
	out(myRunner{})
}

func TestCast_Interface6_FlyerToAny(t *testing.T) {
	src := func(f Flyer) {}
	out, err := django_reflect.CastFunc[func(any)](src)
	mustNoErr(t, err)

	// Calling with non-Flyer panics
	mustPanic(t, func() {
		out(123)
	}, "could not convert")
}

func TestCast_Interface7_PrinterVariadic(t *testing.T) {
	src := func(p Printer) {}
	out, err := django_reflect.CastFunc[func(any)](src)
	mustNoErr(t, err)
	out(myPrinter{})
}

func TestCast_Interface8_CloserReturnsError(t *testing.T) {
	src := func(c Closer) error { return c.Close() }
	out, err := django_reflect.CastFunc[func(any) error](src)
	mustNoErr(t, err)
	mustNoErr(t, out(myCloser{}))
}

func TestCast_Interface9_CalculatorReturnsInt(t *testing.T) {
	src := func(c Calculator) int { return c.Add(5, 5) }
	out, err := django_reflect.CastFunc[func(any) int](src)
	mustNoErr(t, err)
	if v := out(myCalc{}); v != 10 {
		t.Fatalf("expected 10")
	}
}

func TestCast_Interface10_Transformer(t *testing.T) {
	src := func(t Transformer) string { return t.Transform("go") }
	out, err := django_reflect.CastFunc[func(any) string](src)
	mustNoErr(t, err)
	if v := out(myTransformer{}); v != "go!" {
		t.Fatalf("expected go!")
	}
}
