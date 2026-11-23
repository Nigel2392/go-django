# Translations

The `github.com/Nigel2392/go-django/src/core/trans` package defines Go-Django's core localization primitives: the `TranslationBackend` interface, string and plural translation helpers, lazy translatable strings, context locale propagation, localized date/time formatting, and template helpers.

For loading YAML translation catalogs, extracting translatable strings from templates/Go source, and HTTP `Accept-Language` / path middleware, see the [`contrib/translations` app documentation](./apps/translations.md).

---

## Backends and Core Types

```go
package trans

const PACKAGE_PATH = "github.com/Nigel2392/go-django/src/core/trans"

var (
	DefaultBackend              TranslationBackend = &SprintBackend{}
	TRANSLATIONS_DEFAULT_LOCALE                    = language.English
)

type (
	Locale        = string
	Translation   = string
	Untranslated  = string
	TranslateFunc func(ctx context.Context) string

	Lazy struct {
		Text string
		Args []any
	}

	Translatable any // string | func() string | func(context.Context) string | TranslateFunc | Lazy
)

type TranslationBackend interface {
	Translate(ctx context.Context, v Untranslated) Translation
	Translatef(ctx context.Context, v Untranslated, args ...any) Translation
	Pluralize(ctx context.Context, singular, plural Untranslated, n int) Translation
	Pluralizef(ctx context.Context, singular, plural Untranslated, n int, args ...any) Translation
	TimeFormat(ctx context.Context, short bool) Translation
}
```

By default, `DefaultBackend` is `&SprintBackend{}`, which passes strings through unchanged and formats plural rules as `n == 1 ? singular : plural`. Apps such as `contrib/translations` replace `DefaultBackend` during initialization.

---

## Translation Helpers

### Immediate Translation (`T`, `P`)

```go
func T(ctx context.Context, v Untranslated, args ...any) Translation
func P(ctx context.Context, singular, plural Untranslated, n any, args ...any) Translation
```

- **`T`**: Translates `v` using `DefaultBackend.Translate` (or `DefaultBackend.Translatef` when `args` are passed).
- **`P`**: Translates singular/plural strings using count `n`. `n` accepts signed/unsigned integers, slices/arrays/maps (measured by `len`), or numeric strings; any other type panics.

### Deferred and Lazy Translation (`S`, `SP`, `L`, `GetText`)

When declaring labels, help texts, or static titles at package initialization before a request context exists, use `S`, `SP`, or `L`:

```go
func S(v Untranslated, args ...any) TranslateFunc
func SP(singular, plural Untranslated, n any, args ...any) TranslateFunc

func L(text string, args ...any) Lazy
func (l Lazy) T(ctx context.Context) string
func (l Lazy) Deconstruct() (string, []any, map[string]any)

func GetTextFunc(v any) TranslateFunc
func GetText(ctx context.Context, v any) (string, bool)
```

- **`S` / `SP`**: Return a `TranslateFunc` (`func(context.Context) string`) evaluated when a context is available.
- **`L`**: Wraps a string and format arguments in a `Lazy` value that implements [`deconstruct.Deconstructible`](./deconstruct.md) under the path `"django.translations.Lazy"` (and registers its reconstructor in `deconstruct.Constructors` during `init()`).
- **`GetTextFunc` / `GetText`**: Normalize any supported translatable value (`string`, `func() string`, `func(context.Context) string`, `TranslateFunc`, or `Lazy`) into a `TranslateFunc` or translated string. Returns `("", false)` when `v == nil` and panics on unsupported types.

```go
var (
	Greeting = trans.S("Hello, %s!", "world")
	ItemLabel = trans.L("Items")
)

func handleRequest(ctx context.Context) {
	msg := trans.T(ctx, "Welcome back, %s", "Alice")
	countMsg := trans.P(ctx, "%d item", "%d items", 3, 3)
	lazyText, _ := trans.GetText(ctx, ItemLabel)
	_, _, _ = msg, countMsg, lazyText
}
```

---

## Context Locale Helpers

```go
func ContextWithLocale(ctx context.Context, locale language.Tag) context.Context
func LocaleFromContext(ctx context.Context) language.Tag
func DefaultLocale() language.Tag
```

- **`ContextWithLocale`**: Attaches a `golang.org/x/text/language.Tag` to `ctx`.
- **`LocaleFromContext`**: Extracts the attached `language.Tag` or returns `TRANSLATIONS_DEFAULT_LOCALE`.
- **`DefaultLocale`**: Returns `TRANSLATIONS_DEFAULT_LOCALE`.

---

## Time Formatting and Predefined Terms

```go
const (
	SHORT_TIME_FORMAT   = "SHORT_TIME_FORMAT"
	LONG_TIME_FORMAT    = "LONG_TIME_FORMAT"
	DEFAULT_TIME_FORMAT = "%Y-%m-%d %H:%M:%S"
)

func Time(ctx context.Context, t time.Time, format string) Translation
```

