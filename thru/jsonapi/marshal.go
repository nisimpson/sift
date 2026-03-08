package jsonapi

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/nisimpson/sift"
)

// Parse parses JSON:API query parameters into a sift expression.
// Expected format: filter[q]=and(p1,p2)&filter[p1]=eq(status,active)&filter[p2]=gt(age,18)
//
// The main query is in filter[q] and references parameters defined in filter[p1], filter[p2], etc.
// If registry is nil, no custom expressions are supported.
func Parse(query url.Values, registry *sift.Registry) (sift.Expression, error) {
	// Extract filter parameters
	filterParams := make(map[string]string)
	for key, vals := range query {
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

	// Resolve parameter references and parse
	resolved := resolveParameters(mainQuery, filterParams)
	return sift.Parse(resolved, registry)
}

// resolveParameters recursively replaces parameter references with their definitions.
func resolveParameters(expr string, params map[string]string) string {
	// Simple approach: replace parameter references that appear as standalone tokens
	// This handles cases like "and(p1,p2)" -> "and(eq(status,active),gt(age,18))"

	result := expr
	for paramName, paramValue := range params {
		if paramName == "q" {
			continue // Skip the main query
		}

		// Replace parameter references
		// Look for the parameter as a complete token (not part of another word)
		result = replaceParameter(result, paramName, paramValue)
	}

	return result
}

// replaceParameter replaces a parameter reference with its value.
// It ensures we only replace complete tokens, not substrings.
func replaceParameter(expr, paramName, paramValue string) string {
	var result strings.Builder
	i := 0

	for i < len(expr) {
		// Check if we're at a potential parameter reference
		if i+len(paramName) <= len(expr) && expr[i:i+len(paramName)] == paramName {
			// Check if it's a complete token (not part of another identifier)
			before := i == 0 || !isIdentifierChar(expr[i-1])
			after := i+len(paramName) >= len(expr) || !isIdentifierChar(expr[i+len(paramName)])

			if before && after {
				// It's a complete token, replace it
				// Recursively resolve in case the parameter value contains other parameters
				result.WriteString(paramValue)
				i += len(paramName)
				continue
			}
		}

		result.WriteByte(expr[i])
		i++
	}

	return result.String()
}

// isIdentifierChar returns true if the character can be part of an identifier.
func isIdentifierChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_'
}

// Format formats a sift expression into JSON:API query parameters.
// It extracts all leaf conditions as parameters and builds a main query that references them.
// If registry is nil, no custom expressions are supported.
//
// Example:
//
//	Input: and(eq(status,active),gt(age,18))
//	Output: filter[q]=and(p1,p2)&filter[p1]=eq(status,active)&filter[p2]=gt(age,18)
func Format(expr sift.Expression, registry *sift.Registry) (url.Values, error) {
	extractor := &paramExtractor{
		params:   make(map[string]string),
		registry: registry,
	}

	query, err := extractor.extract(expr)
	if err != nil {
		return nil, err
	}

	// Build url.Values
	values := url.Values{}
	values.Set("filter[q]", query)
	for name, value := range extractor.params {
		values.Set(fmt.Sprintf("filter[%s]", name), value)
	}

	return values, nil
}

// paramExtractor extracts leaf conditions as parameters and builds a query structure.
type paramExtractor struct {
	params     map[string]string
	paramCount int
	registry   *sift.Registry
}

// extract recursively processes an expression, extracting parameters.
func (e *paramExtractor) extract(expr sift.Expression) (string, error) {
	switch n := expr.(type) {
	case *sift.Condition:
		return e.extractCondition(n)
	case *sift.AndOperation:
		return e.extractAnd(n)
	case *sift.OrOperation:
		return e.extractOr(n)
	case *sift.NotOperation:
		return e.extractNot(n)
	default:
		// For custom expressions or other types, format them and create a parameter
		formatted, err := sift.Format(expr, e.registry)
		if err != nil {
			return "", err
		}
		return e.createParam(formatted), nil
	}
}

// extractCondition creates a parameter for a condition.
func (e *paramExtractor) extractCondition(cond *sift.Condition) (string, error) {
	// Format the condition using sift.Format
	formatted, err := sift.Format(cond, e.registry)
	if err != nil {
		return "", err
	}

	return e.createParam(formatted), nil
}

// extractAnd processes an AND operation.
func (e *paramExtractor) extractAnd(and *sift.AndOperation) (string, error) {
	left, err := e.extract(and.Left)
	if err != nil {
		return "", err
	}

	right, err := e.extract(and.Right)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("and(%s,%s)", left, right), nil
}

// extractOr processes an OR operation.
func (e *paramExtractor) extractOr(or *sift.OrOperation) (string, error) {
	left, err := e.extract(or.Left)
	if err != nil {
		return "", err
	}

	right, err := e.extract(or.Right)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("or(%s,%s)", left, right), nil
}

// extractNot processes a NOT operation.
func (e *paramExtractor) extractNot(not *sift.NotOperation) (string, error) {
	child, err := e.extract(not.Child)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("not(%s)", child), nil
}

// createParam creates a new parameter and returns its reference.
func (e *paramExtractor) createParam(value string) string {
	e.paramCount++
	paramName := fmt.Sprintf("p%d", e.paramCount)
	e.params[paramName] = value
	return paramName
}
