# Model Relations

Relations between models in the `queries` package are declared in `FieldDefs(ctx context.Context)` using either `attrs.Config` (`RelForeignKey`, `RelOneToOne`) or the typed shortcut constructors in `queries/src/fields` (`fields.ForeignKey`, `fields.OneToOne`, `fields.ManyToMany`).

> **Important:** Reverse relations are only injected into target model metadata when models are registered (via `attrs.RegisterModel` or app `Models()`).

---

## Struct Field Types and Relation Values

| Relation | Struct Field Type | Underlying `queries` Value Type |
|---|---|---|
| ForeignKey (`M:1`) | `*TargetModel` or `*queries.RelFK[*TargetModel]` | `queries.RelationValue` |
| OneToOne (direct) | `*TargetModel` | `attrs.Definer` |
| OneToOne (through table) | `*queries.RelO2O[*TargetModel, *ThroughModel]` | `queries.ThroughRelationValue` |
| ManyToMany (through table) | `*queries.RelM2M[*TargetModel, *ThroughModel]` | `queries.MultiThroughRelationValue` |
| Reverse FK / O2M | `*queries.RelRevFK[attrs.Definer]` (or typed `*queries.RelRevFK[*SourceModel]`) | `queries.MultiRelationValue` |

If you instantiate a struct in Go (`user := &User{}`) and want to access `.Objects()` on a `*RelRevFK`, `*RelO2O`, or `*RelM2M` field before saving or querying it, initialize the struct with `models.Setup(ctx, user)` so those relation fields are bound and non-nil.

---

## 1. Foreign Key (Many-to-One)

A `ForeignKey` stores the target model's primary key (or specified `TargetField`) in a column on the source table:

```go
type Todo struct {
	models.Model `table:"queries-todos"`
	ID           int64
	Title        string
	User         *User
}

func (m *Todo) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true, ReadOnly: true}),
		attrs.NewField(m, "Title", nil),
		attrs.NewField(m, "User", &attrs.Config{
			Null:          true,
			Column:        "user_id",
			RelForeignKey: attrs.Relate(&User{}, "", nil),
		}),
	)
}
```

Using the `fields.ForeignKey` shortcut:

```go
fields.ForeignKey[*User]("User", "user_id", &fields.FieldConfig{
	Nullable: true,
})
```

Creating rows with a foreign key:

```go
user := &User{Name: "Alice"}
user, err := queries.CreateObject(user)

// Link the loaded/created parent directly:
todo1, err := queries.CreateObject(&Todo{
	Title: "Buy milk",
	User:  user,
})

// Or link by primary key:
todo2, err := queries.CreateObject(&Todo{
	Title: "Buy bread",
	User:  &User{ID: user.ID},
})
```

---

## 2. One-to-One

### Direct One-to-One (No Through Model)

```go
type User struct {
	models.Model
	ID      uint64
	Name    string
	Profile *Profile
}

func (m *User) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true, ReadOnly: true}),
		attrs.NewField(m, "Name", nil),
		fields.OneToOne[*Profile]("Profile", &fields.FieldConfig{
			Nullable:   true,
			ColumnName: "profile_id",
			AllowEdit:  true, // OneToOneField defaults AllowEdit to false
		}),
	).WithTableName("queries-users")
}
```

Note that `fields.OneToOne` uses `RelationField.AllowEdit()` / `AllowDBEdit()`, which default to `false` unless `AllowEdit: true` is set in `FieldConfig`. Alternatively, `attrs.NewField(m, "Profile", &attrs.Config{Null: true, Column: "profile_id", RelOneToOne: attrs.Relate(&Profile{}, "", nil)})` defaults `AllowEdit` to `true`.

### One-to-One with a Through Model (`RelO2O`)

Use `*queries.RelO2O[*Target, *Through]` when a One-to-One link goes through an intermediate junction table:

