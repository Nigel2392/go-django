//go:build test
// +build test

package djester

import (
	"fmt"
	"reflect"
	"testing"
)

type Assertion interface {
	Assert(predication bool, message string, args ...any)
	AssertEqual(expected, actual any)
	AssertNotEqual(expected, actual any)
	AssertNil(actual any)
	AssertNotNil(actual any)
	AssertContains(haystack, needle any)
	AssertNotContains(haystack, needle any)
}

type assertion struct {
	t       *testing.T
	verbose bool
}

func (d *assertion) Assert(predication bool, message string, args ...any) {
	d.t.Helper()
	if !predication {
		if len(args) > 0 {
			message = fmt.Sprintf(message, args...)
		}
		d.t.Fatalf("assertion failed: %s", message)
	}
}

func (d *assertion) AssertEqual(expected, actual any) {
	d.t.Helper()
	d.Assert(expected == actual, "expected %v, got %v", expected, actual)
}

func (d *assertion) AssertNotEqual(expected, actual any) {
	d.t.Helper()
	d.Assert(expected != actual, "expected not %v, got %v", expected, actual)
}

func (d *assertion) AssertNil(actual any) {
	d.t.Helper()
	d.Assert(actual == nil, "expected nil, got %v", actual)
}

func (d *assertion) AssertNotNil(actual any) {
	d.t.Helper()
	d.Assert(actual != nil, "expected not nil, got %v", actual)
}

func contains(sliceVal, itemVal reflect.Value) bool {
	if !sliceVal.IsValid() || sliceVal.Kind() != reflect.Slice {
		return false
	}

	for i := 0; i < sliceVal.Len(); i++ {
		elem := sliceVal.Index(i)
		if reflect.DeepEqual(elem.Interface(), itemVal.Interface()) {
			return true
		}
	}

	return false
}

func (d *assertion) AssertContains(haystack, needle any) {
	d.t.Helper()
	if haystack == nil || needle == nil {
		d.t.Fatalf("expected %v to contain %v, but haystack is nil", haystack, needle)
		return
	}

	var (
		sliceV = reflect.ValueOf(haystack)
		itemV  = reflect.ValueOf(needle)
	)

	if contains(sliceV, itemV) {
		return
	}

	if sliceV.Len() < 10 || d.verbose {
		d.t.Fatalf("expected %v to contain %v", haystack, needle)
	} else {
		d.t.Fatalf("expected slice of length %d to contain %v", sliceV.Len(), needle)
	}
}

func (d *assertion) AssertNotContains(haystack, needle any) {
	d.t.Helper()
	if haystack == nil || needle == nil {
		d.t.Fatalf("expected %v to not contain %v, but haystack is nil", haystack, needle)
		return
	}

	var (
		sliceV = reflect.ValueOf(haystack)
		itemV  = reflect.ValueOf(needle)
	)

	if !contains(sliceV, itemV) {
		return
	}

	if sliceV.Len() < 10 || d.verbose {
		d.t.Fatalf("expected %v to not contain %v", haystack, needle)
	} else {
		d.t.Fatalf("expected slice of length %d to not contain %v", sliceV.Len(), needle)
	}
}
