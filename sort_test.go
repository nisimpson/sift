package sift

import (
	"context"
	"testing"
)

// mockSortAdapter is a test adapter that implements SortAdapter.
type mockSortAdapter struct {
	fieldCalled bool
	listCalled  bool
	lastField   *SortField
	lastList    *SortList
}

func newMockSortAdapter() *mockSortAdapter {
	return &mockSortAdapter{}
}

func (m *mockSortAdapter) SortEvaluator(ctx context.Context) *SortEvaluator {
	return &SortEvaluator{
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

func TestSort(t *testing.T) {
	sort := Sort("created_at", SortDesc)

	if len(sort.fields) != 1 {
		t.Errorf("Sort() fields length = %d, want 1", len(sort.fields))
	}

	field := sort.fields[0]
	if field.Name != "created_at" {
		t.Errorf("Sort() field name = %s, want created_at", field.Name)
	}
	if field.Direction != SortDesc {
		t.Errorf("Sort() direction = %s, want %s", field.Direction, SortDesc)
	}
}

func TestSortBuilder_ThenBy(t *testing.T) {
	sort := Sort("created_at", SortDesc).ThenBy("name", SortAsc)

	if len(sort.fields) != 2 {
		t.Errorf("ThenBy() fields length = %d, want 2", len(sort.fields))
	}

	if sort.fields[0].Name != "created_at" {
		t.Errorf("ThenBy() first field name = %s, want created_at", sort.fields[0].Name)
	}
	if sort.fields[1].Name != "name" {
		t.Errorf("ThenBy() second field name = %s, want name", sort.fields[1].Name)
	}
}

func TestSortBuilder_NullsLast(t *testing.T) {
	sort := Sort("email", SortAsc).NullsLast()

	if len(sort.fields) != 1 {
		t.Fatalf("NullsLast() fields length = %d, want 1", len(sort.fields))
	}

	if !sort.fields[0].NullsLast {
		t.Error("NullsLast() should set NullsLast flag to true")
	}
}

func TestSortBuilder_NullsLastOnMultipleFields(t *testing.T) {
	sort := Sort("created_at", SortDesc).
		ThenBy("email", SortAsc).NullsLast().
		ThenBy("name", SortAsc)

	if len(sort.fields) != 3 {
		t.Fatalf("fields length = %d, want 3", len(sort.fields))
	}

	// Only the email field should have NullsLast set
	if sort.fields[0].NullsLast {
		t.Error("created_at should not have NullsLast set")
	}
	if !sort.fields[1].NullsLast {
		t.Error("email should have NullsLast set")
	}
	if sort.fields[2].NullsLast {
		t.Error("name should not have NullsLast set")
	}
}

func TestSortBuilder_Build(t *testing.T) {
	builder := Sort("created_at", SortDesc).ThenBy("name", SortAsc)
	list := builder.Build()

	if len(list.Fields) != 2 {
		t.Errorf("Build() fields length = %d, want 2", len(list.Fields))
	}

	if list.Fields[0].Name != "created_at" {
		t.Errorf("Build() first field name = %s, want created_at", list.Fields[0].Name)
	}
	if list.Fields[1].Name != "name" {
		t.Errorf("Build() second field name = %s, want name", list.Fields[1].Name)
	}
}

func TestSortThru_WithSortList(t *testing.T) {
	list := &SortList{
		Fields: []*SortField{
			{Name: "created_at", Direction: SortDesc},
			{Name: "name", Direction: SortAsc},
		},
	}

	adapter := newMockSortAdapter()
	err := SortThru(context.Background(), adapter, list)

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
	err := SortThru(context.Background(), adapter, sort)

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
	err := SortThru(context.Background(), adapter, field)

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
	// Adapter that doesn't support sorting
	type unsupportedAdapter struct{}
	
	// This test just documents that unsupportedAdapter doesn't implement SortAdapter
	// which provides compile-time safety
	_ = &unsupportedAdapter{}
}

func TestSortThru_PartialSupport(t *testing.T) {
	// Adapter that only supports SortList, not SortField
	adapter := &partialSortAdapter{}
	
	// SortList should work
	list := &SortList{Fields: []*SortField{{Name: "test", Direction: SortAsc}}}
	err := SortThru(context.Background(), adapter, list)
	if err != nil {
		t.Errorf("SortThru() with SortList error = %v", err)
	}
	
	// SortField should fail gracefully
	field := &SortField{Name: "test", Direction: SortAsc}
	err = SortThru(context.Background(), adapter, field)
	if err == nil {
		t.Error("SortThru() with SortField should return error when SortFieldEvaluator is nil")
	}
}

// partialSortAdapter only supports SortList, not SortField
type partialSortAdapter struct{}

func (p *partialSortAdapter) SortEvaluator(ctx context.Context) *SortEvaluator {
	return &SortEvaluator{
		SortListEvaluator: p,
	}
}

func (p *partialSortAdapter) EvaluateSortList(ctx context.Context, list *SortList) error {
	return nil
}

func TestSortDirection_Constants(t *testing.T) {
	if SortAsc != "asc" {
		t.Errorf("SortAsc = %s, want asc", SortAsc)
	}
	if SortDesc != "desc" {
		t.Errorf("SortDesc = %s, want desc", SortDesc)
	}
}

func TestSortBuilder_ChainedCalls(t *testing.T) {
	// Test that all methods return *SortBuilder for chaining
	sort := Sort("a", SortAsc).
		ThenBy("b", SortDesc).
		NullsLast().
		ThenBy("c", SortAsc)

	if len(sort.fields) != 3 {
		t.Errorf("Chained calls resulted in %d fields, want 3", len(sort.fields))
	}

	// Verify the chain worked correctly
	if sort.fields[0].Name != "a" || sort.fields[0].Direction != SortAsc {
		t.Error("First field incorrect")
	}
	if sort.fields[1].Name != "b" || sort.fields[1].Direction != SortDesc || !sort.fields[1].NullsLast {
		t.Error("Second field incorrect")
	}
	if sort.fields[2].Name != "c" || sort.fields[2].Direction != SortAsc {
		t.Error("Third field incorrect")
	}
}
