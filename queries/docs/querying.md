# Querying, inserting, updating and deleting objects

Queries are built using the [`queries.Objects`](./querying/queryset.md#queryset) function, which takes a model type as an argument.

There are also some [helper functions](#helper-functions) for inserting, updating, deleting and counting objects.

We will use the models which are defined in the [models](./models.md) guide.

The database has to [have been setup](./getting_started.md#database-setup) before we can start using the queries package.

---

## Helper functions

There are some helper functions for inserting, updating, deleting and counting objects.

* **`ListObjectsByIDs`**`[T attrs.Definer, T2 any](object T, offset, limit uint64, ids []T2) ([]T, error)`
* **`ListObjects`**`[T attrs.Definer](object T, offset, limit uint64, ordering ...string) ([]T, error)`
* **`GetObject`**`[T attrs.Definer](object T, identifier any) (T, error)`
* **`CountObjects`**`[T attrs.Definer](obj T) (int64, error)`
* **`SaveObject`**`[T attrs.Definer](obj T) error`
* **`CreateObject`**`[T attrs.Definer](obj T) error`
* **`UpdateObject`**`[T attrs.Definer](obj T) (int64, error)`
* **`DeleteObject`**`[T attrs.Definer](obj T) (int64, error)`

These functions are defined in the `queries` package, and are simple to use.

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

`SaveObject` is a helper function saved the object to the database.  
It first will check if the primary key is set to a non-zero value.

* **If it is not set**, it creates a new object.
* **If it is set**, it updates the existing object.

```go
var todo = &Todo{
    ID: 1,
    Title: "Updated Test Todo",
    Description: "This is an updated test todo",
    Done: false,
    User: user,
}

var err = SaveObject[*Todo](t)
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

See [Querying Objects](./querying/queryset.md) for more advanced queries and usage…
