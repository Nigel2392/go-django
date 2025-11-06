# Relations & Joins

This document covers how to define and work with relationships between models in the `go-django-queries` package.

The queries package supports relationships through field definitions and proxy models, enabling you to build complex queries across related data.

---

## 🔗 Types of Relations

### Foreign Key (Many-to-One)

A Foreign Key creates a many-to-one relationship where many instances of the current model can be related to one instance of another model.

```go
type User struct {
    models.Model
    ID       int64
    Name     string
    Email    string
}

type Todo struct {
    models.Model
    ID          int64
    Title       string
    Description string
    Done        bool
    User        *User  // Foreign Key to User
}

func (m *Todo) FieldDefs() attrs.Definitions {
    return m.Model.Define(m,
        attrs.Unbound("ID", &attrs.FieldConfig{Primary: true}),
        attrs.Unbound("Title"),
        attrs.Unbound("Description"),
        attrs.Unbound("Done"),
        attrs.Unbound("User"),
    )
}
```

### One-to-One

A One-to-One relationship links each instance of a model to exactly one instance of another model. This is commonly used for extending models with additional data.

```go
type User struct {
    models.Model
    ID       int64
    Name     string
    Email    string
}

type Profile struct {
    models.Model
    ID       int64
    User     *User  // One-to-One relationship with User
    Bio      string
    Location string
}

func (m *Profile) FieldDefs() attrs.Definitions {
    return m.Model.Define(m,
        attrs.Unbound("ID", &attrs.FieldConfig{Primary: true}),
        attrs.Unbound("User"),
        attrs.Unbound("Bio"),
        attrs.Unbound("Location"),
    )
}
```

### Proxy Models

Proxy models allow you to create relationships between models through embedded structures. This is useful for creating complex hierarchical models:

```go
type Page struct {
    models.Model
    ID          int64
    Title       string
    Description string
    PageID      int64
    PageContentType *contenttypes.BaseContentType[attrs.Definer]
}

type BlogPage struct {
    models.Model
    Proxy               *Page `proxy:"true"`
    PageID              int64
    Author              string
    Tags                []string
    Category            string
    CategoryContentType *contenttypes.BaseContentType[attrs.Definer]
}

func (b *BlogPage) FieldDefs() attrs.Definitions {
    return b.Model.Define(b,
        fields.Embed("Proxy"),
        attrs.Unbound("PageID", &attrs.FieldConfig{Primary: true}),
        attrs.Unbound("Author"),
        attrs.Unbound("Tags"),
        attrs.Unbound("Category"),
        attrs.Unbound("CategoryContentType"),
    )
}
```

---

## 🔍 Querying Relations

### Basic Filtering

Filter by related field values:

```go
// Get all todos for a specific user
todos, err := queries.GetQuerySet(&Todo{}).
    Filter("User", userID).
    All()

// Get all todos for users with specific email
todos, err := queries.GetQuerySet(&Todo{}).
    Filter("User__Email", "user@example.com").
    All()
```

### Using Expressions

Use expressions for more complex relationship queries:

```go
// Filter using expressions
todos, err := queries.GetQuerySet(&Todo{}).
    Filter(expr.Q("User", userID)).
    All()

// Complex filtering with AND/OR
todos, err := queries.GetQuerySet(&Todo{}).
    Filter(expr.Or(
        expr.Q("User", userID),
        expr.Q("Done", true),
    )).
    All()
```

### Joins and Selecting

Select related data to avoid N+1 queries:

```go
// Select related user data
todos, err := queries.GetQuerySet(&Todo{}).
    Select("*", "User.*").
    Filter("Done", false).
    All()

// Access related data
for _, todo := range todos {
    if todo.User != nil {
        fmt.Printf("Todo: %s, User: %s\n", todo.Title, todo.User.Name)
    }
}
```

---

## 📊 Advanced Relationship Queries

### Aggregating Related Data

Count related objects:

```go
// Count todos per user (using annotations)
users, err := queries.GetQuerySet(&User{}).
    Annotate("TodoCount", expr.Count("ID")).
    All()
```

### Complex Filtering

Filter based on related object properties:

```go
// Get users who have completed todos
users, err := queries.GetQuerySet(&User{}).
    Filter(expr.Q("Todo__Done", true)).
    Distinct().
    All()
```

### Subqueries

Use subqueries for complex relationship filtering:

