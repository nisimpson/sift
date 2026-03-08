// Package jsonapi provides a JSON:API filter adapter for the sift filter library.
// It translates between JSON:API query parameters and sift filter expressions.
//
// JSON:API filter format:
//
//	?filter[q]=and(p1,p2)&filter[p1]=eq(status,active)&filter[p2]=gt(age,18)
//
// The adapter extracts all leaf conditions as parameters (p1, p2, etc.) and
// represents the logical structure in the main query (filter[q]).
package jsonapi

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/nisimpson/sift"
)

// Adapter translates between JSON:API query parameters and sift expressions.
type Adapter struct {
	query      string            // Main query expression (e.g., "and(p1,p2)")
	params     map[string]string // Parameter definitions (e.g., {"p1": "eq(status,active)"})
	paramCount int               // Counter for generating parameter names
}

// NewAdapter creates a new JSON:API adapter.
func NewAdapter() *Adapter {
	return &Adapter{
		params: make(map[string]string),
	}
}

// Parse parses JSON:API query parameters into a sift expression.
// Expected format: filter[q]=and(p1,p2)&filter[p1]=eq(status,active)&filter[p2]=gt(age,18)
func (a *Adapter) Parse(values url.Values) (sift.Expression, error) {
	// Extract filter parameters
	filterParams := make(map[string]string)
	for key, vals := range values {
		if strings.HasPrefix(key, "filter[") && strings.HasSuffix(key, "]") {
			paramName := key[7 : len(key)-1] // Extract name between filter[ and ]
			if len(vals) > 0 {
				filterParams[paramName] = vals[0]
			}
		}
	}

	// Get main query
	mainQuery, ok := filterParams["q"]
	if !ok {
		return nil, fmt.Errorf("missing filter[q] parameter")
	}

	// Parse the main query, resolving parameter references
	return a.parseExpression(mainQuery, filterParams)
}

// parseExpression recursively parses a filter expression string.
func (a *Adapter) parseExpression(expr string, params map[string]string) (sift.Expression, error) {
	expr = strings.TrimSpace(expr)

	// Check if this is a parameter reference (e.g., "p1")
	if !strings.Contains(expr, "(") {
		// It's a parameter reference, resolve it
		paramExpr, ok := params[expr]
		if !ok {
			return nil, fmt.Errorf("undefined parameter: %s", expr)
		}
		return a.parseExpression(paramExpr, params)
	}

	// Find the operation name and arguments
	openParen := strings.Index(expr, "(")
	if openParen == -1 {
		return nil, fmt.Errorf("invalid expression: %s", expr)
	}

	operation := expr[:openParen]
	argsStr := expr[openParen+1:]

	// Find matching closing parenthesis
	if !strings.HasSuffix(argsStr, ")") {
		return nil, fmt.Errorf("missing closing parenthesis: %s", expr)
	}
	argsStr = argsStr[:len(argsStr)-1]

	// Parse based on operation type
	switch operation {
	case "and":
		return a.parseLogicalOp(argsStr, params, true)
	case "or":
		return a.parseLogicalOp(argsStr, params, false)
	case "not":
		child, err := a.parseExpression(strings.TrimSpace(argsStr), params)
		if err != nil {
			return nil, err
		}
		return &sift.NotOperation{Child: child}, nil
	default:
		// Check if it's a known condition operation
		if a.isConditionOperation(operation) {
			return a.parseCondition(operation, argsStr)
		}
		// Otherwise, try to parse as a custom expression using sift.Parse
		customExpr, err := sift.Parse(expr)
		if err != nil {
			return nil, fmt.Errorf("unknown operation or invalid custom expression: %s", operation)
		}
		return customExpr, nil
	}
}

