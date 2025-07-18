# Virtual Fields

This document covers virtual fields in the `go-django-queries` package, which allow you to create computed fields that are calculated at query time rather than stored in the database.

Virtual fields enable you to add dynamic, calculated values to your models without modifying your database schema.

---

## 🔮 What are Virtual Fields?

Virtual fields are computed fields that exist only during query execution. They are not stored in the database but are calculated on-the-fly using SQL expressions.

Virtual fields are useful for:
- Calculated values (e.g., full name from first and last name)
- Aggregations (e.g., count of related objects)
- Conditional logic (e.g., status based on multiple conditions)
- Data transformations (e.g., formatting dates or numbers)

---

## 🏗️ Creating Virtual Fields

### Basic Virtual Field

Create a virtual field using an expression:

```go
type User struct {
    models.Model
    ID        int
    FirstName string
    LastName  string
    // FullName is not stored in database but calculated
    FullName  string
}

func (m *User) FieldDefs() attrs.Definitions {
    return m.Model.Define(m,
        attrs.NewField(m, "ID", &attrs.FieldConfig{
            Primary: true,
        }),
        attrs.NewField(m, "FirstName", nil),
        attrs.NewField(m, "LastName", nil),
        // Virtual field that concatenates first and last name
        fields.NewVirtualField[string](m, &m.FullName, "FullName", 
            expr.Concat("FirstName", expr.Value(" "), "LastName")),
    ).WithTableName("users")
}
```

### Virtual Field with Complex Logic

Create virtual fields with more complex expressions:

```go
type Order struct {
    models.Model
    ID          int
    SubTotal    float64
    TaxRate     float64
    ShippingFee float64
    // Virtual fields
    TaxAmount   float64
    Total       float64
    Status      string
}

func (m *Order) FieldDefs() attrs.Definitions {
    return m.Model.Define(m,
        attrs.NewField(m, "ID", &attrs.FieldConfig{
            Primary: true,
        }),
        attrs.NewField(m, "SubTotal", nil),
        attrs.NewField(m, "TaxRate", nil),
        attrs.NewField(m, "ShippingFee", nil),
        // Calculate tax amount
        fields.NewVirtualField[float64](m, &m.TaxAmount, "TaxAmount",
            expr.Mul("SubTotal", "TaxRate")),
        // Calculate total
        fields.NewVirtualField[float64](m, &m.Total, "Total",
            expr.Add("SubTotal", 
                expr.Mul("SubTotal", "TaxRate"),
                "ShippingFee")),
        // Status based on conditions
        fields.NewVirtualField[string](m, &m.Status, "Status",
            expr.Case(
                expr.When(expr.Q("SubTotal__gte", 1000), "Premium"),
                expr.When(expr.Q("SubTotal__gte", 500), "Standard"),
                expr.Default("Basic"),
            )),
    ).WithTableName("orders")
}
```

---

## 🎯 Types of Virtual Fields

### Calculated Fields

Perform mathematical operations on existing fields:

```go
// Rectangle model with calculated area
type Rectangle struct {
    models.Model
    ID     int
    Width  float64
    Height float64
    Area   float64 // Virtual field
}

func (m *Rectangle) FieldDefs() attrs.Definitions {
    return m.Model.Define(m,
        attrs.NewField(m, "ID", &attrs.FieldConfig{Primary: true}),
        attrs.NewField(m, "Width", nil),
        attrs.NewField(m, "Height", nil),
        fields.NewVirtualField[float64](m, &m.Area, "Area",
            expr.Mul("Width", "Height")),
    ).WithTableName("rectangles")
}
```

### String Manipulation Fields

Transform and format text fields:

