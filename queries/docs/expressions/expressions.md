# Expressions

Expressions allow you to build SQL clauses, arithmetic operations, type casts, database function calls, subqueries, and raw parameterized fragments directly inside queries. They live in `github.com/Nigel2392/go-django/queries/src/expr` and write SQL and bound parameters into a `builder.Builder` from `github.com/Nigel2392/go-django/queries/src/expr/builder`.

See the companion expression guides for specific expression families:

- [Lookups and Transforms](./lookups.md)
- [Case Expressions](./cases.md)
- [Constraints](./constraints.md)

---

## Core Interfaces

```go
package builder

type Builder interface {
	io.Writer
	WriteString(string) error
	WriteRune(rune) error
	AddVar(...interface{})
	AddError(...error)
}
```

```go
package expr

type ExpressionBuilder interface {
	BuildExpression() Expression
}

type ResolvedExpression interface {
	SQL(builder.Builder)
}

type Expression interface {
	ResolvedExpression
	Clone() Expression
	Resolve(inf *ExpressionInfo) Expression
}

type NamedExpression interface {
	Expression
	FieldName() string
}

type AnnotationExpression interface {
	NamedExpression
	Annotate() bool
}
```

- `Expression.Resolve(inf *ExpressionInfo) Expression` binds field paths, driver dialects, table aliases, and placeholders onto a cloned copy of the expression before `SQL(sb)` is called.
- `QuerySet` calls `Resolve` automatically during query compilation; you only need to call `Resolve` manually when writing custom composite expressions that wrap child expressions.
- `NamedExpression` associates a result or target field name with an expression (`As`, `Field`, `OuterRef`, `Chain`, `Logical`, `F`, `Cast`, and SQL functions all implement `NamedExpression`).
- `AnnotationExpression` marks expressions that represent named query annotations (`Annotate() bool`). See [Interfaces](../interfaces.md) for `VirtualField`, `AliasField`, `ExpressionInfo`, and `FieldResolver`.

---

## Clause Expressions

`ClauseExpression` represents boolean predicates used in `WHERE` and `HAVING` clauses and combined with `AND`, `OR`, or `NOT`.

```go
type ExprOp string

const (
	OpAnd ExprOp = "AND"
	OpOr  ExprOp = "OR"
)

type ClauseExpression interface {
	Expression
	IsNot() bool
	Not(b bool) ClauseExpression
	And(...Expression) ClauseExpression
	Or(...Expression) ClauseExpression
}
```

Common constructors:

- `expr.Q(fieldLookup string, value ...any) *ExprNode`: splits `fieldLookup` on `"__"` (defaulting to `"exact"`) and creates a lookup node.
- `expr.Expr(field any, operation string, value ...any) *ExprNode`: builds a lookup node for an explicit field name or `NamedExpression` and lookup operation name.
- `expr.And[T any](exprs ...T) *ExprGroup`: groups expressions with `AND`.
- `expr.Or[T any](exprs ...T) *ExprGroup`: groups expressions with `OR`.
- `expr.Express(key any, vals ...any) []ClauseExpression`: converts a `"field__lookup"` string + values, an `Expression` / `[]Expression` / `[]ClauseExpression`, or a `map[string]any` into `[]ClauseExpression`.

```go
qs := queries.GetQuerySet(&Todo{}).
	Filter(
		expr.Q("Done", false).
			And(expr.Q("Priority__gte", 2)).
			Or(expr.Q("Title__icontains", "urgent")),
	)
```

---

## Logical Expressions and Operators

`LogicalOp` and `LogicalExpression` let you chain SQL comparisons, arithmetic, and bitwise operations programmatically.

```go
type LogicalOp string

const (
	EQ  LogicalOp = "="
	NE  LogicalOp = "!="
	GT  LogicalOp = ">"
	LT  LogicalOp = "<"
	GTE LogicalOp = ">="
	LTE LogicalOp = "<="
	IN  LogicalOp = "IN"

	ADD LogicalOp = "+"
	SUB LogicalOp = "-"
	MUL LogicalOp = "*"
	DIV LogicalOp = "/"
	MOD LogicalOp = "%"

	BITAND LogicalOp = "&"
	BITOR  LogicalOp = "|"
	BITXOR LogicalOp = "^"
	BITLSH LogicalOp = "<<"
	BITRSH LogicalOp = ">>"
	BITNOT LogicalOp = "~"
)
```

