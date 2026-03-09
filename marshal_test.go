package sift

import (
	"context"
	"fmt"
	"testing"
)

// TestFormat tests the Format function
func TestFormat(t *testing.T) {
	tests := []struct {
		name    string
		expr    Expression
		want    string
		wantErr bool
	}{
		{
			name: "simple condition",
			expr: &Condition{
				Name:      "status",
				Operation: OperationEQ,
				Value:     "active",
			},
			want: "eq(status,active)",
		},
		{
			name: "condition with special characters",
			expr: &Condition{
				Name:      "email",
				Operation: OperationContains,
				Value:     "user@example.com",
			},
			want: "contains(email,user@example.com)",
		},
		{
			name: "condition with comma in value",
			expr: &Condition{
				Name:      "tags",
				Operation: OperationEQ,
				Value:     "red,blue",
			},
			want: `eq(tags,red\,blue)`,
		},
		{
			name: "condition with parentheses in value",
			expr: &Condition{
				Name:      "formula",
				Operation: OperationEQ,
				Value:     "(a+b)",
			},
			want: `eq(formula,\(a+b\))`,
		},
		{
			name: "condition with backslash in value",
			expr: &Condition{
				Name:      "path",
				Operation: OperationEQ,
				Value:     `C:\Users`,
			},
			want: `eq(path,C:\\Users)`,
		},
		{
			name: "exists operation",
			expr: &Condition{
				Name:      "email",
				Operation: OperationExists,
			},
			want: "exists(email)",
		},
		{
			name: "not_exists operation",
			expr: &Condition{
				Name:      "deleted_at",
				Operation: OperationNotExists,
			},
			want: "not_exists(deleted_at)",
		},
		{
			name: "and operation",
			expr: &AndOperation{
				Left: &Condition{
					Name:      "status",
					Operation: OperationEQ,
					Value:     "active",
				},
				Right: &Condition{
					Name:      "age",
					Operation: OperationGT,
					Value:     "18",
				},
			},
			want: "and(eq(status,active),gt(age,18))",
		},
		{
			name: "or operation",
			expr: &OrOperation{
				Left: &Condition{
					Name:      "role",
					Operation: OperationEQ,
					Value:     "admin",
				},
				Right: &Condition{
					Name:      "role",
					Operation: OperationEQ,
					Value:     "moderator",
				},
			},
			want: "or(eq(role,admin),eq(role,moderator))",
		},
		{
			name: "not operation",
			expr: &NotOperation{
				Child: &Condition{
					Name:      "deleted",
					Operation: OperationEQ,
					Value:     "true",
				},
			},
			want: "not(eq(deleted,true))",
		},
		{
			name: "nested and/or",
			expr: &AndOperation{
				Left: &OrOperation{
					Left: &Condition{
						Name:      "role",
						Operation: OperationEQ,
						Value:     "admin",
					},
					Right: &Condition{
						Name:      "role",
						Operation: OperationEQ,
						Value:     "moderator",
					},
				},
				Right: &Condition{
					Name:      "verified",
					Operation: OperationEQ,
					Value:     "true",
				},
			},
			want: "and(or(eq(role,admin),eq(role,moderator)),eq(verified,true))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Format(tt.expr, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("Format() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Format() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParse tests the Parse function
func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string // Expected String() output
		wantErr bool
	}{
		{
			name:  "simple condition",
			input: "eq(status,active)",
			want:  "eq(status,active)",
		},
		{
			name:  "greater than",
			input: "gt(age,18)",
			want:  "gt(age,18)",
		},
		{
			name:  "contains",
			input: "contains(email,@example.com)",
			want:  "contains(email,@example.com)",
		},
		{
			name:  "begins_with",
			input: "begins_with(name,John)",
			want:  "begins_with(name,John)",
		},
		{
			name:  "exists",
			input: "exists(email)",
			want:  "exists(email,)",
		},
		{
			name:  "not_exists",
			input: "not_exists(deleted_at)",
			want:  "not_exists(deleted_at,)",
		},
		{
			name:  "escaped comma",
			input: `eq(tags,red\,blue)`,
			want:  "eq(tags,red,blue)",
		},
		{
			name:  "escaped parentheses",
			input: `eq(formula,\(a+b\))`,
			want:  "eq(formula,(a+b))",
		},
		{
			name:  "escaped backslash",
			input: `eq(path,C:\\Users)`,
			want:  `eq(path,C:\Users)`,
		},
		{
			name:  "and operation",
			input: "and(eq(status,active),gt(age,18))",
			want:  "and(eq(status,active),gt(age,18))",
		},
		{
			name:  "or operation",
			input: "or(eq(role,admin),eq(role,moderator))",
			want:  "or(eq(role,admin),eq(role,moderator))",
		},
		{
			name:  "not operation",
			input: "not(eq(deleted,true))",
			want:  "not(eq(deleted,true))",
		},
		{
			name:  "nested and/or",
			input: "and(or(eq(role,admin),eq(role,moderator)),eq(verified,true))",
			want:  "and(or(eq(role,admin),eq(role,moderator)),eq(verified,true))",
		},
		{
			name:  "deeply nested",
			input: "and(or(eq(a,1),eq(b,2)),or(eq(c,3),eq(d,4)))",
			want:  "and(or(eq(a,1),eq(b,2)),or(eq(c,3),eq(d,4)))",
		},
		{
			name:  "not with nested and",
			input: "not(and(eq(a,1),eq(b,2)))",
			want:  "not(and(eq(a,1),eq(b,2)))",
		},
		{
			name:    "invalid function",
			input:   "invalid(field,value)",
			wantErr: true,
		},
		{
			name:    "missing closing paren",
			input:   "eq(status,active",
			wantErr: true,
		},
		{
			name:    "missing comma",
			input:   "eq(status active)",
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				gotStr := got.(interface{ String() string }).String()
				if gotStr != tt.want {
					t.Errorf("Parse() = %v, want %v", gotStr, tt.want)
				}
			}
		})
	}
}

