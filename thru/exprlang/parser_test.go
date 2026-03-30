package exprlang

import (
	"reflect"
	"strings"
	"testing"

	"github.com/nisimpson/sift"
)

func TestParse_ComparisonOperators(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  *sift.Condition
	}{
		// == with different value types (Req 1.1, 1.7, 1.8, 1.9)
		{name: "eq string", input: `status == "active"`, want: &sift.Condition{Name: "status", Operation: sift.OperationEQ, Value: "active"}},
		{name: "eq int", input: `age == 25`, want: &sift.Condition{Name: "age", Operation: sift.OperationEQ, Value: "25"}},
		{name: "eq float", input: `score == 3.14`, want: &sift.Condition{Name: "score", Operation: sift.OperationEQ, Value: "3.14"}},
		{name: "eq bool true", input: `verified == true`, want: &sift.Condition{Name: "verified", Operation: sift.OperationEQ, Value: "true"}},
		{name: "eq bool false", input: `active == false`, want: &sift.Condition{Name: "active", Operation: sift.OperationEQ, Value: "false"}},

		// != with different value types (Req 1.2)
		{name: "neq string", input: `status != "deleted"`, want: &sift.Condition{Name: "status", Operation: sift.OperationNEQ, Value: "deleted"}},
		{name: "neq int", input: `total != 0`, want: &sift.Condition{Name: "total", Operation: sift.OperationNEQ, Value: "0"}},
		{name: "neq float", input: `rate != 1.5`, want: &sift.Condition{Name: "rate", Operation: sift.OperationNEQ, Value: "1.5"}},
		{name: "neq bool", input: `enabled != true`, want: &sift.Condition{Name: "enabled", Operation: sift.OperationNEQ, Value: "true"}},

		// < with different value types (Req 1.3)
		{name: "lt string", input: `name < "z"`, want: &sift.Condition{Name: "name", Operation: sift.OperationLT, Value: "z"}},
		{name: "lt int", input: `age < 18`, want: &sift.Condition{Name: "age", Operation: sift.OperationLT, Value: "18"}},
		{name: "lt float", input: `price < 9.99`, want: &sift.Condition{Name: "price", Operation: sift.OperationLT, Value: "9.99"}},
		{name: "lt large int", input: `priority < 10`, want: &sift.Condition{Name: "priority", Operation: sift.OperationLT, Value: "10"}},

		// <= with different value types (Req 1.4)
		{name: "lte string", input: `name <= "m"`, want: &sift.Condition{Name: "name", Operation: sift.OperationLTE, Value: "m"}},
		{name: "lte int", input: `score <= 100`, want: &sift.Condition{Name: "score", Operation: sift.OperationLTE, Value: "100"}},
		{name: "lte float", input: `rate <= 4.5`, want: &sift.Condition{Name: "rate", Operation: sift.OperationLTE, Value: "4.5"}},
		{name: "lte zero", input: `balance <= 0`, want: &sift.Condition{Name: "balance", Operation: sift.OperationLTE, Value: "0"}},

		// > with different value types (Req 1.5)
		{name: "gt string", input: `name > "a"`, want: &sift.Condition{Name: "name", Operation: sift.OperationGT, Value: "a"}},
		{name: "gt int", input: `age > 21`, want: &sift.Condition{Name: "age", Operation: sift.OperationGT, Value: "21"}},
		{name: "gt float", input: `price > 99.99`, want: &sift.Condition{Name: "price", Operation: sift.OperationGT, Value: "99.99"}},
		{name: "gt zero", input: `quantity > 0`, want: &sift.Condition{Name: "quantity", Operation: sift.OperationGT, Value: "0"}},

		// >= with different value types (Req 1.6)
		{name: "gte string", input: `name >= "b"`, want: &sift.Condition{Name: "name", Operation: sift.OperationGTE, Value: "b"}},
		{name: "gte int", input: `rating >= 4`, want: &sift.Condition{Name: "rating", Operation: sift.OperationGTE, Value: "4"}},
		{name: "gte float", input: `score >= 7.5`, want: &sift.Condition{Name: "score", Operation: sift.OperationGTE, Value: "7.5"}},
		{name: "gte large int", input: `views >= 1000`, want: &sift.Condition{Name: "views", Operation: sift.OperationGTE, Value: "1000"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tt.input, err)
			}
			cond, ok := got.(*sift.Condition)
			if !ok {
				t.Fatalf("Parse(%q) returned %T, want *sift.Condition", tt.input, got)
			}
			if !reflect.DeepEqual(cond, tt.want) {
				t.Errorf("Parse(%q) = %+v, want %+v", tt.input, cond, tt.want)
			}
		})
	}
}

