package sql_test

import (
	"context"
	"testing"

	"github.com/nisimpson/sift"
	siftsql "github.com/nisimpson/sift/thru/sql"
)

func TestAdapter_EvaluateCondition(t *testing.T) {
	tests := []struct {
		name      string
		expr      sift.Expression
		wantQuery string
		wantArgs  []interface{}
		wantErr   bool
	}{
		{
			name:      "equal operation",
			expr:      sift.Eq("status", "active"),
			wantQuery: "status = $1",
			wantArgs:  []interface{}{"active"},
		},
		{
			name:      "not equal operation",
			expr:      sift.Neq("status", "deleted"),
			wantQuery: "status != $1",
			wantArgs:  []interface{}{"deleted"},
		},
		{
			name:      "greater than operation",
			expr:      sift.Gt("age", 18),
			wantQuery: "age > $1",
			wantArgs:  []interface{}{int64(18)},
		},
		{
			name:      "less than operation",
			expr:      sift.Lt("age", 65),
			wantQuery: "age < $1",
			wantArgs:  []interface{}{int64(65)},
		},
		{
			name:      "greater than or equal",
			expr:      sift.Gte("score", 90),
			wantQuery: "score >= $1",
			wantArgs:  []interface{}{int64(90)},
		},
		{
			name:      "less than or equal",
			expr:      sift.Lte("score", 100),
			wantQuery: "score <= $1",
			wantArgs:  []interface{}{int64(100)},
		},
		{
			name:      "contains operation",
			expr:      sift.Contains("email", "@example.com"),
			wantQuery: "email LIKE $1",
			wantArgs:  []interface{}{"%@example.com%"},
		},
		{
			name: "begins_with operation",
			expr: &sift.Condition{
				Name:      "name",
				Operation: sift.OperationBeginsWith,
				Value:     "John",
			},
			wantQuery: "name LIKE $1",
			wantArgs:  []interface{}{"John%"},
		},
		{
			name:      "exists operation",
			expr:      sift.Exists("email"),
			wantQuery: "email IS NOT NULL",
			wantArgs:  []interface{}{},
		},
		{
			name:      "not_exists operation",
			expr:      sift.NotExists("deleted_at"),
			wantQuery: "deleted_at IS NULL",
			wantArgs:  []interface{}{},
		},
		{
			name: "in operation",
			expr: &sift.Condition{
				Name:      "status",
				Operation: sift.OperationIn,
				Value:     "active,pending,approved",
			},
			wantQuery: "status IN ($1, $2, $3)",
			wantArgs:  []interface{}{"active", "pending", "approved"},
		},
		{
			name:      "between operation",
			expr:      sift.Between("age", 18, 65),
			wantQuery: "age BETWEEN $1 AND $2",
			wantArgs:  []interface{}{int64(18), int64(65)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := siftsql.NewAdapter()
			err := sift.Thru(context.Background(), adapter, sift.WithFilter(tt.expr))

			if (err != nil) != tt.wantErr {
				t.Errorf("Thru() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if adapter.Query() != tt.wantQuery {
					t.Errorf("Query() = %v, want %v", adapter.Query(), tt.wantQuery)
				}

				if len(adapter.Args()) != len(tt.wantArgs) {
					t.Errorf("Args() length = %v, want %v", len(adapter.Args()), len(tt.wantArgs))
				} else {
					for i, arg := range adapter.Args() {
						if arg != tt.wantArgs[i] {
							t.Errorf("Args()[%d] = %v, want %v", i, arg, tt.wantArgs[i])
						}
					}
				}
			}
		})
	}
}

func TestAdapter_EvaluateAnd(t *testing.T) {
	filter := sift.Eq("status", "active").And(sift.Gt("age", 18))

	adapter := siftsql.NewAdapter()
	err := sift.Thru(context.Background(), adapter, sift.WithFilter(filter))
	if err != nil {
		t.Fatalf("Thru() error = %v", err)
	}

	wantQuery := "(status = $1) AND (age > $2)"
	if adapter.Query() != wantQuery {
		t.Errorf("Query() = %v, want %v", adapter.Query(), wantQuery)
	}

	wantArgs := []interface{}{"active", int64(18)}
	if len(adapter.Args()) != len(wantArgs) {
		t.Errorf("Args() length = %v, want %v", len(adapter.Args()), len(wantArgs))
	}
}

func TestAdapter_EvaluateOr(t *testing.T) {
	filter := sift.Eq("role", "admin").Or(sift.Eq("role", "moderator"))

	adapter := siftsql.NewAdapter()
	err := sift.Thru(context.Background(), adapter, sift.WithFilter(filter))
	if err != nil {
		t.Fatalf("Thru() error = %v", err)
	}

	wantQuery := "(role = $1) OR (role = $2)"
	if adapter.Query() != wantQuery {
		t.Errorf("Query() = %v, want %v", adapter.Query(), wantQuery)
	}

	wantArgs := []interface{}{"admin", "moderator"}
	if len(adapter.Args()) != len(wantArgs) {
		t.Errorf("Args() length = %v, want %v", len(adapter.Args()), len(wantArgs))
	}
}

