# Lookups

Lookup expressions are used to filter querysets based on specific conditions.

They allow you to specify how the filtering should be done, such as checking for equality, inequality, or membership in a set.

## Lookup Expressions

Lookup expressions are defined in the `expr` package through `expr.Express` and `expr.Q` functions.

These expressions can be used to filter querysets based on specific conditions.

## Using Lookups

To use lookups in your querysets, you can use the `Filter` method on the queryset.

For example, to filter a queryset based on a specific field, you can use the following syntax:

```go
qs := queries.GetQuerySet(&MyModel{}).
    Filter("FieldName", "Value")
```

You can also use more complex lookups, such as checking for membership in a set or filtering with case insensitive comparisons:

```go
qs := queries.GetQuerySet(&MyModel{}).
    Filter("FieldName__in", []string{"Value1", "Value2"}).
    Filter("FieldName__iexact", "value")
```

These expressions can be chained together to create more complex queries.

You can also use the `Q` function to create complex queries with multiple conditions:

```go
qs := queries.GetQuerySet(&MyModel{}).
    Filter(expr.Q("FieldName__in", []string{"Value1", "Value2"}).Or(expr.Q("FieldName__iexact", "value")))
```

## Available Lookup Expressions

The following lookup expressions are available:

- string `exact` - Compares if the result is equal to the given value.
- string `contains` - Compares if the result contains the given value.
- string `startswith` - Compares if the result starts with the given value.
- string `endswith` - Compares if the result ends with the given value.
- string `iexact` - Like `exact`, but case insensitive.
- string `icontains` - Like `contains`, but case insensitive.
- string `istartswith` - Like `startswith`, but case insensitive.
- string `iendswith` - Like `endswith`, but case insensitive.
- any `not` - Compares if the result is *not* equal to the given value.
- comparable `gt` - Compares if the result is greater than the given value.
- comparable `lt` - Compares if the result is less than the given value.
- comparable `gte` - Compares if the result is greater than or equal to the given value.
- comparable `lte` - Compares if the result is less than or equal to the given value.
- []any `in` - Compares if the result is in the given set of values.
- bool `isnull` - Compares if the result is null (or not null).
- (int, int) `range` - Compares if the result is within the given range of values.

These expressions can be used in the `Filter` method to filter querysets based on specific conditions.

Often, they can also be used inside of other expressions, such as the `CaseExpression` to create more complex queries.
