# Expressions

Expressions are a powerful way to define complex queries and calculations in your application.

The base expression interface is `expr.Expression`, which can be used to create custom expressions that can be used in queries.

It looks like so:

```go
type Expression interface {
    SQL(sb *strings.Builder) []any
    Clone() Expression
    Resolve(inf *ExpressionInfo) Expression
}
```

Expressions must be resolved before they can be used in queries, which is done by calling the `Resolve` method.

This is done automatically internally when the query is being built, so you don't have to worry about it in most cases  
(except when creating custom expressions which embed other expressions).

## Using Expressions

There are multiple ways to use expressions in your queries, and there are multiple types of expressions available.

### Lookups

Lookups are a type of expression that allows you to filter querysets based on specific conditions.

For more information on lookups, see the [Lookups documentation](./lookups.md).

### Case Expressions

Case expressions allow you to create conditional expressions that can be used in queries.

For more information on case expressions, see the [Case Expressions documentation](./cases.md).

### `ClauseExpression`s

The most common type of expression is the `ClauseExpression`, which is used to define a SQL clause in a query.

These expressions are often joined together with `AND` or `OR` to create complex queries.

You can use the `expr.Q` function to create a `ClauseExpression` that can be used in queries.

It is also possible to use the `expr.Express` function to create a `ClauseExpression` that can be used in queries.

`expr.Express` is a more powerful function which returns a slice of `ClauseExpression`s.

The `ClauseExpression` interface looks the following:

```go
type ClauseExpression interface {
    Expression
    IsNot() bool
    Not(b bool) ClauseExpression
    And(...Expression) ClauseExpression
    Or(...Expression) ClauseExpression
}
```

### `LogicalExpression`s

Logical expressions are used to define logical operations in queries, such as equality, inequality, and arithmetic operations.

They provide a simple way to chain different types of expressions or values together to create complex queries.

A `LogicalOp` is used to define the type of logical operation being performed, such as `=`, `!=`, `>`, `<`, etc.

By itself, it is also an expression - the SQL representation of the LogicalOp is `<space><op><space>`.

```go
type LogicalOp string

const (
    EQ  LogicalOp = "="
    NE  LogicalOp = "!="
    GT  LogicalOp = ">"
    LT  LogicalOp = "<"
    GTE LogicalOp = ">="
    LTE LogicalOp = "<="

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

An example of using a `LogicalExpression` to filter a queryset might look like this:

```go
qs := queries.GetQuerySet(&Product{}).
    Filter(expr.Logical("Price").MUL("Quantity").Eq(1000))
```

### `NamedExpression`s

Named expressions are a type of expression that allows you to define an expression with a specific name, which can be used in queries.

These names might be useful in annotations, or to target a field on the model itself.

```go
type NamedExpression interface {
    Expression
    FieldName() string
}
```

An example of using a `NamedExpression` to annotate a queryset might look like this:

```go
qs := queries.GetQuerySet(&Product{}).
    Select("ID", expr.LOWER("Name"), "Price")
```

This will select the `ID` and `Price` fields, and also fetch the `Name` field as a lowercased value.

#### Special case

Since GO's struct fields cannot start with a lowercase letter, we took advantage of that.

Fieldpaths in expressions can as a matter of fact, start with lowercase letters. These would then indicate a table alias.

Take the following examples.

With an alias:

```go
var expr = expr.LOWER("Name") // translates to column `product`.`name`
```

Without an alias:

```go
var expr = expr.LOWER("p_t.Name") // translates to `p_t`.`name`
```

This is only really useful for raw expressions. The ORM will otherwise handle
any table alias for the specified field.


## Functions to create expressions

### `Chain(expr ...any) NamedExpression`

This will join the expressions together with no separator except for a space. It will return a `NamedExpression` that can be used in queries.

The FieldName is derived from the first `NamedExpression` or fieldname in the chain.

### `Q(fieldLookup string, value ...any) *ExprNode`

This function creates a `ClauseExpression` that can be used to filter querysets based on specific conditions.

It will retrieve the [`Lookup` expression](./lookups.md) for the given field lookup and apply it to the value(s) provided.

### `And(exprs ...Expression) *ExprGroup`

This function creates a group of expressions that are joined together with `AND`.

It returns an `ExprGroup` (ClauseExpression) that can be used in queries.

### `Or(exprs ...Expression) *ExprGroup`

This function creates a group of expressions that are joined together with `OR`.

It returns an `ExprGroup` (ClauseExpression) that can be used in queries.

### `Expr(field any, lookupOperation string, value ...any) *ExprNode`

This function creates a `ClauseExpression` that can be used to filter querysets based on specific conditions.

It will retrieve the [`Lookup` expression](./lookups.md) for the given lookupOperation and apply it to the value(s) provided.

### `Logical(expr ...any) LogicalExpression

This function creates a `LogicalExpression` that can be used to perform logical operations in queries.