`trans.Time` formats `t` according to Python/Django-style `strftime` tokens and localizes weekday names, month names, and AM/PM markers via `ctx`. Passing `trans.SHORT_TIME_FORMAT` or `trans.LONG_TIME_FORMAT` asks `DefaultBackend.TimeFormat(ctx, short)` for the format string.

### Supported Format Specifiers

| Specifier | Output | Notes |
| :--- | :--- | :--- |
| `%a` / `%A` | Short / full weekday name | Uses `ShortMonday`..`ShortSunday` / `Monday`..`Sunday` |
| `%w` | Weekday number | `1` (Monday) through `7` (Sunday) |
| `%b` / `%B` | Short / full month name | Uses `ShortJanuary`..`ShortDecember` / `January`..`December` |
| `%m` / `%-m` | Month number | `01`-`12` / `1`-`12` |
| `%d` / `%-d` | Day of month | `01`-`31` / `1`-`31` |
| `%y` / `%-y` | Year (4-digit / 2-digit) | Same behavior as `%Y` / `%-Y` |
| `%Y` / `%-Y` | Year (4-digit / 2-digit) | e.g. `2025` / `25` |
| `%H` / `%-H` | Hour (24-hour) | See caveat below (padded vs. unpadded are inverted) |
| `%I` / `%-I` | Hour (12-hour) | `01`-`12` / `1`-`12` |
| `%M` / `%-M` | Minute | `00`-`59` / `0`-`59` |
| `%S` / `%-S` | Second | `00`-`59` / `0`-`59` |
| `%f` / `%-f` | Milliseconds | `000`-`999` / `0`-`999` |
| `%F` / `%-F` | Microseconds | `000000`-`999999` / `0`-`999999` |
| `%p` | AM / PM | Uses `AM` and `PM` |
| `%z` / `%Z` | Timezone offset / name | e.g. `+02:00` / `UTC` |
| `%j` | Day of year | `001`-`366` |
| `%U` / `%W` | ISO week number | `00`-`53` |
| `%%` | Literal `%` | Escaped percent sign |

### Predefined `TranslateFunc` Variables

The package also exports pre-built `TranslateFunc` values (`trans.S(...)`) for calendar and meridiem labels:

- Weekdays: `Monday`..`Sunday`, `ShortMonday`..`ShortSunday`
- Months: `January`..`December`, `ShortJanuary`..`ShortDecember`
- Meridiem: `AM`, `PM`, `AMPM`

---

## Template Functions

During initialization (`src/django.go`), Go-Django registers translation functions in the template engine (see [Rendering](./rendering.md)):

- **Global template function**:
  - `Translate(ctx, s, args...)`: Accepts a `context.Context`, `*http.Request`, `nil`, or a raw string as the first argument and delegates to `trans.T`.
- **Request-scoped template functions**:
  - `T(s, args...)`: Calls `trans.T(req.Context(), s, args...)`.
  - `P(single, plural, count, args...)`: Calls `trans.P(req.Context(), single, plural, count, args...)`.
  - `language()`: Calls `trans.LocaleFromContext(req.Context()).String()`.

```html
<html lang="{{ language }}">
<body>
  <h1>{{ T "Welcome, %s" .UserName }}</h1>
  <p>{{ P "%d item in cart" "%d items in cart" .Count .Count }}</p>
</body>
</html>
```

---

## Caveats

- **`trans.Time` panics on malformed formats**: Unknown specifiers, unsupported flags on a specifier, duplicate flags, or unexpected `%` characters cause `trans.Time` to panic.
- **`%H` and `%-H` are inverted in `trans.Time`**: In `src/core/trans/time.go`, `%-H` returns zero-padded `"%02d"` while `%H` returns unpadded `strconv.Itoa(hour)`. Test formatted outputs carefully if zero-padding hours matters.
- **`APPVAR_TRANSLATIONS_DEFAULT_LOCALE` type mismatch**: `src/django_checks.go` validates `django.APPVAR_TRANSLATIONS_DEFAULT_LOCALE` as `string`, whereas `src/django.go` reads it as `language.Tag` (`ConfigGet(a.Settings, APPVAR_TRANSLATIONS_DEFAULT_LOCALE, language.English)`). Setting a `string` fails `ConfigGet[language.Tag]` and keeps `language.English`, while setting a `language.Tag` triggers the `settings.matches.TRANSLATIONS_DEFAULT_LOCALE` check unless silenced with `checks.Shutup("settings.matches.TRANSLATIONS_DEFAULT_LOCALE", true)` (or you set `trans.TRANSLATIONS_DEFAULT_LOCALE` directly).
