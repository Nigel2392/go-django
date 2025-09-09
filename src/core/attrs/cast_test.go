package attrs_test

import (
	"errors"
	"net/mail"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/src/core/attrs"
)

// --- helpers & custom types for interfaces ---

type strConv struct{ s string }

func (s strConv) ToString() string { return s.s }

type intConv struct{ v int64 }

func (i intConv) ToInt() int64 { return i.v }

type floatConv struct{ f float64 }

func (f floatConv) ToFloat() float64 { return f.f }

type boolConv struct{ b bool }

func (b boolConv) ToBool() bool { return b.b }

type timeConv struct{ t time.Time }

func (tc timeConv) ToTime() time.Time { return tc.t }

type hasTime struct{ t time.Time }

func (ht hasTime) Time() time.Time { return ht.t }

type myStringer struct{}

func (myStringer) String() string { return "stringer" }

// --- ToString tests ---

func TestToString_BasicsAndSpecials(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		if got := attrs.ToString(nil); got != "" {
			t.Fatalf("expected empty string, got %q", got)
		}
	})

	t.Run("string passthrough", func(t *testing.T) {
		if got := attrs.ToString("hello"); got != "hello" {
			t.Fatalf("expected %q, got %q", "hello", got)
		}
	})

	t.Run("fmt.Stringer", func(t *testing.T) {
		if got := attrs.ToString(myStringer{}); got != "stringer" {
			t.Fatalf("expected %q, got %q", "stringer", got)
		}
	})

	t.Run("error", func(t *testing.T) {
		err := errors.New("boom")
		if got := attrs.ToString(err); got != "boom" {
			t.Fatalf("expected %q, got %q", "boom", got)
		}
	})

	t.Run("mail.Address pointer", func(t *testing.T) {
		addr := &mail.Address{Address: "a@b.com"}
		if got := attrs.ToString(addr); got != "a@b.com" {
			t.Fatalf("expected %q, got %q", "a@b.com", got)
		}
	})

	t.Run("time.Time RFC3339", func(t *testing.T) {
		ti := time.Date(2023, 7, 1, 12, 34, 56, 0, time.UTC)
		want := ti.Format(time.RFC3339)
		if got := attrs.ToString(ti); got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("pointer to int", func(t *testing.T) {
		v := 42
		if got := attrs.ToString(&v); got != "42" {
			t.Fatalf("expected %q, got %q", "42", got)
		}
	})

	t.Run("slice mixed", func(t *testing.T) {
		in := []any{1, "a", true}
		got := attrs.ToString(in)
		if got != "1, a, true" {
			t.Fatalf("expected %q, got %q", "1, a, true", got)
		}
	})

	t.Run("array ints", func(t *testing.T) {
		in := [3]int{1, 2, 3}
		got := attrs.ToString(in)
		if got != "1, 2, 3" {
			t.Fatalf("expected %q, got %q", "1, 2, 3", got)
		}
	})

	t.Run("ToStringConverter", func(t *testing.T) {
		in := strConv{s: "custom"}
		got := attrs.ToString(in)
		if got != "custom" {
			t.Fatalf("expected %q, got %q", "custom", got)
		}
	})
}

// --- ToInt tests ---

func TestToInt(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		got, err := attrs.ToInt(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 0 {
			t.Fatalf("expected 0, got %d", got)
		}
	})

	t.Run("ToIntConverter", func(t *testing.T) {
		got, err := attrs.ToInt(intConv{v: 99})
		if err != nil || got != 99 {
			t.Fatalf("expected 99, got %d (err=%v)", got, err)
		}
	})

	t.Run("int kinds", func(t *testing.T) {
		got, err := attrs.ToInt(int64(123))
		if err != nil || got != 123 {
			t.Fatalf("expected 123, got %d (err=%v)", got, err)
		}
	})

	t.Run("uint kinds", func(t *testing.T) {
		got, err := attrs.ToInt(uint32(7))
		if err != nil || got != 7 {
			t.Fatalf("expected 7, got %d (err=%v)", got, err)
		}
	})

	t.Run("float kinds (truncate)", func(t *testing.T) {
		got, err := attrs.ToInt(3.7)
		if err != nil || got != 3 {
			t.Fatalf("expected 3, got %d (err=%v)", got, err)
		}
	})

	t.Run("bool true/false", func(t *testing.T) {
		got, err := attrs.ToInt(true)
		if err != nil || got != 1 {
			t.Fatalf("expected 1, got %d (err=%v)", got, err)
		}
		got, err = attrs.ToInt(false)
		if err != nil || got != 0 {
			t.Fatalf("expected 0, got %d (err=%v)", got, err)
		}
	})

	t.Run("string parse", func(t *testing.T) {
		got, err := attrs.ToInt("456")
		if err != nil || got != 456 {
			t.Fatalf("expected 456, got %d (err=%v)", got, err)
		}
	})

	t.Run("unsupported type error", func(t *testing.T) {
		type S struct{ A string }
		_, err := attrs.ToInt(S{A: "x"})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
}

// --- ToFloat tests ---

func TestToFloat(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		got, err := attrs.ToFloat(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 0 {
			t.Fatalf("expected 0, got %f", got)
		}
	})

	t.Run("ToFloatConverter", func(t *testing.T) {
		got, err := attrs.ToFloat(floatConv{f: 9.25})
		if err != nil || got != 9.25 {
			t.Fatalf("expected 9.25, got %f (err=%v)", got, err)
		}
	})

	t.Run("float kinds", func(t *testing.T) {
		got, err := attrs.ToFloat(float32(2.5))
		if err != nil || got != 2.5 {
			t.Fatalf("expected 2.5, got %f (err=%v)", got, err)
		}
	})

	t.Run("int & uint kinds", func(t *testing.T) {
		got, err := attrs.ToFloat(int64(5))
		if err != nil || got != 5 {
			t.Fatalf("expected 5, got %f (err=%v)", got, err)
		}
		got, err = attrs.ToFloat(uint(8))
		if err != nil || got != 8 {
			t.Fatalf("expected 8, got %f (err=%v)", got, err)
		}
	})

	t.Run("bool true/false", func(t *testing.T) {
		got, err := attrs.ToFloat(true)
		if err != nil || got != 1 {
			t.Fatalf("expected 1, got %f (err=%v)", got, err)
		}
		got, err = attrs.ToFloat(false)
		if err != nil || got != 0 {
			t.Fatalf("expected 0, got %f (err=%v)", got, err)
		}
	})

	t.Run("string parse", func(t *testing.T) {
		got, err := attrs.ToFloat("3.14159")
		if err != nil || got != 3.14159 {
			t.Fatalf("expected 3.14159, got %f (err=%v)", got, err)
		}
	})

	t.Run("unsupported type error", func(t *testing.T) {
		type S struct{ A string }
		_, err := attrs.ToFloat(S{A: "x"})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
}

// --- ToBool tests ---

func TestToBool(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		got, err := attrs.ToBool(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != false {
			t.Fatalf("expected false, got %v", got)
		}
	})

	t.Run("ToBoolConverter", func(t *testing.T) {
		got, err := attrs.ToBool(boolConv{b: true})
		if err != nil || got != true {
			t.Fatalf("expected true, got %v (err=%v)", got, err)
		}
	})

	t.Run("bool passthrough", func(t *testing.T) {
		got, err := attrs.ToBool(true)
		if err != nil || got != true {
			t.Fatalf("expected true, got %v (err=%v)", got, err)
		}
	})

	t.Run("int & uint kinds", func(t *testing.T) {
		got, err := attrs.ToBool(int(0))
		if err != nil || got != false {
			t.Fatalf("expected false, got %v (err=%v)", got, err)
		}
		got, err = attrs.ToBool(uint(2))
		if err != nil || got != true {
			t.Fatalf("expected true, got %v (err=%v)", got, err)
		}
	})

	t.Run("string parse true/false", func(t *testing.T) {
		got, err := attrs.ToBool("true")
		if err != nil || got != true {
			t.Fatalf("expected true, got %v (err=%v)", got, err)
		}
		got, err = attrs.ToBool("FALSE")
		if err != nil || got != false {
			t.Fatalf("expected false, got %v (err=%v)", got, err)
		}
	})

	t.Run("string parse invalid -> error", func(t *testing.T) {
		_, err := attrs.ToBool("notabool")
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
}

// --- ToTime tests ---

func TestToTime(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		got, err := attrs.ToTime(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got.IsZero() {
			t.Fatalf("expected zero time, got %v", got)
		}
	})

	t.Run("ToTimeConverter", func(t *testing.T) {
		want := time.Date(2024, 5, 10, 9, 8, 7, 0, time.UTC)
		got, err := attrs.ToTime(timeConv{t: want})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got.Equal(want) {
			t.Fatalf("expected %v, got %v", want, got)
		}
	})

	t.Run("Has Time() method", func(t *testing.T) {
		want := time.Date(2023, 1, 2, 3, 4, 5, 0, time.UTC)
		got, err := attrs.ToTime(hasTime{t: want})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got.Equal(want) {
			t.Fatalf("expected %v, got %v", want, got)
		}
	})

	t.Run("time.Time value passthrough", func(t *testing.T) {
		want := time.Now().UTC().Truncate(time.Second)
		got, err := attrs.ToTime(want)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got.Equal(want) {
			t.Fatalf("expected %v, got %v", want, got)
		}
	})

	t.Run("string RFC3339 parse", func(t *testing.T) {
		want := time.Date(2022, 11, 30, 15, 16, 17, 0, time.UTC)
		in := want.Format(time.RFC3339)
		got, err := attrs.ToTime(in)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got.Equal(want) {
			t.Fatalf("expected %v, got %v", want, got)
		}
	})

	t.Run("int unix seconds", func(t *testing.T) {
		sec := int64(1_690_000_000)
		want := time.Unix(sec, 0)
		got, err := attrs.ToTime(sec)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got.Equal(want) {
			t.Fatalf("expected %v, got %v", want, got)
		}
	})

	t.Run("unsupported type error", func(t *testing.T) {
		_, err := attrs.ToTime(3.14)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
}
