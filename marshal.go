package sift

import (
	"fmt"
	"strconv"
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

// FormatOption configures what to include in the formatted output.
// It also implements Option so it can be used with Thru().
type FormatOption interface {
	Option // Embed Option interface
	applyFormat(*formatConfig)
}

// formatConfig holds the configuration for formatting.
type formatConfig struct {
	filter     Expression
	sort       SortExpression
	pagination PaginationExpression
	registry   *Registry
}

// WithFilter adds a filter expression.
// It can be used with both Format() and Thru().
//
// Example:
//
//	// With Format
//	str, _ := sift.Format(registry, sift.WithFilter(filterExpr))
//
//	// With Thru
//	sift.Thru(ctx, adapter, sift.WithFilter(filterExpr))
func WithFilter(expr Expression) FormatOption {
	return filterOption{expr: expr}
}

func (f filterOption) applyFormat(cfg *formatConfig) {
	cfg.filter = f.expr
}

// WithSort adds a sort expression.
// It can be used with both Format() and Thru().
//
// Example:
//
//	// With Format
//	str, _ := sift.Format(registry, sift.WithSort(sortExpr))
//
//	// With Thru
//	sift.Thru(ctx, adapter, sift.WithSort(sortExpr))
func WithSort(expr SortExpression) FormatOption {
	return sortOption{expr: expr}
}

func (s sortOption) applyFormat(cfg *formatConfig) {
	cfg.sort = s.expr
}

// WithPagination adds a pagination expression.
// It can be used with both Format() and Thru().
//
// Example:
//
//	// With Format
//	str, _ := sift.Format(registry, sift.WithPagination(pageExpr))
//
//	// With Thru
//	sift.Thru(ctx, adapter, sift.WithPagination(pageExpr))
func WithPagination(expr PaginationExpression) FormatOption {
	return paginationOption{expr: expr}
}

func (p paginationOption) applyFormat(cfg *formatConfig) {
	cfg.pagination = p.expr
}

// Format serializes filter, sort, and pagination expressions to a string using prefix notation.
// The format is URL-safe and unambiguous, suitable for query strings.
//
// If registry is nil, no custom expressions are supported.
//
// Supported filter operations:
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
// Sort format:
//
//	field:asc                // single field ascending
//	field:desc               // single field descending
//	field:asc:nullslast      // with nulls last
//	field1:desc,field2:asc   // multiple fields
//
// Pagination format:
//
//	size:20,number:2         // offset-based (page number)
//	size:20,cursor:token123  // cursor-based
//
// Custom expressions registered in the registry are also supported.
//
// Special characters (commas, parentheses, backslashes, colons) in values are escaped with backslash.
//
// Example:
//
//	str, _ := sift.Format(registry,
//	    sift.WithFilter(filterExpr),
//	    sift.WithSort(sortExpr),
//	    sift.WithPagination(pageExpr))
//	// Output: filter(and(eq(status,active),gt(age,18))),sort(created_at:desc,name:asc),page(size:20,number:2)
func Format(registry *Registry, opts ...FormatOption) (string, error) {
	cfg := &formatConfig{registry: registry}
	for _, opt := range opts {
		opt.applyFormat(cfg)
	}

	var parts []string

	if cfg.filter != nil {
		filterStr, err := formatFilter(cfg.filter, cfg.registry)
		if err != nil {
			return "", err
		}
		parts = append(parts, fmt.Sprintf("filter(%s)", filterStr))
	}

	if cfg.sort != nil {
		sortStr, err := formatSort(cfg.sort)
		if err != nil {
			return "", err
		}
		parts = append(parts, fmt.Sprintf("sort(%s)", sortStr))
	}

	if cfg.pagination != nil {
		pageStr, err := formatPagination(cfg.pagination)
		if err != nil {
			return "", err
		}
		parts = append(parts, fmt.Sprintf("page(%s)", pageStr))
	}

	return strings.Join(parts, ","), nil
}

// FormatFilter serializes a filter expression to a string using prefix notation.
// This is a convenience function for formatting only a filter expression.
//
// Example:
//
//	str, _ := sift.FormatFilter(filterExpr, registry)
//	// Output: and(eq(status,active),gt(age,18))
func FormatFilter(expr Expression, registry *Registry) (string, error) {
	return formatFilter(expr, registry)
}

func formatFilter(expr Expression, registry *Registry) (string, error) {
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
		return formatFilter(n.expr, registry)
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
	left, err := formatFilter(a.Left, registry)
	if err != nil {
		return "", err
	}
	right, err := formatFilter(a.Right, registry)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("and(%s,%s)", left, right), nil
}

// formatOr converts an OrOperation expression to prefix notation.
func formatOr(o *OrOperation, registry *Registry) (string, error) {
	left, err := formatFilter(o.Left, registry)
	if err != nil {
		return "", err
	}
	right, err := formatFilter(o.Right, registry)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("or(%s,%s)", left, right), nil
}

// formatNot converts a NotOperation expression to prefix notation.
func formatNot(n *NotOperation, registry *Registry) (string, error) {
	child, err := formatFilter(n.Child, registry)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("not(%s)", child), nil
}

// formatSort serializes a sort expression to a string.
// Format: field:direction or field:direction:nullslast
// Multiple fields are comma-separated: field1:asc,field2:desc
func formatSort(expr SortExpression) (string, error) {
	switch s := expr.(type) {
	case *SortField:
		return formatSortField(s), nil
	case *SortList:
		return formatSortList(s), nil
	case SortBuilder:
		// Convert builder to list
		list := &SortList{Fields: s.fields}
		return formatSortList(list), nil
	default:
		return "", fmt.Errorf("unsupported sort expression type: %T", expr)
	}
}

// formatSortField formats a single sort field.
func formatSortField(field *SortField) string {
	result := fmt.Sprintf("%s:%s", escapeValue(field.Name), field.Direction)
	if field.NullsLast {
		result += ":nullslast"
	}
	return result
}

// formatSortList formats a list of sort fields.
func formatSortList(list *SortList) string {
	var parts []string
	for _, field := range list.Fields {
		parts = append(parts, formatSortField(field))
	}
	return strings.Join(parts, ",")
}

// formatPagination serializes a pagination expression to a string.
// Format: size:N,number:N (offset) or size:N,cursor:token (cursor)
func formatPagination(expr PaginationExpression) (string, error) {
	switch p := expr.(type) {
	case *OffsetPagination:
		return fmt.Sprintf("size:%d,number:%d", p.Size, p.Number), nil
	case *CursorPagination:
		return fmt.Sprintf("size:%d,cursor:%s", p.Size, escapeValue(p.Cursor)), nil
	default:
		return "", fmt.Errorf("unsupported pagination expression type: %T", expr)
	}
}

// escapeValue escapes special characters in values.
// Backslashes, commas, parentheses, and colons are escaped with backslash.
func escapeValue(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	s = strings.ReplaceAll(s, ":", "\\:")
	return s
}

// Parse deserializes a string into a filter expression using prefix notation.
// The input should be in the format produced by [FormatFilter]().
//
// If registry is nil, no custom expressions are supported.
//
// Example:
//
//	registry := sift.NewRegistry()
//	registry.Register("size", SizeFormatter{})
//	expr, _ := sift.ParseFilter("and(eq(status,active),size(tags,gt,5))", registry)
//
// parseFilter deserializes a string into a filter expression using prefix notation.
func parseFilter(s string, registry *Registry) (Expression, error) {
	p := &Parser{
		input:    s,
		pos:      0,
		registry: registry,
	}
	return p.parse()
}

// Query represents a complete query with filter, sort, and pagination.
type Query struct {
	Filter     Expression
	Sort       SortExpression
	Pagination PaginationExpression
}

// ParseQuery deserializes a complete query string into filter, sort, and pagination expressions.
// The input should be in the format produced by Format() with multiple options.
//
// Format: filter(...),sort(...),page(...)
// Any component can be omitted.
//
// Example:
//
//	query, _ := sift.ParseQuery("filter(eq(status,active)),sort(created_at:desc),page(size:20,number:2)", registry)
//	// query.Filter, query.Sort, query.Pagination are all populated
func ParseQuery(s string, registry *Registry) (*Query, error) {
	if s == "" {
		return &Query{}, nil
	}

	query := &Query{}

	// Split by top-level commas (not inside parentheses)
	parts, err := splitTopLevel(s)
	if err != nil {
		return nil, err
	}

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check what type of expression this is
		switch {
		case strings.HasPrefix(part, "filter("):
			// Extract the filter content
			content := strings.TrimPrefix(part, "filter(")
			content = strings.TrimSuffix(content, ")")

			filter, err := parseFilter(content, registry)
			if err != nil {
				return nil, fmt.Errorf("invalid filter: %w", err)
			}
			query.Filter = filter
		case strings.HasPrefix(part, "sort("):
			// Extract the sort content
			content := strings.TrimPrefix(part, "sort(")
			content = strings.TrimSuffix(content, ")")

			sort, err := parseSort(content)
			if err != nil {
				return nil, fmt.Errorf("invalid sort: %w", err)
			}
			query.Sort = sort
		case strings.HasPrefix(part, "page("):
			// Extract the pagination content
			content := strings.TrimPrefix(part, "page(")
			content = strings.TrimSuffix(content, ")")

			page, err := parsePagination(content)
			if err != nil {
				return nil, fmt.Errorf("invalid pagination: %w", err)
			}
			query.Pagination = page
		default:
			return nil, fmt.Errorf("unknown expression type: %s", part)
		}
	}

	return query, nil
}