```go
type Document struct {
	models.Model
	ID     int64
	Target *queries.RelO2O[*TargetDoc, *DocLink]
}

func (d *Document) FieldDefs(ctx context.Context) attrs.Definitions {
	return d.Model.Define(ctx, d,
		attrs.NewField(d, "ID", &attrs.Config{Primary: true}),
		fields.OneToOne[*queries.RelO2O[*TargetDoc, *DocLink]]("Target", &fields.FieldConfig{
			Rel: attrs.Relate(
				&TargetDoc{},
				"",
				&attrs.ThroughModel{
					This:   &DocLink{},
					Source: "SourceModel",
					Target: "TargetModel",
				},
			),
		}),
	)
}

type DocLink struct {
	models.Model
	SourceModel *Document
	TargetModel *TargetDoc
}

func (d *DocLink) FieldDefs(ctx context.Context) attrs.Definitions {
	return d.Model.Define(ctx, d,
		attrs.NewField(d, "SourceModel", &attrs.Config{Column: "source_id"}),
		attrs.NewField(d, "TargetModel", &attrs.Config{Column: "target_id"}),
	)
}
```

`*queries.RelO2O` exposes `.Objects() *queries.RelOneToOneQuerySet[TargetModel]`:

- `Load(ctx context.Context) error`: Loads the target and through object via `Get()` and populates `Object` and `ThroughObject`.
- `SetTarget(target T) (created bool, deleted bool, err error)`: Clears any existing link, inserts `target` if its primary key is zero, and inserts the through row.
- `Clear(ctx context.Context) (bool, error)`: Deletes the through row and clears `Object` / `ThroughObject`.
- Query helpers: `WithContext(ctx)`, `Select(...)`, `Filter(...)`, `OrderBy(...)`, `Limit(...)`, `Offset(...)`, `Exists()`, `Get()`, `All()`, `IterAll()`.

---

## 3. Many-to-Many (`RelM2M`)

Many-to-Many relations always require a `ThroughModel`. The struct field must be `*queries.RelM2M[*TargetModel, *ThroughModel]`:

```go
type Post struct {
	models.Model
	ID   int64
	Tags *queries.RelM2M[*Tag, *PostTag]
}

func (m *Post) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		fields.ManyToMany[*queries.RelM2M[*Tag, *PostTag]]("Tags", &fields.FieldConfig{
			Rel: attrs.Relate(
				&Tag{},
				"",
				&attrs.ThroughModel{
					This:   &PostTag{},
					Source: "PostRef",
					Target: "TagRef",
				},
			),
		}),
	)
}

type PostTag struct {
	models.Model
	PostRef *Post
	TagRef  *Tag
}

func (m *PostTag) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "PostRef", &attrs.Config{Column: "post_id"}),
		attrs.NewField(m, "TagRef", &attrs.Config{Column: "tag_id"}),
	)
}
```

`*queries.RelM2M` exposes `.Objects() *queries.RelManyToManyQuerySet[TargetModel]`:

```go
post := models.Setup(ctx, &Post{})
post, err := queries.CreateObject(post)

tag1 := &Tag{Name: "Golang"} // Zero PK -> auto-inserted by AddTarget/AddTargets/SetTargets
tag2, _ := queries.CreateObject(&Tag{Name: "ORM"})

created, err := post.Tags.Objects().WithContext(ctx).AddTarget(tag1)
addedCount, err := post.Tags.Objects().WithContext(ctx).AddTargets(tag2)
addedCount, err = post.Tags.Objects().WithContext(ctx).SetTargets([]*Tag{tag1, tag2})

removedCount, err := post.Tags.Objects().WithContext(ctx).RemoveTargets(tag1)
clearedCount, err := post.Tags.Objects().WithContext(ctx).ClearTargets()

rows, err := post.Tags.Objects().WithContext(ctx).OrderBy("Name").All()
```

### `ManyToManyQuerySet` Caveats

- **Automatic target creation:** `AddTarget`, `AddTargets`, `SetTargets`, and `RelOneToOneQuerySet.SetTarget` inspect each target's primary key. Any target with a zero primary key is automatically created via `BulkCreate` in the same transaction before creating the through-table rows.
- **`errors.NoChanges`:** `RemoveTargets` returns `errors.NoChanges` if 0 through rows were deleted, and `ClearTargets` returns `errors.NoChanges` if there were no linked through rows to clear. (`SetTargets` ignores `errors.NoChanges` from its internal `ClearTargets` call.)

