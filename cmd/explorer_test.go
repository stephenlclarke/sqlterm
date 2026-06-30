package main

import (
	"strings"
	"testing"

	odbcclient "github.com/jpxcz/sqlterm/odbc_client"
)

func TestExplorerTreeBuildsDatabaseObjectHierarchy(t *testing.T) {
	tree := newExplorerTree(workspaceExplorerSnapshot())
	rows := tree.visibleRows()

	for _, want := range []string{"Databases (1)", "Trading", "Tables (1)", "Views (1)", "Indexes (1)", "Functions (1)", "Data Types (2)"} {
		if !containsExplorerRow(rows, want) {
			t.Fatalf("expected row %q in %#v", want, rows)
		}
	}
}

func TestExplorerTreeNavigationAndExpansion(t *testing.T) {
	tree := newExplorerTree(workspaceExplorerSnapshot())

	tree.moveDown()
	tree.moveDown()
	if rows := tree.visibleRows(); rows[tree.cursor].label != "Tables (1)" {
		t.Fatalf("expected cursor on tables, got %#v", rows[tree.cursor])
	}

	tree.toggle()
	if !containsExplorerRow(tree.visibleRows(), "orders") {
		t.Fatalf("expected expanded table folder rows: %#v", tree.visibleRows())
	}

	tree.moveDown()
	tree.toggle()
	if !containsExplorerRow(tree.visibleRows(), "Columns (2)") {
		t.Fatalf("expected expanded table rows: %#v", tree.visibleRows())
	}

	tree.collapseOrParent()
	if rows := tree.visibleRows(); rows[tree.cursor].label != "orders" {
		t.Fatalf("expected collapse to keep cursor on table, got %#v", rows[tree.cursor])
	}
	tree.collapseOrParent()
	if rows := tree.visibleRows(); rows[tree.cursor].label != "Tables (1)" {
		t.Fatalf("expected second collapse to move to parent, got %#v", rows[tree.cursor])
	}
}

func TestRenderTreeRowsMarksCursorAndExpansionState(t *testing.T) {
	tree := newExplorerTree(workspaceExplorerSnapshot())
	tree.moveDown()
	lines := renderTreeRows(tree, 4)
	joined := strings.Join(lines, "\n")

	for _, want := range []string{">   [-]Trading", "  [+]Tables (1)"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected %q in rendered rows:\n%s", want, joined)
		}
	}
}

func TestExplorerTreeMoveUpExpandAndClampEdges(t *testing.T) {
	tree := newExplorerTree(workspaceExplorerSnapshot())

	tree.moveUp()
	if tree.cursor != 0 {
		t.Fatalf("expected move up at top to stay on first row, got %d", tree.cursor)
	}

	tree.moveDown()
	tree.moveDown()
	tree.expand()
	if !containsExplorerRow(tree.visibleRows(), "orders") {
		t.Fatalf("expected expand to show table rows: %#v", tree.visibleRows())
	}

	tree.cursor = 999
	tree.clampCursor()
	if rows := tree.visibleRows(); tree.cursor != len(rows)-1 {
		t.Fatalf("expected cursor to clamp to last row, got %d for %d rows", tree.cursor, len(rows))
	}

	var empty explorerTree
	empty.clampCursor()
	if empty.cursor != 0 {
		t.Fatalf("expected empty tree cursor to reset, got %d", empty.cursor)
	}
	if node := tree.nodeAt([]int{999}); node != nil {
		t.Fatalf("expected invalid path to return nil, got %#v", node)
	}
	if samePath([]int{1}, []int{1, 2}) {
		t.Fatal("expected paths with different lengths to differ")
	}
}

func containsExplorerRow(rows []explorerRow, label string) bool {
	for _, row := range rows {
		if row.label == label {
			return true
		}
	}
	return false
}

func workspaceExplorerSnapshot() odbcclient.ExplorerSnapshot {
	return odbcclient.ExplorerSnapshot{
		Databases: []odbcclient.ExplorerDatabase{{
			Name: "Trading",
			Tables: []odbcclient.ExplorerTable{{
				Schema: "Trading",
				Name:   "orders",
				Columns: []odbcclient.ExplorerColumn{
					{Name: "id", DataType: "int", Nullable: "NO"},
					{Name: "customer_name", DataType: "varchar", Nullable: "YES"},
				},
				Indexes: []odbcclient.ExplorerIndex{{
					Table:   "orders",
					Name:    "PRIMARY",
					Columns: []string{"id"},
					Unique:  true,
				}},
			}},
			Views: []odbcclient.ExplorerView{{
				Schema: "Trading",
				Name:   "open_orders",
				Columns: []odbcclient.ExplorerColumn{
					{Name: "id", DataType: "int", Nullable: "NO"},
				},
			}},
			Indexes: []odbcclient.ExplorerIndex{{
				Table:   "orders",
				Name:    "PRIMARY",
				Columns: []string{"id"},
				Unique:  true,
			}},
			Functions: []odbcclient.ExplorerRoutine{{
				Schema:     "Trading",
				Name:       "calc_total",
				Type:       "FUNCTION",
				ReturnType: "decimal",
			}},
			DataTypes: []odbcclient.ExplorerDataType{{Name: "decimal"}, {Name: "int"}},
		}},
	}
}
