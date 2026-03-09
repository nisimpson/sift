package dynamodb_test

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/nisimpson/sift"
	siftddb "github.com/nisimpson/sift/thru/dynamodb"
)

func Example() {
	// Create a filter: status = "active" AND age > 18
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
	err := sift.Thru(context.Background(), adapter, sift.WithFilter(filter))
	if err != nil {
		log.Fatal(err)
	}

	// Build the DynamoDB expression
	expr, err := adapter.Expression()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Expression: %s\n", *expr.Condition())
	fmt.Printf("Has 2 names: %v\n", len(expr.Names()) == 2)
	fmt.Printf("Has 2 values: %v\n", len(expr.Values()) == 2)

	// Output:
	// Expression: (#0 = :0) AND (#1 > :1)
	// Has 2 names: true
	// Has 2 values: true
}

func ExampleAdapter_complexFilter() {
	// Complex filter: (role = "admin" OR role = "moderator") AND verified = true
	filter := &sift.AndOperation{
		Left: &sift.OrOperation{
			Left: &sift.Condition{
				Name:      "role",
				Operation: sift.OperationEQ,
				Value:     "admin",
			},
			Right: &sift.Condition{
				Name:      "role",
				Operation: sift.OperationEQ,
				Value:     "moderator",
			},
		},
		Right: &sift.Condition{
			Name:      "verified",
			Operation: sift.OperationEQ,
			Value:     "true",
		},
	}

	adapter := siftddb.NewAdapter()
	sift.Thru(context.Background(), adapter, sift.WithFilter(filter))

	expr, _ := adapter.Expression()
	fmt.Printf("Expression: %s\n", *expr.Condition())
	fmt.Printf("Has 2 attribute names: %v\n", len(expr.Names()) == 2)
	fmt.Printf("Has 3 values: %v\n", len(expr.Values()) == 3)

	// Output:
	// Expression: ((#0 = :0) OR (#0 = :1)) AND (#1 = :2)
	// Has 2 attribute names: true
	// Has 3 values: true
}

func ExampleAdapter_stringOperations() {
	// String operations: email contains "@example.com" AND name begins_with "John"
	filter := &sift.AndOperation{
		Left: &sift.Condition{
			Name:      "email",
			Operation: sift.OperationContains,
			Value:     "@example.com",
		},
		Right: &sift.Condition{
			Name:      "name",
			Operation: sift.OperationBeginsWith,
			Value:     "John",
		},
	}

	adapter := siftddb.NewAdapter()
	sift.Thru(context.Background(), adapter, sift.WithFilter(filter))

	expr, _ := adapter.Expression()
	fmt.Printf("Expression: %s\n", *expr.Condition())
	fmt.Printf("Has 2 names: %v\n", len(expr.Names()) == 2)
	fmt.Printf("Has 2 values: %v\n", len(expr.Values()) == 2)

	// Output:
	// Expression: (contains (#0, :0)) AND (begins_with (#1, :1))
	// Has 2 names: true
	// Has 2 values: true
}

func ExampleAdapter_existenceChecks() {
	// Existence checks: email exists AND deleted_at does not exist
	filter := &sift.AndOperation{
		Left: &sift.Condition{
			Name:      "email",
			Operation: sift.OperationExists,
		},
		Right: &sift.Condition{
			Name:      "deleted_at",
			Operation: sift.OperationNotExists,
		},
	}

	adapter := siftddb.NewAdapter()
	sift.Thru(context.Background(), adapter, sift.WithFilter(filter))

	expr, _ := adapter.Expression()
	fmt.Printf("Expression: %s\n", *expr.Condition())
	fmt.Printf("Has 2 names: %v\n", len(expr.Names()) == 2)
	fmt.Printf("Has 0 values: %v\n", len(expr.Values()) == 0)

	// Output:
	// Expression: (attribute_exists (#0)) AND (attribute_not_exists (#1))
	// Has 2 names: true
	// Has 0 values: true
}

func ExampleAdapter_withQuery() {
	// Example showing usage with DynamoDB Query
	filter := &sift.Condition{
		Name:      "status",
		Operation: sift.OperationEQ,
		Value:     "active",
	}

	adapter := siftddb.NewAdapter()
	sift.Thru(context.Background(), adapter, sift.WithFilter(filter))

	expr, _ := adapter.Expression()

	// Use with Query (pseudo-code, requires actual DynamoDB client)
	input := &dynamodb.QueryInput{
		TableName:                 aws.String("Users"),
		KeyConditionExpression:    aws.String("pk = :pk"),
		FilterExpression:          expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}

	fmt.Printf("Table: %s\n", *input.TableName)
	fmt.Printf("Has filter expression: %v\n", input.FilterExpression != nil)
	fmt.Printf("Has names: %v\n", len(input.ExpressionAttributeNames) > 0)
	fmt.Printf("Has values: %v\n", len(input.ExpressionAttributeValues) > 0)

	// Output:
	// Table: Users
	// Has filter expression: true
	// Has names: true
	// Has values: true
}

