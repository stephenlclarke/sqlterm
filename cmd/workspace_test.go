package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jpxcz/sqlterm/databases"
	odbcclient "github.com/jpxcz/sqlterm/odbc_client"
)

func TestSQLWorkspaceFormatsSQL(t *testing.T) {
	model := newSQLWorkspaceModel(workspaceTestDatabase(), workspaceOptions{FormatAsTable: true}, nil)
	model.editor.SetValue("select name from customers where note = 'select'")

	model = model.formatSQL()

	if got := model.editor.Value(); got != "SELECT name FROM customers WHERE note = 'select'" {
		t.Fatalf("unexpected formatted SQL: %q", got)
	}
	if model.err != nil {
		t.Fatalf("unexpected validation error: %v", model.err)
	}
}

func TestSQLWorkspaceRejectsInvalidSQLBeforeExecution(t *testing.T) {
	model := newSQLWorkspaceModel(workspaceTestDatabase(), workspaceOptions{}, func(context.Context, databases.DatabaseCredentials, string) (odbcclient.QueryResult, error) {
		t.Fatal("executor should not be called")
		return odbcclient.QueryResult{}, nil
	})
	model.editor.SetValue("select from")

	updated, cmd := model.executeSQL()
	model = updated.(sqlWorkspaceModel)

	if cmd != nil {
		t.Fatal("expected invalid SQL to avoid execution command")
	}
	if model.err == nil {
		t.Fatal("expected validation error")
	}
}

func TestSQLWorkspaceExecutesQueryAndStoresResult(t *testing.T) {
	model := newSQLWorkspaceModel(workspaceTestDatabase(), workspaceOptions{FormatAsTable: true}, func(_ context.Context, db databases.DatabaseCredentials, statement string) (odbcclient.QueryResult, error) {
		if db.Key != "dev" {
			t.Fatalf("unexpected database: %#v", db)
		}
		if statement != "SELECT id FROM orders" {
			t.Fatalf("unexpected statement: %q", statement)
		}
		return odbcclient.QueryResult{
			Statement:    statement,
			Columns:      []string{"id"},
			Rows:         [][]string{{"42"}},
			RowsAffected: 1,
			Duration:     time.Millisecond,
			ExecutedAt:   time.Unix(100, 0).UTC(),
		}, nil
	})
	model.editor.SetValue("select id from orders")

	updated, cmd := model.executeSQL()
	model = updated.(sqlWorkspaceModel)
	if cmd == nil {
		t.Fatal("expected execution command")
	}

	updated, _ = model.Update(cmd())
	model = updated.(sqlWorkspaceModel)

	if model.err != nil {
		t.Fatalf("unexpected error: %v", model.err)
	}
	if model.result.RowsAffected != 1 || !strings.Contains(model.View(), "42") {
		t.Fatalf("unexpected model result: %#v\n%s", model.result, model.View())
	}
}

func TestSQLWorkspaceStoresExecutionError(t *testing.T) {
	want := errors.New("query failed")
	model := newSQLWorkspaceModel(workspaceTestDatabase(), workspaceOptions{}, func(context.Context, databases.DatabaseCredentials, string) (odbcclient.QueryResult, error) {
		return odbcclient.QueryResult{}, want
	})
	model.editor.SetValue("select id from orders")

	updated, cmd := model.executeSQL()
	model = updated.(sqlWorkspaceModel)
	updated, _ = model.Update(cmd())
	model = updated.(sqlWorkspaceModel)

	if !errors.Is(model.err, want) {
		t.Fatalf("expected execution error, got %v", model.err)
	}
}

func TestSQLWorkspaceUpdateValidationStatus(t *testing.T) {
	model := newSQLWorkspaceModel(workspaceTestDatabase(), workspaceOptions{}, nil)

	model.editor.SetValue("")
	model.updateValidationStatus()
	if model.err != nil || !strings.Contains(model.status, "Enter SQL") {
		t.Fatalf("unexpected empty status: %q %v", model.status, model.err)
	}

	model.editor.SetValue("select id from orders")
	model.updateValidationStatus()
	if model.err != nil || !strings.Contains(model.status, "OK") {
		t.Fatalf("unexpected valid status: %q %v", model.status, model.err)
	}
}

func TestSQLWorkspaceUpdateFormatsWithCtrlF(t *testing.T) {
	model := newSQLWorkspaceModel(workspaceTestDatabase(), workspaceOptions{}, nil)
	model.editor.SetValue("select id from orders")

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlF})
	model = updated.(sqlWorkspaceModel)

	if cmd != nil {
		t.Fatal("expected format action to avoid async command")
	}
	if got := model.editor.Value(); got != "SELECT id FROM orders" {
		t.Fatalf("unexpected formatted SQL: %q", got)
	}
	if model.err != nil || !strings.Contains(model.status, "OK") {
		t.Fatalf("unexpected status: %q %v", model.status, model.err)
	}
}

