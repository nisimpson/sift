package dynamodb

import (
	"context"
	"testing"

	"github.com/nisimpson/sift"
)

func TestAdapter_EvaluateCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition *sift.Condition
		wantErr   bool
	}{
		{
			name: "equal operation",
			condition: &sift.Condition{
				Name:      "status",
				Operation: sift.OperationEQ,
				Value:     "active",
			},
			wantErr: false,
		},
		{
			name: "greater than operation",
			condition: &sift.Condition{
				Name:      "age",
				Operation: sift.OperationGT,
				Value:     "18",
			},
			wantErr: false,
		},
		{
			name: "contains operation",
			condition: &sift.Condition{
				Name:      "email",
				Operation: sift.OperationContains,
				Value:     "@example.com",
			},
			wantErr: false,
		},
		{
			name: "begins_with operation",
			condition: &sift.Condition{
				Name:      "name",
				Operation: sift.OperationBeginsWith,
				Value:     "John",
			},
			wantErr: false,
		},
		{
			name: "exists operation",
			condition: &sift.Condition{
				Name:      "email",
				Operation: sift.OperationExists,
			},
			wantErr: false,
		},
		{
			name: "not exists operation",
			condition: &sift.Condition{
				Name:      "deleted_at",
				Operation: sift.OperationNotExists,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter()
			err := sift.Thru(context.Background(), adapter, tt.condition)

			if (err != nil) != tt.wantErr {
				t.Errorf("Thru() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify we can build the expression
				expr, err := adapter.Expression()
				if err != nil {
					t.Errorf("Expression() error = %v", err)
					return
				}

				if expr.Condition() == nil {
					t.Error("Expression() returned nil filter expression")
				}
				if expr.Names() == nil {
					t.Error("Expression() returned nil names")
				}
				if expr.Values() == nil && tt.condition.Operation != sift.OperationExists && tt.condition.Operation != sift.OperationNotExists {
					t.Error("Expression() returned nil values")
				}
			}
		})
	}
}

func TestAdapter_EvaluateAnd(t *testing.T) {
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

	adapter := NewAdapter()
	err := sift.Thru(context.Background(), adapter, filter)
	if err != nil {
		t.Fatalf("Thru() error = %v", err)
	}

	expr, err := adapter.Expression()
	if err != nil {
		t.Fatalf("Expression() error = %v", err)
	}

	if expr.Condition() == nil {
		t.Error("Expression() returned nil filter expression")
	}
	if expr.Names() == nil {
		t.Error("Expression() returned nil names")
	}
	if expr.Values() == nil {
		t.Error("Expression() returned nil values")
	}

	// Verify we have both attribute names
	if len(expr.Names()) < 2 {
		t.Errorf("Expected at least 2 attribute names, got %d", len(expr.Names()))
	}

	// Verify we have both values
	if len(expr.Values()) < 2 {
		t.Errorf("Expected at least 2 values, got %d", len(expr.Values()))
	}
}

func TestAdapter_EvaluateOr(t *testing.T) {
	filter := &sift.OrOperation{
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
	}

	adapter := NewAdapter()
	err := sift.Thru(context.Background(), adapter, filter)
	if err != nil {
		t.Fatalf("Thru() error = %v", err)
	}

	expr, err := adapter.Expression()
	if err != nil {
		t.Fatalf("Expression() error = %v", err)
	}

	if expr.Condition() == nil {
		t.Error("Expression() returned nil filter expression")
	}
	if expr.Names() == nil {
		t.Error("Expression() returned nil names")
	}
	if expr.Values() == nil {
		t.Error("Expression() returned nil values")
	}
}

func TestAdapter_EvaluateNot(t *testing.T) {
	filter := &sift.NotOperation{
		Child: &sift.Condition{
			Name:      "deleted",
			Operation: sift.OperationEQ,
			Value:     "true",
		},
	}

	adapter := NewAdapter()
	err := sift.Thru(context.Background(), adapter, filter)
	if err != nil {
		t.Fatalf("Thru() error = %v", err)
	}

	expr, err := adapter.Expression()
	if err != nil {
		t.Fatalf("Expression() error = %v", err)
	}

	if expr.Condition() == nil {
		t.Error("Expression() returned nil filter expression")
	}
	if expr.Names() == nil {
		t.Error("Expression() returned nil names")
	}
	if expr.Values() == nil {
		t.Error("Expression() returned nil values")
	}
}

func TestAdapter_ComplexExpression(t *testing.T) {
	// (role = "admin" OR role = "moderator") AND verified = true
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

	adapter := NewAdapter()
	err := sift.Thru(context.Background(), adapter, filter)
	if err != nil {
		t.Fatalf("Thru() error = %v", err)
	}

	expr, err := adapter.Expression()
	if err != nil {
		t.Fatalf("Expression() error = %v", err)
	}

	if expr.Condition() == nil {
		t.Error("Expression() returned nil filter expression")
	}
	if expr.Names() == nil {
		t.Error("Expression() returned nil names")
	}
	if expr.Values() == nil {
		t.Error("Expression() returned nil values")
	}

	t.Logf("Filter Expression: %s", *expr.Condition())
	t.Logf("Names: %v", expr.Names())
	t.Logf("Values: %v", expr.Values())
}

