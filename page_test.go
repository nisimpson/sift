package sift

import (
	"context"
	"testing"
)

func TestPaginate_OffsetBased(t *testing.T) {
	page := Paginate().Size(20).Number(2)
	
	offsetPage, ok := page.(*OffsetPagination)
	if !ok {
		t.Fatalf("Expected *OffsetPagination, got %T", page)
	}
	
	if offsetPage.Size != 20 {
		t.Errorf("Size = %d, want 20", offsetPage.Size)
	}
	
	if offsetPage.Number != 2 {
		t.Errorf("Number = %d, want 2", offsetPage.Number)
	}
}

func TestPaginate_CursorBased(t *testing.T) {
	page := Paginate().Size(20).Cursor("token123")
	
	cursorPage, ok := page.(*CursorPagination)
	if !ok {
		t.Fatalf("Expected *CursorPagination, got %T", page)
	}
	
	if cursorPage.Size != 20 {
		t.Errorf("Size = %d, want 20", cursorPage.Size)
	}
	
	if cursorPage.Cursor != "token123" {
		t.Errorf("Cursor = %s, want token123", cursorPage.Cursor)
	}
}

func TestPaginate_LastOneWins(t *testing.T) {
	// Call Number() then Cursor() - cursor should win
	page1 := Paginate().Size(20).Cursor("token")
	if _, ok := page1.(*CursorPagination); !ok {
		t.Errorf("Expected CursorPagination when Cursor() is called last")
	}
	
	// Call Cursor() then Number() - number should win
	page2 := Paginate().Size(20).Number(3)
	if _, ok := page2.(*OffsetPagination); !ok {
		t.Errorf("Expected OffsetPagination when Number() is called last")
	}
}

// Mock adapter for testing pagination
type mockPaginationAdapter struct {
	offsetPage *OffsetPagination
	cursorPage *CursorPagination
}

func (m *mockPaginationAdapter) Evaluator(ctx context.Context) *Evaluator {
	return &Evaluator{
		OffsetPaginationEvaluator: m,
		CursorPaginationEvaluator: m,
	}
}

func (m *mockPaginationAdapter) EvaluateOffsetPagination(ctx context.Context, page *OffsetPagination) error {
	m.offsetPage = page
	return nil
}

func (m *mockPaginationAdapter) EvaluateCursorPagination(ctx context.Context, page *CursorPagination) error {
	m.cursorPage = page
	return nil
}

func TestThru_WithPagination_Offset(t *testing.T) {
	page := Paginate().Size(20).Number(2)
	adapter := &mockPaginationAdapter{}
	
	err := Thru(context.Background(), adapter, WithPagination(page))
	if err != nil {
		t.Fatalf("Thru() error = %v", err)
	}
	
	if adapter.offsetPage == nil {
		t.Fatal("Expected offsetPage to be set")
	}
	
	if adapter.offsetPage.Size != 20 {
		t.Errorf("Size = %d, want 20", adapter.offsetPage.Size)
	}
	
	if adapter.offsetPage.Number != 2 {
		t.Errorf("Number = %d, want 2", adapter.offsetPage.Number)
	}
}

func TestThru_WithPagination_Cursor(t *testing.T) {
	page := Paginate().Size(20).Cursor("token123")
	adapter := &mockPaginationAdapter{}
	
	err := Thru(context.Background(), adapter, WithPagination(page))
	if err != nil {
		t.Fatalf("Thru() error = %v", err)
	}
	
	if adapter.cursorPage == nil {
		t.Fatal("Expected cursorPage to be set")
	}
	
	if adapter.cursorPage.Size != 20 {
		t.Errorf("Size = %d, want 20", adapter.cursorPage.Size)
	}
	
	if adapter.cursorPage.Cursor != "token123" {
		t.Errorf("Cursor = %s, want token123", adapter.cursorPage.Cursor)
	}
}
