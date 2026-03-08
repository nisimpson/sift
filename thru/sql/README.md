# Sift SQL Adapter

SQL adapter for the [Sift](https://github.com/nisimpson/sift) universal query filter library. Translates Sift filter expressions into SQL WHERE clause syntax.

## Features

- **Multiple SQL Dialects**: PostgreSQL, MySQL, SQLite, SQL Server
- **Parameterized Queries**: Automatic parameter binding to prevent SQL injection
- **Flexible Configuration**: Quote identifiers, case-sensitive/insensitive matching
- **Type Conversion**: Automatic type detection for numbers, booleans, and strings
- **All Sift Operations**: Full support for comparison, string, collection, and logical operations

## Installation

```bash
go get github.com/nisimpson/sift/thru/sql
```

## Quick Start

```go
import (
    "context"
    "github.com/nisimpson/sift"
    siftsql "github.com/nisimpson/sift/thru/sql"
)

// Create a filter
filter := sift.Eq("status", "active").And(sift.Gt("age", 18))

// Create SQL adapter
adapter := siftsql.NewAdapter()
sift.Thru(context.Background(), adapter, filter)

// Use with database/sql
query := fmt.Sprintf("SELECT * FROM users WHERE %s", adapter.Query())
rows, err := db.Query(query, adapter.Args()...)
```

## Supported Dialects

### PostgreSQL (Default)

Uses `$1`, `$2`, etc. for parameter placeholders:

```go
adapter := siftsql.NewAdapter() // Defaults to PostgreSQL
// Query: status = $1 AND age > $2
```

### MySQL

Uses `?` for parameter placeholders:

```go
config := &siftsql.Config{
    Dialect: siftsql.DialectMySQL,
}
adapter := siftsql.NewAdapterWithConfig(config)
// Query: status = ? AND age > ?
```

### SQLite

Uses `?` for parameter placeholders:

```go
config := &siftsql.Config{
    Dialect: siftsql.DialectSQLite,
}
adapter := siftsql.NewAdapterWithConfig(config)
// Query: status = ? AND age > ?
```

### SQL Server

Uses `@p1`, `@p2`, etc. for parameter placeholders:

```go
config := &siftsql.Config{
    Dialect: siftsql.DialectSQLServer,
}
adapter := siftsql.NewAdapterWithConfig(config)
// Query: status = @p1 AND age > @p2
```

## Configuration Options

```go
type Config struct {
    // Dialect specifies the SQL dialect (PostgreSQL, MySQL, SQLite, SQLServer)
    Dialect Dialect

    // QuoteIdentifiers determines whether to quote column names
    // PostgreSQL/SQLite use double quotes, MySQL uses backticks
    QuoteIdentifiers bool

    // CaseSensitive determines whether string comparisons are case-sensitive
    // When false, uses LOWER() for case-insensitive comparisons
    CaseSensitive bool
}
```

### Quote Identifiers

Useful for column names with special characters or reserved words:

```go
config := &siftsql.Config{
    Dialect:          siftsql.DialectPostgreSQL,
    QuoteIdentifiers: true,
}
adapter := siftsql.NewAdapterWithConfig(config)

filter := sift.Eq("user_name", "john")
sift.Thru(context.Background(), adapter, filter)
// Query: "user_name" = $1
```

MySQL uses backticks:

```go
config := &siftsql.Config{
    Dialect:          siftsql.DialectMySQL,
    QuoteIdentifiers: true,
}
// Query: `user_name` = ?
```

### Case-Insensitive Matching

For case-insensitive string operations:

```go
config := &siftsql.Config{
    CaseSensitive: false,
}
adapter := siftsql.NewAdapterWithConfig(config)

filter := sift.Contains("email", "@EXAMPLE.COM")
sift.Thru(context.Background(), adapter, filter)
// Query: LOWER(email) LIKE LOWER($1)
```

## Supported Operations

### Comparison Operations

| Sift Operation | SQL Output | Example |
|----------------|------------|---------|
| `OperationEQ` | `=` | `status = $1` |
| `OperationNEQ` | `!=` | `status != $1` |
| `OperationLT` | `<` | `age < $1` |
| `OperationLTE` | `<=` | `age <= $1` |
| `OperationGT` | `>` | `age > $1` |
| `OperationGTE` | `>=` | `age >= $1` |

### String Operations

| Sift Operation | SQL Output | Example |
|----------------|------------|---------|
| `OperationContains` | `LIKE` | `email LIKE $1` (with `%value%`) |
| `OperationBeginsWith` | `LIKE` | `name LIKE $1` (with `value%`) |

### Collection Operations

| Sift Operation | SQL Output | Example |
|----------------|------------|---------|
| `OperationIn` | `IN` | `status IN ($1, $2, $3)` |
| `OperationBetween` | `BETWEEN` | `age BETWEEN $1 AND $2` |

### Existence Operations

| Sift Operation | SQL Output | Example |
|----------------|------------|---------|
| `OperationExists` | `IS NOT NULL` | `email IS NOT NULL` |
| `OperationNotExists` | `IS NULL` | `deleted_at IS NULL` |

### Logical Operations

| Sift Operation | SQL Output | Example |
|----------------|------------|---------|
| `AndOperation` | `AND` | `(status = $1) AND (age > $2)` |
| `OrOperation` | `OR` | `(role = $1) OR (role = $2)` |
| `NotOperation` | `NOT` | `NOT (deleted = $1)` |

## Examples

### Simple Condition

```go
filter := sift.Eq("status", "active")

adapter := siftsql.NewAdapter()
sift.Thru(context.Background(), adapter, filter)

// Query: status = $1
// Args: [active]
```

### Complex Filter

```go
// (status = "active" AND age > 18) OR role = "admin"
filter := sift.Eq("status", "active").
    And(sift.Gt("age", 18)).
    Or(sift.Eq("role", "admin"))

adapter := siftsql.NewAdapter()
sift.Thru(context.Background(), adapter, filter)

// Query: ((status = $1) AND (age > $2)) OR (role = $3)
// Args: [active 18 admin]
```

### String Operations

```go
filter := sift.Contains("email", "@example.com").
    And(&sift.Condition{
        Name:      "name",
        Operation: sift.OperationBeginsWith,
        Value:     "John",
    })

adapter := siftsql.NewAdapter()
sift.Thru(context.Background(), adapter, filter)

// Query: (email LIKE $1) AND (name LIKE $2)
// Args: [%@example.com% John%]
```

### IN Operation

```go
filter := &sift.Condition{
    Name:      "status",
    Operation: sift.OperationIn,
    Value:     "active,pending,approved",
}

adapter := siftsql.NewAdapter()
sift.Thru(context.Background(), adapter, filter)

// Query: status IN ($1, $2, $3)
// Args: [active pending approved]
```

### BETWEEN Operation

```go
filter := sift.Between("age", 18, 65)

adapter := siftsql.NewAdapter()
sift.Thru(context.Background(), adapter, filter)

// Query: age BETWEEN $1 AND $2
// Args: [18 65]
```

### NOT Operation

```go
filter := sift.Eq("deleted", "true").Not()

adapter := siftsql.NewAdapter()
sift.Thru(context.Background(), adapter, filter)

// Query: NOT (deleted = $1)
// Args: [true]
```

### With database/sql

```go
import (
    "database/sql"
    _ "github.com/lib/pq" // PostgreSQL driver
)

func FindUsers(db *sql.DB, filter sift.Expression) ([]*User, error) {
    adapter := siftsql.NewAdapter()
    if err := sift.Thru(context.Background(), adapter, filter); err != nil {
        return nil, err
    }

    query := fmt.Sprintf("SELECT id, name, email FROM users WHERE %s", adapter.Query())
    rows, err := db.Query(query, adapter.Args()...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var users []*User
    for rows.Next() {
        var user User
        if err := rows.Scan(&user.ID, &user.Name, &user.Email); err != nil {
            return nil, err
        }
        users = append(users, &user)
    }

    return users, rows.Err()
}
```

### With GORM

```go
import "gorm.io/gorm"

func FindUsers(db *gorm.DB, filter sift.Expression) ([]*User, error) {
    adapter := siftsql.NewAdapter()
    if err := sift.Thru(context.Background(), adapter, filter); err != nil {
        return nil, err
    }

    var users []*User
    result := db.Where(adapter.Query(), adapter.Args()...).Find(&users)
    return users, result.Error
}
```

### With sqlx

```go
import "github.com/jmoiron/sqlx"

func FindUsers(db *sqlx.DB, filter sift.Expression) ([]*User, error) {
    adapter := siftsql.NewAdapter()
    if err := sift.Thru(context.Background(), adapter, filter); err != nil {
        return nil, err
    }

    var users []*User
    query := fmt.Sprintf("SELECT * FROM users WHERE %s", adapter.Query())
    err := db.Select(&users, query, adapter.Args()...)
    return users, err
}
```

## Type Conversion

The adapter automatically converts string values to appropriate types:

```go
filter := sift.Gt("age", 18)
// Args: [int64(18)]

filter := sift.Eq("price", "99.99")
// Args: [float64(99.99)]

filter := sift.Eq("active", "true")
// Args: [bool(true)]

filter := sift.Eq("name", "John Doe")
// Args: ["John Doe"]
```

## Security

The adapter uses parameterized queries to prevent SQL injection:

```go
// Safe - uses parameters
filter := sift.Eq("name", "'; DROP TABLE users; --")
adapter := siftsql.NewAdapter()
sift.Thru(context.Background(), adapter, filter)
// Query: name = $1
// Args: ["'; DROP TABLE users; --"]
```

The malicious input is safely passed as a parameter value, not concatenated into the query string.

## License

MIT License - see [LICENSE](../../LICENSE) file for details
