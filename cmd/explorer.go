package main

import (
	"fmt"
	"strings"

	odbcclient "github.com/jpxcz/sqlterm/odbc_client"
)

type explorerTree struct {
	roots  []explorerNode
	cursor int
}

type explorerNode struct {
	label    string
	children []explorerNode
	expanded bool
}

type explorerRow struct {
	path       []int
	depth      int
	label      string
	expandable bool
	expanded   bool
}

func newExplorerTree(snapshot odbcclient.ExplorerSnapshot) explorerTree {
	root := explorerNode{
		label:    fmt.Sprintf("Databases (%d)", len(snapshot.Databases)),
		expanded: true,
	}
	for _, database := range snapshot.Databases {
		databaseNode := explorerNode{
			label:    database.Name,
			expanded: len(snapshot.Databases) == 1,
			children: []explorerNode{
				tableNodes(database),
				viewNodes(database),
				indexNodes(database),
				functionNodes(database),
				dataTypeNodes(database),
			},
		}
		root.children = append(root.children, databaseNode)
	}

	return explorerTree{roots: []explorerNode{root}}
}

func emptyExplorerTree() explorerTree {
	return explorerTree{roots: []explorerNode{{
		label:    "Databases (0)",
		expanded: true,
	}}}
}

func tableNodes(database odbcclient.ExplorerDatabase) explorerNode {
	folder := explorerNode{label: fmt.Sprintf("Tables (%d)", len(database.Tables))}
	for _, table := range database.Tables {
		tableNode := explorerNode{
			label: qualifiedExplorerName(database.Name, table.Schema, table.Name),
			children: []explorerNode{
				columnNodes(table.Columns),
				tableIndexNodes(table.Indexes),
			},
		}
		folder.children = append(folder.children, tableNode)
	}
	return folder
}

func viewNodes(database odbcclient.ExplorerDatabase) explorerNode {
	folder := explorerNode{label: fmt.Sprintf("Views (%d)", len(database.Views))}
	for _, view := range database.Views {
		viewNode := explorerNode{
			label: qualifiedExplorerName(database.Name, view.Schema, view.Name),
			children: []explorerNode{
				columnNodes(view.Columns),
			},
		}
		folder.children = append(folder.children, viewNode)
	}
	return folder
}

func indexNodes(database odbcclient.ExplorerDatabase) explorerNode {
	folder := explorerNode{label: fmt.Sprintf("Indexes (%d)", len(database.Indexes))}
	for _, index := range database.Indexes {
		folder.children = append(folder.children, explorerNode{label: indexLabel(index)})
	}
	return folder
}

func functionNodes(database odbcclient.ExplorerDatabase) explorerNode {
	folder := explorerNode{label: fmt.Sprintf("Functions (%d)", len(database.Functions))}
	for _, routine := range database.Functions {
		label := qualifiedExplorerName(database.Name, routine.Schema, routine.Name)
		if routine.Type != "" {
			label += " (" + routine.Type + ")"
		}
		if routine.ReturnType != "" {
			label += " -> " + routine.ReturnType
		}
		folder.children = append(folder.children, explorerNode{label: label})
	}
	return folder
}

func dataTypeNodes(database odbcclient.ExplorerDatabase) explorerNode {
	folder := explorerNode{label: fmt.Sprintf("Data Types (%d)", len(database.DataTypes))}
	for _, dataType := range database.DataTypes {
		folder.children = append(folder.children, explorerNode{label: dataType.Name})
	}
	return folder
}

func columnNodes(columns []odbcclient.ExplorerColumn) explorerNode {
	folder := explorerNode{label: fmt.Sprintf("Columns (%d)", len(columns))}
	for _, column := range columns {
		label := column.Name
		if column.DataType != "" {
			label += " " + column.DataType
		}
		if strings.EqualFold(column.Nullable, "NO") {
			label += " NOT NULL"
		}
		folder.children = append(folder.children, explorerNode{label: label})
	}
	return folder
}

func tableIndexNodes(indexes []odbcclient.ExplorerIndex) explorerNode {
	folder := explorerNode{label: fmt.Sprintf("Indexes (%d)", len(indexes))}
	for _, index := range indexes {
		folder.children = append(folder.children, explorerNode{label: indexLabel(index)})
	}
	return folder
}

func indexLabel(index odbcclient.ExplorerIndex) string {
	label := index.Name
	if index.Table != "" {
		label = index.Table + "." + label
	}
	if index.Unique {
		label += " UNIQUE"
	}
	if len(index.Columns) > 0 {
		label += " [" + strings.Join(index.Columns, ", ") + "]"
	}
	return label
}

func qualifiedExplorerName(database string, schema string, name string) string {
	if schema == "" || strings.EqualFold(schema, database) {
		return name
	}
	return schema + "." + name
}

func (t explorerTree) visibleRows() []explorerRow {
	var rows []explorerRow
	for index := range t.roots {
		rows = appendVisibleRows(rows, t.roots[index], []int{index}, 0)
	}
	return rows
}

func appendVisibleRows(rows []explorerRow, node explorerNode, path []int, depth int) []explorerRow {
	rows = append(rows, explorerRow{
		path:       append([]int(nil), path...),
		depth:      depth,
		label:      node.label,
		expandable: len(node.children) > 0,
		expanded:   node.expanded,
	})
	if !node.expanded {
		return rows
	}
	for index := range node.children {
		childPath := append(append([]int(nil), path...), index)
		rows = appendVisibleRows(rows, node.children[index], childPath, depth+1)
	}
	return rows
}

func (t *explorerTree) moveUp() {
	if t.cursor > 0 {
		t.cursor--
	}
}

func (t *explorerTree) moveDown() {
	rows := t.visibleRows()
	if t.cursor < len(rows)-1 {
		t.cursor++
	}
}

func (t *explorerTree) toggle() {
	rows := t.visibleRows()
	if t.cursor < 0 || t.cursor >= len(rows) {
		return
	}
	node := t.nodeAt(rows[t.cursor].path)
	if node == nil || len(node.children) == 0 {
		return
	}
	node.expanded = !node.expanded
	t.clampCursor()
}

func (t *explorerTree) expand() {
	rows := t.visibleRows()
	if t.cursor < 0 || t.cursor >= len(rows) {
		return
	}
	node := t.nodeAt(rows[t.cursor].path)
	if node != nil && len(node.children) > 0 {
		node.expanded = true
	}
}

func (t *explorerTree) collapseOrParent() {
	rows := t.visibleRows()
	if t.cursor < 0 || t.cursor >= len(rows) {
		return
	}
	row := rows[t.cursor]
	node := t.nodeAt(row.path)
	if node != nil && node.expanded && len(node.children) > 0 {
		node.expanded = false
		t.clampCursor()
		return
	}
	if len(row.path) <= 1 {
		return
	}
	parentPath := row.path[:len(row.path)-1]
	for index, candidate := range rows {
		if samePath(candidate.path, parentPath) {
			t.cursor = index
			return
		}
	}
}

func (t *explorerTree) nodeAt(path []int) *explorerNode {
	if len(path) == 0 || path[0] < 0 || path[0] >= len(t.roots) {
		return nil
	}
	node := &t.roots[path[0]]
	for _, index := range path[1:] {
		if index < 0 || index >= len(node.children) {
			return nil
		}
		node = &node.children[index]
	}
	return node
}

func (t *explorerTree) clampCursor() {
	rows := t.visibleRows()
	if len(rows) == 0 {
		t.cursor = 0
		return
	}
	if t.cursor >= len(rows) {
		t.cursor = len(rows) - 1
	}
}

func samePath(left []int, right []int) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
