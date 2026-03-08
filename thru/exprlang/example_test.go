package exprlang_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/nisimpson/sift"
	"github.com/nisimpson/sift/thru/exprlang"
)

// User represents a user in the system.
type User struct {
	Name     string
	Email    string
	Age      int
	Role     string
	Status   string
	Verified bool
}

func Example_basicFilter() {
	// Create a sift filter: status = "active"
	filter := &sift.Condition{
		Name:      "Status",
		Operation: sift.OperationEQ,
		Value:     "active",
	}

	// Translate to expr-lang
	adapter := exprlang.NewAdapter()
	_ = sift.Thru(context.Background(), adapter, filter)

	// Compile and run with expr-lang
	program, _ := expr.Compile(adapter.Expression(), expr.Env(User{}))

	users := []User{
		{Name: "Alice", Status: "active"},
		{Name: "Bob", Status: "inactive"},
		{Name: "Charlie", Status: "active"},
	}

	for _, user := range users {
		output, _ := expr.Run(program, user)
		if output.(bool) {
			fmt.Println(user.Name)
		}
	}

	// Output:
	// Alice
	// Charlie
}

func Example_complexFilter() {
	// Create a complex filter: (status = "active" AND age >= 18) OR role = "admin"
	filter := &sift.OrOperation{
		Left: &sift.AndOperation{
			Left: &sift.Condition{
				Name:      "Status",
				Operation: sift.OperationEQ,
				Value:     "active",
			},
			Right: &sift.Condition{
				Name:      "Age",
				Operation: sift.OperationGTE,
				Value:     "18",
			},
		},
		Right: &sift.Condition{
			Name:      "Role",
			Operation: sift.OperationEQ,
			Value:     "admin",
		},
	}

	// Translate to expr-lang
	adapter := exprlang.NewAdapter()
	_ = sift.Thru(context.Background(), adapter, filter)

	fmt.Println("Expression:", adapter.Expression())

	// Compile and run with expr-lang
	program, _ := expr.Compile(adapter.Expression(), expr.Env(User{}))

	users := []User{
		{Name: "Alice", Status: "active", Age: 25, Role: "user"},
		{Name: "Bob", Status: "inactive", Age: 30, Role: "user"},
		{Name: "Charlie", Status: "active", Age: 16, Role: "user"},
		{Name: "David", Status: "inactive", Age: 40, Role: "admin"},
	}

	fmt.Println("Matching users:")
	for _, user := range users {
		output, _ := expr.Run(program, user)
		if output.(bool) {
			fmt.Printf("  %s (status=%s, age=%d, role=%s)\n",
				user.Name, user.Status, user.Age, user.Role)
		}
	}

	// Output:
	// Expression: ((Status == "active") && (Age >= 18)) || (Role == "admin")
	// Matching users:
	//   Alice (status=active, age=25, role=user)
	//   David (status=inactive, age=40, role=admin)
}

func Example_negation() {
	// Filter: NOT(status = "deleted")
	filter := &sift.NotOperation{
		Child: &sift.Condition{
			Name:      "Status",
			Operation: sift.OperationEQ,
			Value:     "deleted",
		},
	}

	// Translate to expr-lang
	adapter := exprlang.NewAdapter()
	_ = sift.Thru(context.Background(), adapter, filter)

	fmt.Println("Expression:", adapter.Expression())

	// Compile and run with expr-lang
	program, _ := expr.Compile(adapter.Expression(), expr.Env(User{}))

	users := []User{
		{Name: "Alice", Status: "active"},
		{Name: "Bob", Status: "deleted"},
		{Name: "Charlie", Status: "inactive"},
	}

	fmt.Println("Matching users:")
	for _, user := range users {
		output, _ := expr.Run(program, user)
		if output.(bool) {
			fmt.Println(" ", user.Name)
		}
	}

	// Output:
	// Expression: !(Status == "deleted")
	// Matching users:
	//   Alice
	//   Charlie
}

