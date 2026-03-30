package exprlang

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/ast"
	"github.com/nisimpson/sift"
)

// ParseOption configures the parser behavior.
type ParseOption func(*parserConfig)

// WithLenientMode enables lenient parsing where unsupported expr-lang
// constructs are wrapped as exprlang.RawExpression custom sift nodes
// instead of returning errors.
func WithLenientMode() ParseOption {
	return func(c *parserConfig) {
		c.lenient = true
	}
}

// parserConfig holds configuration for the parser.
type parserConfig struct {
	lenient bool // when true, wrap unsupported constructs as RawExpression
}

// walker holds the parser configuration and walks the expr-lang AST.
type walker struct {
	cfg    *parserConfig
	source string // original input string, used for RawExpression extraction
}

// Parse parses an expr-lang expression string into a sift.Expression AST.
// It uses expr.Compile() to parse the string into an expr-lang AST, then
// walks the AST to produce sift nodes.
//
// By default, the parser operates in strict mode where unsupported
// expr-lang constructs return descriptive errors. Use WithLenientMode()
// to wrap unsupported constructs as RawExpression nodes instead.
func Parse(input string, opts ...ParseOption) (sift.Expression, error) {
	if strings.TrimSpace(input) == "" {
		return nil, fmt.Errorf("exprlang: empty input")
	}

	cfg := &parserConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	program, err := expr.Compile(input, expr.AllowUndefinedVariables())
	if err != nil {
		return nil, fmt.Errorf("exprlang: %w", err)
	}

	w := &walker{
		cfg:    cfg,
		source: input,
	}

	return w.walkNode(program.Node())
}

// walkNode dispatches on the concrete ast.Node type and returns a sift.Expression.
func (w *walker) walkNode(node ast.Node) (sift.Expression, error) {
	switch n := node.(type) {
	case *ast.BinaryNode:
		return w.walkBinary(n)
	case *ast.UnaryNode:
		return w.walkUnary(n)
	default:
		return w.handleUnsupported(node, fmt.Sprintf("node type %T", node))
	}
}

// walkBinary handles binary operator nodes.
func (w *walker) walkBinary(node *ast.BinaryNode) (sift.Expression, error) {
	switch node.Operator {
	case "==", "!=", "<", "<=", ">", ">=":
		return w.walkComparison(node)
	case "&&", "and":
		return w.walkAnd(node)
	case "||", "or":
		return w.walkOr(node)
	case "contains":
		return w.walkStringOp(node, sift.OperationContains)
	case "startsWith":
		return w.walkStringOp(node, sift.OperationBeginsWith)
	case "in":
		return w.walkIn(node)
	case "endsWith", "matches":
		return w.handleUnsupported(node, fmt.Sprintf("string operator (%s) has no sift equivalent", node.Operator))
	default:
		return w.handleUnsupported(node, fmt.Sprintf("operator (%s)", node.Operator))
	}
}

// walkUnary handles unary operator nodes.
func (w *walker) walkUnary(node *ast.UnaryNode) (sift.Expression, error) {
	switch node.Operator {
	case "!", "not":
		child, err := w.walkNode(node.Node)
		if err != nil {
			return nil, err
		}
		return &sift.NotOperation{Child: child}, nil
	default:
		return w.handleUnsupported(node, fmt.Sprintf("unary operator (%s)", node.Operator))
	}
}

// walkComparison handles comparison binary operators (==, !=, <, <=, >, >=).
func (w *walker) walkComparison(node *ast.BinaryNode) (sift.Expression, error) {
	// Check for nil comparisons (exists / not-exists).
	if _, isNil := node.Right.(*ast.NilNode); isNil {
		fieldName, ok := extractFieldName(node.Left)
		if !ok {
			return w.handleUnsupported(node, "nil comparison with non-field left-hand side")
		}
		switch node.Operator {
		case "==":
			return &sift.Condition{Name: fieldName, Operation: sift.OperationNotExists}, nil
		case "!=":
			return &sift.Condition{Name: fieldName, Operation: sift.OperationExists}, nil
		default:
			return w.handleUnsupported(node, fmt.Sprintf("nil comparison with operator (%s)", node.Operator))
		}
	}

	fieldName, ok := extractFieldName(node.Left)
	if !ok {
		return w.handleUnsupported(node, "comparison with non-field left-hand side")
	}

	value, ok := extractValue(node.Right)
	if !ok {
		return w.handleUnsupported(node, "comparison with non-literal right-hand side")
	}

	var op sift.Operation
	switch node.Operator {
	case "==":
		op = sift.OperationEQ
	case "!=":
		op = sift.OperationNEQ
	case "<":
		op = sift.OperationLT
	case "<=":
		op = sift.OperationLTE
	case ">":
		op = sift.OperationGT
	case ">=":
		op = sift.OperationGTE
	}

	return &sift.Condition{Name: fieldName, Operation: op, Value: value}, nil
}

