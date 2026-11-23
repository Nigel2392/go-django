# Lookups and Transforms

Lookups define how a field (and any optional transforms applied to it) is compared against filter values inside `QuerySet.Filter`, `expr.Q`, `expr.Expr`, and `expr.Express`.

---

## Lookup Syntax

Append transforms and a lookup name to a field path using double underscores (`__`):

```
FieldPath[__transform...][__lookup]
```

- Use dot notation (`.`) to traverse related models, and double underscores (`__`) only for transforms and the trailing lookup (`"Author.Name__lower__startswith"`). See [Relations](../relations/relations.md).
- When no trailing lookup is recognized, `exact` is assumed (`"Title"` is equivalent to `"Title__exact"`, and `"Title__lower"` applies the `lower` transform with the `exact` lookup).

```go
qs := queries.GetQuerySet(&Article{}).
	Filter("Author.Name__lower__startswith", "alice").
	Filter("Status__in", []string{"published", "featured"}).
	Filter(expr.Q("PublishedAt__isnull", false).Or(expr.Q("Priority__gte", 5)))
```

---

## Built-In Lookups

All constants below are defined in `github.com/Nigel2392/go-django/queries/src/expr`:

| Constant | Name | Arguments | Description |
| :--- | :--- | :--- | :--- |
| `LOOKUP_EXACT` / `DEFAULT_LOOKUP` | `"exact"` | 1 | `=`; normalizes `attrs.Definer` values to their primary key |
| `LOOKUP_NOT` | `"not"` | 1 | `!=`; normalizes `attrs.Definer` values to their primary key |
| `LOOKUP_GT` | `"gt"` | 1 | `>` |
| `LOOKUP_LT` | `"lt"` | 1 | `<` |
| `LOOKUP_GTE` | `"gte"` | 1 | `>=` |
| `LOOKUP_LTE` | `"lte"` | 1 | `<=` |
| `LOOKUP_BITAND` | `"bitand"` | 1 | Bitwise `&` |
| `LOOKUP_BITOR` | `"bitor"` | 1 | Bitwise `\|` |
| `LOOKUP_BITXOR` | `"bitxor"` | 1 | Bitwise `^` |
| `LOOKUP_BITLSH` | `"bitlsh"` | 1 | Bitwise `<<` |
| `LOOKUP_BITRSH` | `"bitrsh"` | 1 | Bitwise `>>` |
| `LOOKUP_BITNOT` | `"bitnot"` | 1 | Bitwise `~` |
| `LOOKUP_IEXACT` | `"iexact"` | 1 (`string` or `Expression`) | Case-insensitive equality |
| `LOOKUP_CONTAINS` | `"contains"` | 1 (`string` or `Expression`) | Case-sensitive substring match (`LIKE '%...%'`) |
| `LOOKUP_ICONTANS` | `"icontains"` | 1 (`string` or `Expression`) | Case-insensitive substring match |
| `LOOKUP_STARTSWITH` | `"startswith"` | 1 (`string` or `Expression`) | Case-sensitive prefix match (`LIKE '...%'`) |
| `LOOKUP_ISTARTSWITH` | `"istartswith"` | 1 (`string` or `Expression`) | Case-insensitive prefix match |
| `LOOKUP_ENDSWITH` | `"endswith"` | 1 (`string` or `Expression`) | Case-sensitive suffix match (`LIKE '%...'`) |
| `LOOKUP_IENDSWITH` | `"iendswith"` | 1 (`string` or `Expression`) | Case-insensitive suffix match |
| `LOOKUP_IN` | `"in"` | 1+ (or slice / `Expression`) | `IN (...)`; flattens nested slices/arrays and supports subqueries |
| `LOOKUP_ISNULL` | `"isnull"` | 0 or 1 | `IS NULL` (default `true`) or `IS NOT NULL` (`false`) |

*(Note: the constant name in package `expr` is spelled `LOOKUP_ICONTANS`, whose string value is `"icontains"`.)*

---

## Built-In Transforms

Transforms wrap the left-hand expression before the lookup runs, and multiple transforms can be chained (`"Title__lower__length__gt"`):

| Transform | Underlying Expression |
| :--- | :--- |
| `lower` | `expr.LOWER(...)` |
| `upper` | `expr.UPPER(...)` |
| `length` | `expr.LENGTH(...)` |
| `count` | `expr.COUNT(...)` |
| `avg` | `expr.AVG(...)` |
| `sum` | `expr.SUM(...)` |
| `min` | `expr.MIN(...)` |
| `max` | `expr.MAX(...)` |

```go
// Filters rows where LENGTH(LOWER(title)) > 10
longTitles, err := queries.GetQuerySet(&Article{}).
	Filter("Title__lower__length__gt", 10).
	All()
```

---

## Custom Lookups and Transforms

You can register custom lookups and transforms globally or restricted to specific `driver.Driver` types:

```go
type Lookup interface {
	Drivers() []driver.Driver
	Name() string
	Arity() (min, max int)
	NormalizeArgs(inf *ExpressionInfo, value []any) ([]any, error)
	Resolve(inf *ExpressionInfo, lhsResolved ResolvedExpression, args []any) LookupExpression
}

type LookupTransform interface {
	Drivers() []driver.Driver
	Name() string
	Resolve(inf *ExpressionInfo, lhsResolved ResolvedExpression) (ResolvedExpression, error)
}

func RegisterLookup(Lookup Lookup)
func RegisterTransforms(transforms ...LookupTransform)
func GetLookup(inf *ExpressionInfo, lookupName string, lhs any, args []any) (func(sb builder.Builder), error)
```

Convenience base structs and error sentinels:

- `expr.BaseLookup`, `expr.LogicalLookup`, `expr.PatternLookup`, `expr.InLookup`, `expr.IsNullLookup`, `expr.RangeLookup`
- `expr.BaseTransform`
- `expr.ErrLookupNotFound`, `expr.ErrTransformNotFound`, `expr.ErrLookupArgsInvalid`
- `expr.LookupField`: optional `attrs.FieldDefinition` interface (`AllowedTransforms() []string`, `AllowedLookups() []string`) exposed via `ResolvedField`

```go
expr.RegisterLookup(&expr.RangeLookup{
	BaseLookup: expr.BaseLookup{
		Identifier: expr.LOOKUP_RANGE,
	},
})
```

---

## Caveats

- **`range` is not registered by default:** `LOOKUP_RANGE` (`"range"`) and `RangeLookup` exist in `expr`, but `"range"` is not registered in `init()`. Register `&expr.RangeLookup{BaseLookup: expr.BaseLookup{Identifier: expr.LOOKUP_RANGE}}` at startup if you want `Field__range` (`BETWEEN ? AND ?`).
- **Unimplemented lookups:** `regex`, `iregex`, and arithmetic lookup names (`add`, `sub`, `mul`, `div`, `mod`) are not registered. Use [`expr.Logical`](./expressions.md#logical-expressions-and-operators) for arithmetic and `expr.Raw` / `expr.F` for driver-specific regex operators.
- **Empty slice with `in`:** Passing an empty slice (`Filter("ID__in", []int{})`) flattens to zero items and renders invalid SQL `IN ()`; guard against empty slices before filtering.
- **`LookupField` metadata only:** `LookupField.AllowedTransforms()` and `AllowedLookups()` populate `ResolvedField` metadata, but the lookup registry does not currently reject disallowed lookups at runtime.
