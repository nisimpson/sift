package sift

import (
	"context"
	"testing"
)

// TestNewExpressionBuilder tests the NewExpressionBuilder function
func TestNewExpressionBuilder(t *testing.T) {
	cond := &Condition{Name: "status", Operation: OperationEQ, Value: "active"}
	builder := NewExpressionBuilder(cond)

	if builder.expr != cond {
		t.Error("Builder expression not set correctly")
	}
}

// TestExpressionBuilderAnd tests the And method
func TestExpressionBuilderAnd(t *testing.T) {
	t.Run("simple and", func(t *testing.T) {
		left := &Condition{Name: "status", Operation: OperationEQ, Value: "active"}
		right := &Condition{Name: "age", Operation: OperationGT, Value: "18"}

		builder := NewExpressionBuilder(left).And(right)

		and, ok := builder.expr.(*AndOperation)
		if !ok {
			t.Fatalf("Expected AndOperation, got %T", builder.expr)
		}
		if and.Right != right {
			t.Error("Right operand not set correctly")
		}
	})

	t.Run("chained and", func(t *testing.T) {
		a := &Condition{Name: "a", Operation: OperationEQ, Value: "1"}
		b := &Condition{Name: "b", Operation: OperationEQ, Value: "2"}
		c := &Condition{Name: "c", Operation: OperationEQ, Value: "3"}

		builder := NewExpressionBuilder(a).And(b).And(c)

		// Verify it's an AndOperation
		_, ok := builder.expr.(*AndOperation)
		if !ok {
			t.Fatalf("Expected AndOperation, got %T", builder.expr)
		}
	})
}

// TestExpressionBuilderOr tests the Or method
func TestExpressionBuilderOr(t *testing.T) {
	t.Run("simple or", func(t *testing.T) {
		left := &Condition{Name: "role", Operation: OperationEQ, Value: "admin"}
		right := &Condition{Name: "role", Operation: OperationEQ, Value: "moderator"}

		builder := NewExpressionBuilder(left).Or(right)

		or, ok := builder.expr.(*OrOperation)
		if !ok {
			t.Fatalf("Expected OrOperation, got %T", builder.expr)
		}
		if or.Right != right {
			t.Error("Right operand not set correctly")
		}
	})

	t.Run("chained or", func(t *testing.T) {
		a := &Condition{Name: "role", Operation: OperationEQ, Value: "admin"}
		b := &Condition{Name: "role", Operation: OperationEQ, Value: "moderator"}
		c := &Condition{Name: "role", Operation: OperationEQ, Value: "user"}

		builder := NewExpressionBuilder(a).Or(b).Or(c)

		// Verify it's an OrOperation
		_, ok := builder.expr.(*OrOperation)
		if !ok {
			t.Fatalf("Expected OrOperation, got %T", builder.expr)
		}
	})
}

// TestExpressionBuilderNot tests the Not method
func TestExpressionBuilderNot(t *testing.T) {
	t.Run("simple not", func(t *testing.T) {
		child := &Condition{Name: "deleted", Operation: OperationEQ, Value: "true"}

		builder := NewExpressionBuilder(child).Not()

		not, ok := builder.expr.(*NotOperation)
		if !ok {
			t.Fatalf("Expected NotOperation, got %T", builder.expr)
		}

		// Child should be a builder
		_, ok = not.Child.(ExpressionBuilder)
		if !ok {
			t.Fatalf("Expected ExpressionBuilder as child, got %T", not.Child)
		}
	})

	t.Run("double not", func(t *testing.T) {
		child := &Condition{Name: "active", Operation: OperationEQ, Value: "true"}

		builder := NewExpressionBuilder(child).Not().Not()

		// Should be: not(not(child))
		outer, ok := builder.expr.(*NotOperation)
		if !ok {
			t.Fatalf("Expected outer NotOperation, got %T", builder.expr)
		}

		// Inner child should be a builder containing a NotOperation
		innerBuilder, ok := outer.Child.(ExpressionBuilder)
		if !ok {
			t.Fatalf("Expected ExpressionBuilder, got %T", outer.Child)
		}

		_, ok = innerBuilder.expr.(*NotOperation)
		if !ok {
			t.Fatalf("Expected inner NotOperation, got %T", innerBuilder.expr)
		}
	})
}

// TestExpressionBuilderAccept tests that builder implements Expression interface
func TestExpressionBuilderAccept(t *testing.T) {
	cond := &Condition{Name: "status", Operation: OperationEQ, Value: "active"}
	builder := NewExpressionBuilder(cond)

	// Create a mock adapter
	adapter := newMockAdapter()

	// Builder should be usable as an Expression
	err := Thru(context.Background(), adapter, WithFilter(builder))
	if err != nil {
		t.Errorf("Thru() with builder error = %v", err)
	}

	if !adapter.conditionCalled {
		t.Error("EvaluateCondition should have been called")
	}
}

