package exprlang_test

import (
	"testing"

	"github.com/nisimpson/sift"
	"github.com/nisimpson/sift/thru/exprlang"
)

func TestFormatter_FormatAndParse(t *testing.T) {
	// Create registry with exprlang custom expressions
	registry := exprlang.NewRegistry()

	tests := []struct {
		name string
		expr sift.Expression
		want string
	}{
		{
			name: "simple expression",
			expr: exprlang.RawExpression("len(tweets) > 10"),
			want: `exprlang(len\(tweets\) > 10)`,
		},
		{
			name: "expression with commas",
			expr: exprlang.RawExpression("split(name, ',')"),
			want: `exprlang(split\(name\, '\,'\))`,
		},
		{
			name: "expression with parentheses",
			expr: exprlang.RawExpression("(a + b) * (c + d)"),
			want: `exprlang(\(a + b\) * \(c + d\))`,
		},
		{
			name: "array function",
			expr: exprlang.ArrayFunction("filter", "tweets", "len(.Content) > 240"),
			want: `exprlang(filter\(tweets\, len\(.Content\) > 240\))`,
		},
		{
			name: "mixed with standard operations",
			expr: sift.Eq("status", "active").And(exprlang.RawExpression("len(comments) > 5")),
			want: `and(eq(status,active),exprlang(len\(comments\) > 5))`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test formatting
			formatted, err := sift.Format(tt.expr, registry)
			if err != nil {
				t.Fatalf("Format() error = %v", err)
			}
			if formatted != tt.want {
				t.Errorf("Format() = %v, want %v", formatted, tt.want)
			}

			// Test parsing (round-trip)
			parsed, err := sift.Parse(formatted, registry)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			// Format again to verify round-trip
			reformatted, err := sift.Format(parsed, registry)
			if err != nil {
				t.Fatalf("Format() after Parse() error = %v", err)
			}
			if reformatted != tt.want {
				t.Errorf("Round-trip Format() = %v, want %v", reformatted, tt.want)
			}
		})
	}
}

func TestFormatter_ParseErrors(t *testing.T) {
	// Create registry with exprlang custom expressions
	registry := exprlang.NewRegistry()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "empty expression",
			input:   "exprlang()",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := sift.Parse(tt.input, registry)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFormatter_ComplexRoundTrip(t *testing.T) {
	// Create registry with exprlang custom expressions
	registry := exprlang.NewRegistry()

	// Create a complex filter with nested operations and custom expressions
	filter := sift.Eq("Status", "published").
		And(exprlang.RawExpression("len(Comments) > 10")).
		Or(sift.In("Tags", "featured").And(exprlang.RawExpression("Views > 1000")))

	// Format to string
	formatted, err := sift.Format(filter, registry)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Parse back
	parsed, err := sift.Parse(formatted, registry)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Format again
	reformatted, err := sift.Format(parsed, registry)
	if err != nil {
		t.Fatalf("Format() after Parse() error = %v", err)
	}

	// Should match original
	if reformatted != formatted {
		t.Errorf("Round-trip failed:\nOriginal:  %s\nReformatted: %s", formatted, reformatted)
	}
}
