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
