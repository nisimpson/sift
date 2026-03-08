package sql_test

import (
	"context"
	"fmt"

	"github.com/nisimpson/sift"
	siftsql "github.com/nisimpson/sift/thru/sql"
)

func Example() {
	// Create a simple filter: status = "active"
	filter := sift.Eq("status", "active")

	// Create SQL adapter
	adapter := siftsql.NewAdapter()
	sift.Thru(context.Background(), adapter, filter)

	fmt.Printf("Query: %s\n", adapter.Query())
	fmt.Printf("Args: %v\n", adapter.Args())

	// Output:
	// Query: status = $1
	// Args: [active]
}

func ExampleAdapter_complexFilter() {
	// Create a complex filter: (status = "active" AND age > 18) OR role = "admin"
	filter := sift.Eq("status", "active").
		And(sift.Gt("age", 18)).
		Or(sift.Eq("role", "admin"))

	adapter := siftsql.NewAdapter()
	sift.Thru(context.Background(), adapter, filter)

	fmt.Printf("Query: %s\n", adapter.Query())
	fmt.Printf("Args: %v\n", adapter.Args())

	// Output:
	// Query: ((status = $1) AND (age > $2)) OR (role = $3)
	// Args: [active 18 admin]
}

func ExampleAdapter_stringOperations() {
	// String operations: contains and begins_with
	filter := sift.Contains("email", "@example.com").
		And(&sift.Condition{
			Name:      "name",
			Operation: sift.OperationBeginsWith,
			Value:     "John",
		})

	adapter := siftsql.NewAdapter()
	sift.Thru(context.Background(), adapter, filter)

	fmt.Printf("Query: %s\n", adapter.Query())
	fmt.Printf("Args: %v\n", adapter.Args())

	// Output:
	// Query: (email LIKE $1) AND (name LIKE $2)
	// Args: [%@example.com% John%]
}

func ExampleAdapter_existenceChecks() {
	// Existence checks: IS NULL and IS NOT NULL
	filter := sift.Exists("email").And(sift.NotExists("deleted_at"))

	adapter := siftsql.NewAdapter()
	sift.Thru(context.Background(), adapter, filter)

	fmt.Printf("Query: %s\n", adapter.Query())

	// Output:
	// Query: (email IS NOT NULL) AND (deleted_at IS NULL)
}

func ExampleAdapter_withDatabase() {
	// Example showing usage with database/sql
	filter := sift.Eq("status", "active").And(sift.Gt("age", 18))

	adapter := siftsql.NewAdapter()
	sift.Thru(context.Background(), adapter, filter)

	// Use with database/sql (pseudo-code)
	query := fmt.Sprintf("SELECT * FROM users WHERE %s", adapter.Query())
	fmt.Printf("SQL: %s\n", query)
	fmt.Printf("Args: %v\n", adapter.Args())

	// In real code:
	// rows, err := db.Query(query, adapter.Args()...)

	// Output:
	// SQL: SELECT * FROM users WHERE (status = $1) AND (age > $2)
	// Args: [active 18]
}

func ExampleNewAdapterWithConfig_mysql() {
	// Configure for MySQL dialect
	config := &siftsql.Config{
		Dialect:          siftsql.DialectMySQL,
		QuoteIdentifiers: true,
	}

	filter := sift.Eq("user_name", "john").And(sift.Gt("age", 18))

	adapter := siftsql.NewAdapterWithConfig(config)
	sift.Thru(context.Background(), adapter, filter)

	fmt.Printf("Query: %s\n", adapter.Query())
	fmt.Printf("Args: %v\n", adapter.Args())

	// Output:
	// Query: (`user_name` = ?) AND (`age` > ?)
	// Args: [john 18]
}

func ExampleNewAdapterWithConfig_sqlserver() {
	// Configure for SQL Server dialect
	config := &siftsql.Config{
		Dialect: siftsql.DialectSQLServer,
	}

	filter := sift.Eq("status", "active").And(sift.Gt("age", 18))

	adapter := siftsql.NewAdapterWithConfig(config)
	sift.Thru(context.Background(), adapter, filter)

	fmt.Printf("Query: %s\n", adapter.Query())
	fmt.Printf("Args: %v\n", adapter.Args())

	// Output:
	// Query: (status = @p1) AND (age > @p2)
	// Args: [active 18]
}

func ExampleAdapter_caseInsensitive() {
	// Case-insensitive string matching
	config := &siftsql.Config{
		Dialect:       siftsql.DialectPostgreSQL,
		CaseSensitive: false,
	}

	filter := sift.Contains("email", "@EXAMPLE.COM")

	adapter := siftsql.NewAdapterWithConfig(config)
	sift.Thru(context.Background(), adapter, filter)

	fmt.Printf("Query: %s\n", adapter.Query())

	// Output:
	// Query: LOWER(email) LIKE LOWER($1)
}

func ExampleAdapter_inOperation() {
	// IN operation with multiple values
	filter := &sift.Condition{
		Name:      "status",
		Operation: sift.OperationIn,
		Value:     "active,pending,approved",
	}

	adapter := siftsql.NewAdapter()
	sift.Thru(context.Background(), adapter, filter)

	fmt.Printf("Query: %s\n", adapter.Query())
	fmt.Printf("Args: %v\n", adapter.Args())

	// Output:
	// Query: status IN ($1, $2, $3)
	// Args: [active pending approved]
}

func ExampleAdapter_betweenOperation() {
	// BETWEEN operation
	filter := sift.Between("age", 18, 65)

	adapter := siftsql.NewAdapter()
	sift.Thru(context.Background(), adapter, filter)

	fmt.Printf("Query: %s\n", adapter.Query())
	fmt.Printf("Args: %v\n", adapter.Args())

	// Output:
	// Query: age BETWEEN $1 AND $2
	// Args: [18 65]
}

func ExampleAdapter_notOperation() {
	// NOT operation
	filter := sift.Eq("deleted", "true").Not()

	adapter := siftsql.NewAdapter()
	sift.Thru(context.Background(), adapter, filter)

	fmt.Printf("Query: %s\n", adapter.Query())
	fmt.Printf("Args: %v\n", adapter.Args())

	// Output:
	// Query: NOT (deleted = $1)
	// Args: [true]
}
