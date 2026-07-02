# Writing Queries

This guide covers practical techniques for writing queries using the `go-django-queries` package.

Building on the [QuerySet Reference](./queryset.md), this document explores patterns for constructing database queries based on actual usage patterns from the test suite.

---

## 🏗️ Query Construction Patterns

### Basic Query Building

Start with simple queries and build complexity incrementally:

```go
// Start with a base queryset
qs := queries.GetQuerySet(&User{})

// Add filters progressively
qs = qs.Filter("IsActive", true)
qs = qs.Filter("CreatedAt__gte", time.Now().AddDate(0, -1, 0))

// Add ordering and limits
qs = qs.OrderBy("-CreatedAt")
qs = qs.Limit(10)

// Execute the query
users, err := qs.All()
```

### Method Chaining

Use method chaining for more readable query construction:

```go
users, err := queries.GetQuerySet(&User{}).
    Filter("IsActive", true).
    Filter("Email__contains", "@example.com").
    OrderBy("-CreatedAt").
    Limit(20).
    All()
```

### Conditional Query Building

Build queries dynamically based on conditions:

```go
func GetUsersWithFilters(country string, minAge int, orderBy string) ([]*User, error) {
    qs := queries.GetQuerySet(&User{})
    
    if country != "" {
        qs = qs.Filter("Country", country)
    }
    
    if minAge > 0 {
        qs = qs.Filter("Age__gte", minAge)
    }
    
    if orderBy != "" {
        qs = qs.OrderBy(orderBy)
    }
    
    return qs.All()
}
```

---

## 🔍 Expression-Based Filtering

### Using Q Expressions

Use `expr.Q()` for complex queries:

```go
// Simple Q expression
users, err := queries.GetQuerySet(&User{}).
    Filter(expr.Q("IsActive", true)).
    All()

// Complex filtering with nested conditions
users, err := queries.GetQuerySet(&User{}).
    Filter(expr.Q("Age__gte", 18)).
    Filter(expr.Q("Country", "US")).
    All()
```

### Logical Operators

Use explicit logical operators for complex conditions:

```go
// OR conditions - use explicit expr.Or()
users, err := queries.GetQuerySet(&User{}).
    Filter(expr.Or(
        expr.Q("Country", "US"),
        expr.Q("Country", "CA"),
    )).
    All()

// AND conditions - use explicit expr.And()
users, err := queries.GetQuerySet(&User{}).
    Filter(expr.And(
        expr.Q("IsActive", true),
        expr.Q("Age__gte", 18),
    )).
    All()

// Complex nested conditions
users, err := queries.GetQuerySet(&User{}).
    Filter(expr.And(
        expr.Q("IsActive", true),
        expr.Or(
            expr.Q("Role", "admin"),
            expr.Q("Role", "moderator"),
        ),
    )).
    All()
```

### Field Expressions

Use field expressions for comparisons between fields:

```go
// Compare fields using expr.Expr
users, err := queries.GetQuerySet(&User{}).
    Filter(expr.Expr("CreatedAt", expr.LOOKUP_LT, "UpdatedAt")).
    All()

// Bitwise operations
pages, err := queries.GetQuerySet(&Page{}).
    Filter(expr.Expr("StatusFlags", expr.LOOKUP_BITAND, StatusFlagPublished)).
    All()
```

---

## 📊 Aggregations and Annotations

### Basic Aggregations

Use aggregation functions for calculations:

```go
// Count records
count, err := queries.GetQuerySet(&User{}).
    Filter("IsActive", true).
    Count()

// Aggregate specific fields
avgAge, err := queries.GetQuerySet(&User{}).
    Filter("IsActive", true).
    Aggregate("Age", "avg")
```

### Annotations

Add computed fields to your queries:

```go
// Annotate with count
users, err := queries.GetQuerySet(&User{}).
    Annotate("PostCount", expr.Count("ID")).
    All()

// Access annotated fields
for _, user := range users {
    fmt.Printf("User: %s, Posts: %d\n", user.Name, user.PostCount)
}
```

### Case Expressions

Use case expressions for conditional logic:

```go
// Simple case expression
users, err := queries.GetQuerySet(&User{}).
    Annotate("UserType", 
        expr.Case(
            expr.When(expr.Q("IsAdmin", true), "Admin"),
            expr.When(expr.Q("IsModerator", true), "Moderator"),
            expr.Value("Regular"),
        ),
    ).
    All()
```

