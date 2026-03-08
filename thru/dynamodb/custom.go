package dynamodb

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/nisimpson/sift"
)

// SizeExpression represents a DynamoDB size() function expression.
// It returns the size of an attribute (string length, list/set size, etc.).
type SizeExpression struct {
	Path string
	Op   sift.Operation
	Size int
}

// Type returns the type identifier for this custom expression.
func (s *SizeExpression) Type() string {
	return "dynamodb_size"
}

// String returns the string representation of the size expression.
func (s *SizeExpression) String() string {
	return fmt.Sprintf("dynamodb_size(%s,%s,%d)", s.Path, s.Op, s.Size)
}

// Size creates a custom expression for DynamoDB's size() function.
// Example: Size("tags", sift.OperationGT, 5) checks if tags list has more than 5 elements
func Size(path string, op sift.Operation, size int) sift.Expression {
	return sift.NewCustomExpression(&SizeExpression{
		Path: path,
		Op:   op,
		Size: size,
	})
}

// AttributeTypeExpression represents a DynamoDB attribute_type() function expression.
// It checks if an attribute is of a specific DynamoDB type.
type AttributeTypeExpression struct {
	Path     string
	AttrType string // S, N, B, SS, NS, BS, L, M, NULL, BOOL
}

// Type returns the type identifier for this custom expression.
func (a *AttributeTypeExpression) Type() string {
	return "dynamodb_attribute_type"
}

// String returns the string representation of the attribute_type expression.
func (a *AttributeTypeExpression) String() string {
	return fmt.Sprintf("dynamodb_attribute_type(%s,%s)", a.Path, a.AttrType)
}

// IsAttributeType creates a custom expression for DynamoDB's attribute_type() function.
// Valid types: "S" (string), "N" (number), "B" (binary), "SS" (string set),
// "NS" (number set), "BS" (binary set), "L" (list), "M" (map), "NULL", "BOOL"
func IsAttributeType(path string, attrType string) sift.Expression {
	return sift.NewCustomExpression(&AttributeTypeExpression{
		Path:     path,
		AttrType: attrType,
	})
}

// Formatter implements sift.CustomFormatter for DynamoDB custom expressions.
type Formatter struct{}

// FormatCustomExpression serializes a DynamoDB custom expression to a string.
func (f Formatter) FormatCustomExpression(expression sift.CustomExpression) (string, error) {
	switch expr := expression.(type) {
	case *SizeExpression:
		return fmt.Sprintf("dynamodb_size(%s,%s,%d)", expr.Path, expr.Op, expr.Size), nil
	case *AttributeTypeExpression:
		return fmt.Sprintf("dynamodb_attribute_type(%s,%s)", expr.Path, expr.AttrType), nil
	default:
		return "", fmt.Errorf("unsupported DynamoDB custom expression type: %T", expression)
	}
}

// ParseCustomExpression deserializes a string to a DynamoDB custom expression.
func (f Formatter) ParseCustomExpression(p *sift.Parser) (sift.CustomExpression, error) {
	// This is called after the function name has been read
	// We need to determine which type based on context, but since we're here,
	// we need to look at what was parsed. For now, return an error as parsing
	// is handled by separate registrations per type.
	return nil, fmt.Errorf("DynamoDB custom expression parsing not implemented")
}

// SizeFormatter handles serialization/deserialization of SizeExpression.
type SizeFormatter struct{}

// FormatCustomExpression serializes a SizeExpression.
func (f SizeFormatter) FormatCustomExpression(expression sift.CustomExpression) (string, error) {
	expr, ok := expression.(*SizeExpression)
	if !ok {
		return "", fmt.Errorf("expected *SizeExpression, got %T", expression)
	}
	return fmt.Sprintf("dynamodb_size(%s,%s,%d)", expr.Path, expr.Op, expr.Size), nil
}

