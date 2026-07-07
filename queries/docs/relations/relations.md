# Model Relations

Relations between models in the `queries` package are declared in `FieldDefs()` using either `attrs.FieldConfig` (for quick inline relations) or the typed shortcut constructors in the `fields` package.

> **Important:** Reverse relations are only automatically registered when the model is registered with `attrs.RegisterModel` — usually done via the app's `Models()` list. The field on the reverse side is injected into the model's metadata and is named after the forward model's type by default (e.g. `TodoSet`), or by the value of `attrs.AttrReverseAliasKey` in the field's `Attributes` map.

---

## Accepted struct field types

The struct field type for a relational field must be a **pointer to the target model**. For reverse (one-to-many / many-to-many) fields that hold a collection, use `*queries.RelRevFK[attrs.Definer]`:

| Relation             | Struct field type       |
|----------------------|-------------------------|
| ForeignKey (M:1)     | `*TargetModel`          |
| OneToOne             | `*TargetModel` (without through) or `*queries.RelO2O[*Target, *Through]` |
| ManyToMany           | `*queries.RelM2M[*TargetModel, *ThroughModel]` |
| Reverse FK / O2O     | `*queries.RelRevFK[attrs.Definer]` |

---

## 1. Foreign Key (Many-to-One)

A ForeignKey is a many-to-one relation. The struct field is a pointer to the related model. Specify `RelForeignKey` and a `Column` name:

```go
type Todo struct {
    models.Model `table:"queries-todos"`
    ID           int
    Title        string
    User         *User
}

func (m *Todo) FieldDefs() attrs.Definitions {
    return m.Model.Define(m,
        attrs.NewField(m, "ID", &attrs.FieldConfig{Primary: true, ReadOnly: true}),
        attrs.NewField(m, "Title", nil),
        attrs.NewField(m, "User", &attrs.FieldConfig{
            Null:          true,
            Column:        "user_id",
            RelForeignKey: attrs.Relate(&User{}, "", nil),
        }),
    )
}
```

Shortcut:

```go
fields.ForeignKey[*User]("User", "user_id")
```

### Creating ForeignKey Relations

To create a ForeignKey relation, you typically set the related model pointer (or just its primary key) on the object and save it. If you want to use the reverse relation (the `TodoSet`) on the `User` *without* fetching it from the database first, you **must** use `models.Setup()`.

```go
// 1. Create the parent object
var user = &User{Name: "Alice"}
queries.CreateObject(user)

// 2. Create the child object with the relation
var todo = &Todo{
    Title: "Buy milk",
    User:  user, // Link the user directly
}
queries.CreateObject(todo)

// 3. Alternatively, link by ID if you only have the ID
var todo2 = &Todo{
    Title: "Buy bread",
    User:  &User{ID: user.ID},
}
queries.CreateObject(todo2)
```

> **🚨 CRITICAL: `models.Setup()`**
> When you instantiate a struct manually in Go (e.g., `user := &User{}`), its reverse relation fields (like `*queries.RelRevFK` or `*queries.RelM2M`) are `nil`.
>
> If you want to interact with `user.TodoSet.Objects()` *before* or *without* saving the user via `CreateObject` (which sets it up for you), you **must** wrap it in `models.Setup(&User{})` to initialize those fields.

---

## 2. One-to-One

A one-to-one relation is configured with `RelOneToOne`.

### Without a Through Model

```go
type User struct {
    models.Model
    ID      uint64
    Name    string
    Profile *Profile
}

func (m *User) FieldDefs() attrs.Definitions {
    return m.Model.Define(m,
        attrs.NewField(m, "ID", &attrs.FieldConfig{Primary: true, ReadOnly: true}),
        attrs.NewField(m, "Name", nil),
        attrs.NewField(m, "Profile", &attrs.FieldConfig{
            Null:        true,
            RelOneToOne: attrs.Relate(&Profile{}, "", nil),
            Column:      "profile_id",
        }),
    ).WithTableName("queries-users")
}
```

Shortcut:

```go
fields.OneToOne[*Profile]("Profile", &fields.FieldConfig{
    ColumnName: "profile_id",
})
```

### With a Through Model

A One-to-One relation can also use a junction/through table. The struct field type must be `*queries.RelO2O[*Target, *Through]`:

```go
type Document struct {
    models.Model
    ID      int64
    Target  *queries.RelO2O[*TargetDoc, *DocLink]
}

func (d *Document) FieldDefs() attrs.Definitions {
    return d.Model.Define(d,
        attrs.NewField(d, "ID", &attrs.FieldConfig{Primary: true}),
        fields.OneToOne[*queries.RelO2O[*TargetDoc, *DocLink]]("Target", &fields.FieldConfig{
            Rel: attrs.Relate(
                &TargetDoc{},
                "", &attrs.ThroughModel{
                    This:   &DocLink{},
                    Source: "SourceModel", // matches DocLink.SourceModel
                    Target: "TargetModel", // matches DocLink.TargetModel
                },
            ),
        }),
    )
}

// The through model itself
type DocLink struct {
    models.Model
    SourceModel *Document    // source
    TargetModel *TargetDoc   // target
}

func (d *DocLink) FieldDefs() attrs.Definitions {
    return d.Model.Define(d,
        attrs.NewField(d, "SourceModel", &attrs.FieldConfig{Column: "source_id"}),
        attrs.NewField(d, "TargetModel", &attrs.FieldConfig{Column: "target_id"}),
    )
}
```

### Creating Through Model Relations

When a relation uses a through model (like this OneToOne, or a manual ManyToMany), you must explicitly save a new instance of the through model to establish the link:

```go
// 1. Create the target
var target = &TargetDoc{Name: "Intro"}
queries.CreateObject(target)

// 2. Create the main object
var doc = &Document{}
queries.CreateObject(doc)

// 3. Manually link the objects using the through model
var link = &DocLink{
    SourceModel: doc,
    TargetModel: target,
}
queries.CreateObject(link)

// Querying the object will now include the through relation
var result, _ = queries.GetQuerySet(&Document{}).
    Select("ID", "Target.*").
    Filter("ID", doc.ID).
    First()

fmt.Println(result.Object.Target.GetValue()) // returns (targetObj, throughObj)
```

---

## 3. Many-to-Many

A many-to-many relation **always** requires a `Through` table. The struct field type must be `*queries.RelM2M[*TargetModel, *ThroughModel]`.

Pass the through model via `attrs.Relate` or via the `fields.FieldConfig`:

```go
type Post struct {
    models.Model
    ID   int
    Tags *queries.RelM2M[*Tag, *PostTag] // Uses RelM2M for ManyToMany
}

func (m *Post) FieldDefs() attrs.Definitions {
    return m.Model.Define(m,
        attrs.NewField(m, "ID", &attrs.FieldConfig{Primary: true}),
        fields.ManyToMany[*queries.RelM2M[*Tag, *PostTag]]("Tags", &fields.FieldConfig{
            Rel: attrs.Relate(
                &Tag{},
                "", &attrs.ThroughModel{
                    This:   &PostTag{},
                    Source: "PostRef", // maps to PostTag.PostRef
                    Target: "TagRef",  // maps to PostTag.TagRef
                },
            ),
        }),
    )
}

// The through model
type PostTag struct {
    models.Model
    PostRef *Post
    TagRef  *Tag
}

func (m *PostTag) FieldDefs() attrs.Definitions {
    return m.Model.Define(m,
        attrs.NewField(m, "PostRef", &attrs.FieldConfig{Column: "post_id"}),
        attrs.NewField(m, "TagRef", &attrs.FieldConfig{Column: "tag_id"}),
    )
}
```

> **Note:** A ManyToMany relation will panic during setup if no `Through` relation is defined.

### Creating Many-to-Many Relations

You can create ManyToMany relations either by explicitly creating the junction table struct (as shown in the OneToOne section), or by using the helper methods on the relation's `Objects()` queryset.

Because `Tags` is a `*queries.RelM2M` field, it must be initialized before you can call `.Objects()` on it. If the object was fetched from the DB or just saved via `CreateObject`, it is ready to use. If it is a fresh struct in memory, you must use `models.Setup()`:

```go
// models.Setup initializes the Tags RelM2M field so it is not nil!
var post = models.Setup(&Post{})
queries.CreateObject(post)

var tag = &Tag{Name: "Golang"}
queries.CreateObject(tag)

// AddTarget creates the through object automatically
created, err := post.Tags.Objects().AddTarget(tag)

// Or add multiple existing targets
post.Tags.Objects().AddTargets(tag1, tag2, tag3)

// You can also sync the entire list (clears existing, adds new)
post.Tags.Objects().SetTargets([]*Tag{tag1, tag2})

// Remove relations
post.Tags.Objects().RemoveTargets(tag1)
post.Tags.Objects().ClearTargets()

// Querying the targets
var tags, _ = post.Tags.Objects().All()
```

---

## 4. Reverse Relations (auto-generated)

When a model is registered with `attrs.RegisterModel`, the `queries` package automatically injects a reverse relation field on the related model.

- ForeignKey `Todo → User` produces a `TodoSet` reverse field on `User` (type `*queries.RelRevFK[attrs.Definer]`)
- OneToOne `User → Profile` produces a `User` reverse field on `Profile`

To hold the reverse relation value on the struct, declare the field with `*queries.RelRevFK[attrs.Definer]`:

```go
type User struct {
    models.Model
    ID                 uint64
    Name               string
    TodoSet            *queries.RelRevFK[attrs.Definer]
}
```

### Working with Reverse Relations

Reverse sets can be queried using `.Objects()` to retrieve the related objects dynamically from the database.

Just like M2M fields, reverse sets must be initialized via `models.Setup()` if you are instantiating the struct manually and want to use the relation before saving.

```go
// Fetch the user from the database (relation is auto-initialized)
var row, _ = queries.GetQuerySet(&User{}).
    Select("ID", "Name", "TodoSet.*"). // eager load the relation
    Filter("ID", 1).
    Get()
var user = row.Object

// Access already eager-loaded models
for _, todo := range user.TodoSet.AsList() {
    fmt.Println(todo)
}

// Or query the reverse relation dynamically using its related queryset
var allTodos, _ = user.TodoSet.Objects().OrderBy("ID").All()
```

---

## Querying Across Relations

Use double-underscore (`__`) notation to filter across related models:

```go
// Fetch all todos where the related user's name starts with "Alice"
var todos, err = queries.GetQuerySet(&Todo{}).
    WithContext(ctx).
    Filter("User__Name__startswith", "Alice").
    All()
```

To eagerly load a related object in the same query, use `Select`:

```go
// Fetch todos and also load the full User object
var todos, err = queries.GetQuerySet(&Todo{}).
    WithContext(ctx).
    Select("*", "User.*").
    All()

// Access the loaded relation
fmt.Println(todos[0].Object.User.Name)
```

To also load nested relations:

```go
// Load Todo → User → Profile
queries.GetQuerySet(&Todo{}).
    Select("*", "User.*", "User.Profile.*").
    All()
```