---

## 🔧 Advanced Query Techniques

### Raw SQL Expressions

Use raw SQL for complex operations:

```go
// Raw SQL expression
users, err := queries.GetQuerySet(&User{}).
    Filter(expr.Raw("AGE > ?", 18)).
    All()

// Raw field expression
users, err := queries.GetQuerySet(&User{}).
    Update(&User{}, 
        expr.F("![UpdatedAt] = CURRENT_TIMESTAMP"),
    )
```

### Subqueries

Use subqueries for complex filtering:

```go
// Subquery for filtering
activeUsers, err := queries.GetQuerySet(&User{}).
    Filter("ID", queries.Subquery(
        queries.GetQuerySet(&Session{}).
            Filter("IsActive", true).
            Values("UserID"),
    )).
    All()
```

### Exists Queries

Check for existence of related records:

```go
// Users with posts
usersWithPosts, err := queries.GetQuerySet(&User{}).
    Filter(expr.Exists(
        queries.GetQuerySet(&Post{}).
            Filter("Author", expr.OuterRef("ID")),
    )).
    All()
```

---

## 🎯 Query Optimization

### Select Related

Always select related data to avoid N+1 queries:

```go
// Load related data in one query
posts, err := queries.GetQuerySet(&Post{}).
    Select("*", "Author.*", "Category.*").
    All()
```

### Prefetch Related

Use prefetch for reverse relationships:

```go
// Prefetch related objects
users, err := queries.GetQuerySet(&User{}).
    Prefetch("Posts").
    All()
```

### Distinct

Remove duplicate results:

```go
// Get unique users who have posts
users, err := queries.GetQuerySet(&User{}).
    Filter(expr.Q("Posts__ID__isnull", false)).
    Distinct().
    All()
```

### Indexes and Database Hints

Use database-specific optimizations:

```go
// Use index hints (database-specific)
users, err := queries.GetQuerySet(&User{}).
    Filter("Email", "user@example.com").
    Extra("USE INDEX (email_idx)").
    All()
```

---

## 🧪 Testing Strategies

### Test Data Setup

Create reusable test data:

```go
func setupTestData(t *testing.T) (*User, []*Post) {
    user := &User{Name: "Test User", Email: "test@example.com"}
    createdUser, err := queries.GetQuerySet(user).Create(user)
    require.NoError(t, err)
    
    posts := []*Post{
        {Title: "Post 1", Author: createdUser},
        {Title: "Post 2", Author: createdUser},
    }
    createdPosts, err := queries.GetQuerySet(&Post{}).BulkCreate(posts)
    require.NoError(t, err)
    
    return createdUser, createdPosts
}
```

### Query Testing

Test complex queries thoroughly:

```go
func TestComplexQuery(t *testing.T) {
    user, posts := setupTestData(t)
    
    // Test complex filtering
    results, err := queries.GetQuerySet(&Post{}).
        Filter(expr.And(
            expr.Q("Author", user.ID),
            expr.Q("Title__contains", "Post"),
        )).
        OrderBy("Title").
        All()
    
    require.NoError(t, err)
    assert.Len(t, results, 2)
    assert.Equal(t, "Post 1", results[0].Title)
}
```

### Performance Testing

Test query performance:

```go
func BenchmarkComplexQuery(b *testing.B) {
    setupTestData(b)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := queries.GetQuerySet(&Post{}).
            Select("*", "Author.*").
            Filter("Published", true).
            OrderBy("-CreatedAt").
            Limit(100).
            All()
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

---

## 📚 Real-World Examples

### User Management

```go
// Get active users with recent activity
activeUsers, err := queries.GetQuerySet(&User{}).
    Filter(expr.And(
        expr.Q("IsActive", true),
        expr.Q("LastLogin__gte", time.Now().AddDate(0, 0, -30)),
    )).
    OrderBy("-LastLogin").
    Limit(100).
    All()
```

### Content Management

```go
// Get published posts by category
posts, err := queries.GetQuerySet(&Post{}).
    Filter(expr.And(
        expr.Q("Published", true),
        expr.Q("Category", category),
    )).
    Select("*", "Author.*").
    OrderBy("-PublishedAt").
    All()
```

### Analytics

```go
// User engagement analytics
stats, err := queries.GetQuerySet(&User{}).
    Annotate("PostCount", expr.Count("Posts")).
    Annotate("CommentCount", expr.Count("Comments")).
    Filter(expr.Q("PostCount__gt", 0)).
    OrderBy("-PostCount").
    All()
