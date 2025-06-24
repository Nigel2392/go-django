# Creating your models

Models are defined using the [`attrs`](https://github.com/Nigel2392/go-django/blob/main/docs/attrs/attrs.md) package.

They should be defined as structs, and should implement the `attrs.Definer` interface.

Models are automatically registered to Go-Django when they are inside of an [apps' `Models()` list](https://github.com/Nigel2392/go-django/blob/main/docs/attrs/attrs.md).

Each model using the queries package is best to embed the `models.Model` struct - though this is not a requirement.  
This will make sure to also setup fields for reverse relations, allow for storing annotations on the model itself
and provide some additional functionality.  
It also allows you to use a `table:"tablename"` tag to specify the table name for the model.

Read more about the [`models.Model`](./model.md) implementation in the [Model documentation](./model.md).

A short example of defining some models with a relationship (we will assume the tables already exist):

```go
type Profile struct {
    models.Model `table:"profiles"` // Embedding Model struct is optional, but recommended
    ID    int
    Name  string
    Email string
}

func (m *Profile) FieldDefs() attrs.Definitions {
    // Use the model to define the fields instead of `attrs.Define`
    return m.Model.Define(m,
        attrs.NewField(m, "ID", &attrs.FieldConfig{
            Primary:  true,
        }),
        attrs.NewField(m, "Name", nil),
        attrs.NewField(m, "Email", nil),
    )
}

type User struct {
    models.Model // Embedding Model struct is optional, but recommended
    ID      int
    Name    string
    Profile *Profile
}

func (m *User) FieldDefs() attrs.Definitions {
    // Use the model to define the fields instead of `attrs.Define`
    return m.Model.Define(m,
        attrs.NewField(m, "ID", &attrs.FieldConfig{
            Primary:  true,
        }),
        attrs.NewField(m, "Name", nil),
        attrs.NewField(m, "Profile", &attrs.FieldConfig{
            RelForeignKey: attrs.Relate(&Profile{}, "", nil),
            Column:        "profile_id",
        }),
    ).WithTableName("users")
}

type Todo struct {
    models.Model // Embedding Model struct is optional, but recommended
    ID          int
    Title       string
    Description string
    Done        bool
    User        *User
}

func (m *Todo) FieldDefs() attrs.Definitions {
    return m.Model.Define(m,
        attrs.NewField(m, "ID", &attrs.FieldConfig{
            Primary:  true,
            ReadOnly: true,
        }),
        attrs.NewField(m, "Title", nil),
        attrs.NewField(m, "Description", nil),
        attrs.NewField(m, "Done", nil),
        attrs.NewField(m, "User", &attrs.FieldConfig{
            RelForeignKey: attrs.Relate(&User{}, "", nil),
            Column:        "user_id",
        }),
    ).WithTableName("todos")
}
```

---

For more information on defining or registering models, see the [attrs documentation](https://github.com/Nigel2392/go-django/blob/main/docs/attrs/attrs.md).

Or continue with [Querying Objects](../querying.md)…
