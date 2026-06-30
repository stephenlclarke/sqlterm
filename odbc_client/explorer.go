package odbcclient

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/jpxcz/sqlterm/databases"
)

const (
	sqlServerDatabasesQuery = "SELECT name FROM sys.databases ORDER BY name"
	schemataQuery           = "SELECT schema_name FROM information_schema.schemata ORDER BY schema_name"
	tablesQuery             = "SELECT table_catalog, table_schema, table_name, table_type FROM information_schema.tables WHERE table_schema NOT IN ('information_schema', 'pg_catalog', 'sys') ORDER BY table_catalog, table_schema, table_type, table_name"
	columnsQuery            = "SELECT table_catalog, table_schema, table_name, column_name, data_type, is_nullable, ordinal_position FROM information_schema.columns WHERE table_schema NOT IN ('information_schema', 'pg_catalog', 'sys') ORDER BY table_catalog, table_schema, table_name, ordinal_position"
	mysqlIndexesQuery       = "SELECT table_schema AS table_catalog, table_schema, table_name, index_name, column_name, CASE WHEN non_unique = 0 THEN 'YES' ELSE 'NO' END AS is_unique FROM information_schema.statistics ORDER BY table_schema, table_name, index_name, seq_in_index"
	sqlServerIndexesQuery   = "SELECT DB_NAME() AS table_catalog, SCHEMA_NAME(t.schema_id) AS table_schema, t.name AS table_name, i.name AS index_name, c.name AS column_name, CASE WHEN i.is_unique = 1 THEN 'YES' ELSE 'NO' END AS is_unique FROM sys.indexes i JOIN sys.index_columns ic ON i.object_id = ic.object_id AND i.index_id = ic.index_id JOIN sys.columns c ON ic.object_id = c.object_id AND ic.column_id = c.column_id JOIN sys.tables t ON i.object_id = t.object_id WHERE i.name IS NOT NULL ORDER BY table_catalog, table_schema, table_name, index_name, ic.key_ordinal"
	routinesQuery           = "SELECT routine_catalog, routine_schema, routine_name, routine_type, data_type FROM information_schema.routines ORDER BY routine_catalog, routine_schema, routine_type, routine_name"
)

type ExplorerSnapshot struct {
	Databases []ExplorerDatabase
	Warnings  []string
}

type ExplorerDatabase struct {
	Name      string
	Tables    []ExplorerTable
	Views     []ExplorerView
	Indexes   []ExplorerIndex
	Functions []ExplorerRoutine
	DataTypes []ExplorerDataType
}

type ExplorerTable struct {
	Schema  string
	Name    string
	Columns []ExplorerColumn
	Indexes []ExplorerIndex
}

type ExplorerView struct {
	Schema  string
	Name    string
	Columns []ExplorerColumn
}

type ExplorerColumn struct {
	Name     string
	DataType string
	Nullable string
	Ordinal  string
}

type ExplorerIndex struct {
	Schema  string
	Table   string
	Name    string
	Columns []string
	Unique  bool
}

type ExplorerRoutine struct {
	Schema     string
	Name       string
	Type       string
	ReturnType string
}

type ExplorerDataType struct {
	Name string
}

func (c Client) FetchExplorer(ctx context.Context, db databases.DatabaseCredentials) (ExplorerSnapshot, error) {
	if c.open == nil {
		c.open = sql.Open
	}

	connectionString, err := ConnectionString(db)
	if err != nil {
		return ExplorerSnapshot{}, err
	}

	connection, err := c.open(driverName, connectionString)
	if err != nil {
		return ExplorerSnapshot{}, fmt.Errorf("open ODBC connection: %w", err)
	}
	defer connection.Close()

	builder := newExplorerBuilder(fallbackDatabaseName(db))
	builder.addDatabase(fallbackDatabaseName(db))

	if rows, err := firstSuccessfulStringRows(ctx, connection, sqlServerDatabasesQuery, schemataQuery); err == nil {
		for _, row := range rows {
			if len(row) > 0 {
				builder.addDatabase(row[0])
			}
		}
	} else {
		builder.warn("databases", err)
	}

	if rows, err := queryStringRows(ctx, connection, tablesQuery); err == nil {
		builder.addTables(rows)
	} else {
		builder.warn("tables", err)
	}

	if rows, err := queryStringRows(ctx, connection, columnsQuery); err == nil {
		builder.addColumns(rows)
	} else {
		builder.warn("columns", err)
	}

	if rows, err := firstSuccessfulStringRows(ctx, connection, mysqlIndexesQuery, sqlServerIndexesQuery); err == nil {
		builder.addIndexes(rows)
	} else {
		builder.warn("indexes", err)
	}

	if rows, err := queryStringRows(ctx, connection, routinesQuery); err == nil {
		builder.addRoutines(rows)
	} else {
		builder.warn("functions", err)
	}

	return builder.snapshot(), nil
}

