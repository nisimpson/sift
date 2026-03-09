// Package sift provides a universal filter expression language using an Abstract Syntax Tree (AST).
// Write filter logic once, then implement backend-specific evaluators to translate it into native queries.
package sift

import (
	"context"
	"errors"
	"fmt"
)

var (
	// ErrUnsupported indicates an operation or node type is not supported by the evaluator.
	ErrUnsupported = errors.ErrUnsupported
)

// Evaluator holds the interface implementations for evaluating filter nodes.
// Backend implementations should embed this in their evaluator types and set
// only the interfaces they support.
type Evaluator struct {
	ConditionEvaluator
	AndEvaluator
	OrEvaluator
	NotEvaluator
	CustomEvaluator
	SortFieldEvaluator
	SortListEvaluator
	OffsetPaginationEvaluator
	CursorPaginationEvaluator
}

// Adapter provides the bridge between the sift filter AST and backend-specific implementations.
// Backends implement this interface to provide their own evaluator that can translate
// filter expressions into native queries or operations.
type Adapter interface {
	// Evaluator returns an evaluator configured for the specific backend.
	// The evaluator should have the appropriate interface implementations set
	// based on what operations the backend supports.
	Evaluator(ctx context.Context) *Evaluator
}

// Option represents a query option that can be applied to an adapter.
// Options include filtering, sorting, and pagination.
type Option interface {
	apply(ctx context.Context, evaluator *Evaluator) error
}

// filterOption wraps a filter expression as an option.
type filterOption struct {
	expr Expression
}

func (f filterOption) apply(ctx context.Context, evaluator *Evaluator) error {
	return f.expr.accept(ctx, evaluator)
}

// WithFilter creates an option that applies a filter expression.
func WithFilter(expr Expression) Option {
	return filterOption{expr: expr}
}

// sortOption wraps a sort expression as an option.
type sortOption struct {
	expr SortExpression
}

func (s sortOption) apply(ctx context.Context, evaluator *Evaluator) error {
	return s.expr.accept(ctx, evaluator)
}

// WithSort creates an option that applies a sort expression.
func WithSort(expr SortExpression) Option {
	return sortOption{expr: expr}
}

// paginationOption wraps a pagination expression as an option.
type paginationOption struct {
	expr PaginationExpression
}

func (p paginationOption) apply(ctx context.Context, evaluator *Evaluator) error {
	return p.expr.accept(ctx, evaluator)
}

// WithPagination creates an option that applies a pagination expression.
func WithPagination(expr PaginationExpression) Option {
	return paginationOption{expr: expr}
}

// Thru evaluates query options using the provided adapter.
// This is the main entry point for processing filters, sorts, and pagination.
//
// Example:
//
//	sift.Thru(ctx, adapter,
//	    sift.WithFilter(filter),
//	    sift.WithSort(sort),
//	    sift.WithPagination(page))
func Thru(ctx context.Context, adapter Adapter, options ...Option) error {
	if adapter == nil {
		return fmt.Errorf("adapter is nil")
	}
	
	evaluator := adapter.Evaluator(ctx)
	for _, opt := range options {
		if err := opt.apply(ctx, evaluator); err != nil {
			return err
		}
	}
	
	return nil
}

// Expression represents an expression in the filter AST.
type Expression interface {
	fmt.Stringer
	accept(ctx context.Context, e *Evaluator) error
}

// Operation defines the type of comparison or logical operation.
type Operation string

const (
	OperationEQ         Operation = "eq"          // equal
	OperationNEQ        Operation = "ne"          // not equal
	OperationLT         Operation = "lt"          // less than
	OperationLTE        Operation = "le"          // less than or equal
	OperationGT         Operation = "gt"          // greater than
	OperationGTE        Operation = "ge"          // greater than or equal
	OperationContains   Operation = "contains"    // substring match
	OperationBeginsWith Operation = "begins_with" // prefix match
	OperationIn         Operation = "in"          // value in list
	OperationExists     Operation = "exists"      // attribute exists
	OperationNotExists  Operation = "not_exists"  // attribute doesn't exist
	OperationBetween    Operation = "between"     // value between two bounds
)

// IsCondition returns true if the operation is a condition operation that requires a value.
// Condition operations include equality, inequality, relational, and pattern matching operations.
func (o Operation) IsCondition() bool {
	switch o {
	case OperationEQ, OperationNEQ, OperationLT, OperationLTE,
		OperationGT, OperationGTE, OperationContains, OperationBeginsWith,
		OperationIn, OperationBetween:
		return true
	}
	return false
}

// IsExistence returns true if the operation is an existence check that only tests
// for the presence or absence of an attribute without comparing values.
func (o Operation) IsExistence() bool {
	switch o {
	case OperationExists, OperationNotExists:
		return true
	}
	return false
}

// Condition represents a single filter condition (e.g., "status = active").
type Condition struct {
	Name      string    // attribute name
	Operation Operation // operation type (use Operation constants)
	Value     string    // comparison value
}

