# Virtual Fields

This document covers virtual fields in the `go-django-queries` package, which allow you to create computed fields that are calculated at query time using SQL expressions.

Virtual fields enable you to add dynamic, calculated values to your models without modifying your database schema.

---

## 🔮 What are Virtual Fields?

Virtual fields are computed fields that exist only during query execution. They are not stored in the database but are calculated on-the-fly using SQL expressions.

Virtual fields are useful for:
- Field calculations (e.g., mathematical operations)
- String manipulations (e.g., concatenation, formatting)
- Conditional logic (e.g., status based on multiple conditions)
- Data aggregations (e.g., counts, sums)

---

## 🏗️ Creating Virtual Fields

### Basic Virtual Field with Annotations

The most common way to create virtual fields is using annotations:

```go
// Add a computed field to calculate user age
users, err := queries.GetQuerySet(&User{}).
    Annotate("Age", expr.F("YEAR(CURRENT_DATE) - YEAR(![BirthDate])", "BirthDate")).
    All()

// Access the virtual field
for _, user := range users {
    fmt.Printf("User: %s, Age: %d\n", user.Name, user.Age)
}
```

### String Concatenation

Create virtual fields that combine multiple fields:

```go
// Concatenate first and last name
users, err := queries.GetQuerySet(&User{}).
    Annotate("FullName", expr.CONCAT(
        expr.Field("FirstName"),
        expr.Value(" "),
        expr.Field("LastName"),
    )).
    All()
```

### Mathematical Operations

Use logical expressions for calculations:

```go
// Calculate total price with tax
products, err := queries.GetQuerySet(&Product{}).
    Annotate("TotalPrice", 
        expr.Logical("Price").MUL(expr.Value(1.08)),
    ).
    All()

// Calculate discount percentage
products, err := queries.GetQuerySet(&Product{}).
    Annotate("DiscountPercent",
        expr.Logical("OriginalPrice").SUB("CurrentPrice").
        DIV("OriginalPrice").MUL(expr.Value(100)),
    ).
    All()
```

---

## 🔍 Conditional Virtual Fields

### Case-When Logic

Create virtual fields based on conditional logic:

```go
// Create status field based on multiple conditions
users, err := queries.GetQuerySet(&User{}).
    Annotate("Status", 
        expr.Case(
            expr.When(expr.Q("IsActive", true), "Active"),
            expr.When(expr.Q("IsBlocked", true), "Blocked"), 
            expr.Value("Inactive"),
        ),
    ).
    All()
```

### Complex Conditional Logic

Use nested conditions for more complex logic:

```go
// User classification based on activity
users, err := queries.GetQuerySet(&User{}).
    Annotate("UserType",
        expr.Case(
            expr.When(expr.Q("IsAdmin", true), "Administrator"),
            expr.When(expr.And(
                expr.Q("PostCount__gte", 10),
                expr.Q("IsActive", true),
            ), "Power User"),
            expr.When(expr.Q("PostCount__gte", 5), "Regular User"),
            expr.Value("New User"),
        ),
    ).
    All()
```

---

## 📊 Aggregation Virtual Fields

### Count Aggregations

Use count functions for aggregating related data:

```go
// Count related objects
users, err := queries.GetQuerySet(&User{}).
    Annotate("PostCount", expr.Count("ID")).
    All()

// Conditional counting using case expressions
users, err := queries.GetQuerySet(&User{}).
    Annotate("PublishedPostCount",
        expr.Count(
            expr.Case(
                expr.When(expr.Q("Posts__Published", true), expr.Field("Posts__ID")),
                expr.Value(nil),
            ),
        ),
    ).
    All()
```

### Other Aggregations

Use various aggregation functions:

```go
// Sum, average, min, max
orders, err := queries.GetQuerySet(&Order{}).
    Annotate("TotalAmount", expr.Sum("OrderItems__Price")).
    Annotate("AverageItemPrice", expr.Avg("OrderItems__Price")).
    Annotate("MinItemPrice", expr.Min("OrderItems__Price")).
    Annotate("MaxItemPrice", expr.Max("OrderItems__Price")).
    All()
```

---

## 🗓️ Date and Time Virtual Fields

### Date Functions

Create virtual fields for date operations:

```go
// Extract parts of dates
posts, err := queries.GetQuerySet(&Post{}).
    Annotate("Year", expr.F("YEAR(![CreatedAt])", "CreatedAt")).
    Annotate("Month", expr.F("MONTH(![CreatedAt])", "CreatedAt")).
    Annotate("DayOfWeek", expr.F("DAYOFWEEK(![CreatedAt])", "CreatedAt")).
    All()
```

### Date Calculations

Calculate time differences:

```go
// Days since creation
posts, err := queries.GetQuerySet(&Post{}).
    Annotate("DaysOld", 
        expr.F("DATEDIFF(CURRENT_DATE, ![CreatedAt])", "CreatedAt"),
    ).
    All()
```

---

## 🔧 Advanced Virtual Field Techniques

### Raw SQL Expressions

Use raw SQL for complex calculations:

```go
// Complex string manipulation
users, err := queries.GetQuerySet(&User{}).
    Annotate("InitializedName",
        expr.Raw("CONCAT(LEFT(first_name, 1), '. ', last_name)"),
    ).
    All()
```

### Logical Expressions

