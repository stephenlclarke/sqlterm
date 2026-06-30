package odbcclient

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "github.com/alexbrainman/odbc"
	"github.com/jpxcz/sqlterm/databases"
)

const driverName = "odbc"

type Opener func(driverName string, dataSourceName string) (*sql.DB, error)

type Client struct {
	open Opener
}

type QueryResult struct {
	Statement    string
	Columns      []string
	Rows         [][]string
	RowsAffected int64
	Duration     time.Duration
	ExecutedAt   time.Time
}

func NewClient() Client {
	return Client{open: sql.Open}
}

func NewClientWithOpener(open Opener) Client {
	return Client{open: open}
}

func (c Client) Execute(ctx context.Context, db databases.DatabaseCredentials, statement string) (QueryResult, error) {
	if c.open == nil {
		c.open = sql.Open
	}

	connectionString, err := ConnectionString(db)
	if err != nil {
		return QueryResult{}, err
	}

	connection, err := c.open(driverName, connectionString)
	if err != nil {
		return QueryResult{}, fmt.Errorf("open ODBC connection: %w", err)
	}
	defer connection.Close()

	started := time.Now()
	result := QueryResult{
		Statement:  statement,
		ExecutedAt: started,
	}

	if returnsRows(statement) {
		rows, err := connection.QueryContext(ctx, statement)
		result.Duration = time.Since(started)
		if err != nil {
			return result, fmt.Errorf("execute query: %w", err)
		}
		defer rows.Close()

		result.Columns, result.Rows, err = readRows(rows)
		if err != nil {
			return result, err
		}
		result.RowsAffected = int64(len(result.Rows))
		return result, nil
	}

	execResult, err := connection.ExecContext(ctx, statement)
	result.Duration = time.Since(started)
	if err != nil {
		return result, fmt.Errorf("execute statement: %w", err)
	}

	rowsAffected, err := execResult.RowsAffected()
	if err == nil {
		result.RowsAffected = rowsAffected
	}

	return result, nil
}

func ConnectionString(db databases.DatabaseCredentials) (string, error) {
	if strings.TrimSpace(db.ConnectionString) != "" {
		return db.ConnectionString, nil
	}

	parts := make([]string, 0, 6)
	if db.DSN != "" {
		parts = append(parts, connectionPart("DSN", db.DSN))
	} else if db.Driver != "" {
		parts = append(parts, connectionPart("Driver", db.Driver))
		if db.Hostname != "" {
			parts = append(parts, connectionPart("Server", db.Hostname))
		}
		if db.Port != "" {
			parts = append(parts, connectionPart("Port", db.Port))
		}
		if db.Database != "" {
			parts = append(parts, connectionPart("Database", db.Database))
		}
	} else if db.Hostname != "" {
		parts = append(parts, connectionPart("DSN", db.Hostname))
	}

	if len(parts) == 0 {
		return "", errors.New("database config requires dsn, connection_string, driver, or hostname")
	}
	if db.Username != "" {
		parts = append(parts, connectionPart("UID", db.Username))
	}
	if db.Password != "" {
		parts = append(parts, connectionPart("PWD", db.Password))
	}

	return strings.Join(parts, ";") + ";", nil
}

func connectionPart(key string, value string) string {
	if strings.ContainsAny(value, ";{}") {
		return key + "={" + strings.ReplaceAll(value, "}", "}}") + "}"
	}

	return key + "=" + value
}

func readRows(rows *sql.Rows) ([]string, [][]string, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, fmt.Errorf("read result columns: %w", err)
	}

	values := make([]sql.NullString, len(columns))
	destinations := make([]any, len(columns))
	for index := range values {
		destinations[index] = &values[index]
	}

	var resultRows [][]string
	for rows.Next() {
		if err := rows.Scan(destinations...); err != nil {
			return nil, nil, fmt.Errorf("scan result row: %w", err)
		}

		row := make([]string, len(columns))
		for index, value := range values {
			if value.Valid {
				row[index] = value.String
			} else {
				row[index] = "NULL"
			}
		}
		resultRows = append(resultRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("read result rows: %w", err)
	}

	return columns, resultRows, nil
}

func returnsRows(statement string) bool {
	fields := strings.Fields(strings.TrimSpace(statement))
	if len(fields) == 0 {
		return false
	}

	switch strings.ToUpper(fields[0]) {
	case "SELECT", "SHOW", "DESCRIBE", "DESC", "EXPLAIN", "WITH":
		return true
	default:
		return false
	}
}

func FormatDuration(duration time.Duration) string {
	if duration < time.Millisecond {
		return strconv.FormatInt(duration.Microseconds(), 10) + "us"
	}

	return duration.Round(time.Millisecond).String()
}