// splitTopLevel splits a string by commas at the top level (not inside parentheses).
// This is used to split "filter(...),sort(...),page(...)" into separate parts.
func splitTopLevel(s string) ([]string, error) {
	var parts []string
	var current strings.Builder
	depth := 0
	escaped := false

	for i := 0; i < len(s); i++ {
		ch := s[i]

		if escaped {
			current.WriteByte(ch)
			escaped = false
			continue
		}

		if ch == '\\' {
			escaped = true
			current.WriteByte(ch)
			continue
		}

		if ch == '(' {
			depth++
			current.WriteByte(ch)
			continue
		}

		if ch == ')' {
			depth--
			current.WriteByte(ch)
			continue
		}

		if ch == ',' && depth == 0 {
			// Top-level comma - split here
			parts = append(parts, current.String())
			current.Reset()
			continue
		}

		current.WriteByte(ch)
	}

	if depth != 0 {
		return nil, fmt.Errorf("mismatched parentheses")
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts, nil
}

// parseSort deserializes a string into a sort expression.
// Format: field:direction or field:direction:nullslast
// Multiple fields are comma-separated: field1:asc,field2:desc
func parseSort(s string) (SortExpression, error) {
	if s == "" {
		return nil, fmt.Errorf("empty sort string")
	}

	// Split by commas (respecting escapes)
	fields := splitSortFields(s)

	var sortFields []*SortField
	for _, fieldStr := range fields {
		field, err := parseSortField(fieldStr)
		if err != nil {
			return nil, err
		}
		sortFields = append(sortFields, field)
	}

	if len(sortFields) == 1 {
		return sortFields[0], nil
	}
	return &SortList{Fields: sortFields}, nil
}

// splitSortFields splits a sort string by commas, respecting escape sequences.
// The escape sequences are preserved in the output for later unescaping.
func splitSortFields(s string) []string {
	var fields []string
	var current strings.Builder
	escaped := false

	for i := 0; i < len(s); i++ {
		ch := s[i]

		if escaped {
			// Keep the backslash and the escaped character
			current.WriteByte('\\')
			current.WriteByte(ch)
			escaped = false
			continue
		}

		if ch == '\\' {
			escaped = true
			continue
		}

		if ch == ',' {
			fields = append(fields, current.String())
			current.Reset()
			continue
		}

		current.WriteByte(ch)
	}

	if current.Len() > 0 {
		fields = append(fields, current.String())
	}

	return fields
}

// parseSortField parses a single sort field.
// Format: field:direction or field:direction:nullslast
func parseSortField(s string) (*SortField, error) {
	// Split by colons (respecting escapes)
	parts := splitByColon(s)

	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid sort field format: %s (expected field:direction)", s)
	}

	field := &SortField{
		Name:      unescapeValue(parts[0]),
		Direction: SortDirection(parts[1]),
	}

	// Check for nullslast flag
	if len(parts) > 2 && parts[2] == "nullslast" {
		field.NullsLast = true
	}

	// Validate direction
	if field.Direction != SortAsc && field.Direction != SortDesc {
		return nil, fmt.Errorf("invalid sort direction: %s (expected asc or desc)", field.Direction)
	}

	return field, nil
}

