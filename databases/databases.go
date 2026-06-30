package databases

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type DatabaseCredentials struct {
	Key              string `json:"key"`
	ShortName        string `json:"shortname"`
	Username         string `json:"username"`
	Hostname         string `json:"hostname"`
	Password         string `json:"password"`
	Port             string `json:"port"`
	DSN              string `json:"dsn"`
	Driver           string `json:"driver"`
	Database         string `json:"database"`
	ConnectionString string `json:"connection_string"`
}

func GetDatabases() ([]DatabaseCredentials, map[string]DatabaseCredentials, []string) {
	databases, databaseMap, databaseKeys, err := LoadDatabases()
	if err != nil {
		fmt.Printf("Exiting: %v\n", err)
		os.Exit(1)
	}

	return databases, databaseMap, databaseKeys
}

func LoadDatabases() ([]DatabaseCredentials, map[string]DatabaseCredentials, []string, error) {
	databases, err := ReadDatabasesJson()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not read databases file correctly: %w", err)
	}

	databaseMap, databaseKeys, err := IndexDatabases(databases)
	if err != nil {
		return nil, nil, nil, err
	}

	return databases, databaseMap, databaseKeys, nil
}

func IndexDatabases(databases []DatabaseCredentials) (map[string]DatabaseCredentials, []string, error) {
	if len(databases) == 0 {
		return nil, nil, errors.New("database config file [~/.config/sqlterm/databases.json] contained no data")
	}

	databaseMap := make(map[string]DatabaseCredentials)
	var databaseKeys []string

	for index, db := range databases {
		if err := validateDatabase(index, db); err != nil {
			return nil, nil, err
		}
		if _, exists := databaseMap[db.Key]; exists {
			return nil, nil, fmt.Errorf("database config key [%s] is duplicated", db.Key)
		}

		databaseMap[db.Key] = db
		databaseKeys = append(databaseKeys, db.Key)
	}

	return databaseMap, databaseKeys, nil
}

func validateDatabase(index int, db DatabaseCredentials) error {
	if strings.TrimSpace(db.Key) == "" {
		return fmt.Errorf("database config entry [%d] is missing required key", index)
	}
	if strings.TrimSpace(db.Username) == "" {
		return fmt.Errorf("database config entry [%s] is missing required username", db.Key)
	}
	if strings.TrimSpace(db.Hostname) == "" {
		if strings.TrimSpace(db.DSN) == "" && strings.TrimSpace(db.Driver) == "" && strings.TrimSpace(db.ConnectionString) == "" {
			return fmt.Errorf("database config entry [%s] is missing required ODBC endpoint", db.Key)
		}
	}
	if strings.ContainsAny(db.Username+db.Hostname+db.Password+db.Port+db.DSN+db.Driver+db.Database+db.ConnectionString, "\r\n") {
		return fmt.Errorf("database config entry [%s] contains an unsupported newline in credentials", db.Key)
	}
	if db.Port != "" {
		port, err := strconv.Atoi(db.Port)
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("database config entry [%s] has invalid port [%s]", db.Key, db.Port)
		}
	}

	return nil
}