func ExampleAdapter_numericTypes() {
	// Numeric types are automatically parsed from strings
	filter := &sift.AndOperation{
		Left: &sift.Condition{
			Name:      "age",
			Operation: sift.OperationGT,
			Value:     "18", // Parsed as int64
		},
		Right: &sift.Condition{
			Name:      "price",
			Operation: sift.OperationLTE,
			Value:     "99.99", // Parsed as float64
		},
	}

	adapter := siftddb.NewAdapter()
	sift.Thru(context.Background(), adapter, sift.WithFilter(filter))

	expr, _ := adapter.Expression()
	fmt.Printf("Expression: %s\n", *expr.Condition())
	fmt.Printf("Has 2 names: %v\n", len(expr.Names()) == 2)
	fmt.Printf("Has 2 values: %v\n", len(expr.Values()) == 2)

	// Output:
	// Expression: (#0 > :0) AND (#1 <= :1)
	// Has 2 names: true
	// Has 2 values: true
}

func ExampleAdapter_booleanType() {
	// Boolean values are automatically parsed from strings
	filter := &sift.Condition{
		Name:      "verified",
		Operation: sift.OperationEQ,
		Value:     "true", // Parsed as bool
	}

	adapter := siftddb.NewAdapter()
	sift.Thru(context.Background(), adapter, sift.WithFilter(filter))

	expr, _ := adapter.Expression()
	fmt.Printf("Expression: %s\n", *expr.Condition())
	fmt.Printf("Has 1 name: %v\n", len(expr.Names()) == 1)
	fmt.Printf("Has 1 value: %v\n", len(expr.Values()) == 1)

	// Output:
	// Expression: #0 = :0
	// Has 1 name: true
	// Has 1 value: true
}

func ExampleNewAdapterWithConfig() {
	// Configure specific attribute types to avoid ambiguity
	config := &siftddb.Config{
		AttributeTypes: map[string]siftddb.AttributeType{
			"age":      siftddb.AttributeTypeNumber,
			"verified": siftddb.AttributeTypeBool,
			"id":       siftddb.AttributeTypeString, // Keep as string even if numeric
		},
	}

	filter := &sift.AndOperation{
		Left: &sift.Condition{
			Name:      "age",
			Operation: sift.OperationGT,
			Value:     "18", // Parsed as number
		},
		Right: &sift.Condition{
			Name:      "id",
			Operation: sift.OperationEQ,
			Value:     "12345", // Kept as string (not parsed as number)
		},
	}

	adapter := siftddb.NewAdapterWithConfig(config)
	sift.Thru(context.Background(), adapter, sift.WithFilter(filter))

	expr, _ := adapter.Expression()
	fmt.Printf("Expression: %s\n", *expr.Condition())
	fmt.Printf("Has 2 names: %v\n", len(expr.Names()) == 2)
	fmt.Printf("Has 2 values: %v\n", len(expr.Values()) == 2)

	// Output:
	// Expression: (#0 > :0) AND (#1 = :1)
	// Has 2 names: true
	// Has 2 values: true
}

func ExampleAdapter_customExpressions_size() {
	// Check if tags list has more than 3 elements
	filter := siftddb.Size("tags", sift.OperationGT, 3)

	adapter := siftddb.NewAdapter()
	sift.Thru(context.Background(), adapter, sift.WithFilter(filter))

	expr, _ := adapter.Expression()
	fmt.Printf("Expression: %s\n", *expr.Condition())
	fmt.Printf("Has 1 name: %v\n", len(expr.Names()) == 1)
	fmt.Printf("Has 1 value: %v\n", len(expr.Values()) == 1)

	// Output:
	// Expression: size (#0) > :0
	// Has 1 name: true
	// Has 1 value: true
}

func ExampleAdapter_customExpressions_attributeType() {
	// Check if metadata is a Map type
	filter := siftddb.IsAttributeType("metadata", "M")

	adapter := siftddb.NewAdapter()
	sift.Thru(context.Background(), adapter, sift.WithFilter(filter))

	expr, _ := adapter.Expression()
	fmt.Printf("Expression: %s\n", *expr.Condition())
	fmt.Printf("Has 1 name: %v\n", len(expr.Names()) == 1)
	fmt.Printf("Has 1 value: %v\n", len(expr.Values()) == 1)

	// Output:
	// Expression: attribute_type (#0, :0)
	// Has 1 name: true
	// Has 1 value: true
}

