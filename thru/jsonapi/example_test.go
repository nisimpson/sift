package jsonapi_test

import (
	"context"
	"fmt"
	"net/url"

	"github.com/nisimpson/sift"
	"github.com/nisimpson/sift/thru/jsonapi"
)

func Example_parse() {
	// Parse JSON:API query parameters to Sift expression
	query := "filter[q]=and(p1,p2)&filter[p1]=eq(status,active)&filter[p2]=gt(age,18)"
	values, _ := url.ParseQuery(query)

	adapter := jsonapi.NewAdapter()
	filter, _ := adapter.Parse(values)

	// Use the filter with any backend
	formatted, _ := sift.Format(filter)
	fmt.Println(formatted)

	// Output: and(eq(status,active),gt(age,18))
}

func Example_format() {
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

	// Format to JSON:API query parameters
	adapter := jsonapi.NewAdapter()
	values, _ := adapter.Format(filter)

	fmt.Println("Main query:", values.Get("filter[q]"))
	fmt.Println("Parameter p1:", values.Get("filter[p1]"))
	fmt.Println("Parameter p2:", values.Get("filter[p2]"))

	// Output:
	// Main query: and(p1,p2)
	// Parameter p1: eq(status,active)
	// Parameter p2: gt(age,18)
}

func Example_wrapper() {
	// Create a mock adapter (in real use, this would be dynamodb.NewAdapter(), etc.)
	mockAdapter := NewMockBackendAdapter()

	// Wrap it with JSON:API support
	wrapped := jsonapi.Wrap(mockAdapter)

	// Parse and evaluate JSON:API query in one step
	query := "filter[q]=and(p1,p2)&filter[p1]=eq(status,active)&filter[p2]=gt(age,18)"
	values, _ := url.ParseQuery(query)

	err := wrapped.ParseThru(context.Background(), values)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Evaluated successfully")
	fmt.Println("Call count:", mockAdapter.CallCount())

	// Output:
	// Evaluated successfully
	// Call count: 3
}

func Example_nestedOperations() {
	// Complex nested filter: status = "active" AND (age > 18 OR role = "admin")
	query := "filter[q]=and(p1,or(p2,p3))&filter[p1]=eq(status,active)&filter[p2]=gt(age,18)&filter[p3]=eq(role,admin)"
	values, _ := url.ParseQuery(query)

	adapter := jsonapi.NewAdapter()
	filter, _ := adapter.Parse(values)

	formatted, _ := sift.Format(filter)
	fmt.Println(formatted)

	// Output: and(eq(status,active),or(gt(age,18),eq(role,admin)))
}

func Example_roundTrip() {
	// Create a Sift expression
	original := &sift.AndOperation{
		Left: &sift.Condition{
			Name:      "status",
			Operation: sift.OperationEQ,
			Value:     "active",
		},
		Right: &sift.Condition{
			Name:      "verified",
			Operation: sift.OperationEQ,
			Value:     "true",
		},
	}

	adapter := jsonapi.NewAdapter()

	// Format to JSON:API
	values, _ := adapter.Format(original)
	queryString := values.Encode()
	fmt.Println("Query string:", queryString)

	// Parse back to Sift
	parsed, _ := adapter.Parse(values)

	// Verify round-trip
	originalStr, _ := sift.Format(original)
	parsedStr, _ := sift.Format(parsed)
	fmt.Println("Round-trip successful:", originalStr == parsedStr)

	// Output:
	// Query string: filter%5Bp1%5D=eq%28status%2Cactive%29&filter%5Bp2%5D=eq%28verified%2Ctrue%29&filter%5Bq%5D=and%28p1%2Cp2%29
	// Round-trip successful: true
}


func Example_customExpressions() {
	// Create a filter with custom expressions
	filter := &sift.AndOperation{
		Left: &sift.Condition{
			Name:      "status",
			Operation: sift.OperationEQ,
			Value:     "active",
		},
		Right: NewMockCustomExpression("tags", "5"),
	}

	// Format to JSON:API
	adapter := jsonapi.NewAdapter()
	values, _ := adapter.Format(filter)

	fmt.Println("Main query:", values.Get("filter[q]"))
	fmt.Println("Parameter p1:", values.Get("filter[p1]"))
	fmt.Println("Parameter p2:", values.Get("filter[p2]"))

	// Output:
	// Main query: and(p1,p2)
	// Parameter p1: eq(status,active)
	// Parameter p2: mock_size(tags,5)
}

func Example_backendWithCustomExpressions() {
	// Simulate receiving JSON:API query with custom expressions
	query := "filter[q]=and(p1,p2)&filter[p1]=eq(status,active)&filter[p2]=mock_size(tags,5)"
	values, _ := url.ParseQuery(query)

	// Create backend adapter that supports custom expressions
	backendAdapter := NewMockBackendAdapter()

	// Wrap with JSON:API support
	wrapped := jsonapi.Wrap(backendAdapter)

	// Parse and evaluate
	err := wrapped.ParseThru(context.Background(), values)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Get the backend expression
	fmt.Println("Backend expression:", backendAdapter.Expression())

	// Output:
	// Backend expression: (status = active) AND (size(mock_size(tags,5)))
}