// TestFormatParseRoundTrip tests that Format and Parse are inverses
func TestFormatParseRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		expr Expression
	}{
		{
			name: "simple condition",
			expr: &Condition{
				Name:      "status",
				Operation: OperationEQ,
				Value:     "active",
			},
		},
		{
			name: "and operation",
			expr: &AndOperation{
				Left: &Condition{
					Name:      "status",
					Operation: OperationEQ,
					Value:     "active",
				},
				Right: &Condition{
					Name:      "age",
					Operation: OperationGT,
					Value:     "18",
				},
			},
		},
		{
			name: "complex nested",
			expr: &AndOperation{
				Left: &OrOperation{
					Left: &Condition{
						Name:      "role",
						Operation: OperationEQ,
						Value:     "admin",
					},
					Right: &Condition{
						Name:      "role",
						Operation: OperationEQ,
						Value:     "moderator",
					},
				},
				Right: &NotOperation{
					Child: &Condition{
						Name:      "deleted",
						Operation: OperationEQ,
						Value:     "true",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Format the expression
			formatted, err := Format(tt.expr, nil)
			if err != nil {
				t.Fatalf("Format() error = %v", err)
			}

			// Parse it back
			parsed, err := Parse(formatted, nil)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			// Format again
			reformatted, err := Format(parsed, nil)
			if err != nil {
				t.Fatalf("Format() second time error = %v", err)
			}

			// Should be identical
			if formatted != reformatted {
				t.Errorf("Round trip failed: original = %v, after round trip = %v", formatted, reformatted)
			}
		})
	}
}

// TestParserMethods tests the exported Parser methods
func TestParserMethods(t *testing.T) {
	t.Run("ReadValue", func(t *testing.T) {
		p := &Parser{input: "hello,world", pos: 0}
		val := p.ReadValue()
		if val != "hello" {
			t.Errorf("ReadValue() = %v, want hello", val)
		}
		if p.pos != 5 {
			t.Errorf("pos = %v, want 5", p.pos)
		}
	})

	t.Run("ReadValue with escape", func(t *testing.T) {
		p := &Parser{input: `hello\,world,next`, pos: 0}
		val := p.ReadValue()
		if val != "hello,world" {
			t.Errorf("ReadValue() = %v, want hello,world", val)
		}
	})

	t.Run("SkipWhitespace", func(t *testing.T) {
		p := &Parser{input: "   hello", pos: 0}
		p.SkipWhitespace()
		if p.pos != 3 {
			t.Errorf("pos = %v, want 3", p.pos)
		}
	})

	t.Run("Expect success", func(t *testing.T) {
		p := &Parser{input: "(hello", pos: 0}
		if !p.Expect('(') {
			t.Error("Expect('(') should return true")
		}
		if p.pos != 1 {
			t.Errorf("pos = %v, want 1", p.pos)
		}
	})

	t.Run("Expect failure", func(t *testing.T) {
		p := &Parser{input: "hello", pos: 0}
		if p.Expect('(') {
			t.Error("Expect('(') should return false")
		}
		if p.pos != 0 {
			t.Errorf("pos = %v, want 0", p.pos)
		}
	})
}

// TestRegistryDuplicatePanic tests that Registry.Register panics on duplicate names
func TestRegistryDuplicatePanic(t *testing.T) {
	registry := NewRegistry()
	registry.Register("test", geoFormatter{})

	// Attempting to register the same name again should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when registering duplicate name")
		}
	}()

	registry.Register("test", geoFormatter{})
}