func TestSQLWorkspaceUpdateRunsWithCtrlR(t *testing.T) {
	model := newSQLWorkspaceModel(workspaceTestDatabase(), workspaceOptions{}, func(context.Context, databases.DatabaseCredentials, string) (odbcclient.QueryResult, error) {
		return odbcclient.QueryResult{Statement: "SELECT 1"}, nil
	})
	model.editor.SetValue("select 1")

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	model = updated.(sqlWorkspaceModel)

	if cmd == nil {
		t.Fatal("expected execution command")
	}
	if !model.running || model.status != "Running query..." {
		t.Fatalf("expected running status, got %t %q", model.running, model.status)
	}
}

func TestSQLWorkspaceUpdateStoresQueryResultMessage(t *testing.T) {
	model := newSQLWorkspaceModel(workspaceTestDatabase(), workspaceOptions{}, nil)
	result := odbcclient.QueryResult{Statement: "SELECT 1"}

	updated, cmd := model.Update(queryExecutedMsg{result: result})
	model = updated.(sqlWorkspaceModel)

	if cmd != nil {
		t.Fatal("expected result message to avoid async command")
	}
	if model.err != nil || model.result.Statement != result.Statement || model.status != "Query completed." {
		t.Fatalf("unexpected model after result: %#v", model)
	}
}

func TestSQLWorkspaceUpdateStoresQueryErrorMessage(t *testing.T) {
	model := newSQLWorkspaceModel(workspaceTestDatabase(), workspaceOptions{}, nil)
	want := errors.New("boom")

	updated, _ := model.Update(queryExecutedMsg{err: want})
	model = updated.(sqlWorkspaceModel)

	if !errors.Is(model.err, want) || model.status != "Query failed." || model.running {
		t.Fatalf("unexpected model after error: %#v", model)
	}
}

func TestSQLWorkspaceUpdateQuit(t *testing.T) {
	model := newSQLWorkspaceModel(workspaceTestDatabase(), workspaceOptions{}, nil)

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
}

func TestSQLWorkspaceExecuteRequiresExecutor(t *testing.T) {
	model := newSQLWorkspaceModel(workspaceTestDatabase(), workspaceOptions{}, nil)
	model.editor.SetValue("select 1")

	updated, cmd := model.executeSQL()
	model = updated.(sqlWorkspaceModel)

	if cmd != nil {
		t.Fatal("expected missing executor to avoid async command")
	}
	if model.err == nil || !strings.Contains(model.err.Error(), "not configured") {
		t.Fatalf("expected executor error, got %v", model.err)
	}
}

func TestSQLWorkspaceInitReturnsBlinkCommand(t *testing.T) {
	model := newSQLWorkspaceModel(workspaceTestDatabase(), workspaceOptions{}, nil)

	if cmd := model.Init(); cmd == nil {
		t.Fatal("expected blink command")
	}
}

func TestSQLWorkspaceViewShowsErrorRunningAndResult(t *testing.T) {
	model := newSQLWorkspaceModel(workspaceTestDatabase(), workspaceOptions{}, nil)
	model.err = errors.New("bad syntax")
	model.running = true
	model.result = odbcclient.QueryResult{
		Statement:  "SELECT 1",
		Columns:    []string{"value"},
		Rows:       [][]string{{"1"}},
		ExecutedAt: time.Unix(100, 0).UTC(),
	}

	view := model.View()
	for _, want := range []string{"SQLTerm - dev (Development)", "Error: bad syntax", "Running query...", "value=1", "Ctrl+F format/validate"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected %q in view:\n%s", want, view)
		}
	}
}

func TestRenderResultTableAndKeyValue(t *testing.T) {
	result := odbcclient.QueryResult{
		Statement:    "SELECT id, name FROM users",
		Columns:      []string{"id", "name"},
		Rows:         [][]string{{"1", "Ada"}},
		RowsAffected: 1,
		Duration:     2 * time.Millisecond,
		ExecutedAt:   time.Unix(100, 0).UTC(),
	}

	table := renderResult(result, workspaceOptions{FormatAsTable: true})
	for _, want := range []string{"Metrics:", "id | name", "1  | Ada"} {
		if !strings.Contains(table, want) {
			t.Fatalf("expected %q in table result:\n%s", want, table)
		}
	}

	keyValue := renderResult(result, workspaceOptions{FormatAsTable: false})
	if !strings.Contains(keyValue, "id=1, name=Ada") {
		t.Fatalf("unexpected key-value result:\n%s", keyValue)
	}
}

func TestRenderResultWithoutColumns(t *testing.T) {
	result := odbcclient.QueryResult{
		Statement:    "UPDATE users SET active = 1",
		RowsAffected: 3,
		Duration:     2 * time.Millisecond,
		ExecutedAt:   time.Unix(100, 0).UTC(),
	}

	rendered := renderResult(result, workspaceOptions{FormatAsTable: true})
	for _, want := range []string{"Executed: UPDATE users SET active = 1", "affected=3", "columns=0"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected %q in rendered result:\n%s", want, rendered)
		}
	}
	if strings.Contains(rendered, "|") {
		t.Fatalf("did not expect table output without columns:\n%s", rendered)
	}
}

func workspaceTestDatabase() databases.DatabaseCredentials {
	return databases.DatabaseCredentials{
		Key:       "dev",
		ShortName: "Development",
		DSN:       "Trading",
		Username:  "user",
		Password:  "secret",
	}
}
