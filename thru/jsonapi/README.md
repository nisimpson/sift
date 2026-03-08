# Sift JSON:API Adapter

A JSON:API filter adapter for the [sift](https://github.com/nisimpson/sift) universal filter library. This adapter provides bidirectional conversion between JSON:API query parameters and Sift expressions, and can wrap other adapters to enable JSON:API support for any backend.

## Installation

```bash
go get github.com/nisimpson/sift/thru/jsonapi
```

## Overview

The JSON:API adapter serves three main purposes:

1. **Parse** JSON:API query parameters → Sift expressions
2. **Format** Sift expressions → JSON:API query parameters  
3. **Wrap** other adapters to enable direct JSON:API → Backend conversion

## JSON:API Filter Format

The adapter uses an opinionated JSON:API filter format:

```
?filter[q]=and(p1,p2)&filter[p1]=eq(status,active)&filter[p2]=gt(age,18)
```

- `filter[q]` - Main query expression with logical structure
- `filter[p1]`, `filter[p2]`, etc. - Parameter definitions for leaf conditions
- All conditions are extracted as parameters
- Logical operations (and/or/not) reference parameters

## Usage

### Parsing JSON:API to Sift

```go
import (
    "net/url"
    
    "github.com/nisimpson/sift"
    "github.com/nisimpson/sift/thru/jsonapi"
)

// Parse JSON:API query parameters
query := "filter[q]=and(p1,p2)&filter[p1]=eq(status,active)&filter[p2]=gt(age,18)"
values, _ := url.ParseQuery(query)

adapter := jsonapi.NewAdapter()
filter, err := adapter.Parse(values)
if err != nil {
    log.Fatal(err)
}

// Use with any backend
dynamoAdapter := dynamodb.NewAdapter()
sift.Thru(ctx, dynamoAdapter, filter)
```

### Formatting Sift to JSON:API

```go
// Create a Sift expression
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

// Format to JSON:API
adapter := jsonapi.NewAdapter()
values, err := adapter.Format(filter)
if err != nil {
    log.Fatal(err)
}

// Get query string
queryString := values.Encode()
// filter%5Bq%5D=and%28p1%2Cp2%29&filter%5Bp1%5D=eq%28status%2Cactive%29&filter%5Bp2%5D=gt%28age%2C18%29

// Access components
mainQuery := values.Get("filter[q]")        // "and(p1,p2)"
param1 := values.Get("filter[p1]")          // "eq(status,active)"
param2 := values.Get("filter[p2]")          // "gt(age,18)"
```

### Wrapping Other Adapters

The most powerful feature is wrapping other adapters to enable direct JSON:API → Backend conversion:

```go
import (
    "github.com/nisimpson/sift/thru/dynamodb"
    "github.com/nisimpson/sift/thru/jsonapi"
)

// Create backend adapter (DynamoDB, expr-lang, SQL, etc.)
dynamoAdapter := dynamodb.NewAdapter()

// Wrap with JSON:API support
wrapped := jsonapi.Wrap(dynamoAdapter)

// Parse JSON:API and evaluate in one step
query := "filter[q]=and(p1,p2)&filter[p1]=eq(status,active)&filter[p2]=gt(age,18)"
values, _ := url.ParseQuery(query)

err := wrapped.ParseThru(ctx, values)
if err != nil {
    log.Fatal(err)
}

// Access the backend adapter's results
expr, _ := dynamoAdapter.Expression()

// Use with DynamoDB
result, err := client.Scan(ctx, &dynamodb.ScanInput{
    TableName:                 aws.String("Users"),
    FilterExpression:          expr.Condition(),
    ExpressionAttributeNames:  expr.Names(),
    ExpressionAttributeValues: expr.Values(),
})
```

## Supported Operations

All Sift operations are supported:

| Sift Operation | JSON:API Format | Example |
|----------------|-----------------|---------|
| `OperationEQ` | `eq(field,value)` | `eq(status,active)` |
| `OperationNEQ` | `neq(field,value)` | `neq(status,deleted)` |
| `OperationLT` | `lt(field,value)` | `lt(age,18)` |
| `OperationLTE` | `lte(field,value)` | `lte(price,100)` |
| `OperationGT` | `gt(field,value)` | `gt(age,18)` |
| `OperationGTE` | `gte(field,value)` | `gte(score,90)` |
| `OperationContains` | `contains(field,value)` | `contains(email,@example.com)` |
| `OperationBeginsWith` | `beginsWith(field,value)` | `beginsWith(name,John)` |
| `OperationIn` | `in(field,value)` | `in(role,admin)` |
| `OperationBetween` | `between(field,value)` | `between(age,18,65)` |
| `OperationExists` | `exists(field)` | `exists(email)` |
| `OperationNotExists` | `notExists(field)` | `notExists(deleted_at)` |

Logical operations:
- `and(p1,p2)` - Logical AND
- `or(p1,p2)` - Logical OR
- `not(p1)` - Logical NOT

## Examples

### Simple Condition

```
?filter[q]=p1&filter[p1]=eq(status,active)
```

Parses to:
```go
&sift.Condition{
    Name:      "status",
    Operation: sift.OperationEQ,
    Value:     "active",
}
```

### AND Operation

```
?filter[q]=and(p1,p2)&filter[p1]=eq(status,active)&filter[p2]=gt(age,18)
```

Parses to:
```go
&sift.AndOperation{
    Left:  &sift.Condition{Name: "status", Operation: sift.OperationEQ, Value: "active"},
    Right: &sift.Condition{Name: "age", Operation: sift.OperationGT, Value: "18"},
}
```

### Nested Operations

```
?filter[q]=and(p1,or(p2,p3))&filter[p1]=eq(status,active)&filter[p2]=gt(age,18)&filter[p3]=eq(role,admin)
```

Parses to:
```go
&sift.AndOperation{
    Left: &sift.Condition{Name: "status", Operation: sift.OperationEQ, Value: "active"},
    Right: &sift.OrOperation{
        Left:  &sift.Condition{Name: "age", Operation: sift.OperationGT, Value: "18"},
        Right: &sift.Condition{Name: "role", Operation: sift.OperationEQ, Value: "admin"},
    },
}
```

## Use Cases

### API Server with Multiple Backends

Build an API that accepts JSON:API filters and works with any backend:

```go
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
    // Parse JSON:API filters from query string
    adapter := jsonapi.NewAdapter()
    filter, err := adapter.Parse(r.URL.Query())
    if err != nil {
        http.Error(w, "Invalid filter", http.StatusBadRequest)
        return
    }
    
    // Use with your backend (DynamoDB, SQL, etc.)
    users, err := h.repo.Find(r.Context(), filter)
    // ...
}
```

### Client Library

Generate JSON:API query strings from Sift expressions:

```go
// Build filter programmatically
filter := &sift.AndOperation{
    Left:  &sift.Condition{Name: "status", Operation: sift.OperationEQ, Value: "active"},
    Right: &sift.Condition{Name: "age", Operation: sift.OperationGT, Value: "18"},
}

// Convert to JSON:API query string
adapter := jsonapi.NewAdapter()
values, _ := adapter.Format(filter)
queryString := values.Encode()

// Make HTTP request
resp, err := http.Get("https://api.example.com/users?" + queryString)
```

### Backend Agnostic Repository

Create repositories that work with any backend through JSON:API:

```go
type UserRepository struct {
    adapter sift.Adapter
}

func (r *UserRepository) FindWithJSONAPI(ctx context.Context, queryParams url.Values) ([]*User, error) {
    wrapped := jsonapi.Wrap(r.adapter)
    if err := wrapped.ParseThru(ctx, queryParams); err != nil {
        return nil, err
    }
    
    // Access backend-specific results
    // (implementation depends on the wrapped adapter)
    return r.executeQuery(ctx)
}
```

## Design Decisions

### Parameter Extraction

All leaf conditions (comparisons) are extracted as parameters. This provides:

- **Readability**: Main query shows logical structure clearly
- **Reusability**: Parameters can be referenced multiple times (future feature)
- **URL Safety**: Complex values are isolated in parameters
- **Consistency**: Predictable format for all queries

### Opinionated Format

The adapter uses a specific JSON:API filter format rather than supporting all possible variations. This provides:

- **Simplicity**: One way to represent filters
- **Interoperability**: Consistent format across all Sift backends
- **Maintainability**: Easier to implement and test

### Bidirectional Conversion

Supporting both Parse and Format enables:

- **Server-side**: Parse incoming JSON:API filters
- **Client-side**: Generate JSON:API query strings
- **Testing**: Round-trip validation
- **Documentation**: Generate examples from code

## Advanced Usage

### Accessing Components

After formatting, you can access individual components:

```go
adapter := jsonapi.NewAdapter()
values, _ := adapter.Format(filter)

// Main query expression
mainQuery := adapter.Query()  // "and(p1,p2)"

// Parameter definitions
params := adapter.Parameters()  // map[string]string{"p1": "eq(...)", "p2": "gt(...)"}

// Full query string
queryString := values.Encode()
```

### Custom Parameter Names

Currently, parameters are auto-generated as `p1`, `p2`, etc. This ensures consistency and avoids naming conflicts.

## Testing

The adapter includes comprehensive tests:

```bash
go test ./...
```

Tests cover:
- Parsing all operation types
- Formatting all operation types
- Round-trip conversion (Format → Parse → Format)
- Wrapper functionality
- Error cases

## Example

See [example_test.go](example_test.go) for complete working examples.

## License

MIT
