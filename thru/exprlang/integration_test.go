package exprlang_test

import (
	"context"
	"testing"
	"time"

	"github.com/expr-lang/expr"
	"github.com/nisimpson/sift"
	"github.com/nisimpson/sift/thru/exprlang"
)

// BlogPost represents a blog post with rich metadata
type BlogPost struct {
	Title       string
	Content     string
	Author      string
	Status      string
	Tags        []string
	Comments    []Comment
	Likes       int
	Views       int
	CreatedAt   time.Time
	PublishedAt *time.Time
}

type Comment struct {
	Author  string
	Content string
	Likes   int
}

func TestIntegration_ComprehensiveFiltering(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	lastWeek := now.Add(-7 * 24 * time.Hour)

	posts := []BlogPost{
		{
			Title:       "Getting Started with Go",
			Content:     "This is a comprehensive guide to Go programming...",
			Author:      "Alice",
			Status:      "published",
			Tags:        []string{"golang", "tutorial", "beginner"},
			Comments:    []Comment{{Author: "Bob", Content: "Great post!", Likes: 5}},
			Likes:       100,
			Views:       1000,
			CreatedAt:   lastWeek,
			PublishedAt: &yesterday,
		},
		{
			Title:     "Advanced Go Patterns",
			Content:   "Deep dive into advanced Go patterns...",
			Author:    "Alice",
			Status:    "draft",
			Tags:      []string{"golang", "advanced"},
			Comments:  []Comment{},
			Likes:     0,
			Views:     0,
			CreatedAt: now,
		},
		{
			Title:       "Python vs Go",
			Content:     "Comparing Python and Go for web development...",
			Author:      "Charlie",
			Status:      "published",
			Tags:        []string{"python", "golang", "comparison"},
			Comments:    []Comment{{Author: "Alice", Content: "Nice comparison", Likes: 10}, {Author: "Bob", Content: "Agreed", Likes: 3}},
			Likes:       50,
			Views:       500,
			CreatedAt:   lastWeek,
			PublishedAt: &lastWeek,
		},
	}

	tests := []struct {
		name     string
		filter   sift.Expression
		expected []string // Expected post titles
	}{
		{
			name:     "standard operations only",
			filter:   sift.Eq("Status", "published").And(sift.In("Tags", "golang")),
			expected: []string{"Getting Started with Go", "Python vs Go"},
		},
		{
			name:     "custom expression - array length",
			filter:   exprlang.RawExpression("len(Comments) > 1"),
			expected: []string{"Python vs Go"},
		},
		{
			name:     "custom expression - string operations",
			filter:   exprlang.RawExpression("Title contains 'Go' and Author startsWith 'A'"),
			expected: []string{"Getting Started with Go", "Advanced Go Patterns"},
		},
		{
			name:     "mixed standard and custom",
			filter:   sift.Eq("Status", "published").And(exprlang.RawExpression("Likes > 50")),
			expected: []string{"Getting Started with Go"},
		},
		{
			name:     "complex custom - array predicates",
			filter:   exprlang.RawExpression("Status == 'published' and any(Comments, .Likes >= 5)"),
			expected: []string{"Getting Started with Go", "Python vs Go"},
		},
		{
			name:     "custom - engagement ratio",
			filter:   exprlang.RawExpression("Status == 'published' and (Likes / Views) > 0.05"),
			expected: []string{"Getting Started with Go", "Python vs Go"},
		},
		{
			name:     "between operation",
			filter:   sift.Between("Likes", 40, 150),
			expected: []string{"Getting Started with Go", "Python vs Go"},
		},
		{
			name:     "string contains",
			filter:   sift.Contains("Title", "Go"),
			expected: []string{"Getting Started with Go", "Advanced Go Patterns", "Python vs Go"},
		},
		{
			name:     "exists check",
			filter:   sift.Exists("PublishedAt"),
			expected: []string{"Getting Started with Go", "Python vs Go"},
		},
		{
			name: "complex nested logic",
			filter: sift.Eq("Status", "published").And(sift.Gt("Likes", 75)).
				Or(sift.Eq("Author", "Alice").And(exprlang.RawExpression("len(Tags) >= 2"))),
			expected: []string{"Getting Started with Go", "Advanced Go Patterns"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Translate to expr-lang
			adapter := exprlang.NewAdapter()
			err := sift.Thru(context.Background(), adapter, tt.filter)
			if err != nil {
				t.Fatalf("Failed to translate filter: %v", err)
			}

			// Compile expression
			program, err := expr.Compile(adapter.Expression(), expr.Env(BlogPost{}))
			if err != nil {
				t.Fatalf("Failed to compile expression '%s': %v", adapter.Expression(), err)
			}

			// Filter posts
			var matched []string
			for _, post := range posts {
				output, err := expr.Run(program, post)
				if err != nil {
					t.Fatalf("Failed to run expression on post '%s': %v", post.Title, err)
				}
				if output.(bool) {
					matched = append(matched, post.Title)
				}
			}

			// Verify results
			if len(matched) != len(tt.expected) {
				t.Errorf("Expected %d matches, got %d\nExpression: %s\nMatched: %v\nExpected: %v",
					len(tt.expected), len(matched), adapter.Expression(), matched, tt.expected)
				return
			}

			for i, title := range matched {
				if title != tt.expected[i] {
					t.Errorf("Match %d: expected '%s', got '%s'", i, tt.expected[i], title)
				}
			}
		})
	}
}