// geoWithin is a test custom expression type
type geoWithin struct {
	field  string
	lat    float64
	lng    float64
	radius float64
}

func (g *geoWithin) Type() string {
	return "geo_within"
}

func (g *geoWithin) String() string {
	return "geo_within(" + g.field + ")"
}

// geoFormatter is a test custom formatter
type geoFormatter struct{}

func (geoFormatter) FormatCustomExpression(expr CustomExpression) (string, error) {
	g := expr.(*geoWithin)
	return "geo_within(" + g.field + ")", nil
}

func (geoFormatter) ParseCustomExpression(p *Parser) (CustomExpression, error) {
	field := p.ReadValue()
	p.Expect(')')
	return &geoWithin{field: field}, nil
}

// errorFormatter is a test formatter that returns errors
type errorFormatter struct{}

func (errorFormatter) FormatCustomExpression(expr CustomExpression) (string, error) {
	return "", fmt.Errorf("format error")
}

func (errorFormatter) ParseCustomExpression(p *Parser) (CustomExpression, error) {
	return nil, fmt.Errorf("parse error")
}

// errorCustom is a test custom expression for error cases
type errorCustom struct{}

func (e *errorCustom) Type() string   { return "error_custom" }
func (e *errorCustom) String() string { return "error_custom()" }

// TestCustomExpressionRegistry tests custom expression registration with registry
func TestCustomExpressionRegistry(t *testing.T) {
	// Create a registry and register the custom expression
	registry := NewRegistry()
	registry.Register("geo_within", geoFormatter{})

	t.Run("format custom expression", func(t *testing.T) {
		custom := &geoWithin{
			field:  "location",
			lat:    40.7128,
			lng:    -74.0060,
			radius: 5000,
		}
		expr := NewCustomExpression(custom)

		formatted, err := Format(expr, registry)
		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}

		expected := "geo_within(location)"
		if formatted != expected {
			t.Errorf("Format() = %v, want %v", formatted, expected)
		}
	})

	t.Run("parse custom expression", func(t *testing.T) {
		input := "geo_within(location)"

		parsed, err := Parse(input, registry)
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		// Should be a customNodeWrapper
		wrapper, ok := parsed.(interface{ String() string })
		if !ok {
			t.Fatalf("Expected expression with String() method, got %T", parsed)
		}

		str := wrapper.String()
		expected := "geo_within(location)"
		if str != expected {
			t.Errorf("String() = %v, want %v", str, expected)
		}
	})

	t.Run("format and parse round trip", func(t *testing.T) {
		custom := &geoWithin{field: "location"}
		expr := NewCustomExpression(custom)

		formatted, err := Format(expr, registry)
		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}

		parsed, err := Parse(formatted, registry)
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		reformatted, err := Format(parsed, registry)
		if err != nil {
			t.Fatalf("Format() second time error = %v", err)
		}

		if formatted != reformatted {
			t.Errorf("Round trip failed: original = %v, after round trip = %v", formatted, reformatted)
		}
	})

	t.Run("custom expression with Thru", func(t *testing.T) {
		custom := &geoWithin{field: "location"}
		expr := NewCustomExpression(custom)

		adapter := newMockAdapter()
		err := Thru(context.Background(), adapter, WithFilter(expr))
		if err != nil {
			t.Errorf("Thru() error = %v", err)
		}

		if !adapter.customCalled {
			t.Error("EvaluateCustom should have been called")
		}
	})
}

