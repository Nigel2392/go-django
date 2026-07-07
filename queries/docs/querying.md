# Querying, inserting, updating and deleting objects

Querysets are built using [`queries.GetQuerySet`](./queryset/queryset.md) which takes a model instance as its argument.

There are also some [helper functions](#helper-functions) for inserting, updating, deleting and counting objects. These helpers internally build a `QuerySet` for you, so they follow the same database selection rules.

We will use the models which are defined in the [models](./models.md) guide.

The database has to [have been setup](./getting_started.md#database-setup) before we can start using the queries package.

---

## Helper functions

The following helper functions are defined in the `queries` package (i.e., `github.com/Nigel2392/go-django/queries/src`):

* **`ListObjectsByIDs`**`[T attrs.Definer, T2 any](object T, offset, limit uint64, ids []T2) ([]T, error)`
* **`ListObjects`**`[T attrs.Definer](object T, offset, limit uint64, ordering ...string) ([]T, error)`
* **`GetObject`**`[T attrs.Definer](object T, identifier any) (T, error)`
* **`CountObjects`**`[T attrs.Definer](obj T) (int64, error)`
* **`SaveObject`**`[T attrs.Definer](obj T) error`
* **`CreateObject`**`[T attrs.Definer](obj T) error`
* **`UpdateObject`**`[T attrs.Definer](obj T) (int64, error)`
* **`DeleteObject`**`[T attrs.Definer](obj T) (int64, error)`

All helpers use the background context (`context.Background()`) unless the model embeds `models.Model` and its `Save()` / `Delete()` hooks supply their own context.

For full control over the context, build a `QuerySet` directly using `queries.GetQuerySet(obj).WithContext(ctx)`.

### ListObjectsByIDs

`ListObjectsByIDs` is a helper function that takes an offset, limit, and a slice of IDs as parameters and returns a slice of objects of type T.

```go
var ids = []int{1, 2, 3}
var todos, err = queries.ListObjectsByIDs[*Todo](&Todo{}, 0, 1000, ids)
```

### ListObjects

`ListObjects` is a helper function that takes an offset and a limit as parameters and returns a slice of objects of type T.

```go
var todos, err = queries.ListObjects[*Todo](&Todo{}, 0, 1000, "-ID")
```

### GetObject

`GetObject` is a helper function that takes an identifier as a parameter and returns the object of type T.

The identifier can be any type, but it is expected to be the primary key of the object.

```go
var todo, err = queries.GetObject[*Todo](&Todo{}, 1)
```

### CountObjects

`CountObjects` is a helper function that counts the number of objects in the database.

```go
var count, err = queries.CountObjects[*Todo](&Todo{})
```

### SaveObject

`SaveObject` saves the object to the database.  
It first checks if the primary key is set to a non-zero value.

* **If it is not set**, it calls `CreateObject`.
* **If it is set**, it calls `UpdateObject`.

If the object implements `models.Saver` (the `Save(context.Context) error` method from `go-django`'s `src/models` package), the `Save` method will be called instead of executing a raw query.

```go
var todo = &Todo{
    Title:       "New Todo",
    Description: "Hello",
}
var err = queries.SaveObject(todo) // Calls CreateObject since ID is zero

todo.Title = "Updated Title"
var err2 = queries.SaveObject(todo) // Calls UpdateObject since ID is set
```

### Inserting Objects

`CreateObject` is a helper function that creates a new object in the database and sets its values
to the ones provided.

```go
var todo = &Todo{Title: "New Task", Done: false}
var err = queries.CreateObject(todo)
fmt.Println(todo.ID)
```

After insertion, the object will have its primary key (`ID`) set.

### Updating Objects

`UpdateObject` is a helper function that updates an existing object in the database.  
It returns the number of affected rows.

```go
todo.Title = "Updated Title"
var updatedCount, err = queries.UpdateObject(todo)
```

### Deleting Objects

`DeleteObject` is a helper function that deletes an object from the database.  
It returns the number of deleted rows.

```go
var deletedCount, err = queries.DeleteObject(todo)
```

---

See [QuerySet Reference](./queryset/queryset.md) for more advanced queries and usage…

See [Writing Queries](./queryset/writing_queries.md) for practical examples of filtering, pagination, ordering, and more.