```go
type Person struct {
    models.Model
    ID           int
    FirstName    string
    LastName     string
    Email        string
    // Virtual fields
    DisplayName  string
    EmailDomain  string
    Initials     string
}

func (m *Person) FieldDefs() attrs.Definitions {
    return m.Model.Define(m,
        attrs.NewField(m, "ID", &attrs.FieldConfig{Primary: true}),
        attrs.NewField(m, "FirstName", nil),
        attrs.NewField(m, "LastName", nil),
        attrs.NewField(m, "Email", nil),
        // Full name with title
        fields.NewVirtualField[string](m, &m.DisplayName, "DisplayName",
            expr.Concat("FirstName", expr.Value(" "), "LastName")),
        // Extract domain from email
        fields.NewVirtualField[string](m, &m.EmailDomain, "EmailDomain",
            expr.FuncSubstring("Email", 
                expr.Add(expr.FuncPosition(expr.Value("@"), "Email"), expr.Value(1)))),
        // Create initials
        fields.NewVirtualField[string](m, &m.Initials, "Initials",
            expr.Concat(
                expr.FuncSubstring("FirstName", expr.Value(1), expr.Value(1)),
                expr.FuncSubstring("LastName", expr.Value(1), expr.Value(1)),
            )),
    ).WithTableName("people")
}
```

### Date and Time Fields

Work with dates and times:

```go
type Event struct {
    models.Model
    ID          int
    Title       string
    StartDate   time.Time
    EndDate     time.Time
    // Virtual fields
    Duration    int    // In days
    IsUpcoming  bool
    MonthYear   string
}

func (m *Event) FieldDefs() attrs.Definitions {
    return m.Model.Define(m,
        attrs.NewField(m, "ID", &attrs.FieldConfig{Primary: true}),
        attrs.NewField(m, "Title", nil),
        attrs.NewField(m, "StartDate", nil),
        attrs.NewField(m, "EndDate", nil),
        // Calculate duration in days
        fields.NewVirtualField[int](m, &m.Duration, "Duration",
            expr.FuncExtract("day", expr.Sub("EndDate", "StartDate"))),
        // Check if event is upcoming
        fields.NewVirtualField[bool](m, &m.IsUpcoming, "IsUpcoming",
            expr.Gt("StartDate", expr.Now())),
        // Format month and year
        fields.NewVirtualField[string](m, &m.MonthYear, "MonthYear",
            expr.FuncToChar("StartDate", expr.Value("Mon YYYY"))),
    ).WithTableName("events")
}
```

---

## 📊 Aggregation Virtual Fields

### Count Relations

Count related objects:

```go
type User struct {
    models.Model
    ID        int
    Name      string
    Email     string
    // Virtual aggregation fields
    TodoCount int
    CompletedTodoCount int
}

func (m *User) FieldDefs() attrs.Definitions {
    return m.Model.Define(m,
        attrs.NewField(m, "ID", &attrs.FieldConfig{Primary: true}),
        attrs.NewField(m, "Name", nil),
        attrs.NewField(m, "Email", nil),
        // Count all todos
        fields.NewVirtualField[int](m, &m.TodoCount, "TodoCount",
            expr.Count("TodoSet.ID")),
        // Count completed todos
        fields.NewVirtualField[int](m, &m.CompletedTodoCount, "CompletedTodoCount",
            expr.Count("TodoSet.ID", expr.Q("TodoSet.Done", true))),
    ).WithTableName("users")
}
```

### Sum and Average Fields

Calculate sums and averages:

```go
type Customer struct {
    models.Model
    ID               int
    Name             string
    // Virtual aggregation fields
    TotalOrders      int
    TotalSpent       float64
    AverageOrderValue float64
}

func (m *Customer) FieldDefs() attrs.Definitions {
    return m.Model.Define(m,
        attrs.NewField(m, "ID", &attrs.FieldConfig{Primary: true}),
        attrs.NewField(m, "Name", nil),
        // Count total orders
        fields.NewVirtualField[int](m, &m.TotalOrders, "TotalOrders",
            expr.Count("OrderSet.ID")),
        // Sum total amount spent
        fields.NewVirtualField[float64](m, &m.TotalSpent, "TotalSpent",
            expr.Sum("OrderSet.Total")),
        // Calculate average order value
        fields.NewVirtualField[float64](m, &m.AverageOrderValue, "AverageOrderValue",
            expr.Avg("OrderSet.Total")),
    ).WithTableName("customers")
}
```

