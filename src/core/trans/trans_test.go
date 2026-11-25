package trans_test

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/src/core/deconstruct"
	"github.com/Nigel2392/go-django/src/core/trans"
	"golang.org/x/text/language"
)

type coreTranslationCall struct {
	method   string
	ctx      context.Context
	singular string
	plural   string
	count    int
	args     []any
	locales  []language.Tag
}

type coreRecordingBackend struct {
	trans.SprintBackend
	calls    []coreTranslationCall
	accepted bool
}

func (b *coreRecordingBackend) Translate(ctx context.Context, text string) string {
	b.calls = append(b.calls, coreTranslationCall{method: "Translate", ctx: ctx, singular: text})
	return "plain translation"
}

func (b *coreRecordingBackend) Translatef(ctx context.Context, text string, args ...any) string {
	b.calls = append(b.calls, coreTranslationCall{method: "Translatef", ctx: ctx, singular: text, args: slices.Clone(args)})
	return "formatted translation"
}

func (b *coreRecordingBackend) Pluralize(ctx context.Context, singular, plural string, count int) string {
	b.calls = append(b.calls, coreTranslationCall{method: "Pluralize", ctx: ctx, singular: singular, plural: plural, count: count})
	return "plain pluralization"
}

func (b *coreRecordingBackend) Pluralizef(ctx context.Context, singular, plural string, count int, args ...any) string {
	b.calls = append(b.calls, coreTranslationCall{method: "Pluralizef", ctx: ctx, singular: singular, plural: plural, count: count, args: slices.Clone(args)})
	return "formatted pluralization"
}

func (b *coreRecordingBackend) IsAcceptedLocale(ctx context.Context, locales ...language.Tag) bool {
	b.calls = append(b.calls, coreTranslationCall{method: "IsAcceptedLocale", ctx: ctx, locales: slices.Clone(locales)})
	return b.accepted
}

func useCoreTranslationBackend(t *testing.T, backend trans.TranslationBackend) {
	t.Helper()
	previous := trans.DefaultBackend
	trans.DefaultBackend = backend
	t.Cleanup(func() { trans.DefaultBackend = previous })
}

func requireCoreCalls(t *testing.T, backend *coreRecordingBackend, want ...coreTranslationCall) {
	t.Helper()
	if !reflect.DeepEqual(backend.calls, want) {
		t.Errorf("backend calls = %+v, want %+v", backend.calls, want)
	}
}

func TestTranslationBackendDispatch(t *testing.T) {
	ctx := trans.ContextWithLocale(t.Context(), language.Dutch)
	for _, tc := range []struct {
		name string
		call func() string
		want string
		last coreTranslationCall
	}{
		{
			name: "unformatted",
			call: func() string { return trans.T(ctx, "Progress: %d%%") },
			want: "plain translation",
			last: coreTranslationCall{method: "Translate", ctx: ctx, singular: "Progress: %d%%"},
		},
		{
			name: "formatted",
			call: func() string { return trans.T(ctx, "%s: %d", "Mara", 12) },
			want: "formatted translation",
			last: coreTranslationCall{method: "Translatef", ctx: ctx, singular: "%s: %d", args: []any{"Mara", 12}},
		},
		{
			name: "one explicit nil argument still formats",
			call: func() string { return trans.T(ctx, "%v", nil) },
			want: "formatted translation",
			last: coreTranslationCall{method: "Translatef", ctx: ctx, singular: "%v", args: []any{nil}},
		},
		{
			name: "plural without formatting",
			call: func() string { return trans.P(ctx, "one file", "many files", "2") },
			want: "plain pluralization",
			last: coreTranslationCall{method: "Pluralize", ctx: ctx, singular: "one file", plural: "many files", count: 2},
		},
		{
			name: "plural count is separate from formatting arguments",
			call: func() string { return trans.P(ctx, "%s found %d file", "%s found %d files", []int{1, 2}, "Mara", 2) },
			want: "formatted pluralization",
			last: coreTranslationCall{method: "Pluralizef", ctx: ctx, singular: "%s found %d file", plural: "%s found %d files", count: 2, args: []any{"Mara", 2}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			backend := &coreRecordingBackend{}
			useCoreTranslationBackend(t, backend)
			if got := tc.call(); got != tc.want {
				t.Errorf("translation = %q, want %q", got, tc.want)
			}
			requireCoreCalls(t, backend, tc.last)
		})
	}
}

