package odbcclient

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jpxcz/sqlterm/databases"
)

func TestConnectionStringUsesExplicitConnectionString(t *testing.T) {
	got, err := ConnectionString(databases.DatabaseCredentials{ConnectionString: "DSN=demo;UID=user;"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "DSN=demo;UID=user;" {
		t.Fatalf("unexpected connection string: %q", got)
	}
}

func TestConnectionStringBuildsDSN(t *testing.T) {
	got, err := ConnectionString(databases.DatabaseCredentials{
		DSN:      "Trading",
		Username: "user",
		Password: "p;ass",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"DSN=Trading", "UID=user", "PWD={p;ass}"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in %q", want, got)
		}
	}
}

func TestConnectionStringBuildsDriverEndpoint(t *testing.T) {
	got, err := ConnectionString(databases.DatabaseCredentials{
		Driver:   "ODBC Driver 18 for SQL Server",
		Hostname: "db.example.com",
		Port:     "1433",
		Database: "orders",
		Username: "user",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"Driver=ODBC Driver 18 for SQL Server", "Server=db.example.com", "Port=1433", "Database=orders", "UID=user"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in %q", want, got)
		}
	}
}

func TestConnectionStringUsesHostnameAsDSNForLegacyConfigs(t *testing.T) {
	got, err := ConnectionString(databases.DatabaseCredentials{
		Hostname: "LegacyDSN",
		Password: "pa}ss",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"DSN=LegacyDSN", "PWD={pa}}ss}"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in %q", want, got)
		}
	}
}

func TestConnectionStringRequiresEndpoint(t *testing.T) {
	_, err := ConnectionString(databases.DatabaseCredentials{Username: "user"})
	if err == nil || !strings.Contains(err.Error(), "requires") {
		t.Fatalf("expected endpoint error, got %v", err)
	}
}

func TestNewClientUsesDefaultOpener(t *testing.T) {
	client := NewClient()
	if client.open == nil {
		t.Fatal("expected default opener")
	}
}

func TestClientExecuteUsesDefaultOpenerWhenUnset(t *testing.T) {
	client := Client{}

	_, err := client.Execute(context.Background(), databases.DatabaseCredentials{}, "SELECT 1")
	if err == nil || !strings.Contains(err.Error(), "requires") {
		t.Fatalf("expected config error before opening connection, got %v", err)
	}
}

func TestClientExecuteQueryReturnsRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT id, name FROM users").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow("1", "Ada").AddRow("2", nil))

	client := NewClientWithOpener(func(driverName string, dataSourceName string) (*sql.DB, error) {
		if driverName != driverNameConstForTest() {
			t.Fatalf("unexpected driver: %s", driverName)
		}
		if !strings.Contains(dataSourceName, "DSN=Trading") {
			t.Fatalf("unexpected data source: %s", dataSourceName)
		}
		return db, nil
	})

	result, err := client.Execute(context.Background(), databases.DatabaseCredentials{DSN: "Trading"}, "SELECT id, name FROM users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Rows) != 2 || result.Rows[1][1] != "NULL" || result.RowsAffected != 2 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestClientExecuteQueryErrorIncludesContext(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT id FROM users").
		WillReturnError(errors.New("query failed"))

	client := NewClientWithOpener(func(string, string) (*sql.DB, error) {
		return db, nil
	})

	result, err := client.Execute(context.Background(), databases.DatabaseCredentials{DSN: "Trading"}, "SELECT id FROM users")
	if err == nil || !strings.Contains(err.Error(), "execute query") {
		t.Fatalf("expected query error, got %v", err)
	}
	if result.Statement != "SELECT id FROM users" || result.Duration == 0 {
		t.Fatalf("expected partial result metadata, got %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestClientExecuteStatementReturnsRowsAffected(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectExec("UPDATE users SET active = 1").
		WillReturnResult(sqlmock.NewResult(0, 3))

	client := NewClientWithOpener(func(string, string) (*sql.DB, error) {
		return db, nil
	})

	result, err := client.Execute(context.Background(), databases.DatabaseCredentials{DSN: "Trading"}, "UPDATE users SET active = 1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RowsAffected != 3 {
		t.Fatalf("expected 3 affected rows, got %d", result.RowsAffected)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestClientExecuteStatementErrorIncludesContext(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectExec("DELETE FROM users").
		WillReturnError(errors.New("delete failed"))

	client := NewClientWithOpener(func(string, string) (*sql.DB, error) {
		return db, nil
	})

	_, err = client.Execute(context.Background(), databases.DatabaseCredentials{DSN: "Trading"}, "DELETE FROM users")
	if err == nil || !strings.Contains(err.Error(), "execute statement") {
		t.Fatalf("expected statement error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestClientExecuteOpenError(t *testing.T) {
	want := errors.New("open failed")
	client := NewClientWithOpener(func(string, string) (*sql.DB, error) {
		return nil, want
	})

	_, err := client.Execute(context.Background(), databases.DatabaseCredentials{DSN: "Trading"}, "SELECT 1")
	if !errors.Is(err, want) {
		t.Fatalf("expected open error, got %v", err)
	}
}

func TestReturnsRows(t *testing.T) {
	for _, statement := range []string{"select 1", "WITH cte AS (SELECT 1) SELECT * FROM cte", "show tables"} {
		if !returnsRows(statement) {
			t.Fatalf("expected %q to return rows", statement)
		}
	}
	if returnsRows("   ") {
		t.Fatal("expected empty statement to avoid row query path")
	}
	if returnsRows("update users set active = 1") {
		t.Fatal("expected update to be treated as a statement")
	}
}

func TestFormatDuration(t *testing.T) {
	if got := FormatDuration(500 * time.Microsecond); got != "500us" {
		t.Fatalf("unexpected microsecond duration: %q", got)
	}
	if got := FormatDuration(1500 * time.Millisecond); got != "1.5s" {
		t.Fatalf("unexpected millisecond duration: %q", got)
	}
}

func driverNameConstForTest() string {
	return driverName
}
