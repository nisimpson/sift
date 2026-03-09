package jsonapi

import (
	"net/url"
	"testing"

	"github.com/nisimpson/sift"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		want    string // Expected sift.Format output
		wantErr bool
	}{
		{
			name:    "simple condition",
			query:   "filter[q]=p1&filter[p1]=eq(status,active)",
			want:    "eq(status,active)",
			wantErr: false,
		},
		{
			name:    "and operation",
			query:   "filter[q]=and(p1,p2)&filter[p1]=eq(status,active)&filter[p2]=gt(age,18)",
			want:    "and(eq(status,active),gt(age,18))",
			wantErr: false,
		},
		{
			name:    "or operation",
			query:   "filter[q]=or(p1,p2)&filter[p1]=eq(role,admin)&filter[p2]=eq(role,moderator)",
			want:    "or(eq(role,admin),eq(role,moderator))",
			wantErr: false,
		},
		{
			name:    "not operation",
			query:   "filter[q]=not(p1)&filter[p1]=eq(deleted,true)",
			want:    "not(eq(deleted,true))",
			wantErr: false,
		},
		{
			name:    "nested operations",
			query:   "filter[q]=and(p1,or(p2,p3))&filter[p1]=eq(status,active)&filter[p2]=gt(age,18)&filter[p3]=eq(role,admin)",
			want:    "and(eq(status,active),or(gt(age,18),eq(role,admin)))",
			wantErr: false,
		},
		{
			name:    "missing main query - no filter returned",
			query:   "filter[p1]=eq(status,active)",
			want:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, err := url.ParseQuery(tt.query)
			if err != nil {
				t.Fatalf("Failed to parse query: %v", err)
			}

			query, err := ParseQuery(values, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if query.Filter == nil {
					if tt.want != "" {
						t.Errorf("ParseQuery() returned nil filter, want %v", tt.want)
					}
				} else {
					gotStr, _ := sift.FormatFilter(query.Filter, nil)
					if gotStr != tt.want {
						t.Errorf("ParseQuery() = %v, want %v", gotStr, tt.want)
					}
				}
			}
		})
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		name       string
		expr       sift.Expression
		wantQuery  string
		wantParams map[string]string
		wantErr    bool
	}{
		{
			name: "simple condition",
			expr: &sift.Condition{
				Name:      "status",
				Operation: sift.OperationEQ,
				Value:     "active",
			},
			wantQuery: "p1",
			wantParams: map[string]string{
				"p1": "eq(status,active)",
			},
			wantErr: false,
		},
		{
			name: "and operation",
			expr: &sift.AndOperation{
				Left: &sift.Condition{
					Name:      "status",
					Operation: sift.OperationEQ,
					Value:     "active",
				},
				Right: &sift.Condition{
					Name:      "age",
					Operation: sift.OperationGT,
					Value:     "18",
				},
			},
			wantQuery: "and(p1,p2)",
			wantParams: map[string]string{
				"p1": "eq(status,active)",
				"p2": "gt(age,18)",
			},
			wantErr: false,
		},
		{
			name: "nested operations",
			expr: &sift.AndOperation{
				Left: &sift.Condition{
					Name:      "status",
					Operation: sift.OperationEQ,
					Value:     "active",
				},
				Right: &sift.OrOperation{
					Left: &sift.Condition{
						Name:      "age",
						Operation: sift.OperationGT,
						Value:     "18",
					},
					Right: &sift.Condition{
						Name:      "role",
						Operation: sift.OperationEQ,
						Value:     "admin",
					},
				},
			},
			wantQuery: "and(p1,or(p2,p3))",
			wantParams: map[string]string{
				"p1": "eq(status,active)",
				"p2": "gt(age,18)",
				"p3": "eq(role,admin)",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, err := Format(tt.expr, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("Format() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Check main query
				gotQuery := values.Get("filter[q]")
				if gotQuery != tt.wantQuery {
					t.Errorf("Format() query = %v, want %v", gotQuery, tt.wantQuery)
				}

				// Check parameters
				for name, wantValue := range tt.wantParams {
					gotValue := values.Get("filter[" + name + "]")
					if gotValue != wantValue {
						t.Errorf("Format() param %s = %v, want %v", name, gotValue, wantValue)
					}
				}
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		expr sift.Expression
	}{
		{
			name: "simple condition",
			expr: &sift.Condition{
				Name:      "status",
				Operation: sift.OperationEQ,
				Value:     "active",
			},
		},
		{
			name: "and operation",
			expr: &sift.AndOperation{
				Left: &sift.Condition{
					Name:      "status",
					Operation: sift.OperationEQ,
					Value:     "active",
				},
				Right: &sift.Condition{
					Name:      "age",
					Operation: sift.OperationGT,
					Value:     "18",
				},
			},
		},
		{
			name: "complex nested",
			expr: &sift.AndOperation{
				Left: &sift.Condition{
					Name:      "status",
					Operation: sift.OperationEQ,
					Value:     "active",
				},
				Right: &sift.OrOperation{
					Left: &sift.Condition{
						Name:      "age",
						Operation: sift.OperationGT,
						Value:     "18",
					},
					Right: &sift.Condition{
						Name:      "role",
						Operation: sift.OperationEQ,
						Value:     "admin",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Format to JSON:API
			values, err := Format(tt.expr, nil)
			if err != nil {
				t.Fatalf("Format() error = %v", err)
			}

			// Parse back
			query, err := ParseQuery(values, nil)
			if err != nil {
				t.Fatalf("ParseQuery() error = %v", err)
			}

			// Compare using sift.FormatFilter
			wantStr, _ := sift.FormatFilter(tt.expr, nil)
			gotStr, _ := sift.FormatFilter(query.Filter, nil)

			if wantStr != gotStr {
				t.Errorf("Round-trip failed: got %v, want %v", gotStr, wantStr)
			}
		})
	}
}

// TestParseQuery tests parsing complete JSON:API query parameters
func TestParseQuery(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr bool
		check   func(*testing.T, *sift.Query)
	}{
		{
			name:  "filter only",
			query: "filter[q]=p1&filter[p1]=eq(status,active)",
			check: func(t *testing.T, q *sift.Query) {
				if q.Filter == nil {
					t.Error("Filter should not be nil")
				}
				if q.Sort != nil {
					t.Error("Sort should be nil")
				}
				if q.Pagination != nil {
					t.Error("Pagination should be nil")
				}
			},
		},
		{
			name:  "sort only",
			query: "sort[created_at]=desc&sort[name]=asc",
			check: func(t *testing.T, q *sift.Query) {
				if q.Filter != nil {
					t.Error("Filter should be nil")
				}
				if q.Sort == nil {
					t.Error("Sort should not be nil")
				}
				if q.Pagination != nil {
					t.Error("Pagination should be nil")
				}
			},
		},
		{
			name:  "pagination only - offset",
			query: "page[size]=20&page[number]=2",
			check: func(t *testing.T, q *sift.Query) {
				if q.Filter != nil {
					t.Error("Filter should be nil")
				}
				if q.Sort != nil {
					t.Error("Sort should be nil")
				}
				if q.Pagination == nil {
					t.Error("Pagination should not be nil")
				}
				
				offsetPage, ok := q.Pagination.(*sift.OffsetPagination)
				if !ok {
					t.Errorf("Expected OffsetPagination, got %T", q.Pagination)
				} else {
					if offsetPage.Size != 20 {
						t.Errorf("Size = %d, want 20", offsetPage.Size)
					}
					if offsetPage.Number != 2 {
						t.Errorf("Number = %d, want 2", offsetPage.Number)
					}
				}
			},
		},
		{
			name:  "pagination only - cursor",
			query: "page[size]=20&page[cursor]=token123",
			check: func(t *testing.T, q *sift.Query) {
				if q.Pagination == nil {
					t.Fatal("Pagination should not be nil")
				}
				
				cursorPage, ok := q.Pagination.(*sift.CursorPagination)
				if !ok {
					t.Errorf("Expected CursorPagination, got %T", q.Pagination)
				} else {
					if cursorPage.Size != 20 {
						t.Errorf("Size = %d, want 20", cursorPage.Size)
					}
					if cursorPage.Cursor != "token123" {
						t.Errorf("Cursor = %s, want token123", cursorPage.Cursor)
					}
				}
			},
		},
		{
			name:  "all three",
			query: "filter[q]=and(p1,p2)&filter[p1]=eq(status,active)&filter[p2]=gt(age,18)&sort[created_at]=desc&sort[name]=asc&page[size]=20&page[number]=2",
			check: func(t *testing.T, q *sift.Query) {
				if q.Filter == nil {
					t.Error("Filter should not be nil")
				}
				if q.Sort == nil {
					t.Error("Sort should not be nil")
				}
				if q.Pagination == nil {
					t.Error("Pagination should not be nil")
				}
			},
		},
		{
			name:    "invalid sort direction",
			query:   "sort[field]=invalid",
			wantErr: true,
		},
		{
			name:    "invalid page size",
			query:   "page[size]=abc&page[number]=2",
			wantErr: true,
		},
		{
			name:    "invalid page number",
			query:   "page[size]=20&page[number]=abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, err := url.ParseQuery(tt.query)
			if err != nil {
				t.Fatalf("Failed to parse query: %v", err)
			}

			got, err := ParseQuery(values, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}