func TestDeferredTranslationUsesBackendAndContextAtCallTime(t *testing.T) {
	creation := &coreRecordingBackend{}
	useCoreTranslationBackend(t, creation)
	plain := trans.S("Title")
	formatted := trans.S("Hello, %s", "Mara")
	plural := trans.SP("record", "records", [2]string{"apple", "pear"})
	formattedPlural := trans.SP("%d record", "%d records", 2, 2)
	lazy := trans.L("Hello, %s", "Abel")
	requireCoreCalls(t, creation)

	dutch := trans.ContextWithLocale(t.Context(), language.Dutch)
	german := trans.ContextWithLocale(t.Context(), language.German)
	invocation := &coreRecordingBackend{}
	useCoreTranslationBackend(t, invocation)
	for _, tc := range []struct {
		name string
		ctx  context.Context
		fn   trans.TranslateFunc
		want string
		call coreTranslationCall
	}{
		{
			"singular", dutch, plain, "plain translation",
			coreTranslationCall{method: "Translate", ctx: dutch, singular: "Title"},
		},
		{
			"formatted singular", dutch, formatted, "formatted translation",
			coreTranslationCall{method: "Translatef", ctx: dutch, singular: "Hello, %s", args: []any{"Mara"}},
		},
		{
			"same closure with another context", german, formatted, "formatted translation",
			coreTranslationCall{method: "Translatef", ctx: german, singular: "Hello, %s", args: []any{"Mara"}},
		},
		{
			"plural", german, plural, "plain pluralization",
			coreTranslationCall{method: "Pluralize", ctx: german, singular: "record", plural: "records", count: 2},
		},
		{
			"formatted plural", dutch, formattedPlural, "formatted pluralization",
			coreTranslationCall{method: "Pluralizef", ctx: dutch, singular: "%d record", plural: "%d records", count: 2, args: []any{2}},
		},
		{
			"lazy value", german, lazy.T, "formatted translation",
			coreTranslationCall{method: "Translatef", ctx: german, singular: "Hello, %s", args: []any{"Abel"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			invocation.calls = nil
			if got := tc.fn(tc.ctx); got != tc.want {
				t.Errorf("translation = %q, want %q", got, tc.want)
			}
			requireCoreCalls(t, invocation, tc.call)
		})
	}
	requireCoreCalls(t, creation)
}

func TestPluralCountAcceptsNumbersAndCollectionLengths(t *testing.T) {
	type countInt int
	type countString string
	type countSlice []int
	ctx := t.Context()
	for _, tc := range []struct {
		name  string
		value any
		want  int
	}{
		{"int", -4, -4},
		{"int8", int8(-1), -1},
		{"int16", int16(0), 0},
		{"int32", int32(1), 1},
		{"int64", int64(123456789), 123456789},
		{"uint", uint(7), 7},
		{"uint8", uint8(8), 8},
		{"uint16", uint16(16), 16},
		{"uint32", uint32(32), 32},
		{"uint64", uint64(123456789), 123456789},
		{"named integer", countInt(-3), -3},
		{"array length", [3]int{43, 44, 45}, 3},
		{"empty array", [0]string{}, 0},
		{"slice length", []string{"first", "second"}, 2},
		{"named slice", countSlice{42}, 1},
		{"nil slice", []int(nil), 0},
		{"map length", map[string]int{"a": 50, "b": 25}, 2},
		{"nil map", map[string]int(nil), 0},
		{"numeric string", "012", 12},
		{"negative string", "-4", -4},
		{"signed named string", countString("+9"), 9},
	} {
		t.Run(tc.name, func(t *testing.T) {
			backend := &coreRecordingBackend{}
			useCoreTranslationBackend(t, backend)
			if got := trans.P(ctx, "file", "files", tc.value); got != "plain pluralization" {
				t.Errorf("P = %q, want backend result", got)
			}
			requireCoreCalls(t, backend, coreTranslationCall{method: "Pluralize", ctx: ctx, singular: "file", plural: "files", count: tc.want})
		})
	}
}

func TestPluralCountRejectsInvalidInputBeforeDispatch(t *testing.T) {
	count := 2
	for _, tc := range []struct {
		name  string
		value any
		want  string
	}{
		{"empty string", "", "failed to parse string"},
		{"nondecimal string", "0x12", "failed to parse string"},
		{"decimal fraction", "2.0", "failed to parse string"},
		{"overflows int64", "9223372036854775808", "value out of range"},
		{"float", 2.0, "unsupported type float64"},
		{"boolean", true, "unsupported type bool"},
		{"pointer", &count, "unsupported type ptr"},
		{"struct", struct{}{}, "unsupported type struct"},
		{"untyped nil", nil, "unsupported type invalid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			backend := &coreRecordingBackend{}
			useCoreTranslationBackend(t, backend)
			assertTranslationPanic(t, tc.want, func() {
				trans.P(t.Context(), "file", "files", tc.value)
			})
			requireCoreCalls(t, backend)
		})
	}
}

