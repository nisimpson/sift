# Sift

A universal query filter and sort library for Go that lets you write filter and sort logic once and use it across multiple backends.

[![Go Reference](https://pkg.go.dev/badge/github.com/nisimpson/sift.svg)](https://pkg.go.dev/github.com/nisimpson/sift)
[![Go Report Card](https://goreportcard.com/badge/github.com/nisimpson/sift)](https://goreportcard.com/report/github.com/nisimpson/sift)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## The Problem

When building applications with multiple data backends (DynamoDB, SQL, MongoDB, Elasticsearch), you end up writing the same filtering and sorting logic multiple times in different query languages. This leads to:

- Code duplication across data access layers
- Inconsistent filter and sort capabilities between backends
- Difficulty switching or adding new backends
- Complex translation logic scattered throughout the codebase

## The Solution

Sift provides a universal filter and sort expression language using an Abstract Syntax Tree (AST). Define your filters and sorts once, then implement backend-specific evaluators to translate them into native queries.

```go
// Define a filter once
filter := &sift.AndOperation{
    Left: &sift.Condition{
        Name:      "status",
        Operation: sift.OperationEQ,
        Value:     "active",
    },
    Right: &sift.Condition{
        Name:      "age",
        Operation: sift.OperationGT,
        Value:     "18",
    },
}

// Use with any backend
adapter := mybackend.NewAdapter()
err := sift.Thru(ctx, adapter, filter)
```

## Installation

```bash
go get github.com/nisimpson/sift
```

### Backend Adapters

- [DynamoDB](./thru/dynamodb) - `go get github.com/nisimpson/sift/thru/dynamodb`
- [SQL](./thru/sql) - `go get github.com/nisimpson/sift/thru/sql` (PostgreSQL, MySQL, SQLite, SQL Server)

## Quick Start

### 1. Create a Filter

```go
import "github.com/nisimpson/sift"

// Simple condition: status = "active"
filter := &sift.Condition{
    Name:      "status",
    Operation: sift.OperationEQ,
    Value:     "active",
}

// Or use the fluent builder API
filter := sift.Eq("status", "active")

// Complex filter: (status = "active" AND age > 18) OR role = "admin"
filter := sift.Eq("status", "active").
    And(sift.Gt("age", 18)).
    Or(sift.Eq("role", "admin"))
```

### Expression Builder

The fluent builder API provides a more readable way to construct filters:

```go
// All comparison operations
sift.Eq("field", "value")           // Equal
sift.Neq("field", "value")          // Not equal
sift.Lt("field", 18)                // Less than
sift.Lte("field", 18)               // Less than or equal
sift.Gt("field", 18)                // Greater than
sift.Gte("field", 18)               // Greater than or equal

// String operations
sift.Contains("email", "@example.com")

// Collection operations
sift.In("status", "active")
sift.Between("age", 18, 65)

// Existence operations
sift.Exists("email")
sift.NotExists("deleted_at")

// Combine with logical operations
filter := sift.Eq("status", "active").
    And(sift.Gt("age", 18)).
    Or(sift.Eq("role", "admin")).
    Not()

// The builder returns an ExpressionBuilder which implements Expression
// and can be used directly with Thru()
err := sift.Thru(ctx, adapter, filter)
```

### Sorting

Create sort expressions using the fluent builder API:

```go
// Sort by a single field
sort := sift.Sort("created_at", sift.SortDesc)

// Sort by multiple fields
sort := sift.Sort("created_at", sift.SortDesc).
    ThenBy("name", sift.SortAsc)

// Control NULL ordering
sort := sift.Sort("email", sift.SortAsc).NullsLast()

// Use with adapter
err := sift.SortThru(ctx, adapter, sort)
```

### 2. Implement an Adapter

```go
type MyAdapter struct {
    query string
    args  []interface{}
}

func NewAdapter() *MyAdapter {
    return &MyAdapter{
        args: make([]interface{}, 0),
    }
}

// Implement the Adapter interface
func (a *MyAdapter) Evaluator(ctx context.Context) *sift.Evaluator {
    return &sift.Evaluator{
        ConditionEvaluator: a,
        AndEvaluator:       a,
        OrEvaluator:        a,
        NotEvaluator:       a,
    }
}

func (a *MyAdapter) EvaluateCondition(ctx context.Context, node *sift.Condition) error {
    switch node.Operation {
    case sift.OperationEQ:
        a.query = fmt.Sprintf("%s = ?", node.Name)
        a.args = append(a.args, node.Value)
    case sift.OperationGT:
        a.query = fmt.Sprintf("%s > ?", node.Name)
        a.args = append(a.args, node.Value)
    // ... handle other operations
    default:
        return sift.ErrorOperationNotSupported(string(node.Operation))
    }
    return nil
}

func (a *MyAdapter) EvaluateAnd(ctx context.Context, node *sift.AndOperation) error {
    leftAdapter := NewAdapter()
    if err := sift.Thru(ctx, leftAdapter, node.Left); err != nil {
        return err
    }
    
    rightAdapter := NewAdapter()
    if err := sift.Thru(ctx, rightAdapter, node.Right); err != nil {
        return err
    }
    
    a.query = fmt.Sprintf("(%s) AND (%s)", leftAdapter.query, rightAdapter.query)
    a.args = append(leftAdapter.args, rightAdapter.args...)
    return nil
}

// Implement EvaluateOr and EvaluateNot similarly...
```

### 3. Use the Adapter

```go
adapter := NewAdapter()
err := sift.Thru(ctx, adapter, filter)
if err != nil {
    log.Fatal(err)
}

// Use the generated query
rows, err := db.Query(adapter.query, adapter.args...)
```

## Supported Operations

### Condition Operations

| Category | Constant | Description | Example |
|----------|----------|-------------|---------|
| **Comparison** | `OperationEQ` | Equal | `eq(status,active)` |
| | `OperationNEQ` | Not equal | `ne(status,deleted)` |
| | `OperationLT` | Less than | `lt(age,18)` |
| | `OperationLTE` | Less than or equal | `le(age,18)` |
| | `OperationGT` | Greater than | `gt(age,18)` |
| | `OperationGTE` | Greater than or equal | `ge(age,18)` |
| **String** | `OperationContains` | Substring match | `contains(email,@example.com)` |
| | `OperationBeginsWith` | Prefix match | `begins_with(name,John)` |
| **Collection** | `OperationIn` | Value in list | `in(status,active,pending)` |
| **Existence** | `OperationExists` | Attribute exists | `exists(email)` |
| | `OperationNotExists` | Attribute doesn't exist | `not_exists(deleted_at)` |
| **Range** | `OperationBetween` | Value between bounds | `between(age,18,65)` |

### Logical Operations

| Type | Description | Example |
|------|-------------|---------|
| `AndOperation` | Logical AND - combines two conditions | `and(eq(status,active),gt(age,18))` |
| `OrOperation` | Logical OR - either condition matches | `or(eq(role,admin),eq(role,moderator))` |
| `NotOperation` | Logical NOT - negates a condition | `not(eq(deleted,true))` |

## Serialization

Sift includes built-in serialization to a URL-safe prefix notation format:

```go
// Serialize (no custom expressions)
filter := &sift.Condition{
    Name:      "status",
    Operation: sift.OperationEQ,
    Value:     "active",
}
str, err := sift.Format(filter, nil)
// str = "eq(status,active)"

// Deserialize (no custom expressions)
expr, err := sift.Parse("and(eq(status,active),gt(age,18))", nil)

// With custom expressions, provide a registry
registry := dynamodb.NewRegistry()
filter := dynamodb.Size("tags", sift.OperationGT, 5)
str, err := sift.Format(filter, registry)
// str = "size(tags,gt,5)"

parsed, err := sift.Parse(str, registry)
```

### Format Examples

```
eq(status,active)                     // status = "active"
gt(age,18)                            // age > 18
contains(email,@example.com)          // email contains "@example.com"
and(eq(status,active),gt(age,18))     // status = "active" AND age > 18
or(eq(role,admin),eq(role,moderator)) // role = "admin" OR role = "moderator"
not(eq(deleted,true))                 // NOT deleted = true
```

This format is ideal for URL query strings:

```
GET /users?filter=and(eq(status,active),gt(age,18))
```

## Sorting

Sift provides sorting support through the `SortThru()` function, following the same visitor pattern as filtering.

### Basic Sorting

```go
// Sort by a single field
sort := sift.Sort("created_at", sift.SortDesc)

adapter := sql.NewAdapter()
sift.SortThru(ctx, adapter, sort)

query := fmt.Sprintf("SELECT * FROM users ORDER BY %s", adapter.OrderBy())
// SELECT * FROM users ORDER BY created_at DESC
```

### Multiple Sort Fields

Use `ThenBy()` to add additional sort fields:

```go
// Sort by created_at DESC, then by name ASC
sort := sift.Sort("created_at", sift.SortDesc).
    ThenBy("name", sift.SortAsc)

adapter := sql.NewAdapter()
sift.SortThru(ctx, adapter, sort)
// ORDER BY created_at DESC, name ASC
```

### NULL Handling

Control where NULL values appear in the sort order:

```go
// NULLs appear last in ascending sort
sort := sift.Sort("email", sift.SortAsc).NullsLast()

// Can be applied to specific fields in multi-field sorts
sort := sift.Sort("created_at", sift.SortDesc).
    ThenBy("email", sift.SortAsc).NullsLast().
    ThenBy("name", sift.SortAsc)
// Only email field has NullsLast
```

### Combining Filtering and Sorting

Use both filtering and sorting together:

```go
// Filter and sort
filter := sift.Eq("status", "active").And(sift.Gt("age", 18))
sort := sift.Sort("created_at", sift.SortDesc).ThenBy("name", sift.SortAsc)

adapter := sql.NewAdapter()
sift.Thru(ctx, adapter, filter)
sift.SortThru(ctx, adapter, sort)

query := fmt.Sprintf("SELECT * FROM users WHERE %s ORDER BY %s",
    adapter.Query(), adapter.OrderBy())
// SELECT * FROM users WHERE (status = $1) AND (age > $2) ORDER BY created_at DESC, name ASC
```

### Sort Directions

Two sort directions are available:

- `sift.SortAsc` - Ascending order (A-Z, 0-9, oldest-newest)
- `sift.SortDesc` - Descending order (Z-A, 9-0, newest-oldest)

### Adapter Support

Adapters choose whether to support sorting by implementing the sort evaluator interfaces:

```go
func (a *Adapter) Evaluator(ctx context.Context) *sift.Evaluator {
    return &sift.Evaluator{
        // Filtering
        ConditionEvaluator: a,
        AndEvaluator:       a,
        // Sorting (optional)
        SortListEvaluator:  a,
    }
}

func (a *Adapter) EvaluateSortList(ctx context.Context, list *sift.SortList) error {
    // Translate sort fields to backend-specific syntax
}
```

## Custom Expressions

Extend Sift with custom expression types for backend-specific operations using the Registry pattern:

```go
// Define a custom expression type
type GeoWithinExpression struct {
    Field  string
    Lat    float64
    Lng    float64
    Radius float64
}

func (g *GeoWithinExpression) Type() string {
    return "geo_within"
}

// Define a formatter
type GeoFormatter struct{}

func (GeoFormatter) FormatCustomExpression(expr sift.CustomExpression) (string, error) {
    g := expr.(*GeoWithinExpression)
    return fmt.Sprintf("geo_within(%s,%f,%f,%f)", 
        g.Field, g.Lat, g.Lng, g.Radius), nil
}

func (GeoFormatter) ParseCustomExpression(p *sift.Parser) (sift.CustomExpression, error) {
    field := p.ReadValue()
    p.Expect(',')
    lat, _ := strconv.ParseFloat(p.ReadValue(), 64)
    p.Expect(',')
    lng, _ := strconv.ParseFloat(p.ReadValue(), 64)
    p.Expect(',')
    radius, _ := strconv.ParseFloat(p.ReadValue(), 64)
    p.Expect(')')
    return &GeoWithinExpression{field, lat, lng, radius}, nil
}

// Create a registry and register the custom expression
registry := sift.NewRegistry()
registry.Register("geo_within", GeoFormatter{})

// Use the custom expression
expr := sift.NewCustomExpression(&GeoWithinExpression{
    Field:  "location",
    Lat:    40.7128,
    Lng:    -74.0060,
    Radius: 5000,
})

// Serialize with registry
str, _ := sift.Format(expr, registry)
// str = "geo_within(location,40.712800,-74.006000,5000.000000)"

// Parse with registry
parsed, _ := sift.Parse(str, registry)
```

### Registry Pattern

The registry pattern prevents naming collisions when multiple backends define custom expressions:

```go
// Each backend provides its own registry
dynamoRegistry := dynamodb.NewRegistry()  // Registers "size", "attribute_type"
exprRegistry := exprlang.NewRegistry()    // Registers "exprlang"

// Use the appropriate registry for your backend
filter := dynamodb.Size("tags", sift.OperationGT, 5)
str, _ := sift.Format(filter, dynamoRegistry)
// str = "size(tags,gt,5)"

// For expressions without custom types, pass nil
filter := sift.Eq("status", "active")
str, _ := sift.Format(filter, nil)
// str = "eq(status,active)"
```

## Use Cases

### Multi-Backend Applications

```go
type UserRepository interface {
    Find(ctx context.Context, filter sift.Expression) ([]*User, error)
}

// DynamoDB implementation
type DynamoUserRepo struct {
    client *dynamodb.Client
}

func (r *DynamoUserRepo) Find(ctx context.Context, filter sift.Expression) ([]*User, error) {
    adapter := dynamodb.NewAdapter()
    sift.Thru(ctx, adapter, filter)
    // Use adapter.Expression(), adapter.Names(), adapter.Values() with DynamoDB
}

// PostgreSQL implementation
type PostgresUserRepo struct {
    db *sql.DB
}

func (r *PostgresUserRepo) Find(ctx context.Context, filter sift.Expression) ([]*User, error) {
    adapter := sqlAdapter.NewAdapter() // Defaults to PostgreSQL
    sift.Thru(ctx, adapter, filter)
    
    query := fmt.Sprintf("SELECT * FROM users WHERE %s", adapter.Query())
    rows, err := r.db.Query(query, adapter.Args()...)
    // ... scan rows into users
}

// MySQL implementation
type MySQLUserRepo struct {
    db *sql.DB
}

func (r *MySQLUserRepo) Find(ctx context.Context, filter sift.Expression) ([]*User, error) {
    config := &sqlAdapter.Config{Dialect: sqlAdapter.DialectMySQL}
    adapter := sqlAdapter.NewAdapterWithConfig(config)
    sift.Thru(ctx, adapter, filter)
    
    query := fmt.Sprintf("SELECT * FROM users WHERE %s", adapter.Query())
    rows, err := r.db.Query(query, adapter.Args()...)
    // ... scan rows into users
}
```

### API Query Parameters

```go
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
    filterStr := r.URL.Query().Get("filter")
    
    // Parse with appropriate registry (or nil if no custom expressions)
    filter, err := sift.Parse(filterStr, nil)
    if err != nil {
        http.Error(w, "Invalid filter", http.StatusBadRequest)
        return
    }
    
    users, err := h.repo.Find(r.Context(), filter)
    // ...
}
```

### Testing

```go
func TestUserFiltering(t *testing.T) {
    filter := &sift.Condition{
        Name:      "status",
        Operation: sift.OperationEQ,
        Value:     "active",
    }
    
    // Test with different backends
    t.Run("DynamoDB", func(t *testing.T) {
        adapter := dynamodb.NewAdapter()
        err := sift.Thru(ctx, adapter, filter)
        // assertions...
    })
    
    t.Run("PostgreSQL", func(t *testing.T) {
        adapter := postgres.NewAdapter()
        err := sift.Thru(ctx, adapter, filter)
        // assertions...
    })
}
```

## Design Philosophy

- **Simple interfaces** - Implement only what you need
- **Type-safe** - Leverage Go's type system
- **Composable** - Build complex filters from simple operations
- **Extensible** - Add custom operations for your backend
- **Backend-agnostic** - Works with any data store

## Contributing

Contributions welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

## License

MIT License - see [LICENSE](LICENSE) file for details

## Acknowledgments

Inspired by the need for consistent filtering across multiple data backends in modern applications.
