//go:build test
// +build test

package djester

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

type Assertion interface {
	Assert(predication bool, message string, args ...any)
	AssertEqual(expected, actual any)
	AssertNotEqual(expected, actual any)
	AssertNil(actual any)
	AssertNotNil(actual any)
	AssertContains(haystack, needle any)
	AssertNotContains(haystack, needle any)

	AssertHTMLString(document string, asserts ...HTMLAssertFunc)
	AssertHTMLDoc(doc *goquery.Document, asserts ...HTMLAssertFunc)
}

type assertion[GOTEST BaseTB] struct {
	test    GOTEST
	verbose bool
}

func Asserter[GOTEST BaseTB](t GOTEST, verbose bool) Assertion {
	return &assertion[GOTEST]{test: t, verbose: verbose || testing.Verbose()}
}

func (d *assertion[TEST]) Assert(predication bool, message string, args ...any) {
	d.test.Helper()
	if !predication {
		if len(args) > 0 {
			message = fmt.Sprintf(message, args...)
		}
		d.test.Fatalf("assertion failed: %s", message)
	}
}

func (d *assertion[TEST]) AssertEqual(expected, actual any) {
	d.test.Helper()
	d.Assert(expected == actual, "expected %v, got %v", expected, actual)
}

func (d *assertion[TEST]) AssertNotEqual(expected, actual any) {
	d.test.Helper()
	d.Assert(expected != actual, "expected not %v, got %v", expected, actual)
}

func (d *assertion[TEST]) AssertNil(actual any) {
	d.test.Helper()
	d.Assert(actual == nil, "expected nil, got %v", actual)
}

func (d *assertion[TEST]) AssertNotNil(actual any) {
	d.test.Helper()
	d.Assert(actual != nil, "expected not nil, got %v", actual)
}

func (r *assertion[TEST]) AssertHTMLString(document string, asserts ...HTMLAssertFunc) {
	r.test.Helper()

	html, err := goquery.NewDocumentFromReader(strings.NewReader(document))
	if err != nil {
		r.test.Fatalf("Failed to parse document: %v", err)
	}

	r.AssertHTMLDoc(html, asserts...)
}

func (r *assertion[TEST]) AssertHTMLDoc(doc *goquery.Document, asserts ...HTMLAssertFunc) {
	r.test.Helper()

	for _, assertFn := range asserts {
		if err := assertFn(doc); err != nil {
			if r.verbose {
				var h, _ = doc.Html()
				r.test.Fatalf("HTML assertion failed: %v:\n%s", err, h)
			} else {
				r.test.Fatalf("HTML assertion failed: %v", err)
			}
		}
	}
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

func (d *assertion[TEST]) AssertContains(haystack, needle any) {
	d.test.Helper()
	if haystack == nil || needle == nil {
		d.test.Fatalf("expected %v to contain %v, but haystack is nil", haystack, needle)
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
		d.test.Fatalf("expected %v to contain %v", haystack, needle)
	} else {
		d.test.Fatalf("expected slice of length %d to contain %v", sliceV.Len(), needle)
	}
}

func (d *assertion[TEST]) AssertNotContains(haystack, needle any) {
	d.test.Helper()
	if haystack == nil || needle == nil {
		d.test.Fatalf("expected %v to not contain %v, but haystack is nil", haystack, needle)
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
		d.test.Fatalf("expected %v to not contain %v", haystack, needle)
	} else {
		d.test.Fatalf("expected slice of length %d to not contain %v", sliceV.Len(), needle)
	}
}
