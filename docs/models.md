# Setting up your models

Models are the core of your application.  
They define the structure of your data and how it interacts with a data store.
In Go-Django, models are defined as structs define the following methods:

- `FieldDefs()` - Returns a list of [field definitions](./attrs/interfaces.md#definer) for the model.
- `Save(context.Context) error` - [Saves](./#saving-models) the model instance to the data store.
- `Delete(context.Context) error` - [Deletes](./#deleting-models) the model instance from the data store.

Go-Django's packages will internally call these methods to save or delete persistent data belonging to the models.

---

## Defining Models

Models can be defined as structs which implement only the `FieldDefs()` method, however apps like `contrib.admin` and `modelforms.ModelForm` will not  
be able to properly interact with these models.

It is also always a good idea to register a [`contenttypes.ContentTypeDefinition`](./contenttypes.md#registering-a-content-type) for your model, so that it can be used in other apps like `contrib.admin` and generally makes it easier to work with your models.

Let's setup an example to be sure your models do adhere to the Go-Django interface.

```go
package myapp

import (
    "context"
    "github.com/Nigel2392/go-django/src/models"
)

// See `Saving Models` and `Deleting Models`
var _ models.Model = &MyModel{}

type MyModel struct {
    Name string `json:"name"`
}


func (m *MyModel) FieldDefs() attrs.Definitions {
    return attrs.AutoDefinitions(m)
}
```

---

## Saving Models

Models can be saved using the `Save()` method.

This method will save the model instance to the data store and return an error if it fails.

```go
func (m *MyModel) Save(ctx context.Context) error {
    // Save the model instance to the data store
    return nil
}
```

---

## Deleting Models

Models can be deleted using the `Delete()` method.

This method will delete the model instance from the data store and return an error if it fails.

```go
func (m *MyModel) Delete(ctx context.Context) error {
    // Delete the model instance from the data store
    return nil
}
```