// parseLogicalOp parses and/or operations.
func (a *Adapter) parseLogicalOp(argsStr string, params map[string]string, isAnd bool) (sift.Expression, error) {
	args, err := a.splitArgs(argsStr)
	if err != nil {
		return nil, err
	}

	if len(args) < 2 {
		return nil, fmt.Errorf("logical operation requires at least 2 arguments")
	}

	// Parse first two arguments
	left, err := a.parseExpression(args[0], params)
	if err != nil {
		return nil, err
	}

	right, err := a.parseExpression(args[1], params)
	if err != nil {
		return nil, err
	}

	// Create the operation
	var result sift.Expression
	if isAnd {
		result = &sift.AndOperation{Left: left, Right: right}
	} else {
		result = &sift.OrOperation{Left: left, Right: right}
	}

	// Chain additional arguments
	for i := 2; i < len(args); i++ {
		next, err := a.parseExpression(args[i], params)
		if err != nil {
			return nil, err
		}

		if isAnd {
			result = &sift.AndOperation{Left: result, Right: next}
		} else {
			result = &sift.OrOperation{Left: result, Right: next}
		}
	}

	return result, nil
}

// isConditionOperation checks if the operation name is a known condition operation.
func (a *Adapter) isConditionOperation(operation string) bool {
	switch operation {
	case "eq", "neq", "lt", "lte", "gt", "gte", "contains", "beginsWith", "in", "between", "exists", "notExists":
		return true
	default:
		return false
	}
}

// parseCondition parses a condition operation.
func (a *Adapter) parseCondition(operation, argsStr string) (sift.Expression, error) {
	args, err := a.splitArgs(argsStr)
	if err != nil {
		return nil, err
	}

	// Map operation name to sift.Operation
	var op sift.Operation
	switch operation {
	case "eq":
		op = sift.OperationEQ
	case "neq":
		op = sift.OperationNEQ
	case "lt":
		op = sift.OperationLT
	case "lte":
		op = sift.OperationLTE
	case "gt":
		op = sift.OperationGT
	case "gte":
		op = sift.OperationGTE
	case "contains":
		op = sift.OperationContains
	case "beginsWith":
		op = sift.OperationBeginsWith
	case "in":
		op = sift.OperationIn
	case "between":
		op = sift.OperationBetween
	case "exists":
		op = sift.OperationExists
		if len(args) != 1 {
			return nil, fmt.Errorf("exists operation requires 1 argument")
		}
		return &sift.Condition{Name: args[0], Operation: op}, nil
	case "notExists":
		op = sift.OperationNotExists
		if len(args) != 1 {
			return nil, fmt.Errorf("notExists operation requires 1 argument")
		}
		return &sift.Condition{Name: args[0], Operation: op}, nil
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}

	// Most operations require 2 arguments (name, value)
	if len(args) != 2 {
		return nil, fmt.Errorf("%s operation requires 2 arguments", operation)
	}

	return &sift.Condition{
		Name:      args[0],
		Operation: op,
		Value:     args[1],
	}, nil
}

