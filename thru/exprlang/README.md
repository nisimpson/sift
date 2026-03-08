# Sift Expr-Lang Adapter

This package provides an [expr-lang](https://expr-lang.org/) adapter for the [Sift](https://github.com/nisimpson/sift) filter library. It translates Sift filter expressions into expr-lang expression syntax, allowing you to use Sift's universal filter language with expr-lang's powerful expression evaluation engine.

## Installation

```bash
go get github.com/nisimpson/sift/thru/exprlang
```

The package automatically registers a custom formatter for expr-lang expressions when imported, enabling full serialization and parsing support with Sift's `Format()` and `Parse()` functions.

## Usage

### Basic Example

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/expr-lang/expr"
    "github.com/nisimpson/sift"
    "github.com/nisimpson/sift/thru/exprlang"
)

type User struct {
    Name   string
    Status string
    Age    int
}

func main() {
    // Create a sift filter
    filter := &sift.Condition{
        Name:      "Status",
        Operation: sift.OperationEQ,
        Value:     "active",
    }
    
    // Translate to expr-lang
    adapter := exprlang.NewAdapter()
    sift.Thru(context.Background(), adapter, filter)
    
    // Get the expression string
    exprStr := adapter.Expression()
    fmt.Println("Expression:", exprStr)
    // Output: Expression: Status == "active"
    
    // Compile and run with expr-lang
    program, _ := expr.Compile(exprStr, expr.Env(User{}))
    
    user := User{Name: "Alice", Status: "active", Age: 25}
    output, _ := expr.Run(program, user)
    fmt.Println("Match:", output.(bool))
    // Output: Match: true
}
```

### Complex Filters

```go
// (status = "active" AND age >= 18) OR role = "admin"
filter := &sift.OrOperation{
    Left: &sift.AndOperation{
        Left: &sift.Condition{
            Name:      "Status",
            Operation: sift.OperationEQ,
            Value:     "active",
        },
        Right: &sift.Condition{
            Name:      "Age",
            Operation: sift.OperationGTE,
            Value:     "18",
        },
    },
    Right: &sift.Condition{
        Name:      "Role",
        Operation: sift.OperationEQ,
        Value:     "admin",
    },
}

adapter := exprlang.NewAdapter()
sift.Thru(context.Background(), adapter, filter)

fmt.Println(adapter.Expression())
// Output: ((Status == "active") && (Age >= 18)) || (Role == "admin")
```

## Supported Operations

### Comparison Operations

| Sift Operation | Expr-Lang Syntax | Example |
|----------------|------------------|---------|
| `OperationEQ` | `==` | `status == "active"` |
| `OperationNEQ` | `!=` | `status != "deleted"` |
| `OperationLT` | `<` | `age < 18` |
| `OperationLTE` | `<=` | `age <= 65` |
| `OperationGT` | `>` | `price > 100` |
| `OperationGTE` | `>=` | `rating >= 4.5` |

### String Operations

| Sift Operation | Expr-Lang Syntax | Example |
|----------------|------------------|---------|
| `OperationContains` | `field contains value` | `email contains "@example.com"` |
| `OperationBeginsWith` | `field startsWith value` | `name startsWith "John"` |

### Collection Operations

| Sift Operation | Expr-Lang Syntax | Example |
|----------------|------------------|---------|
| `OperationIn` | `value in field` | `"admin" in roles` |
| `OperationBetween` | `field >= min and field <= max` | `age >= 18 and age <= 65` |

### Existence Operations

| Sift Operation | Expr-Lang Syntax | Example |
|----------------|------------------|---------|
| `OperationExists` | `field != nil` | `optional_field != nil` |
| `OperationNotExists` | `field == nil` | `deleted_at == nil` |

### Logical Operations

| Sift Operation | Expr-Lang Syntax | Example |
|----------------|------------------|---------|
| `AndOperation` | `&&` | `(a == 1) && (b == 2)` |
| `OrOperation` | `\|\|` | `(role == "admin") \|\| (role == "moderator")` |
| `NotOperation` | `!` | `!(status == "deleted")` |

## Type Handling

The adapter automatically detects and formats values appropriately:

- **Numbers**: `"42"` → `42`, `"3.14"` → `3.14`
- **Booleans**: `"true"` → `true`, `"false"` → `false`
- **Strings**: `"hello"` → `"hello"` (quoted)

This ensures proper type matching when evaluating expressions with expr-lang.

## Custom Expressions

The adapter supports custom expr-lang expressions for advanced use cases that go beyond standard Sift operations. This allows you to leverage the full power of expr-lang's expression language.

Custom expressions are useful for:
- Complex array filtering and transformations
- String manipulation beyond basic operations
- Date/time operations
- Predicates with nested data structures
- Any expr-lang feature not directly mapped to Sift operations

### Helper Functions

Custom expressions are created using helper functions that wrap expr-lang syntax. When serialized as Sift expressions, they use the format `exprlang(...)`.

#### RawExpression

Create a custom expression from any valid expr-lang syntax:

```go
filter := exprlang.RawExpression("len(tweets) > 10 and any(tweets, len(.Content) > 240)")

