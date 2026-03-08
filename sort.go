package sift

import (
	"context"
	"fmt"
)

// SortDirection represents the direction of sorting.
type SortDirection string

const (
	// SortAsc sorts in ascending order (A-Z, 0-9, oldest-newest).
	SortAsc SortDirection = "asc"
	// SortDesc sorts in descending order (Z-A, 9-0, newest-oldest).
	SortDesc SortDirection = "desc"
)

// SortExpression represents a sorting expression in the AST.
type SortExpression interface {
	accept(ctx context.Context, evaluator *Evaluator) error
}

// SortField represents a single field to sort by.
type SortField struct {
	Name      string
	Direction SortDirection
	NullsLast bool // Control NULL ordering (backend-specific support)
}

// accept implements the visitor pattern for SortField.
func (s *SortField) accept(ctx context.Context, evaluator *Evaluator) error {
	if evaluator.SortFieldEvaluator == nil {
		return fmt.Errorf("%w: SortField not supported", ErrUnsupported)
	}
	return evaluator.SortFieldEvaluator.EvaluateSortField(ctx, s)
}

// ThenBy creates a new SortBuilder starting with this field and adding another field.
// This allows chaining sort operations starting from a SortField.
//
// Example:
//
//	field := &sift.SortField{Name: "created_at", Direction: sift.SortDesc}
//	sort := field.ThenBy("name", sift.SortAsc)
func (s SortField) ThenBy(name string, direction SortDirection) SortBuilder {
	return Sort(s.Name, s.Direction).ThenBy(name, direction)
}

// SortList represents multiple sort fields applied in order.
type SortList struct {
	Fields []*SortField
}

// accept implements the visitor pattern for SortList.
func (s *SortList) accept(ctx context.Context, evaluator *Evaluator) error {
	if evaluator.SortListEvaluator == nil {
		return fmt.Errorf("%w: SortList not supported", ErrUnsupported)
	}
	return evaluator.SortListEvaluator.EvaluateSortList(ctx, s)
}

// SortFieldEvaluator evaluates a single sort field.
type SortFieldEvaluator interface {
	EvaluateSortField(ctx context.Context, field *SortField) error
}

// SortListEvaluator evaluates a list of sort fields.
type SortListEvaluator interface {
	EvaluateSortList(ctx context.Context, list *SortList) error
}

// SortThru traverses a sort expression using the visitor pattern.
// It calls the appropriate evaluator methods on the adapter.
//
// Example:
//
//	sort := sift.Sort("created_at", sift.SortDesc).ThenBy("name", sift.SortAsc)
//	adapter := sql.NewAdapter()
//	err := sift.SortThru(ctx, adapter, sort)
func SortThru(ctx context.Context, adapter Adapter, expr SortExpression) error {
	evaluator := adapter.Evaluator(ctx)
	return expr.accept(ctx, evaluator)
}
