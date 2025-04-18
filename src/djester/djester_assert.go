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
	if !predication {
		if len(args) > 0 {
			message = fmt.Sprintf(message, args...)
		}
		d.t.Fatalf("assertion failed: %s", message)
	}
}

func (d *assertion) AssertEqual(expected, actual any) {
	d.Assert(expected == actual, "expected %v, got %v", expected, actual)
}

func (d *assertion) AssertNotEqual(expected, actual any) {
	d.Assert(expected != actual, "expected not %v, got %v", expected, actual)
}

func (d *assertion) AssertNil(actual any) {
	d.Assert(actual == nil, "expected nil, got %v", actual)
}

func (d *assertion) AssertNotNil(actual any) {
	d.Assert(actual != nil, "expected not nil, got %v", actual)
}

func contains(slice reflect.Value, item reflect.Value) bool {
	if slice.IsNil() {
		return false
	}

	sliceV := reflect.ValueOf(slice)
	itemV := reflect.ValueOf(item)

	if sliceV.Kind() != reflect.Slice && sliceV.Kind() != reflect.Array {
		return false
	}

	for i := 0; i < sliceV.Len(); i++ {
		var s = sliceV.Index(i)
		if s.Kind() == reflect.Ptr {
			if s.IsNil() && !itemV.IsNil() {
				continue
			}

			s = s.Elem()
			var itemVal = itemV
			if itemVal.Kind() == reflect.Ptr {
				itemVal = itemVal.Elem()
			}

			if reflect.DeepEqual(s.Interface(), itemVal.Interface()) {
				return true
			}
		}
	}
	return false
}

func (d *assertion) AssertContains(haystack, needle any) {
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
