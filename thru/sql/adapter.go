// Package sql provides a SQL adapter for the sift filter library.
// It translates sift filter expressions into SQL WHERE clause syntax.
package sql

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/nisimpson/sift"
)

// Dialect represents a SQL dialect for query generation.
type Dialect string

const (
	// DialectPostgreSQL generates PostgreSQL-compatible queries with $1, $2, etc. placeholders
	DialectPostgreSQL Dialect = "postgresql"
	// DialectMySQL generates MySQL-compatible queries with ? placeholders
	DialectMySQL Dialect = "mysql"
	// DialectSQLite generates SQLite-compatible queries with ? placeholders
	DialectSQLite Dialect = "sqlite"
	// DialectSQLServer generates SQL Server-compatible queries with @p1, @p2, etc. placeholders
	DialectSQLServer Dialect = "sqlserver"
)

// Config holds configuration options for the SQL adapter.
type Config struct {
	// Dialect specifies the SQL dialect to use for query generation.
	// Defaults to DialectPostgreSQL.
	Dialect Dialect

	// QuoteIdentifiers determines whether to quote column names.
	// PostgreSQL/SQLite use double quotes, MySQL uses backticks.
	QuoteIdentifiers bool

	// CaseSensitive determines whether string comparisons are case-sensitive.
	// When false, uses LOWER() for case-insensitive comparisons.
	CaseSensitive bool
}

// Adapter translates sift filter expressions into SQL WHERE clause syntax.
// It accumulates the SQL query string and parameter values during traversal.
type Adapter struct {
	query      string
	args       []interface{}
	paramCount int
	config     *Config
}

// NewAdapter creates a new SQL adapter with default configuration (PostgreSQL dialect).
func NewAdapter() *Adapter {
	return &Adapter{
		args: make([]interface{}, 0),
		config: &Config{
			Dialect:          DialectPostgreSQL,
			QuoteIdentifiers: false,
			CaseSensitive:    true,
		},
	}
}

// NewAdapterWithConfig creates a new SQL adapter with custom configuration.
func NewAdapterWithConfig(config *Config) *Adapter {
	if config == nil {
		config = &Config{
			Dialect:          DialectPostgreSQL,
			QuoteIdentifiers: false,
			CaseSensitive:    true,
		}
	}
	return &Adapter{
		args:   make([]interface{}, 0),
		config: config,
	}
}

// Evaluator returns a sift evaluator configured for SQL.
func (a *Adapter) Evaluator(ctx context.Context) *sift.Evaluator {
	return &sift.Evaluator{
		ConditionEvaluator: a,
		AndEvaluator:       a,
		OrEvaluator:        a,
		NotEvaluator:       a,
	}
}

// EvaluateCondition translates a sift condition into a SQL WHERE clause fragment.
func (a *Adapter) EvaluateCondition(ctx context.Context, node *sift.Condition) error {
	columnName := a.quoteIdentifier(node.Name)

	switch node.Operation {
	case sift.OperationEQ:
		a.query = fmt.Sprintf("%s = %s", columnName, a.nextPlaceholder())
		a.args = append(a.args, a.parseValue(node.Value))

	case sift.OperationNEQ:
		a.query = fmt.Sprintf("%s != %s", columnName, a.nextPlaceholder())
		a.args = append(a.args, a.parseValue(node.Value))

	case sift.OperationLT:
		a.query = fmt.Sprintf("%s < %s", columnName, a.nextPlaceholder())
		a.args = append(a.args, a.parseValue(node.Value))

	case sift.OperationLTE:
		a.query = fmt.Sprintf("%s <= %s", columnName, a.nextPlaceholder())
		a.args = append(a.args, a.parseValue(node.Value))

	case sift.OperationGT:
		a.query = fmt.Sprintf("%s > %s", columnName, a.nextPlaceholder())
		a.args = append(a.args, a.parseValue(node.Value))

	case sift.OperationGTE:
		a.query = fmt.Sprintf("%s >= %s", columnName, a.nextPlaceholder())
		a.args = append(a.args, a.parseValue(node.Value))

	case sift.OperationContains:
		if a.config.CaseSensitive {
			a.query = fmt.Sprintf("%s LIKE %s", columnName, a.nextPlaceholder())
		} else {
			a.query = fmt.Sprintf("LOWER(%s) LIKE LOWER(%s)", columnName, a.nextPlaceholder())
		}
		a.args = append(a.args, "%"+node.Value+"%")

	case sift.OperationBeginsWith:
		if a.config.CaseSensitive {
			a.query = fmt.Sprintf("%s LIKE %s", columnName, a.nextPlaceholder())
		} else {
			a.query = fmt.Sprintf("LOWER(%s) LIKE LOWER(%s)", columnName, a.nextPlaceholder())
		}
		a.args = append(a.args, node.Value+"%")

	case sift.OperationIn:
		// For IN, we expect comma-separated values
		values := strings.Split(node.Value, ",")
		placeholders := make([]string, len(values))
		for i, val := range values {
			placeholders[i] = a.nextPlaceholder()
			a.args = append(a.args, a.parseValue(strings.TrimSpace(val)))
		}
		a.query = fmt.Sprintf("%s IN (%s)", columnName, strings.Join(placeholders, ", "))

	case sift.OperationBetween:
		// For BETWEEN, we expect comma-separated values: "min,max"
		values := strings.Split(node.Value, ",")
		if len(values) != 2 {
			return fmt.Errorf("BETWEEN requires exactly 2 values, got %d", len(values))
		}
		a.query = fmt.Sprintf("%s BETWEEN %s AND %s", columnName, a.nextPlaceholder(), a.nextPlaceholder())
		a.args = append(a.args, a.parseValue(strings.TrimSpace(values[0])))
		a.args = append(a.args, a.parseValue(strings.TrimSpace(values[1])))

	case sift.OperationExists:
		a.query = fmt.Sprintf("%s IS NOT NULL", columnName)

	case sift.OperationNotExists:
		a.query = fmt.Sprintf("%s IS NULL", columnName)

	default:
		return sift.ErrorOperationNotSupported(string(node.Operation))
	}

	return nil
}