// TestExpressionBuilderComplexExpressions tests complex builder combinations
func TestExpressionBuilderComplexExpressions(t *testing.T) {
	t.Run("(a AND b) OR c", func(t *testing.T) {
		a := &Condition{Name: "a", Operation: OperationEQ, Value: "1"}
		b := &Condition{Name: "b", Operation: OperationEQ, Value: "2"}
		c := &Condition{Name: "c", Operation: OperationEQ, Value: "3"}

		builder := NewExpressionBuilder(a).And(b).Or(c)

		or, ok := builder.expr.(*OrOperation)
		if !ok {
			t.Fatalf("Expected OrOperation, got %T", builder.expr)
		}

		// Left should be a builder containing an AndOperation
		leftBuilder, ok := or.Left.(ExpressionBuilder)
		if !ok {
			t.Fatalf("Expected ExpressionBuilder on left, got %T", or.Left)
		}

		_, ok = leftBuilder.expr.(*AndOperation)
		if !ok {
			t.Fatalf("Expected AndOperation, got %T", leftBuilder.expr)
		}

		if or.Right != c {
			t.Error("OR right operand not correct")
		}
	})

	t.Run("NOT (a OR b)", func(t *testing.T) {
		a := &Condition{Name: "a", Operation: OperationEQ, Value: "1"}
		b := &Condition{Name: "b", Operation: OperationEQ, Value: "2"}

		builder := NewExpressionBuilder(a).Or(b).Not()

		not, ok := builder.expr.(*NotOperation)
		if !ok {
			t.Fatalf("Expected NotOperation, got %T", builder.expr)
		}

		// Child should be a builder containing an OrOperation
		childBuilder, ok := not.Child.(ExpressionBuilder)
		if !ok {
			t.Fatalf("Expected ExpressionBuilder as child, got %T", not.Child)
		}

		_, ok = childBuilder.expr.(*OrOperation)
		if !ok {
			t.Fatalf("Expected OrOperation, got %T", childBuilder.expr)
		}
	})

	t.Run("(a OR b) AND (c OR d)", func(t *testing.T) {
		a := &Condition{Name: "a", Operation: OperationEQ, Value: "1"}
		b := &Condition{Name: "b", Operation: OperationEQ, Value: "2"}
		c := &Condition{Name: "c", Operation: OperationEQ, Value: "3"}
		d := &Condition{Name: "d", Operation: OperationEQ, Value: "4"}

		leftBuilder := NewExpressionBuilder(a).Or(b)
		rightBuilder := NewExpressionBuilder(c).Or(d)
		builder := leftBuilder.And(rightBuilder)

		and, ok := builder.expr.(*AndOperation)
		if !ok {
			t.Fatalf("Expected AndOperation, got %T", builder.expr)
		}

		// Both sides should be builders
		_, ok = and.Left.(ExpressionBuilder)
		if !ok {
			t.Fatalf("Expected ExpressionBuilder on left, got %T", and.Left)
		}

		if and.Right != rightBuilder {
			t.Error("Right operand not correct")
		}
	})
}

// TestExpressionBuilderWithParse tests using builder with parsed expressions
func TestExpressionBuilderWithParse(t *testing.T) {
	// Parse an existing filter
	parsed, err := Parse("eq(status,active)", nil)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Extend it with builder
	newCond := &Condition{Name: "age", Operation: OperationGT, Value: "18"}
	builder := NewExpressionBuilder(parsed).And(newCond)

	// Verify it's an AndOperation
	and, ok := builder.expr.(*AndOperation)
	if !ok {
		t.Fatalf("Expected AndOperation, got %T", builder.expr)
	}

	if and.Right != newCond {
		t.Error("Right operand not correct")
	}

	// Format and check
	formatted, err := Format(builder, nil)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	expected := "and(eq(status,active),gt(age,18))"
	if formatted != expected {
		t.Errorf("Format() = %v, want %v", formatted, expected)
	}
}

// TestExpressionBuilderWithThru tests using builder with Thru
func TestExpressionBuilderWithThru(t *testing.T) {
	a := &Condition{Name: "status", Operation: OperationEQ, Value: "active"}
	b := &Condition{Name: "age", Operation: OperationGT, Value: "18"}

	builder := NewExpressionBuilder(a).And(b)

	adapter := newMockAdapter()
	err := Thru(context.Background(), adapter, WithFilter(builder))
	if err != nil {
		t.Errorf("Thru() error = %v", err)
	}

	if !adapter.andCalled {
		t.Error("EvaluateAnd should have been called")
	}
}