// unregisteredCustom is a test type for error cases
type unregisteredCustom struct{}

func (u *unregisteredCustom) Type() string   { return "unregistered" }
func (u *unregisteredCustom) String() string { return "unregistered()" }

// TestCustomExpressionErrors tests error cases for custom expressions
func TestCustomExpressionErrors(t *testing.T) {
	t.Run("format unregistered custom expression", func(t *testing.T) {
		custom := &unregisteredCustom{}
		expr := NewCustomExpression(custom)

		_, err := Format(expr, nil)
		if err == nil {
			t.Error("Expected error for unregistered custom expression")
		}
	})

	t.Run("parse unregistered custom expression", func(t *testing.T) {
		input := "unknown_custom(field)"

		_, err := Parse(input, nil)
		if err == nil {
			t.Error("Expected error for unknown custom expression")
		}
	})

	t.Run("custom formatter parse error", func(t *testing.T) {
		// Register a formatter that returns an error
		registry := NewRegistry()
		registry.Register("error_custom", errorFormatter{})

		input := "error_custom(field)"
		_, err := Parse(input, registry)
		if err == nil {
			t.Error("Expected error from custom formatter ParseCustomExpression")
		}
	})

	t.Run("custom formatter format error", func(t *testing.T) {
		registry := NewRegistry()
		registry.Register("error_custom", errorFormatter{})
		
		custom := &errorCustom{}
		expr := NewCustomExpression(custom)

		_, err := Format(expr, registry)
		if err == nil {
			t.Error("Expected error from custom formatter FormatCustomExpression")
		}
	})
}

// TestFormatAllOperations tests formatting for all operation types
func TestFormatAllOperations(t *testing.T) {
	tests := []struct {
		name string
		expr *Condition
		want string
	}{
		{
			name: "OperationNEQ",
			expr: &Condition{Name: "status", Operation: OperationNEQ, Value: "inactive"},
			want: "ne(status,inactive)",
		},
		{
			name: "OperationLT",
			expr: &Condition{Name: "age", Operation: OperationLT, Value: "30"},
			want: "lt(age,30)",
		},
		{
			name: "OperationLTE",
			expr: &Condition{Name: "age", Operation: OperationLTE, Value: "30"},
			want: "le(age,30)",
		},
		{
			name: "OperationGTE",
			expr: &Condition{Name: "age", Operation: OperationGTE, Value: "18"},
			want: "ge(age,18)",
		},
		{
			name: "OperationIn",
			expr: &Condition{Name: "status", Operation: OperationIn, Value: "active"},
			want: "in(status,active)",
		},
		{
			name: "OperationBetween",
			expr: &Condition{Name: "age", Operation: OperationBetween, Value: "18"},
			want: "between(age,18)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Format(tt.expr, nil)
			if err != nil {
				t.Errorf("Format() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Format() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestFormatUnsupportedOperation tests formatting with unsupported operation
func TestFormatUnsupportedOperation(t *testing.T) {
	expr := &Condition{
		Name:      "field",
		Operation: Operation("unsupported"),
		Value:     "value",
	}

	_, err := Format(expr, nil)
	if err == nil {
		t.Error("Expected error for unsupported operation")
	}
}

// TestParseErrors tests various parsing error conditions
func TestParseErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "missing function name",
			input: "()",
		},
		{
			name:  "missing opening paren",
			input: "eq status,active)",
		},
		{
			name:  "missing field in condition",
			input: "eq(,value)",
		},
		{
			name:  "missing comma in condition",
			input: "eq(field value)",
		},
		{
			name:  "missing closing paren in condition",
			input: "eq(field,value",
		},
		{
			name:  "missing field in exists",
			input: "exists()",
		},
		{
			name:  "missing closing paren in exists",
			input: "exists(field",
		},
		{
			name:  "missing comma in and",
			input: "and(eq(a,1) eq(b,2))",
		},
		{
			name:  "missing closing paren in and",
			input: "and(eq(a,1),eq(b,2)",
		},
		{
			name:  "error in left side of and",
			input: "and(invalid,eq(b,2))",
		},
		{
			name:  "error in right side of and",
			input: "and(eq(a,1),invalid)",
		},
		{
			name:  "missing comma in or",
			input: "or(eq(a,1) eq(b,2))",
		},
		{
			name:  "missing closing paren in or",
			input: "or(eq(a,1),eq(b,2)",
		},
		{
			name:  "error in left side of or",
			input: "or(invalid,eq(b,2))",
		},
		{
			name:  "error in right side of or",
			input: "or(eq(a,1),invalid)",
		},
		{
			name:  "missing closing paren in not",
			input: "not(eq(a,1)",
		},
		{
			name:  "error in child of not",
			input: "not(invalid)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input, nil)
			if err == nil {
				t.Errorf("Parse(%q) expected error, got nil", tt.input)
			}
		})
	}
}

// TestParseAllOperations tests parsing for all operation types
func TestParseAllOperations(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "OperationNEQ",
			input: "ne(status,inactive)",
			want:  "ne(status,inactive)",
		},
		{
			name:  "OperationLT",
			input: "lt(age,30)",
			want:  "lt(age,30)",
		},
		{
			name:  "OperationLTE",
			input: "le(age,30)",
			want:  "le(age,30)",
		},
		{
			name:  "OperationGTE",
			input: "ge(age,18)",
			want:  "ge(age,18)",
		},
		{
			name:  "OperationIn",
			input: "in(status,active)",
			want:  "in(status,active)",
		},
		{
			name:  "OperationBetween",
			input: "between(age,18)",
			want:  "between(age,18)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input, nil)
			if err != nil {
				t.Errorf("Parse() error = %v", err)
				return
			}
			gotStr := got.(interface{ String() string }).String()
			if gotStr != tt.want {
				t.Errorf("Parse() = %v, want %v", gotStr, tt.want)
			}
		})
	}
}