func TestParse_NilChecks(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  *sift.Condition
	}{
		// Req 4.3: field == nil → OperationNotExists
		{name: "eq nil", input: `field == nil`, want: &sift.Condition{Name: "field", Operation: sift.OperationNotExists}},
		// Req 4.2: field != nil → OperationExists
		{name: "neq nil", input: `field != nil`, want: &sift.Condition{Name: "field", Operation: sift.OperationExists}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tt.input, err)
			}
			cond, ok := got.(*sift.Condition)
			if !ok {
				t.Fatalf("Parse(%q) returned %T, want *sift.Condition", tt.input, got)
			}
			if !reflect.DeepEqual(cond, tt.want) {
				t.Errorf("Parse(%q) = %+v, want %+v", tt.input, cond, tt.want)
			}
		})
	}
}

func TestParse_EmptyInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "empty string", input: ""},
		{name: "whitespace only", input: "   "},
		{name: "tabs and newlines", input: "\t\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			if err == nil {
				t.Fatalf("Parse(%q) expected error, got nil", tt.input)
			}
			if !strings.Contains(err.Error(), "empty input") {
				t.Errorf("Parse(%q) error = %q, want error containing 'empty input'", tt.input, err.Error())
			}
		})
	}
}

func TestParse_InvalidSyntax(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "missing operand", input: `status ==`},
		{name: "double operator", input: `a == == b`},
		{name: "unmatched paren", input: `(a == 1`},
		{name: "garbage input", input: `@#$%`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			if err == nil {
				t.Fatalf("Parse(%q) expected error, got nil", tt.input)
			}
			if !strings.Contains(err.Error(), "exprlang:") {
				t.Errorf("Parse(%q) error = %q, want error prefixed with 'exprlang:'", tt.input, err.Error())
			}
		})
	}
}

