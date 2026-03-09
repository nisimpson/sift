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
			err := sift.Thru(context.Background(), adapter, sift.WithFilter(tt.condition))

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
	err := sift.Thru(context.Background(), adapter, sift.WithFilter(filter))
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
	err := sift.Thru(context.Background(), adapter, sift.WithFilter(filter))
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
	err := sift.Thru(context.Background(), adapter, sift.WithFilter(filter))
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
	err := sift.Thru(context.Background(), adapter, sift.WithFilter(filter))
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
			err := sift.Thru(context.Background(), adapter, sift.WithFilter(tt.condition))

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
	err := sift.Thru(context.Background(), adapter, sift.WithFilter(filter))
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
	err := sift.Thru(context.Background(), adapter, sift.WithFilter(filter))
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
	err := sift.Thru(context.Background(), adapter, sift.WithFilter(filter))
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

func TestAdapter_EvaluateSortField(t *testing.T) {
	tests := []struct {
		name             string
		sortExpr         sift.SortExpression
		wantForward      *bool
		wantErr          bool
	}{
		{
			name:        "ascending sort",
			sortExpr:    &sift.SortField{Name: "created_at", Direction: sift.SortAsc},
			wantForward: boolPtr(true),
			wantErr:     false,
		},
		{
			name:        "descending sort",
			sortExpr:    &sift.SortField{Name: "created_at", Direction: sift.SortDesc},
			wantForward: boolPtr(false),
			wantErr:     false,
		},
		{
			name: "multiple fields - only first is used",
			sortExpr: &sift.SortList{
				Fields: []*sift.SortField{
					{Name: "created_at", Direction: sift.SortDesc},
					{Name: "name", Direction: sift.SortAsc},
				},
			},
			wantForward: boolPtr(false), // Only first field matters
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter()
			err := sift.Thru(context.Background(), adapter, sift.WithSort(tt.sortExpr))

			if (err != nil) != tt.wantErr {
				t.Errorf("SortThru() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got := adapter.ScanIndexForward()
			if (got == nil) != (tt.wantForward == nil) {
				t.Errorf("ScanIndexForward() = %v, want %v", got, tt.wantForward)
				return
			}

			if got != nil && tt.wantForward != nil && *got != *tt.wantForward {
				t.Errorf("ScanIndexForward() = %v, want %v", *got, *tt.wantForward)
			}
		})
	}
}

func TestAdapter_SortBuilder(t *testing.T) {
	tests := []struct {
		name        string
		buildSort   func() sift.SortExpression
		wantForward *bool
	}{
		{
			name: "sort ascending",
			buildSort: func() sift.SortExpression {
				return sift.Sort("created_at", sift.SortAsc)
			},
			wantForward: boolPtr(true),
		},
		{
			name: "sort descending",
			buildSort: func() sift.SortExpression {
				return sift.Sort("created_at", sift.SortDesc)
			},
			wantForward: boolPtr(false),
		},
		{
			name: "sort with then by",
			buildSort: func() sift.SortExpression {
				return sift.Sort("created_at", sift.SortDesc).ThenBy("name", sift.SortAsc)
			},
			wantForward: boolPtr(false), // Only first field used
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter()
			sortExpr := tt.buildSort()
			err := sift.Thru(context.Background(), adapter, sift.WithSort(sortExpr))
			if err != nil {
				t.Fatalf("SortThru() error = %v", err)
			}

			got := adapter.ScanIndexForward()
			if (got == nil) != (tt.wantForward == nil) {
				t.Errorf("ScanIndexForward() = %v, want %v", got, tt.wantForward)
				return
			}

			if got != nil && tt.wantForward != nil && *got != *tt.wantForward {
				t.Errorf("ScanIndexForward() = %v, want %v", *got, *tt.wantForward)
			}
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}


func TestAdapter_Pagination_Cursor(t *testing.T) {
	tests := []struct {
		name      string
		page      sift.PaginationExpression
		wantLimit *int32
		wantToken string
	}{
		{
			name:      "with size and cursor",
			page:      sift.Paginate().Size(20).Cursor("token123"),
			wantLimit: int32Ptr(20),
			wantToken: "token123",
		},
		{
			name:      "with size only",
			page:      sift.Paginate().Size(50).Cursor(""),
			wantLimit: int32Ptr(50),
			wantToken: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter()
			err := sift.Thru(context.Background(), adapter, sift.WithPagination(tt.page))
			if err != nil {
				t.Fatalf("Thru() error = %v", err)
			}

			limit := adapter.Limit()
			if (limit == nil) != (tt.wantLimit == nil) {
				t.Errorf("Limit() = %v, want %v", limit, tt.wantLimit)
			} else if limit != nil && tt.wantLimit != nil && *limit != *tt.wantLimit {
				t.Errorf("Limit() = %d, want %d", *limit, *tt.wantLimit)
			}

			if adapter.Token() != tt.wantToken {
				t.Errorf("Token() = %s, want %s", adapter.Token(), tt.wantToken)
			}
		})
	}
}

func TestAdapter_FilterSortPage(t *testing.T) {
	filter := sift.Eq("status", "active")
	sort := sift.Sort("created_at", sift.SortDesc)
	page := sift.Paginate().Size(20).Cursor("token123")

	adapter := NewAdapter()
	err := sift.Thru(context.Background(), adapter,
		sift.WithFilter(filter),
		sift.WithSort(sort),
		sift.WithPagination(page))

	if err != nil {
		t.Fatalf("Thru() error = %v", err)
	}

	// Check filter
	expr, err := adapter.Expression()
	if err != nil {
		t.Fatalf("Expression() error = %v", err)
	}
	if expr.Condition() == nil {
		t.Error("Expected condition to be set")
	}

	// Check sort
	forward := adapter.ScanIndexForward()
	if forward == nil || *forward != false {
		t.Errorf("ScanIndexForward() = %v, want false", forward)
	}

	// Check pagination
	limit := adapter.Limit()
	if limit == nil || *limit != 20 {
		t.Errorf("Limit() = %v, want 20", limit)
	}

	if adapter.Token() != "token123" {
		t.Errorf("Token() = %s, want token123", adapter.Token())
	}
}

func int32Ptr(i int32) *int32 {
	return &i
}