See more about the [`LogicalExpression` interface](#logicalexpressions) above.

### `Raw(statement string, value ...any) Expression`

The Raw function creates a raw SQL expression that can be used in queries.

It allows you to write custom SQL statements directly, which can be useful for complex queries or when you need to use database-specific features.

Model fields can easily be referenced in the statement by using the `![field_name]` syntax, which will be replaced with the field's SQL representation.

### `F(statement any, value ...any) NamedExpression`

F creates a new RawNamedExpression or chainExpr with the given statement and values.

It parses the statement to extract the fields and values, and returns a pointer to the new RawNamedExpression.

The first field in the statement is used as the field name for the expression, and the rest of the fields are used as placeholders for the values.

The statement should contain placeholders for the fields and values, which will be replaced with the actual values.

The placeholders for fields should be in the format ![FieldName], and the placeholders for values should be in the format ?[Index],  
or the values should use the regular SQL placeholder directly (database driver dependent).

Example usage:

#### sets the field name to the first field found in the statement, I.E. `![Age]`

```go
expr := F("![Age] + ?[1] + ![Height] + ?[2] * ?[1]", 3, 4)
fmt.Println(expr.SQL()) // prints: "table.age + ? + table.height + ? * ?"
fmt.Println(expr.Args()) // prints: [3, 4, 3]
```

#### sets the field name to the first field found in the statement, I.E. `![Height]`

```go
expr := F("? + ? + ![Height] + ? * ?", 4, 5, 6, 7)
fmt.Println(expr.SQL()) // prints: "? + ? + table.height + ? * ?"
fmt.Println(expr.Args()) // prints: [4, 5, 6, 7]
```

### `String(fld string) Expression`

String creates a new `Expression` from the given string.

The string is presumed to be the *raw* SQL statement, no further processing is done on it.

### `Field(fld string) NamedExpression`

Field creates a new `NamedExpression` from the given field name.

It is a convenience function that returns a `NamedExpression` with the field name set to the given string.

The SQL output of this expression will be the database reference for the field, such as `table.field_name`.

### `Value(v any, unsafe ...bool) Expression`

Value creates a new `Expression` from the given value.

It allows for passing any value to SQL, using the database driver's placeholder syntax.

If `unsafe` is set to true, the value will be used as a raw value in the SQL statement, without any escaping or quoting,
the placeholder is unused.

## Function Expressions

The following functions can be used to create expressions that can be used in queries.

These functions return a `NamedExpression` that can be used in queries, and they often represent common SQL functions or operations.

### `Cast(typ CastType, col any, value ...any) NamedExpression`

The `Cast` function creates a `NamedExpression` that casts the given column to a specific type.

* `CastString(col any, value ...any) NamedExpression` (strictly 1 argument(s))
* `CastText(col any, value ...any) NamedExpression` (strictly 0 argument(s))
* `CastInt(col any, value ...any) NamedExpression` (strictly 0 argument(s))
* `CastFloat(col any, value ...any) NamedExpression` (strictly 2 argument(s))
* `CastBool(col any, value ...any) NamedExpression` (strictly 0 argument(s))
* `CastDate(col any, value ...any) NamedExpression` (strictly 0 argument(s))
* `CastTime(col any, value ...any) NamedExpression` (strictly 0 argument(s))
* `CastBytes(col any, value ...any) NamedExpression` (strictly 0 argument(s))
* `CastDecimal(col any, value ...any) NamedExpression` (strictly 2 argument(s))
* `CastJSON(col any, value ...any) NamedExpression` (strictly 0 argument(s))
* `CastUUID(col any, value ...any) NamedExpression` (strictly 0 argument(s))
* `CastNull(col any, value ...any) NamedExpression` (strictly 0 argument(s))
* `CastArray(col any, value ...any) NamedExpression` (strictly 0 argument(s))

The following `CastType` values are provided:

```go
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
```

### `FuncSum(expr ...any) *Function`

The `FuncSum` function creates a `Function` expression that calculates the sum of the given expression(s).

It compiles to the SQL `SUM` function, which is used to calculate the total sum of a numeric column.

I.e. `FuncSum("Price")` will compile to `SUM(table.price)`.

### `FuncCount(expr ...any) *Function`

The `FuncCount` function creates a `Function` expression that counts the number of rows that match the given expression(s).

It compiles to the SQL `COUNT` function, which is used to count the number of rows in a result set.

### `FuncAvg(expr ...any) *Function`

The `FuncAvg` function creates a `Function` expression that calculates the average of the given expression(s).

It compiles to the SQL `AVG` function, which is used to calculate the average value of a numeric column.

### `FuncMax(expr ...any) *Function`

The `FuncMax` function creates a `Function` expression that calculates the maximum value of the given expression(s).

It compiles to the SQL `MAX` function, which is used to find the maximum value in a result set.

### `FuncMin(expr ...any) *Function`

The `FuncMin` function creates a `Function` expression that calculates the minimum value of the given expression(s).

It compiles to the SQL `MIN` function, which is used to find the minimum value in a result set.

### `FuncCoalesce(expr ...any) *Function`

The `FuncCoalesce` function creates a `Function` expression that returns the first non-null value from the given expression(s).

It compiles to the SQL `COALESCE` function, which is used to return the first non-null value in a list of expressions.

### `FuncConcat(expr ...any) *Function`

The `FuncConcat` function creates a `Function` expression that concatenates the given expression(s) together.

It compiles to the SQL `CONCAT` function, which is used to concatenate strings together.

### `FuncSubstr(expr any, start, length any) *Function`

The `FuncSubstr` function creates a `Function` expression that extracts a substring from the given expression.

It compiles to the SQL `SUBSTRING` function, which is used to extract a substring from a string.

### `FuncUpper(expr any) *Function`

The `FuncUpper` function creates a `Function` expression that converts the given expression to uppercase.

It compiles to the SQL `UPPER` function, which is used to convert a string to uppercase.

### `FuncLower(expr any) *Function`

The `FuncLower` function creates a `Function` expression that converts the given expression to lowercase.

It compiles to the SQL `LOWER` function, which is used to convert a string to lowercase.

### `FuncLength(expr any) *Function`

The `FuncLength` function creates a `Function` expression that calculates the length of the given expression.

It compiles to the SQL `LENGTH` function, which is used to calculate the length of a string.

### `FuncNow() *Function`

The `FuncNow` function creates a `Function` expression that returns the current date and time.

It compiles to the SQL `NOW` function, which is used to get the current date and time from the database.