func TestParse_LogicalOperators(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  sift.Expression
	}{
		// && operator (Req 2.1)
		{
			name:  "and symbolic",
			input: `status == "active" && age > 18`,
			want: &sift.AndOperation{
				Left:  &sift.Condition{Name: "status", Operation: sift.OperationEQ, Value: "active"},
				Right: &sift.Condition{Name: "age", Operation: sift.OperationGT, Value: "18"},
			},
		},
		// and keyword (Req 2.2)
		{
			name:  "and keyword",
			input: `status == "active" and age > 18`,
			want: &sift.AndOperation{
				Left:  &sift.Condition{Name: "status", Operation: sift.OperationEQ, Value: "active"},
				Right: &sift.Condition{Name: "age", Operation: sift.OperationGT, Value: "18"},
			},
		},
		// || operator (Req 2.3)
		{
			name:  "or symbolic",
			input: `role == "admin" || role == "mod"`,
			want: &sift.OrOperation{
				Left:  &sift.Condition{Name: "role", Operation: sift.OperationEQ, Value: "admin"},
				Right: &sift.Condition{Name: "role", Operation: sift.OperationEQ, Value: "mod"},
			},
		},
		// or keyword (Req 2.4)
		{
			name:  "or keyword",
			input: `role == "admin" or role == "mod"`,
			want: &sift.OrOperation{
				Left:  &sift.Condition{Name: "role", Operation: sift.OperationEQ, Value: "admin"},
				Right: &sift.Condition{Name: "role", Operation: sift.OperationEQ, Value: "mod"},
			},
		},
		// ! operator (Req 2.5)
		{
			name:  "not symbolic",
			input: `!(status == "deleted")`,
			want: &sift.NotOperation{
				Child: &sift.Condition{Name: "status", Operation: sift.OperationEQ, Value: "deleted"},
			},
		},
		// not keyword (Req 2.6)
		{
			name:  "not keyword",
			input: `not (status == "deleted")`,
			want: &sift.NotOperation{
				Child: &sift.Condition{Name: "status", Operation: sift.OperationEQ, Value: "deleted"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tt.input, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParse_StringOperators(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  *sift.Condition
	}{
		// contains (Req 3.1)
		{name: "contains", input: `name contains "foo"`, want: &sift.Condition{Name: "name", Operation: sift.OperationContains, Value: "foo"}},
		{name: "contains phrase", input: `bio contains "hello world"`, want: &sift.Condition{Name: "bio", Operation: sift.OperationContains, Value: "hello world"}},
		// startsWith (Req 3.2)
		{name: "startsWith", input: `email startsWith "admin"`, want: &sift.Condition{Name: "email", Operation: sift.OperationBeginsWith, Value: "admin"}},
		{name: "startsWith prefix", input: `url startsWith "https"`, want: &sift.Condition{Name: "url", Operation: sift.OperationBeginsWith, Value: "https"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tt.input, err)
			}
			cond, ok := got.(*sift.Condition)
			if !ok {
				t.Fatalf("Parse(%q) returned %T, want *sift.Condition", tt.input, got)
			}
			if !reflect.DeepEqual(cond, tt.want) {
				t.Errorf("Parse(%q) = %+v, want %+v", tt.input, cond, tt.want)
			}
		})
	}
}

func TestParse_MembershipOperator(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  *sift.Condition
	}{
		// "value" in field (Req 4.1) — note reversed operand order
		{name: "in string", input: `"admin" in roles`, want: &sift.Condition{Name: "roles", Operation: sift.OperationIn, Value: "admin"}},
		{name: "in another value", input: `"active" in tags`, want: &sift.Condition{Name: "tags", Operation: sift.OperationIn, Value: "active"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tt.input, err)
			}
			cond, ok := got.(*sift.Condition)
			if !ok {
				t.Fatalf("Parse(%q) returned %T, want *sift.Condition", tt.input, got)
			}
			if !reflect.DeepEqual(cond, tt.want) {
				t.Errorf("Parse(%q) = %+v, want %+v", tt.input, cond, tt.want)
			}
		})
	}
}

