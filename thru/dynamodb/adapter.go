// Package dynamodb provides a DynamoDB adapter for the sift filter library.
// It translates sift filter expressions into DynamoDB expression syntax.
package dynamodb

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/nisimpson/sift"
)

// AttributeType represents the type of a DynamoDB attribute.
type AttributeType string

const (
	AttributeTypeString AttributeType = "string"
	AttributeTypeNumber AttributeType = "number"
	AttributeTypeBool   AttributeType = "bool"
	AttributeTypeAuto   AttributeType = "auto" // Auto-detect from string value
)

// Config holds configuration options for the DynamoDB adapter.
type Config struct {
	// AttributeTypes maps attribute names to their types.
	// If an attribute is not in this map, auto-detection is used.
	AttributeTypes map[string]AttributeType
}

// Adapter translates sift filter expressions into DynamoDB expression syntax.
// It accumulates the filter expression, attribute names, and attribute values
// during traversal.
type Adapter struct {
	builder          expression.Builder
	cond             expression.ConditionBuilder
	hasCond          bool
	config           *Config
	scanIndexForward *bool // nil means not set, true = ascending, false = descending
}

// NewAdapter creates a new DynamoDB adapter with default configuration (auto-detect types).
func NewAdapter() *Adapter {
	return &Adapter{
		config: &Config{
			AttributeTypes: make(map[string]AttributeType),
		},
	}
}

// NewAdapterWithConfig creates a new DynamoDB adapter with custom configuration.
func NewAdapterWithConfig(config *Config) *Adapter {
	if config == nil {
		config = &Config{
			AttributeTypes: make(map[string]AttributeType),
		}
	}
	if config.AttributeTypes == nil {
		config.AttributeTypes = make(map[string]AttributeType)
	}
	return &Adapter{
		config: config,
	}
}

// Evaluator returns a sift evaluator configured for DynamoDB.
func (a *Adapter) Evaluator(ctx context.Context) *sift.Evaluator {
	return &sift.Evaluator{
		ConditionEvaluator: a,
		AndEvaluator:       a,
		OrEvaluator:        a,
		NotEvaluator:       a,
		CustomEvaluator:    a,
		SortFieldEvaluator: a,
		SortListEvaluator:  a,
	}
}

// EvaluateCondition translates a sift condition into a DynamoDB condition.
func (a *Adapter) EvaluateCondition(ctx context.Context, node *sift.Condition) error {
	name := expression.Name(node.Name)

	// Determine the type for this attribute
	attrType, ok := a.config.AttributeTypes[node.Name]
	if !ok {
		attrType = AttributeTypeAuto
	}

	// Parse value based on configured or auto-detected type
	value := a.parseValueWithType(node.Value, attrType)

	switch node.Operation {
	case sift.OperationEQ:
		a.cond = name.Equal(value)
	case sift.OperationNEQ:
		a.cond = name.NotEqual(value)
	case sift.OperationLT:
		a.cond = name.LessThan(value)
	case sift.OperationLTE:
		a.cond = name.LessThanEqual(value)
	case sift.OperationGT:
		a.cond = name.GreaterThan(value)
	case sift.OperationGTE:
		a.cond = name.GreaterThanEqual(value)
	case sift.OperationContains:
		a.cond = name.Contains(node.Value) // always uses string value
	case sift.OperationBeginsWith:
		a.cond = name.BeginsWith(node.Value) // always uses string value
	case sift.OperationIn:
		a.cond = name.In(value)
	case sift.OperationExists:
		a.cond = name.AttributeExists()
	case sift.OperationNotExists:
		a.cond = name.AttributeNotExists()
	case sift.OperationBetween:
		// For between, we expect value to be comma-separated: "min,max"
		a.cond = name.Between(value, value)
	default:
		return sift.ErrorOperationNotSupported(string(node.Operation))
	}

	a.hasCond = true
	return nil
}

// EvaluateAnd combines two conditions with logical AND.
func (a *Adapter) EvaluateAnd(ctx context.Context, node *sift.AndOperation) error {
	leftAdapter := &Adapter{config: a.config}
	if err := sift.Thru(ctx, leftAdapter, node.Left); err != nil {
		return err
	}

	rightAdapter := &Adapter{config: a.config}
	if err := sift.Thru(ctx, rightAdapter, node.Right); err != nil {
		return err
	}

	a.cond = leftAdapter.cond.And(rightAdapter.cond)
	a.hasCond = true
	return nil
}

