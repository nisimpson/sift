package jsonapi_test

import (
	"context"
	"fmt"

	"github.com/nisimpson/sift"
)

// MockBackendAdapter is a comprehensive mock adapter for testing.
// It demonstrates all standard operations plus custom expressions.
type MockBackendAdapter struct {
	expression string
	callCount  int
}

func NewMockBackendAdapter() *MockBackendAdapter {
	return &MockBackendAdapter{}
}

func (m *MockBackendAdapter) Evaluator(ctx context.Context) *sift.Evaluator {
	return &sift.Evaluator{
		ConditionEvaluator: m,
		AndEvaluator:       m,
		OrEvaluator:        m,
		NotEvaluator:       m,
		CustomEvaluator:    m,
	}
}

func (m *MockBackendAdapter) EvaluateCondition(ctx context.Context, node *sift.Condition) error {
	m.callCount++
	switch node.Operation {
	case sift.OperationEQ:
		m.expression = fmt.Sprintf("%s = %s", node.Name, node.Value)
	case sift.OperationNEQ:
		m.expression = fmt.Sprintf("%s != %s", node.Name, node.Value)
	case sift.OperationLT:
		m.expression = fmt.Sprintf("%s < %s", node.Name, node.Value)
	case sift.OperationLTE:
		m.expression = fmt.Sprintf("%s <= %s", node.Name, node.Value)
	case sift.OperationGT:
		m.expression = fmt.Sprintf("%s > %s", node.Name, node.Value)
	case sift.OperationGTE:
		m.expression = fmt.Sprintf("%s >= %s", node.Name, node.Value)
	case sift.OperationContains:
		m.expression = fmt.Sprintf("contains(%s, %s)", node.Name, node.Value)
	case sift.OperationBeginsWith:
		m.expression = fmt.Sprintf("beginsWith(%s, %s)", node.Name, node.Value)
	case sift.OperationIn:
		m.expression = fmt.Sprintf("%s IN %s", node.Name, node.Value)
	case sift.OperationBetween:
		m.expression = fmt.Sprintf("%s BETWEEN %s", node.Name, node.Value)
	case sift.OperationExists:
		m.expression = fmt.Sprintf("exists(%s)", node.Name)
	case sift.OperationNotExists:
		m.expression = fmt.Sprintf("notExists(%s)", node.Name)
	default:
		return sift.ErrorOperationNotSupported(string(node.Operation))
	}
	return nil
}

func (m *MockBackendAdapter) EvaluateAnd(ctx context.Context, node *sift.AndOperation) error {
	m.callCount++
	leftAdapter := &MockBackendAdapter{callCount: m.callCount}
	if err := sift.Thru(ctx, leftAdapter, node.Left); err != nil {
		return err
	}
	m.callCount = leftAdapter.callCount

	rightAdapter := &MockBackendAdapter{callCount: m.callCount}
	if err := sift.Thru(ctx, rightAdapter, node.Right); err != nil {
		return err
	}
	m.callCount = rightAdapter.callCount

	m.expression = fmt.Sprintf("(%s) AND (%s)", leftAdapter.expression, rightAdapter.expression)
	return nil
}

func (m *MockBackendAdapter) EvaluateOr(ctx context.Context, node *sift.OrOperation) error {
	m.callCount++
	leftAdapter := &MockBackendAdapter{callCount: m.callCount}
	if err := sift.Thru(ctx, leftAdapter, node.Left); err != nil {
		return err
	}
	m.callCount = leftAdapter.callCount

	rightAdapter := &MockBackendAdapter{callCount: m.callCount}
	if err := sift.Thru(ctx, rightAdapter, node.Right); err != nil {
		return err
	}
	m.callCount = rightAdapter.callCount

	m.expression = fmt.Sprintf("(%s) OR (%s)", leftAdapter.expression, rightAdapter.expression)
	return nil
}

func (m *MockBackendAdapter) EvaluateNot(ctx context.Context, node *sift.NotOperation) error {
	m.callCount++
	childAdapter := &MockBackendAdapter{callCount: m.callCount}
	if err := sift.Thru(ctx, childAdapter, node.Child); err != nil {
		return err
	}
	m.callCount = childAdapter.callCount

	m.expression = fmt.Sprintf("NOT (%s)", childAdapter.expression)
	return nil
}

