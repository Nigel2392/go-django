# Case Expressions

Case expressions let you build SQL `CASE WHEN ... THEN ... ELSE ... END` conditional logic directly inside queries, typically inside `Annotate`, `Select`, or compound expressions.

---

## Creating a Case Expression

You can construct a `*expr.CaseExpression` either by chaining methods on `expr.Case()` or by passing `expr.When(...)` clauses directly to `expr.Case(cases...)`.

### Chained Builder Style

```go
rows, err := queries.GetQuerySet(&Todo{}).
	Annotate("Status",
		expr.Case().
			When("Done", true).Then("Completed").
			When(expr.Logical("Priority").GTE(5)).Then("Urgent").
			Default("Pending"),
	).
	ValuesList("ID", "Title", "Status")
```

### Constructor Argument Style

```go
rows, err := queries.GetQuerySet(&Todo{}).
	Annotate("Tier",
		expr.Case(
			expr.When(
				expr.Q("Title__icontains", "CRITICAL").
					Or(expr.Q("Priority", 1)),
			).Then("High"),
			expr.When("Priority", 2).Then("Medium"),
			"Low", // trailing non-when value becomes the ELSE default
		),
	).
	ValuesList("ID", "Tier")
```

Both produce SQL of the form:

```sql
CASE WHEN ... THEN ? WHEN ... THEN ? ELSE ? END
```

---

## Signatures and `.When(...)` Inputs

```go
type WhenExpression = *when

func When(keyOrExpr interface{}, vals ...any) WhenExpression
func (w *when) Then(value any) WhenExpression

func Case(cases ...any) *CaseExpression
func (c *CaseExpression) When(keyOrExpr interface{}, vals ...any) *CaseExpression
func (c *CaseExpression) Then(value any) *CaseExpression
func (c *CaseExpression) Default(value any) *CaseExpression
```

`When(keyOrExpr, vals...)` accepts:

1. A `string` field lookup plus one or more values (`"Done", true` or `"Title__iendswith", "3"`), converted internally via `expr.Q(keyOrExpr, vals...)`. Passing zero values panics.
2. A `LogicalExpression` or `ClauseExpression`, with an optional single `then` value passed directly as `vals[0]` (`When(expr.Logical("Done").EQ(false), "Pending")`).
3. An `ExpressionBuilder` or any other `Expression`.

`Case(cases...)` accepts `*when` (`WhenExpression`), `[]*when`, an `Expression` / `ExpressionBuilder`, or any literal value wrapped via `expr.Value` to serve as the `ELSE` default.

---

## Notes and Caveats

- **In-place mutation:** Unlike `QuerySet` builder methods, `(*CaseExpression).When`, `Then`, and `Default` mutate the `*CaseExpression` receiver in place.
- **Panics after resolution:** Once a `*CaseExpression` has been resolved (`Resolve`), calling `When`, `Then`, or `Default` on it panics (`"CaseExpression was already used..."`). Calling `SQL()` before `Resolve()` also panics.
- **Avoid double `Then` calls:** Calling `Then(...)` on a `*CaseExpression` with no preceding `When(...)` panics. Calling `.Then(...)` after `When(logicalOrClauseExpr, thenVal)` (where `thenVal` was already provided as the second argument) also panics (`"then value is already set, cannot set it again"`).
- **Incomplete `When` clauses panic:** Calling `Clone()` or `SQL()` when any `When` clause is missing its condition or `Then` value panics.
- **Single default rule:** Passing more than one default expression to `Case(...)` panics; using `(*CaseExpression).Default(...)` replaces any previously set default before resolution.
