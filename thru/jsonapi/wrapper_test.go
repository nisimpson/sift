package jsonapi_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/nisimpson/sift/thru/jsonapi"
)

func TestWrapper_ParseThru(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantCalls int
		wantErr   bool
	}{
		{
			name:      "simple condition",
			query:     "filter[q]=p1&filter[p1]=eq(status,active)",
			wantCalls: 1,
			wantErr:   false,
		},
		{
			name:      "and operation",
			query:     "filter[q]=and(p1,p2)&filter[p1]=eq(status,active)&filter[p2]=gt(age,18)",
			wantCalls: 3, // AndOperation + 2 Conditions
			wantErr:   false,
		},
		{
			name:      "nested operations",
			query:     "filter[q]=and(p1,or(p2,p3))&filter[p1]=eq(status,active)&filter[p2]=gt(age,18)&filter[p3]=eq(role,admin)",
			wantCalls: 5, // AndOperation + Condition + OrOperation + 2 Conditions
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, err := url.ParseQuery(tt.query)
			if err != nil {
				t.Fatalf("Failed to parse query: %v", err)
			}

			mockAdapter := NewMockBackendAdapter()
			wrapper := jsonapi.Wrap(mockAdapter)

			err = wrapper.ParseThru(context.Background(), values)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseThru() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if mockAdapter.CallCount() != tt.wantCalls {
					t.Errorf("ParseThru() call count = %d, want %d", mockAdapter.CallCount(), tt.wantCalls)
				}
			}
		})
	}
}

func TestWrapper_Adapter(t *testing.T) {
	mockAdapter := NewMockBackendAdapter()
	wrapper := jsonapi.Wrap(mockAdapter)

	if wrapper.Adapter() != mockAdapter {
		t.Error("Adapter() did not return the wrapped adapter")
	}
}

func TestWrapper_JSONAPIAdapter(t *testing.T) {
	mockAdapter := NewMockBackendAdapter()
	wrapper := jsonapi.Wrap(mockAdapter)

	if wrapper.JSONAPIAdapter() == nil {
		t.Error("JSONAPIAdapter() returned nil")
	}
}