func ExampleAdapter_customExpressions_mixed() {
	// Combine custom and standard expressions:
	// status = "active" AND size(tags) > 3
	filter := sift.Eq("status", "active").And(siftddb.Size("tags", sift.OperationGT, 3))

	adapter := siftddb.NewAdapter()
	sift.Thru(context.Background(), adapter, sift.WithFilter(filter))

	expr, _ := adapter.Expression()
	fmt.Printf("Expression: %s\n", *expr.Condition())
	fmt.Printf("Has 2 names: %v\n", len(expr.Names()) == 2)
	fmt.Printf("Has 2 values: %v\n", len(expr.Values()) == 2)

	// Output:
	// Expression: (#0 = :0) AND (size (#1) > :1)
	// Has 2 names: true
	// Has 2 values: true
}

func ExampleAdapter_customExpressions_serialization() {
	// Custom expressions can be serialized and parsed
	registry := siftddb.NewRegistry()
	filter := siftddb.Size("tags", sift.OperationGT, 5)

	// Serialize to string
	str, _ := sift.FormatFilter(filter, registry)
	fmt.Printf("Serialized: %s\n", str)

	// Parse back (wrap with filter() for ParseQuery)
	query, _ := sift.ParseQuery("filter("+str+")", registry)

	// Use with adapter
	adapter := siftddb.NewAdapter()
	sift.Thru(context.Background(), adapter, sift.WithFilter(query.Filter))

	expr, _ := adapter.Expression()
	fmt.Printf("Expression: %s\n", *expr.Condition())

	// Output:
	// Serialized: size(tags,gt,5)
	// Expression: size (#0) > :0
}

func ExampleAdapter_sorting() {
	// Sort by created_at in descending order (newest first)
	sort := sift.Sort("created_at", sift.SortDesc)

	adapter := siftddb.NewAdapter()
	sift.Thru(context.Background(), adapter, sift.WithSort(sort))

	forward := adapter.ScanIndexForward()
	fmt.Printf("ScanIndexForward: %v\n", *forward)
	fmt.Printf("Direction: descending\n")

	// Output:
	// ScanIndexForward: false
	// Direction: descending
}

func ExampleAdapter_sortingAscending() {
	// Sort by created_at in ascending order (oldest first)
	sort := sift.Sort("created_at", sift.SortAsc)

	adapter := siftddb.NewAdapter()
	sift.Thru(context.Background(), adapter, sift.WithSort(sort))

	forward := adapter.ScanIndexForward()
	fmt.Printf("ScanIndexForward: %v\n", *forward)
	fmt.Printf("Direction: ascending\n")

	// Output:
	// ScanIndexForward: true
	// Direction: ascending
}

func ExampleAdapter_filterAndSort() {
	// Combine filtering and sorting
	filter := sift.Eq("status", "active")
	sort := sift.Sort("created_at", sift.SortDesc)

	adapter := siftddb.NewAdapter()
	sift.Thru(context.Background(), adapter, sift.WithFilter(filter))
	sift.Thru(context.Background(), adapter, sift.WithSort(sort))

	expr, _ := adapter.Expression()
	forward := adapter.ScanIndexForward()

	// Use with Query
	input := &dynamodb.QueryInput{
		TableName:                 aws.String("Users"),
		KeyConditionExpression:    aws.String("pk = :pk"),
		FilterExpression:          expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ScanIndexForward:          forward,
	}

	fmt.Printf("Table: %s\n", *input.TableName)
	fmt.Printf("Has filter: %v\n", input.FilterExpression != nil)
	fmt.Printf("ScanIndexForward: %v\n", *input.ScanIndexForward)

	// Output:
	// Table: Users
	// Has filter: true
	// ScanIndexForward: false
}

func ExampleAdapter_sortingLimitation() {
	// DynamoDB only supports sorting by the sort key
	// Multiple sort fields are accepted but only the first is used
	sort := sift.Sort("created_at", sift.SortDesc).
		ThenBy("name", sift.SortAsc)

	adapter := siftddb.NewAdapter()
	sift.Thru(context.Background(), adapter, sift.WithSort(sort))

	forward := adapter.ScanIndexForward()
	fmt.Printf("ScanIndexForward: %v\n", *forward)
	fmt.Printf("Note: Only first field (created_at) affects sort direction\n")

	// Output:
	// ScanIndexForward: false
	// Note: Only first field (created_at) affects sort direction
}
