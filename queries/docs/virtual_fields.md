# Virtual Fields

Virtual fields are fields that appear in your model's `FieldDefs()` definition but map to a **computed SQL expression** rather than a plain database column.

They are backed by `fields.ExpressionField[T]`, created with `fields.NewVirtualField[T]()`, and implement the `queries.VirtualField` interface so the ORM knows to render them as SQL expressions in `SELECT` rather than as column references.

---

## When to use Virtual Fields

- Concatenated string columns (e.g. `CONCAT(first_name, ' ', last_name)`)
- Lowercased / uppercased search columns
- Computed aggregations inside a row (e.g. `COALESCE(price, 0)`)

---

## Signature

```go
func NewVirtualField[T any](forModel attrs.Definer, dst any, name string, expr expr.Expression) *ExpressionField[T]
```

- `T` — the Go type the result should be scanned into (e.g. `string`, `int`)
- `forModel` — the model instance this field belongs to (`t` in `FieldDefs`)
- `dst` — where the scanned value is stored; **two forms** are shown below
- `name` — the alias used in the query and in `Annotations`
- `expr` — any `expr.Expression` value (see `expr.CONCAT`, `expr.LOWER`, `expr.UPPER`, etc.)

---

## Two Usage Patterns

### 1. Model embeds `models.Model` — use the model as `dst`

When the outer struct embeds `models.Model`, the result is stored in `model.Annotations` and can be read from there:

```go
type TestStruct struct {
    models.Model
    ID   int64
    Name string
    Text string
}

func (t *TestStruct) FieldDefs() attrs.Definitions {
    return t.Model.Define(t,
        attrs.NewField(t, "ID", &attrs.FieldConfig{
            Column:  "id",
            Primary: true,
        }),
        attrs.NewField(t, "Name", &attrs.FieldConfig{Column: "name"}),
        attrs.NewField(t, "Text", &attrs.FieldConfig{Column: "text"}),
        // Pass 't' as dst: values land in t.Annotations["TestNameText"]
        fields.NewVirtualField[string](t, t, "TestNameText", expr.CONCAT(
            "Name", expr.Value(" ", true), "Text",
        )),
        fields.NewVirtualField[string](t, t, "TestNameLower", expr.LOWER("Name")),
        fields.NewVirtualField[string](t, t, "TestNameUpper", expr.UPPER("Name")),
    ).WithTableName("test_struct")
}
```

Accessing the result:

```go
row, _ := queries.GetQuerySet(&TestStruct{}).WithContext(ctx).Filter("ID", 1).First()
fmt.Println(row.Object.Annotations["TestNameLower"]) // "john doe"
```

### 2. Struct does NOT embed `models.Model` — use a field pointer as `dst`

When `models.Model` is not embedded, pass a pointer to the struct field that should receive the value:

```go
type TestStructNoObject struct {
    ID            int64
    Name          string
    Text          string
    TestNameText  string
    TestNameLower string
    TestNameUpper string
}

func (t *TestStructNoObject) FieldDefs() attrs.Definitions {
    return attrs.Define[*TestStructNoObject, any](t,
        attrs.NewField(t, "ID", &attrs.FieldConfig{Column: "id", Primary: true}),
        attrs.NewField(t, "Name", &attrs.FieldConfig{Column: "name"}),
        attrs.NewField(t, "Text", &attrs.FieldConfig{Column: "text"}),
        // Pass &t.TestNameText as dst: value is scanned directly into the field
        fields.NewVirtualField[string](t, &t.TestNameText,  "TestNameText",  expr.CONCAT("Name", expr.Value(" ", true), "Text")),
        fields.NewVirtualField[string](t, &t.TestNameLower, "TestNameLower", expr.LOWER("Name")),
        fields.NewVirtualField[string](t, &t.TestNameUpper, "TestNameUpper", expr.UPPER("Name")),
    ).WithTableName("test_struct_no_object")
}
```

Accessing the result:

```go
row, _ := queries.GetQuerySet(&TestStructNoObject{}).WithContext(ctx).Filter("ID", 1).First()
fmt.Println(row.Object.TestNameLower) // "john doe"
```

---

## Important Notes

- Virtual fields are **always included** in every `SELECT` query — they cannot be opted out of.
- The `expr` package provides many built-in functions: `expr.CONCAT`, `expr.LOWER`, `expr.UPPER`, `expr.Value`, `expr.FuncCoalesce`, etc. See the [expressions documentation](../expressions/expressions.md) for the full list.
- Virtual fields are **read-only** — they are never written back to the database.
