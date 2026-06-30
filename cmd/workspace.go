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
type metadataExplorer func(context.Context, databases.DatabaseCredentials) (odbcclient.ExplorerSnapshot, error)

type queryExecutedMsg struct {
	result odbcclient.QueryResult
	err    error
}

type explorerLoadedMsg struct {
	snapshot odbcclient.ExplorerSnapshot
	err      error
}

type activePane int

const (
	paneExplorer activePane = iota
	paneEditor
)

const (
	defaultWorkspaceWidth  = 124
	defaultWorkspaceHeight = 32
	minEditorWidth         = 48
	explorerPanelWidth     = 38
	panelGutterWidth       = 2
)

func runSQLWorkspace(input io.Reader, output io.Writer, db databases.DatabaseCredentials, options workspaceOptions) error {
	client := odbcclient.NewClient()
	return runSQLWorkspaceWithExecutor(input, output, db, options, client.Execute, client.FetchExplorer)
}

func runSQLWorkspaceWithExecutor(input io.Reader, output io.Writer, db databases.DatabaseCredentials, options workspaceOptions, executor queryExecutor, explorer metadataExplorer) error {
	program := tea.NewProgram(
		newSQLWorkspaceModel(db, options, executor, explorer),
		tea.WithInput(input),
		tea.WithOutput(output),
	)

	_, err := program.Run()
	return err
}

type sqlWorkspaceModel struct {
	database        databases.DatabaseCredentials
	options         workspaceOptions
	editor          textarea.Model
	executor        queryExecutor
	explorer        metadataExplorer
	tree            explorerTree
	result          odbcclient.QueryResult
	status          string
	explorerStatus  string
	explorerErr     error
	explorerLoading bool
	err             error
	running         bool
	activePane      activePane
	width           int
	height          int
}

func newSQLWorkspaceModel(db databases.DatabaseCredentials, options workspaceOptions, executor queryExecutor, explorer metadataExplorer) sqlWorkspaceModel {
	editor := textarea.New()
	editor.Placeholder = "SELECT * FROM table_name WHERE id = 1"
	editor.Prompt = "sql> "
	editor.SetHeight(6)
	editor.SetWidth(defaultWorkspaceWidth - explorerPanelWidth - panelGutterWidth)
	editor.Focus()

	return sqlWorkspaceModel{
		database:        db,
		options:         options,
		editor:          editor,
		executor:        executor,
		explorer:        explorer,
		tree:            emptyExplorerTree(),
		status:          "Enter SQL, Ctrl+F to format, Ctrl+R to run, Ctrl+C to quit.",
		explorerStatus:  "Loading database explorer...",
		explorerLoading: explorer != nil,
		activePane:      paneEditor,
		width:           defaultWorkspaceWidth,
		height:          defaultWorkspaceHeight,
	}
}

func (m sqlWorkspaceModel) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.loadExplorer())
}

func (m sqlWorkspaceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch message := msg.(type) {
	case tea.KeyMsg:
		switch message.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.toggleActivePane()
			return m, nil
		case "ctrl+e":
			m.explorerLoading = true
			m.explorerErr = nil
			m.explorerStatus = "Refreshing database explorer..."
			return m, m.loadExplorer()
		case "ctrl+f":
			return m.formatSQL(), nil
		case "ctrl+r":
			return m.executeSQL()
		}
		if m.activePane == paneExplorer {
			return m.updateExplorer(message)
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
	case explorerLoadedMsg:
		m.explorerLoading = false
		if message.err != nil {
			m.explorerErr = message.err
			m.explorerStatus = "Explorer metadata unavailable."
			return m, nil
		}
		m.explorerErr = nil
		m.tree = newExplorerTree(message.snapshot)
		m.explorerStatus = explorerStatus(message.snapshot)
		return m, nil
	case tea.WindowSizeMsg:
		m.resize(message.Width, message.Height)
	}

	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)
	m.updateValidationStatus()

	return m, cmd
}

func (m sqlWorkspaceModel) View() string {
	left := m.renderExplorerPanel()
	right := m.renderQueryPanel()
	return joinPanels(left, right, explorerPanelWidth, panelGutterWidth)
}

func (m sqlWorkspaceModel) renderQueryPanel() string {
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
	builder.WriteString("\n\nTab explorer | Ctrl+F format/validate | Ctrl+R run | Ctrl+C quit\n")

	return builder.String()
}

