package jsonapi_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/nisimpson/sift"
	"github.com/nisimpson/sift/thru/jsonapi"
)

func TestIntegration_JSONAPIToBackend(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		wantExpression string
	}{
		{
			name:           "simple condition",
			query:          "filter[q]=p1&filter[p1]=eq(status,active)",
			wantExpression: "status = active",
		},
		{
			name:           "and operation",
			query:          "filter[q]=and(p1,p2)&filter[p1]=eq(status,active)&filter[p2]=gt(age,18)",
			wantExpression: "(status = active) AND (age > 18)",
		},
		{
			name:           "or operation",
			query:          "filter[q]=or(p1,p2)&filter[p1]=eq(role,admin)&filter[p2]=eq(role,moderator)",
			wantExpression: "(role = admin) OR (role = moderator)",
		},
		{
			name:           "nested operations",
			query:          "filter[q]=and(p1,or(p2,p3))&filter[p1]=eq(status,active)&filter[p2]=gt(age,18)&filter[p3]=eq(role,admin)",
			wantExpression: "(status = active) AND ((age > 18) OR (role = admin))",
		},
		{
			name:           "not operation",
			query:          "filter[q]=not(p1)&filter[p1]=eq(deleted,true)",
			wantExpression: "NOT (deleted = true)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, err := url.ParseQuery(tt.query)
			if err != nil {
				t.Fatalf("Failed to parse query: %v", err)
			}

			// Create backend adapter
			backendAdapter := NewMockBackendAdapter()

			// Wrap with JSON:API
			wrapped := jsonapi.Wrap(backendAdapter)

			// Parse and evaluate
			err = wrapped.ParseThru(context.Background(), values)
			if err != nil {
				t.Fatalf("ParseThru() error = %v", err)
			}

			// Check the backend expression
			if backendAdapter.Expression() != tt.wantExpression {
				t.Errorf("Expression = %v, want %v", backendAdapter.Expression(), tt.wantExpression)
			}
		})
	}
}

func TestIntegration_BackendToJSONAPI(t *testing.T) {
	tests := []struct {
		name       string
		expr       sift.Expression
		wantQuery  string
		wantParams map[string]string
	}{
		{
			name: "simple condition",
			expr: &sift.Condition{
				Name:      "status",
				Operation: sift.OperationEQ,
				Value:     "active",
			},
			wantQuery: "p1",
			wantParams: map[string]string{
				"p1": "eq(status,active)",
			},
		},
		{
			name: "and operation",
			expr: &sift.AndOperation{
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
			},
			wantQuery: "and(p1,p2)",
			wantParams: map[string]string{
				"p1": "eq(status,active)",
				"p2": "gt(age,18)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := jsonapi.NewAdapter()
			values, err := adapter.Format(tt.expr)
			if err != nil {
				t.Fatalf("Format() error = %v", err)
			}

			// Check main query
			gotQuery := values.Get("filter[q]")
			if gotQuery != tt.wantQuery {
				t.Errorf("Query = %v, want %v", gotQuery, tt.wantQuery)
			}

			// Check parameters
			for name, wantValue := range tt.wantParams {
				gotValue := values.Get("filter[" + name + "]")
				if gotValue != wantValue {
					t.Errorf("Param %s = %v, want %v", name, gotValue, wantValue)
				}
			}
		})
	}
}

func TestIntegration_FullRoundTrip(t *testing.T) {
	// Start with a Sift expression
	original := &sift.AndOperation{
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
	jsonapiAdapter := jsonapi.NewAdapter()
	values, err := jsonapiAdapter.Format(original)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Parse back to Sift
	parsed, err := jsonapiAdapter.Parse(values)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Evaluate with backend adapter
	backendAdapter := NewMockBackendAdapter()
	err = sift.Thru(context.Background(), backendAdapter, parsed)
	if err != nil {
		t.Fatalf("Thru() error = %v", err)
	}

	// Check the backend expression
	wantExpression := "(status = active) AND (age > 18)"
	if backendAdapter.Expression() != wantExpression {
		t.Errorf("Expression = %v, want %v", backendAdapter.Expression(), wantExpression)
	}
}

func TestIntegration_CustomExpressions(t *testing.T) {
	// Test with custom expressions
	filter := &sift.AndOperation{
		Left: &sift.Condition{
			Name:      "status",
			Operation: sift.OperationEQ,
			Value:     "active",
		},
		Right: NewMockCustomExpression("tags", "5"),
	}

	// Format to JSON:API
	jsonapiAdapter := jsonapi.NewAdapter()
	values, err := jsonapiAdapter.Format(filter)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Check that custom expression was formatted
	query := values.Get("filter[q]")
	if query != "and(p1,p2)" {
		t.Errorf("Query = %v, want and(p1,p2)", query)
	}

	// Check parameters include custom expression
	p2 := values.Get("filter[p2]")
	if p2 != "mock_size(tags,5)" {
		t.Errorf("Parameter p2 = %v, want mock_size(tags,5)", p2)
	}

	// Parse back and evaluate
	parsed, err := jsonapiAdapter.Parse(values)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	backendAdapter := NewMockBackendAdapter()
	err = sift.Thru(context.Background(), backendAdapter, parsed)
	if err != nil {
		t.Fatalf("Thru() error = %v", err)
	}

	// Check the backend expression includes custom expression
	wantExpression := "(status = active) AND (size(mock_size(tags,5)))"
	if backendAdapter.Expression() != wantExpression {
		t.Errorf("Expression = %v, want %v", backendAdapter.Expression(), wantExpression)
	}
}
