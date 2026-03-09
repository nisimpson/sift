package exprlang

import (
	"context"
	"testing"

	"github.com/nisimpson/sift"
)

func TestAdapter_EvaluateCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition *sift.Condition
		want      string
		wantErr   bool
	}{
		{
			name: "equality with string",
			condition: &sift.Condition{
				Name:      "status",
				Operation: sift.OperationEQ,
				Value:     "active",
			},
			want: `status == "active"`,
		},
		{
			name: "equality with number",
			condition: &sift.Condition{
				Name:      "age",
				Operation: sift.OperationEQ,
				Value:     "25",
			},
			want: "age == 25",
		},
		{
			name: "equality with boolean",
			condition: &sift.Condition{
				Name:      "verified",
				Operation: sift.OperationEQ,
				Value:     "true",
			},
			want: "verified == true",
		},
		{
			name: "not equal",
			condition: &sift.Condition{
				Name:      "status",
				Operation: sift.OperationNEQ,
				Value:     "deleted",
			},
			want: `status != "deleted"`,
		},
		{
			name: "less than",
			condition: &sift.Condition{
				Name:      "age",
				Operation: sift.OperationLT,
				Value:     "18",
			},
			want: "age < 18",
		},
		{
			name: "less than or equal",
			condition: &sift.Condition{
				Name:      "score",
				Operation: sift.OperationLTE,
				Value:     "100",
			},
			want: "score <= 100",
		},
		{
			name: "greater than",
			condition: &sift.Condition{
				Name:      "price",
				Operation: sift.OperationGT,
				Value:     "99.99",
			},
			want: "price > 99.99",
		},
		{
			name: "greater than or equal",
			condition: &sift.Condition{
				Name:      "rating",
				Operation: sift.OperationGTE,
				Value:     "4.5",
			},
			want: "rating >= 4.5",
		},
		{
			name: "in",
			condition: &sift.Condition{
				Name:      "roles",
				Operation: sift.OperationIn,
				Value:     "admin",
			},
			want: `"admin" in roles`,
		},
		{
			name: "contains",
			condition: &sift.Condition{
				Name:      "email",
				Operation: sift.OperationContains,
				Value:     "@example.com",
			},
			want: `email contains "@example.com"`,
		},
		{
			name: "begins with",
			condition: &sift.Condition{
				Name:      "name",
				Operation: sift.OperationBeginsWith,
				Value:     "John",
			},
			want: `name startsWith "John"`,
		},
		{
			name: "exists",
			condition: &sift.Condition{
				Name:      "optional_field",
				Operation: sift.OperationExists,
			},
			want: "optional_field != nil",
		},
		{
			name: "not exists",
			condition: &sift.Condition{
				Name:      "deleted_at",
				Operation: sift.OperationNotExists,
			},
			want: "deleted_at == nil",
		},
		{
			name: "between",
			condition: &sift.Condition{
				Name:      "age",
				Operation: sift.OperationBetween,
				Value:     "18,65",
			},
			want: "age >= 18 and age <= 65",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter()
			err := sift.Thru(context.Background(), adapter, sift.WithFilter(tt.condition))

			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got := adapter.Expression(); got != tt.want {
				t.Errorf("EvaluateCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdapter_EvaluateAnd(t *testing.T) {
	filter := sift.Eq("status", "active").And(sift.Gt("age", 18))

	adapter := NewAdapter()
	err := sift.Thru(context.Background(), adapter, sift.WithFilter(filter))
	if err != nil {
		t.Fatalf("EvaluateAnd() error = %v", err)
	}

	want := `(status == "active") && (age > 18)`
	if got := adapter.Expression(); got != want {
		t.Errorf("EvaluateAnd() = %v, want %v", got, want)
	}
}

func TestAdapter_EvaluateOr(t *testing.T) {
	filter := sift.Eq("role", "admin").Or(sift.Eq("role", "moderator"))

	adapter := NewAdapter()
	err := sift.Thru(context.Background(), adapter, sift.WithFilter(filter))
	if err != nil {
		t.Fatalf("EvaluateOr() error = %v", err)
	}

	want := `(role == "admin") || (role == "moderator")`
	if got := adapter.Expression(); got != want {
		t.Errorf("EvaluateOr() = %v, want %v", got, want)
	}
}

func TestAdapter_EvaluateNot(t *testing.T) {
	filter := sift.Eq("deleted", true).Not()

	adapter := NewAdapter()
	err := sift.Thru(context.Background(), adapter, sift.WithFilter(filter))
	if err != nil {
		t.Fatalf("EvaluateNot() error = %v", err)
	}

	want := "!(deleted == true)"
	if got := adapter.Expression(); got != want {
		t.Errorf("EvaluateNot() = %v, want %v", got, want)
	}
}

func TestAdapter_ComplexExpression(t *testing.T) {
	// (status = "active" AND age > 18) OR (role = "admin")
	filter := sift.Eq("status", "active").And(sift.Gt("age", 18)).Or(sift.Eq("role", "admin"))

	adapter := NewAdapter()
	err := sift.Thru(context.Background(), adapter, sift.WithFilter(filter))
	if err != nil {
		t.Fatalf("ComplexExpression() error = %v", err)
	}

	want := `((status == "active") && (age > 18)) || (role == "admin")`
	if got := adapter.Expression(); got != want {
		t.Errorf("ComplexExpression() = %v, want %v", got, want)
	}
}

func TestAdapter_NestedLogic(t *testing.T) {
	// NOT((verified = true AND age > 18) OR role = "guest")
	filter := sift.Eq("verified", true).And(sift.Gt("age", 18)).Or(sift.Eq("role", "guest")).Not()

	adapter := NewAdapter()
	err := sift.Thru(context.Background(), adapter, sift.WithFilter(filter))
	if err != nil {
		t.Fatalf("NestedLogic() error = %v", err)
	}

	want := `!(((verified == true) && (age > 18)) || (role == "guest"))`
	if got := adapter.Expression(); got != want {
		t.Errorf("NestedLogic() = %v, want %v", got, want)
	}
}

func TestAdapter_CustomExpression(t *testing.T) {
	tests := []struct {
		name string
		expr sift.Expression
		want string
	}{
		{
			name: "raw expression",
			expr: RawExpression("len(tweets) > 10"),
			want: "len(tweets) > 10",
		},
		{
			name: "predicate",
			expr: Predicate("len(.Content) > 240"),
			want: "len(.Content) > 240",
		},
		{
			name: "array function",
			expr: ArrayFunction("filter", "tweets", "len(.Content) > 240"),
			want: "filter(tweets, len(.Content) > 240)",
		},
		{
			name: "string function",
			expr: StringFunction("upper", "name"),
			want: "upper(name)",
		},
		{
			name: "date function",
			expr: DateFunction("now"),
			want: "now()",
		},
		{
			name: "date function with args",
			expr: DateFunction("date", `"2023-08-14"`),
			want: `date("2023-08-14")`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter()
			err := sift.Thru(context.Background(), adapter, sift.WithFilter(tt.expr))

			if err != nil {
				t.Errorf("CustomExpression() error = %v", err)
				return
			}

			if got := adapter.Expression(); got != tt.want {
				t.Errorf("CustomExpression() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdapter_MixedCustomAndStandard(t *testing.T) {
	// (status = "active" AND len(tweets) > 10) OR custom predicate
	filter := sift.Eq("status", "active").
		And(RawExpression("len(tweets) > 10")).
		Or(Predicate("all(.Comments, len(.Content) < 100)"))

	adapter := NewAdapter()
	err := sift.Thru(context.Background(), adapter, sift.WithFilter(filter))
	if err != nil {
		t.Fatalf("MixedCustomAndStandard() error = %v", err)
	}

	want := `((status == "active") && (len(tweets) > 10)) || (all(.Comments, len(.Content) < 100))`
	if got := adapter.Expression(); got != want {
		t.Errorf("MixedCustomAndStandard() = %v, want %v", got, want)
	}
}

func TestCustomExpression_SiftSerialization(t *testing.T) {
	tests := []struct {
		name     string
		expr     sift.Expression
		wantSift string // Sift serialization format
		wantExpr string // Expr-lang output
	}{
		{
			name:     "raw expression",
			expr:     RawExpression("len(tweets) > 10"),
			wantSift: "exprlang(len(tweets) > 10)",
			wantExpr: "len(tweets) > 10",
		},
		{
			name:     "array function",
			expr:     ArrayFunction("filter", "tweets", "len(.Content) > 240"),
			wantSift: "exprlang(filter(tweets, len(.Content) > 240))",
			wantExpr: "filter(tweets, len(.Content) > 240)",
		},
		{
			name: "mixed with standard operations",
			expr: &sift.AndOperation{
				Left: &sift.Condition{
					Name:      "status",
					Operation: sift.OperationEQ,
					Value:     "active",
				},
				Right: RawExpression("len(tweets) > 10"),
			},
			wantSift: "and(eq(status,active),exprlang(len(tweets) > 10))",
			wantExpr: `(status == "active") && (len(tweets) > 10)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Sift serialization
			if got := tt.expr.String(); got != tt.wantSift {
				t.Errorf("Sift serialization = %v, want %v", got, tt.wantSift)
			}

			// Test expr-lang output
			adapter := NewAdapter()
			err := sift.Thru(context.Background(), adapter, sift.WithFilter(tt.expr))
			if err != nil {
				t.Fatalf("Failed to translate: %v", err)
			}
			if got := adapter.Expression(); got != tt.wantExpr {
				t.Errorf("Expr-lang output = %v, want %v", got, tt.wantExpr)
			}
		})
	}
}