func (m *MockBackendAdapter) EvaluateCustom(ctx context.Context, node sift.CustomExpression) error {
	m.callCount++
	// Handle mock custom expressions
	switch node.Type() {
	case "mock_size":
		m.expression = fmt.Sprintf("size(%s)", node.String())
	case "mock_type":
		m.expression = fmt.Sprintf("type(%s)", node.String())
	default:
		// For unknown custom types, just use their string representation
		m.expression = fmt.Sprintf("custom(%s)", node.String())
	}
	return nil
}

func (m *MockBackendAdapter) Expression() string {
	return m.expression
}

func (m *MockBackendAdapter) CallCount() int {
	return m.callCount
}

// MockCustomExpression is a mock custom expression for testing.
type MockCustomExpression struct {
	field string
	value string
}

func (m *MockCustomExpression) Type() string {
	return "mock_size"
}

func (m *MockCustomExpression) String() string {
	return fmt.Sprintf("mock_size(%s,%s)", m.field, m.value)
}

// NewMockCustomExpression creates a mock custom expression.
func NewMockCustomExpression(field, value string) sift.Expression {
	return sift.NewCustomExpression(&MockCustomExpression{
		field: field,
		value: value,
	})
}

// MockTypeExpression is another mock custom expression for testing.
type MockTypeExpression struct {
	field    string
	typeName string
}

func (m *MockTypeExpression) Type() string {
	return "mock_type"
}

func (m *MockTypeExpression) String() string {
	return fmt.Sprintf("mock_type(%s,%s)", m.field, m.typeName)
}

// NewMockTypeExpression creates a mock type expression.
func NewMockTypeExpression(field, typeName string) sift.Expression {
	return sift.NewCustomExpression(&MockTypeExpression{
		field:    field,
		typeName: typeName,
	})
}

// MockCustomFormatter handles serialization/deserialization of mock custom expressions.
type MockCustomFormatter struct{}

func (f MockCustomFormatter) FormatCustomExpression(expr sift.CustomExpression) (string, error) {
	return expr.String(), nil
}

func (f MockCustomFormatter) ParseCustomExpression(p *sift.Parser) (sift.CustomExpression, error) {
	// Read field
	field := p.ReadValue()
	if field == "" {
		return nil, fmt.Errorf("expected field")
	}

	p.SkipWhitespace()
	if !p.Expect(',') {
		return nil, fmt.Errorf("expected ',' after field")
	}
	p.SkipWhitespace()

	// Read value
	value := p.ReadValue()
	if value == "" {
		return nil, fmt.Errorf("expected value")
	}

	p.SkipWhitespace()
	if !p.Expect(')') {
		return nil, fmt.Errorf("expected ')' after value")
	}

	return &MockCustomExpression{
		field: field,
		value: value,
	}, nil
}

// MockTypeFormatter handles serialization/deserialization of mock type expressions.
type MockTypeFormatter struct{}

func (f MockTypeFormatter) FormatCustomExpression(expr sift.CustomExpression) (string, error) {
	return expr.String(), nil
}

func (f MockTypeFormatter) ParseCustomExpression(p *sift.Parser) (sift.CustomExpression, error) {
	// Read field
	field := p.ReadValue()
	if field == "" {
		return nil, fmt.Errorf("expected field")
	}

	p.SkipWhitespace()
	if !p.Expect(',') {
		return nil, fmt.Errorf("expected ',' after field")
	}
	p.SkipWhitespace()

	// Read type name
	typeName := p.ReadValue()
	if typeName == "" {
		return nil, fmt.Errorf("expected type name")
	}

	p.SkipWhitespace()
	if !p.Expect(')') {
		return nil, fmt.Errorf("expected ')' after type name")
	}

	return &MockTypeExpression{
		field:    field,
		typeName: typeName,
	}, nil
}

// init registers the mock custom expression formatters.
func init() {
	sift.RegisterCustomExpression(&MockCustomExpression{}, MockCustomFormatter{})
	sift.RegisterCustomExpression(&MockTypeExpression{}, MockTypeFormatter{})
}
