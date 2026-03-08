package sift

import (
	"context"
	"fmt"
)

// ExpressionBuilder provides a fluent interface for constructing complex filter expressions.
// It wraps an Expression and allows chaining of logical operations like And, Or, and Not.
type ExpressionBuilder struct {
	expr Expression
}

// NewExpressionBuilder creates a new ExpressionBuilder that wraps the given expression.
// This allows the expression to be used with the fluent interface for chaining
// logical operations like And, Or, and Not.
func NewExpressionBuilder(expr Expression) ExpressionBuilder {
	return ExpressionBuilder{expr: expr}
}

func buildFromCondition(field string, op Operation, val any) ExpressionBuilder {
	return NewExpressionBuilder(&Condition{
		Name:      field,
		Operation: op,
		Value:     fmt.Sprintf("%v", val),
	})
}

// Lt creates a new ExpressionBuilder for a "less than" comparison.
// The resulting expression will be true if the field value is less than the given value.
func Lt[T comparable](field string, val T) ExpressionBuilder {
	return buildFromCondition(field, OperationLT, val)
}

// Lte creates a new ExpressionBuilder for a "less than or equal" comparison.
// The resulting expression will be true if the field value is less than or equal to the given value.
func Lte[T comparable](field string, val T) ExpressionBuilder {
	return buildFromCondition(field, OperationLTE, val)
}

// Gt creates a new ExpressionBuilder for a "greater than" comparison.
// The resulting expression will be true if the field value is greater than the given value.
func Gt[T comparable](field string, val T) ExpressionBuilder {
	return buildFromCondition(field, OperationGT, val)
}

// Gte creates a new ExpressionBuilder for a "greater than or equal" comparison.
// The resulting expression will be true if the field value is greater than or equal to the given value.
func Gte[T comparable](field string, val T) ExpressionBuilder {
	return buildFromCondition(field, OperationGTE, val)
}

// Eq creates a new ExpressionBuilder for an equality comparison.
// The resulting expression will be true if the field value equals the given value.
func Eq[T comparable](field string, val T) ExpressionBuilder {
	return buildFromCondition(field, OperationEQ, val)
}

// Neq creates a new ExpressionBuilder for a "not equal" comparison.
// The resulting expression will be true if the field value does not equal the given value.
func Neq[T comparable](field string, val T) ExpressionBuilder {
	return buildFromCondition(field, OperationNEQ, val)
}

// Contains creates a new ExpressionBuilder for a containment check.
// The resulting expression will be true if the field value contains the given value.
func Contains[T comparable](field string, val T) ExpressionBuilder {
	return buildFromCondition(field, OperationContains, val)
}

// In creates a new ExpressionBuilder for membership testing.
// The resulting expression will be true if the field value is in the given collection.
func In[T comparable](field string, val T) ExpressionBuilder {
	return buildFromCondition(field, OperationIn, val)
}

// Between creates a new ExpressionBuilder for range checking.
// The resulting expression will be true if the field value is between min and max (inclusive).
func Between[T comparable](field string, min, max T) ExpressionBuilder {
	return buildFromCondition(field, OperationBetween, fmt.Sprintf("%v,%v", min, max))
}

// Exists creates a new ExpressionBuilder that checks if a field exists.
// The resulting expression will be true if the field is present.
func Exists(field string) ExpressionBuilder {
	return NewExpressionBuilder(&Condition{
		Name:      field,
		Operation: OperationExists,
	})
}

// NotExists creates a new ExpressionBuilder that checks if a field does not exist.
// The resulting expression will be true if the field is not present.
func NotExists(field string) ExpressionBuilder {
	return NewExpressionBuilder(&Condition{
		Name:      field,
		Operation: OperationNotExists,
	})
}

// accept implements the Expression interface by delegating to the wrapped expression.
// It allows the Builder to be used anywhere an Expression is expected.
func (b ExpressionBuilder) accept(ctx context.Context, e *Evaluator) error {
	return b.expr.accept(ctx, e)
}

// String returns the string representation of the wrapped expression.
func (b ExpressionBuilder) String() string { return b.expr.String() }

// And creates a new Builder that represents the logical AND of this Builder and the given expression.
// The resulting expression will be true only if both operands evaluate to true.
func (b ExpressionBuilder) And(expr Expression) ExpressionBuilder {
	return ExpressionBuilder{
		expr: &AndOperation{Left: b, Right: expr},
	}
}

// Or creates a new Builder that represents the logical OR of this Builder and the given expression.
// The resulting expression will be true if either operand evaluates to true.
func (b ExpressionBuilder) Or(expr Expression) ExpressionBuilder {
	return ExpressionBuilder{
		expr: &OrOperation{Left: b, Right: expr},
	}
}

// Not creates a new Builder that represents the logical NOT of this Builder.
// The resulting expression will be true if this Builder evaluates to false, and vice versa.
func (b ExpressionBuilder) Not() ExpressionBuilder {
	return ExpressionBuilder{
		expr: &NotOperation{Child: b},
	}
}

// SortBuilder provides a fluent API for building sort expressions.
type SortBuilder struct {
	fields []*SortField
}

// Sort creates a new sort expression with a single field.
//
// Example:
//
//	sort := sift.Sort("created_at", sift.SortDesc)
func Sort(name string, direction SortDirection) SortBuilder {
	return SortBuilder{
		fields: []*SortField{{Name: name, Direction: direction}},
	}
}

// ThenBy adds another sort field to the expression.
// Fields are applied in the order they are added.
//
// Example:
//
//	sort := sift.Sort("created_at", sift.SortDesc).ThenBy("name", sift.SortAsc)
func (b SortBuilder) ThenBy(name string, direction SortDirection) SortBuilder {
	b.fields = append(b.fields, &SortField{Name: name, Direction: direction})
	return b
}

// NullsLast sets the NullsLast flag on the most recently added field.
// This controls whether NULL values appear last in the sort order.
// Note: Not all backends support this feature.
//
// Example:
//
//	sort := sift.Sort("email", sift.SortAsc).NullsLast()
func (b SortBuilder) NullsLast() SortBuilder {
	if len(b.fields) > 0 {
		b.fields[len(b.fields)-1].NullsLast = true
	}
	return b
}

// accept implements SortExpression for SortBuilder.
// This allows SortBuilder to be used directly without calling Build().
func (b SortBuilder) accept(ctx context.Context, evaluator *Evaluator) error {
	list := &SortList{Fields: b.fields}
	return list.accept(ctx, evaluator)
}
