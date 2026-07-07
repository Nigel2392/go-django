# Writing Queries

The `queries` package provides a powerful and fluent `QuerySet` API to query models in your database without writing raw SQL.

## Basic Retrieval

### Retrieving All Objects

To retrieve all objects from a table, use the `.All()` method.

```go
var todos, err = queries.GetQuerySet(&Todo{}).WithContext(ctx).All()
```

### Retrieving a Single Object

To retrieve a single object, you can use `.First()`, `.Last()`, or `.Get()`.

```go
// Get the first object in the table
var firstTodo, err = queries.GetQuerySet(&Todo{}).WithContext(ctx).First()

// Get the last object in the table
var lastTodo, err = queries.GetQuerySet(&Todo{}).WithContext(ctx).Last()

// Get exactly one object (returns qerr.ErrMultipleRows if more than one matches)
var todo, err = queries.GetQuerySet(&Todo{}).WithContext(ctx).Filter("ID", 1).Get()
```

## Filtering

You can filter results using the `.Filter()` method. Multiple filters can be chained together.

```go
// Retrieve all "done" todos
var doneTodos, err = queries.GetQuerySet(&Todo{}).WithContext(ctx).Filter("Done", true).All()

// Retrieve todos with ID > 5
var recentTodos, err = queries.GetQuerySet(&Todo{}).WithContext(ctx).Filter("ID__gt", 5).All()
```

For more advanced filters, see [Lookups](../expressions/lookups.md).

## Pagination (Limit and Offset)

To limit the number of results or skip a certain number of results, use `.Limit()` and `.Offset()`.

```go
// Retrieve the first 10 todos
var pageOne, err = queries.GetQuerySet(&Todo{}).WithContext(ctx).Limit(10).All()

// Retrieve the next 10 todos (page 2)
var pageTwo, err = queries.GetQuerySet(&Todo{}).WithContext(ctx).Limit(10).Offset(10).All()
```

## Ordering

You can order results using `.OrderBy()`. Prefix the field name with `-` for descending order.

```go
// Order by title ascending
var asc, err = queries.GetQuerySet(&Todo{}).WithContext(ctx).OrderBy("Title").All()

// Order by ID descending
var desc, err = queries.GetQuerySet(&Todo{}).WithContext(ctx).OrderBy("-ID").All()
```

## Selecting specific fields

If you only need a subset of fields, you can use `.Select()` to optimize the query.

```go
// Only fetch ID and Title
var titles, err = queries.GetQuerySet(&Todo{}).WithContext(ctx).Select("ID", "Title").All()
```

## Updating and Deleting

QuerySets can also perform bulk updates and deletions based on the filters applied.

```go
// Mark all pending todos as done
var updatedCount, err = queries.GetQuerySet(&Todo{}).
    WithContext(ctx).
    Filter("Done", false).
    Update(nil, expr.Update("Done", true))

// Delete all done todos
var deletedCount, err = queries.GetQuerySet(&Todo{}).
    WithContext(ctx).
    Filter("Done", true).
    Delete()
```

## ExplicitSave

By default, when `QuerySet.Create` or `QuerySet.Update` is called, the ORM will invoke the model's `Save(ctx context.Context)` method (if it implements `models.ContextSaver`) to let the model handle its own persistence logic.

This causes an **infinite recursion** problem if you try to call `queries.GetQuerySet(...).Create(...)` or `.Update(...)` *from inside* the model's own `Save` or `SaveObject` method — the ORM will call `Save`, which calls the queryset, which calls `Save` again.

To break this cycle, call `.ExplicitSave()` on the queryset. This tells the ORM to **skip** calling the model's `Save` method and instead perform the database INSERT / UPDATE directly:

```go
func (m *Todo) SaveObject(ctx context.Context, cnf models.SaveConfig) error {
    // Do your custom logic here...
    m.Title = strings.TrimSpace(m.Title)

    // Use ExplicitSave to avoid infinite recursion — the ORM will not
    // call Save() on this model again, just run the SQL directly.
    if m.Saved() {
        _, err := queries.GetQuerySet(&Todo{}).WithContext(ctx).ExplicitSave().Update(m)
        return err
    }
    _, err := queries.GetQuerySet(&Todo{}).WithContext(ctx).ExplicitSave().Create(m)
    return err
}
```

> **When to use it:** Any time you call `Create` or `Update` from inside a model's `SaveObject`, `BeforeCreate`, `AfterCreate`, `BeforeUpdate`, or `AfterUpdate` hook.

---

## Scopes

Scopes allow you to encapsulate common query logic into reusable functions.

```go
func DoneTodos(qs queries.QuerySet[*Todo], internals *queries.QuerySetInternals) queries.QuerySet[*Todo] {
    return qs.Filter("Done", true)
}

// Applying the scope
var todos, err = queries.GetQuerySet(&Todo{}).Scope(DoneTodos).All()
```