// Sift serialization: exprlang(len(tweets) > 10 and any(tweets, len(.Content) > 240))
// Expr-lang output: len(tweets) > 10 and any(tweets, len(.Content) > 240)
```

#### Predicate

Create a predicate expression (commonly used with array functions):

```go
filter := exprlang.Predicate("len(.Content) > 240")
```

#### ArrayFunction

Create an array function expression:

```go
filter := exprlang.ArrayFunction("filter", "tweets", "len(.Content) > 240")
// Generates: filter(tweets, len(.Content) > 240)
```

#### StringFunction

Create a string function expression:

```go
filter := exprlang.StringFunction("upper", "name")
// Generates: upper(name)

filter := exprlang.StringFunction("split", "email", `","`)
// Generates: split(email, ",")
```

#### DateFunction

Create a date function expression:

```go
filter := exprlang.DateFunction("now")
// Generates: now()

filter := exprlang.DateFunction("date", `"2023-08-14"`)
// Generates: date("2023-08-14")
```

### Common Use Cases

#### Array Filtering

Filter arrays based on complex conditions:

```go
// Find users with more than 5 tweets where any tweet has > 100 characters
filter := exprlang.RawExpression("len(Tweets) > 5 and any(Tweets, len(.Content) > 100)")
```

#### Array Transformations

```go
// Check if all comments are short
filter := exprlang.RawExpression("all(Comments, len(.Text) < 200)")

// Count tweets with high engagement
filter := exprlang.RawExpression("count(Tweets, .Likes > 100) > 10")

// Find first tweet with specific content
filter := exprlang.RawExpression("find(Tweets, .Content contains 'golang') != nil")
```

#### String Operations

```go
// Case-insensitive comparison
filter := exprlang.RawExpression("lower(Name) == 'john doe'")

// String manipulation
filter := exprlang.RawExpression("len(trim(Bio)) > 50")

// Multiple string checks
filter := exprlang.RawExpression("Email contains '@' and Email endsWith '.com'")
```

#### Date/Time Operations

```go
// Recent posts (within last 24 hours)
filter := exprlang.RawExpression("now() - CreatedAt < duration('24h')")

// Posts from specific date range
filter := exprlang.RawExpression("CreatedAt >= date('2023-01-01') and CreatedAt < date('2024-01-01')")

// Weekend posts
filter := exprlang.RawExpression("CreatedAt.Weekday() in [0, 6]")
```

#### Nested Data Access

```go
// Access nested fields with optional chaining
filter := exprlang.RawExpression("Author?.Profile?.Verified == true")

// Nil coalescing
filter := exprlang.RawExpression("Author?.Name ?? 'Anonymous'")
```

#### Complex Predicates

```go
// Nested array filtering
filter := exprlang.RawExpression(`
    any(Posts, {
        let post = #;
        any(.Comments, .AuthorId == post.AuthorId)
    })
`)
```

### Combining Standard and Custom Expressions

You can mix standard Sift operations with custom expr-lang expressions:

```go
filter := sift.Eq("Status", "published").
    And(sift.In("Tags", "golang").Or(exprlang.RawExpression("len(Comments) > 10")))
