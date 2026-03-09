package sift

import (
	"context"
	"errors"
	"testing"
)

// Mock adapter for testing
type mockAdapter struct {
	evaluator         *Evaluator
	conditionCalled   bool
	andCalled         bool
	orCalled          bool
	notCalled         bool
	customCalled      bool
	lastCondition     *Condition
	lastAnd           *AndOperation
	lastOr            *OrOperation
	lastNot           *NotOperation
	lastCustom        CustomExpression
	shouldReturnError bool
}

func newMockAdapter() *mockAdapter {
	m := &mockAdapter{}
	m.evaluator = &Evaluator{
		ConditionEvaluator: m,
		AndEvaluator:       m,
		OrEvaluator:        m,
		NotEvaluator:       m,
		CustomEvaluator:    m,
	}
	return m
}

func (m *mockAdapter) Evaluator(ctx context.Context) *Evaluator {
	return m.evaluator
}

func (m *mockAdapter) EvaluateCondition(ctx context.Context, node *Condition) error {
	m.conditionCalled = true
	m.lastCondition = node
	if m.shouldReturnError {
		return errors.New("mock error")
	}
	return nil
}

func (m *mockAdapter) EvaluateAnd(ctx context.Context, node *AndOperation) error {
	m.andCalled = true
	m.lastAnd = node
	if m.shouldReturnError {
		return errors.New("mock error")
	}
	return nil
}

func (m *mockAdapter) EvaluateOr(ctx context.Context, node *OrOperation) error {
	m.orCalled = true
	m.lastOr = node
	if m.shouldReturnError {
		return errors.New("mock error")
	}
	return nil
}

func (m *mockAdapter) EvaluateNot(ctx context.Context, node *NotOperation) error {
	m.notCalled = true
	m.lastNot = node
	if m.shouldReturnError {
		return errors.New("mock error")
	}
	return nil
}

func (m *mockAdapter) EvaluateCustom(ctx context.Context, node CustomExpression) error {
	m.customCalled = true
	m.lastCustom = node
	if m.shouldReturnError {
		return errors.New("mock error")
	}
	return nil
}

// TestThru tests the main entry point
func TestThru(t *testing.T) {
	ctx := context.Background()

	t.Run("nil adapter", func(t *testing.T) {
		filter := &Condition{
			Name:      "test",
			Operation: OperationEQ,
			Value:     "value",
		}
		err := Thru(ctx, nil, WithFilter(filter))
		if err == nil {
			t.Error("Expected error for nil adapter")
		}
	})

	t.Run("condition evaluation", func(t *testing.T) {
		adapter := newMockAdapter()
		filter := &Condition{
			Name:      "status",
			Operation: OperationEQ,
			Value:     "active",
		}
		err := Thru(ctx, adapter, WithFilter(filter))
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !adapter.conditionCalled {
			t.Error("EvaluateCondition was not called")
		}
		if adapter.lastCondition != filter {
			t.Error("Wrong condition passed to evaluator")
		}
	})

	t.Run("and operation", func(t *testing.T) {
		adapter := newMockAdapter()
		filter := &AndOperation{
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
		}
		err := Thru(ctx, adapter, WithFilter(filter))
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !adapter.andCalled {
			t.Error("EvaluateAnd was not called")
		}
	})

	t.Run("or operation", func(t *testing.T) {
		adapter := newMockAdapter()
		filter := &OrOperation{
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
		}
		err := Thru(ctx, adapter, WithFilter(filter))
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !adapter.orCalled {
			t.Error("EvaluateOr was not called")
		}
	})

	t.Run("not operation", func(t *testing.T) {
		adapter := newMockAdapter()
		filter := &NotOperation{
			Child: &Condition{
				Name:      "deleted",
				Operation: OperationEQ,
				Value:     "true",
			},
		}
		err := Thru(ctx, adapter, WithFilter(filter))
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !adapter.notCalled {
			t.Error("EvaluateNot was not called")
		}
	})
}