func (m sqlWorkspaceModel) renderExplorerPanel() string {
	maxRows := m.height - 8
	if maxRows < 8 {
		maxRows = 8
	}
	lines := []string{"Explorer"}
	if m.activePane == paneExplorer {
		lines[0] += " *"
	}
	lines = append(lines, strings.Repeat("-", explorerPanelWidth))
	switch {
	case m.explorerLoading:
		lines = append(lines, "Loading metadata...")
	case m.explorerErr != nil:
		lines = append(lines, "Unavailable: "+m.explorerErr.Error())
	default:
		lines = append(lines, m.explorerStatus)
	}
	lines = append(lines, "")
	lines = append(lines, renderTreeRows(m.tree, maxRows)...)
	lines = append(lines, "", "Tab query | Enter expand | Ctrl+E refresh")
	return strings.Join(clipLines(lines, explorerPanelWidth), "\n")
}

func (m sqlWorkspaceModel) loadExplorer() tea.Cmd {
	if m.explorer == nil {
		return func() tea.Msg {
			return explorerLoadedMsg{err: fmt.Errorf("database explorer is not configured")}
		}
	}

	explorer := m.explorer
	database := m.database
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		snapshot, err := explorer(ctx, database)
		return explorerLoadedMsg{snapshot: snapshot, err: err}
	}
}

func (m *sqlWorkspaceModel) toggleActivePane() {
	if m.activePane == paneExplorer {
		m.activePane = paneEditor
		m.editor.Focus()
		return
	}

	m.activePane = paneExplorer
	m.editor.Blur()
}

func (m sqlWorkspaceModel) updateExplorer(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "up", "k":
		m.tree.moveUp()
	case "down", "j":
		m.tree.moveDown()
	case "enter", " ":
		m.tree.toggle()
	case "right", "l":
		m.tree.expand()
	case "left", "h":
		m.tree.collapseOrParent()
	}

	return m, nil
}

func (m *sqlWorkspaceModel) resize(width int, height int) {
	if width > 0 {
		m.width = width
	}
	if height > 0 {
		m.height = height
	}
	editorWidth := m.width - explorerPanelWidth - panelGutterWidth
	if editorWidth < minEditorWidth {
		editorWidth = minEditorWidth
	}
	m.editor.SetWidth(editorWidth)
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

func explorerStatus(snapshot odbcclient.ExplorerSnapshot) string {
	if len(snapshot.Warnings) > 0 {
		return fmt.Sprintf("Loaded with %d warning(s).", len(snapshot.Warnings))
	}

	return fmt.Sprintf("Loaded %d database(s).", len(snapshot.Databases))
}

func renderTreeRows(tree explorerTree, maxRows int) []string {
	rows := tree.visibleRows()
	if len(rows) == 0 {
		return []string{"No metadata loaded."}
	}
	start := 0
	if tree.cursor >= maxRows {
		start = tree.cursor - maxRows + 1
	}
	end := start + maxRows
	if end > len(rows) {
		end = len(rows)
	}

	rendered := make([]string, 0, end-start)
	for index := start; index < end; index++ {
		row := rows[index]
		cursor := " "
		if index == tree.cursor {
			cursor = ">"
		}
		prefix := "   "
		if row.expandable {
			if row.expanded {
				prefix = "[-]"
			} else {
				prefix = "[+]"
			}
		}
		rendered = append(rendered, fmt.Sprintf("%s %s%s%s", cursor, strings.Repeat("  ", row.depth), prefix, row.label))
	}
	return rendered
}

func clipLines(lines []string, width int) []string {
	clipped := make([]string, len(lines))
	for index, line := range lines {
		clipped[index] = clipLine(line, width)
	}
	return clipped
}

func joinPanels(left string, right string, leftWidth int, gutter int) string {
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")
	lineCount := len(leftLines)
	if len(rightLines) > lineCount {
		lineCount = len(rightLines)
	}

	var builder strings.Builder
	for index := 0; index < lineCount; index++ {
		leftLine := ""
		if index < len(leftLines) {
			leftLine = leftLines[index]
		}
		rightLine := ""
		if index < len(rightLines) {
			rightLine = rightLines[index]
		}
		builder.WriteString(padRight(clipLine(leftLine, leftWidth), leftWidth))
		builder.WriteString(strings.Repeat(" ", gutter))
		builder.WriteString(rightLine)
		if index < lineCount-1 {
			builder.WriteByte('\n')
		}
	}

	return builder.String()
}

func clipLine(line string, width int) string {
	if width <= 0 || len(line) <= width {
		return line
	}
	if width <= 3 {
		return line[:width]
	}
	return line[:width-3] + "..."
}

func padRight(line string, width int) string {
	if len(line) >= width {
		return line
	}
	return line + strings.Repeat(" ", width-len(line))
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
