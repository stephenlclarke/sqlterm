package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/jpxcz/sqlterm/databases"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, databases.LoadDatabases, runDatabaseSelector, runSQLWorkspace); err != nil {
		fmt.Fprintf(os.Stderr, "Exiting: %v\n", err)
		os.Exit(1)
	}
}

type databaseLoader func() ([]databases.DatabaseCredentials, map[string]databases.DatabaseCredentials, []string, error)
type databaseSelector func(io.Reader, io.Writer, []databases.DatabaseCredentials) (databases.DatabaseCredentials, error)
type sqlWorkspace func(io.Reader, io.Writer, databases.DatabaseCredentials, workspaceOptions) error

type workspaceOptions struct {
	FormatAsTable bool
}

func run(args []string, input io.Reader, output io.Writer, loadDatabases databaseLoader, selectDatabase databaseSelector, openWorkspace sqlWorkspace) error {
	dbs, databaseMap, databaseKeys, err := loadDatabases()
	if err != nil {
		return err
	}
	if len(dbs) == 0 {
		return fmt.Errorf("database config file [~/.config/sqlterm/databases.json] contained no data")
	}

	fs := flag.NewFlagSet("sqlterm", flag.ContinueOnError)
	fs.SetOutput(output)

	var dbEnv string
	fs.StringVar(&dbEnv, "env", dbs[0].Key, "One of the following database environment keys; ["+strings.Join(databaseKeys, ",")+"]")

	var table tableFlag
	fs.Var(&table, "table", "Format SQL tables [YES / NO]")
	if err := fs.Parse(args); err != nil {
		return err
	}

	flagset := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) { flagset[f.Name] = true })

	if flagset["env"] {
		dbCreds, ok := databaseMap[dbEnv]
		if !ok {
			return fmt.Errorf("unknown database environment [%s]; valid values are [%s]", dbEnv, strings.Join(databaseKeys, ","))
		}

		return openWorkspace(input, output, dbCreds, workspaceOptions{FormatAsTable: bool(table)})
	}

	dbCreds, err := selectDatabase(input, output, dbs)
	if err != nil {
		return err
	}

	return openWorkspace(input, output, dbCreds, workspaceOptions{FormatAsTable: bool(table)})
}

type tableFlag bool

func (f *tableFlag) Set(value string) error {
	parsed, err := strconv.ParseBool(value)
	if err == nil {
		*f = tableFlag(parsed)
		return nil
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "yes", "y":
		*f = true
	case "no", "n":
		*f = false
	default:
		return fmt.Errorf("invalid table value [%s]; use YES or NO", value)
	}

	return nil
}

func (f *tableFlag) String() string {
	if f != nil && bool(*f) {
		return "YES"
	}

	return "NO"
}

func (f *tableFlag) IsBoolFlag() bool {
	return true
}
