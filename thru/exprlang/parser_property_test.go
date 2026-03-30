package exprlang

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/expr-lang/expr"
	"github.com/nisimpson/sift"
	"pgregory.net/rapid"
)

// Feature: exprlang-to-sift-parser, Property 1: Format-then-parse round trip
// Validates: Requirements 1.1–1.9, 2.1, 2.3, 2.5, 2.7, 2.8, 3.1, 3.2, 4.1, 4.2, 4.3, 6.1, 6.2, 8.1

// genFieldName generates valid Go identifier field names that won't
// collide with expr-lang keywords.
func genFieldName() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		first := rapid.RuneFrom([]rune("abcdefghijklmnopqrstuvwxyz")).Draw(t, "first")
		restLen := rapid.IntRange(1, 7).Draw(t, "restLen")
		rest := make([]rune, restLen)
		for i := range rest {
			rest[i] = rapid.RuneFrom([]rune("abcdefghijklmnopqrstuvwxyz0123456789")).Draw(t, "rest")
		}
		name := string(first) + string(rest)
		// Avoid expr-lang reserved words
		reserved := map[string]bool{
			"in": true, "not": true, "and": true, "or": true,
			"true": true, "false": true, "nil": true,
			"contains": true, "startsWith": true, "endsWith": true, "matches": true,
			"any": true, "all": true, "filter": true, "map": true,
			"count": true, "none": true, "len": true, "upper": true,
			"lower": true, "trim": true, "split": true,
		}
		if reserved[name] {
			return "f" + name
		}
		return name
	})
}

// genStringValue generates safe string values that won't be reinterpreted
// by the adapter's formatValue as numbers or booleans.
func genStringValue() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		// Use length >= 2 and only lowercase letters to avoid collisions
		// with boolean strings (t, f, true, false, etc.) and numeric strings.
		length := rapid.IntRange(2, 10).Draw(t, "len")
		chars := make([]rune, length)
		for i := range chars {
			chars[i] = rapid.RuneFrom([]rune("abcdeghijklmnopqrsuvwxyz")).Draw(t, "char")
		}
		s := string(chars)
		// Reject values that strconv.ParseBool would accept
		switch s {
		case "true", "false":
			return "xx" + s
		}
		return s
	})
}

// genIntValueStr generates integer value strings.
func genIntValueStr() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		v := rapid.IntRange(0, 9999).Draw(t, "int")
		return fmt.Sprintf("%d", v)
	})
}

// genComparisonCondition generates a Condition with a comparison operation.
func genComparisonCondition() *rapid.Generator[*sift.Condition] {
	return rapid.Custom(func(t *rapid.T) *sift.Condition {
		field := genFieldName().Draw(t, "field")
		opIdx := rapid.IntRange(0, 5).Draw(t, "opIdx")
		ops := []sift.Operation{
			sift.OperationEQ, sift.OperationNEQ,
			sift.OperationLT, sift.OperationLTE,
			sift.OperationGT, sift.OperationGTE,
		}
		op := ops[opIdx]

		// Choose between string and int values
		isString := rapid.Bool().Draw(t, "isString")
		var value string
		if isString {
			value = genStringValue().Draw(t, "strVal")
		} else {
			value = genIntValueStr().Draw(t, "intVal")
		}

		return &sift.Condition{Name: field, Operation: op, Value: value}
	})
}

// genStringOpCondition generates a Condition with contains or startsWith.
func genStringOpCondition() *rapid.Generator[*sift.Condition] {
	return rapid.Custom(func(t *rapid.T) *sift.Condition {
		field := genFieldName().Draw(t, "field")
		value := genStringValue().Draw(t, "value")
		isContains := rapid.Bool().Draw(t, "isContains")
		op := sift.OperationContains
		if !isContains {
			op = sift.OperationBeginsWith
		}
		return &sift.Condition{Name: field, Operation: op, Value: value}
	})
}

