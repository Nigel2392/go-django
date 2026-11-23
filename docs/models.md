# Models

Models are the core of your application. They define the structure of your data and how your application interacts with a persistent data store.

In Go-Django, any struct that implements `attrs.Definer` can act as a model. To integrate with persistence helpers across the framework, models commonly implement the interfaces in `github.com/Nigel2392/go-django/src/models` or register hooks for saving and deleting:

- `FieldDefs(context.Context) attrs.Definitions` — Returns the [field definitions](./attrs/interfaces.md) for the model.
- `Save(context.Context) error` or `Save() error` — [Saves](#saving-models) the model instance to the data store.
- `Delete(context.Context) error` or `Delete() error` — [Deletes](#deleting-models) the model instance from the data store.

---

## Model Interfaces

The `models` package (`github.com/Nigel2392/go-django/src/models`) defines lightweight interfaces for persistence and a reflection helper to inspect which lifecycle methods a model implements:

```go
package models

type ContextSaver interface {
	Save(c context.Context) error
}

type Saver interface {
	Save() error
}

type ContextDeleter interface {
	Delete(c context.Context) error
}

type Deleter interface {
	Delete() error
}

type ModelFunc func(c context.Context, m attrs.Definer) (changed bool, err error)

const (
	MODEL_SAVE_HOOK   = "django.Model.Save"
	MODEL_DELETE_HOOK = "django.Model.Delete"
)

func ImplementsMethods(m attrs.Definer) (definesSave, definesDelete bool)
```

- **`ContextSaver` / `Saver`**: Implemented by models that define their own persistence logic when saved.
- **`ContextDeleter` / `Deleter`**: Implemented by models that define their own deletion logic.
- **`ImplementsMethods`**: Reports whether `m` implements either save interface (`ContextSaver` or `Saver`) and either delete interface (`ContextDeleter` or `Deleter`).
- **`ModelFunc`**: Hook function signature used with `MODEL_SAVE_HOOK` and `MODEL_DELETE_HOOK`.

---

## Saving Models

The `SaveModel` function saves an `attrs.Definer` instance:

```go
func SaveModel(c context.Context, m attrs.Definer) (saved bool, err error)
```

When called, `SaveModel` executes the following steps in order:

1. If `m == nil`, it returns `(false, nil)`.
2. If `m` implements `Setup(ctx context.Context, obj attrs.Definer) error`, `SaveModel` calls `m.Setup(c, m)` first and aborts immediately if it returns an error.
3. If `m` implements `ContextSaver`, it calls `s.Save(c)` and returns `(true, err)`.
4. If `m` implements `Saver`, it calls `s.Save()` and returns `(true, err)`.
5. Otherwise, it iterates through hooks registered on `models.MODEL_SAVE_HOOK` via [goldcrest](./hooks.md) until a hook returns `saved == true` or a non-nil `err`.

If no save method or matching hook handles the model, `SaveModel` returns `(false, nil)`.

---

## Deleting Models

The `DeleteModel` function deletes an `attrs.Definer` instance:

```go
func DeleteModel(c context.Context, m attrs.Definer) (deleted bool, err error)
```

When called, `DeleteModel` executes the following steps in order:

1. If `m == nil`, it returns `(false, nil)`.
2. If `m` implements `ContextDeleter`, it calls `d.Delete(c)` and returns `(true, err)`.
3. If `m` implements `Deleter`, it calls `d.Delete()` and returns `(true, err)`.
4. Otherwise, it iterates through hooks registered on `models.MODEL_DELETE_HOOK` via [goldcrest](./hooks.md) until a hook returns `deleted == true` or a non-nil `err`.

If no delete method or matching hook handles the model, `DeleteModel` returns `(false, nil)`.

---

## Defining Models

A model struct only needs to implement `FieldDefs(context.Context) attrs.Definitions` to work with `attrs`, forms, and the ORM. However, registering your models on your app's [`ModelObjects`](./apps.md) slice ensures that:

- `django` automatically registers each model with `attrs.RegisterModel` and creates a [`contenttypes.ContentTypeDefinition`](./contenttypes.md) for it during initialization.
- Apps such as `contrib/admin`, choosers, and `forms` can inspect fields, labels, and instances consistently.

### Example Model Definition

```go
package users

import (
	"context"
	"time"

	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

type User struct {
	models.Model
	ID        int64
	Username  string
	IsActive  bool
	CreatedAt time.Time
}

func (u *User) FieldDefs(ctx context.Context) attrs.Definitions {
	return u.Model.Define(ctx, u,
		attrs.NewField(u, "ID", &attrs.Config{
			Primary:  true,
			ReadOnly: true,
		}),
		attrs.NewField(u, "Username", &attrs.Config{
			MaxLength: 150,
		}),
		attrs.NewField(u, "IsActive", &attrs.Config{}),
		attrs.NewField(u, "CreatedAt", &attrs.Config{
			ReadOnly: true,
		}),
	)
}
```

You can also register custom fallback persistence behavior via `goldcrest` when your models do not implement `Save` or `Delete` directly:

```go
goldcrest.Register(models.MODEL_SAVE_HOOK, 0, models.ModelFunc(
	func(ctx context.Context, m attrs.Definer) (bool, error) {
		// Custom save handling; return (true, nil) when handled.
		return false, nil
	},
))
```

Note that importing `github.com/Nigel2392/go-django/queries/src` automatically registers default `MODEL_SAVE_HOOK` and `MODEL_DELETE_HOOK` handlers that persist `attrs.Definer` models through `queries.GetQuerySet`.

---

## Querying Models

To query and persist models in a SQL database, Go-Django uses the [`queries`](https://github.com/Nigel2392/go-django/tree/main/queries) ORM package (`github.com/Nigel2392/go-django/queries/src`).

Using `queries.GetQuerySet`, you can fluently build and execute queries (`Filter`, `OrderBy`, `SelectRelated`, `Preload`, etc.). Calling `.All()` returns `queries.Rows[*User]` and an `error`:

```go
package users

import (
	"context"

	queries "github.com/Nigel2392/go-django/queries/src"
)

func GetActiveUsers(ctx context.Context) (queries.Rows[*User], error) {
	return queries.GetQuerySet(&User{}).
		WithContext(ctx).
		Filter("IsActive", true).
		OrderBy("-CreatedAt").
		All()
}
```

You can iterate directly over the returned model pointers using `rows.Objects()`:

```go
rows, err := GetActiveUsers(ctx)
if err != nil {
	return err
}

for user := range rows.Objects() {
	// Work with *User
	_ = user
}
```

For full documentation on model base types, querysets, expressions, drivers, and migrations, refer to the [queries documentation](../queries/docs/README.md) and [queries models guide](../queries/docs/models/models.md).
