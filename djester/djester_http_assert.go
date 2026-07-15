//go:build test
// +build test

package djester

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type HTMLAssertFunc func(doc *goquery.Document) error

type ResponseAssertion interface {
	Assertion
	AssertHTML(asserts ...HTMLAssertFunc)
}

type responseAssertion struct {
	assertion[baseTB]
	response *TestResponse
}

func (r *responseAssertion) AssertHTML(asserts ...HTMLAssertFunc) {
	r.assertion.test.Helper()

	doc, err := r.response.DOM()
	if err != nil {
		r.assertion.test.Fatalf("failed to parse HTML DOM: %v", err)
	}

	r.assertion.AssertHTMLDoc(doc, asserts...)
}

// HasElement ensures at least one element matches the CSS selector.
func HasElement(selector string) HTMLAssertFunc {
	return func(doc *goquery.Document) error {
		if doc.Find(selector).Length() == 0 {
			return fmt.Errorf("expected to find element matching selector %q", selector)
		}
		return nil
	}
}

// DoesNotHaveElement ensures no elements match the CSS selector.
func DoesNotHaveElement(selector string) HTMLAssertFunc {
	return func(doc *goquery.Document) error {
		if count := doc.Find(selector).Length(); count > 0 {
			return fmt.Errorf("expected 0 elements matching %q, found %d", selector, count)
		}
		return nil
	}
}

// HasText ensures that at least one element matching the selector contains the expected text.
func HasText(selector, expectedText string) HTMLAssertFunc {
	return func(doc *goquery.Document) error {
		sel := doc.Find(selector)
		if sel.Length() == 0 {
			return fmt.Errorf("no element found for selector %q to check text", selector)
		}

		found := false
		sel.Each(func(i int, s *goquery.Selection) {
			if strings.Contains(s.Text(), expectedText) {
				found = true
			}
		})

		if !found {
			return fmt.Errorf("expected element %q to contain text %q", selector, expectedText)
		}
		return nil
	}
}

// HasAttribute ensures that at least one matching element has the specified attribute and value.
func HasAttribute(selector, attrName, expectedValue string) HTMLAssertFunc {
	return func(doc *goquery.Document) error {
		sel := doc.Find(selector)
		if sel.Length() == 0 {
			return fmt.Errorf("no element found for selector %q to check attribute", selector)
		}

		found := false
		sel.Each(func(i int, s *goquery.Selection) {
			if val, exists := s.Attr(attrName); exists && val == expectedValue {
				found = true
			}
		})

		if !found {
			return fmt.Errorf("expected element %q to have attribute %q=%q", selector, attrName, expectedValue)
		}
		return nil
	}
}

// HasElementCount ensures exactly N elements match the selector.
func HasElementCount(selector string, expectedCount int) HTMLAssertFunc {
	return func(doc *goquery.Document) error {
		actualCount := doc.Find(selector).Length()
		if actualCount != expectedCount {
			return fmt.Errorf("expected %d elements matching %q, but found %d", expectedCount, selector, actualCount)
		}
		return nil
	}
}
