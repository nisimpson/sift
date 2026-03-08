package exprlang

import (
	"fmt"
	"strings"

	"github.com/nisimpson/sift"
)

// CustomExpression represents an expr-lang specific expression that doesn't
// map directly to standard Sift operations.
type CustomExpression struct {
	expr string
}

// NewCustomExpression creates a new custom expr-lang expression.
// The expression string should be valid expr-lang syntax.
func NewCustomExpression(expr string) sift.Expression {
	return sift.NewCustomExpression(&CustomExpression{expr: expr})
}

// Type returns the type identifier for this custom expression.
func (c *CustomExpression) Type() string {
	return "exprlang"
}

// String returns the string representation of the custom expression.
func (c *CustomExpression) String() string {
	return fmt.Sprintf("exprlang(%s)", c.expr)
}

// Expression returns the raw expr-lang expression string.
func (c *CustomExpression) Expression() string {
	return c.expr
}

// Formatter implements sift.CustomFormatter for expr-lang custom expressions.
type Formatter struct{}

// FormatCustomExpression serializes an expr-lang custom expression to a string.
func (f Formatter) FormatCustomExpression(expression sift.CustomExpression) (string, error) {
	custom, ok := expression.(*CustomExpression)
	if !ok {
		return "", fmt.Errorf("expected *exprlang.CustomExpression, got %T", expression)
	}
	// Only escape commas and closing parentheses that would interfere with parsing
	// Parentheses within the expression are part of expr-lang syntax and should be preserved
	escaped := escapeExpression(custom.expr)
	return fmt.Sprintf("exprlang(%s)", escaped), nil
}

// ParseCustomExpression deserializes a string to an expr-lang custom expression.
func (f Formatter) ParseCustomExpression(p *sift.Parser) (sift.CustomExpression, error) {
	// We need to read the entire expr-lang expression, which may contain parentheses.
	// Since ReadValue() stops at unescaped closing parens, we need a custom approach.
	// We'll read everything as a single value and rely on proper escaping.
	
	// For now, use ReadValue() which handles escaped commas correctly
	// Users must escape commas in their expr-lang expressions when serializing
	expr := p.ReadValue()
	if expr == "" {
		return nil, fmt.Errorf("expected expr-lang expression")
	}

	// Unescape the expression
	unescaped := unescapeExpression(expr)

	p.SkipWhitespace()
	if !p.Expect(')') {
		return nil, fmt.Errorf("expected ')' after expr-lang expression")
	}

	return &CustomExpression{expr: unescaped}, nil
}

// escapeExpression escapes Sift delimiter characters in expr-lang expressions.
// We must escape backslashes, commas, and parentheses so they don't interfere with Sift parsing.
func escapeExpression(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}

// unescapeExpression unescapes Sift delimiter characters.
func unescapeExpression(s string) string {
	s = strings.ReplaceAll(s, "\\)", ")")
	s = strings.ReplaceAll(s, "\\(", "(")
	s = strings.ReplaceAll(s, "\\,", ",")
	s = strings.ReplaceAll(s, "\\\\", "\\")
	return s
}

// init registers the expr-lang custom expression formatter.
func init() {
	sift.RegisterCustomExpression(&CustomExpression{}, Formatter{})
}

// Predicate creates a custom expression using expr-lang predicate syntax.
// Example: Predicate("len(.Content) > 240")
func Predicate(expr string) sift.Expression {
	return NewCustomExpression(expr)
}

// ArrayFunction creates a custom expression for array functions.
// Example: ArrayFunction("filter", "tweets", "len(.Content) > 240")
func ArrayFunction(fn, array, predicate string) sift.Expression {
	return NewCustomExpression(fmt.Sprintf("%s(%s, %s)", fn, array, predicate))
}

// StringFunction creates a custom expression for string functions.
// Example: StringFunction("upper", "name")
func StringFunction(fn string, args ...string) sift.Expression {
	return NewCustomExpression(fmt.Sprintf("%s(%s)", fn, strings.Join(args, ", ")))
}

// DateFunction creates a custom expression for date functions.
// Example: DateFunction("now")
func DateFunction(fn string, args ...string) sift.Expression {
	if len(args) == 0 {
		return NewCustomExpression(fmt.Sprintf("%s()", fn))
	}
	return NewCustomExpression(fmt.Sprintf("%s(%s)", fn, strings.Join(args, ", ")))
}

// RawExpression creates a custom expression from raw expr-lang syntax.
// Use this for any expr-lang expression that doesn't fit other helpers.
func RawExpression(expr string) sift.Expression {
	return NewCustomExpression(expr)
}