func TestAdapter_EvaluateNot(t *testing.T) {
	filter := sift.Eq("deleted", "true").Not()

	adapter := siftsql.NewAdapter()
	err := sift.Thru(context.Background(), adapter, sift.WithFilter(filter))
	if err != nil {
		t.Fatalf("Thru() error = %v", err)
	}

	wantQuery := "NOT (deleted = $1)"
	if adapter.Query() != wantQuery {
		t.Errorf("Query() = %v, want %v", adapter.Query(), wantQuery)
	}

	wantArgs := []interface{}{true}
	if len(adapter.Args()) != len(wantArgs) {
		t.Errorf("Args() length = %v, want %v", len(adapter.Args()), len(wantArgs))
	}
}

func TestAdapter_ComplexExpression(t *testing.T) {
	// (status = "active" OR status = "pending") AND age > 18
	filter := sift.Eq("status", "active").
		Or(sift.Eq("status", "pending")).
		And(sift.Gt("age", 18))

	adapter := siftsql.NewAdapter()
	err := sift.Thru(context.Background(), adapter, sift.WithFilter(filter))
	if err != nil {
		t.Fatalf("Thru() error = %v", err)
	}

	wantQuery := "((status = $1) OR (status = $2)) AND (age > $3)"
	if adapter.Query() != wantQuery {
		t.Errorf("Query() = %v, want %v", adapter.Query(), wantQuery)
	}

	t.Logf("Query: %s", adapter.Query())
	t.Logf("Args: %v", adapter.Args())
}

func TestAdapter_Dialects(t *testing.T) {
	filter := sift.Eq("status", "active").And(sift.Gt("age", 18))

	tests := []struct {
		name         string
		dialect      siftsql.Dialect
		wantQuery    string
		wantArgCount int
	}{
		{
			name:         "PostgreSQL",
			dialect:      siftsql.DialectPostgreSQL,
			wantQuery:    "(status = $1) AND (age > $2)",
			wantArgCount: 2,
		},
		{
			name:         "MySQL",
			dialect:      siftsql.DialectMySQL,
			wantQuery:    "(status = ?) AND (age > ?)",
			wantArgCount: 2,
		},
		{
			name:         "SQLite",
			dialect:      siftsql.DialectSQLite,
			wantQuery:    "(status = ?) AND (age > ?)",
			wantArgCount: 2,
		},
		{
			name:         "SQL Server",
			dialect:      siftsql.DialectSQLServer,
			wantQuery:    "(status = @p1) AND (age > @p2)",
			wantArgCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &siftsql.Config{
				Dialect: tt.dialect,
			}
			adapter := siftsql.NewAdapterWithConfig(config)
			err := sift.Thru(context.Background(), adapter, sift.WithFilter(filter))
			if err != nil {
				t.Fatalf("Thru() error = %v", err)
			}

			if adapter.Query() != tt.wantQuery {
				t.Errorf("Query() = %v, want %v", adapter.Query(), tt.wantQuery)
			}

			if len(adapter.Args()) != tt.wantArgCount {
				t.Errorf("Args() length = %v, want %v", len(adapter.Args()), tt.wantArgCount)
			}
		})
	}
}

func TestAdapter_QuoteIdentifiers(t *testing.T) {
	filter := sift.Eq("user_name", "john")

	tests := []struct {
		name      string
		dialect   siftsql.Dialect
		wantQuery string
	}{
		{
			name:      "PostgreSQL with quotes",
			dialect:   siftsql.DialectPostgreSQL,
			wantQuery: `"user_name" = $1`,
		},
		{
			name:      "MySQL with quotes",
			dialect:   siftsql.DialectMySQL,
			wantQuery: "`user_name` = ?",
		},
		{
			name:      "SQLite with quotes",
			dialect:   siftsql.DialectSQLite,
			wantQuery: `"user_name" = ?`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &siftsql.Config{
				Dialect:          tt.dialect,
				QuoteIdentifiers: true,
			}
			adapter := siftsql.NewAdapterWithConfig(config)
			err := sift.Thru(context.Background(), adapter, sift.WithFilter(filter))
			if err != nil {
				t.Fatalf("Thru() error = %v", err)
			}

			if adapter.Query() != tt.wantQuery {
				t.Errorf("Query() = %v, want %v", adapter.Query(), tt.wantQuery)
			}
		})
	}
}