// TestFormatAndErrors tests error propagation in formatAnd
func TestFormatAndErrors(t *testing.T) {
	t.Run("error in left side", func(t *testing.T) {
		expr := &AndOperation{
			Left: &Condition{
				Name:      "field",
				Operation: Operation("invalid"),
				Value:     "value",
			},
			Right: &Condition{
				Name:      "status",
				Operation: OperationEQ,
				Value:     "active",
			},
		}

		_, err := Format(expr, nil)
		if err == nil {
			t.Error("Expected error from left side")
		}
	})

	t.Run("error in right side", func(t *testing.T) {
		expr := &AndOperation{
			Left: &Condition{
				Name:      "status",
				Operation: OperationEQ,
				Value:     "active",
			},
			Right: &Condition{
				Name:      "field",
				Operation: Operation("invalid"),
				Value:     "value",
			},
		}

		_, err := Format(expr, nil)
		if err == nil {
			t.Error("Expected error from right side")
		}
	})
}

// TestFormatOrErrors tests error propagation in formatOr
func TestFormatOrErrors(t *testing.T) {
	t.Run("error in left side", func(t *testing.T) {
		expr := &OrOperation{
			Left: &Condition{
				Name:      "field",
				Operation: Operation("invalid"),
				Value:     "value",
			},
			Right: &Condition{
				Name:      "status",
				Operation: OperationEQ,
				Value:     "active",
			},
		}

		_, err := Format(expr, nil)
		if err == nil {
			t.Error("Expected error from left side")
		}
	})

	t.Run("error in right side", func(t *testing.T) {
		expr := &OrOperation{
			Left: &Condition{
				Name:      "status",
				Operation: OperationEQ,
				Value:     "active",
			},
			Right: &Condition{
				Name:      "field",
				Operation: Operation("invalid"),
				Value:     "value",
			},
		}

		_, err := Format(expr, nil)
		if err == nil {
			t.Error("Expected error from right side")
		}
	})
}

// TestFormatNotErrors tests error propagation in formatNot
func TestFormatNotErrors(t *testing.T) {
	expr := &NotOperation{
		Child: &Condition{
			Name:      "field",
			Operation: Operation("invalid"),
			Value:     "value",
		},
	}

	_, err := Format(expr, nil)
	if err == nil {
		t.Error("Expected error from child")
	}
}