```go
// Get todos from active users
activeTodos, err := queries.GetQuerySet(&Todo{}).
    Filter("User", queries.Subquery(
        queries.GetQuerySet(&User{}).
            Filter("IsActive", true).
            Values("ID"),
    )).
    All()
```

---

## 🎯 Best Practices

### Performance Optimization

1. **Use Select Related**: Always use `Select()` to load related data in one query
2. **Avoid N+1 Queries**: Load related data upfront rather than in loops
3. **Use Indexes**: Create database indexes on foreign key fields

### Code Organization

1. **Clear Relationships**: Make relationships explicit in model definitions
2. **Consistent Naming**: Use consistent naming conventions for related fields
3. **Documentation**: Document complex relationships in code comments

### Testing

```go
func TestUserTodos(t *testing.T) {
    // Create test user
    user := &User{Name: "Test User", Email: "test@example.com"}
    _, err := queries.GetQuerySet(user).Create(user)
    require.NoError(t, err)
    
    // Create test todos
    todos := []*Todo{
        {Title: "Todo 1", Done: false, User: user},
        {Title: "Todo 2", Done: true, User: user},
    }
    _, err = queries.GetQuerySet(&Todo{}).BulkCreate(todos)
    require.NoError(t, err)
    
    // Test relationship query
    userTodos, err := queries.GetQuerySet(&Todo{}).
        Filter("User", user.ID).
        All()
    require.NoError(t, err)
    assert.Len(t, userTodos, 2)
}
```

---

## 🔧 Advanced Features

### Generic Relations

Use content types for generic foreign keys:

```go
type Comment struct {
    models.Model
    ID              int64
    Content         string
    ContentType     *contenttypes.BaseContentType[attrs.Definer]
    ObjectID        int64
}

func (c *Comment) FieldDefs() attrs.Definitions {
    return c.Model.Define(c,
        attrs.Unbound("ID", &attrs.FieldConfig{Primary: true}),
        attrs.Unbound("Content"),
        attrs.Unbound("ContentType"),
        attrs.Unbound("ObjectID"),
    )
}
```

### Proxy Field Relations

Create complex join conditions using proxy fields:

```go
type BlogPageCategory struct {
    models.Model
    *BlogPage
    Category string
}

func (b *BlogPageCategory) FieldDefs() attrs.Definitions {
    return b.Model.Define(b,
        fields.Embed("BlogPage"),
        attrs.Unbound("Category", &attrs.FieldConfig{Primary: true}),
    )
}
```

This comprehensive guide covers the relationship patterns used in the go-django-queries package. All examples are based on actual test patterns and should work with the current implementation.

```go
type Profile struct {
    models.Model
    ID    int
    Name  string
    Email string
    User  *User  // One-to-One relationship
}

func (m *Profile) FieldDefs() attrs.Definitions {
    return m.Model.Define(m,
        attrs.NewField(m, "ID", &attrs.FieldConfig{
            Primary: true,
        }),
        attrs.NewField(m, "Name", nil),
        attrs.NewField(m, "Email", nil),
        attrs.NewField(m, "User", &attrs.FieldConfig{
            RelOneToOne: attrs.Relate(&User{}, "", nil),
            Column:      "user_id",
        }),
    ).WithTableName("profiles")
}
```

### One-to-Many (Reverse Foreign Key)

When you define a Foreign Key, the related model automatically gets a reverse One-to-Many relationship.

```go
// User model automatically has a reverse relationship to Todo
// This is accessible via the model's reverse relations
```

### Many-to-Many

A Many-to-Many relationship allows multiple instances of one model to be related to multiple instances of another model.

```go
type Author struct {
    models.Model
    ID    int
    Name  string
    Books []*Book  // Many-to-Many relationship
}

type Book struct {
    models.Model
    ID      int
    Title   string
    Authors []*Author  // Many-to-Many relationship
}

func (m *Author) FieldDefs() attrs.Definitions {
    return m.Model.Define(m,
        attrs.NewField(m, "ID", &attrs.FieldConfig{
            Primary: true,
        }),
        attrs.NewField(m, "Name", nil),
        attrs.NewField(m, "Books", &attrs.FieldConfig{
            RelManyToMany: attrs.Relate(&Book{}, "authors", nil),
        }),
    ).WithTableName("authors")
}

func (m *Book) FieldDefs() attrs.Definitions {
    return m.Model.Define(m,
        attrs.NewField(m, "ID", &attrs.FieldConfig{
            Primary: true,
        }),
        attrs.NewField(m, "Title", nil),
        attrs.NewField(m, "Authors", &attrs.FieldConfig{
            RelManyToMany: attrs.Relate(&Author{}, "books", nil),
        }),
    ).WithTableName("books")
}
```