func Example_inOperation() {
	// Create a type with a slice field for the 'in' operator
	type Team struct {
		Name  string
		Roles []string
	}

	// Filter: "admin" in Roles
	filter := &sift.Condition{
		Name:      "Roles",
		Operation: sift.OperationIn,
		Value:     "admin",
	}

	// Translate to expr-lang
	adapter := exprlang.NewAdapter()
	_ = sift.Thru(context.Background(), adapter, filter)

	fmt.Println("Expression:", adapter.Expression())

	// Compile and run with expr-lang
	program, _ := expr.Compile(adapter.Expression(), expr.Env(Team{}))

	teams := []Team{
		{Name: "Engineering", Roles: []string{"admin", "developer"}},
		{Name: "Marketing", Roles: []string{"user", "editor"}},
		{Name: "Operations", Roles: []string{"admin", "operator"}},
	}

	fmt.Println("Matching teams:")
	for _, team := range teams {
		output, _ := expr.Run(program, team)
		if output.(bool) {
			fmt.Printf("  %s\n", team.Name)
		}
	}

	// Output:
	// Expression: "admin" in Roles
	// Matching teams:
	//   Engineering
	//   Operations
}


func Example_stringOperations() {
	// Filter: email contains "@example.com" AND name starts with "John"
	filter := &sift.AndOperation{
		Left: &sift.Condition{
			Name:      "Email",
			Operation: sift.OperationContains,
			Value:     "@example.com",
		},
		Right: &sift.Condition{
			Name:      "Name",
			Operation: sift.OperationBeginsWith,
			Value:     "John",
		},
	}

	// Translate to expr-lang
	adapter := exprlang.NewAdapter()
	_ = sift.Thru(context.Background(), adapter, filter)

	fmt.Println("Expression:", adapter.Expression())

	// Compile and run with expr-lang
	program, _ := expr.Compile(adapter.Expression(), expr.Env(User{}))

	users := []User{
		{Name: "John Doe", Email: "john@example.com"},
		{Name: "Jane Smith", Email: "jane@example.com"},
		{Name: "Johnny Walker", Email: "johnny@example.com"},
		{Name: "John Adams", Email: "john@other.com"},
	}

	fmt.Println("Matching users:")
	for _, user := range users {
		output, _ := expr.Run(program, user)
		if output.(bool) {
			fmt.Printf("  %s <%s>\n", user.Name, user.Email)
		}
	}

	// Output:
	// Expression: (Email contains "@example.com") && (Name startsWith "John")
	// Matching users:
	//   John Doe <john@example.com>
	//   Johnny Walker <johnny@example.com>
}

func Example_betweenOperation() {
	// Filter: age between 18 and 65
	filter := &sift.Condition{
		Name:      "Age",
		Operation: sift.OperationBetween,
		Value:     "18,65",
	}

	// Translate to expr-lang
	adapter := exprlang.NewAdapter()
	_ = sift.Thru(context.Background(), adapter, filter)

	fmt.Println("Expression:", adapter.Expression())

	// Compile and run with expr-lang
	program, _ := expr.Compile(adapter.Expression(), expr.Env(User{}))

	users := []User{
		{Name: "Alice", Age: 25},
		{Name: "Bob", Age: 16},
		{Name: "Charlie", Age: 70},
		{Name: "David", Age: 45},
	}

	fmt.Println("Matching users:")
	for _, user := range users {
		output, _ := expr.Run(program, user)
		if output.(bool) {
			fmt.Printf("  %s (age=%d)\n", user.Name, user.Age)
		}
	}

	// Output:
	// Expression: Age >= 18 and Age <= 65
	// Matching users:
	//   Alice (age=25)
	//   David (age=45)
}