// genInCondition generates a Condition with the In operation.
func genInCondition() *rapid.Generator[*sift.Condition] {
	return rapid.Custom(func(t *rapid.T) *sift.Condition {
		field := genFieldName().Draw(t, "field")
		value := genStringValue().Draw(t, "value")
		return &sift.Condition{Name: field, Operation: sift.OperationIn, Value: value}
	})
}

// genExistsCondition generates an Exists or NotExists condition.
func genExistsCondition() *rapid.Generator[*sift.Condition] {
	return rapid.Custom(func(t *rapid.T) *sift.Condition {
		field := genFieldName().Draw(t, "field")
		isExists := rapid.Bool().Draw(t, "isExists")
		op := sift.OperationExists
		if !isExists {
			op = sift.OperationNotExists
		}
		return &sift.Condition{Name: field, Operation: op}
	})
}

// genBetweenCondition generates a Between condition with integer bounds.
func genBetweenCondition() *rapid.Generator[*sift.Condition] {
	return rapid.Custom(func(t *rapid.T) *sift.Condition {
		field := genFieldName().Draw(t, "field")
		min := rapid.IntRange(0, 499).Draw(t, "min")
		max := rapid.IntRange(500, 9999).Draw(t, "max")
		return &sift.Condition{
			Name:      field,
			Operation: sift.OperationBetween,
			Value:     fmt.Sprintf("%d,%d", min, max),
		}
	})
}

// genLeafExpression generates a random leaf sift.Expression (no logical ops).
func genLeafExpression() *rapid.Generator[sift.Expression] {
	return rapid.Custom(func(t *rapid.T) sift.Expression {
		kind := rapid.IntRange(0, 4).Draw(t, "leafKind")
		switch kind {
		case 0:
			return genComparisonCondition().Draw(t, "comparison")
		case 1:
			return genStringOpCondition().Draw(t, "stringOp")
		case 2:
			return genInCondition().Draw(t, "in")
		case 3:
			return genExistsCondition().Draw(t, "exists")
		default:
			return genBetweenCondition().Draw(t, "between")
		}
	})
}

// genExpression generates a random sift.Expression tree with bounded depth.
func genExpression(maxDepth int) *rapid.Generator[sift.Expression] {
	return rapid.Custom(func(t *rapid.T) sift.Expression {
		if maxDepth <= 0 {
			return genLeafExpression().Draw(t, "leaf")
		}

		// At depth > 0, choose between leaf and logical ops
		kind := rapid.IntRange(0, 6).Draw(t, "exprKind")
		switch {
		case kind <= 3:
			// Leaf expression (higher probability to keep trees manageable)
			return genLeafExpression().Draw(t, "leaf")
		case kind == 4:
			// AND
			left := genExpression(maxDepth - 1).Draw(t, "andLeft")
			right := genExpression(maxDepth - 1).Draw(t, "andRight")
			return &sift.AndOperation{Left: left, Right: right}
		case kind == 5:
			// OR
			left := genExpression(maxDepth - 1).Draw(t, "orLeft")
			right := genExpression(maxDepth - 1).Draw(t, "orRight")
			return &sift.OrOperation{Left: left, Right: right}
		default:
			// NOT
			child := genExpression(maxDepth - 1).Draw(t, "notChild")
			return &sift.NotOperation{Child: child}
		}
	})
}

// formatExpression formats a sift.Expression to an expr-lang string via the Adapter.
func formatExpression(expr sift.Expression) (string, error) {
	adapter := NewAdapter()
	err := sift.Thru(context.Background(), adapter, sift.WithFilter(expr))
	if err != nil {
		return "", err
	}
	return adapter.Expression(), nil
}

func TestProperty_FormatThenParseRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random expression tree with max depth 3
		original := genExpression(3).Draw(t, "expr")

		// Format to expr-lang string
		formatted, err := formatExpression(original)
		if err != nil {
			t.Fatalf("format failed: %v", err)
		}

		// Parse back
		parsed, err := Parse(formatted)
		if err != nil {
			t.Fatalf("parse(%q) failed: %v", formatted, err)
		}

		// Compare AST equivalence
		if !reflect.DeepEqual(original, parsed) {
			t.Fatalf("round-trip mismatch:\noriginal:  %+v\nformatted: %s\nparsed:    %+v", original, formatted, parsed)
		}
	})
}