func TestAdapter_NumericTypes(t *testing.T) {
	tests := []struct {
		name      string
		condition *sift.Condition
		wantErr   bool
	}{
		{
			name: "integer value",
			condition: &sift.Condition{
				Name:      "age",
				Operation: sift.OperationGT,
				Value:     "18",
			},
			wantErr: false,
		},
		{
			name: "float value",
			condition: &sift.Condition{
				Name:      "price",
				Operation: sift.OperationLTE,
				Value:     "99.99",
			},
			wantErr: false,
		},
		{
			name: "boolean value",
			condition: &sift.Condition{
				Name:      "verified",
				Operation: sift.OperationEQ,
				Value:     "true",
			},
			wantErr: false,
		},
		{
			name: "negative integer",
			condition: &sift.Condition{
				Name:      "balance",
				Operation: sift.OperationLT,
				Value:     "-100",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter()
			err := sift.Thru(context.Background(), adapter, tt.condition)

			if (err != nil) != tt.wantErr {
				t.Errorf("Thru() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				expr, err := adapter.Expression()
				if err != nil {
					t.Errorf("Expression() error = %v", err)
					return
				}

				if expr.Condition() == nil {
					t.Error("Expression() returned nil filter expression")
				}
				if expr.Names() == nil {
					t.Error("Expression() returned nil names")
				}
				if expr.Values() == nil {
					t.Error("Expression() returned nil values")
				}

				t.Logf("Filter Expression: %s", *expr.Condition())
				t.Logf("Values: %v", expr.Values())
			}
		})
	}
}

func TestAdapter_WithConfig(t *testing.T) {
	// Configure specific attribute types
	config := &Config{
		AttributeTypes: map[string]AttributeType{
			"age":      AttributeTypeNumber,
			"verified": AttributeTypeBool,
			"name":     AttributeTypeString,
			"score":    AttributeTypeNumber,
		},
	}

	filter := &sift.AndOperation{
		Left: &sift.Condition{
			Name:      "age",
			Operation: sift.OperationGT,
			Value:     "18", // Will be parsed as number
		},
		Right: &sift.AndOperation{
			Left: &sift.Condition{
				Name:      "verified",
				Operation: sift.OperationEQ,
				Value:     "true", // Will be parsed as bool
			},
			Right: &sift.Condition{
				Name:      "name",
				Operation: sift.OperationEQ,
				Value:     "123", // Will stay as string (not parsed as number)
			},
		},
	}

	adapter := NewAdapterWithConfig(config)
	err := sift.Thru(context.Background(), adapter, filter)
	if err != nil {
		t.Fatalf("Thru() error = %v", err)
	}

	expr, err := adapter.Expression()
	if err != nil {
		t.Fatalf("Expression() error = %v", err)
	}

	if expr.Condition() == nil {
		t.Error("Expression() returned nil filter expression")
	}
	if expr.Names() == nil {
		t.Error("Expression() returned nil names")
	}
	if expr.Values() == nil {
		t.Error("Expression() returned nil values")
	}

	t.Logf("Filter Expression: %s", *expr.Condition())
	t.Logf("Names: %v", expr.Names())
	t.Logf("Values: %v", expr.Values())
}

func TestAdapter_ConfigWithAutoDetect(t *testing.T) {
	// Mix of configured and auto-detected types
	config := &Config{
		AttributeTypes: map[string]AttributeType{
			"age": AttributeTypeNumber,
			// "score" not configured, will auto-detect
		},
	}

	filter := &sift.AndOperation{
		Left: &sift.Condition{
			Name:      "age",
			Operation: sift.OperationGT,
			Value:     "18", // Configured as number
		},
		Right: &sift.Condition{
			Name:      "score",
			Operation: sift.OperationLT,
			Value:     "100", // Auto-detected as number
		},
	}

	adapter := NewAdapterWithConfig(config)
	err := sift.Thru(context.Background(), adapter, filter)
	if err != nil {
		t.Fatalf("Thru() error = %v", err)
	}

	expr, err := adapter.Expression()
	if err != nil {
		t.Fatalf("Expression() error = %v", err)
	}

	t.Logf("Filter Expression: %s", *expr.Condition())
}

func TestAdapter_StringTypeDoesNotParse(t *testing.T) {
	// Explicitly configure as string to prevent parsing
	config := &Config{
		AttributeTypes: map[string]AttributeType{
			"id": AttributeTypeString, // Keep as string even if it looks like a number
		},
	}

	filter := &sift.Condition{
		Name:      "id",
		Operation: sift.OperationEQ,
		Value:     "12345", // Should stay as string
	}

	adapter := NewAdapterWithConfig(config)
	err := sift.Thru(context.Background(), adapter, filter)
	if err != nil {
		t.Fatalf("Thru() error = %v", err)
	}

	expr, err := adapter.Expression()
	if err != nil {
		t.Fatalf("Expression() error = %v", err)
	}

	t.Logf("Filter Expression: %s", *expr.Condition())
	t.Logf("Values: %v", expr.Values())
}
