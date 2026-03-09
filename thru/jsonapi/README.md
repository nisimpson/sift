# Sift JSON:API Marshaler

Simple marshaling between JSON:API query parameters and Sift expressions.

## Installation

```bash
go get github.com/nisimpson/sift/thru/jsonapi
```

## API

Two main functions:

```go
// Parse JSON:API query parameters to complete Query (filter, sort, pagination)
func ParseQuery(query url.Values, registry *sift.Registry) (*sift.Query, error)

// Format Sift expression to JSON:API query parameters
func Format(expr sift.Expression, registry *sift.Registry) (url.Values, error)
```

## Usage

### Parse JSON:API to Sift

```go
import (
    "net/url"
    
    "github.com/nisimpson/sift"
    "github.com/nisimpson/sift/thru/jsonapi"
)

// Parse complete query (filter, sort, pagination)
query := r.URL.Query()
result, err := jsonapi.ParseQuery(query, registry)
if err != nil {
    http.Error(w, "Invalid query", http.StatusBadRequest)
    return
}

// Use with any backend
dynamoAdapter := dynamodb.NewAdapter()
sift.Thru(ctx, dynamoAdapter,
    sift.WithFilter(result.Filter),
    sift.WithSort(result.Sort),
    sift.WithPagination(result.Pagination),
)
```

### Format Sift to JSON:API

```go
// Create a filter
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
values, err := jsonapi.Format(filter)
if err != nil {
    log.Fatal(err)
}

// Use in HTTP request
queryString := values.Encode()
// filter%5Bq%5D=and%28p1%2Cp2%29&filter%5Bp1%5D=eq%28status%2Cactive%29&filter%5Bp2%5D=gt%28age%2C18%29
```

## JSON:API Format

### Filter
```
?filter[q]=and(p1,p2)&filter[p1]=eq(status,active)&filter[p2]=gt(age,18)
```

- `filter[q]` - Main query with logical structure
- `filter[p1]`, `filter[p2]` - Parameters for leaf conditions
- All conditions extracted as parameters
- Logical operations reference parameters

### Sort
```
?sort[created_at]=desc&sort[name]=asc
```

- `sort[field]` - Field name with direction (asc or desc)
- Multiple fields supported

### Pagination
```
?page[size]=20&page[number]=2          // Offset-based
?page[size]=20&page[cursor]=token123   // Cursor-based
```

- `page[size]` - Number of items per page (required)
- `page[number]` - Page number for offset-based pagination
- `page[cursor]` - Cursor token for cursor-based pagination

### Complete Query
```
?filter[q]=and(p1,p2)&filter[p1]=eq(status,active)&filter[p2]=gt(age,18)&sort[created_at]=desc&sort[name]=asc&page[size]=20&page[number]=2
```

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

## Custom Expressions

Custom expressions registered via `sift.RegisterCustomExpression()` work automatically:

```go
// Register custom expression (in your adapter package)
sift.RegisterCustomExpression(&SizeExpression{}, SizeFormatter{})

// Use in filter
filter := &sift.AndOperation{
    Left:  &sift.Condition{Name: "status", Operation: sift.OperationEQ, Value: "active"},
    Right: dynamodb.Size("tags", sift.OperationGT, 5),
}

// Format to JSON:API
values, _ := jsonapi.Format(filter, registry)
// filter[q]=and(p1,p2)&filter[p1]=eq(status,active)&filter[p2]=dynamodb_size(tags,gt,5)

// Parse back
result, _ := jsonapi.ParseQuery(values, registry)
// Works automatically if SizeFormatter is registered
```

## Use Cases

### API Server

Accept JSON:API filters in your HTTP handlers:

```go
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
    // Parse JSON:API query from query string
    query, err := jsonapi.ParseQuery(r.URL.Query(), registry)
    if err != nil {
        http.Error(w, "Invalid query", http.StatusBadRequest)
        return
    }
    
    // Use with your backend (DynamoDB, SQL, etc.)
    users, err := h.repo.Find(r.Context(), query.Filter)
    if err != nil {
        http.Error(w, "Query failed", http.StatusInternalServerError)
        return
    }
    
    json.NewEncoder(w).Encode(users)
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
values, _ := jsonapi.Format(filter, registry)
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
    // Parse JSON:API to Sift
    query, err := jsonapi.ParseQuery(queryParams, registry)
    if err != nil {
        return nil, err
    }
    
    // Evaluate with backend adapter
    if err := sift.Thru(ctx, r.adapter, sift.WithFilter(query.Filter)); err != nil {
        return nil, err
    }
    
    // Execute query (implementation depends on adapter)
    return r.executeQuery(ctx)
}
```

## Supported Operations

All Sift operations are supported:

| Sift Operation | JSON:API Format | Example |
|----------------|-----------------|---------|
| `OperationEQ` | `eq(field,value)` | `eq(status,active)` |
| `OperationNEQ` | `ne(field,value)` | `ne(status,deleted)` |
| `OperationLT` | `lt(field,value)` | `lt(age,18)` |
| `OperationLTE` | `le(field,value)` | `le(price,100)` |
| `OperationGT` | `gt(field,value)` | `gt(age,18)` |
| `OperationGTE` | `ge(field,value)` | `ge(score,90)` |
| `OperationContains` | `contains(field,value)` | `contains(email,@example.com)` |
| `OperationBeginsWith` | `begins_with(field,value)` | `begins_with(name,John)` |
| `OperationIn` | `in(field,value)` | `in(role,admin)` |
| `OperationBetween` | `between(field,value)` | `between(age,18,65)` |
| `OperationExists` | `exists(field)` | `exists(email)` |
| `OperationNotExists` | `not_exists(field)` | `not_exists(deleted_at)` |

Logical operations:
- `and(p1,p2)` - Logical AND
- `or(p1,p2)` - Logical OR
- `not(p1)` - Logical NOT

## Design

The JSON:API marshaler is intentionally simple:

- **Stateless**: No internal state, just pure functions
- **Leverages Sift**: Uses `sift.ParseQuery()` and `sift.Format()` for expression handling
- **Parameter extraction**: Automatically extracts leaf conditions as parameters
- **Custom expression support**: Works with any registered custom expressions

The marshaler focuses solely on the URL format ↔ Sift expression conversion, delegating all other concerns to the Sift library and backend adapters.

## Testing

See [marshal_test.go](marshal_test.go) for complete examples and test cases.

## License

MIT
