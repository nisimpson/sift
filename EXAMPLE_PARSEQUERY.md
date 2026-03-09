# ParseQuery Example

This example demonstrates how `ParseQuery()` can extract filter, sort, and pagination from a single combined string.

## Basic Usage

```go
package main

import (
    "fmt"
    "github.com/nisimpson/sift"
)

func main() {
    // Combined query string (e.g., from a single query parameter)
    queryString := "filter(and(eq(status,active),gt(age,18))),sort(created_at:desc,name:asc),page(size:20,number:2)"
    
    // Parse the entire query at once
    query, err := sift.ParseQuery(queryString, nil)
    if err != nil {
        panic(err)
    }
    
    // Access individual components
    fmt.Printf("Filter: %v\n", query.Filter)
    fmt.Printf("Sort: %v\n", query.Sort)
    fmt.Printf("Pagination: %v\n", query.Pagination)
    
    // Use with Thru
    adapter := myAdapter{}
    sift.Thru(ctx, adapter,
        sift.WithFilter(query.Filter),
        sift.WithSort(query.Sort),
        sift.WithPagination(query.Pagination),
    )
}
```

## Partial Queries

`ParseQuery()` handles any combination of components:

```go
// Just filter
query, _ := sift.ParseQuery("filter(eq(status,active))", nil)
// query.Filter is populated, query.Sort and query.Pagination are nil

// Just sort
query, _ := sift.ParseQuery("sort(created_at:desc)", nil)
// query.Sort is populated, others are nil

// Filter and sort, no pagination
query, _ := sift.ParseQuery("filter(eq(status,active)),sort(created_at:desc)", nil)
// query.Filter and query.Sort are populated, query.Pagination is nil
```

## HTTP Handler Example

```go
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
    // Get combined query from single parameter
    // GET /users?q=filter(eq(status,active)),sort(created_at:desc),page(size:20,number:2)
    queryString := r.URL.Query().Get("q")
    
    // Parse everything at once
    query, err := sift.ParseQuery(queryString, h.registry)
    if err != nil {
        http.Error(w, fmt.Sprintf("Invalid query: %v", err), http.StatusBadRequest)
        return
    }
    
    // Execute query
    adapter := dynamodb.NewAdapter()
    if err := sift.Thru(r.Context(), adapter,
        sift.WithFilter(query.Filter),
        sift.WithSort(query.Sort),
        sift.WithPagination(query.Pagination),
    ); err != nil {
        http.Error(w, "Query failed", http.StatusInternalServerError)
        return
    }
    
    // Get results from adapter and return
    users := adapter.Results()
    json.NewEncoder(w).Encode(users)
}
```

## Comparison: Separate vs Combined

### Separate Parameters (Traditional)
```
GET /users?filter=eq(status,active)&sort=created_at:desc&page=size:20,number:2
```

```go
// Use ParseQuery for all components
query, _ := sift.ParseQuery("filter(eq(status,active)),sort(created_at:desc),page(size:20,number:2)", registry)

// Access individual components
filter := query.Filter
sort := query.Sort
page := query.Pagination
```

**Pros:**
- More RESTful
- Easier to construct URLs manually
- Each component is independent

**Cons:**
- Multiple query parameters
- More verbose
- Need to parse each separately

### Combined Parameter (with ParseQuery)
```
GET /users?q=filter(eq(status,active)),sort(created_at:desc),page(size:20,number:2)
```

```go
query, _ := sift.ParseQuery(r.URL.Query().Get("q"), registry)
```

**Pros:**
- Single query parameter
- Parse once
- Easier to pass around as a unit
- Cleaner for complex queries

**Cons:**
- Longer URL
- Less conventional
- Harder to construct manually

## Round-Trip Example

```go
// Build a query
filter := sift.Eq("status", "active").And(sift.Gt("age", 18))
sort := sift.Sort("created_at", sift.SortDesc)
page := sift.Paginate().Size(20).Number(2)

// Format to string
queryString, _ := sift.Format(nil,
    sift.WithFilter(filter),
    sift.WithSort(sort),
    sift.WithPagination(page),
)
// "filter(and(eq(status,active),gt(age,18))),sort(created_at:desc),page(size:20,number:2)"

// Parse back
query, _ := sift.ParseQuery(queryString, nil)

// Use the parsed query
sift.Thru(ctx, adapter,
    sift.WithFilter(query.Filter),
    sift.WithSort(query.Sort),
    sift.WithPagination(query.Pagination),
)
```

## Error Handling

`ParseQuery()` validates each component and returns descriptive errors:

```go
query, err := sift.ParseQuery("filter(invalid),sort(field:baddir)", nil)
if err != nil {
    // err will indicate which component failed and why
    // e.g., "invalid sort: invalid sort direction: baddir (expected asc or desc)"
}
```

## When to Use ParseQuery

Use `ParseQuery()` for all parsing needs:
- Parse single combined query strings
- Parse individual components (just wrap with component prefix)
- Simpler API with one function to learn
- Consistent error handling

Examples:
```go
// Parse just filter
query, _ := sift.ParseQuery("filter(eq(status,active))", registry)
filter := query.Filter

// Parse just sort
query, _ := sift.ParseQuery("sort(created_at:desc)", nil)
sort := query.Sort

// Parse all components
query, _ := sift.ParseQuery("filter(...),sort(...),page(...)", registry)
```
