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
			name:    "missing main query",
			query:   "filter[p1]=eq(status,active)",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, err := url.ParseQuery(tt.query)
			if err != nil {
				t.Fatalf("Failed to parse query: %v", err)
			}

			got, err := Parse(values, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				gotStr, _ := sift.Format(got, nil)
				if gotStr != tt.want {
					t.Errorf("Parse() = %v, want %v", gotStr, tt.want)
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
			parsed, err := Parse(values, nil)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			// Compare using sift.Format
			wantStr, _ := sift.Format(tt.expr, nil)
			gotStr, _ := sift.Format(parsed, nil)

			if wantStr != gotStr {
				t.Errorf("Round-trip failed: got %v, want %v", gotStr, wantStr)
			}
		})
	}
}