### Many-to-Many with Through Model

Sometimes you need to store additional information about the relationship itself. This is done using a "through" model.

```go
type Author struct {
    models.Model
    ID    int
    Name  string
    Books []*Book  // Many-to-Many through AuthorBook
}

type Book struct {
    models.Model
    ID      int
    Title   string
    Authors []*Author  // Many-to-Many through AuthorBook
}

type AuthorBook struct {
    models.Model
    ID       int
    Author   *Author
    Book     *Book
    Role     string     // Additional field: author's role in the book
    OrderBy  int        // Additional field: order of authorship
}

func (m *Author) FieldDefs() attrs.Definitions {
    return m.Model.Define(m,
        attrs.NewField(m, "ID", &attrs.FieldConfig{
            Primary: true,
        }),
        attrs.NewField(m, "Name", nil),
        attrs.NewField(m, "Books", &attrs.FieldConfig{
            RelManyToMany: attrs.Relate(&Book{}, "authors", &AuthorBook{}),
        }),
    ).WithTableName("authors")
}

func (m *AuthorBook) FieldDefs() attrs.Definitions {
    return m.Model.Define(m,
        attrs.NewField(m, "ID", &attrs.FieldConfig{
            Primary: true,
        }),
        attrs.NewField(m, "Author", &attrs.FieldConfig{
            RelForeignKey: attrs.Relate(&Author{}, "", nil),
            Column:        "author_id",
        }),
        attrs.NewField(m, "Book", &attrs.FieldConfig{
            RelForeignKey: attrs.Relate(&Book{}, "", nil),
            Column:        "book_id",
        }),
        attrs.NewField(m, "Role", nil),
        attrs.NewField(m, "OrderBy", nil),
    ).WithTableName("author_books")
}
```

---

## 📝 Querying Relations

### Forward Relations

Forward relations are accessed by referencing the field name directly:

```go
// Get all todos with their related user
todos, err := queries.GetQuerySet(&Todo{}).
    Select("*", "User.*").
    All()

// Filter by related field
todos, err := queries.GetQuerySet(&Todo{}).
    Filter("User.Name", "John").
    All()

// Order by related field
todos, err := queries.GetQuerySet(&Todo{}).
    OrderBy("User.Name").
    All()
```

### Reverse Relations

Reverse relations are accessed using the model name followed by "Set":

```go
// Get all users with their related todos
users, err := queries.GetQuerySet(&User{}).
    Select("*", "TodoSet.*").
    All()

// Filter by reverse relation
users, err := queries.GetQuerySet(&User{}).
    Filter("TodoSet.Done", false).
    All()

// Count related objects
users, err := queries.GetQuerySet(&User{}).
    Annotate("TodoCount", expr.Count("TodoSet.ID")).
    All()
```

### Many-to-Many Relations

```go
// Get all authors with their books
authors, err := queries.GetQuerySet(&Author{}).
    Select("*", "Books.*").
    All()

// Filter by many-to-many relation
authors, err := queries.GetQuerySet(&Author{}).
    Filter("Books.Title__contains", "Harry Potter").
    All()

// Get books with their authors
books, err := queries.GetQuerySet(&Book{}).
    Select("*", "Authors.*").
    All()
```

### Through Model Relations

When using through models, you can access both the related object and the through model:

```go
// Get authors with books and the through model information
authors, err := queries.GetQuerySet(&Author{}).
    Select("*", "Books.*", "BooksThrough.*").
    All()

// Filter by through model fields
authors, err := queries.GetQuerySet(&Author{}).
    Filter("BooksThrough.Role", "Primary Author").
    All()

// Order by through model fields
authors, err := queries.GetQuerySet(&Author{}).
    OrderBy("BooksThrough.OrderBy").
    All()
```

---

## 🔍 Advanced Querying

### Nested Relations

You can access nested relations using dot notation:

```go
// Get todos with user and user's profile
todos, err := queries.GetQuerySet(&Todo{}).
    Select("*", "User.*", "User.Profile.*").
    All()

// Filter by nested relation
todos, err := queries.GetQuerySet(&Todo{}).
    Filter("User.Profile.Email__contains", "@example.com").
    All()
```

### Joins

The queries package automatically generates appropriate JOIN clauses based on the selected fields:

```go
// This generates an INNER JOIN to User table
todos, err := queries.GetQuerySet(&Todo{}).
    Select("*", "User.*").
    All()

// This generates a LEFT JOIN to User table
todos, err := queries.GetQuerySet(&Todo{}).
    Select("*", "User.*").
    Filter("User.ID__isnull", true).
    All()
```

### Prefetch Relations

For better performance when accessing multiple related objects:

```go
// Prefetch all related users in a single query
todos, err := queries.GetQuerySet(&Todo{}).
    Select("*", "User.*").
    All()

// Access the related user without additional queries
for _, row := range todos {
    todo := row.Value()
    user := todo.User // Already loaded, no additional query
}
```

### Aggregations on Relations

```go
// Count related objects
users, err := queries.GetQuerySet(&User{}).
    Annotate("TodoCount", expr.Count("TodoSet.ID")).
    All()

// Sum related fields
users, err := queries.GetQuerySet(&User{}).
    Annotate("CompletedTodos", expr.Count("TodoSet.ID")).
    Filter("TodoSet.Done", true).
    All()

// Average of related fields
categories, err := queries.GetQuerySet(&Category{}).
    Annotate("AvgTodos", expr.Avg("TodoSet.ID")).
    All()
```

---

## 💡 Best Practices

### Performance Considerations

1. **Select Only What You Need**: Use `Select()` to specify which fields and relations to load
2. **Avoid N+1 Queries**: Use relations in `Select()` to prefetch related data
3. **Use Aggregations**: Instead of loading all related objects, use aggregations when you only need counts or sums

```go
// Good: Load user data with todos in one query
users, err := queries.GetQuerySet(&User{}).
    Select("*", "TodoSet.*").
    All()

// Bad: This will cause N+1 queries
users, err := queries.GetQuerySet(&User{}).All()
for _, row := range users {
    user := row.Value()
    // This causes an additional query for each user
    todos, _ := queries.GetQuerySet(&Todo{}).Filter("User", user.ID).All()
}
```

### Naming Conventions

1. **Field Names**: Use the related model name for the field (e.g., `User *User`)
2. **Reverse Relations**: Accessed as `ModelNameSet` (e.g., `TodoSet`)
3. **Through Models**: Accessed as `RelationNameThrough` (e.g., `BooksThrough`)

### Error Handling

```go
// Always check for errors when querying relations
todos, err := queries.GetQuerySet(&Todo{}).
    Select("*", "User.*").
    All()
if err != nil {
    // Handle error appropriately
    return fmt.Errorf("failed to fetch todos with users: %w", err)
}
```

---

## 🧪 Testing Relations

When testing models with relations, make sure to:

1. Create test data for all related models
2. Test both forward and reverse relations
3. Test filtering and ordering by related fields
4. Test aggregations on related fields

```go
func TestTodoUserRelation(t *testing.T) {
    // Create test user
    user := &User{Name: "Test User", Email: "test@example.com"}
    err := queries.CreateObject(user)
    assert.NoError(t, err)

    // Create test todo
    todo := &Todo{
        Title: "Test Todo",
        User:  user,
    }
    err = queries.CreateObject(todo)
    assert.NoError(t, err)

    // Test forward relation
    todos, err := queries.GetQuerySet(&Todo{}).
        Select("*", "User.*").
        Filter("ID", todo.ID).
        All()
    assert.NoError(t, err)
    assert.Len(t, todos, 1)
    assert.Equal(t, user.Name, todos[0].Value().User.Name)

    // Test reverse relation
    users, err := queries.GetQuerySet(&User{}).
        Select("*", "TodoSet.*").
        Filter("ID", user.ID).
        All()
    assert.NoError(t, err)
    assert.Len(t, users, 1)
    // Access todos through reverse relation
    userData := users[0].Value().DataStore()
    todoSet, exists := userData.GetValue("TodoSet")
    assert.True(t, exists)
    // todoSet should contain the related todos
}
```

---

Continue with [QuerySet Reference](../queryset/queryset.md) for more advanced querying techniques…