// TestOperationMethods tests Operation type methods
func TestOperationMethods(t *testing.T) {
	t.Run("IsCondition", func(t *testing.T) {
		conditionOps := []Operation{
			OperationEQ, OperationNEQ, OperationLT, OperationLTE,
			OperationGT, OperationGTE, OperationContains, OperationBeginsWith,
			OperationIn, OperationBetween,
		}
		for _, op := range conditionOps {
			if !op.IsCondition() {
				t.Errorf("Operation %s should be a condition operation", op)
			}
		}

		nonConditionOps := []Operation{
			OperationExists, OperationNotExists,
		}
		for _, op := range nonConditionOps {
			if op.IsCondition() {
				t.Errorf("Operation %s should not be a condition operation", op)
			}
		}
	})

	t.Run("IsExistence", func(t *testing.T) {
		existenceOps := []Operation{
			OperationExists, OperationNotExists,
		}
		for _, op := range existenceOps {
			if !op.IsExistence() {
				t.Errorf("Operation %s should be an existence operation", op)
			}
		}

		nonExistenceOps := []Operation{
			OperationEQ, OperationNEQ, OperationLT, OperationLTE,
			OperationGT, OperationGTE, OperationContains, OperationBeginsWith,
			OperationIn, OperationBetween,
		}
		for _, op := range nonExistenceOps {
			if op.IsExistence() {
				t.Errorf("Operation %s should not be an existence operation", op)
			}
		}
	})
}

// TestCondition tests Condition struct
func TestCondition(t *testing.T) {
	t.Run("basic condition", func(t *testing.T) {
		cond := &Condition{
			Name:      "age",
			Operation: OperationGT,
			Value:     "18",
		}
		if cond.Name != "age" {
			t.Errorf("Expected Name 'age', got '%s'", cond.Name)
		}
		if cond.Operation != OperationGT {
			t.Errorf("Expected Operation OperationGT, got '%s'", cond.Operation)
		}
		if cond.Value != "18" {
			t.Errorf("Expected Value '18', got '%s'", cond.Value)
		}
	})

	t.Run("unsupported evaluator", func(t *testing.T) {
		adapter := newMockAdapter()
		adapter.evaluator.ConditionEvaluator = nil // Remove support
		cond := &Condition{
			Name:      "test",
			Operation: OperationEQ,
			Value:     "value",
		}
		err := Thru(context.Background(), adapter, WithFilter(cond))
		if err == nil {
			t.Error("Expected error for unsupported evaluator")
		}
		if !errors.Is(err, ErrUnsupported) {
			t.Errorf("Expected ErrUnsupported, got %v", err)
		}
	})
}

// TestAndOperation tests AndOperation struct
func TestAndOperation(t *testing.T) {
	t.Run("basic and", func(t *testing.T) {
		left := &Condition{Name: "a", Operation: OperationEQ, Value: "1"}
		right := &Condition{Name: "b", Operation: OperationEQ, Value: "2"}
		and := &AndOperation{Left: left, Right: right}

		if and.Left != left {
			t.Error("Left operand not set correctly")
		}
		if and.Right != right {
			t.Error("Right operand not set correctly")
		}
	})

	t.Run("unsupported evaluator", func(t *testing.T) {
		adapter := newMockAdapter()
		adapter.evaluator.AndEvaluator = nil
		and := &AndOperation{
			Left:  &Condition{Name: "a", Operation: OperationEQ, Value: "1"},
			Right: &Condition{Name: "b", Operation: OperationEQ, Value: "2"},
		}
		err := Thru(context.Background(), adapter, WithFilter(and))
		if err == nil {
			t.Error("Expected error for unsupported evaluator")
		}
		if !errors.Is(err, ErrUnsupported) {
			t.Errorf("Expected ErrUnsupported, got %v", err)
		}
	})
}

