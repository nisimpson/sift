# Requirements Document

## Introduction

This feature adds the reverse direction to the `thru/exprlang` adapter package: parsing an expr-lang expression string into a `sift.Expression` AST. The existing adapter converts `sift.Expression → expr-lang string`; this feature adds `expr-lang string → sift.Expression`. The primary use case is accepting filter expressions from URL query parameters in expr-lang syntax (e.g. `?filter=status == "active" && age > 18`) and converting them into the sift AST so they can be forwarded to any backend adapter (DynamoDB, SQL, etc.).

Only the subset of expr-lang that maps to sift operations is supported. By default, unsupported constructs are rejected with descriptive errors rather than silently wrapped, because downstream backends cannot execute arbitrary expr-lang. However, a lenient mode option allows unsupported constructs to be wrapped as `exprlang.RawExpression` custom sift nodes, letting downstream adapters decide whether to accept or reject them.

## Glossary

- **Parser**: The function or component in the `thru/exprlang` package that accepts an expr-lang expression string and produces a `sift.Expression` AST.
- **Sift_AST**: The tree of `sift.Expression` nodes (`Condition`, `AndOperation`, `OrOperation`, `NotOperation`) representing a filter.
- **Expr_Lang_String**: A string written in expr-lang syntax (e.g. `status == "active" && age > 18`).
- **Formatter**: The existing `Adapter` in `thru/exprlang` that converts a `sift.Expression` into an Expr_Lang_String (the forward direction).
- **Supported_Subset**: The set of expr-lang constructs that have a direct mapping to sift operations: comparison operators (`==`, `!=`, `<`, `<=`, `>`, `>=`), logical operators in both symbolic and keyword forms (`&&` / `and`, `||` / `or`, `!` / `not`), string operators (`contains`, `startsWith`), membership (`in`), nil checks (`field != nil`, `field == nil`), and range checks (mapped to `between`). Note: expr-lang also defines `endsWith` and `matches` string operators, but sift has no corresponding operations; these are treated as Unsupported_Constructs.
- **Unsupported_Construct**: Any expr-lang construct outside the Supported_Subset, including but not limited to: string operators without a sift equivalent (`endsWith`, `matches`), function calls (e.g. `len()`, `upper()`), array predicates (`any()`, `all()`, `filter()`), arithmetic expressions, ternary operators, and pipe operators.
- **Strict_Mode**: The default parsing mode in which Unsupported_Constructs cause the Parser to return a descriptive error.
- **Lenient_Mode**: An optional parsing mode in which Unsupported_Constructs are wrapped as `exprlang.RawExpression` custom sift nodes instead of returning errors, allowing downstream adapters to decide whether to accept or reject them.
- **Parse_Option**: A functional option (Go idiomatic `func(*config)` pattern) passed to the Parser to configure its behavior (e.g. enabling Lenient_Mode).

## Requirements

### Requirement 1: Parse Comparison Operators

**User Story:** As a developer, I want to parse expr-lang comparison expressions into sift Conditions, so that simple field comparisons from query parameters become sift AST nodes.

#### Acceptance Criteria

1. WHEN an Expr_Lang_String containing `field == value` is provided, THE Parser SHALL produce a `sift.Condition` with `OperationEQ`, the field name, and the value.
2. WHEN an Expr_Lang_String containing `field != value` is provided, THE Parser SHALL produce a `sift.Condition` with `OperationNEQ`, the field name, and the value.
3. WHEN an Expr_Lang_String containing `field < value` is provided, THE Parser SHALL produce a `sift.Condition` with `OperationLT`, the field name, and the value.
4. WHEN an Expr_Lang_String containing `field <= value` is provided, THE Parser SHALL produce a `sift.Condition` with `OperationLTE`, the field name, and the value.
5. WHEN an Expr_Lang_String containing `field > value` is provided, THE Parser SHALL produce a `sift.Condition` with `OperationGT`, the field name, and the value.
6. WHEN an Expr_Lang_String containing `field >= value` is provided, THE Parser SHALL produce a `sift.Condition` with `OperationGTE`, the field name, and the value.
7. WHEN the value in a comparison is a quoted string (e.g. `"active"`), THE Parser SHALL store the unquoted string as the Condition value.
8. WHEN the value in a comparison is a numeric literal (integer or float), THE Parser SHALL store the numeric string representation as the Condition value.
9. WHEN the value in a comparison is a boolean literal (`true` or `false`), THE Parser SHALL store the boolean string representation as the Condition value.

### Requirement 2: Parse Logical Operators

**User Story:** As a developer, I want to parse expr-lang logical operators into sift logical operations, so that compound filter expressions are correctly represented in the AST.

#### Acceptance Criteria