---

## 🔄 Conditional Virtual Fields

### Case-When Logic

Create virtual fields with conditional logic:

```go
type Employee struct {
    models.Model
    ID          int
    Name        string
    Salary      float64
    Department  string
    YearsOfService int
    // Virtual fields
    SalaryGrade string
    Seniority   string
    BonusRate   float64
}

func (m *Employee) FieldDefs() attrs.Definitions {
    return m.Model.Define(m,
        attrs.NewField(m, "ID", &attrs.FieldConfig{Primary: true}),
        attrs.NewField(m, "Name", nil),
        attrs.NewField(m, "Salary", nil),
        attrs.NewField(m, "Department", nil),
        attrs.NewField(m, "YearsOfService", nil),
        // Salary grade based on salary range
        fields.NewVirtualField[string](m, &m.SalaryGrade, "SalaryGrade",
            expr.Case(
                expr.When(expr.Q("Salary__gte", 100000), "A"),
                expr.When(expr.Q("Salary__gte", 80000), "B"),
                expr.When(expr.Q("Salary__gte", 60000), "C"),
                expr.Default("D"),
            )),
        // Seniority based on years of service
        fields.NewVirtualField[string](m, &m.Seniority, "Seniority",
            expr.Case(
                expr.When(expr.Q("YearsOfService__gte", 10), "Senior"),
                expr.When(expr.Q("YearsOfService__gte", 5), "Mid-Level"),
                expr.When(expr.Q("YearsOfService__gte", 2), "Junior"),
                expr.Default("Entry-Level"),
            )),
        // Bonus rate based on department and seniority
        fields.NewVirtualField[float64](m, &m.BonusRate, "BonusRate",
            expr.Case(
                expr.When(expr.Q("Department", "Sales").And(expr.Q("YearsOfService__gte", 5)), 0.15),
                expr.When(expr.Q("Department", "Sales"), 0.10),
                expr.When(expr.Q("Department", "Engineering").And(expr.Q("YearsOfService__gte", 3)), 0.12),
                expr.When(expr.Q("Department", "Engineering"), 0.08),
                expr.Default(0.05),
            )),
    ).WithTableName("employees")
}
```

---

## 🏃 Using Virtual Fields in Queries

### Select Virtual Fields

Include virtual fields in your queries:

```go
// Query with virtual fields
users, err := queries.GetQuerySet(&User{}).
    Select("*", "FullName", "TodoCount").
    All()

// Access virtual field values
for _, row := range users {
    user := row.Value()
    fmt.Printf("User: %s, Todos: %d\n", user.FullName, user.TodoCount)
}
```

### Filter by Virtual Fields

Filter results using virtual fields:

```go
// Find users with many todos
activeUsers, err := queries.GetQuerySet(&User{}).
    Select("*", "TodoCount").
    Filter("TodoCount__gte", 10).
    All()

// Find premium orders
premiumOrders, err := queries.GetQuerySet(&Order{}).
    Select("*", "Status", "Total").
    Filter("Status", "Premium").
    All()
```

### Order by Virtual Fields

Order results by virtual field values:

```go
// Order users by todo count
users, err := queries.GetQuerySet(&User{}).
    Select("*", "TodoCount").
    OrderBy("-TodoCount").
    All()

// Order events by duration
events, err := queries.GetQuerySet(&Event{}).
    Select("*", "Duration").
    OrderBy("Duration").
    All()
```

---

## 🔍 Advanced Virtual Field Patterns

### Nested Virtual Fields

Create virtual fields that reference other virtual fields:

```go
type Product struct {
    models.Model
    ID          int
    Name        string
    Price       float64
    Cost        float64
    // Virtual fields
    Margin      float64
    MarginPct   float64
    ProfitLevel string
}

func (m *Product) FieldDefs() attrs.Definitions {
    return m.Model.Define(m,
        attrs.NewField(m, "ID", &attrs.FieldConfig{Primary: true}),
        attrs.NewField(m, "Name", nil),
        attrs.NewField(m, "Price", nil),
        attrs.NewField(m, "Cost", nil),
        // Calculate margin
        fields.NewVirtualField[float64](m, &m.Margin, "Margin",
            expr.Sub("Price", "Cost")),
        // Calculate margin percentage
        fields.NewVirtualField[float64](m, &m.MarginPct, "MarginPct",
            expr.Mul(
                expr.Div(expr.Sub("Price", "Cost"), "Price"),
                expr.Value(100),
            )),
        // Profit level based on margin percentage
        fields.NewVirtualField[string](m, &m.ProfitLevel, "ProfitLevel",
            expr.Case(
                expr.When(expr.Gt(
                    expr.Div(expr.Sub("Price", "Cost"), "Price"), 
                    expr.Value(0.5),
                ), "High"),
                expr.When(expr.Gt(
                    expr.Div(expr.Sub("Price", "Cost"), "Price"), 
                    expr.Value(0.3),
                ), "Medium"),
                expr.Default("Low"),
            )),
    ).WithTableName("products")
}
```

### Virtual Fields with Subqueries

Use subqueries in virtual fields:

```go
type Category struct {
    models.Model
    ID              int
    Name            string
    // Virtual fields using subqueries
    ProductCount    int
    AvgProductPrice float64
    TopProductName  string
}

func (m *Category) FieldDefs() attrs.Definitions {
    return m.Model.Define(m,
        attrs.NewField(m, "ID", &attrs.FieldConfig{Primary: true}),
        attrs.NewField(m, "Name", nil),
        // Count products in category
        fields.NewVirtualField[int](m, &m.ProductCount, "ProductCount",
            expr.Count("ProductSet.ID")),
        // Average product price
        fields.NewVirtualField[float64](m, &m.AvgProductPrice, "AvgProductPrice",
            expr.Avg("ProductSet.Price")),
        // Name of most expensive product
        fields.NewVirtualField[string](m, &m.TopProductName, "TopProductName",
            expr.Subquery(
                queries.GetQuerySet(&Product{}).
                    Select("Name").
                    Filter("Category", expr.OuterRef("ID")).
                    OrderBy("-Price").
                    Limit(1),
            )),
    ).WithTableName("categories")
}
```

---

## 🔧 Custom Virtual Field Expressions

### Creating Custom Expressions

Build custom expressions for virtual fields:

```go
// Custom distance calculation expression
type DistanceExpression struct {
    fromLat, fromLng, toLat, toLng float64
}

func (d *DistanceExpression) SQL(inf *expr.ExpressionInfo) (string, []interface{}) {
    return `6371 * acos(
        cos(radians(?)) * cos(radians(?)) * 
        cos(radians(?) - radians(?)) + 
        sin(radians(?)) * sin(radians(?))
    )`, []interface{}{d.fromLat, d.toLat, d.toLng, d.fromLng, d.fromLat, d.toLat}
}

func (d *DistanceExpression) Clone() expr.Expression {
    return &DistanceExpression{
        fromLat: d.fromLat,
        fromLng: d.fromLng,
        toLat:   d.toLat,
        toLng:   d.toLng,
    }
}

func (d *DistanceExpression) Resolve(inf *expr.ExpressionInfo) expr.Expression {
    return d
}

// Use in virtual field
type Location struct {
    models.Model
    ID            int
    Name          string
    Latitude      float64
    Longitude     float64
    // Virtual field for distance to specific point
    DistanceToCenter float64
}

func (m *Location) FieldDefs() attrs.Definitions {
    centerLat, centerLng := 40.7128, -74.0060 // NYC coordinates
    
    return m.Model.Define(m,
        attrs.NewField(m, "ID", &attrs.FieldConfig{Primary: true}),
        attrs.NewField(m, "Name", nil),
        attrs.NewField(m, "Latitude", nil),
        attrs.NewField(m, "Longitude", nil),
        fields.NewVirtualField[float64](m, &m.DistanceToCenter, "DistanceToCenter",
            &DistanceExpression{
                fromLat: centerLat,
                fromLng: centerLng,
                toLat:   0, // Will be replaced with actual latitude
                toLng:   0, // Will be replaced with actual longitude
            }),
    ).WithTableName("locations")
}
```

