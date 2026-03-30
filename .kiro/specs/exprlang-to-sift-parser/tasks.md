# Implementation Plan: Expr-Lang to Sift Parser

## Overview

Implement a parser that converts expr-lang expression strings into `sift.Expression` AST nodes. The parser uses `expr.Compile()` to obtain an expr-lang AST, then walks it to produce sift nodes. A single new file `thru/exprlang/parser.go` contains all parser logic, with tests in `thru/exprlang/parser_test.go` and a README update for documentation.

## Tasks

- [x] 1. Implement core parser scaffold and comparison operators
  - [x] 1.1 Create `thru/exprlang/parser.go` with public API and internal types
    - Define `ParseOption`, `parserConfig`, `walker` types
    - Implement `Parse(input string, opts ...ParseOption) (sift.Expression, error)` function
    - Implement `WithLenientMode() ParseOption` functional option
    - Call `expr.Compile()` with `expr.AllowUndefinedVariables()` to obtain the AST root
    - Handle empty input and compile errors
    - Implement `walkNode()` dispatch skeleton for `*ast.BinaryNode` and `*ast.UnaryNode`
    - Implement `extractValue()` for string, integer, float, boolean, and nil literals
    - Implement `extractFieldName()` for `*ast.IdentifierNode`
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_

  - [x] 1.2 Implement comparison operator handling in `walkNode`
    - Handle `==`, `!=`, `<`, `<=`, `>`, `>=` binary operators producing `sift.Condition` nodes
    - Detect nil right-hand side for `==` (OperationNotExists) and `!=` (OperationExists)
    - Store unquoted strings, numeric string representations, and boolean string representations as Condition values
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7, 1.8, 1.9, 4.2, 4.3_

  - [x] 1.3 Write unit tests for comparison operators and nil checks
    - Table-driven tests for each comparison operator with string, int, float, and bool values
    - Tests for `field == nil` → OperationNotExists and `field != nil` → OperationExists
    - Tests for empty input and invalid syntax errors
    - _Requirements: 1.1–1.9, 4.2, 4.3, 7.2, 7.3_

- [x] 2. Implement logical operators and string operators
  - [x] 2.1 Implement logical operator handling in `walkNode`
    - Handle `&&` / `and` producing `sift.AndOperation`
    - Handle `||` / `or` producing `sift.OrOperation`
    - Handle `!` / `not` unary operator producing `sift.NotOperation`
    - Recursively walk left/right/child sub-expressions
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 2.8_

  - [x] 2.2 Implement string operator handling in `walkNode`
    - Handle `contains` binary operator producing `sift.Condition{OperationContains}`
    - Handle `startsWith` binary operator producing `sift.Condition{OperationBeginsWith}`
    - _Requirements: 3.1, 3.2_

  - [x] 2.3 Implement membership (`in`) operator handling
    - Handle `in` binary operator producing `sift.Condition{OperationIn}`
    - Note reversed operand order: `"value" in field` → `Condition{Name: field, Value: value}`
    - _Requirements: 4.1_

  - [x] 2.4 Write unit tests for logical, string, and membership operators
    - Table-driven tests for `&&`, `and`, `||`, `or`, `!`, `not`
    - Tests for `contains`, `startsWith`
    - Tests for `"value" in field`
    - Tests for operator precedence and parenthesized grouping
    - _Requirements: 2.1–2.8, 3.1, 3.2, 4.1_