1. WHEN an Expr_Lang_String containing `expr1 && expr2` is provided, THE Parser SHALL produce a `sift.AndOperation` with the left and right sub-expressions.
2. WHEN an Expr_Lang_String containing `expr1 and expr2` is provided, THE Parser SHALL produce a `sift.AndOperation` with the left and right sub-expressions (keyword alias for `&&`).
3. WHEN an Expr_Lang_String containing `expr1 || expr2` is provided, THE Parser SHALL produce a `sift.OrOperation` with the left and right sub-expressions.
4. WHEN an Expr_Lang_String containing `expr1 or expr2` is provided, THE Parser SHALL produce a `sift.OrOperation` with the left and right sub-expressions (keyword alias for `||`).
5. WHEN an Expr_Lang_String containing `!(expr)` is provided, THE Parser SHALL produce a `sift.NotOperation` with the child expression.
6. WHEN an Expr_Lang_String containing `not expr` is provided, THE Parser SHALL produce a `sift.NotOperation` with the child expression (keyword alias for `!`).
7. WHEN an Expr_Lang_String contains parenthesized grouping (e.g. `(a == 1) && (b == 2)`), THE Parser SHALL respect the grouping and produce the correct AST structure.
8. WHEN an Expr_Lang_String contains mixed `&&`/`and` and `||`/`or` operators, THE Parser SHALL respect expr-lang operator precedence (`&&`/`and` binds tighter than `||`/`or`).

### Requirement 3: Parse String Operators

**User Story:** As a developer, I want to parse expr-lang string operators into sift string conditions, so that `contains` and `startsWith` expressions are supported, and unsupported string operators (`endsWith`, `matches`) are handled correctly.

#### Acceptance Criteria

1. WHEN an Expr_Lang_String containing `field contains "value"` is provided, THE Parser SHALL produce a `sift.Condition` with `OperationContains`, the field name, and the unquoted value.
2. WHEN an Expr_Lang_String containing `field startsWith "value"` is provided, THE Parser SHALL produce a `sift.Condition` with `OperationBeginsWith`, the field name, and the unquoted value.
3. WHILE the Parser is operating in Strict_Mode, WHEN an Expr_Lang_String contains `field endsWith "value"`, THE Parser SHALL return a descriptive error indicating that `endsWith` has no corresponding sift operation.
4. WHILE the Parser is operating in Lenient_Mode, WHEN an Expr_Lang_String contains `field endsWith "value"`, THE Parser SHALL wrap the expression as an `exprlang.RawExpression` custom sift node.
5. WHILE the Parser is operating in Strict_Mode, WHEN an Expr_Lang_String contains `field matches "pattern"`, THE Parser SHALL return a descriptive error indicating that `matches` (regex) has no corresponding sift operation.
6. WHILE the Parser is operating in Lenient_Mode, WHEN an Expr_Lang_String contains `field matches "pattern"`, THE Parser SHALL wrap the expression as an `exprlang.RawExpression` custom sift node.

### Requirement 4: Parse Membership and Nil Checks

**User Story:** As a developer, I want to parse expr-lang `in` and nil-check expressions into sift conditions, so that membership tests and existence checks are supported.

#### Acceptance Criteria

1. WHEN an Expr_Lang_String containing `"value" in field` is provided, THE Parser SHALL produce a `sift.Condition` with `OperationIn`, the field name, and the value.
2. WHEN an Expr_Lang_String containing `field != nil` is provided, THE Parser SHALL produce a `sift.Condition` with `OperationExists` and the field name.
3. WHEN an Expr_Lang_String containing `field == nil` is provided, THE Parser SHALL produce a `sift.Condition` with `OperationNotExists` and the field name.

### Requirement 5: Reject Unsupported Constructs (Strict Mode — Default)

**User Story:** As a developer, I want the parser to reject expr-lang constructs that have no sift equivalent by default, so that I get clear errors instead of silent failures at the backend layer.

#### Acceptance Criteria

1. WHILE the Parser is operating in Strict_Mode (the default), WHEN an Expr_Lang_String contains a function call (e.g. `len(items) > 5`), THE Parser SHALL return a descriptive error indicating the construct is unsupported.
2. WHILE the Parser is operating in Strict_Mode, WHEN an Expr_Lang_String contains an array predicate (e.g. `any(items, .price > 10)`), THE Parser SHALL return a descriptive error indicating the construct is unsupported.
3. WHILE the Parser is operating in Strict_Mode, WHEN an Expr_Lang_String contains arithmetic expressions (e.g. `price * quantity > 100`), THE Parser SHALL return a descriptive error indicating the construct is unsupported.
4. WHILE the Parser is operating in Strict_Mode, WHEN an Expr_Lang_String contains a ternary or conditional expression, THE Parser SHALL return a descriptive error indicating the construct is unsupported.
5. WHILE the Parser is operating in Strict_Mode, WHEN an Expr_Lang_String contains a pipe operator or chained method call, THE Parser SHALL return a descriptive error indicating the construct is unsupported.
6. WHILE the Parser is operating in Strict_Mode, WHEN an Expr_Lang_String contains a string operator without a sift equivalent (e.g. `field endsWith "value"` or `field matches "pattern"`), THE Parser SHALL return a descriptive error indicating the operator is unsupported.
7. THE Parser SHALL include the unsupported construct type in the error message to aid debugging.

### Requirement 6: Parse Between Expressions