---

## 🎯 Performance Considerations

### Virtual Field Performance

Virtual fields are calculated at query time, so consider:

1. **Database Load**: Complex virtual fields increase query complexity
2. **Indexing**: Virtual fields cannot be indexed directly
3. **Caching**: Consider caching results for expensive calculations

### Optimization Strategies

```go
// Good: Simple virtual fields
fields.NewVirtualField[string](m, &m.FullName, "FullName",
    expr.Concat("FirstName", expr.Value(" "), "LastName"))

// Consider caching: Complex aggregations
fields.NewVirtualField[float64](m, &m.ComplexScore, "ComplexScore",
    expr.Add(
        expr.Mul("Factor1", expr.Value(0.3)),
        expr.Mul("Factor2", expr.Value(0.5)),
        expr.Mul("Factor3", expr.Value(0.2)),
    ))

// Use indexes on underlying fields
// CREATE INDEX idx_users_first_last ON users(first_name, last_name);
```

---

## 🧪 Testing Virtual Fields

### Testing Virtual Field Logic

Test virtual fields with unit tests:

```go
func TestUserVirtualFields(t *testing.T) {
    // Create test user
    user := &User{
        FirstName: "John",
        LastName:  "Doe",
    }
    err := queries.CreateObject(user)
    assert.NoError(t, err)
    
    // Query with virtual fields
    users, err := queries.GetQuerySet(&User{}).
        Select("*", "FullName").
        Filter("ID", user.ID).
        All()
    
    assert.NoError(t, err)
    assert.Len(t, users, 1)
    
    resultUser := users[0].Value()
    assert.Equal(t, "John Doe", resultUser.FullName)
}

func TestOrderVirtualFields(t *testing.T) {
    // Create test order
    order := &Order{
        SubTotal:    100.0,
        TaxRate:     0.08,
        ShippingFee: 10.0,
    }
    err := queries.CreateObject(order)
    assert.NoError(t, err)
    
    // Query with virtual fields
    orders, err := queries.GetQuerySet(&Order{}).
        Select("*", "TaxAmount", "Total", "Status").
        Filter("ID", order.ID).
        All()
    
    assert.NoError(t, err)
    assert.Len(t, orders, 1)
    
    resultOrder := orders[0].Value()
    assert.Equal(t, 8.0, resultOrder.TaxAmount)
    assert.Equal(t, 118.0, resultOrder.Total)
    assert.Equal(t, "Basic", resultOrder.Status)
}
```

---

## 💡 Best Practices

### Design Guidelines

1. **Keep It Simple**: Virtual fields should be easy to understand and maintain
2. **Performance Aware**: Consider the performance impact of complex calculations
3. **Naming**: Use clear, descriptive names for virtual fields
4. **Documentation**: Document complex virtual field logic

### Common Patterns

```go
// Good: Clear, simple virtual fields
FullName    string // Concatenation
Age         int    // Date calculation
IsActive    bool   // Boolean logic
TotalCount  int    // Simple aggregation

// Be careful with: Complex calculations that might be slow
ComplexScore float64 // Multiple joins and calculations
```

### Error Handling

```go
// Handle potential errors in virtual field expressions
func (m *SafeCalculation) FieldDefs() attrs.Definitions {
    return m.Model.Define(m,
        // ... other fields ...
        fields.NewVirtualField[float64](m, &m.SafeRatio, "SafeRatio",
            expr.Case(
                expr.When(expr.Q("Denominator", 0), expr.Value(0.0)),
                expr.Default(expr.Div("Numerator", "Denominator")),
            )),
    )
}
```

---

Continue with [Models](./models/models.md) to learn more about model definitions and relationships…