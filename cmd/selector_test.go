package main

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jpxcz/sqlterm/databases"
)

func TestDatabaseSelectionModelSelectsHighlightedDatabase(t *testing.T) {
	model := newDatabaseSelectionModel(testDatabases())

	model = updateSelectionModel(t, model, tea.KeyMsg{Type: tea.KeyDown})
	model = updateSelectionModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	selected, err := model.selectedDatabase()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected.Key != "prod" {
		t.Fatalf("expected prod database, got %#v", selected)
	}
}

func TestDatabaseSelectionModelInit(t *testing.T) {
	model := newDatabaseSelectionModel(testDatabases())

	if cmd := model.Init(); cmd != nil {
		t.Fatalf("expected no init command, got %#v", cmd)
	}
}

func TestDatabaseSelectionModelIgnoresNonKeyMessages(t *testing.T) {
	model := newDatabaseSelectionModel(testDatabases())

	model = updateSelectionModel(t, model, tea.WindowSizeMsg{Width: 120, Height: 40})

	if model.cursor != 0 || model.selectedIndex != -1 || model.err != nil {
		t.Fatalf("expected non-key message to leave model unchanged: %#v", model)
	}
}

func TestDatabaseSelectionModelSupportsVimKeysAndBounds(t *testing.T) {
	model := newDatabaseSelectionModel(testDatabases())

	model = updateSelectionModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if model.cursor != 0 {
		t.Fatalf("expected cursor to stay at top, got %d", model.cursor)
	}

	model = updateSelectionModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = updateSelectionModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = updateSelectionModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if model.cursor != 1 {
		t.Fatalf("expected cursor to stay at bottom, got %d", model.cursor)
	}

	model = updateSelectionModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if model.cursor != 0 {
		t.Fatalf("expected cursor to move back to top, got %d", model.cursor)
	}
}

func TestDatabaseSelectionModelSupportsHomeEndAndNumericSelection(t *testing.T) {
	model := newDatabaseSelectionModel(testDatabases())

	model = updateSelectionModel(t, model, tea.KeyMsg{Type: tea.KeyEnd})
	if model.cursor != 1 {
		t.Fatalf("expected cursor at end, got %d", model.cursor)
	}

	model = updateSelectionModel(t, model, tea.KeyMsg{Type: tea.KeyHome})
	if model.cursor != 0 {
		t.Fatalf("expected cursor at home, got %d", model.cursor)
	}

	model = updateSelectionModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	selected, err := model.selectedDatabase()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected.Key != "prod" {
		t.Fatalf("expected numeric selection to choose prod, got %#v", selected)
	}
}

func TestDatabaseSelectionModelIgnoresOutOfRangeNumericSelection(t *testing.T) {
	model := newDatabaseSelectionModel(testDatabases())

	model = updateSelectionModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'9'}})

	if _, err := model.selectedDatabase(); !errors.Is(err, errSelectionCancelled) {
		t.Fatalf("expected no database to be selected, got %v", err)
	}
}

func TestDatabaseSelectionModelCancels(t *testing.T) {
	model := newDatabaseSelectionModel(testDatabases())

	model = updateSelectionModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	if _, err := model.selectedDatabase(); !errors.Is(err, errSelectionCancelled) {
		t.Fatalf("expected cancellation error, got %v", err)
	}
}

func TestDatabaseSelectionModelView(t *testing.T) {
	model := newDatabaseSelectionModel(testDatabases())
	view := model.View()

	for _, want := range []string{
		"Select a database to connect",
		"> [1] dev - Development (localhost)",
		"  [2] prod - Production (prod.example.com)",
		"Use up/down or j/k to move",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected %q in view:\n%s", want, view)
		}
	}
}

func TestDatabaseSelectionModelEmptyView(t *testing.T) {
	model := newDatabaseSelectionModel(nil)

	if view := model.View(); !strings.Contains(view, "No databases are configured") {
		t.Fatalf("unexpected empty view: %q", view)
	}
	if _, err := model.selectedDatabase(); !errors.Is(err, errSelectionCancelled) {
		t.Fatalf("expected empty selection to cancel, got %v", err)
	}
}

func TestNumericSelection(t *testing.T) {
	if index, ok := numericSelection("1", 2); !ok || index != 0 {
		t.Fatalf("expected first item selection, got %d %v", index, ok)
	}
	for _, key := range []string{"0", "3", "10", "x"} {
		if index, ok := numericSelection(key, 2); ok {
			t.Fatalf("expected %q to be rejected, got %d", key, index)
		}
	}
}

func updateSelectionModel(t *testing.T, model databaseSelectionModel, msg tea.Msg) databaseSelectionModel {
	t.Helper()

	updated, _ := model.Update(msg)
	selection, ok := updated.(databaseSelectionModel)
	if !ok {
		t.Fatalf("expected databaseSelectionModel, got %T", updated)
	}

	return selection
}

func testDatabases() []databases.DatabaseCredentials {
	return []databases.DatabaseCredentials{
		{
			Key:       "dev",
			ShortName: "Development",
			Username:  "devuser",
			Hostname:  "localhost",
			Password:  "secret",
			Port:      "3307",
		},
		{
			Key:       "prod",
			ShortName: "Production",
			Username:  "produser",
			Hostname:  "prod.example.com",
			Password:  "secret",
			Port:      "3306",
		},
	}
}
