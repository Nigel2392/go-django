# Writing Queries

This guide provides practical examples for reading, filtering, preloading, subqueries, raw SQL, bulk updates, scopes, and transactions using `*queries.QuerySet[T]`.

---

## Basic Retrieval

### Retrieving All Objects

```go
todos, err := queries.GetQuerySet(&Todo{}).
	WithContext(ctx).
	All()
```

### Retrieving a Single Object

```go
// First row (errors.NoRows if empty)
firstTodo, err := queries.GetQuerySet(&Todo{}).WithContext(ctx).First()

// Last row (reverses ordering, errors.NoRows if empty)
lastTodo, err := queries.GetQuerySet(&Todo{}).WithContext(ctx).Last()

// Exactly one row (errors.NoRows if 0, errors.MultipleRows if > 1)
todo, err := queries.GetQuerySet(&Todo{}).
	WithContext(ctx).
	Filter("ID", 1).
	Get()
if err != nil {
	if errors.Is(err, errors.NoRows) {
		return nil
	}
	return err
}
```

### Streaming with `IterAll`

Use `IterAll()` to iterate row-by-row using Go range-over-func without pre-allocating the full result slice up front:

```go
count, seq, err := queries.GetQuerySet(&Todo{}).
	WithContext(ctx).
	Filter("Done", false).
	IterAll()
if err != nil {
	return err
}

_ = count
for row, err := range seq {
	if err != nil {
		return err
	}
	fmt.Println(row.Object.Title)
}
```

---

## Filtering, Negation, and Lookups

Chain `.Filter()` calls for `AND` conditions, or use `expr.Q(...).Not(true)` to negate (since there is no `Exclude` method):

```go
// All incomplete todos with ID > 5
todos, err := queries.GetQuerySet(&Todo{}).
	WithContext(ctx).
	Filter("ID__gt", 5).
	Filter(expr.Q("Done", true).Not(true)).
	All()
```

Combine conditions with `PushFilterOR` / `PushFilterAND`:

```go
todos, err := queries.GetQuerySet(&Todo{}).
	WithContext(ctx).
	Filter("Done", false).
	Filter(queries.PushFilterOR(expr.Q("User.Name", "Alice"))).
	All()
```

See [Lookups](../expressions/lookups.md) and [Expressions](../expressions/expressions.md) for all lookups and transforms.

---

## Pagination and Ordering

```go
pageTwo, err := queries.GetQuerySet(&Todo{}).
	WithContext(ctx).
	OrderBy("Done", "-ID").
	Limit(10).
	Offset(10).
	All()
```

Remember that `Objects()` defaults `Limit` to `1000`. Call `.Limit(0)` if you want no `LIMIT` clause.

---

## Selecting Fields and Eager Loading Relations

### `Select` and `SelectRelated` (FK / One-to-One `JOIN`s)

```go
// Select explicit fields only (omitting "*" excludes virtual fields)
titles, err := queries.GetQuerySet(&Todo{}).
	WithContext(ctx).
	Select("ID", "Title").
	All()

// Eagerly JOIN and select all fields of User (FK) and User.Profile (nested O2O)
withUsers, err := queries.GetQuerySet(&Todo{}).
	WithContext(ctx).
	SelectRelated("User", "User.Profile").
	All()
```

Note that `SelectRelated` modifies the `QuerySet` in place and only supports single-valued relations (`ForeignKey` / `OneToOne`).

### `Preload` with `NoJoins` and Custom QuerySets (O2M / M2M)

Use `Preload` for `OneToMany` (`RelRevFK`) and `ManyToMany` (`RelM2M`) relations, optionally passing `queries.NoJoins(...)` or a custom `queries.Preload{...}` QuerySet:

```go
// Simple preload without extra parent JOINs
users, err := queries.GetQuerySet(&User{}).
	WithContext(ctx).
	Preload(queries.NoJoins("TodoSet")).
	All()

// Preload with custom filtering & ordering on the related QuerySet
users, err = queries.GetQuerySet(&User{}).
	WithContext(ctx).
	Preload(queries.Preload{
		Path:     "TodoSet",
		NoJoins:  true,
		QuerySet: queries.Objects[attrs.Definer](&Todo{}).Filter("Done", false).OrderBy("-ID"),
	}).
	All()
```

---

## Subqueries, `OuterRef`, and `EXISTS`

Correlate subqueries with outer columns using `queries.Subquery`, `expr.OuterRef`, and `expr.EXISTS`:

```go
// Annotate each user with their latest todo title
latestTodoSubquery := queries.GetQuerySet(&Todo{}).
	Select("Title").
	Filter("User.ID", expr.OuterRef("ID")).
	OrderBy("-ID").
	Limit(1)

users, err := queries.GetQuerySet(&User{}).
	WithContext(ctx).
	Select("ID", "Name").
	Annotate("LatestTodoTitle", queries.Subquery(latestTodoSubquery)).
	Filter("LatestTodoTitle__isnull", false).
	All()

for _, row := range users {
	fmt.Println(row.Object.Name, row.Annotations["LatestTodoTitle"])
}
```

Filter rows where a correlated subquery exists using `expr.EXISTS`:

