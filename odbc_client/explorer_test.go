package odbcclient

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jpxcz/sqlterm/databases"
)

func TestClientFetchExplorerBuildsMetadataSnapshot(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery(sqlServerDatabasesQuery).
		WillReturnError(errors.New("not sql server"))
	mock.ExpectQuery(schemataQuery).
		WillReturnRows(sqlmock.NewRows([]string{"schema_name"}).AddRow("Trading"))
	mock.ExpectQuery(tablesQuery).
		WillReturnRows(sqlmock.NewRows([]string{"table_catalog", "table_schema", "table_name", "table_type"}).
			AddRow("def", "Trading", "orders", "BASE TABLE").
			AddRow("def", "Trading", "open_orders", "VIEW"))
	mock.ExpectQuery(columnsQuery).
		WillReturnRows(sqlmock.NewRows([]string{"table_catalog", "table_schema", "table_name", "column_name", "data_type", "is_nullable", "ordinal_position"}).
			AddRow("def", "Trading", "orders", "id", "int", "NO", "1").
			AddRow("def", "Trading", "orders", "customer_name", "varchar", "YES", "2").
			AddRow("def", "Trading", "open_orders", "id", "int", "NO", "1"))
	mock.ExpectQuery(mysqlIndexesQuery).
		WillReturnRows(sqlmock.NewRows([]string{"table_catalog", "table_schema", "table_name", "index_name", "column_name", "is_unique"}).
			AddRow("Trading", "Trading", "orders", "PRIMARY", "id", "YES"))
	mock.ExpectQuery(routinesQuery).
		WillReturnRows(sqlmock.NewRows([]string{"routine_catalog", "routine_schema", "routine_name", "routine_type", "data_type"}).
			AddRow("def", "Trading", "calc_total", "FUNCTION", "decimal"))

	client := NewClientWithOpener(func(driverName string, dataSourceName string) (*sql.DB, error) {
		if driverName != driverNameConstForTest() {
			t.Fatalf("unexpected driver: %s", driverName)
		}
		if !strings.Contains(dataSourceName, "DSN=Trading") {
			t.Fatalf("unexpected data source: %s", dataSourceName)
		}
		return db, nil
	})

	snapshot, err := client.FetchExplorer(context.Background(), databases.DatabaseCredentials{DSN: "Trading"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snapshot.Warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", snapshot.Warnings)
	}
	database := findExplorerDatabase(t, snapshot, "Trading")
	if len(database.Tables) != 1 || database.Tables[0].Name != "orders" {
		t.Fatalf("unexpected tables: %#v", database.Tables)
	}
	if len(database.Tables[0].Columns) != 2 || database.Tables[0].Columns[0].Name != "id" {
		t.Fatalf("unexpected table columns: %#v", database.Tables[0].Columns)
	}
	if len(database.Views) != 1 || database.Views[0].Name != "open_orders" || len(database.Views[0].Columns) != 1 {
		t.Fatalf("unexpected views: %#v", database.Views)
	}
	if len(database.Indexes) != 1 || database.Indexes[0].Name != "PRIMARY" || !database.Indexes[0].Unique {
		t.Fatalf("unexpected indexes: %#v", database.Indexes)
	}
	if len(database.Tables[0].Indexes) != 1 || database.Tables[0].Indexes[0].Columns[0] != "id" {
		t.Fatalf("unexpected table indexes: %#v", database.Tables[0].Indexes)
	}
	if len(database.Functions) != 1 || database.Functions[0].Name != "calc_total" {
		t.Fatalf("unexpected functions: %#v", database.Functions)
	}
	for _, want := range []string{"decimal", "int", "varchar"} {
		if !hasExplorerDataType(database, want) {
			t.Fatalf("expected data type %q in %#v", want, database.DataTypes)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestClientFetchExplorerKeepsFallbackDatabaseWhenMetadataUnsupported(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for _, statement := range []string{
		sqlServerDatabasesQuery,
		schemataQuery,
		tablesQuery,
		columnsQuery,
		mysqlIndexesQuery,
		sqlServerIndexesQuery,
		routinesQuery,
	} {
		mock.ExpectQuery(statement).WillReturnError(errors.New("unsupported metadata query"))
	}

	client := NewClientWithOpener(func(string, string) (*sql.DB, error) {
		return db, nil
	})

	snapshot, err := client.FetchExplorer(context.Background(), databases.DatabaseCredentials{DSN: "Trading"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	findExplorerDatabase(t, snapshot, "Trading")
	if len(snapshot.Warnings) != 5 {
		t.Fatalf("expected warnings for unsupported categories, got %#v", snapshot.Warnings)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestClientFetchExplorerReturnsOpenError(t *testing.T) {
	want := errors.New("open failed")
	client := NewClientWithOpener(func(string, string) (*sql.DB, error) {
		return nil, want
	})

	_, err := client.FetchExplorer(context.Background(), databases.DatabaseCredentials{DSN: "Trading"})
	if !errors.Is(err, want) {
		t.Fatalf("expected open error, got %v", err)
	}
}

func TestExplorerHelperFunctions(t *testing.T) {
	if got := databaseName("def", "Trading", "fallback"); got != "Trading" {
		t.Fatalf("expected schema database name, got %q", got)
	}
	if got := databaseName("catalog", "schema", "fallback"); got != "catalog" {
		t.Fatalf("expected catalog database name, got %q", got)
	}
	if got := databaseName("", "", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback database name, got %q", got)
	}
	if got := fallbackDatabaseName(databases.DatabaseCredentials{Database: "orders", DSN: "Trading"}); got != "orders" {
		t.Fatalf("expected database field fallback first, got %q", got)
	}
	if got := fallbackDatabaseName(databases.DatabaseCredentials{Key: "dev"}); got != "dev" {
		t.Fatalf("expected key fallback, got %q", got)
	}
	if got := fallbackDatabaseName(databases.DatabaseCredentials{}); got != "connected" {
		t.Fatalf("expected connected fallback, got %q", got)
	}
	if !isTruthy("true") || isTruthy("no") {
		t.Fatal("unexpected truthy parsing")
	}
	if got := qualifiedName("dbo", "Orders"); got != "dbo.orders" {
		t.Fatalf("unexpected qualified name: %q", got)
	}
	if got := qualifiedName("", "Orders"); got != "orders" {
		t.Fatalf("unexpected unqualified name: %q", got)
	}
}

func findExplorerDatabase(t *testing.T, snapshot ExplorerSnapshot, name string) ExplorerDatabase {
	t.Helper()

	for _, database := range snapshot.Databases {
		if database.Name == name {
			return database
		}
	}
	t.Fatalf("expected database %q in %#v", name, snapshot.Databases)
	return ExplorerDatabase{}
}

func hasExplorerDataType(database ExplorerDatabase, name string) bool {
	for _, dataType := range database.DataTypes {
		if dataType.Name == name {
			return true
		}
	}
	return false
}