// Feature: exprlang-to-sift-parser, Property 2: Parse-then-format evaluation equivalence
// Validates: Requirements 8.2

// fieldInfo tracks the expected type for a field name based on the operation it appears in.
type fieldInfo struct {
	name string
	kind string // "int", "string", "exists", "in", "between"
}

// collectFields walks a sift.Expression tree and collects field names with their expected types.
func collectFields(expr sift.Expression) []fieldInfo {
	var fields []fieldInfo
	collectFieldsRecursive(expr, &fields)
	return fields
}

func collectFieldsRecursive(expr sift.Expression, fields *[]fieldInfo) {
	switch e := expr.(type) {
	case *sift.Condition:
		switch e.Operation {
		case sift.OperationExists, sift.OperationNotExists:
			*fields = append(*fields, fieldInfo{name: e.Name, kind: "exists"})
		case sift.OperationIn:
			*fields = append(*fields, fieldInfo{name: e.Name, kind: "in"})
		case sift.OperationBetween:
			*fields = append(*fields, fieldInfo{name: e.Name, kind: "between"})
		case sift.OperationContains, sift.OperationBeginsWith:
			*fields = append(*fields, fieldInfo{name: e.Name, kind: "string"})
		default:
			// Comparison ops: check if value looks numeric
			if _, err := fmt.Sscanf(e.Value, "%d", new(int)); err == nil {
				*fields = append(*fields, fieldInfo{name: e.Name, kind: "int"})
			} else {
				*fields = append(*fields, fieldInfo{name: e.Name, kind: "string"})
			}
		}
	case *sift.AndOperation:
		collectFieldsRecursive(e.Left, fields)
		collectFieldsRecursive(e.Right, fields)
	case *sift.OrOperation:
		collectFieldsRecursive(e.Left, fields)
		collectFieldsRecursive(e.Right, fields)
	case *sift.NotOperation:
		collectFieldsRecursive(e.Child, fields)
	}
}

// buildRandomEnv builds a random map[string]interface{} environment for evaluating
// an expression, using the field info to generate appropriate typed values.
func buildRandomEnv(t *rapid.T, fields []fieldInfo) map[string]interface{} {
	env := make(map[string]interface{})
	for _, f := range fields {
		if _, exists := env[f.name]; exists {
			continue // already set by a prior occurrence
		}
		switch f.kind {
		case "int", "between":
			env[f.name] = rapid.IntRange(0, 9999).Draw(t, "envInt_"+f.name)
		case "string":
			env[f.name] = genStringValue().Draw(t, "envStr_"+f.name)
		case "in":
			// expr-lang "in" checks membership in a collection; provide a []interface{} of strings
			n := rapid.IntRange(1, 5).Draw(t, "envInLen_"+f.name)
			slice := make([]interface{}, n)
			for i := range slice {
				slice[i] = genStringValue().Draw(t, fmt.Sprintf("envInVal_%s_%d", f.name, i))
			}
			env[f.name] = slice
		case "exists":
			// Randomly include or exclude the field
			include := rapid.Bool().Draw(t, "envExists_"+f.name)
			if include {
				env[f.name] = genStringValue().Draw(t, "envExistsVal_"+f.name)
			} else {
				delete(env, f.name)
			}
		}
	}
	return env
}

