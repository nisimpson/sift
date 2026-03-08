package jsonapi

import (
	"net/url"
	"strings"
	"testing"

	"github.com/nisimpson/sift"
)

func TestAdapter_Parse(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		want    sift.Expression
		wantErr bool
	}{
		{
			name:  "simple equality",
			query: "filter[q]=p1&filter[p1]=eq(status,active)",
			want: &sift.Condition{
				Name:      "status",
				Operation: sift.OperationEQ,
				Value:     "active",
			},
			wantErr: false,
		},
		{
			name:  "and operation",
			query: "filter[q]=and(p1,p2)&filter[p1]=eq(status,active)&filter[p2]=gt(age,18)",
			want: &sift.AndOperation{
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
			wantErr: false,
		},
		{
			name:  "or operation",
			query: "filter[q]=or(p1,p2)&filter[p1]=eq(role,admin)&filter[p2]=eq(role,moderator)",
			want: &sift.OrOperation{
				Left: &sift.Condition{
					Name:      "role",
					Operation: sift.OperationEQ,
					Value:     "admin",
				},
				Right: &sift.Condition{
					Name:      "role",
					Operation: sift.OperationEQ,
					Value:     "moderator",
				},
			},
			wantErr: false,
		},
		{
			name:  "not operation",
			query: "filter[q]=not(p1)&filter[p1]=eq(deleted,true)",
			want: &sift.NotOperation{
				Child: &sift.Condition{
					Name:      "deleted",
					Operation: sift.OperationEQ,
					Value:     "true",
				},
			},
			wantErr: false,
		},
		{
			name:  "nested operations",
			query: "filter[q]=and(p1,or(p2,p3))&filter[p1]=eq(status,active)&filter[p2]=gt(age,18)&filter[p3]=eq(role,admin)",
			want: &sift.AndOperation{
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
			wantErr: false,
		},
		{
			name:  "contains operation",
			query: "filter[q]=p1&filter[p1]=contains(email,@example.com)",
			want: &sift.Condition{
				Name:      "email",
				Operation: sift.OperationContains,
				Value:     "@example.com",
			},
			wantErr: false,
		},
		{
			name:  "exists operation",
			query: "filter[q]=p1&filter[p1]=exists(email)",
			want: &sift.Condition{
				Name:      "email",
				Operation: sift.OperationExists,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, err := url.ParseQuery(tt.query)
			if err != nil {
				t.Fatalf("Failed to parse query: %v", err)
			}

			adapter := NewAdapter()
			got, err := adapter.Parse(values)

			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Compare expressions using Format
				wantStr, _ := sift.Format(tt.want)
				gotStr, _ := sift.Format(got)
				if wantStr != gotStr {
					t.Errorf("Parse() = %v, want %v", gotStr, wantStr)
				}
			}
		})
	}
}

func TestAdapter_Format(t *testing.T) {
	tests := []struct {
		name       string
		expr       sift.Expression
		wantQuery  string
		wantParams map[string]string
		wantErr    bool
	}{
		{
			name: "simple equality",
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
			name: "or operation",
			expr: &sift.OrOperation{
				Left: &sift.Condition{
					Name:      "role",
					Operation: sift.OperationEQ,
					Value:     "admin",
				},
				Right: &sift.Condition{
					Name:      "role",
					Operation: sift.OperationEQ,
					Value:     "moderator",
				},
			},
			wantQuery: "or(p1,p2)",
			wantParams: map[string]string{
				"p1": "eq(role,admin)",
				"p2": "eq(role,moderator)",
			},
			wantErr: false,
		},
		{
			name: "not operation",
			expr: &sift.NotOperation{
				Child: &sift.Condition{
					Name:      "deleted",
					Operation: sift.OperationEQ,
					Value:     "true",
				},
			},
			wantQuery: "not(p1)",
			wantParams: map[string]string{
				"p1": "eq(deleted,true)",
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
			adapter := NewAdapter()
			values, err := adapter.Format(tt.expr)

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

				// Check we don't have extra parameters
				for key := range values {
					if key == "filter[q]" {
						continue
					}
					if !strings.HasPrefix(key, "filter[") {
						continue
					}
					paramName := key[7 : len(key)-1]
					if _, ok := tt.wantParams[paramName]; !ok {
						t.Errorf("Format() unexpected parameter: %s", paramName)
					}
				}
			}
		})
	}
}

func TestAdapter_RoundTrip(t *testing.T) {
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
		{
			name: "not operation",
			expr: &sift.NotOperation{
				Child: &sift.Condition{
					Name:      "deleted",
					Operation: sift.OperationEQ,
					Value:     "true",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter()

			// Format to JSON:API
			values, err := adapter.Format(tt.expr)
			if err != nil {
				t.Fatalf("Format() error = %v", err)
			}

			// Parse back to Sift
			parsed, err := adapter.Parse(values)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			// Compare using Format
			wantStr, _ := sift.Format(tt.expr)
			gotStr, _ := sift.Format(parsed)

			if wantStr != gotStr {
				t.Errorf("Round-trip failed: got %v, want %v", gotStr, wantStr)
			}
		})
	}
}