// TestOrOperation tests OrOperation struct
func TestOrOperation(t *testing.T) {
	t.Run("basic or", func(t *testing.T) {
		left := &Condition{Name: "a", Operation: OperationEQ, Value: "1"}
		right := &Condition{Name: "b", Operation: OperationEQ, Value: "2"}
		or := &OrOperation{Left: left, Right: right}

		if or.Left != left {
			t.Error("Left operand not set correctly")
		}
		if or.Right != right {
			t.Error("Right operand not set correctly")
		}
	})

	t.Run("unsupported evaluator", func(t *testing.T) {
		adapter := newMockAdapter()
		adapter.evaluator.OrEvaluator = nil
		or := &OrOperation{
			Left:  &Condition{Name: "a", Operation: OperationEQ, Value: "1"},
			Right: &Condition{Name: "b", Operation: OperationEQ, Value: "2"},
		}
		err := Thru(context.Background(), adapter, WithFilter(or))
		if err == nil {
			t.Error("Expected error for unsupported evaluator")
		}
		if !errors.Is(err, ErrUnsupported) {
			t.Errorf("Expected ErrUnsupported, got %v", err)
		}
	})
}

// TestNotOperation tests NotOperation struct
func TestNotOperation(t *testing.T) {
	t.Run("basic not", func(t *testing.T) {
		child := &Condition{Name: "deleted", Operation: OperationEQ, Value: "true"}
		not := &NotOperation{Child: child}

		if not.Child != child {
			t.Error("Child not set correctly")
		}
	})

	t.Run("unsupported evaluator", func(t *testing.T) {
		adapter := newMockAdapter()
		adapter.evaluator.NotEvaluator = nil
		not := &NotOperation{
			Child: &Condition{Name: "a", Operation: OperationEQ, Value: "1"},
		}
		err := Thru(context.Background(), adapter, WithFilter(not))
		if err == nil {
			t.Error("Expected error for unsupported evaluator")
		}
		if !errors.Is(err, ErrUnsupported) {
			t.Errorf("Expected ErrUnsupported, got %v", err)
		}
	})
}

// testCustom is a test implementation of CustomExpression
type testCustom struct {
	field string
}

func (tc *testCustom) Type() string {
	return "test_custom"
}

func (tc *testCustom) String() string {
	return "test_custom(" + tc.field + ")"
}

// TestCustomExpression tests custom expression functionality
func TestCustomExpression(t *testing.T) {
	t.Run("basic custom", func(t *testing.T) {
		custom := &testCustom{field: "value"}
		expr := NewCustomExpression(custom)

		adapter := newMockAdapter()
		err := Thru(context.Background(), adapter, WithFilter(expr))
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !adapter.customCalled {
			t.Error("EvaluateCustom was not called")
		}
		if adapter.lastCustom != custom {
			t.Error("Wrong custom expression passed to evaluator")
		}
	})

	t.Run("unsupported evaluator", func(t *testing.T) {
		adapter := newMockAdapter()
		adapter.evaluator.CustomEvaluator = nil
		custom := &testCustom{field: "value"}
		expr := NewCustomExpression(custom)

		err := Thru(context.Background(), adapter, WithFilter(expr))
		if err == nil {
			t.Error("Expected error for unsupported evaluator")
		}
		if !errors.Is(err, ErrUnsupported) {
			t.Errorf("Expected ErrUnsupported, got %v", err)
		}
	})
}

// TestErrorFunctions tests error helper functions
func TestErrorFunctions(t *testing.T) {
	t.Run("ErrorOperationNotSupported", func(t *testing.T) {
		err := ErrorOperationNotSupported("custom_op")
		if err == nil {
			t.Error("Expected error")
		}
		if !errors.Is(err, ErrUnsupported) {
			t.Error("Error should wrap ErrUnsupported")
		}
		errMsg := err.Error()
		if errMsg == "" {
			t.Error("Error message should not be empty")
		}
	})

	t.Run("ErrorNodeNotSupported", func(t *testing.T) {
		cond := &Condition{Name: "test", Operation: OperationEQ, Value: "value"}
		err := ErrorNodeNotSupported(cond)
		if err == nil {
			t.Error("Expected error")
		}
		if !errors.Is(err, ErrUnsupported) {
			t.Error("Error should wrap ErrUnsupported")
		}
		errMsg := err.Error()
		if errMsg == "" {
			t.Error("Error message should not be empty")
		}
	})
}