func Example_customExpression() {
	type Tweet struct {
		Content string
		Likes   int
	}

	type UserWithTweets struct {
		Name   string
		Tweets []Tweet
	}

	// Use custom expr-lang expression for complex filtering
	// Filter users who have more than 5 tweets with content longer than 100 chars
	filter := exprlang.RawExpression("len(Tweets) > 5 and any(Tweets, len(.Content) > 100)")

	// Translate to expr-lang
	adapter := exprlang.NewAdapter()
	_ = sift.Thru(context.Background(), adapter, filter)

	fmt.Println("Expression:", adapter.Expression())

	// Compile and run with expr-lang
	program, _ := expr.Compile(adapter.Expression(), expr.Env(UserWithTweets{}))

	users := []UserWithTweets{
		{
			Name: "Alice",
			Tweets: []Tweet{
				{Content: "Short", Likes: 10},
				{Content: strings.Repeat("Long tweet content ", 10), Likes: 50},
				{Content: "Another short", Likes: 5},
				{Content: strings.Repeat("More long content ", 10), Likes: 30},
				{Content: "Short again", Likes: 8},
				{Content: strings.Repeat("Yet another long one ", 10), Likes: 40},
			},
		},
		{
			Name: "Bob",
			Tweets: []Tweet{
				{Content: "Short", Likes: 10},
				{Content: "Also short", Likes: 5},
			},
		},
	}

	fmt.Println("Matching users:")
	for _, user := range users {
		output, _ := expr.Run(program, user)
		if output.(bool) {
			fmt.Printf("  %s (tweets=%d)\n", user.Name, len(user.Tweets))
		}
	}

	// Output:
	// Expression: len(Tweets) > 5 and any(Tweets, len(.Content) > 100)
	// Matching users:
	//   Alice (tweets=6)
}

func Example_mixedStandardAndCustom() {
	type Post struct {
		Title    string
		Status   string
		Tags     []string
		Comments []string
	}

	// Combine standard Sift operations with custom expr-lang expressions
	// Filter: status = "published" AND (has "golang" tag OR has more than 10 comments)
	filter := &sift.AndOperation{
		Left: &sift.Condition{
			Name:      "Status",
			Operation: sift.OperationEQ,
			Value:     "published",
		},
		Right: &sift.OrOperation{
			Left: &sift.Condition{
				Name:      "Tags",
				Operation: sift.OperationIn,
				Value:     "golang",
			},
			Right: exprlang.RawExpression("len(Comments) > 10"),
		},
	}

	// Translate to expr-lang
	adapter := exprlang.NewAdapter()
	_ = sift.Thru(context.Background(), adapter, filter)

	fmt.Println("Expression:", adapter.Expression())

	// Compile and run with expr-lang
	program, _ := expr.Compile(adapter.Expression(), expr.Env(Post{}))

	posts := []Post{
		{Title: "Go Tutorial", Status: "published", Tags: []string{"golang", "tutorial"}, Comments: make([]string, 5)},
		{Title: "Python Guide", Status: "published", Tags: []string{"python"}, Comments: make([]string, 15)},
		{Title: "Draft Post", Status: "draft", Tags: []string{"golang"}, Comments: make([]string, 20)},
	}

	fmt.Println("Matching posts:")
	for _, post := range posts {
		output, _ := expr.Run(program, post)
		if output.(bool) {
			fmt.Printf("  %s\n", post.Title)
		}
	}

	// Output:
	// Expression: (Status == "published") && (("golang" in Tags) || (len(Comments) > 10))
	// Matching posts:
	//   Go Tutorial
	//   Python Guide
}


func Example_serialization() {
	// Custom expressions serialize as exprlang(...) in Sift format
	
	// Simple custom expression
	filter1 := exprlang.RawExpression("len(tweets) > 10")
	fmt.Println("Custom expression:")
	fmt.Println("  Sift format:", filter1.String())
	
	adapter1 := exprlang.NewAdapter()
	_ = sift.Thru(context.Background(), adapter1, filter1)
	fmt.Println("  Expr-lang:", adapter1.Expression())
	
	// Mixed standard and custom
	filter2 := &sift.AndOperation{
		Left: &sift.Condition{
			Name:      "Status",
			Operation: sift.OperationEQ,
			Value:     "active",
		},
		Right: exprlang.RawExpression("len(Comments) > 5"),
	}
	fmt.Println("\nMixed expression:")
	fmt.Println("  Sift format:", filter2.String())
	
	adapter2 := exprlang.NewAdapter()
	_ = sift.Thru(context.Background(), adapter2, filter2)
	fmt.Println("  Expr-lang:", adapter2.Expression())

	// Output:
	// Custom expression:
	//   Sift format: exprlang(len(tweets) > 10)
	//   Expr-lang: len(tweets) > 10
	//
	// Mixed expression:
	//   Sift format: and(eq(Status,active),exprlang(len(Comments) > 5))
	//   Expr-lang: (Status == "active") && (len(Comments) > 5)
}