// TestExpressionBuilderString tests the String method
func TestExpressionBuilderString(t *testing.T) {
	t.Run("simple condition", func(t *testing.T) {
		cond := &Condition{Name: "status", Operation: OperationEQ, Value: "active"}
		builder := NewExpressionBuilder(cond)

		str := builder.String()
		expected := "eq(status,active)"
		if str != expected {
			t.Errorf("String() = %v, want %v", str, expected)
		}
	})

	t.Run("and operation", func(t *testing.T) {
		a := &Condition{Name: "a", Operation: OperationEQ, Value: "1"}
		b := &Condition{Name: "b", Operation: OperationEQ, Value: "2"}
		builder := NewExpressionBuilder(a).And(b)

		str := builder.String()
		expected := "and(eq(a,1),eq(b,2))"
		if str != expected {
			t.Errorf("String() = %v, want %v", str, expected)
		}
	})

	t.Run("complex expression", func(t *testing.T) {
		a := &Condition{Name: "a", Operation: OperationEQ, Value: "1"}
		b := &Condition{Name: "b", Operation: OperationEQ, Value: "2"}
		c := &Condition{Name: "c", Operation: OperationEQ, Value: "3"}
		builder := NewExpressionBuilder(a).And(b).Or(c)

		str := builder.String()
		expected := "or(and(eq(a,1),eq(b,2)),eq(c,3))"
		if str != expected {
			t.Errorf("String() = %v, want %v", str, expected)
		}
	})
}

// TestBuilderConvenienceFunctions tests the convenience builder functions
func TestBuilderConvenienceFunctions(t *testing.T) {
	t.Run("Eq", func(t *testing.T) {
		builder := Eq("status", "active")
		cond, ok := builder.expr.(*Condition)
		if !ok {
			t.Fatalf("Expected Condition, got %T", builder.expr)
		}
		if cond.Name != "status" || cond.Operation != OperationEQ || cond.Value != "active" {
			t.Errorf("Eq() created incorrect condition: %+v", cond)
		}
	})

	t.Run("Neq", func(t *testing.T) {
		builder := Neq("status", "deleted")
		cond, ok := builder.expr.(*Condition)
		if !ok {
			t.Fatalf("Expected Condition, got %T", builder.expr)
		}
		if cond.Operation != OperationNEQ {
			t.Errorf("Neq() operation = %v, want %v", cond.Operation, OperationNEQ)
		}
	})

	t.Run("Lt", func(t *testing.T) {
		builder := Lt("age", 18)
		cond, ok := builder.expr.(*Condition)
		if !ok {
			t.Fatalf("Expected Condition, got %T", builder.expr)
		}
		if cond.Operation != OperationLT || cond.Value != "18" {
			t.Errorf("Lt() created incorrect condition: %+v", cond)
		}
	})

	t.Run("Lte", func(t *testing.T) {
		builder := Lte("age", 18)
		cond, ok := builder.expr.(*Condition)
		if !ok {
			t.Fatalf("Expected Condition, got %T", builder.expr)
		}
		if cond.Operation != OperationLTE {
			t.Errorf("Lte() operation = %v, want %v", cond.Operation, OperationLTE)
		}
	})

	t.Run("Gt", func(t *testing.T) {
		builder := Gt("age", 18)
		cond, ok := builder.expr.(*Condition)
		if !ok {
			t.Fatalf("Expected Condition, got %T", builder.expr)
		}
		if cond.Operation != OperationGT {
			t.Errorf("Gt() operation = %v, want %v", cond.Operation, OperationGT)
		}
	})

	t.Run("Gte", func(t *testing.T) {
		builder := Gte("age", 18)
		cond, ok := builder.expr.(*Condition)
		if !ok {
			t.Fatalf("Expected Condition, got %T", builder.expr)
		}
		if cond.Operation != OperationGTE {
			t.Errorf("Gte() operation = %v, want %v", cond.Operation, OperationGTE)
		}
	})

	t.Run("Contains", func(t *testing.T) {
		builder := Contains("email", "@example.com")
		cond, ok := builder.expr.(*Condition)
		if !ok {
			t.Fatalf("Expected Condition, got %T", builder.expr)
		}
		if cond.Operation != OperationContains {
			t.Errorf("Contains() operation = %v, want %v", cond.Operation, OperationContains)
		}
	})

	t.Run("In", func(t *testing.T) {
		builder := In("status", "active")
		cond, ok := builder.expr.(*Condition)
		if !ok {
			t.Fatalf("Expected Condition, got %T", builder.expr)
		}
		if cond.Operation != OperationIn {
			t.Errorf("In() operation = %v, want %v", cond.Operation, OperationIn)
		}
	})

	t.Run("Between", func(t *testing.T) {
		builder := Between("age", 18, 65)
		cond, ok := builder.expr.(*Condition)
		if !ok {
			t.Fatalf("Expected Condition, got %T", builder.expr)
		}
		if cond.Operation != OperationBetween || cond.Value != "18,65" {
			t.Errorf("Between() created incorrect condition: %+v", cond)
		}
	})

	t.Run("Exists", func(t *testing.T) {
		builder := Exists("email")
		cond, ok := builder.expr.(*Condition)
		if !ok {
			t.Fatalf("Expected Condition, got %T", builder.expr)
		}
		if cond.Operation != OperationExists || cond.Name != "email" {
			t.Errorf("Exists() created incorrect condition: %+v", cond)
		}
	})

	t.Run("NotExists", func(t *testing.T) {
		builder := NotExists("deleted_at")
		cond, ok := builder.expr.(*Condition)
		if !ok {
			t.Fatalf("Expected Condition, got %T", builder.expr)
		}
		if cond.Operation != OperationNotExists || cond.Name != "deleted_at" {
			t.Errorf("NotExists() created incorrect condition: %+v", cond)
		}
	})
}