// splitArgs splits comma-separated arguments, respecting nested parentheses.
func (a *Adapter) splitArgs(argsStr string) ([]string, error) {
	var args []string
	var current strings.Builder
	depth := 0
	inQuotes := false

	for i, ch := range argsStr {
		switch ch {
		case '"':
			inQuotes = !inQuotes
			current.WriteRune(ch)
		case '(':
			if !inQuotes {
				depth++
			}
			current.WriteRune(ch)
		case ')':
			if !inQuotes {
				depth--
				if depth < 0 {
					return nil, fmt.Errorf("unmatched closing parenthesis at position %d", i)
				}
			}
			current.WriteRune(ch)
		case ',':
			if !inQuotes && depth == 0 {
				args = append(args, strings.TrimSpace(current.String()))
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	if inQuotes {
		return nil, fmt.Errorf("unclosed quote in arguments")
	}
	if depth != 0 {
		return nil, fmt.Errorf("unmatched parentheses")
	}

	// Add the last argument
	if current.Len() > 0 {
		args = append(args, strings.TrimSpace(current.String()))
	}

	return args, nil
}

// Format formats a sift expression into JSON:API query parameters.
func (a *Adapter) Format(expr sift.Expression) (url.Values, error) {
	a.params = make(map[string]string)
	a.paramCount = 0

	// Build the query and extract parameters
	query, err := a.formatExpression(expr)
	if err != nil {
		return nil, err
	}

	a.query = query

	// Build url.Values
	values := url.Values{}
	values.Set("filter[q]", a.query)
	for name, value := range a.params {
		values.Set(fmt.Sprintf("filter[%s]", name), value)
	}

	return values, nil
}

// formatExpression recursively formats a sift expression.
func (a *Adapter) formatExpression(expr sift.Expression) (string, error) {
	switch e := expr.(type) {
	case *sift.Condition:
		return a.formatCondition(e)
	case *sift.AndOperation:
		return a.formatLogicalOp(e.Left, e.Right, "and")
	case *sift.OrOperation:
		return a.formatLogicalOp(e.Left, e.Right, "or")
	case *sift.NotOperation:
		child, err := a.formatExpression(e.Child)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("not(%s)", child), nil
	default:
		// Try to format as a custom expression using sift.Format
		// This handles customNodeWrapper and other custom types
		formatted, err := sift.Format(expr)
		if err != nil {
			return "", fmt.Errorf("unsupported expression type: %T", expr)
		}
		// Register as a parameter
		a.paramCount++
		paramName := fmt.Sprintf("p%d", a.paramCount)
		a.params[paramName] = formatted
		return paramName, nil
	}
}

// formatCondition formats a condition and registers it as a parameter.
func (a *Adapter) formatCondition(cond *sift.Condition) (string, error) {
	a.paramCount++
	paramName := fmt.Sprintf("p%d", a.paramCount)

	// Format the condition
	var condStr string
	switch cond.Operation {
	case sift.OperationEQ:
		condStr = fmt.Sprintf("eq(%s,%s)", cond.Name, cond.Value)
	case sift.OperationNEQ:
		condStr = fmt.Sprintf("neq(%s,%s)", cond.Name, cond.Value)
	case sift.OperationLT:
		condStr = fmt.Sprintf("lt(%s,%s)", cond.Name, cond.Value)
	case sift.OperationLTE:
		condStr = fmt.Sprintf("lte(%s,%s)", cond.Name, cond.Value)
	case sift.OperationGT:
		condStr = fmt.Sprintf("gt(%s,%s)", cond.Name, cond.Value)
	case sift.OperationGTE:
		condStr = fmt.Sprintf("gte(%s,%s)", cond.Name, cond.Value)
	case sift.OperationContains:
		condStr = fmt.Sprintf("contains(%s,%s)", cond.Name, cond.Value)
	case sift.OperationBeginsWith:
		condStr = fmt.Sprintf("beginsWith(%s,%s)", cond.Name, cond.Value)
	case sift.OperationIn:
		condStr = fmt.Sprintf("in(%s,%s)", cond.Name, cond.Value)
	case sift.OperationBetween:
		condStr = fmt.Sprintf("between(%s,%s)", cond.Name, cond.Value)
	case sift.OperationExists:
		condStr = fmt.Sprintf("exists(%s)", cond.Name)
	case sift.OperationNotExists:
		condStr = fmt.Sprintf("notExists(%s)", cond.Name)
	default:
		return "", fmt.Errorf("unsupported operation: %s", cond.Operation)
	}

	a.params[paramName] = condStr
	return paramName, nil
}

// formatLogicalOp formats a logical operation (and/or).
func (a *Adapter) formatLogicalOp(left, right sift.Expression, op string) (string, error) {
	leftStr, err := a.formatExpression(left)
	if err != nil {
		return "", err
	}

	rightStr, err := a.formatExpression(right)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s(%s,%s)", op, leftStr, rightStr), nil
}

// Query returns the main query expression (e.g., "and(p1,p2)").
func (a *Adapter) Query() string {
	return a.query
}

// Parameters returns the parameter definitions.
func (a *Adapter) Parameters() map[string]string {
	return a.params
}

// QueryString returns the complete query string.
func (a *Adapter) QueryString() string {
	values, _ := a.Format(nil) // This won't work, need to store the expression
	return values.Encode()
}