```go
incompleteTodos := queries.GetQuerySet(&Todo{}).
	Select("ID").
	Filter("User.ID", expr.OuterRef("ID")).
	Filter("Done", false)

usersWithPending, err := queries.GetQuerySet(&User{}).
	WithContext(ctx).
	Filter(expr.EXISTS(queries.Subquery(incompleteTodos))).
	All()
```

---

## `Union` Queries

Combine compatible QuerySets with `Union` (both sides must select the same number of columns with compatible types):

```go
pending := queries.GetQuerySet(&Todo{}).
	Select("ID", "Title", "Done").
	Filter("Done", false)

done := queries.GetQuerySet(&Todo{}).
	Select("ID", "Title", "Done").
	Filter("Done", true)

combined, err := pending.Union(done).
	OrderBy("-ID").
	All()
```

---

## Updating and Deleting

Use `expr.As(fieldName, expr)` (such as `expr.Value(true)`) for expression-based updates:

```go
// Mark all pending todos as done
updatedCount, err := queries.GetQuerySet(&Todo{}).
	WithContext(ctx).
	Filter("Done", false).
	Update(nil, expr.As("Done", expr.Value(true)))

// Or using a map of expressions:
updatedCount, err = queries.GetQuerySet(&Todo{}).
	WithContext(ctx).
	Filter("ID", 1).
	Update(nil, map[string]expr.Expression{
		"Done": expr.Value(false),
	})

// Delete matching rows
deletedCount, err := queries.GetQuerySet(&Todo{}).
	WithContext(ctx).
	Filter("Done", true).
	Delete()
```

Remember that when updating a struct instance `qs.Update(todo)` without `.Select(...)`, zero-valued fields (`false`, `0`, `""`) are skipped unless `.Select("Done")` or `expr.As("Done", expr.Value(false))` is used.

---

## ExplicitSave

When `QuerySet.Create` or `QuerySet.Update` is called without `ExplicitSave()`, the ORM delegates to `models.ContextSaver.Save(ctx)` if the model implements it. Calling `Create` or `Update` from inside `SaveObject` or an actor hook without `.ExplicitSave()` would cause infinite recursion.

Use `.ExplicitSave()` to execute the SQL directly:

```go
func (m *Todo) SaveObject(ctx context.Context, cnf models.SaveConfig) error {
	m.Title = strings.TrimSpace(m.Title)

	if m.Saved() {
		_, err := queries.GetQuerySet(&Todo{}).
			WithContext(ctx).
			ExplicitSave().
			Filter("ID", m.ID).
			Update(m)
		return err
	}

	_, err := queries.GetQuerySet(&Todo{}).
		WithContext(ctx).
		ExplicitSave().
		Create(m)
	return err
}
```

---

## Scopes

Scopes encapsulate reusable QuerySet transforms. Note that `Scope` mutates the receiving `QuerySet` in place:

```go
func DoneTodos(qs *queries.QuerySet[*Todo], internals *queries.QuerySetInternals) *queries.QuerySet[*Todo] {
	return qs.Filter("Done", true)
}

todos, err := queries.GetQuerySet(&Todo{}).
	WithContext(ctx).
	Scope(DoneTodos).
	All()
```

---

## Raw SQL (`Row`, `Rows`, `Exec`)

`QuerySet.Row`, `Rows`, and `Exec` parse statement placeholders (`![Field]`, `TABLE(SELF)`, `?[1]`), rebind parameters for the active driver, and participate in the QuerySet's context and transaction:

```go
qs := queries.GetQuerySet(&Todo{}).WithContext(ctx)

// Single row
row := qs.Row("SELECT ![ID], ![Title] FROM TABLE(SELF) WHERE ![ID] = ?[1]", 1)
var id int64
var title string
if err := row.Scan(&id, &title); err != nil {
	return err
}

// Multiple rows
sqlRows, err := qs.Rows("SELECT ![ID], ![Title] FROM TABLE(SELF) WHERE ![Done] = ?[1]", false)
if err != nil {
	return err
}
defer sqlRows.Close()
for sqlRows.Next() {
	if err := sqlRows.Scan(&id, &title); err != nil {
		return err
	}
}

// Exec
res, err := qs.Exec("UPDATE TABLE(SELF) SET ![Done] = ?[1] WHERE ![ID] = ?[2]", true, 1)
```

---

## Transactions and `CommitContext`

### Using `RunInTransaction`

```go
err := queries.RunInTransaction(ctx, func(txCtx context.Context, newQS queries.ObjectsFunc[*Todo]) (bool, error) {
	todo, err := newQS(&Todo{}).Create(&Todo{Title: "Transactional"})
	if err != nil {
		return false, err // rolls back
	}
	_ = todo
	return true, nil // commits
})
```

Any `QuerySet` bound to `txCtx` (via `WithContext(txCtx)` or `queries.QSC(txCtx, ...)`) automatically shares the transaction, even for different model types on the same database. Remember that `RunInTransaction` recovers panics, rolls back, logs the panic, and returns `nil`.

### Dry-Run / Validation Context (`CommitContext`)

Pass `queries.CommitContext(ctx, false)` to run setup, actors, and `Validate` across `BulkCreate`, `BulkUpdate`, `Delete`, and `StartTransaction` without writing to or committing the database:

```go
dryCtx := queries.CommitContext(ctx, false)
_, err := queries.GetQuerySet(&Todo{}).
	WithContext(dryCtx).
	BulkCreate([]*Todo{{Title: "Validate only"}})
```
