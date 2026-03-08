# Sift - Universal Query Filter Library for Go

## Overview

Sift is a Go library that provides a universal filter expression language using an Abstract Syntax Tree (AST). Write your filter logic once, then implement backend-specific evaluators to translate it into native queries for DynamoDB, SQL databases, MongoDB, Elasticsearch, or any other data store.

## Core Concept

Sift uses the visitor pattern to traverse a filter AST. You define filter expressions using Sift's types (`Condition`, `AndOperation`, `OrOperation`, `NotOperation`), then implement evaluator interfaces for your specific backends. The library provides a single entry point `sift.Thru(ctx, evaluator, filter)` that walks the tree and calls your evaluator methods.

```go
// Define a filter once
filter := &sift.AndOperation{
    Left: &sift.Condition{
        Name:      "status",
        Operation: sift.OperationEQ,
        Value:     "active",
    },
    Right: &sift.Condition{
        Name:      "email",
        Operation: sift.OperationContains,
        Value:     "@example.com",
    },
}

// Use with DynamoDB
dynamoAdapter := dynamodb.NewAdapter()
sift.Thru(ctx, dynamoAdapter, filter)
// Generates: status = :v1 AND contains(email, :v2)

// Use with SQL
sqlAdapter := sql.NewAdapter()
sift.Thru(ctx, sqlAdapter, filter)
// Generates: status = ? AND email LIKE ?
```

## Key Features

### Comprehensive Operation Support

**Comparison Operations:**
- `OperationEQ`, `OperationNEQ` - equality/inequality
- `OperationLT`, `OperationLTE`, `OperationGT`, `OperationGTE` - relational comparisons

**String Operations:**
- `OperationContains` - substring matching
- `OperationBeginsWith` - prefix matching

**Collection Operations:**
- `OperationIn` - value in list
- `OperationBetween` - value between bounds

**Existence Operations:**
- `OperationExists` - attribute exists
- `OperationNotExists` - attribute doesn't exist

**Logical Operations:**
- `AndOperation`, `OrOperation`, `NotOperation` - full boolean logic support

### Backend Agnostic

Implement evaluator interfaces for any backend:
- DynamoDB
- PostgreSQL / MySQL / SQLite
- MongoDB
- Elasticsearch
- Redis
- In-memory filtering
- Custom query languages

### Type Safe

Uses Go's type system and interfaces to ensure correct implementation:

```go
// Adapter interface - backends implement this
type Adapter interface {
    Evaluator(ctx context.Context) *Evaluator
}

// Core evaluator struct returned by Adapter.Evaluator()
type Evaluator struct {
    ConditionEvaluator
    AndEvaluator
    OrEvaluator
    NotEvaluator
    CustomEvaluator
}

// Implement only the interfaces you need
type ConditionEvaluator interface {
    EvaluateCondition(ctx context.Context, node *Condition) error
}

type AndEvaluator interface {
    EvaluateAnd(ctx context.Context, node *AndOperation) error
}

type OrEvaluator interface {
    EvaluateOr(ctx context.Context, node *OrOperation) error
}

type NotEvaluator interface {
    EvaluateNot(ctx context.Context, node *NotOperation) error
}

type CustomEvaluator interface {
    EvaluateCustom(ctx context.Context, node CustomExpression) error
}
```

The `Operation` type is a typed constant, providing compile-time safety for switch statements in evaluator implementations.

### Composable

Build complex filters by composing simple operations:

```go
filter := &sift.OrOperation{
    Left: &sift.AndOperation{
        Left: &sift.Condition{
            Name:      "role",
            Operation: sift.OperationEQ,
            Value:     "admin",
        },
        Right: &sift.Condition{
            Name:      "verified",
            Operation: sift.OperationEQ,
            Value:     "true",
        },
    },
    Right: &sift.Condition{
        Name:      "email",
        Operation: sift.OperationBeginsWith,
        Value:     "support@",
    },
}
// (role = "admin" AND verified = true) OR email begins_with "support@"
```

### Serializable