```

This guide provides practical patterns for writing queries in the go-django-queries package, all based on actual usage patterns from the test suite.
    qs := queries.GetQuerySet(&User{}).Select("*", "Profile.*")
    
    if country != "" {
        qs = qs.Filter("Profile.Country", country)
    }
    
    if minAge > 0 {
        qs = qs.Filter("Profile.Age__gte", minAge)
    }
    
    if orderBy != "" {
        qs = qs.OrderBy(orderBy)
    } else {
        qs = qs.OrderBy("-CreatedAt")
    }
    
    return qs.All()
}
```

---

## 🔍 Advanced Filtering

### Complex Conditions with Expressions

Use expressions for complex filtering conditions:

```go
// Using Q expressions for complex conditions
users, err := queries.GetQuerySet(&User{}).
    Filter(expr.Q("Age__gte", 18).And(expr.Q("Country", "US").Or(expr.Q("Country", "CA")))).
    All()

// Using raw expressions
users, err := queries.GetQuerySet(&User{}).
    Filter(expr.Raw("UPPER(name) LIKE ?", "%JOHN%")).
    All()
```

### Multiple Filter Conditions

Combine multiple filters effectively:

```go
// All conditions must be true (AND)
users, err := queries.GetQuerySet(&User{}).
    Filter("IsActive", true).
    Filter("Age__gte", 18).
    Filter("Country__in", []string{"US", "CA", "UK"}).
    All()

// Using OR conditions
users, err := queries.GetQuerySet(&User{}).
    Filter(expr.Q("Country", "US").Or(expr.Q("Age__gte", 21))).
    All()
```

### Filtering by Related Models

Filter by fields in related models:

```go
// Filter by foreign key relation
todos, err := queries.GetQuerySet(&Todo{}).
    Filter("User.IsActive", true).
    Filter("User.Profile.Country", "US").
    All()

// Filter by reverse relation
users, err := queries.GetQuerySet(&User{}).
    Filter("TodoSet.Done", false).
    Filter("TodoSet.Priority", "high").
    All()
```

### Date and Time Filtering

Work with date and time fields:

```go
// Date range queries
now := time.Now()
lastMonth := now.AddDate(0, -1, 0)

todos, err := queries.GetQuerySet(&Todo{}).
    Filter("CreatedAt__gte", lastMonth).
    Filter("CreatedAt__lte", now).
    All()

// Date component queries
todos, err := queries.GetQuerySet(&Todo{}).
    Filter("CreatedAt__year", 2023).
    Filter("CreatedAt__month", 12).
    All()
```

---

## 📊 Aggregations and Annotations

### Basic Aggregations

Use aggregations to compute summary statistics:

```go
// Count total users
count, err := queries.GetQuerySet(&User{}).Count()

// Aggregate multiple values
stats, err := queries.GetQuerySet(&Todo{}).
    Aggregate(map[string]expr.Expression{
        "total_count":      expr.Count("ID"),
        "completed_count":  expr.Count("ID", expr.Q("Done", true)),
        "avg_priority":     expr.Avg("Priority"),
        "max_created_at":   expr.Max("CreatedAt"),
    })
```

### Annotations

Add computed fields to your results:

```go
// Annotate with calculated fields
users, err := queries.GetQuerySet(&User{}).
    Annotate("TodoCount", expr.Count("TodoSet.ID")).
    Annotate("CompletedTodos", expr.Count("TodoSet.ID", expr.Q("TodoSet.Done", true))).
    Annotate("CompletionRate", expr.Div(
        expr.Count("TodoSet.ID", expr.Q("TodoSet.Done", true)),
        expr.Count("TodoSet.ID"),
    )).
    All()
```

### Complex Annotations

Create sophisticated calculated fields:

```go
// Complex annotation with case expressions
users, err := queries.GetQuerySet(&User{}).
    Annotate("UserType", expr.Case(
        expr.When(expr.Q("TodoSet__count__gte", 10), "Power User"),
        expr.When(expr.Q("TodoSet__count__gte", 5), "Regular User"),
        expr.Default("New User"),
    )).
    All()
```

---

## 🎯 Optimization Techniques

### Selective Field Loading

Load only the fields you need:

```go
// Load specific fields only
users, err := queries.GetQuerySet(&User{}).
    Select("ID", "Name", "Email").
    All()

// Load related fields selectively
todos, err := queries.GetQuerySet(&Todo{}).
    Select("*", "User.Name", "User.Email").
    All()
```