**User Story:** As a developer, I want to parse range-check patterns into sift `between` conditions, so that expressions like `age >= 18 and age <= 65` map to `OperationBetween`.

#### Acceptance Criteria

1. WHEN an Expr_Lang_String contains a pattern `field >= min and field <= max` where both sides reference the same field, THE Parser SHALL produce a `sift.Condition` with `OperationBetween`, the field name, and the value formatted as `"min,max"`.
2. WHEN an Expr_Lang_String contains a pattern `field >= min && field <= max` where both sides reference the same field, THE Parser SHALL produce a `sift.Condition` with `OperationBetween`, the field name, and the value formatted as `"min,max"`.

### Requirement 7: Public API and Integration

**User Story:** As a developer, I want a clean public function in the `thru/exprlang` package to parse expr-lang strings, so that I can integrate it into HTTP handlers and other entry points.

#### Acceptance Criteria

1. THE Parser SHALL be exposed as a public function in the `thru/exprlang` package that accepts an Expr_Lang_String, zero or more Parse_Options, and returns a `sift.Expression` and an error.
2. WHEN the input Expr_Lang_String is empty, THE Parser SHALL return an error indicating empty input.
3. WHEN the input Expr_Lang_String is syntactically invalid expr-lang, THE Parser SHALL return an error from the expr-lang compiler with position information.
4. THE Parser SHALL use `expr.Compile()` from the `github.com/expr-lang/expr` library to parse the Expr_Lang_String into an expr-lang AST, then walk that AST to produce the Sift_AST.
5. THE Parser SHALL accept Parse_Options using the Go functional options pattern (`func(*parserConfig)`) to configure behavior such as enabling Lenient_Mode.

### Requirement 8: Round-Trip Property

**User Story:** As a developer, I want the parser and formatter to be inverses of each other for the Supported_Subset, so that I can trust that no information is lost during conversion.

#### Acceptance Criteria

1. FOR ALL valid Sift_AST expressions composed only of Supported_Subset operations, formatting to an Expr_Lang_String via the Formatter and then parsing back via the Parser SHALL produce a semantically equivalent Sift_AST.
2. FOR ALL valid Expr_Lang_Strings composed only of Supported_Subset constructs, parsing to a Sift_AST via the Parser and then formatting back via the Formatter SHALL produce an Expr_Lang_String that, when compiled by expr-lang, evaluates identically to the original.

### Requirement 9: Lenient Mode — Wrap Unsupported Constructs as RawExpressions

**User Story:** As a developer, I want an option to wrap unsupported expr-lang constructs as `exprlang.RawExpression` custom sift nodes instead of failing, so that downstream adapters capable of handling raw expr-lang (e.g. an in-memory evaluator) can process them while adapters that cannot (e.g. DynamoDB) can reject them on their own terms.

#### Acceptance Criteria

1. WHEN the Parser is configured with the Lenient_Mode Parse_Option, THE Parser SHALL wrap Unsupported_Constructs as `exprlang.RawExpression` sift custom nodes instead of returning errors.
2. WHEN the Parser is in Lenient_Mode and an Expr_Lang_String contains a function call (e.g. `len(items) > 5`), THE Parser SHALL produce a sift custom node wrapping the unsupported sub-expression as an `exprlang.RawExpression`.
3. WHEN the Parser is in Lenient_Mode and an Expr_Lang_String contains an array predicate (e.g. `any(items, .price > 10)`), THE Parser SHALL produce a sift custom node wrapping the unsupported sub-expression as an `exprlang.RawExpression`.
4. WHEN the Parser is in Lenient_Mode and an Expr_Lang_String mixes supported and unsupported constructs (e.g. `status == "active" && len(items) > 5`), THE Parser SHALL produce a sift `AndOperation` where the supported side is a `sift.Condition` and the unsupported side is an `exprlang.RawExpression` custom node.
5. WHEN a downstream adapter receives a sift custom node produced by Lenient_Mode, THE downstream adapter SHALL be able to inspect the node type and decide whether to evaluate or reject the expression.
6. WHERE the Lenient_Mode Parse_Option is not provided, THE Parser SHALL default to Strict_Mode behavior as defined in Requirement 5.

### Requirement 10: README Documentation

**User Story:** As a developer, I want the `thru/exprlang/README.md` to document the new Parse function, Strict and Lenient modes, and the future custom function registry possibility, so that consumers of the package can discover and understand the parsing capability.

#### Acceptance Criteria

1. THE README SHALL document the public Parse function, its signature, and basic usage examples.
2. THE README SHALL describe Strict_Mode (default) and Lenient_Mode, including how to enable Lenient_Mode via Parse_Options.
3. THE README SHALL include a note about the future custom function registry possibility described in the Future Considerations section of this document.

## Future Considerations

The following items are explicitly out of scope for the initial implementation but represent natural follow-up work:

- **Custom Function Registry**: A custom function registry for the Parser is a natural follow-up. This would allow consumers to register handlers that map specific expr-lang constructs (e.g. `endsWith`, custom functions) to structured sift nodes at parse time, rather than relying on post-processing RawExpressions from Lenient Mode.