func TestAdapter_CaseInsensitive(t *testing.T) {
	filter := sift.Contains("email", "@EXAMPLE.COM")

	config := &siftsql.Config{
		Dialect:       siftsql.DialectPostgreSQL,
		CaseSensitive: false,
	}
	adapter := siftsql.NewAdapterWithConfig(config)
	err := sift.Thru(context.Background(), adapter, sift.WithFilter(filter))
	if err != nil {
		t.Fatalf("Thru() error = %v", err)
	}

	wantQuery := "LOWER(email) LIKE LOWER($1)"
	if adapter.Query() != wantQuery {
		t.Errorf("Query() = %v, want %v", adapter.Query(), wantQuery)
	}
}

func TestAdapter_NumericTypes(t *testing.T) {
	tests := []struct {
		name     string
		expr     sift.Expression
		wantType interface{}
	}{
		{
			name:     "integer value",
			expr:     sift.Gt("age", 18),
			wantType: int64(18),
		},
		{
			name:     "float value",
			expr:     sift.Lte("price", "99.99"),
			wantType: float64(99.99),
		},
		{
			name:     "boolean value",
			expr:     sift.Eq("active", "true"),
			wantType: true,
		},
		{
			name:     "string value",
			expr:     sift.Eq("name", "John Doe"),
			wantType: "John Doe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := siftsql.NewAdapter()
			err := sift.Thru(context.Background(), adapter, sift.WithFilter(tt.expr))
			if err != nil {
				t.Fatalf("Thru() error = %v", err)
			}

			if len(adapter.Args()) != 1 {
				t.Fatalf("Expected 1 arg, got %d", len(adapter.Args()))
			}

			arg := adapter.Args()[0]
			if arg != tt.wantType {
				t.Errorf("Arg type = %T(%v), want %T(%v)", arg, arg, tt.wantType, tt.wantType)
			}
		})
	}
}


func TestAdapter_Pagination_Offset(t *testing.T) {
	tests := []struct {
		name       string
		page       sift.PaginationExpression
		wantLimit  int
		wantOffset int
	}{
		{
			name:       "first page",
			page:       sift.Paginate().Size(20).Number(1),
			wantLimit:  20,
			wantOffset: 0,
		},
		{
			name:       "second page",
			page:       sift.Paginate().Size(20).Number(2),
			wantLimit:  20,
			wantOffset: 20,
		},
		{
			name:       "third page with size 50",
			page:       sift.Paginate().Size(50).Number(3),
			wantLimit:  50,
			wantOffset: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := siftsql.NewAdapter()
			err := sift.Thru(context.Background(), adapter, sift.WithPagination(tt.page))
			if err != nil {
				t.Fatalf("Thru() error = %v", err)
			}

			if adapter.Limit() != tt.wantLimit {
				t.Errorf("Limit() = %d, want %d", adapter.Limit(), tt.wantLimit)
			}

			if adapter.Offset() != tt.wantOffset {
				t.Errorf("Offset() = %d, want %d", adapter.Offset(), tt.wantOffset)
			}
		})
	}
}

func TestAdapter_Pagination_DefaultLimit(t *testing.T) {
	config := &siftsql.Config{
		Dialect:      siftsql.DialectPostgreSQL,
		DefaultLimit: 100,
	}
	adapter := siftsql.NewAdapterWithConfig(config)

	// No pagination specified - should use default
	if adapter.Limit() != 100 {
		t.Errorf("Limit() = %d, want 100 (default)", adapter.Limit())
	}

	// With pagination - should override default
	page := sift.Paginate().Size(20).Number(1)
	err := sift.Thru(context.Background(), adapter, sift.WithPagination(page))
	if err != nil {
		t.Fatalf("Thru() error = %v", err)
	}

	if adapter.Limit() != 20 {
		t.Errorf("Limit() = %d, want 20 (from pagination)", adapter.Limit())
	}
}

func TestAdapter_FilterSortPage(t *testing.T) {
	filter := sift.Eq("status", "active").And(sift.Gt("age", 18))
	sort := sift.Sort("created_at", sift.SortDesc).ThenBy("name", sift.SortAsc)
	page := sift.Paginate().Size(20).Number(2)

	adapter := siftsql.NewAdapter()
	err := sift.Thru(context.Background(), adapter,
		sift.WithFilter(filter),
		sift.WithSort(sort),
		sift.WithPagination(page))

	if err != nil {
		t.Fatalf("Thru() error = %v", err)
	}

	// Check filter
	if adapter.Query() == "" {
		t.Error("Expected query to be set")
	}

	// Check sort
	expectedOrderBy := "created_at DESC, name ASC"
	if adapter.OrderBy() != expectedOrderBy {
		t.Errorf("OrderBy() = %s, want %s", adapter.OrderBy(), expectedOrderBy)
	}

	// Check pagination
	if adapter.Limit() != 20 {
		t.Errorf("Limit() = %d, want 20", adapter.Limit())
	}

	if adapter.Offset() != 20 {
		t.Errorf("Offset() = %d, want 20", adapter.Offset())
	}
}