// splitByColon splits a string by colons, respecting escape sequences.
// The escape sequences are preserved in the output for later unescaping.
func splitByColon(s string) []string {
	var parts []string
	var current strings.Builder
	escaped := false

	for i := 0; i < len(s); i++ {
		ch := s[i]

		if escaped {
			// Keep the backslash and the escaped character
			current.WriteByte('\\')
			current.WriteByte(ch)
			escaped = false
			continue
		}

		if ch == '\\' {
			escaped = true
			continue
		}

		if ch == ':' {
			parts = append(parts, current.String())
			current.Reset()
			continue
		}

		current.WriteByte(ch)
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// parsePagination deserializes a string into a pagination expression.
// Format: size:N,number:N (offset) or size:N,cursor:token (cursor)
func parsePagination(s string) (PaginationExpression, error) {
	if s == "" {
		return nil, fmt.Errorf("empty pagination string")
	}

	// Split by commas (respecting escapes)
	parts := splitSortFields(s) // Reuse the same splitting logic

	// Parse into key-value pairs
	params := make(map[string]string)
	for _, part := range parts {
		kv := splitByColon(part)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid pagination parameter: %s", part)
		}
		params[kv[0]] = unescapeValue(kv[1])
	}

	// Get size (required)
	sizeStr, ok := params["size"]
	if !ok {
		return nil, fmt.Errorf("missing required parameter: size")
	}
	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid size value: %s", sizeStr)
	}

	// Check for cursor or number
	if cursor, ok := params["cursor"]; ok {
		return &CursorPagination{
			Size:   size,
			Cursor: cursor,
		}, nil
	}

	if numberStr, ok := params["number"]; ok {
		number, err := strconv.Atoi(numberStr)
		if err != nil {
			return nil, fmt.Errorf("invalid number value: %s", numberStr)
		}
		return &OffsetPagination{
			Size:   size,
			Number: number,
		}, nil
	}

	return nil, fmt.Errorf("missing required parameter: cursor or number")
}

// unescapeValue unescapes special characters in values.
func unescapeValue(s string) string {
	var result strings.Builder
	escaped := false

	for i := 0; i < len(s); i++ {
		ch := s[i]

		if escaped {
			result.WriteByte(ch)
			escaped = false
			continue
		}

		if ch == '\\' {
			escaped = true
			continue
		}

		result.WriteByte(ch)
	}

	return result.String()
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