---

## 4. Reverse Relations (`RelRevFK`)

When a forward relation model is registered, `queries` (via `attrs` and `fields.RelationField`) registers a reverse field on the target model unless `NoReverseRelation: true` is set:

| Forward Relation | Default Reverse Field Name | Override |
|---|---|---|
| `ForeignKey` (`RelManyToOne`) from `Todo` -> `User` | `TodoSet` (`<ModelName>Set`) | `FieldConfig.ReverseName` or `attrs.Config.ReverseAlias` |
| `ManyToMany` (`RelManyToMany`) from `Post` -> `Tag` | `PostSet` (`<ModelName>Set`) | `FieldConfig.ReverseName` |
| `OneToOne` (`RelOneToOne`) from `User` -> `Profile` | `UserReverse` (`<ModelName>Reverse`) | `FieldConfig.ReverseName` or `attrs.Config.ReverseAlias` |

Declare `*queries.RelRevFK[attrs.Definer]` on the target struct to hold reverse objects and query them with `.Objects() *queries.RelOneToManyQuerySet[T]`:

```go
type User struct {
	models.Model
	ID      uint64
	Name    string
	TodoSet *queries.RelRevFK[attrs.Definer]
}

// Eager-loaded reverse access:
row, err := queries.GetQuerySet(&User{}).
	WithContext(ctx).
	Select("ID", "Name", "TodoSet.*").
	Filter("ID", 1).
	Get()
for _, todo := range row.Object.TodoSet.AsList() {
	fmt.Println(todo)
}

// Lazy/dynamic reverse query:
user := models.Setup(ctx, &User{ID: 1})
todos, err := user.TodoSet.Objects().WithContext(ctx).OrderBy("ID").All()
```

---

## Querying Across Relations

### Dot Notation for Relation Traversal

Related fields always use **dot notation** (`.`), with `__` reserved for lookups and transforms on the final field:

```go
todos, err := queries.GetQuerySet(&Todo{}).
	WithContext(ctx).
	Filter("User.Name__startswith", "Alice").
	All()
```

Using `"User__Name__startswith"` will fail field resolution because `__` separates lookups/transforms rather than relation hops.

### Eager Loading: `Select`, `SelectRelated`, and `Preload`

1. **`Select`:** Loads specific columns or whole related models inline via `JOIN`s (for `ForeignKey` / `OneToOne`), or registers a preload for multi-valued relations:

   ```go
   todos, err := queries.GetQuerySet(&Todo{}).
   	WithContext(ctx).
   	Select("*", "User.*", "User.Profile.*").
   	All()
   ```

2. **`SelectRelated(fields ...string)`:** Adds `JOIN`s and selects all `ForSelectAll` columns for single-valued (`ForeignKey` and `OneToOne`) relation paths. Note that `SelectRelated` panics on `RelOneToMany` or `RelManyToMany` relations and modifies the `QuerySet` in place without cloning.

3. **`Preload(fields ...any)`:** Loads related objects via separate queries (required for custom query configuration on `OneToMany` / `ManyToMany`, and useful to avoid wide `JOIN`s). Accepts:
   - `string`: relation path such as `"TodoSet"` or `"Tags"`
   - `queries.NoJoins("TodoSet")`: ensures no parent `JOIN` is added for the preload
   - `queries.Preload{Path: "TodoSet", NoJoins: true, QuerySet: customQS}`: attaches a custom `*QuerySet[attrs.Definer]` to filter, order, or `SelectRelated` inside the preloaded query:

   ```go
   users, err := queries.GetQuerySet(&User{}).
   	WithContext(ctx).
   	Preload(queries.Preload{
   		Path:     "TodoSet",
   		NoJoins:  true,
   		QuerySet: queries.Objects[attrs.Definer](&Todo{}).Filter("Done", false).OrderBy("-ID"),
   	}).
   	All()
   ```

See [The QuerySet API](../queryset/queryset.md) and [Fields](../fields.md) for more details.