// EvaluateAnd combines two conditions with logical AND.
func (a *Adapter) EvaluateAnd(ctx context.Context, node *sift.AndOperation) error {
	leftAdapter := NewAdapterWithConfig(a.config)
	if err := sift.Thru(ctx, leftAdapter, node.Left); err != nil {
		return err
	}

	rightAdapter := NewAdapterWithConfig(a.config)
	rightAdapter.paramCount = leftAdapter.paramCount
	if err := sift.Thru(ctx, rightAdapter, node.Right); err != nil {
		return err
	}

	a.query = fmt.Sprintf("(%s) AND (%s)", leftAdapter.query, rightAdapter.query)
	a.args = append(leftAdapter.args, rightAdapter.args...)
	a.paramCount = rightAdapter.paramCount
	return nil
}

// EvaluateOr combines two conditions with logical OR.
func (a *Adapter) EvaluateOr(ctx context.Context, node *sift.OrOperation) error {
	leftAdapter := NewAdapterWithConfig(a.config)
	if err := sift.Thru(ctx, leftAdapter, node.Left); err != nil {
		return err
	}

	rightAdapter := NewAdapterWithConfig(a.config)
	rightAdapter.paramCount = leftAdapter.paramCount
	if err := sift.Thru(ctx, rightAdapter, node.Right); err != nil {
		return err
	}

	a.query = fmt.Sprintf("(%s) OR (%s)", leftAdapter.query, rightAdapter.query)
	a.args = append(leftAdapter.args, rightAdapter.args...)
	a.paramCount = rightAdapter.paramCount
	return nil
}

// EvaluateNot negates a condition.
func (a *Adapter) EvaluateNot(ctx context.Context, node *sift.NotOperation) error {
	childAdapter := NewAdapterWithConfig(a.config)
	if err := sift.Thru(ctx, childAdapter, node.Child); err != nil {
		return err
	}

	a.query = fmt.Sprintf("NOT (%s)", childAdapter.query)
	a.args = childAdapter.args
	a.paramCount = childAdapter.paramCount
	return nil
}

// Query returns the generated SQL WHERE clause (without the WHERE keyword).
func (a *Adapter) Query() string {
	return a.query
}

// Args returns the parameter values for the query.
func (a *Adapter) Args() []interface{} {
	return a.args
}

// nextPlaceholder returns the next parameter placeholder based on the dialect.
func (a *Adapter) nextPlaceholder() string {
	a.paramCount++
	switch a.config.Dialect {
	case DialectPostgreSQL:
		return fmt.Sprintf("$%d", a.paramCount)
	case DialectSQLServer:
		return fmt.Sprintf("@p%d", a.paramCount)
	case DialectMySQL, DialectSQLite:
		return "?"
	default:
		return "?"
	}
}

// quoteIdentifier quotes a column name if QuoteIdentifiers is enabled.
func (a *Adapter) quoteIdentifier(name string) string {
	if !a.config.QuoteIdentifiers {
		return name
	}

	switch a.config.Dialect {
	case DialectMySQL:
		return "`" + name + "`"
	case DialectPostgreSQL, DialectSQLite, DialectSQLServer:
		return `"` + name + `"`
	default:
		return name
	}
}

// parseValue attempts to parse the string value into the most appropriate type.
func (a *Adapter) parseValue(s string) interface{} {
	// Try parsing as int
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}

	// Try parsing as float
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}

	// Try parsing as bool
	if b, err := strconv.ParseBool(s); err == nil {
		return b
	}

	// Fall back to string
	return s
}