func firstSuccessfulStringRows(ctx context.Context, connection *sql.DB, statements ...string) ([][]string, error) {
	var lastErr error
	for _, statement := range statements {
		rows, err := queryStringRows(ctx, connection, statement)
		if err == nil {
			return rows, nil
		}
		lastErr = err
	}

	return nil, lastErr
}

func queryStringRows(ctx context.Context, connection *sql.DB, statement string) ([][]string, error) {
	rows, err := connection.QueryContext(ctx, statement)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	_, resultRows, err := readRows(rows)
	return resultRows, err
}

type explorerBuilder struct {
	fallback string
	byName   map[string]*ExplorerDatabase
	warnings []string
}

func newExplorerBuilder(fallback string) *explorerBuilder {
	return &explorerBuilder{
		fallback: fallback,
		byName:   make(map[string]*ExplorerDatabase),
	}
}

func (b *explorerBuilder) addDatabase(name string) *ExplorerDatabase {
	name = strings.TrimSpace(name)
	if name == "" {
		name = b.fallback
	}
	if database, ok := b.byName[name]; ok {
		return database
	}

	database := &ExplorerDatabase{Name: name}
	b.byName[name] = database
	return database
}

func (b *explorerBuilder) addTables(rows [][]string) {
	for _, row := range rows {
		if len(row) < 4 {
			continue
		}

		database := b.addDatabase(databaseName(row[0], row[1], b.fallback))
		table := ExplorerTable{Schema: row[1], Name: row[2]}
		if isView(row[3]) {
			if b.findView(database, row[1], row[2]) == nil {
				database.Views = append(database.Views, ExplorerView{Schema: row[1], Name: row[2]})
			}
			continue
		}
		if b.findTable(database, row[1], row[2]) == nil {
			database.Tables = append(database.Tables, table)
		}
	}
}

func (b *explorerBuilder) addColumns(rows [][]string) {
	for _, row := range rows {
		if len(row) < 7 {
			continue
		}

		database := b.addDatabase(databaseName(row[0], row[1], b.fallback))
		column := ExplorerColumn{
			Name:     row[3],
			DataType: row[4],
			Nullable: row[5],
			Ordinal:  row[6],
		}
		b.addDataType(database, column.DataType)

		if view := b.findView(database, row[1], row[2]); view != nil {
			view.Columns = append(view.Columns, column)
			continue
		}
		table := b.ensureTable(database, row[1], row[2])
		table.Columns = append(table.Columns, column)
	}
}

func (b *explorerBuilder) addIndexes(rows [][]string) {
	for _, row := range rows {
		if len(row) < 6 {
			continue
		}

		database := b.addDatabase(databaseName(row[0], row[1], b.fallback))
		index := b.ensureIndex(database, row[1], row[2], row[3], isTruthy(row[5]))
		if row[4] != "" {
			index.Columns = append(index.Columns, row[4])
		}
		table := b.ensureTable(database, row[1], row[2])
		tableIndex := b.ensureTableIndex(table, row[3], isTruthy(row[5]))
		if row[4] != "" {
			tableIndex.Columns = append(tableIndex.Columns, row[4])
		}
	}
}

func (b *explorerBuilder) addRoutines(rows [][]string) {
	for _, row := range rows {
		if len(row) < 5 {
			continue
		}

		database := b.addDatabase(databaseName(row[0], row[1], b.fallback))
		database.Functions = append(database.Functions, ExplorerRoutine{
			Schema:     row[1],
			Name:       row[2],
			Type:       row[3],
			ReturnType: row[4],
		})
		b.addDataType(database, row[4])
	}
}

func (b *explorerBuilder) ensureTable(database *ExplorerDatabase, schema string, name string) *ExplorerTable {
	if table := b.findTable(database, schema, name); table != nil {
		return table
	}
	database.Tables = append(database.Tables, ExplorerTable{Schema: schema, Name: name})
	return &database.Tables[len(database.Tables)-1]
}

func (b *explorerBuilder) findTable(database *ExplorerDatabase, schema string, name string) *ExplorerTable {
	for index := range database.Tables {
		if database.Tables[index].Schema == schema && database.Tables[index].Name == name {
			return &database.Tables[index]
		}
	}
	return nil
}