```go
type LogicalExpression interface {
	Expression
	Scope(LogicalOp, Expression) LogicalExpression
	EQ(key interface{}, vals ...interface{}) LogicalExpression
	NE(key interface{}, vals ...interface{}) LogicalExpression
	GT(key interface{}, vals ...interface{}) LogicalExpression
	LT(key interface{}, vals ...interface{}) LogicalExpression
	GTE(key interface{}, vals ...interface{}) LogicalExpression
	LTE(key interface{}, vals ...interface{}) LogicalExpression
	ADD(key interface{}, vals ...interface{}) LogicalExpression
	SUB(key interface{}, vals ...interface{}) LogicalExpression
	MUL(key interface{}, vals ...interface{}) LogicalExpression
	DIV(key interface{}, vals ...interface{}) LogicalExpression
	MOD(key interface{}, vals ...interface{}) LogicalExpression
	BITAND(key interface{}, vals ...interface{}) LogicalExpression
	BITOR(key interface{}, vals ...interface{}) LogicalExpression
	BITXOR(key interface{}, vals ...interface{}) LogicalExpression
	BITLSH(key interface{}, vals ...interface{}) LogicalExpression
	BITRSH(key interface{}, vals ...interface{}) LogicalExpression
	BITNOT(key interface{}, vals ...interface{}) LogicalExpression
}
```

Use `expr.Logical(expr ...any)` to start a logical chain, and `expr.Op(op any) (LogicalOp, bool)` to parse operator strings like `"="`, `"+"`, or `"<<"`:

```go
qs := queries.GetQuerySet(&Product{}).
	Filter(expr.Logical("Price").MUL("Quantity").GTE(1000))
```

> **Note:** Methods on `LogicalExpression` (`EQ`, `NE`, `ADD`, etc.) treat plain `string` arguments as field names (`expr.Field`). When comparing against a string literal, wrap it in `expr.Value(...)`:
>
> ```go
> expr.Logical("Status").EQ(expr.Value("published"))
> ```

---

## Fields, Values, Aliases, and Chaining

### `expr.Field(fld string) NamedExpression`

References a model field by path (`"Name"` or dot-separated relation path `"Author.Name"`).

Because Go struct field names always start with an uppercase letter, a lowercase first segment (such as `"p_t.Name"`) is treated as an explicit table alias in raw queries rather than a relation traversal.

### `expr.As(name string, e Expression) NamedExpression`

Attaches an output or target name to an expression and implements `AnnotationExpression`.

```go
// Annotate an aggregate onto each row:
qs := queries.GetQuerySet(&Author{}).
	Annotate("BookCount", expr.COUNT("Books.ID"))

// Or pass As directly to Select / Update:
_, err := queries.GetQuerySet(&Product{}).
	Filter("Category", "books").
	Update(&Product{}, expr.As("Price", expr.Logical("Price").MUL(1.1)))
```

### `expr.Value` / `expr.V` / `expr.Values`

```go
func Value(v any, unsafe ...bool) Expression
func V(v any, unsafe ...bool) Expression
func Values(vs ...any) Expression
```

- `Value` (and its shorthand `V`) binds a parameter placeholder unless `unsafe` is `true`, in which case the formatted literal is inlined directly into SQL (do not pass untrusted input with `unsafe=true`). If `v` already implements `Expression` or `ExpressionBuilder`, it is returned as an expression; if `v` is an `attrs.Definer`, it is normalized to its primary key value.
- `Values` emits parenthesized parameter lists `(?, ?, ...)` for `IN` clauses or multi-value SQL constructs, and panics if called with zero arguments.

> **Caveat on PostgreSQL:** On `*drivers.DriverPostgres`, `value.SQL` emits `::INT` for all signed integer kinds (`int` through `int64`) and rejects unsigned integers (`uint` through `uint64`) with `"unsupported value type ... in expression"`. `values.SQL` emits `::INT` for both signed and unsigned integer kinds. Avoid passing values exceeding 32-bit signed integer range through `expr.Value` / `expr.Values` on PostgreSQL unless explicitly cast via SQL/raw statements.

