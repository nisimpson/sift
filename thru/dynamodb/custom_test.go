package dynamodb_test

import (
	"context"
	"testing"

	"github.com/nisimpson/sift"
	"github.com/nisimpson/sift/thru/dynamodb"
)

func TestCustomExpression_Size(t *testing.T) {
	tests := []struct {
		name    string
		expr    sift.Expression
		wantErr bool
	}{
		{
			name:    "size greater than",
			expr:    dynamodb.Size("tags", sift.OperationGT, 5),
			wantErr: false,
		},
		{
			name:    "size equal",
			expr:    dynamodb.Size("items", sift.OperationEQ, 10),
			wantErr: false,
		},
		{
			name:    "size less than or equal",
			expr:    dynamodb.Size("name", sift.OperationLTE, 100),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := dynamodb.NewAdapter()
			err := sift.Thru(context.Background(), adapter, tt.expr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Size() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify we can build the expression
				_, err := adapter.Expression()
				if err != nil {
					t.Errorf("Build() error = %v", err)
				}
			}
		})
	}
}

func TestCustomExpression_AttributeType(t *testing.T) {
	tests := []struct {
		name     string
		expr     sift.Expression
		wantErr  bool
	}{
		{
			name:    "attribute type string",
			expr:    dynamodb.IsAttributeType("name", "S"),
			wantErr: false,
		},
		{
			name:    "attribute type list",
			expr:    dynamodb.IsAttributeType("tags", "L"),
			wantErr: false,
		},
		{
			name:    "attribute type map",
			expr:    dynamodb.IsAttributeType("metadata", "M"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := dynamodb.NewAdapter()
			err := sift.Thru(context.Background(), adapter, tt.expr)

			if (err != nil) != tt.wantErr {
				t.Errorf("IsAttributeType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify we can build the expression
				_, err := adapter.Expression()
				if err != nil {
					t.Errorf("Build() error = %v", err)
				}
			}
		})
	}
}

func TestCustomExpression_MixedWithStandard(t *testing.T) {
	// status = "active" AND size(tags) > 3
	filter := sift.Eq("status", "active").And(dynamodb.Size("tags", sift.OperationGT, 3))

	adapter := dynamodb.NewAdapter()
	err := sift.Thru(context.Background(), adapter, filter)
	if err != nil {
		t.Fatalf("Mixed expression error = %v", err)
	}

	// Verify we can build the expression
	_, err = adapter.Expression()
	if err != nil {
		t.Errorf("Expression() error = %v", err)
	}
}

func TestCustomExpression_Serialization(t *testing.T) {
	// Create registry with DynamoDB custom expressions
	registry := dynamodb.NewRegistry()

	tests := []struct {
		name string
		expr sift.Expression
		want string
	}{
		{
			name: "size expression",
			expr: dynamodb.Size("tags", sift.OperationGT, 5),
			want: "size(tags,gt,5)",
		},
		{
			name: "attribute type expression",
			expr: dynamodb.IsAttributeType("metadata", "M"),
			want: "attribute_type(metadata,M)",
		},
		{
			name: "mixed with standard",
			expr: sift.Eq("status", "active").And(dynamodb.Size("tags", sift.OperationGT, 3)),
			want: "and(eq(status,active),size(tags,gt,3))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test formatting
			formatted, err := sift.Format(tt.expr, registry)
			if err != nil {
				t.Fatalf("Format() error = %v", err)
			}
			if formatted != tt.want {
				t.Errorf("Format() = %v, want %v", formatted, tt.want)
			}

			// Test parsing (round-trip)
			parsed, err := sift.Parse(formatted, registry)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			// Format again to verify round-trip
			reformatted, err := sift.Format(parsed, registry)
			if err != nil {
				t.Fatalf("Format() after Parse() error = %v", err)
			}
			if reformatted != tt.want {
				t.Errorf("Round-trip Format() = %v, want %v", reformatted, tt.want)
			}
		})
	}
}
