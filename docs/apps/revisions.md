# Revisions

The `revisions` app (`github.com/Nigel2392/go-django/contrib/revisions`) stores timestamped JSON snapshots of any Go-Django model (`attrs.Definer`) along with the user who created each revision, allowing you to inspect history and reconstruct historical model instances.

## Table of Contents

- [Installation and Migrations](#installation-and-migrations)
- [The `Revision` and `TypedRevision[T]` Models](#the-revision-and-typedrevisiont-models)
  - [Reconstructing Objects with `AsObject` and `ObjectFromDB`](#reconstructing-objects-with-asobject-and-objectfromdb)
- [Helper Functions](#helper-functions)
- [Querying with `RevisionQuerySet`](#querying-with-revisionqueryset)
- [Customizing Revision Serialization and Metadata](#customizing-revision-serialization-and-metadata)
- [Notes and Caveats](#notes-and-caveats)

---

## Installation and Migrations

Add `revisions.NewAppConfig` to `django.Apps(...)`. Your models must also be registered with `contenttypes` (see [Content Types](../contenttypes.md)) so `revisions` can resolve type names and restore instances:

```go
package main

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/auth"
	"github.com/Nigel2392/go-django/contrib/revisions"
	"github.com/Nigel2392/go-django/contrib/session"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := drivers.Open(context.Background(), "sqlite3", "./app.db")
	if err != nil {
		panic(err)
	}

	app := django.App(
		django.Configure(map[string]any{
			django.APPVAR_DATABASE: db,
		}),
		django.Apps(
			session.NewAppConfig,
			auth.NewAppConfig,
			revisions.NewAppConfig,
			migrator.NewAppConfig,
		),
	)

	if err := app.Initialize(); err != nil {
		panic(err)
	}
}
```

`revisions.NewAppConfig()` wraps a `*migrator.MigratorAppConfig` with embedded migrations under `migrations/revisions`:

- **With `migrator` installed**: Run `makemigrations` and `migrate` to create `revisions_revision`.
- **Without `migrator` installed**: `app.Init` automatically creates the `revisions_revision` table and indexes via `schemaEditor.CreateTable(context.Background(), table, true)`.
- **Admin registration**: `revisions` registers its model with `contrib/admin` solely as a hidden, read-only internal entry (`RegisterToAdminMenu: false`, all CRUD views disallowed).
- **`revisions.App()`**: Returns the active `*RevisionsAppConfig` singleton (panics if not yet initialized).

---

## The `Revision` and `TypedRevision[T]` Models

```go
type Revision struct {
	models.Model `table:"revisions_revision"`
	ID           int64      `json:"id"`
	ObjectID     string     `json:"object_id"`
	ContentType  string     `json:"content_type"`
	Data         string     `json:"data"`
	User         users.User `json:"user"`
	CreatedAt    time.Time  `json:"created_at"`
}

func NewRevision(forObj attrs.Definer, user users.User, getRevInfo ...QueryInfoFunc) (*Revision, error)
func (r *Revision) SetObject(obj attrs.Definer) error
func (r *Revision) AsObject(ctx context.Context) (attrs.Definer, error)
func (r *Revision) ObjectFromDB(ctx context.Context) (attrs.Definer, error)

type TypedRevision[T attrs.Definer] Revision

func (r *TypedRevision[T]) AsObject(ctx context.Context) (T, error)
func (r *TypedRevision[T]) ObjectFromDB(ctx context.Context) (T, error)
```

- `ObjectID` stores the JSON-marshaled primary key of the target object; `ContentType` stores its `contenttypes` type name.
- `Data` holds the JSON snapshot of the model (including proxy fields).
- `BeforeCreate` populates `CreatedAt` with `time.Now()` when zero; default ordering (`OrderBy()`) is `"-CreatedAt"`.
- `SetObject(obj)` verifies that `obj`'s primary key and content type match the revision and updates `r.Data`.
- `TypedRevision[T]` lets you cast a `*Revision` (`typed := (*revisions.TypedRevision[*Article])(rev)`) to receive a typed `T` directly from `AsObject` and `ObjectFromDB` (returning zero `T, nil` on `errors.NoRows`).

### Reconstructing Objects with `AsObject` and `ObjectFromDB`

`AsObject(ctx)` and `ObjectFromDB(ctx)` behave differently depending on whether you want a live database record merged with the revision or a pure in-memory snapshot:

- `ObjectFromDB(ctx)` unmarshals `r.ObjectID` into the model's primary key type and calls `contenttypes.DefinitionForType(r.ContentType).Instance(ctx, newPrimaryKey)` to fetch the current database row **without** applying `r.Data`.
- `AsObject(ctx)` checks `queries.IsCommitContext(ctx)`:
  - When `queries.IsCommitContext(ctx)` is `true` (the **default** for `context.Background()` and standard contexts), it calls `r.ObjectFromDB(ctx)` first so fields not present in `r.Data` (such as read-only fields or missing columns) retain their database values, and then overlays `r.Data` via `UnmarshalRevisionData`.
  - When `queries.IsCommitContext(ctx)` is `false` (for example with `queries.CommitContext(ctx, false)`), it skips the database fetch, allocates a fresh instance via `attrs.NewObject`, and populates it solely from `r.Data`.

```go
// Reconstruct solely from the stored JSON snapshot (no DB query for the target object):
snapshotCtx := queries.CommitContext(ctx, false)
obj, err := rev.AsObject(snapshotCtx)
if err != nil {
	return err
}
article := obj.(*Article)

// Or load the current DB record first, then overlay the revision snapshot:
mergedObj, err := rev.AsObject(ctx)
```

---

## Helper Functions

The package provides high-level CRUD helpers for creating, reading, updating, and deleting revisions:

```go
type QueryInfoFunc func(ctx context.Context, obj attrs.Definer) (pk any, contentType string, err error)

func CreateRevision(ctx context.Context, forObj attrs.Definer, user users.User, getRevInfo ...QueryInfoFunc) (*Revision, error)
func CreateDatedRevision(ctx context.Context, forObj attrs.Definer, user users.User, at time.Time, getRevInfo ...QueryInfoFunc) (*Revision, error)
func LatestRevision(ctx context.Context, obj attrs.Definer, getRevInfo ...QueryInfoFunc) (*Revision, error)
func ListRevisions(ctx context.Context, limit, offset int) ([]*Revision, error)
func GetRevisionByID(ctx context.Context, id int64) (*Revision, error)
func GetRevisionsByObject(ctx context.Context, obj attrs.Definer, limit int, offset int, getRevInfo ...QueryInfoFunc) ([]*Revision, error)
func UpdateRevision(ctx context.Context, rev *Revision) error
func DeleteRevision(ctx context.Context, rev *Revision) error
func DeleteRevisionsByObject(ctx context.Context, obj attrs.Definer, getRevInfo ...QueryInfoFunc) (int64, error)
```

Example lifecycle:

```go
// Create a revision for an article
rev, err := revisions.CreateRevision(ctx, article, currentUser)
if err != nil {
	return err
}

// Fetch the most recent revision (returns errors.NoRows if none exists)
latest, err := revisions.LatestRevision(ctx, article)

// List up to 20 revisions for this object ordered newest-first
history, err := revisions.GetRevisionsByObject(ctx, article, 20, 0)

// List recent revisions across all models
allRecent, err := revisions.ListRevisions(ctx, 50, 0)

// Update or delete a specific revision, or delete all revisions for an object
_ = revisions.UpdateRevision(ctx, latest)
_ = revisions.DeleteRevision(ctx, latest)
deletedCount, err := revisions.DeleteRevisionsByObject(ctx, article)
```

---

## Querying with `RevisionQuerySet`

For filtered queries, use `NewRevisionQuerySet()`, which wraps `*queries.QuerySet[*Revision]` with revision-specific helpers:

```go
func NewRevisionQuerySet() *RevisionQuerySet

func (qs *RevisionQuerySet) Base() *queries.QuerySet[*Revision]
func (qs *RevisionQuerySet) ForObjects(objs ...attrs.Definer) *RevisionQuerySet
func (qs *RevisionQuerySet) Types(types ...any) *RevisionQuerySet
func (qs *RevisionQuerySet) Users(users ...users.User) *RevisionQuerySet
func (qs *RevisionQuerySet) Since(t time.Time) *RevisionQuerySet
```

- `ForObjects(objs...)` groups objects by content type and builds an `expr.Or(...)` predicate over `ContentType` and `ObjectID` / `ObjectID__in`.
- `Types(types...)` accepts `string` (type name or alias resolved with `contenttypes.ReverseAlias`), `attrs.Definer`, or `contenttypes.ContentType`.
- `Users(users...)` filters by `User__in`.
- `Since(t)` filters by `CreatedAt__gte`.

```go
since := time.Now().Add(-7 * 24 * time.Hour)
rows, err := revisions.NewRevisionQuerySet().
	WithContext(ctx).
	Types(&Article{}).
	ForObjects(articleA, articleB).
	Users(currentUser).
	Since(since).
	OrderBy("-CreatedAt").
	All()
```

See [QuerySet](../../queries/docs/README.md) for standard `QuerySet` methods such as `Limit`, `Offset`, `SelectRelated`, and `All`.

---

## Customizing Revision Serialization and Metadata

Models can implement any of these optional interfaces to customize how revisions are identified and serialized:

```go
type RevisionInfoDefiner interface {
	GetRevisionInfo(ctx context.Context) (pk any, contentType string, err error)
}

type RevisionDataFieldExcluder interface {
	ExcludeFromRevisionData() []string
}

type RevisionDataUnMarshaller interface {
	UnmarshalRevisionData(data []byte) error
}

type RevisionDataMarshaller interface {
	MarshalRevisionData() (map[string]any, error)
}
```

- **`RevisionInfoDefiner`**: Overrides both `contenttypes` lookup and `attrs.PrimaryKey` when resolving `(pk, contentType)`.
- **`RevisionDataFieldExcluder`**: Returns a slice of field names to skip during default `GetRevisionData` and `UnmarshalRevisionData`.
- **`RevisionDataUnMarshaller`**: Intercepts `UnmarshalRevisionData(obj, data, opts...)` to decode `data` manually.

You can also invoke serialization helpers directly with `MarshallerOption` functions:

```go
type MarshallerOption func(*marshallerOption)

func MarshallerSkipProxyFields(b bool) MarshallerOption
func MarshallerAllowReadOnlyFields(b bool) MarshallerOption
func MarshallerGetFieldValue(fn func(def attrs.Field) (any, error)) MarshallerOption

func GetRevisionData(obj attrs.Definer, opts ...MarshallerOption) (map[string]any, error)
func MarshalRevisionData(obj attrs.Definer, opts ...MarshallerOption) ([]byte, error)
func UnmarshalRevisionData(obj attrs.Definer, data []byte, opts ...MarshallerOption) error
```

By default (with no options passed), `GetRevisionData` and `UnmarshalRevisionData`:

- Walk proxy fields recursively using `queries.ProxyFields(obj)` and nest their maps under the proxy field name (disable with `MarshallerSkipProxyFields(true)`).
- Skip read-only fields (`!def.AllowEdit()`) unless `MarshallerAllowReadOnlyFields(true)` is supplied.
- Skip `RelManyToMany` and `RelOneToMany` relations.

---

## Notes and Caveats

- **`AllowDataMarshaller` / `RevisionDataMarshaller` caveat**: In `GetRevisionData`, `RevisionDataMarshaller.MarshalRevisionData()` is gated by `o.AllowDataMarshaller` (`if m, ok := obj.(RevisionDataMarshaller); ok && o.AllowDataMarshaller`). However, `AllowDataMarshaller` is unexported on `marshallerOption` and there is no exported `MarshallerOption` helper to set it to `true`. As a result, `GetRevisionData`, `MarshalRevisionData`, and `CreateRevision` always run the default field/proxy serializer rather than calling `MarshalRevisionData()`. By contrast, `RevisionDataUnMarshaller.UnmarshalRevisionData` **does** run unconditionally during `UnmarshalRevisionData`.
- **`AsObject` requires a working `ContentTypeDefinition` (and DB row unless commit context is disabled)**: Because `queries.IsCommitContext(ctx)` defaults to `true`, calling `rev.AsObject(ctx)` with a plain context attempts `rev.ObjectFromDB(ctx)`, which fails if the object no longer exists in the database or if its `ContentTypeDefinition.GetInstance` is unset. Pass `queries.CommitContext(ctx, false)` when you want pure snapshot reconstruction without hitting the database.
- **`CreateDatedRevision` panic on conflicting `User`**: If you pass an existing `*Revision` that already has a non-nil `User` together with a non-nil `user` argument to `CreateDatedRevision`, it panics.
- **`ForObjects` panic**: Passing an object whose `getIdAndContentType` lookup fails (for example, an unregistered content type) causes `RevisionQuerySet.ForObjects` to panic.