### `expr.String(s string) Expression`

Wraps a raw SQL fragment string with no parameter binding (`String("COUNT(*)")`).

### `expr.Chain(expr ...any) NamedExpression`

Concatenates expressions, `LogicalOp` operators, `ExpressionBuilder` values, or field name strings in sequence without extra separators (`LogicalOp` itself writes a leading and trailing space). Its `FieldName()` comes from the first `NamedExpression` or string field name.

---

## Subqueries and Outer References

Use `expr.OuterRef(fld string) NamedExpression` inside a subquery to reference a field from its enclosing parent query:

```go
subq := queries.GetQuerySet(&Comment{}).
	Filter("Post", expr.OuterRef("ID")).
	Limit(1)

posts, err := queries.GetQuerySet(&Post{}).
	Filter(expr.EXISTS(subq.Select("ID"))).
	All()
```

See [QuerySet](../queryset/queryset.md) for `Subquery` and `EXISTS` usage, and `expr.MakeSubqueryContext` / `expr.AddParentSubqueryContext` for custom subquery resolution.

---

## Raw Expressions and Statement Parser

For database-specific SQL that still needs safe model/column resolution and parameter binding, use `expr.Raw` or `expr.F`:

```go
func Raw(statement string, value ...any) Expression
func F(statement any, value ...any) NamedExpression
```

- `Raw` parses `statement` into an `*ExpressionStatement` with bound `value` arguments.
- `F` does the same, or chains expressions when given a `NamedExpression` / `Expression`, and sets `FieldName()` to the first `![Field]` found (panicking if a raw string statement contains no `![...]` field placeholder).

Supported placeholders in `ParseExprStatement` (`expr.PARSER`):

| Syntax | Description | Example |
| :--- | :--- | :--- |
| `![FieldPath]` | Resolves a field (or `alias.Field`) on the current model | `![Price] * ?[1]` |
| `#[pkg.Model.Field]` / `#[alias.pkg.Model.Field]` | Resolves a field on another registered content-type model | `#[auth.User.Email]` |
| `'ident'` / `'table.col'` | Quotes identifier segments with `ExpressionInfo.QuoteIdentifier` | `'custom_col'` |
| `TABLE(SELF)`, `TABLE(Rel)`, `TABLE(Rel as alias)`, `TABLE(pkg.Model)` | Emits quoted table name (and alias) | `TABLE(Author as a)` |
| `?` | Positional bound parameter (or embedded `Expression`) | `?[1]` or `?` |
| `?[1]` | 1-based indexed bound parameter (reusable multiple times; `?[0]` errors) | `![Age] BETWEEN ?[1] AND ?[2]` |
| `EXPR(0)` / `EXPR(Name)` | Embeds an `expr.Expression` supplied via `expr.PARSER.Expressions(...)` | `EXPR(FilterCond)` |

```go
stmt := expr.F(
	"COALESCE(![NickName], ![Name]) || ?[1]",
	" (verified)",
)

raw := expr.Raw(
	"EXISTS (SELECT 1 FROM TABLE(Comments as c) WHERE ![c.Post] = ![ID] AND EXPR(ActiveOnly))",
	expr.PARSER.Expressions(map[string]expr.Expression{
		"ActiveOnly": expr.Q("c.Approved", true),
	}),
)
```

---

## Type Casting

Use `expr.Cast` or its typed convenience constructors to emit driver-appropriate `CAST(...)` SQL:

```go
type CastType uint

const (
	CastTypeUnknown CastType = iota
	CastTypeString
	CastTypeText
	CastTypeInt
	CastTypeFloat
	CastTypeBool
	CastTypeDate
	CastTypeTime
	CastTypeBytes
	CastTypeDecimal
	CastTypeJSON
	CastTypeUUID
	CastTypeNull
	CastTypeArray
)

func Cast(typ CastType, col any, value ...any) NamedExpression
func RegisterCastType(castType CastType, entry CastFuncEntry, drivers ...driver.Driver)
```