func TestProperty_ParseThenFormatEvaluationEquivalence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// 1. Generate a random expression tree
		original := genExpression(3).Draw(t, "expr")

		// 2. Format to expr-lang string (string1)
		str1, err := formatExpression(original)
		if err != nil {
			t.Fatalf("format original failed: %v", err)
		}

		// 3. Parse string1 back to sift AST
		parsed, err := Parse(str1)
		if err != nil {
			t.Fatalf("parse(%q) failed: %v", str1, err)
		}

		// 4. Format parsed AST to string2
		str2, err := formatExpression(parsed)
		if err != nil {
			t.Fatalf("format parsed failed: %v", err)
		}

		// 5. Build random environment with appropriate field values
		fields := collectFields(original)
		env := buildRandomEnv(t, fields)

		// 6. Compile both strings
		prog1, err := expr.Compile(str1, expr.AllowUndefinedVariables(), expr.AsBool())
		if err != nil {
			t.Fatalf("compile str1 %q failed: %v", str1, err)
		}
		prog2, err := expr.Compile(str2, expr.AllowUndefinedVariables(), expr.AsBool())
		if err != nil {
			t.Fatalf("compile str2 %q failed: %v", str2, err)
		}

		// 7. Evaluate both against the random environment
		result1, err := expr.Run(prog1, env)
		if err != nil {
			t.Fatalf("eval str1 %q failed: %v", str1, err)
		}
		result2, err := expr.Run(prog2, env)
		if err != nil {
			t.Fatalf("eval str2 %q failed: %v", str2, err)
		}

		// 8. Verify identical boolean results
		b1, ok1 := result1.(bool)
		b2, ok2 := result2.(bool)
		if !ok1 || !ok2 {
			t.Fatalf("non-boolean results: str1=%v (%T), str2=%v (%T)", result1, result1, result2, result2)
		}
		if b1 != b2 {
			t.Fatalf("evaluation mismatch:\nstr1: %s → %v\nstr2: %s → %v\nenv: %v", str1, b1, str2, b2, env)
		}
	})
}

// Feature: exprlang-to-sift-parser, Property 3: Strict mode rejects unsupported constructs
// Validates: Requirements 5.1–5.7, 9.6

// unsupportedTemplates contains expr-lang patterns that compile successfully
// with AllowUndefinedVariables() but use constructs unsupported by the parser
// in strict mode. Each template uses %s for a random field name and %d for
// a random integer value.
var unsupportedTemplates = []struct {
	// format is a Go fmt template; args are filled by the generator
	format string
	// kind describes the category for documentation
	kind string
}{
	// Function calls (Req 5.1)
	{format: "len(%s) > %d", kind: "function call"},
	{format: "upper(%s) == \"X\"", kind: "function call"},
	{format: "lower(%s) == \"x\"", kind: "function call"},
	// String operators without sift equivalent (Req 5.6)
	{format: "%s endsWith \"val\"", kind: "unsupported string op"},
	{format: "%s matches \"pat\"", kind: "unsupported string op"},
	// Arithmetic (Req 5.3)
	{format: "%s + %d > %d", kind: "arithmetic"},
	{format: "%s * %d > %d", kind: "arithmetic"},
	{format: "%s - %d > %d", kind: "arithmetic"},
	// Array predicates (Req 5.2)
	{format: "any(%s, .x > %d)", kind: "array predicate"},
	{format: "all(%s, .x > %d)", kind: "array predicate"},
	// Ternary (Req 5.4)
	{format: "%s > %d ? \"a\" : \"b\"", kind: "ternary"},
	// Pipe operator (Req 5.5)
	{format: "%s | upper()", kind: "pipe operator"},
}

// genUnsupportedExprString generates a random expr-lang string that contains
// at least one unsupported construct. It picks a template from the list and
// fills in random field names and integer values.
func genUnsupportedExprString() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		idx := rapid.IntRange(0, len(unsupportedTemplates)-1).Draw(t, "templateIdx")
		tmpl := unsupportedTemplates[idx]
		field := genFieldName().Draw(t, "field")
		n1 := rapid.IntRange(1, 100).Draw(t, "n1")
		n2 := rapid.IntRange(1, 100).Draw(t, "n2")

		// Count how many format verbs the template expects
		var result string
		switch strings.Count(tmpl.format, "%") {
		case 1:
			result = fmt.Sprintf(tmpl.format, field)
		case 2:
			result = fmt.Sprintf(tmpl.format, field, n1)
		case 3:
			result = fmt.Sprintf(tmpl.format, field, n1, n2)
		default:
			result = fmt.Sprintf(tmpl.format, field)
		}
		return result
	})
}

