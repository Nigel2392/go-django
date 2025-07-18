# Writing Queries

This guide covers advanced techniques for writing complex queries using the `go-django-queries` package.

Building on the [QuerySet Reference](./queryset.md), this document explores practical patterns and advanced use cases for constructing sophisticated database queries.

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
    Select("*", "Profile.*").
    Filter("IsActive", true).
    Filter("Profile.Country", "US").
    OrderBy("-CreatedAt").
    Limit(20).
    All()
```

### Conditional Query Building

Build queries dynamically based on conditions:

```go
func GetUsersWithFilters(country string, minAge int, orderBy string) ([]User, error) {
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