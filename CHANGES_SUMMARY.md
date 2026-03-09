# Summary of Changes

## Overview

Implemented unified formatting and parsing for Sift expressions with a focus on `ParseQuery()` as the main public API.

## Key Changes

### 1. Unified Format API
- `Format(registry, ...FormatOption)` - Accepts options for filter, sort, and pagination
- `WithFilter()`, `WithSort()`, `WithPagination()` - Work with both `Format()` and `Thru()`
- Output format: `filter(...),sort(...),page(...)`

### 2. ParseQuery as Main API
- **`ParseQuery(s, registry)`** - Main public API for parsing complete queries
- Extracts filter, sort, and pagination from a single combined string
- Returns `Query` struct with all three components
- Handles any combination of components (all, some, or one)
- Smart parenthesis matching to avoid splitting inside nested expressions

### 3. Removed Individual Parse Functions
Individual parse functions have been removed from the public API:
- `ParseFilter()` - Removed (use `ParseQuery()` instead)
- `ParseSort()` - Removed (use `ParseQuery()` instead)
- `ParsePagination()` - Removed (use `ParseQuery()` instead)
- `Parse()` - Removed (use `ParseQuery()` instead)

Internal lowercase versions exist for use by `ParseQuery()`:
- `parseFilter()` - Internal use only
- `parseSort()` - Internal use only
- `parsePagination()` - Internal use only

### 4. JSON:API Support
Added `ParseQuery()` to jsonapi package:
- `jsonapi.ParseQuery(url.Values, registry)` - Parses complete JSON:API queries
- Supports filter, sort, and pagination in JSON:API bracket notation
- Filter: `filter[q]=...&filter[p1]=...`
- Sort: `sort[field]=asc`
- Pagination: `page[size]=20&page[number]=2` or `page[size]=20&page[cursor]=token`

## Format Specifications

### Standard Format (sift.Format)
```
filter(and(eq(status,active),gt(age,18))),sort(created_at:desc,name:asc),page(size:20,number:2)
```

### JSON:API Format (jsonapi.ParseQuery)
```
filter[q]=and(p1,p2)&filter[p1]=eq(status,active)&filter[p2]=gt(age,18)&sort[created_at]=desc&sort[name]=asc&page[size]=20&page[number]=2
```

## Usage Examples

### Standard Format
```go
// Format
str, _ := sift.Format(registry,
    sift.WithFilter(filterExpr),
    sift.WithSort(sortExpr),
    sift.WithPagination(pageExpr))

// Parse
query, _ := sift.ParseQuery(str, registry)

// Use
sift.Thru(ctx, adapter,
    sift.WithFilter(query.Filter),
    sift.WithSort(query.Sort),
    sift.WithPagination(query.Pagination))
```

### JSON:API Format
```go
// Parse from HTTP request
query, _ := jsonapi.ParseQuery(r.URL.Query(), registry)

// Use
sift.Thru(ctx, adapter,
    sift.WithFilter(query.Filter),
    sift.WithSort(query.Sort),
    sift.WithPagination(query.Pagination))
```

## Design Rationale

### Why ParseQuery as Main API?

1. **Simpler mental model** - One function to parse everything
2. **More flexible** - Handles any combination of components
3. **Better for HTTP** - Parse once from a single query parameter
4. **Cleaner code** - No need to call three separate functions
5. **Easier error handling** - One call, one error check

### Why Deprecate Individual Parse Functions?

1. **Reduces API surface** - Fewer functions to learn and maintain
2. **Encourages best practice** - Use the unified API
3. **Still available** - Backward compatibility maintained
4. **Internal use** - Unexported versions used internally by ParseQuery

### Why Keep Format Packages Independent?

1. **Different output types** - Standard returns string, JSON:API returns url.Values
2. **Different semantics** - Each format has unique conventions
3. **Clear ownership** - Each package owns its implementation
4. **Easier to extend** - Add new formats without modifying core

## Files Modified

### Core Package
- `marshal.go` - Added ParseQuery, Query struct, made parse functions internal
- `marshal_test.go` - Tests for ParseQuery and round-trip validation
- `sift.go` - Removed duplicate With* declarations
- `builder_test.go` - Updated to use new API