Convenience functions (with required argument counts):

- `CastString(col, length)` (strictly 1 extra argument — even on SQLite where SQL renders `CAST(%s AS TEXT)`)
- `CastFloat(col, precision, scale)` (strictly 2 extra arguments — even on SQLite where SQL renders `CAST(%s AS REAL)`)
- `CastDecimal(col, precision, scale)` (strictly 2 extra arguments)
- Zero extra arguments: `CastText`, `CastInt`, `CastBool`, `CastDate`, `CastTime`, `CastBytes`, `CastJSON`, `CastUUID`, `CastNull`, `CastArray`

```go
qs := queries.GetQuerySet(&Product{}).
	Annotate("PriceInt", expr.CastInt("Price"))
```

---

## Built-In SQL Functions

All built-in SQL function constructors return `LogicalNamedExpressionFunc` (implementing both `NamedExpression` and `LogicalExpression`):

| Constructor | SQL / Behavior |
| :--- | :--- |
| `expr.SUM(expr ...any)` | `SUM(...)` |
| `expr.COUNT(expr ...any)` | `COUNT(...)` |
| `expr.AVG(expr ...any)` | `AVG(...)` |
| `expr.MAX(expr ...any)` | `MAX(...)` |
| `expr.MIN(expr ...any)` | `MIN(...)` |
| `expr.COALESCE(expr ...any)` | `COALESCE(a, b, ...)` (requires at least 2 expressions) |
| `expr.CONCAT(expr ...any)` | `CONCAT(a, b, ...)` on MySQL/MariaDB/Postgres; `(a \|\| b ...)` on SQLite (requires at least 2) |
| `expr.SUBSTR(expr any, start, length any)` | `SUBSTRING(x, start, len)` on MySQL/MariaDB, `SUBSTRING(x FROM start FOR len)` on Postgres, `SUBSTR(x, start, len)` on SQLite (pass `nil` for `length` to omit it) |
| `expr.EXISTS(expr any)` | `EXISTS (<subquery>)` |
| `expr.UPPER(expr any)` | `UPPER(...)` |
| `expr.LOWER(expr any)` | `LOWER(...)` |
| `expr.LENGTH(expr any)` | `LENGTH(...)` |
| `expr.NOW()` | `NOW()` on MySQL/MariaDB; `CURRENT_TIMESTAMP` on Postgres/SQLite |
| `expr.UTCNOW()` | `UTC_TIMESTAMP()` on MySQL/MariaDB; `CURRENT_TIMESTAMP AT TIME ZONE 'UTC'` on Postgres; `julianday('now')` on SQLite |
| `expr.LOCALTIMESTAMP()` | `LOCALTIMESTAMP()` on MySQL/MariaDB; `LOCALTIMESTAMP` on Postgres; `CURRENT_TIMESTAMP` on SQLite |
| `expr.DATE(expr any)` | `DATE(...)` |
| `expr.DATE_FORMAT(expr any, format string)` | `DATE_FORMAT(x, 'fmt')` on MySQL/MariaDB; `TO_CHAR(x, 'fmt')` on Postgres; `STRFTIME('fmt', x)` on SQLite |

Register custom SQL functions globally or per driver with `expr.RegisterFunc`:

```go
expr.RegisterFunc("TRIM", func(d driver.Driver, value []expr.Expression, funcParams []any) (string, []any, error) {
	// ...
})
```

*(Note: `"TRIM"` is registered internally in the function registry, though it does not currently have a top-level Go wrapper function in `expr`.)*

---

## Deconstruction

Core `expr` types (`ExprNode`, `ExprGroup`, `logicalChainExpr`, `chainExpr`, `RawNamedExpression`, `ExpressionStatement`, `Function`, `CaseExpression`, `when`, `castExpr`, `String`, `field`, `value`, `values`, `namedExpression`, `outerRef`, `LogicalOp`) implement `deconstruct.Deconstructible` and register JSON constructors under `"queries.expr.*"`. See [Deconstruction](../../../docs/deconstruct.md) for serialization details.