Built-in serialization to URL-safe prefix notation:

```go
// Serialize to string
str, _ := sift.Format(filter)
// "and(eq(status,active),gt(age,18))"

// Parse from string
filter, _ := sift.Parse("and(eq(status,active),gt(age,18))")

// Perfect for URL query strings
// GET /users?filter=and(eq(status,active),gt(age,18))
```

## Use Cases

### Multi-Backend Applications

Build applications that can switch between different databases without rewriting query logic:

```go
type UserRepository interface {
    Find(ctx context.Context, filter sift.Expression) ([]*User, error)
}

type DynamoUserRepo struct {
    client *dynamodb.Client
}

func (r *DynamoUserRepo) Find(ctx context.Context, filter sift.Expression) ([]*User, error) {
    adapter := dynamodb.NewAdapter()
    sift.Thru(ctx, adapter, filter)
    // Use adapter.Expression(), adapter.Names(), adapter.Values()
}

type PostgresUserRepo struct {
    db *sql.DB
}

func (r *PostgresUserRepo) Find(ctx context.Context, filter sift.Expression) ([]*User, error) {
    adapter := postgres.NewAdapter()
    sift.Thru(ctx, adapter, filter)
    // Use adapter.Query(), adapter.Args()
}
```

### API Query Parameters

Parse user-provided filter strings from query parameters:

```go
// GET /users?filter=and(eq(status,active),eq(role,admin))
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
    filterStr := r.URL.Query().Get("filter")
    filter, err := sift.Parse(filterStr)
    if err != nil {
        http.Error(w, "Invalid filter", http.StatusBadRequest)
        return
    }
    
    users, err := h.repo.Find(r.Context(), filter)
    // ...
}
```

### Testing

Write tests once that work across all backends:

```go
func TestUserFiltering(t *testing.T) {
    filter := &sift.Condition{
        Name:      "status",
        Operation: sift.OperationEQ,
        Value:     "active",
    }
    
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

### Query Builder UI

Build visual query builders that generate Sift expressions:

```go
// User selects: "email" "contains" "@gmail.com"
filter := &sift.Condition{
    Name:      "email",
    Operation: sift.OperationContains,
    Value:     "@gmail.com",
}

// Serialize for storage or transmission
filterStr, _ := sift.Format(filter)
// "contains(email,@gmail.com)"
```

## Implementation Pattern

### Adapter Structure

Backend adapters follow this pattern:

```go
type MyAdapter struct {
    // Accumulator fields for building queries
    query  string
    args   []interface{}
}

func NewAdapter() *MyAdapter {
    return &MyAdapter{
        args: make([]interface{}, 0),
    }
}