func TestProperty_StrictModeRejectsUnsupported(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := genUnsupportedExprString().Draw(t, "input")
		_, err := Parse(input)
		if err == nil {
			t.Fatalf("Parse(%q) expected error in strict mode, got nil", input)
		}
		if !strings.Contains(err.Error(), "unsupported construct") {
			t.Fatalf("Parse(%q) error = %q, want 'unsupported construct' in message", input, err.Error())
		}
	})
}

// Feature: exprlang-to-sift-parser, Property 4: Lenient mode wraps unsupported constructs
// Validates: Requirements 9.1–9.4

func TestProperty_LenientModeWrapsUnsupported(t *testing.T) {
	t.Run("pure unsupported", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			input := genUnsupportedExprString().Draw(t, "input")
			expr, err := Parse(input, WithLenientMode())
			if err != nil {
				t.Fatalf("Parse(%q) in lenient mode unexpected error: %v", input, err)
			}
			if expr == nil {
				t.Fatalf("Parse(%q) in lenient mode returned nil", input)
			}
			// Verify it contains a RawExpression
			if !strings.Contains(expr.String(), "exprlang(") {
				t.Fatalf("Parse(%q) in lenient mode: expected RawExpression wrapping, got %q", input, expr.String())
			}
		})
	})

	t.Run("mixed supported and unsupported", func(t *testing.T) {
		// Use only unsupported templates that produce boolean-typed expressions,
		// so they can be combined with && without expr-lang type errors.
		// Templates like ternary (returns string) and pipe (returns string)
		// cannot be combined with && since expr-lang rejects bool && string.
		boolUnsupportedTemplates := []struct {
			format string
			kind   string
		}{
			{format: "len(%s) > %d", kind: "function call"},
			{format: "%s + %d > %d", kind: "arithmetic"},
			{format: "%s * %d > %d", kind: "arithmetic"},
			{format: "%s - %d > %d", kind: "arithmetic"},
			{format: "any(%s, .x > %d)", kind: "array predicate"},
			{format: "all(%s, .x > %d)", kind: "array predicate"},
		}

		genBoolUnsupported := rapid.Custom(func(t *rapid.T) string {
			idx := rapid.IntRange(0, len(boolUnsupportedTemplates)-1).Draw(t, "templateIdx")
			tmpl := boolUnsupportedTemplates[idx]
			f := genFieldName().Draw(t, "field2")
			n1 := rapid.IntRange(1, 100).Draw(t, "n1")
			n2 := rapid.IntRange(1, 100).Draw(t, "n2")
			switch strings.Count(tmpl.format, "%") {
			case 2:
				return fmt.Sprintf(tmpl.format, f, n1)
			default:
				return fmt.Sprintf(tmpl.format, f, n1, n2)
			}
		})

		rapid.Check(t, func(t *rapid.T) {
			// Generate a mixed expression: supported && unsupported
			field := genFieldName().Draw(t, "field")
			value := genStringValue().Draw(t, "value")
			unsupported := genBoolUnsupported.Draw(t, "unsupported")
			input := fmt.Sprintf("%s == \"%s\" && %s", field, value, unsupported)

			expr, err := Parse(input, WithLenientMode())
			if err != nil {
				t.Fatalf("Parse(%q) in lenient mode unexpected error: %v", input, err)
			}
			if expr == nil {
				t.Fatalf("Parse(%q) in lenient mode returned nil", input)
			}
			// Should be an AndOperation
			andOp, ok := expr.(*sift.AndOperation)
			if !ok {
				t.Fatalf("expected *sift.AndOperation, got %T", expr)
			}
			// Left should be a supported Condition
			_, ok = andOp.Left.(*sift.Condition)
			if !ok {
				t.Fatalf("expected left to be *sift.Condition, got %T", andOp.Left)
			}
			// Right should contain RawExpression
			if !strings.Contains(andOp.Right.String(), "exprlang(") {
				t.Fatalf("expected right to contain RawExpression, got %q", andOp.Right.String())
			}
		})
	})
}