### Efficient Relation Loading

Optimize relation loading to avoid N+1 queries:

```go
// Good: Load relations in the initial query
todos, err := queries.GetQuerySet(&Todo{}).
    Select("*", "User.*", "Category.*").
    All()

// Bad: This causes N+1 queries
todos, err := queries.GetQuerySet(&Todo{}).All()
for _, row := range todos {
    todo := row.Value()
    // Additional query for each todo
    user, _ := queries.GetObject(&User{}, todo.UserID)
}
```

### Bulk Operations

Use bulk operations for better performance:

```go
// Bulk create
newTodos := []*Todo{
    {Title: "Todo 1", UserID: 1},
    {Title: "Todo 2", UserID: 2},
    {Title: "Todo 3", UserID: 3},
}
createdTodos, err := queries.GetQuerySet(&Todo{}).
    BulkCreate(newTodos)

// Bulk update
err = queries.GetQuerySet(&Todo{}).
    Filter("Done", false).
    BulkUpdate(newTodos, expr.Named("Done", true))
```

---

## 🔄 Scopes and Reusable Queries

### Defining Scopes

Create reusable query scopes:

```go
// Define common query scopes
func ActiveUsersScope(qs queries.QuerySet[*User], internals *queries.QuerySetInternals) queries.QuerySet[*User] {
    return qs.Filter("IsActive", true)
}

func RecentUsersScope(qs queries.QuerySet[*User], internals *queries.QuerySetInternals) queries.QuerySet[*User] {
    return qs.Filter("CreatedAt__gte", time.Now().AddDate(0, -1, 0))
}

// Use scopes in queries
users, err := queries.GetQuerySet(&User{}).
    Scope(ActiveUsersScope, RecentUsersScope).
    OrderBy("-CreatedAt").
    All()
```

### Query Builders

Create query builder functions for complex queries:

```go
func BuildUserQuery(filters map[string]interface{}) queries.QuerySet[*User] {
    qs := queries.GetQuerySet(&User{}).Select("*", "Profile.*")
    
    if country, ok := filters["country"]; ok {
        qs = qs.Filter("Profile.Country", country)
    }
    
    if minAge, ok := filters["min_age"]; ok {
        qs = qs.Filter("Profile.Age__gte", minAge)
    }
    
    if search, ok := filters["search"]; ok {
        qs = qs.Filter(expr.Q("Name__icontains", search).Or(
            expr.Q("Email__icontains", search),
        ))
    }
    
    return qs
}

// Use the builder
users, err := BuildUserQuery(map[string]interface{}{
    "country": "US",
    "min_age": 18,
    "search":  "john",
}).All()
```

---

## 🔧 Raw SQL and Custom Queries

### Raw SQL Queries

Use raw SQL when needed:

```go
// Raw SQL query
rows, err := queries.GetQuerySet(&User{}).
    Raw("SELECT * FROM users WHERE created_at > ? AND country = ?", 
        time.Now().AddDate(0, -1, 0), "US")

// Raw SQL with result scanning
type UserStats struct {
    Country string
    Count   int
}

var stats []UserStats
rows, err := queries.GetQuerySet(&User{}).
    Raw("SELECT country, COUNT(*) as count FROM users GROUP BY country")
if err != nil {
    return err
}
defer rows.Close()

for rows.Next() {
    var stat UserStats
    err := rows.Scan(&stat.Country, &stat.Count)
    if err != nil {
        return err
    }
    stats = append(stats, stat)
}
```

### Custom Expressions

Create custom expressions for complex calculations:

```go
// Custom expression for distance calculation
type DistanceExpression struct {
    lat1, lon1, lat2, lon2 float64
}

func (d *DistanceExpression) SQL(sb *strings.Builder) []interface{} {
    sb.WriteString("6371 * acos(cos(radians(?)) * cos(radians(lat)) * cos(radians(lon) - radians(?)) + sin(radians(?)) * sin(radians(lat)))")
    return []interface{}{d.lat1, d.lon1, d.lat1}
}

// Use in queries
locations, err := queries.GetQuerySet(&Location{}).
    Annotate("Distance", &DistanceExpression{
        lat1: userLat, lon1: userLon,
        lat2: 0, lon2: 0, // These will be filled from database
    }).
    Filter("Distance__lte", 50). // Within 50km
    OrderBy("Distance").
    All()
```