// walkAnd handles && / and operators. Checks for between pattern first.
func (w *walker) walkAnd(node *ast.BinaryNode) (sift.Expression, error) {
	// Try between detection before normal AND handling.
	if between, ok := w.tryBetween(node); ok {
		return between, nil
	}

	left, err := w.walkNode(node.Left)
	if err != nil {
		return nil, err
	}
	right, err := w.walkNode(node.Right)
	if err != nil {
		return nil, err
	}
	return &sift.AndOperation{Left: left, Right: right}, nil
}

// walkOr handles || / or operators.
func (w *walker) walkOr(node *ast.BinaryNode) (sift.Expression, error) {
	left, err := w.walkNode(node.Left)
	if err != nil {
		return nil, err
	}
	right, err := w.walkNode(node.Right)
	if err != nil {
		return nil, err
	}
	return &sift.OrOperation{Left: left, Right: right}, nil
}

// walkStringOp handles contains and startsWith binary operators.
func (w *walker) walkStringOp(node *ast.BinaryNode, op sift.Operation) (sift.Expression, error) {
	fieldName, ok := extractFieldName(node.Left)
	if !ok {
		return w.handleUnsupported(node, "string operator with non-field left-hand side")
	}
	value, ok := extractValue(node.Right)
	if !ok {
		return w.handleUnsupported(node, "string operator with non-literal right-hand side")
	}
	return &sift.Condition{Name: fieldName, Operation: op, Value: value}, nil
}

// walkIn handles the in membership operator.
// Note: expr-lang uses reversed operand order: "value" in field.
func (w *walker) walkIn(node *ast.BinaryNode) (sift.Expression, error) {
	// In expr-lang: left is the value, right is the field.
	value, ok := extractValue(node.Left)
	if !ok {
		return w.handleUnsupported(node, "in operator with non-literal left-hand side")
	}
	fieldName, ok := extractFieldName(node.Right)
	if !ok {
		return w.handleUnsupported(node, "in operator with non-field right-hand side")
	}
	return &sift.Condition{Name: fieldName, Operation: sift.OperationIn, Value: value}, nil
}

// tryBetween checks if a BinaryNode represents a between pattern:
//
//	field >= min && field <= max  (or with 'and' keyword)
//
// Returns (sift.Expression, true) if matched, (nil, false) otherwise.
func (w *walker) tryBetween(node *ast.BinaryNode) (*sift.Condition, bool) {
	leftBin, leftOk := node.Left.(*ast.BinaryNode)
	rightBin, rightOk := node.Right.(*ast.BinaryNode)
	if !leftOk || !rightOk {
		return nil, false
	}

	// Identify which side is >= and which is <=.
	var gteNode, lteNode *ast.BinaryNode
	if leftBin.Operator == ">=" && rightBin.Operator == "<=" {
		gteNode, lteNode = leftBin, rightBin
	} else if leftBin.Operator == "<=" && rightBin.Operator == ">=" {
		lteNode, gteNode = leftBin, rightBin
	} else {
		return nil, false
	}

	// Both sides must reference the same field.
	gteField, ok1 := extractFieldName(gteNode.Left)
	lteField, ok2 := extractFieldName(lteNode.Left)
	if !ok1 || !ok2 || gteField != lteField {
		return nil, false
	}

	minVal, ok1 := extractValue(gteNode.Right)
	maxVal, ok2 := extractValue(lteNode.Right)
	if !ok1 || !ok2 {
		return nil, false
	}

	return &sift.Condition{
		Name:      gteField,
		Operation: sift.OperationBetween,
		Value:     minVal + "," + maxVal,
	}, true
}

// extractValue converts an expr-lang literal node to its string representation
// suitable for sift.Condition.Value.
func extractValue(node ast.Node) (string, bool) {
	switch n := node.(type) {
	case *ast.StringNode:
		return n.Value, true
	case *ast.IntegerNode:
		return strconv.Itoa(n.Value), true
	case *ast.FloatNode:
		return strconv.FormatFloat(n.Value, 'f', -1, 64), true
	case *ast.BoolNode:
		return strconv.FormatBool(n.Value), true
	case *ast.NilNode:
		return "", false // nil is handled specially in walkComparison
	default:
		return "", false
	}
}

// extractFieldName extracts the field name from an identifier node.
func extractFieldName(node ast.Node) (string, bool) {
	if id, ok := node.(*ast.IdentifierNode); ok {
		return id.Value, true
	}
	return "", false
}

// handleUnsupported either returns an error (strict mode) or wraps the
// node as a RawExpression (lenient mode).
func (w *walker) handleUnsupported(node ast.Node, reason string) (sift.Expression, error) {
	if w.cfg.lenient {
		loc := node.Location()
		raw := w.source[loc.From:loc.To]
		return RawExpression(raw), nil
	}
	return nil, fmt.Errorf("exprlang: unsupported construct: %s", reason)
}
