package sift

import (
	"context"
	"fmt"
	"testing"
)

// TestFormatFilter tests the FormatFilter function
func TestFormatFilter(t *testing.T) {
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
			want: `eq(path,C\:\\Users)`, // Backslash and colon are both escaped
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
			got, err := FormatFilter(tt.expr, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("FormatFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("FormatFilter() = %v, want %v", got, tt.want)
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
			query, err := ParseQuery("filter("+tt.input+")", nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				gotStr := query.Filter.(interface{ String() string }).String()
				if gotStr != tt.want {
					t.Errorf("ParseQuery() = %v, want %v", gotStr, tt.want)
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
			// Format the expression using FormatFilter
			formatted, err := FormatFilter(tt.expr, nil)
			if err != nil {
				t.Fatalf("FormatFilter() error = %v", err)
			}

			// Parse it back using ParseQuery
			query, err := ParseQuery("filter("+formatted+")", nil)
			if err != nil {
				t.Fatalf("ParseQuery() error = %v", err)
			}

			// Format again
			reformatted, err := FormatFilter(query.Filter, nil)
			if err != nil {
				t.Fatalf("FormatFilter() second time error = %v", err)
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

		formatted, err := Format(registry, WithFilter(expr))
		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}

		expected := "filter(geo_within(location))"
		if formatted != expected {
			t.Errorf("Format() = %v, want %v", formatted, expected)
		}
	})

	t.Run("parse custom expression", func(t *testing.T) {
		input := "geo_within(location)"

		query, err := ParseQuery("filter("+input+")", registry)
		if err != nil {
			t.Fatalf("ParseQuery() error = %v", err)
		}

		// Should be a customNodeWrapper
		wrapper, ok := query.Filter.(interface{ String() string })
		if !ok {
			t.Fatalf("Expected expression with String() method, got %T", query.Filter)
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

		formatted, err := Format(registry, WithFilter(expr))
		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}

		// Parse the complete query
		query, err := ParseQuery(formatted, registry)
		if err != nil {
			t.Fatalf("ParseQuery() error = %v", err)
		}

		reformatted, err := Format(registry, WithFilter(query.Filter))
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

		_, err := Format(nil, WithFilter(expr))
		if err == nil {
			t.Error("Expected error for unregistered custom expression")
		}
	})

	t.Run("parse unregistered custom expression", func(t *testing.T) {
		input := "unknown_custom(field)"

		_, err := ParseQuery("filter("+input+")", nil)
		if err == nil {
			t.Error("Expected error for unknown custom expression")
		}
	})

	t.Run("custom formatter parse error", func(t *testing.T) {
		// Register a formatter that returns an error
		registry := NewRegistry()
		registry.Register("error_custom", errorFormatter{})

		input := "error_custom(field)"
		_, err := ParseQuery("filter("+input+")", registry)
		if err == nil {
			t.Error("Expected error from custom formatter ParseCustomExpression")
		}
	})

	t.Run("custom formatter format error", func(t *testing.T) {
		registry := NewRegistry()
		registry.Register("error_custom", errorFormatter{})

		custom := &errorCustom{}
		expr := NewCustomExpression(custom)

		_, err := Format(registry, WithFilter(expr))
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
			got, err := FormatFilter(tt.expr, nil)
			if err != nil {
				t.Errorf("FormatFilter() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("FormatFilter() = %v, want %v", got, tt.want)
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

	_, err := FormatFilter(expr, nil)
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
			_, err := ParseQuery("filter("+tt.input+")", nil)
			if err == nil {
				t.Errorf("ParseQuery(%q) expected error, got nil", tt.input)
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
			input: "filter(ne(status,inactive))",
			want:  "ne(status,inactive)",
		},
		{
			name:  "OperationLT",
			input: "filter(lt(age,30))",
			want:  "lt(age,30)",
		},
		{
			name:  "OperationLTE",
			input: "filter(le(age,30))",
			want:  "le(age,30)",
		},
		{
			name:  "OperationGTE",
			input: "filter(ge(age,18))",
			want:  "ge(age,18)",
		},
		{
			name:  "OperationIn",
			input: "filter(in(status,active))",
			want:  "in(status,active)",
		},
		{
			name:  "OperationBetween",
			input: "filter(between(age,18))",
			want:  "between(age,18)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := ParseQuery(tt.input, nil)
			if err != nil {
				t.Errorf("ParseQuery() error = %v", err)
				return
			}
			if query.Filter == nil {
				t.Error("ParseQuery() returned nil filter")
				return
			}
			gotStr := query.Filter.(interface{ String() string }).String()
			if gotStr != tt.want {
				t.Errorf("ParseQuery() = %v, want %v", gotStr, tt.want)
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

		_, err := FormatFilter(expr, nil)
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

		_, err := FormatFilter(expr, nil)
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

		_, err := FormatFilter(expr, nil)
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

		_, err := FormatFilter(expr, nil)
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

	_, err := FormatFilter(expr, nil)
	if err == nil {
		t.Error("Expected error from child")
	}
}

// TestFormat tests the new unified Format function
func TestFormat(t *testing.T) {
	filter := &Condition{
		Name:      "status",
		Operation: OperationEQ,
		Value:     "active",
	}
	sort := Sort("created_at", SortDesc).ThenBy("name", SortAsc)
	page := Paginate().Size(20).Number(2)

	t.Run("filter only", func(t *testing.T) {
		got, err := Format(nil, WithFilter(filter))
		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}
		want := "filter(eq(status,active))"
		if got != want {
			t.Errorf("Format() = %v, want %v", got, want)
		}
	})

	t.Run("sort only", func(t *testing.T) {
		got, err := Format(nil, WithSort(sort))
		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}
		want := "sort(created_at:desc,name:asc)"
		if got != want {
			t.Errorf("Format() = %v, want %v", got, want)
		}
	})

	t.Run("pagination only", func(t *testing.T) {
		got, err := Format(nil, WithPagination(page))
		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}
		want := "page(size:20,number:2)"
		if got != want {
			t.Errorf("Format() = %v, want %v", got, want)
		}
	})

	t.Run("all combined", func(t *testing.T) {
		got, err := Format(nil,
			WithFilter(filter),
			WithSort(sort),
			WithPagination(page),
		)
		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}
		want := "filter(eq(status,active)),sort(created_at:desc,name:asc),page(size:20,number:2)"
		if got != want {
			t.Errorf("Format() = %v, want %v", got, want)
		}
	})

	t.Run("filter and sort", func(t *testing.T) {
		got, err := Format(nil,
			WithFilter(filter),
			WithSort(sort),
		)
		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}
		want := "filter(eq(status,active)),sort(created_at:desc,name:asc)"
		if got != want {
			t.Errorf("Format() = %v, want %v", got, want)
		}
	})
}

// TestFormatSort tests sort expression formatting
func TestFormatSort(t *testing.T) {
	tests := []struct {
		name string
		expr SortExpression
		want string
	}{
		{
			name: "single field ascending",
			expr: &SortField{Name: "name", Direction: SortAsc},
			want: "name:asc",
		},
		{
			name: "single field descending",
			expr: &SortField{Name: "created_at", Direction: SortDesc},
			want: "created_at:desc",
		},
		{
			name: "single field with nulls last",
			expr: &SortField{Name: "email", Direction: SortAsc, NullsLast: true},
			want: "email:asc:nullslast",
		},
		{
			name: "multiple fields",
			expr: &SortList{
				Fields: []*SortField{
					{Name: "created_at", Direction: SortDesc},
					{Name: "name", Direction: SortAsc},
				},
			},
			want: "created_at:desc,name:asc",
		},
		{
			name: "builder with multiple fields",
			expr: Sort("created_at", SortDesc).ThenBy("name", SortAsc),
			want: "created_at:desc,name:asc",
		},
		{
			name: "field with special characters",
			expr: &SortField{Name: "user:name", Direction: SortAsc},
			want: "user\\:name:asc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := formatSort(tt.expr)
			if err != nil {
				t.Fatalf("formatSort() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("formatSort() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestFormatPagination tests pagination expression formatting
func TestFormatPagination(t *testing.T) {
	tests := []struct {
		name string
		expr PaginationExpression
		want string
	}{
		{
			name: "offset pagination",
			expr: &OffsetPagination{Size: 20, Number: 2},
			want: "size:20,number:2",
		},
		{
			name: "cursor pagination",
			expr: &CursorPagination{Size: 20, Cursor: "token123"},
			want: "size:20,cursor:token123",
		},
		{
			name: "cursor with special characters",
			expr: &CursorPagination{Size: 20, Cursor: "token:123,abc"},
			want: "size:20,cursor:token\\:123\\,abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := formatPagination(tt.expr)
			if err != nil {
				t.Fatalf("formatPagination() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("formatPagination() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSortPaginationRoundTrip tests that format and parse are inverses
func TestSortPaginationRoundTrip(t *testing.T) {
	t.Run("sort round trip", func(t *testing.T) {
		original := Sort("created_at", SortDesc).ThenBy("name", SortAsc).ThenBy("email", SortAsc).NullsLast()

		formatted, err := Format(nil, WithSort(original))
		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}

		query, err := ParseQuery(formatted, nil)
		if err != nil {
			t.Fatalf("ParseQuery() error = %v", err)
		}

		reformatted, err := Format(nil, WithSort(query.Sort))
		if err != nil {
			t.Fatalf("Format() second time error = %v", err)
		}

		if formatted != reformatted {
			t.Errorf("Round trip failed: original = %v, after round trip = %v", formatted, reformatted)
		}
	})

	t.Run("offset pagination round trip", func(t *testing.T) {
		original := &OffsetPagination{Size: 20, Number: 2}

		formatted, err := Format(nil, WithPagination(original))
		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}

		query, err := ParseQuery(formatted, nil)
		if err != nil {
			t.Fatalf("ParseQuery() error = %v", err)
		}

		reformatted, err := Format(nil, WithPagination(query.Pagination))
		if err != nil {
			t.Fatalf("Format() second time error = %v", err)
		}

		if formatted != reformatted {
			t.Errorf("Round trip failed: original = %v, after round trip = %v", formatted, reformatted)
		}
	})

	t.Run("cursor pagination round trip", func(t *testing.T) {
		original := &CursorPagination{Size: 20, Cursor: "token:123,abc"}

		formatted, err := Format(nil, WithPagination(original))
		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}

		query, err := ParseQuery(formatted, nil)
		if err != nil {
			t.Fatalf("ParseQuery() error = %v", err)
		}

		reformatted, err := Format(nil, WithPagination(query.Pagination))
		if err != nil {
			t.Fatalf("Format() second time error = %v", err)
		}

		if formatted != reformatted {
			t.Errorf("Round trip failed: original = %v, after round trip = %v", formatted, reformatted)
		}
	})
}

// TestParseQuery tests parsing complete query strings
func TestParseQuery(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, *Query)
	}{
		{
			name:  "filter only",
			input: "filter(eq(status,active))",
			check: func(t *testing.T, q *Query) {
				if q.Filter == nil {
					t.Error("Filter should not be nil")
				}
				if q.Sort != nil {
					t.Error("Sort should be nil")
				}
				if q.Pagination != nil {
					t.Error("Pagination should be nil")
				}
			},
		},
		{
			name:  "sort only",
			input: "sort(created_at:desc)",
			check: func(t *testing.T, q *Query) {
				if q.Filter != nil {
					t.Error("Filter should be nil")
				}
				if q.Sort == nil {
					t.Error("Sort should not be nil")
				}
				if q.Pagination != nil {
					t.Error("Pagination should be nil")
				}
			},
		},
		{
			name:  "pagination only",
			input: "page(size:20,number:2)",
			check: func(t *testing.T, q *Query) {
				if q.Filter != nil {
					t.Error("Filter should be nil")
				}
				if q.Sort != nil {
					t.Error("Sort should be nil")
				}
				if q.Pagination == nil {
					t.Error("Pagination should not be nil")
				}
			},
		},
		{
			name:  "filter and sort",
			input: "filter(eq(status,active)),sort(created_at:desc)",
			check: func(t *testing.T, q *Query) {
				if q.Filter == nil {
					t.Error("Filter should not be nil")
				}
				if q.Sort == nil {
					t.Error("Sort should not be nil")
				}
				if q.Pagination != nil {
					t.Error("Pagination should be nil")
				}
			},
		},
		{
			name:  "all three",
			input: "filter(and(eq(status,active),gt(age,18))),sort(created_at:desc,name:asc),page(size:20,number:2)",
			check: func(t *testing.T, q *Query) {
				if q.Filter == nil {
					t.Error("Filter should not be nil")
				}
				if q.Sort == nil {
					t.Error("Sort should not be nil")
				}
				if q.Pagination == nil {
					t.Error("Pagination should not be nil")
				}
				
				// Verify pagination details
				offsetPage, ok := q.Pagination.(*OffsetPagination)
				if !ok {
					t.Errorf("Expected OffsetPagination, got %T", q.Pagination)
				} else {
					if offsetPage.Size != 20 {
						t.Errorf("Size = %d, want 20", offsetPage.Size)
					}
					if offsetPage.Number != 2 {
						t.Errorf("Number = %d, want 2", offsetPage.Number)
					}
				}
			},
		},
		{
			name:  "empty string",
			input: "",
			check: func(t *testing.T, q *Query) {
				if q.Filter != nil || q.Sort != nil || q.Pagination != nil {
					t.Error("All fields should be nil for empty input")
				}
			},
		},
		{
			name:  "with spaces",
			input: "filter(eq(status,active)) , sort(created_at:desc) , page(size:20,number:2)",
			check: func(t *testing.T, q *Query) {
				if q.Filter == nil || q.Sort == nil || q.Pagination == nil {
					t.Error("All fields should be populated")
				}
			},
		},
		{
			name:    "invalid filter",
			input:   "filter(invalid)",
			wantErr: true,
		},
		{
			name:    "invalid sort",
			input:   "sort(field:invalid)",
			wantErr: true,
		},
		{
			name:    "invalid pagination",
			input:   "page(invalid)",
			wantErr: true,
		},
		{
			name:    "unknown expression type",
			input:   "unknown(something)",
			wantErr: true,
		},
		{
			name:    "mismatched parentheses",
			input:   "filter(eq(status,active)",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseQuery(tt.input, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

// TestParseQueryRoundTrip tests that Format and ParseQuery are inverses
func TestParseQueryRoundTrip(t *testing.T) {
	filter := &Condition{
		Name:      "status",
		Operation: OperationEQ,
		Value:     "active",
	}
	sort := Sort("created_at", SortDesc).ThenBy("name", SortAsc)
	page := Paginate().Size(20).Number(2)

	// Format
	formatted, err := Format(nil,
		WithFilter(filter),
		WithSort(sort),
		WithPagination(page),
	)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Parse
	parsed, err := ParseQuery(formatted, nil)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	// Verify all components are present
	if parsed.Filter == nil {
		t.Error("Filter should not be nil")
	}
	if parsed.Sort == nil {
		t.Error("Sort should not be nil")
	}
	if parsed.Pagination == nil {
		t.Error("Pagination should not be nil")
	}

	// Format again
	reformatted, err := Format(nil,
		WithFilter(parsed.Filter),
		WithSort(parsed.Sort),
		WithPagination(parsed.Pagination),
	)
	if err != nil {
		t.Fatalf("Format() second time error = %v", err)
	}

	// Should be identical
	if formatted != reformatted {
		t.Errorf("Round trip failed:\nOriginal:    %s\nReformatted: %s", formatted, reformatted)
	}
}