// Implement the Adapter interface
func (a *MyAdapter) Evaluator(ctx context.Context) *sift.Evaluator {
    // Return evaluator with only the interfaces you support
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

// Expose getters for accumulated results
func (a *MyAdapter) Query() string { return a.query }
func (a *MyAdapter) Args() []interface{} { return a.args }
```

## Architecture

```
┌─────────────────────────────────────────┐
│         Application Layer               │
│  (Business Logic, API Handlers)         │
└─────────────────┬───────────────────────┘
                  │
                  │ Uses Sift Expressions
                  │
┌─────────────────▼───────────────────────┐
│            Sift Library                 │
│  ┌─────────────────────────────────┐   │
│  │  Expression Types (AST)         │   │
│  │  - Condition                    │   │
│  │  - AndOperation                 │   │
│  │  - OrOperation                  │   │
│  │  - NotOperation                 │   │
│  │  - CustomExpression             │   │
│  └─────────────────────────────────┘   │
│  ┌─────────────────────────────────┐   │
│  │  Evaluator Interfaces           │   │
│  │  - ConditionEvaluator           │   │
│  │  - AndEvaluator                 │   │
│  │  - OrEvaluator                  │   │
│  │  - NotEvaluator                 │   │
│  │  - CustomEvaluator              │   │
│  └─────────────────────────────────┘   │
│  ┌─────────────────────────────────┐   │
│  │  Serialization                  │   │
│  │  - Format() / Parse()           │   │
│  │  - CustomFormatter registry     │   │
│  └─────────────────────────────────┘   │
└─────────────────┬───────────────────────┘
                  │
                  │ Implements Evaluators
                  │
┌─────────────────▼───────────────────────┐
│      Backend Implementations            │
│  ┌──────────┐  ┌──────────┐  ┌───────┐ │
│  │ DynamoDB │  │   SQL    │  │ Mongo │ │
│  │ Evaluator│  │ Evaluator│  │ Eval  │ │
│  └──────────┘  └──────────┘  └───────┘ │
└─────────────────────────────────────────┘
```

## Custom Expressions

Extend Sift with backend-specific operations:

```go
// Define custom expression type
type GeoWithinExpression struct {
    Field  string
    Lat    float64
    Lng    float64
    Radius float64
}

func (g *GeoWithinExpression) Type() string {
    return "geo_within"
}

// Define formatter for serialization
type GeoFormatter struct{}

func (GeoFormatter) FormatCustomExpression(expr sift.CustomExpression) (string, error) {
    g := expr.(*GeoWithinExpression)
    return fmt.Sprintf("geo_within(%s,%f,%f,%f)", g.Field, g.Lat, g.Lng, g.Radius), nil
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

// Register the custom expression
func init() {
    sift.RegisterCustomExpression(&GeoWithinExpression{}, GeoFormatter{})
}

// Use it
expr := sift.NewCustomExpression(&GeoWithinExpression{
    Field:  "location",
    Lat:    40.7128,
    Lng:    -74.0060,
    Radius: 5000,
})
```

## Implementation Status

### Completed
- [x] Core AST expression types (Condition, AndOperation, OrOperation, NotOperation)
- [x] Visitor pattern with `sift.Thru()` entry point
- [x] Evaluator interfaces (ConditionEvaluator, AndEvaluator, OrEvaluator, NotEvaluator, CustomEvaluator)
- [x] Typed Operation constants for compile-time safety
- [x] Serialization/deserialization (Format/Parse) with URL-safe prefix notation
- [x] Custom expression support with registry and formatter interface
- [x] Comprehensive documentation and examples

### In Progress
- [ ] Backend implementations (DynamoDB, PostgreSQL, etc.)
- [ ] Comprehensive test suite
- [ ] Example applications

### Future
- [ ] Query optimization hints
- [ ] Type validation utilities
- [ ] CLI tool for testing filters
- [ ] Web playground

## Design Principles

1. **Single Entry Point**: All filter evaluation goes through `sift.Thru(ctx, adapter, filter)` - the `accept()` method is private to enforce this pattern.

2. **Adapter Pattern**: Backends implement the `Adapter` interface with an `Evaluator(ctx)` method that returns a configured `*sift.Evaluator`. This allows context-aware evaluator creation.

3. **Stateful Accumulator**: Adapters accumulate state during traversal. Implementors expose getters to retrieve results after `Thru()` completes.

4. **Implement What You Need**: The `Evaluator` struct embeds interfaces - return only the ones your backend supports in `Evaluator(ctx)`. Unsupported operations return `ErrUnsupported`.

5. **Type Safety**: The `Operation` type provides compile-time checking for switch statements in evaluator implementations.

6. **Extensibility**: Custom expressions allow backend-specific operations while integrating with the standard AST.

7. **URL-Safe Serialization**: Prefix notation format is unambiguous and works in query strings without encoding issues.

## Key Naming Conventions

- `Expression` - Interface for any part of the filter tree
- `Condition` - Struct for leaf nodes (field comparisons)
- `AndOperation`, `OrOperation`, `NotOperation` - Structs for logical operations
- `CustomExpression` - Interface for custom expression types
- `Operation` - Typed constant for comparison operations (OperationEQ, OperationGT, etc.)

## Why "Sift"?

The name captures what this library does:
- **Sifting** through data to find what you need
- Short, memorable, easy to type
- Evokes filtering and separation
- Available as a Go package name
