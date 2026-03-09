// Package exprlang provides an expr-lang adapter for the sift filter library.
// It translates sift filter expressions into expr-lang expression syntax.
package exprlang

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/nisimpson/sift"
)

// Adapter translates sift filter expressions into expr-lang expression syntax.
// It accumulates the expression string during traversal.
type Adapter struct {
	expr string
}

// NewAdapter creates a new expr-lang adapter.
func NewAdapter() *Adapter {
	return &Adapter{}
}

// Evaluator returns a sift evaluator configured for expr-lang.
func (a *Adapter) Evaluator(ctx context.Context) *sift.Evaluator {
	return &sift.Evaluator{
		ConditionEvaluator: a,
		AndEvaluator:       a,
		OrEvaluator:        a,
		NotEvaluator:       a,
		CustomEvaluator:    a,
	}
}

// EvaluateCondition translates a sift condition into an expr-lang expression.
func (a *Adapter) EvaluateCondition(ctx context.Context, node *sift.Condition) error {
	name := node.Name
	value := a.formatValue(node.Value)

	switch node.Operation {
	case sift.OperationEQ:
		a.expr = fmt.Sprintf("%s == %s", name, value)
	case sift.OperationNEQ:
		a.expr = fmt.Sprintf("%s != %s", name, value)
	case sift.OperationLT:
		a.expr = fmt.Sprintf("%s < %s", name, value)
	case sift.OperationLTE:
		a.expr = fmt.Sprintf("%s <= %s", name, value)
	case sift.OperationGT:
		a.expr = fmt.Sprintf("%s > %s", name, value)
	case sift.OperationGTE:
		a.expr = fmt.Sprintf("%s >= %s", name, value)
	case sift.OperationIn:
		// Use 'in' operator for membership testing
		a.expr = fmt.Sprintf("%s in %s", value, name)
	case sift.OperationContains:
		// Use contains operator (infix notation)
		a.expr = fmt.Sprintf("%s contains %s", name, a.quoteString(node.Value))
	case sift.OperationBeginsWith:
		// Use startsWith operator (infix notation)
		a.expr = fmt.Sprintf("%s startsWith %s", name, a.quoteString(node.Value))
	case sift.OperationExists:
		// Check if field is not nil
		a.expr = fmt.Sprintf("%s != nil", name)
	case sift.OperationNotExists:
		// Check if field is nil
		a.expr = fmt.Sprintf("%s == nil", name)
	case sift.OperationBetween:
		// For between, expect value to be comma-separated: "min,max"
		parts := strings.SplitN(node.Value, ",", 2)
		if len(parts) != 2 {
			return fmt.Errorf("between operation requires two values separated by comma")
		}
		min := a.formatValue(strings.TrimSpace(parts[0]))
		max := a.formatValue(strings.TrimSpace(parts[1]))
		a.expr = fmt.Sprintf("%s >= %s and %s <= %s", name, min, name, max)
	default:
		return sift.ErrorOperationNotSupported(string(node.Operation))
	}

	return nil
}

// EvaluateAnd combines two expressions with logical AND.
func (a *Adapter) EvaluateAnd(ctx context.Context, node *sift.AndOperation) error {
	leftAdapter := NewAdapter()
	if err := sift.Thru(ctx, leftAdapter, sift.WithFilter(node.Left)); err != nil {
		return err
	}

	rightAdapter := NewAdapter()
	if err := sift.Thru(ctx, rightAdapter, sift.WithFilter(node.Right)); err != nil {
		return err
	}

	a.expr = fmt.Sprintf("(%s) && (%s)", leftAdapter.expr, rightAdapter.expr)
	return nil
}

// EvaluateOr combines two expressions with logical OR.
func (a *Adapter) EvaluateOr(ctx context.Context, node *sift.OrOperation) error {
	leftAdapter := NewAdapter()
	if err := sift.Thru(ctx, leftAdapter, sift.WithFilter(node.Left)); err != nil {
		return err
	}

	rightAdapter := NewAdapter()
	if err := sift.Thru(ctx, rightAdapter, sift.WithFilter(node.Right)); err != nil {
		return err
	}

	a.expr = fmt.Sprintf("(%s) || (%s)", leftAdapter.expr, rightAdapter.expr)
	return nil
}

// EvaluateNot negates an expression.
func (a *Adapter) EvaluateNot(ctx context.Context, node *sift.NotOperation) error {
	childAdapter := NewAdapter()
	if err := sift.Thru(ctx, childAdapter, sift.WithFilter(node.Child)); err != nil {
		return err
	}

	a.expr = fmt.Sprintf("!(%s)", childAdapter.expr)
	return nil
}

// EvaluateCustom handles custom expr-lang expressions.
func (a *Adapter) EvaluateCustom(ctx context.Context, node sift.CustomExpression) error {
	// Only handle CustomExpression types from this package
	if custom, ok := node.(*CustomExpression); ok {
		a.expr = custom.Expression()
		return nil
	}
	return sift.ErrorOperationNotSupported(node.Type())
}

// Expression returns the accumulated expr-lang expression string.
func (a *Adapter) Expression() string {
	return a.expr
}

// formatValue attempts to parse the string value and format it appropriately for expr-lang.
// Numbers are left unquoted, strings are quoted, booleans are converted to true/false.
func (a *Adapter) formatValue(s string) string {
	// Try parsing as int
	if _, err := strconv.ParseInt(s, 10, 64); err == nil {
		return s
	}

	// Try parsing as float
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return s
	}

	// Try parsing as bool
	if b, err := strconv.ParseBool(s); err == nil {
		return strconv.FormatBool(b)
	}

	// Fall back to quoted string
	return a.quoteString(s)
}

// quoteString properly quotes a string value for expr-lang, escaping special characters.
func (a *Adapter) quoteString(s string) string {
	return strconv.Quote(s)
}