// ParseCustomExpression deserializes a SizeExpression.
func (f SizeFormatter) ParseCustomExpression(p *sift.Parser) (sift.CustomExpression, error) {
	// Read path
	path := p.ReadValue()
	if path == "" {
		return nil, fmt.Errorf("expected path")
	}

	p.SkipWhitespace()
	if !p.Expect(',') {
		return nil, fmt.Errorf("expected ',' after path")
	}
	p.SkipWhitespace()

	// Read operation
	opStr := p.ReadValue()
	if opStr == "" {
		return nil, fmt.Errorf("expected operation")
	}

	p.SkipWhitespace()
	if !p.Expect(',') {
		return nil, fmt.Errorf("expected ',' after operation")
	}
	p.SkipWhitespace()

	// Read size
	sizeStr := p.ReadValue()
	if sizeStr == "" {
		return nil, fmt.Errorf("expected size")
	}

	var size int
	if _, err := fmt.Sscanf(sizeStr, "%d", &size); err != nil {
		return nil, fmt.Errorf("invalid size: %w", err)
	}

	p.SkipWhitespace()
	if !p.Expect(')') {
		return nil, fmt.Errorf("expected ')' after size")
	}

	return &SizeExpression{
		Path: path,
		Op:   sift.Operation(opStr),
		Size: size,
	}, nil
}

// AttributeTypeFormatter handles serialization/deserialization of AttributeTypeExpression.
type AttributeTypeFormatter struct{}

// FormatCustomExpression serializes an AttributeTypeExpression.
func (f AttributeTypeFormatter) FormatCustomExpression(expression sift.CustomExpression) (string, error) {
	expr, ok := expression.(*AttributeTypeExpression)
	if !ok {
		return "", fmt.Errorf("expected *AttributeTypeExpression, got %T", expression)
	}
	return fmt.Sprintf("dynamodb_attribute_type(%s,%s)", expr.Path, expr.AttrType), nil
}

// ParseCustomExpression deserializes an AttributeTypeExpression.
func (f AttributeTypeFormatter) ParseCustomExpression(p *sift.Parser) (sift.CustomExpression, error) {
	// Read path
	path := p.ReadValue()
	if path == "" {
		return nil, fmt.Errorf("expected path")
	}

	p.SkipWhitespace()
	if !p.Expect(',') {
		return nil, fmt.Errorf("expected ',' after path")
	}
	p.SkipWhitespace()

	// Read type
	attrType := p.ReadValue()
	if attrType == "" {
		return nil, fmt.Errorf("expected attribute type")
	}

	p.SkipWhitespace()
	if !p.Expect(')') {
		return nil, fmt.Errorf("expected ')' after type")
	}

	return &AttributeTypeExpression{
		Path:     path,
		AttrType: attrType,
	}, nil
}

// init registers the DynamoDB custom expression formatters.
func init() {
	sift.RegisterCustomExpression(&SizeExpression{}, SizeFormatter{})
	sift.RegisterCustomExpression(&AttributeTypeExpression{}, AttributeTypeFormatter{})
}

// EvaluateCustom handles DynamoDB-specific custom expressions.
func (a *Adapter) EvaluateCustom(ctx context.Context, node sift.CustomExpression) error {
	switch expr := node.(type) {
	case *SizeExpression:
		return a.evaluateSize(expr)
	case *AttributeTypeExpression:
		return a.evaluateAttributeType(expr)
	default:
		return sift.ErrorOperationNotSupported(node.Type())
	}
}

// evaluateSize translates a size expression to DynamoDB.
func (a *Adapter) evaluateSize(expr *SizeExpression) error {
	size := expression.Name(expr.Path).Size()
	value := expression.Value(expr.Size)

	switch expr.Op {
	case sift.OperationEQ:
		a.cond = size.Equal(value)
	case sift.OperationNEQ:
		a.cond = size.NotEqual(value)
	case sift.OperationLT:
		a.cond = size.LessThan(value)
	case sift.OperationLTE:
		a.cond = size.LessThanEqual(value)
	case sift.OperationGT:
		a.cond = size.GreaterThan(value)
	case sift.OperationGTE:
		a.cond = size.GreaterThanEqual(value)
	default:
		return sift.ErrorOperationNotSupported(string(expr.Op))
	}

	a.hasCond = true
	return nil
}

// evaluateAttributeType translates an attribute_type expression to DynamoDB.
func (a *Adapter) evaluateAttributeType(expr *AttributeTypeExpression) error {
	name := expression.Name(expr.Path)
	a.cond = name.AttributeType(expression.DynamoDBAttributeType(expr.AttrType))
	a.hasCond = true
	return nil
}
