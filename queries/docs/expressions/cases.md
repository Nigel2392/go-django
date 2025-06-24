# Case Expressions

Case expressions allow you to define conditional logic directly in your queries, much like SQL’s `CASE WHEN ... THEN ... ELSE ... END`.

They’re used for annotations, conditional selects, or building logic based on different field values.

## Creating a Case Expression

Use the `expr.Case()` function to start a new case expression.

You can then chain `.When(...).Then(...)` calls, and optionally `.Default(...)`.

### Basic Example

```go
qs := queries.GetQuerySet(&Todo{}).
    Annotate("Status",
        expr.Case().
            When("Done", true).Then("Completed").
            When(expr.Logical("Done").EQ(false), "Pending").
            Default("Unknown"),
    )
```

This will produce SQL like:

```sql
CASE
  WHEN table.done = ? THEN ?
  WHEN table.done = ? THEN ?
  ELSE ?
END
```

Using the global functions like `expr.When(...)` and `expr.Case(...)` also allows you to create more complex case expressions.

Really, this is more about preference - do as you like!

```go
qs := queries.GetQuerySet(&Todo{}).
    Annotate("TitleIndex",
        expr.Case(
            expr.When(
                expr.Q("Title__icontains", "NOT FOUND").
                    Or(expr.Q("Title", "CaseExpression1")),
            ).Then("ONE"),
            expr.When("Title", "CaseExpression2").Then("TWO"),
            expr.When("Title__iendswith", "3").Then("THREE"),
            "Unknown",
        ),
    )
```

### `.When(...)` Usage

The `When` function supports:

* A `string` field lookup + value(s)
* A `LogicalExpression` or `ClauseExpression`
* Any `Expression` + optional value

It returns a `*when` struct that can be chained with `.Then(...)`.

You can also call `Case(...cases)` with a list of `when` and default expressions.

## Function Reference

```go
func Case(cases ...any) *CaseExpression
func (c *CaseExpression) When(keyOrExpr interface{}, vals ...any) *CaseExpression
func (c *CaseExpression) Then(value any) *CaseExpression
func (c *CaseExpression) Default(value any) *CaseExpression
```

## Notes

* Each `When(...).Then(...)` pair adds a `WHEN` clause.
* Only one default is allowed.
* Calling `.Then(...)` without a matching `.When(...)` will panic.