---

## 🔀 Subqueries and Exists

### Subqueries

Use subqueries for complex filtering:

```go
// Subquery to find users with incomplete todos
incompleteSubquery := queries.GetQuerySet(&Todo{}).
    Filter("Done", false).
    Filter("User", expr.OuterRef("ID")).
    ValuesList("ID")

users, err := queries.GetQuerySet(&User{}).
    Filter("ID__in", incompleteSubquery).
    All()
```

### Exists Queries

Use exists for efficient filtering:

```go
// Find users who have at least one incomplete todo
users, err := queries.GetQuerySet(&User{}).
    Filter(expr.Exists(
        queries.GetQuerySet(&Todo{}).
            Filter("User", expr.OuterRef("ID")).
            Filter("Done", false),
    )).
    All()
```

---

## 📈 Performance Monitoring

### Query Analysis

Monitor and analyze query performance:

```go
// Get query information
qs := queries.GetQuerySet(&User{}).
    Select("*", "Profile.*").
    Filter("IsActive", true)

users, err := qs.All()
if err != nil {
    return err
}

// Access the latest query information
queryInfo := qs.LatestQuery()
log.Printf("SQL: %s", queryInfo.SQL())
log.Printf("Args: %v", queryInfo.Args())
```

### Database Indexes

Ensure proper indexes for query performance:

```sql
-- Create indexes for commonly filtered/ordered fields
CREATE INDEX idx_users_is_active ON users(is_active);
CREATE INDEX idx_users_created_at ON users(created_at);
CREATE INDEX idx_todos_user_id_done ON todos(user_id, done);
CREATE INDEX idx_profiles_country ON profiles(country);
```

---

## 🧪 Testing Query Patterns

### Unit Testing Queries

Test your query patterns:

```go
func TestUserQueryBuilder(t *testing.T) {
    // Setup test data
    user1 := &User{Name: "John", Country: "US", Age: 25}
    user2 := &User{Name: "Jane", Country: "CA", Age: 17}
    
    queries.CreateObject(user1)
    queries.CreateObject(user2)
    
    // Test query builder
    users, err := BuildUserQuery(map[string]interface{}{
        "country": "US",
        "min_age": 18,
    }).All()
    
    assert.NoError(t, err)
    assert.Len(t, users, 1)
    assert.Equal(t, "John", users[0].Value().Name)
}
```

### Integration Testing

Test complex queries with real data:

```go
func TestComplexUserQuery(t *testing.T) {
    // Setup complex test scenario
    setupTestData(t)
    
    // Test complex query
    users, err := queries.GetQuerySet(&User{}).
        Select("*", "Profile.*").
        Filter("IsActive", true).
        Filter("TodoSet.Done", false).
        Annotate("TodoCount", expr.Count("TodoSet.ID")).
        Having("TodoCount__gte", 5).
        OrderBy("-TodoCount").
        All()
    
    assert.NoError(t, err)
    assert.True(t, len(users) > 0)
    
    // Verify annotations
    for _, row := range users {
        todoCount, exists := row.GetAnnotation("TodoCount")
        assert.True(t, exists)
        assert.GreaterOrEqual(t, todoCount.(int64), int64(5))
    }
}
```

---

## 💡 Best Practices

### Query Organization

1. **Group Related Queries**: Keep related queries in the same module
2. **Use Descriptive Names**: Name your query functions clearly
3. **Document Complex Queries**: Add comments for complex query logic
4. **Test Edge Cases**: Test queries with empty results, large datasets, etc.

### Performance Guidelines

1. **Use Indexes**: Ensure proper database indexes for filtered/ordered fields
2. **Limit Results**: Always use `Limit()` for large datasets
3. **Select Specific Fields**: Don't select unnecessary fields
4. **Monitor Query Performance**: Use query analysis tools

### Error Handling

```go
func GetUserWithTodos(userID int) (*User, error) {
    users, err := queries.GetQuerySet(&User{}).
        Select("*", "TodoSet.*").
        Filter("ID", userID).
        All()
    
    if err != nil {
        return nil, fmt.Errorf("failed to fetch user %d: %w", userID, err)
    }
    
    if len(users) == 0 {
        return nil, fmt.Errorf("user %d not found", userID)
    }
    
    return users[0].Value(), nil
}
```

---

Continue with [Expressions](../expressions/expressions.md) to learn about building complex query expressions…