# Setting up your models

Models are the core of your application.  
They define the structure of your data and how it interacts with a data store.
In Go-Django, models are defined as structs define the following methods:

- `FieldDefs()` - Returns a list of [field definitions](./attrs/interfaces.md#definer) for the model.
- `Save(context.Context) error` - [Saves](./#saving-models) the model instance to the data store.
- `Delete(context.Context) error` - [Deletes](./#deleting-models) the model instance from the data store.

Go-Django's packages will internally call these methods to save or delete persistent data belonging to the models.

---

## Helper functions

There are 2 helper functions that can be used to interact with models.

### SaveModel

The `SaveModel` function can be used to save a model instance.

It will call the `Save()` method on the model instance, if it exists.

```go
SaveModel(context.Context, attrs.Definer) (saved bool, err error)
```

If the model does not have a `Save` method, it will call a chain of [hooks](./hooks.md#hooks) that can be used to save the model.

These functions must be of type `models.ModelFunc`, the hook that is used to register the function is `models.MODEL_SAVE_HOOK`.

### DeleteModel

The `DeleteModel` function can be used to delete a model instance.

It will call the `Delete()` method on the model instance, if it exists.

```go
DeleteModel(context.Context, attrs.Definer) (deleted bool, err error)
```

If the model does not have a `Delete` method, it will call a chain of [hooks](./hooks.md#hooks) that can be used to delete the model.

These functions must be of type `models.ModelFunc`, the hook that is used to register the function is `models.MODEL_DELETE_HOOK`.

## Defining Models

Models can be defined as structs which implement only the `FieldDefs()` method, however apps like `contrib.admin` and `modelforms.ModelForm` will not  
be able to properly interact with these models.

It is also always a good idea to register a [`contenttypes.ContentTypeDefinition`](./contenttypes.md#registering-a-content-type) for your model, so that it can be used in other apps like `contrib.admin` and generally makes it easier to work with your models.
