package sift

import (
	"context"
	"fmt"
)

// PaginationExpression represents a pagination expression in the AST.
type PaginationExpression interface {
	accept(ctx context.Context, evaluator *Evaluator) error
}

// OffsetPagination represents offset-based pagination (page number + size).
// This is commonly used with SQL databases using LIMIT/OFFSET.
type OffsetPagination struct {
	Size   int // Number of items per page
	Number int // Page number (1-based)
}

// accept implements the visitor pattern for OffsetPagination.
func (o *OffsetPagination) accept(ctx context.Context, evaluator *Evaluator) error {
	if evaluator.OffsetPaginationEvaluator == nil {
		return fmt.Errorf("%w: OffsetPagination not supported", ErrUnsupported)
	}
	return evaluator.OffsetPaginationEvaluator.EvaluateOffsetPagination(ctx, o)
}

// CursorPagination represents cursor-based pagination (cursor + size).
// This is commonly used with DynamoDB, GraphQL, and other cursor-based APIs.
type CursorPagination struct {
	Size   int    // Number of items per page
	Cursor string // Opaque cursor token for the next page
}

// accept implements the visitor pattern for CursorPagination.
func (c *CursorPagination) accept(ctx context.Context, evaluator *Evaluator) error {
	if evaluator.CursorPaginationEvaluator == nil {
		return fmt.Errorf("%w: CursorPagination not supported", ErrUnsupported)
	}
	return evaluator.CursorPaginationEvaluator.EvaluateCursorPagination(ctx, c)
}

// OffsetPaginationEvaluator evaluates offset-based pagination.
type OffsetPaginationEvaluator interface {
	EvaluateOffsetPagination(ctx context.Context, page *OffsetPagination) error
}

// CursorPaginationEvaluator evaluates cursor-based pagination.
type CursorPaginationEvaluator interface {
	EvaluateCursorPagination(ctx context.Context, page *CursorPagination) error
}