// And creates a logical AND operation between this condition and another expression.
// Returns a Builder that can be used to chain additional operations.
func (e *Condition) And(expr Expression) ExpressionBuilder {
	return NewExpressionBuilder(e).And(expr)
}

// Or creates a logical OR operation between this condition and another expression.
// Returns a Builder that can be used to chain additional operations.
func (e *Condition) Or(expr Expression) ExpressionBuilder {
	return NewExpressionBuilder(e).Or(expr)
}

// Not creates a logical NOT operation, negating this condition.
// Returns an ExpressionBuilder that can be used to chain additional operations.
func (e *Condition) Not() ExpressionBuilder {
	return NewExpressionBuilder(e).Not()
}

func (e *Condition) accept(ctx context.Context, eval *Evaluator) error {
	if eval.ConditionEvaluator == nil {
		return ErrorNodeNotSupported(e)
	}
	return eval.EvaluateCondition(ctx, e)
}

// String returns the string representation of the condition in the format "operation(name,value)".
func (c Condition) String() string {
	return fmt.Sprintf("%s(%s,%s)", c.Operation, c.Name, c.Value)
}

// ConditionEvaluator evaluates single filter conditions.
// Implement this interface to handle basic comparisons in your backend.
type ConditionEvaluator interface {
	EvaluateCondition(ctx context.Context, node *Condition) error
}

// AndOperation represents a logical AND between two filter expressions.
type AndOperation struct {
	Left  Expression
	Right Expression
}

// AndEvaluator evaluates AND operations.
// Implement this interface to support logical AND in your backend.
type AndEvaluator interface {
	EvaluateAnd(ctx context.Context, node *AndOperation) error
}

func (a *AndOperation) accept(ctx context.Context, eval *Evaluator) error {
	if eval.AndEvaluator == nil {
		return ErrorNodeNotSupported(a)
	}
	return eval.EvaluateAnd(ctx, a)
}

// String returns the string representation of the AND operation.
func (a AndOperation) String() string {
	return fmt.Sprintf("and(%s,%s)", a.Left, a.Right)
}

// OrOperation represents a logical OR between two filter expressions.
type OrOperation struct {
	Left  Expression
	Right Expression
}

// OrEvaluator evaluates OR operations.
// Implement this interface to support logical OR in your backend.
type OrEvaluator interface {
	EvaluateOr(ctx context.Context, node *OrOperation) error
}

func (o *OrOperation) accept(ctx context.Context, eval *Evaluator) error {
	if eval.OrEvaluator == nil {
		return ErrorNodeNotSupported(o)
	}
	return eval.EvaluateOr(ctx, o)
}

// String returns the string representation of the OR operation.
func (o OrOperation) String() string {
	return fmt.Sprintf("or(%s,%s)", o.Left, o.Right)
}

// NotOperation represents a logical NOT (negation) of a filter expression.
type NotOperation struct {
	Child Expression
}

// NotEvaluator evaluates NOT operations.
// Implement this interface to support logical negation in your backend.
type NotEvaluator interface {
	EvaluateNot(ctx context.Context, node *NotOperation) error
}

func (n *NotOperation) accept(ctx context.Context, eval *Evaluator) error {
	if eval.NotEvaluator == nil {
		return ErrorNodeNotSupported(n)
	}
	return eval.EvaluateNot(ctx, n)
}

// String returns the string representation of the NOT operation.
func (n NotOperation) String() string {
	return fmt.Sprintf("not(%s)", n.Child)
}

// CustomExpression allows backends to define custom node types beyond the standard operations.
type CustomExpression interface {
	fmt.Stringer
	Type() string
}

type customNodeWrapper struct {
	node CustomExpression
}

// NewCustomExpression wraps a custom node implementation to make it compatible with the Expression interface.
// This allows backends to define their own node types while integrating with the standard AST.
func NewCustomExpression(custom CustomExpression) Expression {
	return &customNodeWrapper{node: custom}
}

func (w customNodeWrapper) accept(ctx context.Context, eval *Evaluator) error {
	if eval.CustomEvaluator == nil {
		return ErrorOperationNotSupported(w.node.Type())
	}
	return eval.EvaluateCustom(ctx, w.node)
}

// String returns the string representation of the wrapped custom node.
func (w customNodeWrapper) String() string {
	return w.node.String()
}

// CustomEvaluator evaluates custom node types.
// Implement this interface to support backend-specific operations.
type CustomEvaluator interface {
	EvaluateCustom(ctx context.Context, node CustomExpression) error
}

// ErrorOperationNotSupported returns an error indicating the operation is not supported.
func ErrorOperationNotSupported(operation string) error {
	return fmt.Errorf("%w: operation not supported: %v", ErrUnsupported, operation)
}

// ErrorNodeNotSupported returns an error indicating the expression type is not supported.
func ErrorNodeNotSupported[T Expression](expr T) error {
	return fmt.Errorf("%w: expression not supported: %T", ErrUnsupported, expr)
}
