package sift

import (
	"fmt"
	"strings"
)

// CustomFormatter handles serialization and deserialization of [CustomExpression] types.
type CustomFormatter interface {
	// FormatCustomExpression serializes a custom expression to a string.
	FormatCustomExpression(expression CustomExpression) (string, error)

	// ParseCustomExpression deserializes a string to a [CustomExpression].
	// The parser has already consumed the function name and opening parenthesis.
	// The parser should consume up to and including the closing parenthesis.
	ParseCustomExpression(p *Parser) (CustomExpression, error)
}

// Registry holds custom expression formatters.
// Use NewRegistry to create a registry and Register to add formatters.
type Registry struct {
	formatters map[string]CustomFormatter
}

// NewRegistry creates a new custom expression registry.
func NewRegistry() *Registry {
	return &Registry{
		formatters: make(map[string]CustomFormatter),
	}
}

// Register registers a custom formatter by name.
// The name is what appears in the serialized format (e.g., "size", "geo_within").
// Panics if a formatter with the same name is already registered.
//
// Example:
//
//	registry := sift.NewRegistry()
//	registry.Register("size", SizeFormatter{})
func (r *Registry) Register(name string, formatter CustomFormatter) {
	if _, exists := r.formatters[name]; exists {
		panic(fmt.Sprintf("sift: formatter already registered for name: %s", name))
	}
	r.formatters[name] = formatter
}

// Format serializes a filter expression to a string using prefix notation.
// The format is URL-safe and unambiguous, suitable for query strings.
//
// If registry is nil, no custom expressions are supported.
//
// Supported operations:
//
//	eq(field,value)          // equal
//	ne(field,value)          // not equal
//	lt(field,value)          // less than
//	le(field,value)          // less than or equal
//	gt(field,value)          // greater than
//	ge(field,value)          // greater than or equal
//	contains(field,value)    // substring match
//	begins_with(field,value) // prefix match
//	in(field,value)          // value in list
//	exists(field)            // field exists
//	not_exists(field)        // field doesn't exist
//	between(field,value)     // value between bounds
//	and(expr1,expr2)         // logical AND
//	or(expr1,expr2)          // logical OR
//	not(expr)                // logical NOT
//
// Custom expressions registered in the registry are also supported.
//
// Special characters (commas, parentheses, backslashes) in values are escaped with backslash.
//
// Example:
//
//	registry := sift.NewRegistry()
//	registry.Register("size", SizeFormatter{})
//	str, _ := sift.Format(expr, registry)
//	// Output: and(eq(status,active),size(tags,gt,5))
func Format(expr Expression, registry *Registry) (string, error) {
	return format(expr, registry)
}

func format(expr Expression, registry *Registry) (string, error) {
	switch n := expr.(type) {
	case *Condition:
		return formatCondition(n)
	case *AndOperation:
		return formatAnd(n, registry)
	case *OrOperation:
		return formatOr(n, registry)
	case *NotOperation:
		return formatNot(n, registry)
	case ExpressionBuilder:
		// Unwrap the builder and format the inner expression
		return format(n.expr, registry)
	case *customNodeWrapper:
		// Unwrap and format the custom expression
		return formatCustom(n.node, registry)
	default:
		return "", fmt.Errorf("unsupported expression type: %T", expr)
	}
}

// formatCustom formats a custom expression using the registry.
func formatCustom(n CustomExpression, registry *Registry) (string, error) {
	if registry == nil {
		return "", fmt.Errorf("no registry provided for custom expression type: %s", n.Type())
	}

	key := n.Type()
	formatter, ok := registry.formatters[key]
	if !ok {
		return "", fmt.Errorf("no formatter registered for custom expression type: %s", key)
	}
	return formatter.FormatCustomExpression(n)
}

