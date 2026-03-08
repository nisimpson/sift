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
	accept(ctx context.Context, evaluator *SortEvaluator) error
}

// SortField represents a single field to sort by.
type SortField struct {
	Name      string
	Direction SortDirection
	NullsLast bool // Control NULL ordering (backend-specific support)
}

// accept implements the visitor pattern for SortField.
func (s *SortField) accept(ctx context.Context, evaluator *SortEvaluator) error {
	if evaluator.SortFieldEvaluator == nil {
		return fmt.Errorf("%w: SortField not supported", ErrUnsupported)
	}
	return evaluator.SortFieldEvaluator.EvaluateSortField(ctx, s)
}

// SortList represents multiple sort fields applied in order.
type SortList struct {
	Fields []*SortField
}

// accept implements the visitor pattern for SortList.
func (s *SortList) accept(ctx context.Context, evaluator *SortEvaluator) error {
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

// SortEvaluator holds the evaluator interfaces for sort expressions.
// Adapters populate only the interfaces they support.
type SortEvaluator struct {
	SortFieldEvaluator
	SortListEvaluator
}

// SortAdapter is the interface that adapters must implement to support sorting.
type SortAdapter interface {
	// SortEvaluator returns a sort evaluator configured for the adapter.
	SortEvaluator(ctx context.Context) *SortEvaluator
}

// SortThru traverses a sort expression using the visitor pattern.
// It calls the appropriate evaluator methods on the adapter.
//
// Example:
//
//	sort := sift.Sort("created_at", sift.SortDesc).ThenBy("name", sift.SortAsc)
//	adapter := sql.NewAdapter()
//	err := sift.SortThru(ctx, adapter, sort)
func SortThru(ctx context.Context, adapter SortAdapter, expr SortExpression) error {
	evaluator := adapter.SortEvaluator(ctx)
	return expr.accept(ctx, evaluator)
}

// SortBuilder provides a fluent API for building sort expressions.
type SortBuilder struct {
	fields []*SortField
}

// Sort creates a new sort expression with a single field.
//
// Example:
//
//	sort := sift.Sort("created_at", sift.SortDesc)
func Sort(name string, direction SortDirection) *SortBuilder {
	return &SortBuilder{
		fields: []*SortField{{Name: name, Direction: direction}},
	}
}

// ThenBy adds another sort field to the expression.
// Fields are applied in the order they are added.
//
// Example:
//
//	sort := sift.Sort("created_at", sift.SortDesc).ThenBy("name", sift.SortAsc)
func (b *SortBuilder) ThenBy(name string, direction SortDirection) *SortBuilder {
	b.fields = append(b.fields, &SortField{Name: name, Direction: direction})
	return b
}

// NullsLast sets the NullsLast flag on the most recently added field.
// This controls whether NULL values appear last in the sort order.
// Note: Not all backends support this feature.
//
// Example:
//
//	sort := sift.Sort("email", sift.SortAsc).NullsLast()
func (b *SortBuilder) NullsLast() *SortBuilder {
	if len(b.fields) > 0 {
		b.fields[len(b.fields)-1].NullsLast = true
	}
	return b
}

// Build returns the sort expression as a SortList.
// This is useful when you need to pass the expression to functions
// that expect a SortExpression interface.
func (b *SortBuilder) Build() *SortList {
	return &SortList{Fields: b.fields}
}

// accept implements SortExpression for SortBuilder.
// This allows SortBuilder to be used directly without calling Build().
func (b *SortBuilder) accept(ctx context.Context, evaluator *SortEvaluator) error {
	return b.Build().accept(ctx, evaluator)
}
