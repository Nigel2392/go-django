# Specific Models (`queries/src/specific`)

Package `github.com/Nigel2392/go-django/queries/src/specific` provides polymorphic base-to-concrete subtype storage and batched loading. You query a single base table (such as `GenericComment` or `BaseNotification`), and `SpecificQuerySet` resolves and fetches the concrete subtype tables in batches grouped by `ContentType`.

---

## Core Types and Interfaces

```go
type Model interface {
	attrs.Definer
	SpecificObject() *Specific[attrs.Definer]
}

type TModel[T any] interface {
	attrs.Definer
	Embedder() T
	SpecificObject() *Specific[attrs.Definer]
}

type Specific[EMBEDDER attrs.Definer] struct {
	SpecificID          string                                       `json:"specific_id,omitempty"`
	SpecificContentType *contenttypes.BaseContentType[attrs.Definer] `json:"specific_content_type,omitempty"`
	Instance            attrs.Definer                                `json:"-" proxy:"-"`
	// unexported embedder reference
}

type SpecificModel[ORIGINAL Model, SPECIFIC attrs.Definer] struct {
	Original ORIGINAL
	Specific SPECIFIC
}
```

- Embed `specific.Specific[*MyBaseModel]` in your base model (making it satisfy `specific.Model` and `specific.TModel[*MyBaseModel]`). Note `Instance` and `embedder` are tagged with `` `proxy:"-"` `` so `models.Model` does not treat them as proxy relations.
- `(*Specific[E]).Fields(ctx context.Context) []attrs.Field`: Injects read-only `SpecificID` and `SpecificContentType` fields (with `FormField` returning `nil`) into `FieldDefs`.
- `(*Specific[E]).SetupEmbedded(ctx, embedder)` and `PreSaveEmbedded(ctx, embedder)`: Automatically resolve `Instance` from `embedder.RootContext(ctx).Root` (or keep an explicitly assigned `Instance`), populate `SpecificContentType`, and JSON-marshal `Instance`'s primary key into `SpecificID`.
- `(*Specific[E]).Specific(ctx context.Context, refresh bool) (attrs.Definer, error)`: Returns the cached `Instance` when `!refresh && Instance != nil`, or queries the concrete model by `SpecificID`, copies base fields into the concrete embedder, and updates `Instance`.

---

## Defining a Polymorphic Base and Concrete Subtypes

```go
package models

import (
	"context"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/specific"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

type GenericComment[T any] struct {
	specific.Specific[*GenericComment[T]]
	ID   drivers.ULID
	Text string
}

func (b *GenericComment[T]) BeforeSave(ctx context.Context) error {
	if b.ID.IsZero() {
		b.ID = drivers.NewULID()
	}
	return nil
}

func (b *GenericComment[T]) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make[*GenericComment[T], any](ctx, b,
		attrs.NewField(b, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(b, "Text", nil),
		b.Specific.Fields,
	).WithTableName("generic_comments")
}

type CommentWithAuthor struct {
	*GenericComment[CommentWithAuthor]
	Author *Author
}

func (b *CommentWithAuthor) FieldDefs(ctx context.Context) attrs.Definitions {
	if b.GenericComment == nil {
		b.GenericComment = &GenericComment[CommentWithAuthor]{}
	}
	return attrs.Make(ctx, b,
		attrs.NewField(b, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(b, "Author", &attrs.Config{
			Column:        "author_id",
			RelForeignKey: attrs.Relate(&Author{}, "", nil),
		}),
	).WithTableName("comments_with_author")
}
```

Ensure all base and concrete types are registered (`attrs.RegisterModel` / app `Models()`) so `contenttypes.DefinitionForType` can resolve concrete types at query time.

---

## `SpecificQuerySet` and `GetSpecificQuerySet`

`GetSpecificQuerySet` wraps any `queries.AbstractQuerySet` with batched concrete subtype resolution:

```go
type SpecificQuerySetOptions[ORIGINAL Model, SPECIFIC attrs.Definer] struct {
	GetSpecificQuerySet func(
		targetContentType *contenttypes.ContentTypeDefinition,
		target SPECIFIC,
	) queries.BaseReadQuerySet[SPECIFIC, *queries.QuerySet[SPECIFIC]]
}

func GetSpecificQuerySet[ORIGINAL Model, SPECIFIC attrs.Definer, QS queries.AbstractQuerySet[ORIGINAL, QS]](
	qs QS,
	opts SpecificQuerySetOptions[ORIGINAL, SPECIFIC],
) *SpecificQuerySet[ORIGINAL, SPECIFIC, QS]
```

`SpecificQuerySet` embeds `*queries.WrappedQuerySet` and implements `BaseSpecificQuerySet[ORIGINAL, SPECIFIC, QS]` (`queries.BaseReadQuerySet` of `SpecificModel[ORIGINAL, SPECIFIC]` and `queries.BaseWriteQuerySet` of `ORIGINAL`):

```go
baseQS := queries.GetQuerySetWithContext(ctx, &GenericComment[CommentWithAuthor]{})
opts := specific.SpecificQuerySetOptions[*GenericComment[CommentWithAuthor], attrs.Definer]{
	GetSpecificQuerySet: func(ct *contenttypes.ContentTypeDefinition, target attrs.Definer) queries.BaseReadQuerySet[attrs.Definer, *queries.QuerySet[attrs.Definer]] {
		// Optionally customize preloads/SelectRelated per concrete subtype:
		if ct.TypeName() == contenttypes.NewContentType(&CommentWithAuthor{}).TypeName() {
			return queries.GetQuerySetWithContext(ctx, target).SelectRelated("Author")
		}
		return queries.GetQuerySetWithContext(ctx, target)
	},
}

sqs := specific.GetSpecificQuerySet(baseQS, opts).WithContext(ctx)

rows, err := sqs.Filter("Text__icontains", "hello").All()
if err != nil {
	return err
}
for _, row := range rows {
	orig := row.Object.Original
	concrete := row.Object.Specific // hydrated concrete model instance
	_ = orig
	_ = concrete
}
```

### Batched Execution in `All`, `Get`, `First`, and `Last`

1. `All()` iterates the base query rows via `IterAll()`, grouping `SpecificPK()` values by `SpecificContentType.TypeName()` in an ordered map.
2. For each distinct concrete content type, it builds a `QuerySet` (via `opts.GetSpecificQuerySet` if provided, or `queries.GetQuerySet(model)`), filters with `Filter("<pkField>__in", ids)`, and runs `.All()` — issuing **one query per subtype** regardless of how many rows matched.
3. Each fetched subtype row is matched by primary key back to `row.Object.Specific`.
4. `Get()`, `First()`, and `Last()` all delegate to `All()`, so they also return populated `Specific` instances.

---

## Caveats

- **`IterAll()` does NOT populate `Specific`:** `SpecificQuerySet.IterAll()` streams `SpecificModel{Original: row.Object}` with `Specific` left `nil`, because batched subtype fetching requires collecting all `(ContentType, PK)` pairs first. If you iterate with `IterAll()`, call `row.Object.Original.SpecificObject().Specific(ctx, false)` per row (or use `All()` for batched loading).
- **`SpecificPK()` expectations and panic:** `SpecificID` stores the JSON-marshaled primary key of the concrete subtype (`SetupEmbedded` / `PreSaveEmbedded`). When `Instance == nil` and `SpecificID != ""`, `SpecificPK()` calls `json.Unmarshal` into the target subtype's primary key type and **panics** if `SpecificID` is invalid JSON (`"could not unmarshal pk"`). If `SpecificID` is empty (`SpecificPK()` returns `nil`) or the concrete row no longer exists in the target table, `All()` returns an `errors.CodeNoRows` error.
- **Assigning concrete objects before saving base rows:** Because `SpecificID` is populated from the concrete `Instance`'s primary key during `PreSaveEmbedded` / `SetupEmbedded`, ensure the concrete instance has its primary key generated or saved before saving the base `GenericComment` row.