```

This generates:
```
(Status == "published") && (("golang" in Tags) || (len(Comments) > 10))
```

When serialized as a Sift expression, this becomes:
```
and(eq(Status,published),or(in(Tags,golang),exprlang(len(Comments) > 10)))
```

This demonstrates how custom expressions integrate seamlessly with standard Sift operations while maintaining the ability to serialize and parse the entire filter tree.

### Available Expr-Lang Features

#### Operators

- **Arithmetic**: `+`, `-`, `*`, `/`, `%`, `^` or `**`
- **Comparison**: `==`, `!=`, `<`, `>`, `<=`, `>=`
- **Logical**: `and` or `&&`, `or` or `||`, `not` or `!`
- **String**: `contains`, `startsWith`, `endsWith`, `+` (concatenation)
- **Membership**: `in`, `.`, `?.` (optional chaining)
- **Range**: `..` (e.g., `1..10`)
- **Slice**: `[:]` (e.g., `array[1:3]`)
- **Pipe**: `|` (e.g., `name | lower() | split(" ")`)
- **Ternary**: `?:` (e.g., `age >= 18 ? "adult" : "minor"`)
- **Nil coalescing**: `??` (e.g., `name ?? "Unknown"`)

#### Array Functions

- `all(array, predicate)` - All elements satisfy predicate
- `any(array, predicate)` - Any element satisfies predicate
- `one(array, predicate)` - Exactly one element satisfies predicate
- `none(array, predicate)` - No elements satisfy predicate
- `map(array, predicate)` - Transform array elements
- `filter(array, predicate)` - Filter array elements
- `find(array, predicate)` - Find first matching element
- `findIndex(array, predicate)` - Find index of first match
- `count(array, predicate)` - Count matching elements
- `sum(array)` - Sum of numbers
- `mean(array)` - Average of numbers
- `first(array)` - First element
- `last(array)` - Last element
- `sort(array)` - Sort array
- `reverse(array)` - Reverse array
- `flatten(array)` - Flatten nested arrays
- `uniq(array)` - Remove duplicates

#### String Functions

- `upper(str)` - Convert to uppercase
- `lower(str)` - Convert to lowercase
- `trim(str)` - Remove whitespace
- `split(str, delimiter)` - Split into array
- `replace(str, old, new)` - Replace substring
- `indexOf(str, substring)` - Find substring index
- `hasPrefix(str, prefix)` - Check prefix
- `hasSuffix(str, suffix)` - Check suffix

#### Date Functions

- `now()` - Current date/time
- `date(str)` - Parse date string
- `duration(str)` - Parse duration (e.g., "1h", "30m")
- Date methods: `.Year()`, `.Month()`, `.Day()`, `.Hour()`, `.Weekday()`

#### Type Conversion

- `int(v)` - Convert to integer
- `float(v)` - Convert to float
- `string(v)` - Convert to string
- `toJSON(v)` - Convert to JSON string
- `fromJSON(v)` - Parse JSON string

#### Miscellaneous

- `len(v)` - Length of array, map, or string
- `type(v)` - Get type name
- `get(v, index)` - Safe index access

### Best Practices

1. **Use standard Sift operations when possible** - They're more portable across adapters
2. **Combine standard and custom** - Use custom expressions only for features not available in Sift
3. **Keep expressions readable** - Break complex logic into multiple filters when possible
4. **Test thoroughly** - Custom expressions bypass Sift's type safety
5. **Document custom expressions** - Add comments explaining complex logic

### Serialization

Custom expressions serialize using the `exprlang(...)` format in Sift's string representation. The expr-lang adapter automatically registers a custom formatter that handles serialization and parsing.

```go
// Standard Sift operation
filter := &sift.Condition{Name: "status", Operation: sift.OperationEQ, Value: "active"}
sift.Format(filter) // "eq(status,active)"

// Custom expression
filter := exprlang.RawExpression("len(tweets) > 10")
sift.Format(filter) // "exprlang(len\\(tweets\\) > 10)"

// Mixed
filter := &sift.AndOperation{
    Left: &sift.Condition{Name: "status", Operation: sift.OperationEQ, Value: "active"},
    Right: exprlang.RawExpression("len(tweets) > 10"),
}
sift.Format(filter) // "and(eq(status,active),exprlang(len\\(tweets\\) > 10))"
```

The formatter automatically:
- **Escapes** special Sift characters (backslashes, commas, parentheses) in expr-lang expressions
- **Parses** escaped expressions back to their original form
- **Supports round-trip** serialization (Format → Parse → Format produces the same result)

```go
// Round-trip example
original := exprlang.RawExpression("(a + b) * (c + d)")
formatted, _ := sift.Format(original)           // "exprlang(\\(a + b\\) * \\(c + d\\))"
parsed, _ := sift.Parse(formatted)              // Reconstructs the expression
reformatted, _ := sift.Format(parsed)           // Same as formatted
```

This allows custom expressions to be:
- Serialized to strings for storage or transmission
- Combined with standard Sift operations
- Parsed back from stored strings
- Logged and debugged easily

## Use Cases

### API Query Filters

```go
// Parse user-provided filter from query string
filterStr := r.URL.Query().Get("filter")
filter, _ := sift.Parse(filterStr)