// formatCondition converts a Condition expression to prefix notation.
func formatCondition(e *Condition) (string, error) {
	// For most operations, the operation name is the function name
	switch e.Operation {
	case OperationExists:
		return fmt.Sprintf("exists(%s)", e.Name), nil
	case OperationNotExists:
		return fmt.Sprintf("not_exists(%s)", e.Name), nil
	case OperationEQ, OperationNEQ, OperationLT, OperationLTE, OperationGT, OperationGTE,
		OperationContains, OperationBeginsWith, OperationIn, OperationBetween:
		// Escape commas and parentheses in values
		escapedValue := escapeValue(e.Value)
		return fmt.Sprintf("%s(%s,%s)", e.Operation, e.Name, escapedValue), nil
	default:
		return "", ErrorOperationNotSupported(string(e.Operation))
	}
}

// formatAnd converts an AndOperation expression to prefix notation.
func formatAnd(a *AndOperation, registry *Registry) (string, error) {
	left, err := format(a.Left, registry)
	if err != nil {
		return "", err
	}
	right, err := format(a.Right, registry)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("and(%s,%s)", left, right), nil
}

// formatOr converts an OrOperation expression to prefix notation.
func formatOr(o *OrOperation, registry *Registry) (string, error) {
	left, err := format(o.Left, registry)
	if err != nil {
		return "", err
	}
	right, err := format(o.Right, registry)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("or(%s,%s)", left, right), nil
}

// formatNot converts a NotOperation expression to prefix notation.
func formatNot(n *NotOperation, registry *Registry) (string, error) {
	child, err := format(n.Child, registry)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("not(%s)", child), nil
}

// escapeValue escapes special characters in values.
// Backslashes, commas, and parentheses are escaped with backslash.
func escapeValue(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}

// Parse deserializes a string into a filter expression using prefix notation.
// The input should be in the format produced by [Format]().
//
// If registry is nil, no custom expressions are supported.
//
// Example:
//
//	registry := sift.NewRegistry()
//	registry.Register("size", SizeFormatter{})
//	expr, _ := sift.Parse("and(eq(status,active),size(tags,gt,5))", registry)
func Parse(s string, registry *Registry) (Expression, error) {
	p := &Parser{
		input:    s,
		pos:      0,
		registry: registry,
	}
	return p.parse()
}

// Parser is a recursive descent parser for prefix notation filter expressions.
// It is exported to allow [CustomFormatter] parsers to access parsing utilities.
type Parser struct {
	input    string    // input string to parse
	pos      int       // current position in input
	registry *Registry // custom expression registry
}

// parse is the main parsing entry point. It reads a function call and dispatches
// to the appropriate parsing method based on the function name.
func (p *Parser) parse() (Expression, error) {
	p.SkipWhitespace()
	if p.pos >= len(p.input) {
		return nil, fmt.Errorf("unexpected end of input")
	}

	// Read function name
	fn := p.readIdentifier()
	if fn == "" {
		return nil, fmt.Errorf("expected function name at position %d", p.pos)
	}

	p.SkipWhitespace()
	if !p.Expect('(') {
		return nil, fmt.Errorf("expected '(' after function name at position %d", p.pos)
	}

	op := Operation(fn)
	if op.IsCondition() {
		return p.parseCondition(op)
	} else if op.IsExistence() {
		return p.parseExistence(op)
	}

	switch fn {
	case "and":
		return p.parseAnd()
	case "or":
		return p.parseOr()
	case "not":
		return p.parseNot()
	default:
		// Try custom expression parsers
		return p.parseCustom(fn)
	}
}

// parseCustom attempts to parse a custom expression using the registry.
func (p *Parser) parseCustom(fn string) (Expression, error) {
	if p.registry == nil {
		return nil, fmt.Errorf("unknown function: %s (no registry provided)", fn)
	}

	formatter, ok := p.registry.formatters[fn]
	if !ok {
		return nil, fmt.Errorf("unknown function: %s", fn)
	}

	custom, err := formatter.ParseCustomExpression(p)
	if err != nil {
		return nil, err
	}

	return NewCustomExpression(custom), nil
}

// parseCondition parses a comparison condition: fn(field,value)
func (p *Parser) parseCondition(op Operation) (Expression, error) {
	// Read field name
	field := p.ReadValue()
	if field == "" {
		return nil, fmt.Errorf("expected field name at position %d", p.pos)
	}

	p.SkipWhitespace()
	if !p.Expect(',') {
		return nil, fmt.Errorf("expected ',' after field name at position %d", p.pos)
	}
	p.SkipWhitespace()

	// Read value
	value := p.ReadValue()

	p.SkipWhitespace()
	if !p.Expect(')') {
		return nil, fmt.Errorf("expected ')' at position %d", p.pos)
	}

	// Function name is the operation
	return &Condition{
		Name:      field,
		Operation: op,
		Value:     value,
	}, nil
}