// TestBuilderConvenienceFunctionsChaining tests chaining with convenience functions
func TestBuilderConvenienceFunctionsChaining(t *testing.T) {
	t.Run("Eq().And()", func(t *testing.T) {
		builder := Eq("status", "active").And(Gt("age", 18))

		and, ok := builder.expr.(*AndOperation)
		if !ok {
			t.Fatalf("Expected AndOperation, got %T", builder.expr)
		}

		// Verify left side
		leftBuilder, ok := and.Left.(ExpressionBuilder)
		if !ok {
			t.Fatalf("Expected ExpressionBuilder on left, got %T", and.Left)
		}
		leftCond, ok := leftBuilder.expr.(*Condition)
		if !ok || leftCond.Operation != OperationEQ {
			t.Error("Left side not correct")
		}

		// Verify right side
		rightBuilder, ok := and.Right.(ExpressionBuilder)
		if !ok {
			t.Fatalf("Expected ExpressionBuilder on right, got %T", and.Right)
		}
		rightCond, ok := rightBuilder.expr.(*Condition)
		if !ok || rightCond.Operation != OperationGT {
			t.Error("Right side not correct")
		}
	})

	t.Run("complex chain", func(t *testing.T) {
		// (status = "active" AND age > 18) OR role = "admin"
		builder := Eq("status", "active").
			And(Gt("age", 18)).
			Or(Eq("role", "admin"))

		or, ok := builder.expr.(*OrOperation)
		if !ok {
			t.Fatalf("Expected OrOperation, got %T", builder.expr)
		}

		// Left should be an AND
		leftBuilder, ok := or.Left.(ExpressionBuilder)
		if !ok {
			t.Fatalf("Expected ExpressionBuilder on left, got %T", or.Left)
		}
		_, ok = leftBuilder.expr.(*AndOperation)
		if !ok {
			t.Error("Left side should be AndOperation")
		}

		// Right should be Eq
		rightBuilder, ok := or.Right.(ExpressionBuilder)
		if !ok {
			t.Fatalf("Expected ExpressionBuilder on right, got %T", or.Right)
		}
		rightCond, ok := rightBuilder.expr.(*Condition)
		if !ok || rightCond.Operation != OperationEQ {
			t.Error("Right side not correct")
		}
	})

	t.Run("with Not()", func(t *testing.T) {
		builder := Eq("deleted", "true").Not()

		not, ok := builder.expr.(*NotOperation)
		if !ok {
			t.Fatalf("Expected NotOperation, got %T", builder.expr)
		}

		childBuilder, ok := not.Child.(ExpressionBuilder)
		if !ok {
			t.Fatalf("Expected ExpressionBuilder as child, got %T", not.Child)
		}

		cond, ok := childBuilder.expr.(*Condition)
		if !ok || cond.Operation != OperationEQ {
			t.Error("Child not correct")
		}
	})
}