func (b *explorerBuilder) findView(database *ExplorerDatabase, schema string, name string) *ExplorerView {
	for index := range database.Views {
		if database.Views[index].Schema == schema && database.Views[index].Name == name {
			return &database.Views[index]
		}
	}
	return nil
}

func (b *explorerBuilder) ensureIndex(database *ExplorerDatabase, schema string, table string, name string, unique bool) *ExplorerIndex {
	for index := range database.Indexes {
		if database.Indexes[index].Schema == schema && database.Indexes[index].Table == table && database.Indexes[index].Name == name {
			return &database.Indexes[index]
		}
	}
	database.Indexes = append(database.Indexes, ExplorerIndex{
		Schema: schema,
		Table:  table,
		Name:   name,
		Unique: unique,
	})
	return &database.Indexes[len(database.Indexes)-1]
}

func (b *explorerBuilder) ensureTableIndex(table *ExplorerTable, name string, unique bool) *ExplorerIndex {
	for index := range table.Indexes {
		if table.Indexes[index].Name == name {
			return &table.Indexes[index]
		}
	}
	table.Indexes = append(table.Indexes, ExplorerIndex{
		Schema: table.Schema,
		Table:  table.Name,
		Name:   name,
		Unique: unique,
	})
	return &table.Indexes[len(table.Indexes)-1]
}

func (b *explorerBuilder) addDataType(database *ExplorerDatabase, name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	for _, dataType := range database.DataTypes {
		if strings.EqualFold(dataType.Name, name) {
			return
		}
	}
	database.DataTypes = append(database.DataTypes, ExplorerDataType{Name: name})
}

func (b *explorerBuilder) warn(category string, err error) {
	if err != nil {
		b.warnings = append(b.warnings, fmt.Sprintf("%s unavailable: %v", category, err))
	}
}

func (b *explorerBuilder) snapshot() ExplorerSnapshot {
	databases := make([]ExplorerDatabase, 0, len(b.byName))
	for _, database := range b.byName {
		sortDatabase(database)
		databases = append(databases, *database)
	}
	sort.Slice(databases, func(i, j int) bool {
		return strings.ToLower(databases[i].Name) < strings.ToLower(databases[j].Name)
	})

	return ExplorerSnapshot{
		Databases: databases,
		Warnings:  append([]string(nil), b.warnings...),
	}
}

func sortDatabase(database *ExplorerDatabase) {
	sort.Slice(database.Tables, func(i, j int) bool {
		return qualifiedName(database.Tables[i].Schema, database.Tables[i].Name) < qualifiedName(database.Tables[j].Schema, database.Tables[j].Name)
	})
	sort.Slice(database.Views, func(i, j int) bool {
		return qualifiedName(database.Views[i].Schema, database.Views[i].Name) < qualifiedName(database.Views[j].Schema, database.Views[j].Name)
	})
	sort.Slice(database.Indexes, func(i, j int) bool {
		return qualifiedName(database.Indexes[i].Table, database.Indexes[i].Name) < qualifiedName(database.Indexes[j].Table, database.Indexes[j].Name)
	})
	sort.Slice(database.Functions, func(i, j int) bool {
		return qualifiedName(database.Functions[i].Schema, database.Functions[i].Name) < qualifiedName(database.Functions[j].Schema, database.Functions[j].Name)
	})
	sort.Slice(database.DataTypes, func(i, j int) bool {
		return strings.ToLower(database.DataTypes[i].Name) < strings.ToLower(database.DataTypes[j].Name)
	})
}

func databaseName(catalog string, schema string, fallback string) string {
	catalog = strings.TrimSpace(catalog)
	schema = strings.TrimSpace(schema)
	if catalog != "" && !strings.EqualFold(catalog, "def") {
		return catalog
	}
	if schema != "" {
		return schema
	}
	return fallback
}

func fallbackDatabaseName(db databases.DatabaseCredentials) string {
	for _, value := range []string{db.Database, db.DSN, db.Hostname, db.Key} {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return "connected"
}

func isView(tableType string) bool {
	return strings.Contains(strings.ToUpper(tableType), "VIEW")
}

func isTruthy(value string) bool {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "1", "T", "TRUE", "Y", "YES":
		return true
	default:
		return false
	}
}

func qualifiedName(namespace string, name string) string {
	if namespace == "" {
		return strings.ToLower(name)
	}
	return strings.ToLower(namespace + "." + name)
}