// EvaluateOr combines two conditions with logical OR.
func (a *Adapter) EvaluateOr(ctx context.Context, node *sift.OrOperation) error {
	leftAdapter := &Adapter{config: a.config}
	if err := sift.Thru(ctx, leftAdapter, node.Left); err != nil {
		return err
	}

	rightAdapter := &Adapter{config: a.config}
	if err := sift.Thru(ctx, rightAdapter, node.Right); err != nil {
		return err
	}

	a.cond = leftAdapter.cond.Or(rightAdapter.cond)
	a.hasCond = true
	return nil
}

// EvaluateNot negates a condition.
func (a *Adapter) EvaluateNot(ctx context.Context, node *sift.NotOperation) error {
	childAdapter := &Adapter{config: a.config}
	if err := sift.Thru(ctx, childAdapter, node.Child); err != nil {
		return err
	}

	a.cond = expression.Not(childAdapter.cond)
	a.hasCond = true
	return nil
}

// Expression builds and returns the complete DynamoDB expression.
// This can be used directly with DynamoDB Query or Scan operations.
func (a *Adapter) Expression() (expression.Expression, error) {
	if !a.hasCond {
		return expression.Expression{}, fmt.Errorf("no condition built")
	}

	a.builder = expression.NewBuilder().WithCondition(a.cond)
	expr, err := a.builder.Build()
	if err != nil {
		return expression.Expression{}, fmt.Errorf("failed to build expression: %w", err)
	}

	return expr, nil
}

// Condition returns the built condition builder for use with expression.NewBuilder().
// This is useful if you want to combine the sift filter with other expression components
// like projections, updates, or key conditions.
func (a *Adapter) Condition() expression.ConditionBuilder {
	return a.cond
}

// parseValueWithType parses a string value according to the specified type.
func (a *Adapter) parseValueWithType(s string, attrType AttributeType) expression.ValueBuilder {
	switch attrType {
	case AttributeTypeString:
		return expression.Value(s)
	case AttributeTypeNumber:
		// Try int first, then float
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return expression.Value(i)
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return expression.Value(f)
		}
		// Fall back to string if parsing fails
		return expression.Value(s)
	case AttributeTypeBool:
		if b, err := strconv.ParseBool(s); err == nil {
			return expression.Value(b)
		}
		// Fall back to string if parsing fails
		return expression.Value(s)
	case AttributeTypeAuto:
		return parseValue(s)
	default:
		return expression.Value(s)
	}
}

// parseValue attempts to parse the string value into the most appropriate type.
// It tries to parse as int, float, and bool before falling back to string.
// This ensures proper type matching with DynamoDB attributes.
func parseValue(s string) expression.ValueBuilder {
	// Try parsing as int
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return expression.Value(i)
	}

	// Try parsing as float
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return expression.Value(f)
	}

	// Try parsing as bool
	if b, err := strconv.ParseBool(s); err == nil {
		return expression.Value(b)
	}

	// Fall back to string
	return expression.Value(s)
}

// EvaluateSortField sets the ScanIndexForward parameter based on the sort direction.
// DynamoDB only supports sorting by a single field (the sort key), so this method
// only processes the first sort field and ignores subsequent fields.
//
// Note: DynamoDB sorting is limited to the sort key of the table or index being queried.
// This method sets the ScanIndexForward parameter which controls the sort direction:
// - true (ascending): items are returned in ascending order by sort key
// - false (descending): items are returned in descending order by sort key
func (a *Adapter) EvaluateSortField(ctx context.Context, field *sift.SortField) error {
	// Only set if not already set (DynamoDB only supports single field sorting)
	if a.scanIndexForward == nil {
		forward := field.Direction == sift.SortAsc
		a.scanIndexForward = &forward
	}
	return nil
}

// EvaluateSortList processes a list of sort fields.
// DynamoDB only supports sorting by a single field (the sort key), so this method
// only processes the first sort field and ignores subsequent fields.
func (a *Adapter) EvaluateSortList(ctx context.Context, list *sift.SortList) error {
	if len(list.Fields) > 0 {
		return a.EvaluateSortField(ctx, list.Fields[0])
	}
	return nil
}

// ScanIndexForward returns the ScanIndexForward parameter for DynamoDB Query operations.
// Returns nil if no sort was specified, true for ascending order, false for descending order.
//
// Usage with AWS SDK v2:
//
//	adapter := dynamodb.NewAdapter()
//	sift.SortThru(ctx, adapter, sortExpr)
//	if forward := adapter.ScanIndexForward(); forward != nil {
//	    input.ScanIndexForward = forward
//	}
func (a *Adapter) ScanIndexForward() *bool {
	return a.scanIndexForward
}