// TestComplexExpressions tests complex nested expressions
func TestComplexExpressions(t *testing.T) {
	t.Run("nested and/or", func(t *testing.T) {
		// (a = 1 OR b = 2) AND (c = 3 OR d = 4)
		filter := &AndOperation{
			Left: &OrOperation{
				Left:  &Condition{Name: "a", Operation: OperationEQ, Value: "1"},
				Right: &Condition{Name: "b", Operation: OperationEQ, Value: "2"},
			},
			Right: &OrOperation{
				Left:  &Condition{Name: "c", Operation: OperationEQ, Value: "3"},
				Right: &Condition{Name: "d", Operation: OperationEQ, Value: "4"},
			},
		}

		adapter := newMockAdapter()
		err := Thru(context.Background(), adapter, WithFilter(filter))
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !adapter.andCalled {
			t.Error("EvaluateAnd should have been called")
		}
	})

	t.Run("not with nested and", func(t *testing.T) {
		// NOT (a = 1 AND b = 2)
		filter := &NotOperation{
			Child: &AndOperation{
				Left:  &Condition{Name: "a", Operation: OperationEQ, Value: "1"},
				Right: &Condition{Name: "b", Operation: OperationEQ, Value: "2"},
			},
		}

		adapter := newMockAdapter()
		err := Thru(context.Background(), adapter, WithFilter(filter))
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !adapter.notCalled {
			t.Error("EvaluateNot should have been called")
		}
	})
}

// TestOperationConstants tests that all operation constants are defined
func TestOperationConstants(t *testing.T) {
	operations := []Operation{
		OperationEQ,
		OperationNEQ,
		OperationLT,
		OperationLTE,
		OperationGT,
		OperationGTE,
		OperationContains,
		OperationBeginsWith,
		OperationIn,
		OperationExists,
		OperationNotExists,
		OperationBetween,
	}

	for _, op := range operations {
		if string(op) == "" {
			t.Errorf("Operation constant is empty")
		}
	}
}

// TestConditionConvenienceMethods tests the And, Or, Not methods on Condition
func TestConditionConvenienceMethods(t *testing.T) {
	t.Run("Condition.And", func(t *testing.T) {
		left := Condition{Name: "a", Operation: OperationEQ, Value: "1"}
		right := &Condition{Name: "b", Operation: OperationEQ, Value: "2"}

		result := left.And(right)

		and, ok := result.expr.(*AndOperation)
		if !ok {
			t.Fatalf("Expected AndOperation, got %T", result.expr)
		}
		if and.Right != right {
			t.Error("Right operand not set correctly")
		}
	})

	t.Run("Condition.Or", func(t *testing.T) {
		left := Condition{Name: "a", Operation: OperationEQ, Value: "1"}
		right := &Condition{Name: "b", Operation: OperationEQ, Value: "2"}

		result := left.Or(right)

		or, ok := result.expr.(*OrOperation)
		if !ok {
			t.Fatalf("Expected OrOperation, got %T", result.expr)
		}
		if or.Right != right {
			t.Error("Right operand not set correctly")
		}
	})

	t.Run("Condition.Not", func(t *testing.T) {
		cond := Condition{Name: "deleted", Operation: OperationEQ, Value: "true"}

		result := cond.Not()

		not, ok := result.expr.(*NotOperation)
		if !ok {
			t.Fatalf("Expected NotOperation, got %T", result.expr)
		}

		// Child should be a builder wrapping the condition
		_, ok = not.Child.(ExpressionBuilder)
		if !ok {
			t.Fatalf("Expected ExpressionBuilder as child, got %T", not.Child)
		}
	})
}
