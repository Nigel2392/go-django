package translations

import (
	"context"
	"fmt"
	"io/fs"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/elliotchance/orderedmap/v2"
)

type finderModel struct {
	Title    string
	Internal string
}

func (m *finderModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		attrs.NewField(m, "Title", &attrs.Config{Label: "Display title"}),
		attrs.NewField(m, "Internal", &attrs.Config{Label: "Internal field"}),
	)
}

func TestModelTranslationsFinder(t *testing.T) {
	setTestDjango(t, django.Config(nil))
	finder := &godjangoModelsFinder{}
	includeAll := func(string) bool { return false }
	if got, err := finder.Find(nil, includeAll); err != nil || len(got) != 0 {
		t.Fatalf("Find without apps = (%v, %v), want no matches", got, err)
	}

	django.Global.Apps = orderedmap.NewOrderedMap[string, django.AppConfig]()
	for _, name := range []string{"pages", "private"} {
		app := apps.NewAppConfig(name)
		app.ModelObjects = []attrs.Definer{&finderModel{}}
		django.Global.Apps.Set(name, app)
	}
	got, err := finder.Find(nil, func(path string) bool {
		return strings.HasPrefix(path, ".models/private/") || strings.HasSuffix(path, "/Internal")
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []Translation{
		{Path: ".models/pages/finderModel", Line: 1, Col: 1, Text: "finderModel", Comment: "[ModelFinder]:\ttranslations.finderModel"},
		{Path: ".models/pages/finderModel/fields/Title", Line: 1, Col: 0, Text: "Display title", Comment: "[ModelFinder.Field]:\ttranslations.finderModel.Title"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("model matches = %+v, want %+v", got, want)
	}
}

func TestTemplateTranslationsFinder(t *testing.T) {
	const pages = `Intro
{{ T "Hello\nworld" }}
{{ P "file" "files" .Count }}
{{ Translate .Context "Open" }}
{{ Translate "Close" }}
{{ if .Ready }}{{ T ` + "`Ready now`" + ` }}{{ end }}
{{ T .Variable }}{{ T "" }}
{{ T "Keep going" }}`
	files := fstest.MapFS{
		"templates/pages.html": {Data: []byte(pages)},
		"templates/skip.html":  {Data: []byte(`{{ T "Excluded" }}`)},
		"templates/wrong.md":   {Data: []byte(`{{ T "Unsupported extension" }}`)},
		"templates/notice.txt": {Data: []byte(`{{ T "Notice" }}`)},
	}
	finder := templateTranslationsFinder{extensions: []string{"html", "txt"}}
	got, err := finder.Find(files, func(path string) bool { return path == "templates/skip.html" })
	if err != nil {
		t.Fatal(err)
	}
	want := []Translation{
		{Path: "templates/notice.txt", Line: 1, Text: "Notice"},
		{Path: "templates/pages.html", Line: 2, Text: "Hello\nworld"},
		{Path: "templates/pages.html", Line: 3, Text: "file", Plural: "files"},
		{Path: "templates/pages.html", Line: 4, Text: "Open"},
		{Path: "templates/pages.html", Line: 5, Text: "Close"},
		{Path: "templates/pages.html", Line: 6, Text: "Ready now"},
		{Path: "templates/pages.html", Line: 8, Text: "Keep going"},
	}
	if len(got) != len(want) {
		t.Fatalf("found %d messages, want %d: %+v", len(got), len(want), got)
	}
	for i, entry := range got {
		if entry.Col < 1 || !strings.HasPrefix(entry.Comment, "[TemplateFinder]:\t{{") {
			t.Errorf("match %d has invalid position or comment: %+v", i, entry)
		}
		entry.Col, entry.Comment = 0, ""
		if !reflect.DeepEqual(entry, want[i]) {
			t.Errorf("match %d = %+v, want %+v", i, entry, want[i])
		}
	}
}

func TestParseGoTemplateTCalls(t *testing.T) {
	const source = `Intro {{ T "First" | printf "%s" }} {{ Translate .Context "Second" }}`
	seen := 0
	for _, match := range parseGoTemplateTCalls(source, templateTranslationMatchers) {
		seen++
		if match.singular != "First" || match.plural != "" || match.col != strings.Index(source, "T ")+1 {
			t.Errorf("first match = %+v", match)
		}
		break
	}
	if seen != 1 {
		t.Errorf("iterator returned %d matches before stopping, want 1", seen)
	}
}

func TestGoTranslationsFinder(t *testing.T) {
	withImport := func(source string) []byte {
		return []byte(fmt.Sprintf(source, trans.PACKAGE_PATH))
	}
	files := fstest.MapFS{
		"views/default.go": {Data: withImport(`package views
import %q
type page struct{}
func (*page) Render() {
	trans.S("Hello\nworld")
	trans.T(ctx, "Open")
	trans.P(ctx, "file", "files", count)
	trans.SP("child", "children", count)
	trans.S(dynamic)
	trans.S(42)
	trans.P(ctx, "ignored", dynamic, count)
	trans.T()
	other.S("wrong package")
	objects.Translator.S("not an identifier")
}`)},
		"views/aliased.go": {Data: withImport(`package views
import i18n %q
func Render() { i18n.S("Aliased") }`)},
		"views/dotted.go": {Data: withImport(`package views
import . %q
func Render() { S("Dot import") }`)},
		"views/custom.go": {Data: []byte(`package views
func Render() { localize.S("Configured alias") }`)},
		"views/skip_test.go": {Data: withImport(`package views
import %q
func Render() { trans.S("Ignored test") }`)},
		"views/excluded.go": {Data: withImport(`package views
import %q
func Render() { trans.S("Excluded") }`)},
		"views/broken.go":  {Data: []byte("package views\nfunc Broken(")},
		"views/readme.txt": {Data: []byte("trans.S(\"Not Go\")")},
	}
	finder := goTranslationsFinder{packageAliases: []string{"localize"}}
	got, err := finder.Find(files, func(path string) bool { return path == "views/excluded.go" })
	if err != nil {
		t.Fatal(err)
	}

	want := []Translation{
		{Path: "views/aliased.go", Line: 3, Text: "Aliased", Comment: "[GoFileFinder]:\tRender"},
		{Path: "views/custom.go", Line: 2, Text: "Configured alias", Comment: "[GoFileFinder]:\tRender"},
		{Path: "views/default.go", Line: 5, Text: "Hello\nworld", Comment: "[GoFileFinder]:\t*page.Render"},
		{Path: "views/default.go", Line: 6, Text: "Open", Comment: "[GoFileFinder]:\t*page.Render"},
		{Path: "views/default.go", Line: 7, Text: "file", Plural: "files", Comment: "[GoFileFinder]:\t*page.Render"},
		{Path: "views/default.go", Line: 8, Text: "child", Plural: "children", Comment: "[GoFileFinder]:\t*page.Render"},
		{Path: "views/dotted.go", Line: 3, Text: "Dot import", Comment: "[GoFileFinder]:\tRender"},
	}
	if len(got) != len(want) {
		t.Fatalf("found %d messages, want %d: %+v", len(got), len(want), got)
	}
	for i, entry := range got {
		if entry.Col < 1 {
			t.Errorf("match %d has invalid column: %+v", i, entry)
		}
		entry.Col = 0
		if !reflect.DeepEqual(entry, want[i]) {
			t.Errorf("match %d = %+v, want %+v", i, entry, want[i])
		}
	}
}

type unreadableSourceFS struct {
	fstest.MapFS
}

func (f unreadableSourceFS) ReadFile(name string) ([]byte, error) {
	return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrPermission}
}

func TestSourceFindersReportReadErrors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		path   string
		finder Finder
	}{
		{"template", "blocked.html", &templateTranslationsFinder{extensions: []string{"html"}}},
		{"go", "blocked.go", &goTranslationsFinder{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fsys := unreadableSourceFS{fstest.MapFS{tc.path: {}}}
			got, err := tc.finder.Find(fsys, func(string) bool { return false })
			if len(got) != 0 || err == nil || !strings.Contains(err.Error(), tc.path) || !strings.Contains(err.Error(), fs.ErrPermission.Error()) {
				t.Errorf("Find = (%v, %v), want read error for %s", got, err, tc.path)
			}
		})
	}
}
