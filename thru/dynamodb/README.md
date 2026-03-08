# Sift DynamoDB Adapter

A DynamoDB adapter for the [sift](https://github.com/nisimpson/sift) universal filter library.

## Installation

```bash
go get github.com/nisimpson/sift/thru/dynamodb
```

## Usage

```go
import (
    "context"
    
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/nisimpson/sift"
    siftddb "github.com/nisimpson/sift/thru/dynamodb"
)

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

// Create adapter and evaluate
adapter := siftddb.NewAdapter()
err := sift.Thru(context.Background(), adapter, filter)
if err != nil {
    log.Fatal(err)
}

// Build the DynamoDB expression
expr, err := adapter.Expression()
if err != nil {
    log.Fatal(err)
}

// Use with DynamoDB Scan
result, err := client.Scan(ctx, &dynamodb.ScanInput{
    TableName:                 aws.String("Users"),
    FilterExpression:          expr.Condition(),
    ExpressionAttributeNames:  expr.Names(),
    ExpressionAttributeValues: expr.Values(),
})
```

## Supported Operations

All sift operations are supported:

- `OperationEQ` → `name = value`
- `OperationNEQ` → `name <> value`
- `OperationLT` → `name < value`
- `OperationLTE` → `name <= value`
- `OperationGT` → `name > value`
- `OperationGTE` → `name >= value`
- `OperationContains` → `contains(name, value)`
- `OperationBeginsWith` → `begins_with(name, value)`
- `OperationIn` → `name IN (value)`
- `OperationExists` → `attribute_exists(name)`
- `OperationNotExists` → `attribute_not_exists(name)`
- `OperationBetween` → `name BETWEEN value1 AND value2`

Logical operations (AND, OR, NOT) are fully supported.

## Sorting

The DynamoDB adapter supports sorting through the `ScanIndexForward` parameter. This controls the sort direction when querying a table or index with a sort key.

### Basic Sorting

```go
// Sort by created_at in descending order (newest first)
sort := sift.Sort("created_at", sift.SortDesc)

adapter := siftddb.NewAdapter()
sift.SortThru(context.Background(), adapter, sort)

// Use with Query
result, err := client.Query(ctx, &dynamodb.QueryInput{
    TableName:              aws.String("Users"),
    KeyConditionExpression: aws.String("pk = :pk"),
    ScanIndexForward:       adapter.ScanIndexForward(), // false for descending
})
```

### Combining Filtering and Sorting

```go
// Filter and sort together
filter := sift.Eq("status", "active")
sort := sift.Sort("created_at", sift.SortDesc)

adapter := siftddb.NewAdapter()
sift.Thru(ctx, adapter, filter)
sift.SortThru(ctx, adapter, sort)

expr, _ := adapter.Expression()

result, err := client.Query(ctx, &dynamodb.QueryInput{
    TableName:                 aws.String("Users"),
    KeyConditionExpression:    aws.String("pk = :pk"),
    FilterExpression:          expr.Condition(),
    ExpressionAttributeNames:  expr.Names(),
    ExpressionAttributeValues: expr.Values(),
    ScanIndexForward:          adapter.ScanIndexForward(),
})
```

### DynamoDB Sorting Limitations

DynamoDB has specific limitations for sorting:

1. **Sort Key Only**: DynamoDB can only sort by the sort key of the table or index being queried
2. **Single Field**: Only the first sort field in a `SortList` is used
3. **Query Operations**: Sorting only applies to Query operations, not Scan operations

The adapter handles these limitations by:
- Setting `ScanIndexForward` based on the first sort field's direction
- Ignoring additional sort fields (DynamoDB doesn't support multi-field sorting)
- Returning `nil` if no sort was specified

```go
// Multiple sort fields - only first is used
sort := sift.Sort("created_at", sift.SortDesc).
    ThenBy("name", sift.SortAsc)

adapter := siftddb.NewAdapter()
sift.SortThru(ctx, adapter, sort)

// ScanIndexForward will be false (from created_at DESC)
// The name field is ignored
forward := adapter.ScanIndexForward() // false
```

### Sort Directions

- `sift.SortAsc` → `ScanIndexForward = true` (ascending order)
- `sift.SortDesc` → `ScanIndexForward = false` (descending order)

## Type Handling

The adapter automatically parses string values into appropriate DynamoDB types:

- **Integers**: `"18"` → `int64(18)`
- **Floats**: `"99.99"` → `float64(99.99)`
- **Booleans**: `"true"` → `bool(true)`
- **Strings**: Any value that doesn't parse as above remains a string

This ensures proper type matching with DynamoDB attributes. For example:

```go
filter := &sift.Condition{
    Name:      "age",
    Operation: sift.OperationGT,
    Value:     "18", // Automatically parsed as int64
}
```

This will correctly match against a DynamoDB Number attribute, avoiding type mismatch issues.

### Configuring Attribute Types

For more control over type parsing, you can configure specific attribute types:

```go
config := &dynamodb.Config{
    AttributeTypes: map[string]dynamodb.AttributeType{
        "age":      dynamodb.AttributeTypeNumber,  // Always parse as number
        "verified": dynamodb.AttributeTypeBool,    // Always parse as bool
        "id":       dynamodb.AttributeTypeString,  // Keep as string (don't parse)
        "score":    dynamodb.AttributeTypeAuto,    // Auto-detect (default)
    },
}

adapter := dynamodb.NewAdapterWithConfig(config)
```

This is useful when:
- You have numeric IDs that should remain strings (e.g., `"00123"`)
- You want to avoid ambiguity in type detection
- You need consistent type handling across your application

Attributes not in the configuration map will use auto-detection.

## Custom Expressions

The DynamoDB adapter supports DynamoDB-specific operations through custom expressions:

### Size Function

Check the size of an attribute (string length, list/set size, etc.):

```go
import siftddb "github.com/nisimpson/sift/thru/dynamodb"

// Check if tags list has more than 5 elements
filter := siftddb.Size("tags", sift.OperationGT, 5)

// Check if name is exactly 10 characters
filter := siftddb.Size("name", sift.OperationEQ, 10)

// Combine with standard operations
filter := sift.Eq("status", "active").And(siftddb.Size("tags", sift.OperationGT, 3))
```

Supported operations: `EQ`, `NEQ`, `LT`, `LTE`, `GT`, `GTE`

### Attribute Type Function

Check if an attribute is of a specific DynamoDB type:

```go
// Check if metadata is a Map
filter := siftddb.IsAttributeType("metadata", "M")

// Check if tags is a List
filter := siftddb.IsAttributeType("tags", "L")

// Check if name is a String
filter := siftddb.IsAttributeType("name", "S")
```

Valid types:
- `"S"` - String
- `"N"` - Number
- `"B"` - Binary
- `"SS"` - String Set
- `"NS"` - Number Set
- `"BS"` - Binary Set
- `"L"` - List
- `"M"` - Map
- `"NULL"` - Null
- `"BOOL"` - Boolean

### Serialization

Custom expressions serialize to Sift format and can be parsed back:

```go
// Create expression
filter := siftddb.Size("tags", sift.OperationGT, 5)

// Serialize
str, _ := sift.Format(filter)
// "dynamodb_size(tags,gt,5)"

// Parse back
parsed, _ := sift.Parse(str)

// Use with adapter
adapter := siftddb.NewAdapter()
sift.Thru(ctx, adapter, parsed)
```

## Advanced Usage

### Combining with Other Expression Components

You can access the condition builder directly to combine with other expression components:

```go
import "github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"

adapter := siftddb.NewAdapter()
sift.Thru(ctx, adapter, filter)

// Get the condition builder
condition := adapter.Condition()

// Combine with projection
builder := expression.NewBuilder().
    WithCondition(condition).
    WithProjection(expression.NamesList(
        expression.Name("id"),
        expression.Name("name"),
        expression.Name("email"),
    ))

expr, _ := builder.Build()

// Use with Query or Scan
result, err := client.Scan(ctx, &dynamodb.ScanInput{
    TableName:                 aws.String("Users"),
    FilterExpression:          expr.Condition(),
    ProjectionExpression:      expr.Projection(),
    ExpressionAttributeNames:  expr.Names(),
    ExpressionAttributeValues: expr.Values(),
})
```

## Example

See [example_test.go](example_test.go) for a complete working example.

## License

MIT
