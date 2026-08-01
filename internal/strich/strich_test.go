package strich

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/src/core/trans"
)

type strichTest struct {
	Delim          rune
	Haystack       string
	Needle         []string
	Check          func(chk Checker[string], haystack string, needle []string) bool
	ExpectedFound  int
	ExpectedDelims int
	ExpectedFunc   bool
	ExpectedLabel  string
	Display        CheckerDisplay[string]
}

func IS(c Checker[string], haystack string, needle []string) bool {
	return c.Is(haystack, needle...)
}

func CONTAINS(c Checker[string], haystack string, needle []string) bool {
	return c.IsAny(haystack, needle...)
}

var checkerDisplay = CheckerDisplay[string]{
	Delimiter: " / ",
	NotFoundLabel: func(ctx context.Context, s string) string {
		return "UNKNOWN"
	},
	Labels: map[string]func(context.Context) string{
		"SUCCESS": trans.S("label_SUCCESS"),
		"A":       trans.S("label_A"),
		"B":       trans.S("label_B"),
		"C":       trans.S("label_C"),
	},
}

var strichTests = []strichTest{
	{
		Delim:          '|',
		Haystack:       "SUCCESS",
		Needle:         []string{"SUCCESS"},
		ExpectedFound:  1,
		ExpectedDelims: 0,
	},
	{
		Delim:          '|',
		Haystack:       "SUCCESS|A|B|C",
		Needle:         []string{"SUCCESS", "C"},
		ExpectedFound:  2,
		ExpectedDelims: 3,
	},
	{
		Delim:          '|',
		Haystack:       "SUCCESS:A:B:C",
		Needle:         []string{"SUCCESS", "C"},
		ExpectedFound:  0,
		ExpectedDelims: 0,
	},
	{
		Delim:          ':',
		Haystack:       "SUCCESS:A:B:C",
		Needle:         []string{"B", "A", "SUCCESS", "C"},
		ExpectedFound:  4,
		ExpectedDelims: 3,
		Check:          IS,
		ExpectedFunc:   true,
	},
	{
		Delim:          ':',
		Haystack:       "SUCCESS:A:B:C",
		Needle:         []string{"B", "A", "SUCCESS", "C", "FAIL"},
		ExpectedFound:  4,
		ExpectedDelims: 3,
		Check:          IS,
		ExpectedFunc:   false,
	},
	{
		Delim:          ':',
		Haystack:       "SUCCESS:A:B:C",
		Needle:         []string{"B", "FAIL"},
		ExpectedFound:  1,
		ExpectedDelims: 3,
		Check:          CONTAINS,
		ExpectedFunc:   true,
	},
	{
		Delim:          ':',
		Haystack:       "SUCCESS:A:B:C",
		Needle:         []string{"FAIL"},
		ExpectedFound:  0,
		ExpectedDelims: 3,
		Check:          IS,
		ExpectedFunc:   false,
	},
	{
		Delim:          ':',
		Haystack:       "SUCCESS:A:B:C",
		Needle:         []string{"FAIL"},
		ExpectedFound:  0,
		ExpectedDelims: 3,
		Check:          CONTAINS,
		ExpectedFunc:   false,
	},

	// Labels (translations)
	{
		Delim:          ':',
		Haystack:       "SUCCESS:A:B:C",
		Needle:         []string{"B", "A", "SUCCESS", "C", "FAIL"},
		ExpectedFound:  4,
		ExpectedDelims: 3,
		Check:          IS,
		ExpectedFunc:   false,
		Display:        checkerDisplay,
		ExpectedLabel:  "label_B / label_A / label_SUCCESS / label_C / UNKNOWN",
	},
	{
		Delim:          ':',
		Haystack:       "SUCCESS:A:B:C",
		Needle:         []string{"B", "FAIL"},
		ExpectedFound:  1,
		ExpectedDelims: 3,
		Check:          CONTAINS,
		ExpectedFunc:   true,
		Display:        checkerDisplay,
		ExpectedLabel:  "label_B / UNKNOWN",
	},
	{
		Delim:          ':',
		Haystack:       "SUCCESS:A:B:C",
		Needle:         []string{"FAIL"},
		ExpectedFound:  0,
		ExpectedDelims: 3,
		Check:          IS,
		ExpectedFunc:   false,
		Display:        checkerDisplay,
		ExpectedLabel:  "UNKNOWN",
	},
	{
		Delim:          ':',
		Haystack:       "SUCCESS:A:B:C",
		Needle:         []string{"FAIL"},
		ExpectedFound:  0,
		ExpectedDelims: 3,
		Check:          CONTAINS,
		ExpectedFunc:   false,
		Display:        checkerDisplay,
		ExpectedLabel:  "UNKNOWN",
	},
}

func TestStrich(t *testing.T) {
	for _, tst := range strichTests {
		var extraLabel string
		if tst.Check != nil {
			extraLabel += "Func"
		}

		if tst.ExpectedLabel != "" {
			extraLabel += "Labels"
		}

		t.Run(fmt.Sprintf("Test%s({%s}-%d-%d)", extraLabel, tst.Haystack, tst.ExpectedFound, tst.ExpectedDelims), func(t *testing.T) {
			var chk = Checker[string]{
				Delimiter: tst.Delim,
				Display:   tst.Display,
			}

			found, delims := chk.contains(tst.Haystack, tst.Needle)

			t.Logf(
				"Is: %t\n\tIsAll: %t\n\tIsAny: %t",
				found == len(tst.Needle),
				found == len(tst.Needle),
				found > 0,
			)

			if found != tst.ExpectedFound {
				t.Errorf("found != expected (%d != %d) %q: %v", found, tst.ExpectedFound, tst.Haystack, tst.Needle)
			}
			if delims != tst.ExpectedDelims {
				t.Errorf("delims != expected (%d != %d) %q: %v", delims, tst.ExpectedDelims, tst.Haystack, tst.Needle)
			}

			if tst.Check != nil && tst.ExpectedFunc != tst.Check(chk, tst.Haystack, tst.Needle) {
				t.Errorf("Test check function did not yield %t", tst.ExpectedFunc)
			}

			if tst.ExpectedLabel != "" {
				var cmp string
				if cmp = chk.StringOf(t.Context(), strings.Join(tst.Needle, string(tst.Delim))); cmp != tst.ExpectedLabel {
					t.Errorf("Label does not match expected: %q != %q", cmp, tst.ExpectedLabel)
				}

				t.Logf("Label: %q", cmp)
			}
		})
	}
}