### JSON:API Package
- `thru/jsonapi/marshal.go` - Added ParseQuery with sort/pagination support
- `thru/jsonapi/marshal_test.go` - Tests for ParseQuery
- `thru/jsonapi/README.md` - Updated documentation

### Other Packages
- `thru/dynamodb/*_test.go` - Updated to use FormatFilter/ParseFilter
- `thru/exprlang/*_test.go` - Updated to use FormatFilter/ParseFilter

### Documentation
- `FORMATTING.md` - Complete formatting and parsing guide
- `EXAMPLE_PARSEQUERY.md` - Detailed examples and use cases
- `CHANGES_SUMMARY.md` - This file

## Migration Guide

### Before
```go
// Separate parsing
filter, _ := sift.ParseFilter(filterStr, registry)
sort, _ := sift.ParseSort(sortStr)
page, _ := sift.ParsePagination(pageStr)

sift.Thru(ctx, adapter,
    sift.WithFilter(filter),
    sift.WithSort(sort),
    sift.WithPagination(page))
```

### After
```go
// Unified parsing
query, _ := sift.ParseQuery(queryStr, registry)

sift.Thru(ctx, adapter,
    sift.WithFilter(query.Filter),
    sift.WithSort(query.Sort),
    sift.WithPagination(query.Pagination))
```

### HTTP Handler - Before
```go
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
    q := r.URL.Query()
    
    var filter sift.Expression
    if s := q.Get("filter"); s != "" {
        filter, _ = sift.ParseFilter(s, registry)
    }
    
    var sort sift.SortExpression
    if s := q.Get("sort"); s != "" {
        sort, _ = sift.ParseSort(s)
    }
    
    var page sift.PaginationExpression
    if s := q.Get("page"); s != "" {
        page, _ = sift.ParsePagination(s)
    }
    
    // Use filter, sort, page...
}
```

### HTTP Handler - After (Option 1: Separate Parameters)
```go
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
    q := r.URL.Query()
    
    // Build combined query string
    var parts []string
    if s := q.Get("filter"); s != "" {
        parts = append(parts, "filter("+s+")")
    }
    if s := q.Get("sort"); s != "" {
        parts = append(parts, "sort("+s+")")
    }
    if s := q.Get("page"); s != "" {
        parts = append(parts, "page("+s+")")
    }
    
    query, _ := sift.ParseQuery(strings.Join(parts, ","), registry)
    // Use query.Filter, query.Sort, query.Pagination...
}
```

### HTTP Handler - After (Option 2: Single Parameter)
```go
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
    // GET /users?q=filter(eq(status,active)),sort(created_at:desc),page(size:20,number:2)
    queryStr := r.URL.Query().Get("q")
    
    query, _ := sift.ParseQuery(queryStr, registry)
    // Use query.Filter, query.Sort, query.Pagination...
}
```

### HTTP Handler - After (Option 3: JSON:API)
```go
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
    // GET /users?filter[q]=...&sort[field]=...&page[size]=...
    query, _ := jsonapi.ParseQuery(r.URL.Query(), registry)
    // Use query.Filter, query.Sort, query.Pagination...
}
```

## Testing

All tests passing:
- Core package: Format, ParseQuery, round-trip validation
- JSON:API package: ParseQuery with filter, sort, pagination
- Backward compatibility: Deprecated functions still work
- All thru packages: Updated and passing

## Backward Compatibility

**Breaking Change**: The library has not been published yet, so the following functions were removed without deprecation:
- `Parse()` - Use `ParseQuery()` instead
- `ParseFilter()` - Use `ParseQuery()` instead
- `ParseSort()` - Use `ParseQuery()` instead
- `ParsePagination()` - Use `ParseQuery()` instead

Existing functionality:
- `FormatFilter()` available for formatting filter expressions
- `Format()` available for unified formatting with options

## Future Considerations

1. **Additional formats** - GraphQL, OData, etc. can follow the same pattern
2. **Query validation** - Could add validation helpers for Query struct
3. **Query builders** - Could add fluent builders for Query construction
4. **Serialization** - Could add JSON/YAML serialization for Query struct
