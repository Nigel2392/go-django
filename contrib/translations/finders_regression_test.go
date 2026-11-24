package translations

import (
	"fmt"
	"slices"
	"strconv"
	"testing"
	"testing/fstest"

	"github.com/Nigel2392/go-django/src/core/trans"
)

func TestFindTemplatePluralsWithOtherCallsInAction(t *testing.T) {
	for _, tc := range []struct {
		name, action string
		want         [][2]string
	}{
		{"plural before another call", `Start {{ printf "%s %s" (P "item" "items" .Count) (T "in stock") }}`, [][2]string{{"item", "items"}, {"in stock", ""}}},
		{"two plural calls", `{{ printf "%s %s" (P "child" "children" .Count) (P "foot" "feet" .Count) }}`, [][2]string{{"child", "children"}, {"foot", "feet"}}},
		{"raw plural strings", "{{ P `clé` `clés` .Count }}", [][2]string{{"clé", "clés"}}},
		{"raw singular with quoted plural", "{{ P `clef` \"clefs\" .Count }}", [][2]string{{"clef", "clefs"}}},
		{"quoted singular with raw plural", "{{ P \"clef\" `clefs` .Count }}", [][2]string{{"clef", "clefs"}}},
		{"invalid plural leaves later call", `{{ printf "%s %s" (P .Singular .Plural .Count) (T "valid") }}`, [][2]string{{"valid", ""}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := fstest.MapFS{"fragment.html": {Data: []byte(tc.action)}}
			finder := templateTranslationsFinder{extensions: []string{"html"}}
			found, err := finder.Find(files, func(string) bool { return false })
			if err != nil {
				t.Fatal(err)
			}
			var got [][2]string
			for _, entry := range found {
				got = append(got, [2]string{entry.Text, entry.Plural})
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("extracted %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGoFinderRejectsNonLiteralArguments(t *testing.T) {
	for _, tc := range []struct {
		name, call string
	}{
		{"missing S argument", `trans.S()`},
		{"non-string S argument", `trans.S('a')`},
		{"computed S argument", `trans.S("one" + " two")`},
		{"missing T argument", `trans.T(ctx)`},
		{"dynamic T argument", `trans.T(ctx, title)`},
		{"missing P plural", `trans.P(ctx, "item")`},
		{"dynamic P singular", `trans.P(ctx, singular, "items", count)`},
		{"numeric P plural", `trans.P(ctx, "item", 2, count)`},
		{"missing SP count", `trans.SP("item", "items")`},
		{"dynamic SP singular", `trans.SP(singular, "items", count)`},
		{"dynamic SP plural", `trans.SP("item", plural, count)`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := fmt.Sprintf("package fixture\nimport %q\nfunc example() { %s; trans.S(\"valid\") }\n", trans.PACKAGE_PATH, tc.call)
			files := fstest.MapFS{"fixture.go": {Data: []byte(source)}}
			finder := goTranslationsFinder{}
			got, err := finder.Find(files, func(string) bool { return false })
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 || got[0].Text != "valid" || got[0].Line != 3 {
				t.Errorf("extracted %+v after %s, want only valid", got, tc.call)
			}
		})
	}
}

func FuzzGoTranslationLiteralExtraction(f *testing.F) {
	for _, pair := range [][2]string{
		{"file", "files"},
		{"", ""},
		{"élément 🏷️", "éléments"},
		{"line\nnext\t\"quoted\"", "literal \\ and " + "`"},
		{"\x00\u2028", "\xff\xfe"},
		{"{{ T \"inside\" }}", "{{ P \"inside\" \"forms\" }}"},
	} {
		f.Add(pair[0], pair[1])
	}
	f.Fuzz(func(t *testing.T, singular, plural string) {
		if len(singular)+len(plural) > 4096 {
			t.Skip()
		}
		quotedSingular, quotedPlural := strconv.Quote(singular), strconv.Quote(plural)
		source := fmt.Sprintf("package fixture\nimport %q\nfunc example() {\ntrans.S(%s)\ntrans.P(ctx, %s, %s, count)\n}\n",
			trans.PACKAGE_PATH, quotedSingular, quotedSingular, quotedPlural)
		files := fstest.MapFS{"fixture.go": {Data: []byte(source)}}
		finder := goTranslationsFinder{}
		got, err := finder.Find(files, func(string) bool { return false })
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Fatalf("extracted %d entries, want 2: %+v", len(got), got)
		}
		if got[0].Text != singular || got[0].Plural != "" || got[0].Line != 4 || got[0].Col != len("trans.S(")+1 {
			t.Errorf("singular extraction = %+v, want text %q at the S argument", got[0], singular)
		}
		if got[1].Text != singular || got[1].Plural != plural || got[1].Line != 5 || got[1].Col != len("trans.P(ctx, ")+1 {
			t.Errorf("plural extraction = %+v, want forms (%q, %q) at the P argument", got[1], singular, plural)
		}
	})
}
