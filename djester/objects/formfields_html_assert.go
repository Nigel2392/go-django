package objects

import (
	"fmt"

	"github.com/Nigel2392/go-django/djester"
	"github.com/PuerkitoBio/goquery"
)

func AssertInputPresent(fieldName string) djester.HTMLAssertFunc {
	return func(doc *goquery.Document) error {
		sel := doc.Find(fmt.Sprintf("#id_%s", fieldName))
		if sel.Length() != 1 {
			return fmt.Errorf("%d selections found, expected 1", sel.Length())
		}
		return nil
	}
}

type HTMLAttributeCheck struct {
	Attr  string
	Check func(value string, ok bool) (successfulTest bool)
}

func AttributePresent(attr string) HTMLAttributeCheck {
	return HTMLAttributeCheck{
		Attr: attr,
		Check: func(value string, ok bool) (successfulTest bool) {
			return value != "" && ok
		},
	}
}

func AttributeNotPresent(attr string) HTMLAttributeCheck {
	return HTMLAttributeCheck{
		Attr: attr,
		Check: func(value string, ok bool) (successfulTest bool) {
			return !ok
		},
	}
}

func AttributeValueEQ(attr, val string) HTMLAttributeCheck {
	return HTMLAttributeCheck{
		Attr: attr,
		Check: func(value string, ok bool) (successfulTest bool) {
			return ok && value == val
		},
	}
}

func AttributeValueNEQ(attr, val string) HTMLAttributeCheck {
	return HTMLAttributeCheck{
		Attr: attr,
		Check: func(value string, ok bool) (successfulTest bool) {
			return value != val
		},
	}
}

func AssertInputAttributes(fieldName string, checkAttrs ...HTMLAttributeCheck) djester.HTMLAssertFunc {
	if len(checkAttrs) == 0 {
		panic("checks must be provided to AssertInputAttributes")
	}

	return func(doc *goquery.Document) error {

		sel := doc.Find(fmt.Sprintf("#id_%s", fieldName))
		if sel.Length() != 1 {
			return fmt.Errorf("%d selections found, expected 1", sel.Length())
		}

		for _, a := range checkAttrs {
			v, ok := sel.Attr(a.Attr)
			if !a.Check(v, ok) {
				return fmt.Errorf("check for attribute %q was unsuccesful", a.Attr)
			}
		}

		return nil
	}
}

func AssertInputValueMatches(fieldName string, provided string) djester.HTMLAssertFunc {
	return func(doc *goquery.Document) error {
		sel := doc.Find(fmt.Sprintf("#id_%s", fieldName))
		if sel.Length() != 1 {
			return fmt.Errorf("%d selections found, expected 1", sel.Length())
		}

		value, ok := sel.Attr("value")
		if !ok {
			return fmt.Errorf("expected value in selection: %v", sel)
		}

		if value != provided {
			return fmt.Errorf("value %q does not match test value %q", value, provided)
		}

		return nil
	}
}
