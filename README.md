# Sift

A universal query library for Go that lets you write filter, sort, and pagination logic once and use it across multiple backends.

[![Go Reference](https://pkg.go.dev/badge/github.com/nisimpson/sift.svg)](https://pkg.go.dev/github.com/nisimpson/sift)
[![Go Report Card](https://goreportcard.com/badge/github.com/nisimpson/sift)](https://goreportcard.com/report/github.com/nisimpson/sift)
[![CI](https://github.com/nisimpson/sift/workflows/CI/badge.svg)](https://github.com/nisimpson/sift/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## The Problem

When building applications with multiple data backends (DynamoDB, SQL, MongoDB, Elasticsearch), you end up writing the same filtering, sorting, and pagination logic multiple times in different query languages. This leads to:

- Code duplication across data access layers
- Inconsistent query capabilities between backends
- Difficulty switching or adding new backends
- Complex translation logic scattered throughout the codebase

## The Solution

Sift provides a universal query expression language using an Abstract Syntax Tree (AST). Define your filters, sorts, and pagination once, then implement backend-specific evaluators to translate them into native queries.

```go
// Define query components once
filter := sift.Eq("status", "active").And(sift.Gt("age", 18))
sort := sift.Sort("created_at", sift.SortDesc)
page := sift.Paginate().Size(20).Number(2)

// Use with any backend
adapter := mybackend.NewAdapter()
err := sift.Thru(ctx, adapter,
    sift.WithFilter(filter),
    sift.WithSort(sort),
    sift.WithPagination(page))
```

## Installation

```bash
go get github.com/nisimpson/sift
```

### Backend Adapters

- [DynamoDB](./thru/dynamodb) - `go get github.com/nisimpson/sift/thru/dynamodb`
- [SQL](./thru/sql) - `go get github.com/nisimpson/sift/thru/sql` (PostgreSQL, MySQL, SQLite, SQL Server)

## Quick Start

### Unified API

Sift uses a unified API with options for all query operations:

```go
import "github.com/nisimpson/sift"

// Create filter, sort, and pagination
filter := sift.Eq("status", "active").And(sift.Gt("age", 18))
sort := sift.Sort("created_at", sift.SortDesc)
page := sift.Paginate().Size(20).Number(1)

// Apply all at once
adapter := sql.NewAdapter()
err := sift.Thru(ctx, adapter,
    sift.WithFilter(filter),
    sift.WithSort(sort),
    sift.WithPagination(page))

// Or apply individually
err := sift.Thru(ctx, adapter, sift.WithFilter(filter))
err = sift.Thru(ctx, adapter, sift.WithSort(sort))
err = sift.Thru(ctx, adapter, sift.WithPagination(page))
```

### 1. Filtering

```go
// Simple condition: status = "active"
filter := sift.Eq("status", "active")

// Complex filter: (status = "active" AND age > 18) OR role = "admin"
filter := sift.Eq("status", "active").
    And(sift.Gt("age", 18)).
    Or(sift.Eq("role", "admin"))
```

### Expression Builder

The fluent builder API provides a readable way to construct filters:

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
    Or(sift.Eq("role", "admin"))
```

### 2. Sorting

```go
// Sort by a single field
sort := sift.Sort("created_at", sift.SortDesc)

// Sort by multiple fields
sort := sift.Sort("created_at", sift.SortDesc).
    ThenBy("name", sift.SortAsc)

// Control NULL ordering
sort := sift.Sort("email", sift.SortAsc).NullsLast()
```

### 3. Pagination

Sift supports both offset-based and cursor-based pagination:

```go
// Offset-based (SQL databases)
page := sift.Paginate().Size(20).Number(2)  // Page 2, 20 items per page

// Cursor-based (DynamoDB, GraphQL APIs)
page := sift.Paginate().Size(20).Cursor("token123")
```

### 4. Implement an Adapter

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
        ConditionEvaluator:        a,
        AndEvaluator:              a,
        OrEvaluator:               a,
        NotEvaluator:              a,
        SortListEvaluator:         a,  // Optional: for sorting support
        OffsetPaginationEvaluator: a,  // Optional: for pagination support
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
    if err := sift.Thru(ctx, leftAdapter, sift.WithFilter(node.Left)); err != nil {
        return err
    }
    
    rightAdapter := NewAdapter()
    if err := sift.Thru(ctx, rightAdapter, sift.WithFilter(node.Right)); err != nil {
        return err
    }
    
    a.query = fmt.Sprintf("(%s) AND (%s)", leftAdapter.query, rightAdapter.query)
    a.args = append(leftAdapter.args, rightAdapter.args...)
    return nil
}

// Implement EvaluateOr, EvaluateNot, EvaluateSortList, EvaluateOffsetPagination...
```

### 5. Use the Adapter

```go
filter := sift.Eq("status", "active")
sort := sift.Sort("created_at", sift.SortDesc)
page := sift.Paginate().Size(20).Number(1)

adapter := NewAdapter()
err := sift.Thru(ctx, adapter,
    sift.WithFilter(filter),
    sift.WithSort(sort),
    sift.WithPagination(page))

if err != nil {
    log.Fatal(err)
}

// Use the generated query
query := fmt.Sprintf("SELECT * FROM users WHERE %s ORDER BY %s LIMIT %d OFFSET %d",
    adapter.Query(), adapter.OrderBy(), adapter.Limit(), adapter.Offset())
rows, err := db.Query(query, adapter.Args()...)
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

Sift provides sorting support through the unified `Thru()` API with `WithSort()` option.

### Basic Sorting

```go
// Sort by a single field
sort := sift.Sort("created_at", sift.SortDesc)

adapter := sql.NewAdapter()
sift.Thru(ctx, adapter, sift.WithSort(sort))

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
sift.Thru(ctx, adapter, sift.WithSort(sort))
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
sift.Thru(ctx, adapter,
    sift.WithFilter(filter),
    sift.WithSort(sort))

query := fmt.Sprintf("SELECT * FROM users WHERE %s ORDER BY %s",
    adapter.Query(), adapter.OrderBy())
// SELECT * FROM users WHERE (status = $1) AND (age > $2) ORDER BY created_at DESC, name ASC
```

### Sort Directions

Two sort directions are available:

- `sift.SortAsc` - Ascending order (A-Z, 0-9, oldest-newest)
- `sift.SortDesc` - Descending order (Z-A, 9-0, newest-oldest)

## Pagination

Sift supports both offset-based and cursor-based pagination strategies.

### Offset-Based Pagination

Used with SQL databases (LIMIT/OFFSET):

```go
// First page (20 items)
page := sift.Paginate().Size(20).Number(1)

// Second page
page := sift.Paginate().Size(20).Number(2)

adapter := sql.NewAdapter()
sift.Thru(ctx, adapter, sift.WithPagination(page))

query := fmt.Sprintf("SELECT * FROM users LIMIT %d OFFSET %d",
    adapter.Limit(), adapter.Offset())
// SELECT * FROM users LIMIT 20 OFFSET 20
```

### Cursor-Based Pagination

Used with DynamoDB, GraphQL, and other cursor-based APIs:

```go
// First page
page := sift.Paginate().Size(20).Cursor("")

// Next page with cursor from previous response
page := sift.Paginate().Size(20).Cursor("token123")

adapter := dynamodb.NewAdapter()
sift.Thru(ctx, adapter, sift.WithPagination(page))

input := &dynamodb.QueryInput{
    TableName: aws.String("Users"),
    Limit:     adapter.Limit(),  // *int32
}
if token := adapter.Token(); token != "" {
    input.ExclusiveStartKey = decodeToken(token)
}
```

### Default Limits (SQL)

For SQL databases, you can configure a default limit to prevent unbounded queries:

```go
config := &sql.Config{
    Dialect:      sql.DialectPostgreSQL,
    DefaultLimit: 100,  // Applied when no pagination specified
}
adapter := sql.NewAdapterWithConfig(config)

// No pagination - uses default limit of 100
sift.Thru(ctx, adapter, sift.WithFilter(filter))
fmt.Println(adapter.Limit())  // 100

// With pagination - overrides default
page := sift.Paginate().Size(20).Number(1)
sift.Thru(ctx, adapter, sift.WithPagination(page))
fmt.Println(adapter.Limit())  // 20
```

### Complete Example

Combining filter, sort, and pagination:

```go
// SQL (offset-based)
filter := sift.Eq("status", "active").And(sift.Gt("age", 18))
sort := sift.Sort("created_at", sift.SortDesc)
page := sift.Paginate().Size(20).Number(2)

adapter := sql.NewAdapter()
sift.Thru(ctx, adapter,
    sift.WithFilter(filter),
    sift.WithSort(sort),
    sift.WithPagination(page))

query := fmt.Sprintf("SELECT * FROM users WHERE %s ORDER BY %s LIMIT %d OFFSET %d",
    adapter.Query(), adapter.OrderBy(), adapter.Limit(), adapter.Offset())
// SELECT * FROM users WHERE (status = $1) AND (age > $2) ORDER BY created_at DESC LIMIT 20 OFFSET 20

// DynamoDB (cursor-based)
filter := sift.Eq("status", "active")
sort := sift.Sort("created_at", sift.SortDesc)
page := sift.Paginate().Size(20).Cursor("token123")

adapter := dynamodb.NewAdapter()
sift.Thru(ctx, adapter,
    sift.WithFilter(filter),
    sift.WithSort(sort),
    sift.WithPagination(page))

expr, _ := adapter.Expression()
input := &dynamodb.QueryInput{
    TableName:                 aws.String("Users"),
    FilterExpression:          expr.Condition(),
    ExpressionAttributeNames:  expr.Names(),
    ExpressionAttributeValues: expr.Values(),
    ScanIndexForward:          adapter.ScanIndexForward(),
    Limit:                     adapter.Limit(),
}
if token := adapter.Token(); token != "" {
    input.ExclusiveStartKey = decodeToken(token)
}
```

### Adapter Support

Adapters choose which pagination strategy to support:

```go
func (a *Adapter) Evaluator(ctx context.Context) *sift.Evaluator {
    return &sift.Evaluator{
        // Filtering
        ConditionEvaluator: a,
        AndEvaluator:       a,
        // Sorting
        SortListEvaluator:  a,
        // Pagination (choose one or both)
        OffsetPaginationEvaluator: a,  // For SQL
        CursorPaginationEvaluator: a,  // For DynamoDB, GraphQL
    }
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
    Find(ctx context.Context, opts ...sift.Option) ([]*User, error)
}

// DynamoDB implementation
type DynamoUserRepo struct {
    client *dynamodb.Client
}

func (r *DynamoUserRepo) Find(ctx context.Context, opts ...sift.Option) ([]*User, error) {
    adapter := dynamodb.NewAdapter()
    sift.Thru(ctx, adapter, opts...)
    
    expr, _ := adapter.Expression()
    input := &dynamodb.QueryInput{
        TableName:                 aws.String("Users"),
        FilterExpression:          expr.Condition(),
        ExpressionAttributeNames:  expr.Names(),
        ExpressionAttributeValues: expr.Values(),
        ScanIndexForward:          adapter.ScanIndexForward(),
        Limit:                     adapter.Limit(),
    }
    // ... execute query and scan results
}

// PostgreSQL implementation
type PostgresUserRepo struct {
    db *sql.DB
}

func (r *PostgresUserRepo) Find(ctx context.Context, opts ...sift.Option) ([]*User, error) {
    adapter := sqlAdapter.NewAdapter() // Defaults to PostgreSQL
    sift.Thru(ctx, adapter, opts...)
    
    query := fmt.Sprintf("SELECT * FROM users WHERE %s ORDER BY %s LIMIT %d OFFSET %d",
        adapter.Query(), adapter.OrderBy(), adapter.Limit(), adapter.Offset())
    rows, err := r.db.Query(query, adapter.Args()...)
    // ... scan rows into users
}

// Usage - same interface for both backends
filter := sift.Eq("status", "active")
sort := sift.Sort("created_at", sift.SortDesc)
page := sift.Paginate().Size(20).Number(1)

users, err := repo.Find(ctx,
    sift.WithFilter(filter),
    sift.WithSort(sort),
    sift.WithPagination(page))
```

### API Query Parameters

```go
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
    // Parse query parameters
    filterStr := r.URL.Query().Get("filter")
    sortStr := r.URL.Query().Get("sort")
    pageNum, _ := strconv.Atoi(r.URL.Query().Get("page"))
    pageSize, _ := strconv.Atoi(r.URL.Query().Get("size"))
    
    // Build options
    var opts []sift.Option
    
    if filterStr != "" {
        filter, err := sift.Parse(filterStr, nil)
        if err != nil {
            http.Error(w, "Invalid filter", http.StatusBadRequest)
            return
        }
        opts = append(opts, sift.WithFilter(filter))
    }
    
    if sortStr != "" {
        // Parse sort string (e.g., "created_at:desc,name:asc")
        sort := parseSortString(sortStr)
        opts = append(opts, sift.WithSort(sort))
    }
    
    if pageSize > 0 {
        page := sift.Paginate().Size(pageSize).Number(pageNum)
        opts = append(opts, sift.WithPagination(page))
    }
    
    users, err := h.repo.Find(r.Context(), opts...)
    // ...
}
```

### Testing

```go
func TestUserFiltering(t *testing.T) {
    filter := sift.Eq("status", "active")
    sort := sift.Sort("created_at", sift.SortDesc)
    page := sift.Paginate().Size(20).Number(1)
    
    // Test with different backends
    t.Run("DynamoDB", func(t *testing.T) {
        adapter := dynamodb.NewAdapter()
        err := sift.Thru(ctx, adapter,
            sift.WithFilter(filter),
            sift.WithSort(sort),
            sift.WithPagination(page))
        // assertions...
    })
    
    t.Run("PostgreSQL", func(t *testing.T) {
        adapter := postgres.NewAdapter()
        err := sift.Thru(ctx, adapter,
            sift.WithFilter(filter),
            sift.WithSort(sort),
            sift.WithPagination(page))
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

### Development Setup

```bash
# Clone the repository
git clone https://github.com/nisimpson/sift.git
cd sift

# Install development tools
make install-tools

# Run tests
make test

# Run all checks (formatting, linting, tests)
make check
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run tests without race detector (faster)
make test-short

# Run benchmarks
make bench
```

### Code Quality

```bash
# Format code
make fmt

# Run linter
make lint

# Run go vet
make vet

# Run all checks
make check
```

### Continuous Integration

All pull requests are automatically tested with:
- Multiple Go versions (1.21, 1.22, 1.23)
- Race detector
- golangci-lint
- Formatting checks

See [.github/workflows/README.md](.github/workflows/README.md) for details.

## License

MIT License - see [LICENSE](LICENSE) file for details

## Acknowledgments

Inspired by the need for consistent filtering across multiple data backends in modern applications.