Use logical expressions for field operations:

```go
// Update with calculated values
updated, err := queries.GetQuerySet(&Product{}).
    Select("Price", "DiscountRate").
    Update(&Product{},
        expr.As("FinalPrice", 
            expr.Logical("Price").MUL(
                expr.Logical(expr.Value(1)).SUB("DiscountRate"),
            ),
        ),
    )
```

---

## 🎯 Performance Considerations

### Indexing Virtual Fields

While virtual fields aren't stored, consider indexing the underlying fields:

```sql
-- Index fields used in virtual field calculations
CREATE INDEX idx_users_birth_date ON users(birth_date);
CREATE INDEX idx_posts_created_at ON posts(created_at);
```

### Query Optimization

Optimize virtual field queries:

```go
// Use virtual fields in filtering efficiently
users, err := queries.GetQuerySet(&User{}).
    Annotate("Age", expr.F("YEAR(CURRENT_DATE) - YEAR(![BirthDate])", "BirthDate")).
    Filter("Age__gte", 18).  // Filter on virtual field
    All()
```

### Caching Strategies

For expensive virtual field calculations, consider caching:

```go
// Cache results of expensive calculations
type UserWithStats struct {
    User
    PostCount    int    `json:"post_count"`
    AvgRating    float64 `json:"avg_rating"`
    LastPostDate time.Time `json:"last_post_date"`
}

// Use computed fields for frequently accessed data
func GetUserStats(userID int64) (*UserWithStats, error) {
    // Check cache first
    if cached := getFromCache(userID); cached != nil {
        return cached, nil
    }
    
    // Compute and cache
    user, err := queries.GetQuerySet(&User{}).
        Annotate("PostCount", expr.Count("Posts")).
        Annotate("AvgRating", expr.Avg("Posts__Rating")).
        Annotate("LastPostDate", expr.Max("Posts__CreatedAt")).
        Filter("ID", userID).
        First()
    
    if err != nil {
        return nil, err
    }
    
    result := &UserWithStats{
        User:         *user,
        PostCount:    user.PostCount,
        AvgRating:    user.AvgRating,
        LastPostDate: user.LastPostDate,
    }
    
    setCache(userID, result)
    return result, nil
}
```

---

## 🧪 Testing Virtual Fields

### Unit Tests

Test virtual field calculations:

```go
func TestVirtualFields(t *testing.T) {
    // Setup test data
    user := &User{
        FirstName: "John",
        LastName:  "Doe",
        BirthDate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
    }
    _, err := queries.GetQuerySet(user).Create(user)
    require.NoError(t, err)
    
    // Test virtual field calculation
    result, err := queries.GetQuerySet(&User{}).
        Annotate("FullName", expr.CONCAT(
            expr.Field("FirstName"),
            expr.Value(" "),
            expr.Field("LastName"),
        )).
        Filter("ID", user.ID).
        First()
    
    require.NoError(t, err)
    assert.Equal(t, "John Doe", result.FullName)
}
```

### Integration Tests

Test virtual fields in complex scenarios:

```go
func TestComplexVirtualFields(t *testing.T) {
    // Test conditional virtual fields
    users, err := queries.GetQuerySet(&User{}).
        Annotate("Status",
            expr.Case(
                expr.When(expr.Q("IsActive", true), "Active"),
                expr.Value("Inactive"),
            ),
        ).
        Filter("Status", "Active").
        All()
    
    require.NoError(t, err)
    for _, user := range users {
        assert.Equal(t, "Active", user.Status)
        assert.True(t, user.IsActive)
    }
}
```

---

## 📚 Real-World Examples

### E-commerce

```go
// Product pricing with discounts
products, err := queries.GetQuerySet(&Product{}).
    Annotate("FinalPrice",
        expr.Case(
            expr.When(expr.Q("OnSale", true),
                expr.Logical("Price").MUL(
                    expr.Logical(expr.Value(1)).SUB("DiscountRate"),
                ),
            ),
            expr.Field("Price"),
        ),
    ).
    Annotate("Savings",
        expr.Case(
            expr.When(expr.Q("OnSale", true),
                expr.Logical("Price").SUB("FinalPrice"),
            ),
            expr.Value(0),
        ),
    ).
    All()
```

### User Analytics

```go
// User engagement metrics
users, err := queries.GetQuerySet(&User{}).
    Annotate("EngagementScore",
        expr.Logical("PostCount").MUL(expr.Value(2)).ADD(
            expr.Logical("CommentCount").MUL(expr.Value(1)),
        ).ADD(
            expr.Logical("LikeCount").MUL(expr.Value(0.5)),
        ),
    ).
    OrderBy("-EngagementScore").
    All()
```

### Content Management

```go
// Article statistics
articles, err := queries.GetQuerySet(&Article{}).
    Annotate("WordCount", expr.LENGTH("Content")).
    Annotate("ReadingTime", 
        expr.Logical(expr.LENGTH("Content")).DIV(expr.Value(200)), // Assuming 200 WPM
    ).
    Annotate("Popularity",
        expr.Logical("ViewCount").MUL(expr.Value(0.3)).ADD(
            expr.Logical("ShareCount").MUL(expr.Value(0.7)),
        ),
    ).
    All()
```

This comprehensive guide covers virtual fields in the go-django-queries package, providing practical examples based on actual usage patterns.