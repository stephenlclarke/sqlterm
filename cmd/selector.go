package main

import (
	"errors"
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jpxcz/sqlterm/databases"
)

var errSelectionCancelled = errors.New("database selection cancelled")

func runDatabaseSelector(input io.Reader, output io.Writer, dbs []databases.DatabaseCredentials) (databases.DatabaseCredentials, error) {
	program := tea.NewProgram(
		newDatabaseSelectionModel(dbs),
		tea.WithInput(input),
		tea.WithOutput(output),
	)

	model, err := program.Run()
	if err != nil {
		return databases.DatabaseCredentials{}, fmt.Errorf("database selection failed: %w", err)
	}

	selection, ok := model.(databaseSelectionModel)
	if !ok {
		return databases.DatabaseCredentials{}, errors.New("database selection returned an unexpected model")
	}

	return selection.selectedDatabase()
}

type databaseSelectionModel struct {
	databases     []databases.DatabaseCredentials
	cursor        int
	selectedIndex int
	err           error
}

func newDatabaseSelectionModel(dbs []databases.DatabaseCredentials) databaseSelectionModel {
	return databaseSelectionModel{
		databases:     dbs,
		selectedIndex: -1,
	}
}

func (m databaseSelectionModel) Init() tea.Cmd {
	return nil
}

func (m databaseSelectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "ctrl+c", "esc", "q":
		m.err = errSelectionCancelled
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.databases)-1 {
			m.cursor++
		}
	case "home":
		m.cursor = 0
	case "end":
		if len(m.databases) > 0 {
			m.cursor = len(m.databases) - 1
		}
	case "enter", " ":
		if len(m.databases) > 0 {
			m.selectedIndex = m.cursor
			return m, tea.Quit
		}
	default:
		if index, ok := numericSelection(key.String(), len(m.databases)); ok {
			m.cursor = index
			m.selectedIndex = index
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m databaseSelectionModel) View() string {
	if len(m.databases) == 0 {
		return "No databases are configured.\n"
	}

	var builder strings.Builder
	builder.WriteString("Select a database to connect\n\n")
	for index, db := range m.databases {
		cursor := " "
		if index == m.cursor {
			cursor = ">"
		}
		fmt.Fprintf(&builder, "%s [%d] %s", cursor, index+1, db.Key)
		if db.ShortName != "" {
			fmt.Fprintf(&builder, " - %s", db.ShortName)
		}
		if db.Hostname != "" {
			fmt.Fprintf(&builder, " (%s)", db.Hostname)
		}
		builder.WriteByte('\n')
	}
	builder.WriteString("\nUse up/down or j/k to move, enter to connect, q to quit.\n")

	return builder.String()
}

func (m databaseSelectionModel) selectedDatabase() (databases.DatabaseCredentials, error) {
	if m.err != nil {
		return databases.DatabaseCredentials{}, m.err
	}
	if m.selectedIndex < 0 || m.selectedIndex >= len(m.databases) {
		return databases.DatabaseCredentials{}, errSelectionCancelled
	}

	return m.databases[m.selectedIndex], nil
}

func numericSelection(key string, databaseCount int) (int, bool) {
	if len(key) != 1 || key[0] < '1' || key[0] > '9' {
		return 0, false
	}

	index := int(key[0] - '1')
	if index >= databaseCount {
		return 0, false
	}

	return index, true
}