// TestBuilderConvenienceFunctionsWithThru tests using convenience functions with Thru
func TestBuilderConvenienceFunctionsWithThru(t *testing.T) {
	builder := Eq("status", "active").And(Gt("age", 18))

	adapter := newMockAdapter()
	err := Thru(context.Background(), adapter, WithFilter(builder))
	if err != nil {
		t.Errorf("Thru() error = %v", err)
	}

	if !adapter.andCalled {
		t.Error("EvaluateAnd should have been called")
	}
}

// TestBuilderConvenienceFunctionsWithFormat tests using convenience functions with Format
func TestBuilderConvenienceFunctionsWithFormat(t *testing.T) {
	builder := Eq("status", "active").And(Gt("age", 18))

	formatted, err := Format(builder, nil)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	expected := "and(eq(status,active),gt(age,18))"
	if formatted != expected {
		t.Errorf("Format() = %v, want %v", formatted, expected)
	}
}

// Sort Builder Tests

func TestSort(t *testing.T) {
	sort := Sort("created_at", SortDesc)

	if len(sort.fields) != 1 {
		t.Errorf("Sort() fields length = %d, want 1", len(sort.fields))
	}

	field := sort.fields[0]
	if field.Name != "created_at" {
		t.Errorf("Sort() field name = %s, want created_at", field.Name)
	}
	if field.Direction != SortDesc {
		t.Errorf("Sort() direction = %s, want %s", field.Direction, SortDesc)
	}
}

func TestSortBuilder_ThenBy(t *testing.T) {
	sort := Sort("created_at", SortDesc).ThenBy("name", SortAsc)

	if len(sort.fields) != 2 {
		t.Errorf("ThenBy() fields length = %d, want 2", len(sort.fields))
	}

	if sort.fields[0].Name != "created_at" {
		t.Errorf("ThenBy() first field name = %s, want created_at", sort.fields[0].Name)
	}
	if sort.fields[1].Name != "name" {
		t.Errorf("ThenBy() second field name = %s, want name", sort.fields[1].Name)
	}
}

func TestSortBuilder_NullsLast(t *testing.T) {
	sort := Sort("email", SortAsc).NullsLast()

	if len(sort.fields) != 1 {
		t.Fatalf("NullsLast() fields length = %d, want 1", len(sort.fields))
	}

	if !sort.fields[0].NullsLast {
		t.Error("NullsLast() should set NullsLast flag to true")
	}
}

func TestSortBuilder_NullsLastOnMultipleFields(t *testing.T) {
	sort := Sort("created_at", SortDesc).
		ThenBy("email", SortAsc).NullsLast().
		ThenBy("name", SortAsc)

	if len(sort.fields) != 3 {
		t.Fatalf("fields length = %d, want 3", len(sort.fields))
	}

	// Only the email field should have NullsLast set
	if sort.fields[0].NullsLast {
		t.Error("created_at should not have NullsLast set")
	}
	if !sort.fields[1].NullsLast {
		t.Error("email should have NullsLast set")
	}
	if sort.fields[2].NullsLast {
		t.Error("name should not have NullsLast set")
	}
}

func TestSortBuilder_Build(t *testing.T) {
	builder := Sort("created_at", SortDesc).ThenBy("name", SortAsc)

	if len(builder.fields) != 2 {
		t.Errorf("Build() fields length = %d, want 2", len(builder.fields))
	}

	if builder.fields[0].Name != "created_at" {
		t.Errorf("Build() first field name = %s, want created_at", builder.fields[0].Name)
	}
	if builder.fields[1].Name != "name" {
		t.Errorf("Build() second field name = %s, want name", builder.fields[1].Name)
	}
}

func TestSortBuilder_ChainedCalls(t *testing.T) {
	// Test that all methods return SortBuilder for chaining
	sort := Sort("a", SortAsc).
		ThenBy("b", SortDesc).
		NullsLast().
		ThenBy("c", SortAsc)

	if len(sort.fields) != 3 {
		t.Errorf("Chained calls resulted in %d fields, want 3", len(sort.fields))
	}

	// Verify the chain worked correctly
	if sort.fields[0].Name != "a" || sort.fields[0].Direction != SortAsc {
		t.Error("First field incorrect")
	}
	if sort.fields[1].Name != "b" || sort.fields[1].Direction != SortDesc || !sort.fields[1].NullsLast {
		t.Error("Second field incorrect")
	}
	if sort.fields[2].Name != "c" || sort.fields[2].Direction != SortAsc {
		t.Error("Third field incorrect")
	}
}

func TestSortDirection_Constants(t *testing.T) {
	if SortAsc != "asc" {
		t.Errorf("SortAsc = %s, want asc", SortAsc)
	}
	if SortDesc != "desc" {
		t.Errorf("SortDesc = %s, want desc", SortDesc)
	}
}
