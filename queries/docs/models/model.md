# Model implementation details

The `Model` struct is an important, but not mandatory part of the queries package.  
We will explain its purpose and how it can be used to enhance your models, as well as some caveats to be aware of.  
The `Model` struct is designed to be embedded in your own `Definer` structs, either directly or indirectly through other structs (NEVER POINTERS!).  
Do note that `Model` struct does not implement the [`attrs.Definer`](https://github.com/Nigel2392/go-django/blob/main/docs/attrs/attrs.md) interface itself.

---

## Package interfaces

The `models` package provides an interface for models to implement, which is used to provide extra functionality to the model.  
It allows for custom save logic to be implemented, trying to call the embedder's `SaveObject()` method if it exists.  
When the embedder of the `Model` struct implements the `SaveableObject` interface, it will automatically call the `SaveObject()` method of the embedder,
the embedder should then call the `Model.SaveObject()` method to handle the actual saving of the model.

```go
type SaveableObject interface {
    SaveObject(ctx context.Context, cnf SaveConfig) error
}
```

---

## Extra Functionality

It provides extra functionality for models, it will:

- Setup reverse relation fields automatically
- Store annotations on the model
- Automatically setup proxy model relations (an embedded pointer to another model)
- Provide a `GetQuerySet()` method to get a `QuerySet` for the model which automatically includes a join to the proxy model.
- Provide a `table` on the model field to specify the table name
- Keep track of the model's state, such as whether it has been saved or not
- Automatically save the model adhering to the [models.ContextSaver](https://github.com/Nigel2392/go-django/blob/main/docs/models.md#saving-models) interface
- Store through relations (many-to-many) when the model is part of the target- end of a many-to-many or one-to-one relation
- Cache field definitions on the `Model` struct for faster access

---

## Usage

To use the `Model` struct, simply embed it in your own model struct.

```go
type BaseModel struct {
    models.Model
}

type MyModel struct {
    // models.Model // Embedding Model struct is recommended
    BaseModel       // This is also OK to directly embed the model struct.
    // *BaseModel   // This is NOT OK.
    ID   int
    Name string
}

func (m *MyModel) FieldDefs() attrs.Definitions {
    // Use the model to define the fields instead of `attrs.Define`
    return m.Model.Define(m,
        attrs.NewField(m, "ID", &attrs.FieldConfig{
            Primary: true,
        }),
        attrs.NewField(m, "Name"),
    )
}
```

This will give you access to all the extra functionality provided by the `Model` struct, such as reverse relations, annotations, and the `GetQuerySet()` method.
To use the model directly in the code without having fetched it from the database, please look at the [Caveats](#caveats) section below.

Noticing the above `FieldDefs` method, we are using the `Model.Define()` method to define the fields of the model.  
This is a convenience method which will not only define the fields, but it also sets up the model's reverse relations, and a field to a possible proxy model.

---

## Retrieving Annotated Values

Models can be directly annotated by the queryset, allowing you to retrieve additional computed values alongside the model's fields.

For this example we will assume that the model has been saved to the database, and we want to retrieve an annotated value from it.

```go
func main() {
    var myModelRow, err = queries.GetQuerySet(&MyModel{}).
        Filter("ID", 1).
        Select("ID", "Name").
        Annotate("NameLower", expr.FuncLower("Name")).
        First()
    if err != nil {
        panic(err)
    }

    var myModel = myModelRow.Object
    fmt.Println("ID:", myModel.ID)
    fmt.Println("Name:", myModel.Name)
    fmt.Println("NameLower:", myModel.Annotations["NameLower"])
}
```

---

## Saving Models

To save models, you can implement either one of two interfaces,
but we recommend you implement the `models.SaveableObject` interface, and then call up to the `Model` struct's `SaveObject()` method.

```go
func (m *MyModel) SaveObject(ctx context.Context, db *sql.DB) error {
    // Custom save logic here
    m.Name = fmt.Sprintf("[Saved] %s", m.Name)

    // Call the Model's SaveObject method to handle the actual saving
    return m.Model.SaveObject(ctx, db)
}
```

---

## Caveats

While the model struct does provide a lot of extra functionality, it requires some extra care when using it.

For example, when using the model directly in the code without having fetched it from the database, the model's internals are not properly setup,  
this should be done by calling either Setup on the object passing the eventual target model (the embedder of the model struct) to said Setup method,  
or by using `models.Setup(modelObj)` to setup the model object properly.

```go
func main() {
    // Create a new model object
    myModel := models.Setup(&MyModel{
        ID:   1,
        Name: "Test Model",
    })

    // Now you can use the model's functionality
    fmt.Println(myModel.Saved()) // false
}
```

---

## Interfaces implemented by the model

The `Model` struct implements the following interfaces:

- `models.ContextSaver`
  - Used internally to save the model using the context and database connection.
- `queries.CanSetup`
  - Allows the base model to be setup with the embedder model, this is a requirement  
    for the model to work properly.
- `queries.DataModel`
  - Provides the model with the ability to be used as a data model in queries,  
    this is useful for storing annotations and reverse relations.
- `queries.Annotator`
  - Allows the model to store annotations on itself.
- `queries.ThroughModelSetter`
  - Allows through relations to be set on the model itself (many-to-many) when it is part  
    of the target end of a many-to-many or one-to-one relation.
- `queries.ActsAfterSave`
  - Used internally to signal that the model has been saved, this is used to trigger
    any actions that need to be performed after the model has been saved.  
    If a sub-model implements this method, it must also call the base model's `ActsAfterSave` method.
- `queries.ActsAfterQuery`
  - Used internally to signal that the model has been queried,  
    this is used to trigger any actions that need to be performed after the model has been queried.  
    If a sub-model implements this method, it must also call the base model's `ActsAfterQuery` method.
- `attrs.CanSignalChanged`
  - Used internally to signal changes to the model, such as when a field is changed.
- `attrs.CanCreateObject[attrs.Definer]`
  - Allows the model to create a new object from itself, skipping the setup requirement because
    it is already setup.

---

## Model Methods

The `Model` struct provides methods for working with the model,  
these methods give access to the model's functionality, such as it's state other items.

### `Define(def attrs.Definer, flds ...any) *attrs.ObjectDefinitions`

Defines the fields of the model, and sets up the reverse relations and proxy model field if applicable.

### `DataStore() queries.ModelDataStore`

Returns the data store for the model, which is used to store annotations and reverse relations.

### `ModelMeta() attrs.ModelMeta`

Returns the model's metadata, which includes the model's name, table name, and field definitions.  
See [attrs.ModelMeta](https://github.com/Nigel2392/go-django/blob/main/docs/attrs/model-meta.md) for more information.

### `Object() attrs.Definer`

Returns the model object itself, which can be used to access the model's fields and methods.

### `PK() attrs.Field`

Returns the primary key field of the model, which can be used to access the model's primary key value.  
This field can be nil if the model does not have a primary key defined.

### `Save(ctx context.Context) error`

The base save method for the model, which will save the model to the database.  
It will call the `SaveObject()` method of the model, which can be implemented to provide custom save logic.  
If it is not implemented, this will automatically call the `SaveObject()` method of the `Model` struct itself.

### `SaveObject(ctx context.Context, cnf SaveConfig) (err error)`

The method to save the model object to the database, which can be implemented to provide custom save logic.
It will automatically save all model fields if possible, as well as it's proxy model if applicable.

### `Saved() bool`

Returns whether the model has been saved to the database or not.
It checks if the model has the `fromDB` flag set to true, or has a non-zero primary key value.

### `Setup(def attrs.Definer) error`

Sets up the model object with the given definer, which *must* be the embedder of the model struct.  
This is used to properly initialize the model's internals, such as reverse relations and proxy models.

### `State() *ModelState`

Returns the model's state, providing information about wether direct fields have been set or changed,
and providing access to the model's initial state.

---

For more information about proxy models, refer to the [Proxy Models](./proxy_models.md) documentation.
