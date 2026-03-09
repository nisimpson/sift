# Sift Formatting and Parsing

This document describes the unified formatting and parsing API for Sift expressions.

## Overview

Sift now supports formatting and parsing for:
- Filter expressions (existing)
- Sort expressions (new)
- Pagination expressions (new)

All three can be combined in a single `Format()` call using the option pattern.

## Unified Format API

```go
// Format with all options
str, err := sift.Format(registry,
    sift.WithFilter(filterExpr),
    sift.WithSort(sortExpr),
    sift.WithPagination(pageExpr),
)
// Output: "filter(and(eq(status,active),gt(age,18))),sort(created_at:desc,name:asc),page(size:20,number:2)"

// Format individual components
str, err := sift.Format(registry, sift.WithFilter(filterExpr))
str, err := sift.Format(registry, sift.WithSort(sortExpr))
str, err := sift.Format(registry, sift.WithPagination(pageExpr))
```

## Standalone Parse Functions

The main public API is `ParseQuery()` which handles complete query strings:

```go
// Parse complete query string
query, err := sift.ParseQuery("filter(and(eq(status,active),gt(age,18))),sort(created_at:desc),page(size:20,number:2)", registry)
// query.Filter, query.Sort, query.Pagination are all populated
```

For backward compatibility, individual parse functions are still available but deprecated:

```go
// Use ParseQuery for all parsing
query, err := sift.ParseQuery("filter(and(eq(status,active),gt(age,18)))", registry)
filter := query.Filter

query, err = sift.ParseQuery("sort(created_at:desc,name:asc)", nil)
sort := query.Sort

query, err = sift.ParseQuery("page(size:20,number:2)", nil)
page := query.Pagination
```

The `ParseQuery()` function is smart enough to extract filter, sort, and pagination from a single combined string. It handles:
- Any combination of components (all three, just filter, just sort, etc.)
- Proper parenthesis matching to avoid splitting inside nested expressions
- Whitespace tolerance
- Returns a `Query` struct with separate fields for each component

## Format Specifications

### Filter Format

Uses prefix notation (unchanged from before):

```
eq(field,value)          // equal
ne(field,value)          // not equal
lt(field,value)          // less than
le(field,value)          // less than or equal
gt(field,value)          // greater than
ge(field,value)          // greater than or equal
contains(field,value)    // substring match
begins_with(field,value) // prefix match
in(field,value)          // value in list
exists(field)            // field exists
not_exists(field)        // field doesn't exist
between(field,value)     // value between bounds
and(expr1,expr2)         // logical AND
or(expr1,expr2)          // logical OR
not(expr)                // logical NOT
```

### Sort Format

Colon-separated format:

```
field:asc                // single field ascending
field:desc               // single field descending
field:asc:nullslast      // with nulls last
field1:desc,field2:asc   // multiple fields (comma-separated)
```

### Pagination Format

Key-value pairs separated by colons and commas:

```
size:20,number:2         // offset-based (page number)
size:20,cursor:token123  // cursor-based
```

## Escaping

Special characters are escaped with backslash:
- Backslash: `\\`
- Comma: `\,`
- Parenthesis: `\(` and `\)`
- Colon: `\:`

Examples:
```
eq(path,C\:\\Users)              // Path with backslash and colon
sort(user\\:name:asc)            // Field name with colon
page(size:20,cursor:token\\,123) // Cursor with comma
```

## With* Options

The `With*` functions return `FormatOption` which implements both:
- `FormatOption` interface (for use with `Format()`)
- `Option` interface (for use with `Thru()`)

This means you can use the same options with both APIs:

```go
// With Format
str, _ := sift.Format(registry, sift.WithFilter(f), sift.WithSort(s))

// With Thru
sift.Thru(ctx, adapter, sift.WithFilter(f), sift.WithSort(s))
```

## Query Type

The `Query` struct holds all three expression types:

```go
type Query struct {
    Filter     Expression
    Sort       SortExpression
    Pagination PaginationExpression
}
```

This is returned by `ParseQuery()` and can be used to pass around complete queries:

```go
// Parse once
query, _ := sift.ParseQuery(queryString, registry)

// Use in multiple places
sift.Thru(ctx, adapter1, 
    sift.WithFilter(query.Filter),
    sift.WithSort(query.Sort),
    sift.WithPagination(query.Pagination))

sift.Thru(ctx, adapter2,
    sift.WithFilter(query.Filter),
    sift.WithSort(query.Sort),
    sift.WithPagination(query.Pagination))
```

## Format-Specific Implementations