- [x] 3. Implement between detection and unsupported construct handling
  - [x] 3.1 Implement `tryBetween` pattern matcher
    - Check for `field >= min && field <= max` or `field >= min and field <= max` pattern
    - Verify both sides reference the same field name
    - Extract min/max values and produce `sift.Condition{OperationBetween, Value: "min,max"}`
    - Call `tryBetween` before falling through to normal AND handling in `walkNode`
    - _Requirements: 6.1, 6.2_

  - [x] 3.2 Implement `handleUnsupported` for strict and lenient modes
    - In strict mode: return descriptive error with construct type (e.g. `exprlang: unsupported construct: function call (len)`)
    - In lenient mode: wrap unsupported sub-expression as `exprlang.RawExpression` custom sift node
    - Handle unsupported string operators (`endsWith`, `matches`) via `handleUnsupported`
    - Handle function calls, array predicates, arithmetic, ternary, pipe operators via `handleUnsupported`
    - _Requirements: 3.3, 3.4, 3.5, 3.6, 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 9.1, 9.2, 9.3, 9.4, 9.6_

  - [x] 3.3 Write unit tests for between detection
    - Test `field >= min and field <= max` → OperationBetween
    - Test `field >= min && field <= max` → OperationBetween
    - Test that `field > min and field < max` does NOT collapse to between (produces AndOperation)
    - Test between with integer and float values
    - _Requirements: 6.1, 6.2_

  - [x] 3.4 Write unit tests for strict mode rejection
    - Test function calls, array predicates, arithmetic, ternary, pipe operators return errors
    - Test `endsWith` and `matches` return errors with descriptive messages
    - Verify error messages contain the unsupported construct type
    - _Requirements: 5.1–5.7_

  - [x] 3.5 Write unit tests for lenient mode wrapping
    - Test function calls wrapped as RawExpression
    - Test array predicates wrapped as RawExpression
    - Test `endsWith` and `matches` wrapped as RawExpression
    - Test mixed supported/unsupported: `status == "active" && len(items) > 5` → AndOperation with Condition + RawExpression
    - _Requirements: 9.1, 9.2, 9.3, 9.4_

- [x] 4. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 5. Implement property-based tests for correctness properties
  - [x] 5.1 Write property test for format-then-parse round trip
    - **Property 1: Format-then-parse round trip**
    - Generate random `sift.Expression` trees (bounded depth) using only supported operations
    - Format via `Adapter`, parse back via `Parse`, compare AST equivalence
    - Minimum 100 iterations
    - **Validates: Requirements 1.1–1.9, 2.1, 2.3, 2.5, 2.7, 2.8, 3.1, 3.2, 4.1, 4.2, 4.3, 6.1, 6.2, 8.1**

  - [x] 5.2 Write property test for parse-then-format evaluation equivalence
    - **Property 2: Parse-then-format evaluation equivalence**
    - Generate random `sift.Expression` trees, format to string, parse back, format again
    - Compile both strings with expr-lang and evaluate against random input data
    - Verify identical boolean results
    - Minimum 100 iterations
    - **Validates: Requirements 8.2**

  - [x] 5.3 Write property test for strict mode rejection
    - **Property 3: Strict mode rejects unsupported constructs**
    - Generate random expr-lang strings containing unsupported constructs
    - Parse in strict mode and verify non-nil error with descriptive message
    - Minimum 100 iterations
    - **Validates: Requirements 5.1–5.7, 9.6**

  - [x] 5.4 Write property test for lenient mode wrapping
    - **Property 4: Lenient mode wraps unsupported constructs**
    - Generate random expr-lang strings with unsupported constructs (possibly mixed with supported)
    - Parse in lenient mode and verify non-nil result without error
    - Verify unsupported sub-expressions are wrapped as RawExpression nodes
    - Minimum 100 iterations
    - **Validates: Requirements 9.1–9.4**

- [x] 6. Update README documentation
  - [x] 6.1 Update `thru/exprlang/README.md` with parser documentation
    - Document the `Parse` function signature and basic usage examples
    - Describe strict mode (default) and lenient mode with `WithLenientMode()` option
    - Include a note about the future custom function registry possibility
    - Add examples for parsing comparison, logical, string, membership, between, and nil-check expressions
    - _Requirements: 10.1, 10.2, 10.3_

- [x] 7. Final checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties from the design document
- Unit tests validate specific examples and edge cases
- The parser reuses existing types (`RawExpression`, `CustomExpression`) from `custom.go` — no new sift types needed