func TestGetTextResolvesLabelsAndPreservesContext(t *testing.T) {
	type contextKey struct{}
	ctx := context.WithValue(t.Context(), contextKey{}, "from context")
	useCoreTranslationBackend(t, &trans.SprintBackend{})
	contextText := func(c context.Context) string { return c.Value(contextKey{}).(string) }
	for _, tc := range []struct {
		name  string
		value any
		want  string
		ok    bool
	}{
		{"nil", nil, "", false},
		{"empty string", "", "", true},
		{"string", "A title", "A title", true},
		{"string with literal format directive", "Progress %d%%", "Progress %d%%", true},
		{"function", func() string { return "from callback" }, "from callback", true},
		{"context function", contextText, "from context", true},
		{"named translation function", trans.TranslateFunc(contextText), "from context", true},
		{"nil zero-argument function", (func() string)(nil), "", false},
		{"nil context function", (func(context.Context) string)(nil), "", false},
		{"nil named translation function", trans.TranslateFunc(nil), "", false},
		{"deferred singular function", trans.S("Welcome, %s", "Mara"), "Welcome, Mara", true},
		{"deferred plural function", trans.SP("%d file", "%d files", 2, 2), "2 files", true},
		{"lazy value", trans.L("%s %d", "Item", 7), "Item 7", true},
		{"zero lazy value", trans.Lazy{}, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, ok := trans.GetText(ctx, tc.value); got != tc.want || ok != tc.ok {
				t.Errorf("GetText = (%q, %t), want (%q, %t)", got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestGetTextFuncDefersCallbacks(t *testing.T) {
	calls := 0
	translate := trans.GetTextFunc(func() string {
		calls++
		return fmt.Sprintf("Label %d", calls)
	})
	if calls != 0 {
		t.Fatalf("creating translation function invoked its callback %d times", calls)
	}
	for _, want := range []string{"Label 1", "Label 2"} {
		if got := translate(t.Context()); got != want {
			t.Errorf("translation = %q, want %q", got, want)
		}
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}

func TestGetTextRejectsUnsupportedValues(t *testing.T) {
	for _, value := range []any{23, []string{"title"}, struct{}{}, func(int) string { return "" }} {
		t.Run(fmt.Sprintf("%T", value), func(t *testing.T) {
			assertTranslationPanic(t, fmt.Sprintf("unsupported type %T", value), func() {
				trans.GetText(t.Context(), value)
			})
		})
	}
}

func assertTranslationPanic(t *testing.T, want string, fn func()) {
	t.Helper()
	defer func() {
		t.Helper()
		if value := recover(); value == nil || !strings.Contains(fmt.Sprint(value), want) {
			t.Errorf("panic = %v, want a message containing %q", value, want)
		}
	}()
	fn()
}

func TestLazySerializationRoundTrip(t *testing.T) {
	useCoreTranslationBackend(t, &trans.SprintBackend{})
	for _, tc := range []struct {
		name  string
		value trans.Lazy
		want  string
	}{
		{"plain", trans.L("A label"), "A label"},
		{"empty", trans.L(""), ""},
		{"argument order and types", trans.L("%s: %.2f (%t)", "Total", 12.5, true), "Total: 12.50 (true)"},
		{"explicit nil argument", trans.L("%v", nil), "<nil>"},
		{"Unicode and quotes", trans.L("Cita: %q", "¡Hola, 🌎!"), "Cita: \"¡Hola, 🌎!\""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path, args, kwargs := tc.value.Deconstruct()
			if path != "django.translations.Lazy" || len(args) != 0 || kwargs["text"] != tc.value.Text || !reflect.DeepEqual(kwargs["args"], tc.value.Args) {
				t.Fatalf("Deconstruct = (%q, %+v, %+v), want Lazy constructor with the text and arguments", path, args, kwargs)
			}
			data, err := json.Marshal(tc.value, deconstruct.Constructors.JSONOptions())
			if err != nil {
				t.Fatal(err)
			}
			var decoded deconstruct.Deconstructible
			if err := deconstruct.Constructors.Unmarshal(data, &decoded); err != nil {
				t.Fatal(err)
			}
			lazy, ok := decoded.(trans.Lazy)
			if !ok || lazy.Text != tc.value.Text || !slices.Equal(lazy.Args, tc.value.Args) {
				t.Fatalf("decoded value = %#v, want %#v", decoded, tc.value)
			}
			if got := lazy.T(t.Context()); got != tc.want {
				t.Errorf("decoded translation = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSprintBackendFormattingAndPluralRules(t *testing.T) {
	backend := &trans.SprintBackend{}
	ctx := trans.ContextWithLocale(t.Context(), language.Japanese)
	for _, tc := range []struct {
		name string
		got  string
		want string
	}{
		{"unchanged percent directive", backend.Translate(ctx, "Progress %d%%"), "Progress %d%%"},
		{"unchanged text", backend.Translate(ctx, "こんにちは 🌎"), "こんにちは 🌎"},
		{"empty text", backend.Translate(ctx, ""), ""},
		{"formatting", backend.Translatef(ctx, "%s %04d %.2f", "Item", 7, 12.5), "Item 0007 12.50"},
		{"format literal percent", backend.Translatef(ctx, "%d%%", 100), "100%"},
		{"one", backend.Pluralize(ctx, "record", "records", 1), "record"},
		{"zero", backend.Pluralize(ctx, "record", "records", 0), "records"},
		{"negative count", backend.Pluralize(ctx, "record", "records", -1), "records"},
		{"two", backend.Pluralize(ctx, "record", "records", 2), "records"},
		{"formatted singular", backend.Pluralizef(ctx, "%s file", "%s files", 1, "One"), "One file"},
		{"formatted plural", backend.Pluralizef(ctx, "%s file", "%s files", 3, "Three"), "Three files"},
		{"default short time format", backend.TimeFormat(ctx, true), trans.DEFAULT_TIME_FORMAT},
		{"default long time format", backend.TimeFormat(ctx, false), trans.DEFAULT_TIME_FORMAT},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("result = %q, want %q", tc.got, tc.want)
			}
		})
	}
}

func TestTranslationAcceptanceDispatch(t *testing.T) {
	ctx := trans.ContextWithLocale(t.Context(), language.German)
	backend := &coreRecordingBackend{}
	useCoreTranslationBackend(t, backend)
	if trans.IsAcceptedLocale(ctx) {
		t.Error("IsAcceptedLocale without candidates returned true")
	}
	requireCoreCalls(t, backend)

	for _, accepted := range []bool{false, true} {
		t.Run(fmt.Sprint(accepted), func(t *testing.T) {
			backend.calls = nil
			backend.accepted = accepted
			if got := trans.IsAcceptedLocale(ctx, language.Japanese, language.German); got != accepted {
				t.Errorf("IsAcceptedLocale = %t, want backend decision %t", got, accepted)
			}
			requireCoreCalls(t, backend, coreTranslationCall{
				method: "IsAcceptedLocale", ctx: ctx, locales: []language.Tag{language.Japanese, language.German},
			})
		})
	}
	if !(&trans.SprintBackend{}).IsAcceptedLocale(ctx, language.Japanese) {
		t.Error("SprintBackend rejected a translation locale")
	}
}