// parseExistence parses an existence check: exists(field) or not_exists(field)
func (p *Parser) parseExistence(op Operation) (Expression, error) {
	field := p.ReadValue()
	if field == "" {
		return nil, fmt.Errorf("expected field name at position %d", p.pos)
	}

	p.SkipWhitespace()
	if !p.Expect(')') {
		return nil, fmt.Errorf("expected ')' at position %d", p.pos)
	}

	return &Condition{
		Name:      field,
		Operation: op,
		Value:     "",
	}, nil
}

// parseAnd parses a logical AND: and(expr1,expr2)
func (p *Parser) parseAnd() (Expression, error) {
	left, err := p.parse()
	if err != nil {
		return nil, err
	}

	p.SkipWhitespace()
	if !p.Expect(',') {
		return nil, fmt.Errorf("expected ',' in and() at position %d", p.pos)
	}
	p.SkipWhitespace()

	right, err := p.parse()
	if err != nil {
		return nil, err
	}

	p.SkipWhitespace()
	if !p.Expect(')') {
		return nil, fmt.Errorf("expected ')' at position %d", p.pos)
	}

	return &AndOperation{Left: left, Right: right}, nil
}

// parseOr parses a logical OR: or(expr1,expr2)
func (p *Parser) parseOr() (Expression, error) {
	left, err := p.parse()
	if err != nil {
		return nil, err
	}

	p.SkipWhitespace()
	if !p.Expect(',') {
		return nil, fmt.Errorf("expected ',' in or() at position %d", p.pos)
	}
	p.SkipWhitespace()

	right, err := p.parse()
	if err != nil {
		return nil, err
	}

	p.SkipWhitespace()
	if !p.Expect(')') {
		return nil, fmt.Errorf("expected ')' at position %d", p.pos)
	}

	return &OrOperation{Left: left, Right: right}, nil
}

// parseNot parses a logical NOT: not(expr)
func (p *Parser) parseNot() (Expression, error) {
	child, err := p.parse()
	if err != nil {
		return nil, err
	}

	p.SkipWhitespace()
	if !p.Expect(')') {
		return nil, fmt.Errorf("expected ')' at position %d", p.pos)
	}

	return &NotOperation{Child: child}, nil
}

// readIdentifier reads a function name (letters and underscores only).
func (p *Parser) readIdentifier() string {
	start := p.pos
	for p.pos < len(p.input) {
		ch := p.input[p.pos]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_' {
			p.pos++
		} else {
			break
		}
	}
	return p.input[start:p.pos]
}

// ReadValue reads a field name or value, handling escape sequences.
// Stops at unescaped commas or closing parentheses.
// This method is exported for use by custom expression parsers.
func (p *Parser) ReadValue() string {
	var result strings.Builder
	escaped := false

	for p.pos < len(p.input) {
		ch := p.input[p.pos]

		if escaped {
			result.WriteByte(ch)
			escaped = false
			p.pos++
			continue
		}

		if ch == '\\' {
			escaped = true
			p.pos++
			continue
		}

		// Stop at unescaped delimiters
		if ch == ',' || ch == ')' {
			break
		}

		result.WriteByte(ch)
		p.pos++
	}

	return result.String()
}

// SkipWhitespace advances the position past any whitespace characters.
// This method is exported for use by [CustomFormatter] parsers.
func (p *Parser) SkipWhitespace() {
	for p.pos < len(p.input) && p.input[p.pos] == ' ' {
		p.pos++
	}
}

// Expect checks if the current character matches the expected character.
// If it matches, advances the position and returns true.
// This method is exported for use by [CustomFormatter] parsers.
func (p *Parser) Expect(ch byte) bool {
	if p.pos < len(p.input) && p.input[p.pos] == ch {
		p.pos++
		return true
	}
	return false
}
