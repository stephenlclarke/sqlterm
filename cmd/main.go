package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/jpxcz/sqlterm/databases"
	mysqlclient "github.com/jpxcz/sqlterm/mysql_client"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, databases.LoadDatabases, mysqlclient.ExecMySqlClient); err != nil {
		fmt.Fprintf(os.Stderr, "Exiting: %v\n", err)
		os.Exit(1)
	}
}

type databaseLoader func() ([]databases.DatabaseCredentials, map[string]databases.DatabaseCredentials, []string, error)
type mysqlExecutor func(mysqlclient.Config) error

func run(args []string, input io.Reader, output io.Writer, loadDatabases databaseLoader, execMySQL mysqlExecutor) error {
	dbs, databaseMap, databaseKeys, err := loadDatabases()
	if err != nil {
		return err
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

		return execMySQL(configFromCredentials(dbCreds, bool(table)))
	}

	fmt.Fprintln(output, "Welcome, please select one of the databases to connect")
	for i, db := range dbs {
		fmt.Fprintf(output, "[%d] %s - %s\n", i, db.Key, db.ShortName)
	}

	var selection int
	if _, err := fmt.Fscan(input, &selection); err != nil {
		return fmt.Errorf("could not read database selection: %w", err)
	}
	if selection < 0 || selection >= len(dbs) {
		return fmt.Errorf("option [%d] selected is not in range of the databases", selection)
	}

	return execMySQL(configFromCredentials(dbs[selection], bool(table)))
}

func configFromCredentials(db databases.DatabaseCredentials, formatAsTable bool) mysqlclient.Config {
	return mysqlclient.Config{
		Username:      db.Username,
		Hostname:      db.Hostname,
		Password:      db.Password,
		Port:          db.Port,
		FormatAsTable: formatAsTable,
	}
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