func TestParse_PrecedenceAndGrouping(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  sift.Expression
	}{
		// && binds tighter than || (Req 2.8)
		// a || b && c  →  a || (b && c)
		{
			name:  "and binds tighter than or",
			input: `x == 1 || y == 2 && z == 3`,
			want: &sift.OrOperation{
				Left: &sift.Condition{Name: "x", Operation: sift.OperationEQ, Value: "1"},
				Right: &sift.AndOperation{
					Left:  &sift.Condition{Name: "y", Operation: sift.OperationEQ, Value: "2"},
					Right: &sift.Condition{Name: "z", Operation: sift.OperationEQ, Value: "3"},
				},
			},
		},
		// Parentheses override precedence (Req 2.7)
		// (a || b) && c
		{
			name:  "parens override precedence",
			input: `(x == 1 || y == 2) && z == 3`,
			want: &sift.AndOperation{
				Left: &sift.OrOperation{
					Left:  &sift.Condition{Name: "x", Operation: sift.OperationEQ, Value: "1"},
					Right: &sift.Condition{Name: "y", Operation: sift.OperationEQ, Value: "2"},
				},
				Right: &sift.Condition{Name: "z", Operation: sift.OperationEQ, Value: "3"},
			},
		},
		// and keyword binds tighter than or keyword (Req 2.8)
		{
			name:  "and keyword binds tighter than or keyword",
			input: `x == 1 or y == 2 and z == 3`,
			want: &sift.OrOperation{
				Left: &sift.Condition{Name: "x", Operation: sift.OperationEQ, Value: "1"},
				Right: &sift.AndOperation{
					Left:  &sift.Condition{Name: "y", Operation: sift.OperationEQ, Value: "2"},
					Right: &sift.Condition{Name: "z", Operation: sift.OperationEQ, Value: "3"},
				},
			},
		},
		// Parenthesized grouping with keyword operators (Req 2.7)
		{
			name:  "parens with keyword operators",
			input: `(x == 1 or y == 2) and z == 3`,
			want: &sift.AndOperation{
				Left: &sift.OrOperation{
					Left:  &sift.Condition{Name: "x", Operation: sift.OperationEQ, Value: "1"},
					Right: &sift.Condition{Name: "y", Operation: sift.OperationEQ, Value: "2"},
				},
				Right: &sift.Condition{Name: "z", Operation: sift.OperationEQ, Value: "3"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tt.input, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParse_Between(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  sift.Expression
	}{
		// Req 6.1: field >= min and field <= max → OperationBetween
		{
			name:  "between with and keyword integers",
			input: `age >= 18 and age <= 65`,
			want:  &sift.Condition{Name: "age", Operation: sift.OperationBetween, Value: "18,65"},
		},
		// Req 6.2: field >= min && field <= max → OperationBetween
		{
			name:  "between with && operator integers",
			input: `age >= 18 && age <= 65`,
			want:  &sift.Condition{Name: "age", Operation: sift.OperationBetween, Value: "18,65"},
		},
		// Between with float values
		{
			name:  "between with and keyword floats",
			input: `score >= 1.5 and score <= 9.9`,
			want:  &sift.Condition{Name: "score", Operation: sift.OperationBetween, Value: "1.5,9.9"},
		},
		{
			name:  "between with && operator floats",
			input: `price >= 0.99 && price <= 99.99`,
			want:  &sift.Condition{Name: "price", Operation: sift.OperationBetween, Value: "0.99,99.99"},
		},
		// Negative: field > min and field < max does NOT collapse to between
		{
			name:  "gt and lt does not collapse to between",
			input: `age > 18 and age < 65`,
			want: &sift.AndOperation{
				Left:  &sift.Condition{Name: "age", Operation: sift.OperationGT, Value: "18"},
				Right: &sift.Condition{Name: "age", Operation: sift.OperationLT, Value: "65"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tt.input, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParse_StrictMode(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantContains string // substring expected in the error message after the prefix
	}{
		// Req 5.1: function calls rejected
		{
			name:         "function call in comparison",
			input:        `len(items) > 5`,
			wantContains: "unsupported construct",
		},
		// Req 5.6: endsWith has no sift equivalent
		{
			name:         "endsWith operator",
			input:        `name endsWith "son"`,
			wantContains: "string operator (endsWith) has no sift equivalent",
		},
		// Req 5.6: matches has no sift equivalent
		{
			name:         "matches operator",
			input:        `name matches "^J.*"`,
			wantContains: "string operator (matches) has no sift equivalent",
		},
		// Req 5.3: arithmetic expressions rejected
		{
			name:         "arithmetic operator",
			input:        `price * 2 > 100`,
			wantContains: "unsupported construct",
		},
		// Req 5.4: ternary/conditional expression rejected
		{
			name:         "ternary conditional",
			input:        `age > 18 ? "adult" : "minor"`,
			wantContains: "unsupported construct",
		},
		// Req 5.5: pipe operator rejected
		{
			name:         "pipe operator",
			input:        `name | upper()`,
			wantContains: "unsupported construct",
		},
		// Req 5.2: array predicate any rejected
		{
			name:         "array predicate any",
			input:        `any(items, .price > 10)`,
			wantContains: "unsupported construct",
		},
		// Req 5.2: array predicate all rejected
		{
			name:         "array predicate all",
			input:        `all(items, .price > 10)`,
			wantContains: "unsupported construct",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			if err == nil {
				t.Fatalf("Parse(%q) expected error, got nil", tt.input)
			}
			errMsg := err.Error()
			if !strings.Contains(errMsg, "exprlang: unsupported construct:") {
				t.Errorf("Parse(%q) error = %q, want prefix 'exprlang: unsupported construct:'", tt.input, errMsg)
			}
			if !strings.Contains(errMsg, tt.wantContains) {
				t.Errorf("Parse(%q) error = %q, want substring %q", tt.input, errMsg, tt.wantContains)
			}
		})
	}
}

func TestParse_LenientMode(t *testing.T) {
	t.Run("function call wrapped as RawExpression", func(t *testing.T) {
		// Req 9.1, 9.2: function calls wrapped as RawExpression in lenient mode
		expr, err := Parse(`len(items) > 5`, WithLenientMode())
		if err != nil {
			t.Fatalf("Parse() unexpected error: %v", err)
		}
		if expr == nil {
			t.Fatal("Parse() returned nil expression")
		}
		got := expr.String()
		if !strings.Contains(got, "exprlang(") {
			t.Errorf("expected RawExpression wrapping, got String() = %q", got)
		}
	})

	t.Run("array predicate wrapped as RawExpression", func(t *testing.T) {
		// Req 9.3: array predicates wrapped as RawExpression in lenient mode
		expr, err := Parse(`any(items, .price > 10)`, WithLenientMode())
		if err != nil {
			t.Fatalf("Parse() unexpected error: %v", err)
		}
		if expr == nil {
			t.Fatal("Parse() returned nil expression")
		}
		got := expr.String()
		if !strings.Contains(got, "exprlang(") {
			t.Errorf("expected RawExpression wrapping, got String() = %q", got)
		}
	})

	t.Run("endsWith wrapped as RawExpression", func(t *testing.T) {
		// Req 3.4: endsWith wrapped in lenient mode
		expr, err := Parse(`name endsWith "son"`, WithLenientMode())
		if err != nil {
			t.Fatalf("Parse() unexpected error: %v", err)
		}
		if expr == nil {
			t.Fatal("Parse() returned nil expression")
		}
		got := expr.String()
		if !strings.Contains(got, "exprlang(") {
			t.Errorf("expected RawExpression wrapping, got String() = %q", got)
		}
	})

	t.Run("matches wrapped as RawExpression", func(t *testing.T) {
		// Req 3.6: matches wrapped in lenient mode
		expr, err := Parse(`name matches "^J.*"`, WithLenientMode())
		if err != nil {
			t.Fatalf("Parse() unexpected error: %v", err)
		}
		if expr == nil {
			t.Fatal("Parse() returned nil expression")
		}
		got := expr.String()
		if !strings.Contains(got, "exprlang(") {
			t.Errorf("expected RawExpression wrapping, got String() = %q", got)
		}
	})

	t.Run("mixed supported and unsupported produces AndOperation", func(t *testing.T) {
		// Req 9.4: status == "active" && len(items) > 5
		// Left should be *sift.Condition with OperationEQ, right should be RawExpression
		expr, err := Parse(`status == "active" && len(items) > 5`, WithLenientMode())
		if err != nil {
			t.Fatalf("Parse() unexpected error: %v", err)
		}
		if expr == nil {
			t.Fatal("Parse() returned nil expression")
		}

		andOp, ok := expr.(*sift.AndOperation)
		if !ok {
			t.Fatalf("expected *sift.AndOperation, got %T", expr)
		}

		// Left side: supported condition
		cond, ok := andOp.Left.(*sift.Condition)
		if !ok {
			t.Fatalf("expected left to be *sift.Condition, got %T", andOp.Left)
		}
		if cond.Name != "status" || cond.Operation != sift.OperationEQ || cond.Value != "active" {
			t.Errorf("left condition = %+v, want {Name:status Operation:eq Value:active}", cond)
		}

		// Right side: unsupported construct wrapped as RawExpression
		rightStr := andOp.Right.String()
		if !strings.Contains(rightStr, "exprlang(") {
			t.Errorf("expected right to be RawExpression, got String() = %q", rightStr)
		}
	})
}
