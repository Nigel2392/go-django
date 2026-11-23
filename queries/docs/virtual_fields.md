# Virtual Fields

Virtual fields appear in your model's `FieldDefs(ctx context.Context)` definition but map to a **computed SQL expression** rather than a physical table column.

They are backed by `*fields.ExpressionField[T]`, constructed with `fields.NewVirtualField[T]`, and implement `queries.VirtualField` (`SQL(chain []string, inf *expr.ExpressionInfo) (string, []any)`) and `queries.AliasField` (`Alias() string`).

---

## When to Use Virtual Fields

- Concatenated display columns (such as `CONCAT(first_name, ' ', last_name)`)
- Normalized search columns (such as `LOWER(name)` or `UPPER(name)`)
- Per-row fallback or conditional values (such as `COALESCE(discount, 0)`)

---

## Signature

```go
func NewVirtualField[T any](forModel attrs.Definer, dst any, name string, expr expr.Expression) *ExpressionField[T]
```

- `T`: The Go type the result maps to (for example `string`, `int64`).
- `forModel`: The model instance (`t` in `FieldDefs(ctx)`).
- `dst`: Storage destination for scanned values — either the model itself (when embedding `models.Model`) or a pointer to a struct field.
- `name`: Field name and SQL column alias (`Alias()`).
- `expr`: Any `expr.Expression` (such as `expr.CONCAT`, `expr.LOWER`, `expr.UPPER`, `expr.COALESCE`).

---

## Two Usage Patterns

### 1. Model Embeds `models.Model` (model as `dst`)

When the struct embeds `models.Model` (which implements `queries.DataModel`), passing the model pointer as `dst` stores scanned virtual field values in `t.Annotations` (and its `DataStore()`):

```go
type TestStruct struct {
	models.Model
	ID   int64
	Name string
	Text string
}

func (t *TestStruct) FieldDefs(ctx context.Context) attrs.Definitions {
	return t.Model.Define(ctx, t,
		attrs.NewField(t, "ID", &attrs.Config{
			Column:  "id",
			Primary: true,
		}),
		attrs.NewField(t, "Name", &attrs.Config{Column: "name"}),
		attrs.NewField(t, "Text", &attrs.Config{Column: "text"}),
		fields.NewVirtualField[string](t, t, "TestNameText", expr.CONCAT(
			"Name", expr.Value(" ", true), "Text",
		)),
		fields.NewVirtualField[string](t, t, "TestNameLower", expr.LOWER("Name")),
		fields.NewVirtualField[string](t, t, "TestNameUpper", expr.UPPER("Name")),
	).WithTableName("test_struct")
}
```

Reading the computed value:

```go
row, err := queries.GetQuerySet(&TestStruct{}).
	WithContext(ctx).
	Filter("ID", 1).
	First()
if err != nil {
	return err
}
fmt.Println(row.Object.Annotations["TestNameLower"])
```

### 2. Struct Does Not Embed `models.Model` (field pointer as `dst`)

When `models.Model` is not embedded, pass a pointer to the struct field that should receive the computed value:

```go
type TestStructNoObject struct {
	ID            int64
	Name          string
	Text          string
	TestNameText  string
	TestNameLower string
	TestNameUpper string
}

func (t *TestStructNoObject) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make[*TestStructNoObject, any](ctx, t,
		attrs.NewField(t, "ID", &attrs.Config{Column: "id", Primary: true}),
		attrs.NewField(t, "Name", &attrs.Config{Column: "name"}),
		attrs.NewField(t, "Text", &attrs.Config{Column: "text"}),
		fields.NewVirtualField[string](t, &t.TestNameText, "TestNameText", expr.CONCAT("Name", expr.Value(" ", true), "Text")),
		fields.NewVirtualField[string](t, &t.TestNameLower, "TestNameLower", expr.LOWER("Name")),
		fields.NewVirtualField[string](t, &t.TestNameUpper, "TestNameUpper", expr.UPPER("Name")),
	).WithTableName("test_struct_no_object")
}
```

Reading the computed value:

```go
row, err := queries.GetQuerySet(&TestStructNoObject{}).
	WithContext(ctx).
	Filter("ID", 1).
	First()
if err != nil {
	return err
}
fmt.Println(row.Object.TestNameLower)
```

---

## `SELECT` Behavior and Notes

- **Included when `*` is selected:** Because `ExpressionField` implements `AliasField`, `queries.ForSelectAll(f)` returns `true` for virtual fields. Whenever a query selects all fields (default `.All()`/`.First()`/`.Get()` or `.Select("*")`), virtual fields are automatically included.
- **Excluded when omitting `*` in `.Select(...)`:** Calling `.Select("ID", "Name")` without `"*"` and without naming the virtual field **excludes** virtual fields from the generated `SELECT` list. You can still select a specific virtual field explicitly by name (for example `.Select("ID", "TestNameLower")`).
- **Read-only (`AllowDBEdit() == false`):** Virtual fields have `ColumnName() == ""` and are never written on `INSERT` or `UPDATE`.
- **Usable across relations:** `ExpressionField.SQL(chain []string, inf *expr.ExpressionInfo)` resolves the inner expression with `inf.ForExpressionChain(chain)`, so virtual fields on related models work in `Select("User.*")` and `Filter(...)` as well.

See [Expressions](./expressions/expressions.md) for all built-in functions such as `expr.CONCAT`, `expr.LOWER`, `expr.UPPER`, and `expr.COALESCE`.
