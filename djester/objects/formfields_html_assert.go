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

func valueError(value string, provided []string) error {
	return fmt.Errorf("value %q does not match test value %q", value, provided)
}

func AssertInputValueMatches(fieldName string, provided ...string) djester.HTMLAssertFunc {
	var ok bool
	var provMap = make(map[string]struct{})
	for _, p := range provided {
		provMap[p] = struct{}{}
	}

	return func(doc *goquery.Document) error {
		sel := doc.Find(fmt.Sprintf("#id_%s", fieldName))
		if sel.Length() != 1 {
			return fmt.Errorf("%d selections found, expected 1", sel.Length())
		}

		var value string
		switch {
		case sel.Is("select[multiple]"):
			var optSel = sel.Find("option")
			var selectedOpts = optSel.Filter("option[selected]")
			for _, node := range selectedOpts.EachIter() {
				value, ok = node.Attr("value")
				if !ok {
					h, err := node.Html()
					if err != nil {
						return err
					}
					return fmt.Errorf("value not found in node %s", h)
				}

				if _, ok := provMap[value]; !ok {
					return valueError(value, provided)
				}
			}

			return nil

		case sel.Is("select"):
			var optSel = sel.Find("option")
			var selectedOpt = optSel.Filter("option[selected]")
			value, ok = selectedOpt.Attr("value")

		case sel.Is("textarea"):
			var innerText = sel.Text()
			value = innerText
			ok = true

		default:
			value, ok = sel.Attr("value")
		}

		if !ok && len(provided) > 0 {
			h, _ := sel.Html()
			return fmt.Errorf("expected value %v in selection: %v", provided, h)
		}

		if _, ok := provMap[value]; !ok {
			return valueError(value, provided)
		}

		return nil
	}
}
