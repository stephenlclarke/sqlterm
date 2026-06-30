package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jpxcz/sqlterm/databases"
	odbcclient "github.com/jpxcz/sqlterm/odbc_client"
	"github.com/jpxcz/sqlterm/sqltext"
)

type queryExecutor func(context.Context, databases.DatabaseCredentials, string) (odbcclient.QueryResult, error)

type queryExecutedMsg struct {
	result odbcclient.QueryResult
	err    error
}

func runSQLWorkspace(input io.Reader, output io.Writer, db databases.DatabaseCredentials, options workspaceOptions) error {
	client := odbcclient.NewClient()
	return runSQLWorkspaceWithExecutor(input, output, db, options, client.Execute)
}

func runSQLWorkspaceWithExecutor(input io.Reader, output io.Writer, db databases.DatabaseCredentials, options workspaceOptions, executor queryExecutor) error {
	program := tea.NewProgram(
		newSQLWorkspaceModel(db, options, executor),
		tea.WithInput(input),
		tea.WithOutput(output),
	)

	_, err := program.Run()
	return err
}

type sqlWorkspaceModel struct {
	database databases.DatabaseCredentials
	options  workspaceOptions
	editor   textarea.Model
	executor queryExecutor
	result   odbcclient.QueryResult
	status   string
	err      error
	running  bool
}

func newSQLWorkspaceModel(db databases.DatabaseCredentials, options workspaceOptions, executor queryExecutor) sqlWorkspaceModel {
	editor := textarea.New()
	editor.Placeholder = "SELECT * FROM table_name WHERE id = 1"
	editor.Prompt = "sql> "
	editor.SetHeight(6)
	editor.SetWidth(96)
	editor.Focus()

	return sqlWorkspaceModel{
		database: db,
		options:  options,
		editor:   editor,
		executor: executor,
		status:   "Enter SQL, Ctrl+F to format, Ctrl+R to run, Ctrl+C to quit.",
	}
}

func (m sqlWorkspaceModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m sqlWorkspaceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch message := msg.(type) {
	case tea.KeyMsg:
		switch message.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+f":
			return m.formatSQL(), nil
		case "ctrl+r":
			return m.executeSQL()
		}
	case queryExecutedMsg:
		m.running = false
		if message.err != nil {
			m.err = message.err
			m.status = "Query failed."
			return m, nil
		}
		m.err = nil
		m.result = message.result
		m.status = "Query completed."
		return m, nil
	}

	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)
	m.updateValidationStatus()

	return m, cmd
}

func (m sqlWorkspaceModel) View() string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "SQLTerm - %s", m.database.Key)
	if m.database.ShortName != "" {
		fmt.Fprintf(&builder, " (%s)", m.database.ShortName)
	}
	builder.WriteString("\n\n")
	builder.WriteString(m.editor.View())
	builder.WriteString("\n\n")
	builder.WriteString(m.status)
	if m.err != nil {
		builder.WriteString("\nError: ")
		builder.WriteString(m.err.Error())
	}
	if m.running {
		builder.WriteString("\nRunning query...")
	}
	if m.result.Statement != "" {
		builder.WriteString("\n\n")
		builder.WriteString(renderResult(m.result, m.options))
	}
	builder.WriteString("\n\nCtrl+F format/validate | Ctrl+R run | Ctrl+C quit\n")

	return builder.String()
}

func (m sqlWorkspaceModel) formatSQL() sqlWorkspaceModel {
	statement, err := sqltext.Prepare(m.editor.Value())
	m.editor.SetValue(statement)
	if err != nil {
		m.err = err
		m.status = "SQL syntax needs attention."
		return m
	}

	m.err = nil
	m.status = "SQL syntax OK."
	return m
}

func (m sqlWorkspaceModel) executeSQL() (tea.Model, tea.Cmd) {
	m = m.formatSQL()
	if m.err != nil {
		return m, nil
	}
	if m.executor == nil {
		m.err = fmt.Errorf("SQL executor is not configured")
		m.status = "Query failed."
		return m, nil
	}

	statement := m.editor.Value()
	m.running = true
	m.status = "Running query..."

	return m, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		result, err := m.executor(ctx, m.database, statement)
		return queryExecutedMsg{result: result, err: err}
	}
}

func (m *sqlWorkspaceModel) updateValidationStatus() {
	if strings.TrimSpace(m.editor.Value()) == "" {
		m.err = nil
		m.status = "Enter SQL, Ctrl+F to format, Ctrl+R to run, Ctrl+C to quit."
		return
	}

	if err := sqltext.Validate(sqltext.UppercaseKeywords(m.editor.Value())); err != nil {
		m.err = err
		m.status = "SQL syntax needs attention."
		return
	}

	m.err = nil
	m.status = "SQL syntax OK."
}

func renderResult(result odbcclient.QueryResult, options workspaceOptions) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "Executed: %s\n", result.Statement)
	fmt.Fprintf(&builder, "Metrics: duration=%s rows=%d affected=%d columns=%d at=%s\n",
		odbcclient.FormatDuration(result.Duration),
		len(result.Rows),
		result.RowsAffected,
		len(result.Columns),
		result.ExecutedAt.Format(time.RFC3339),
	)

	if len(result.Columns) == 0 {
		return builder.String()
	}

	if options.FormatAsTable {
		builder.WriteString(renderTable(result.Columns, result.Rows))
		return builder.String()
	}

	for _, row := range result.Rows {
		for index, column := range result.Columns {
			if index > 0 {
				builder.WriteString(", ")
			}
			fmt.Fprintf(&builder, "%s=%s", column, row[index])
		}
		builder.WriteByte('\n')
	}

	return builder.String()
}

func renderTable(columns []string, rows [][]string) string {
	widths := make([]int, len(columns))
	for index, column := range columns {
		widths[index] = len(column)
	}
	for _, row := range rows {
		for index, value := range row {
			if len(value) > widths[index] {
				widths[index] = len(value)
			}
		}
	}

	var builder strings.Builder
	writeTableRow(&builder, columns, widths)
	for index, width := range widths {
		if index > 0 {
			builder.WriteString("-+-")
		}
		builder.WriteString(strings.Repeat("-", width))
	}
	builder.WriteByte('\n')
	for _, row := range rows {
		writeTableRow(&builder, row, widths)
	}

	return builder.String()
}

func writeTableRow(builder *strings.Builder, values []string, widths []int) {
	for index, value := range values {
		if index > 0 {
			builder.WriteString(" | ")
		}
		fmt.Fprintf(builder, "%-*s", widths[index], value)
	}
	builder.WriteByte('\n')
}