// Translate to expr-lang
adapter := exprlang.NewAdapter()
sift.Thru(ctx, adapter, filter)

// Compile once, reuse for all records
program, _ := expr.Compile(adapter.Expression(), expr.Env(Record{}))

// Filter records
var results []Record
for _, record := range allRecords {
    match, _ := expr.Run(program, record)
    if match.(bool) {
        results = append(results, record)
    }
}
```

### In-Memory Filtering

```go
func FilterUsers(users []User, filter sift.Expression) []User {
    adapter := exprlang.NewAdapter()
    sift.Thru(context.Background(), adapter, filter)
    
    program, _ := expr.Compile(adapter.Expression(), expr.Env(User{}))
    
    var filtered []User
    for _, user := range users {
        match, _ := expr.Run(program, user)
        if match.(bool) {
            filtered = append(filtered, user)
        }
    }
    return filtered
}
```

### Dynamic Business Rules

```go
// Store filter expressions as business rules
rules := map[string]sift.Expression{
    "premium_users": &sift.AndOperation{
        Left: &sift.Condition{
            Name:      "Subscription",
            Operation: sift.OperationEQ,
            Value:     "premium",
        },
        Right: &sift.Condition{
            Name:      "Active",
            Operation: sift.OperationEQ,
            Value:     "true",
        },
    },
}

// Evaluate rules at runtime
adapter := exprlang.NewAdapter()
sift.Thru(ctx, adapter, rules["premium_users"])
program, _ := expr.Compile(adapter.Expression(), expr.Env(User{}))
isPremium, _ := expr.Run(program, currentUser)
```

## Performance

The adapter is designed for performance:

1. **Compile Once**: Compile the expr-lang program once and reuse it for multiple evaluations
2. **Zero Allocation**: The adapter builds the expression string with minimal allocations
3. **Type Safety**: Expr-lang provides compile-time type checking when using `expr.Env()`
4. **Use type-safe environments**: Pass struct types to `expr.Env()` for compile-time checking
5. **Avoid expensive operations in loops**: Be mindful of nested array operations
6. **Use predicates efficiently**: Leverage short-circuit evaluation with `and`/`or`

```go
// Good: Compile once, run many times
adapter := exprlang.NewAdapter()
sift.Thru(ctx, adapter, filter)
program, _ := expr.Compile(adapter.Expression(), expr.Env(User{}))

for _, user := range users {
    match, _ := expr.Run(program, user)
    // ...
}

// Bad: Compiling in the loop
for _, user := range users {
    adapter := exprlang.NewAdapter()
    sift.Thru(ctx, adapter, filter)
    program, _ := expr.Compile(adapter.Expression(), expr.Env(User{}))
    match, _ := expr.Run(program, user)
    // ...
}
```

## Limitations

1. **Between Operation**: Requires comma-separated values (e.g., `"18,65"`)
2. **In Operation**: The expr-lang `in` operator expects the field to be a collection (array, slice, map)
3. **Custom Expressions**: Only `exprlang.CustomExpression` types are supported; custom expressions from other adapters will return an error

## Examples

See [example_test.go](example_test.go) for comprehensive examples including:

- Basic filtering
- Complex logical expressions
- String operations (contains, startsWith)
- Collection membership (in operator)
- Between operations
- Existence checks
- Custom expr-lang expressions
- Mixing standard Sift operations with custom expressions
- Array predicates and functions
- Serialization of custom expressions
- Real-world use cases

## License

This package is part of the Sift project and follows the same license.

## Links

- [Sift Library](https://github.com/nisimpson/sift)
- [Expr-Lang Documentation](https://expr-lang.org/)
- [Expr-Lang Language Definition](https://expr-lang.org/docs/language-definition)
- [Expr-Lang Getting Started](https://expr-lang.org/docs/getting-started)
- [Expr-Lang GitHub](https://github.com/expr-lang/expr)