Each format package (like `jsonapi`) provides its own `Format()` and `ParseQuery()` functions:

```go
// JSON:API format - filter only
values, err := jsonapi.Format(filterExpr, registry)
// Returns url.Values with filter[q]=..., filter[p1]=...

expr, err := jsonapi.Parse(r.URL.Query(), registry)

// JSON:API format - complete query
query, err := jsonapi.ParseQuery(r.URL.Query(), registry)
// Parses filter[q]=..., sort[field]=..., page[size]=... into Query struct
```

### JSON:API Query Format

JSON:API uses bracket notation for all components:

**Filter:**
```
filter[q]=and(p1,p2)&filter[p1]=eq(status,active)&filter[p2]=gt(age,18)
```

**Sort:**
```
sort[created_at]=desc&sort[name]=asc
```

**Pagination:**
```
page[size]=20&page[number]=2          // Offset-based
page[size]=20&page[cursor]=token123   // Cursor-based
```

**Complete Query:**
```
filter[q]=and(p1,p2)&filter[p1]=eq(status,active)&filter[p2]=gt(age,18)&sort[created_at]=desc&page[size]=20&page[number]=2
```

This keeps format implementations independent while sharing the same option types and Query struct.

## Examples

### Complete Query

```go
filter := sift.Eq("status", "active").And(sift.Gt("age", 18))
sort := sift.Sort("created_at", sift.SortDesc).ThenBy("name", sift.SortAsc)
page := sift.Paginate().Size(20).Number(2)

str, _ := sift.Format(nil,
    sift.WithFilter(filter),
    sift.WithSort(sort),
    sift.WithPagination(page),
)
// "filter(and(eq(status,active),gt(age,18))),sort(created_at:desc,name:asc),page(size:20,number:2)"
```

### HTTP Query String

```go
// Option 1: Parse from separate query parameters
// GET /users?filter=and(eq(status,active),gt(age,18))&sort=created_at:desc&page=size:20,number:2

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query()
    
    // Build combined query string from separate parameters
    var parts []string
    if filterStr := query.Get("filter"); filterStr != "" {
        parts = append(parts, "filter("+filterStr+")")
    }
    
    if sortStr := query.Get("sort"); sortStr != "" {
        parts = append(parts, "sort("+sortStr+")")
    }
    
    if pageStr := query.Get("page"); pageStr != "" {
        parts = append(parts, "page("+pageStr+")")
    }
    
    // Parse all at once
    q, _ := sift.ParseQuery(strings.Join(parts, ","), registry)
    
    adapter := dynamodb.NewAdapter()
    sift.Thru(ctx, adapter,
        sift.WithFilter(q.Filter),
        sift.WithSort(q.Sort),
        sift.WithPagination(q.Pagination),
    )
}

// Option 2: Parse from single combined query parameter
// GET /users?q=filter(eq(status,active)),sort(created_at:desc),page(size:20,number:2)

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
    queryStr := r.URL.Query().Get("q")
    
    query, err := sift.ParseQuery(queryStr, registry)
    if err != nil {
        http.Error(w, "Invalid query", http.StatusBadRequest)
        return
    }
    
    adapter := dynamodb.NewAdapter()
    sift.Thru(ctx, adapter,
        sift.WithFilter(query.Filter),
        sift.WithSort(query.Sort),
        sift.WithPagination(query.Pagination),
    )
}
```

### Round-Trip

```go
// Format
original := sift.Sort("created_at", sift.SortDesc).ThenBy("name", sift.SortAsc)
formatted, _ := sift.Format(nil, sift.WithSort(original))
// "sort(created_at:desc,name:asc)"

// Parse
query, _ := sift.ParseQuery("sort(created_at:desc,name:asc)", nil)

// Format again
reformatted, _ := sift.Format(nil, sift.WithSort(query.Sort))
// "sort(created_at:desc,name:asc)" - identical to original
```

## API Reference

The main parsing API:
- `ParseQuery(s, registry)` - Parse complete query strings with filter, sort, and pagination

The main formatting APIs:
- `Format(registry, ...FormatOption)` - Unified formatting with options
- `FormatFilter(expr, registry)` - Format filter expressions only

## Design Principles

1. **Unified formatting** - One `Format()` function with options for all expression types
2. **Unified parsing** - One `ParseQuery()` function that handles all components
3. **Consistent options** - Same `With*()` functions work with both `Format()` and `Thru()`
4. **Format independence** - Each format package (jsonapi, etc.) owns its implementation
5. **URL-safe** - All formats use characters safe for query strings
6. **Escape-aware** - Proper handling of special characters in values
