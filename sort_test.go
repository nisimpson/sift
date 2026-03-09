package sift

import (
	"context"
	"testing"
)

// mockSortAdapter is a test adapter that implements Adapter with sort support.
type mockSortAdapter struct {
	fieldCalled bool
	listCalled  bool
	lastField   *SortField
	lastList    *SortList
}

func newMockSortAdapter() *mockSortAdapter {
	return &mockSortAdapter{}
}

func (m *mockSortAdapter) Evaluator(ctx context.Context) *Evaluator {
	return &Evaluator{
		SortFieldEvaluator: m,
		SortListEvaluator:  m,
	}
}

func (m *mockSortAdapter) EvaluateSortField(ctx context.Context, field *SortField) error {
	m.fieldCalled = true
	m.lastField = field
	return nil
}

func (m *mockSortAdapter) EvaluateSortList(ctx context.Context, list *SortList) error {
	m.listCalled = true
	m.lastList = list
	return nil
}

func TestSortThru_WithSortList(t *testing.T) {
	list := &SortList{
		Fields: []*SortField{
			{Name: "created_at", Direction: SortDesc},
			{Name: "name", Direction: SortAsc},
		},
	}

	adapter := newMockSortAdapter()
	err := Thru(context.Background(), adapter, WithSort(list))

	if err != nil {
		t.Errorf("SortThru() error = %v", err)
	}

	if !adapter.listCalled {
		t.Error("EvaluateSortList should have been called")
	}

	if adapter.lastList != list {
		t.Error("EvaluateSortList should have received the correct list")
	}
}

func TestSortThru_WithSortBuilder(t *testing.T) {
	sort := Sort("created_at", SortDesc).ThenBy("name", SortAsc)

	adapter := newMockSortAdapter()
	err := Thru(context.Background(), adapter, WithSort(sort))

	if err != nil {
		t.Errorf("SortThru() error = %v", err)
	}

	if !adapter.listCalled {
		t.Error("EvaluateSortList should have been called")
	}

	if len(adapter.lastList.Fields) != 2 {
		t.Errorf("EvaluateSortList received %d fields, want 2", len(adapter.lastList.Fields))
	}
}

func TestSortThru_WithSortField(t *testing.T) {
	field := &SortField{Name: "created_at", Direction: SortDesc}

	adapter := newMockSortAdapter()
	err := Thru(context.Background(), adapter, WithSort(field))

	if err != nil {
		t.Errorf("SortThru() error = %v", err)
	}

	if !adapter.fieldCalled {
		t.Error("EvaluateSortField should have been called")
	}

	if adapter.lastField != field {
		t.Error("EvaluateSortField should have received the correct field")
	}
}

func TestSortThru_UnsupportedEvaluator(t *testing.T) {
	adapter := &unsupportedSortAdapter{}
	sort := Sort("created_at", SortDesc)

	err := Thru(context.Background(), adapter, WithSort(sort))
	if err == nil {
		t.Error("SortThru() should return error when sort evaluators are not populated")
	}
}

// unsupportedSortAdapter doesn't support sorting
type unsupportedSortAdapter struct{}

func (u *unsupportedSortAdapter) Evaluator(ctx context.Context) *Evaluator {
	return &Evaluator{
		// No sort evaluators populated
	}
}

func TestSortThru_PartialSupport(t *testing.T) {
	// Adapter that only supports SortList, not SortField
	adapter := &partialSortAdapter{}

	// SortList should work
	list := &SortList{Fields: []*SortField{{Name: "test", Direction: SortAsc}}}
	err := Thru(context.Background(), adapter, WithSort(list))
	if err != nil {
		t.Errorf("SortThru() with SortList error = %v", err)
	}

	// SortField should fail gracefully
	field := &SortField{Name: "test", Direction: SortAsc}
	err = Thru(context.Background(), adapter, WithSort(field))
	if err == nil {
		t.Error("SortThru() with SortField should return error when SortFieldEvaluator is nil")
	}
}

// partialSortAdapter only supports SortList, not SortField
type partialSortAdapter struct{}

func (p *partialSortAdapter) Evaluator(ctx context.Context) *Evaluator {
	return &Evaluator{
		SortListEvaluator: p,
	}
}

func (p *partialSortAdapter) EvaluateSortList(ctx context.Context, list *SortList) error {
	return nil
}
