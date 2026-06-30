package databases

import (
	"strings"
	"testing"
)

func TestIndexDatabasesRequiresKey(t *testing.T) {
	_, _, err := IndexDatabases([]DatabaseCredentials{{
		Username: "user",
		Hostname: "localhost",
	}})
	if err == nil || !strings.Contains(err.Error(), "missing required key") {
		t.Fatalf("expected missing key error, got %v", err)
	}
}

func TestIndexDatabasesRejectsEmptyConfig(t *testing.T) {
	_, _, err := IndexDatabases(nil)
	if err == nil || !strings.Contains(err.Error(), "contained no data") {
		t.Fatalf("expected empty config error, got %v", err)
	}
}

func TestIndexDatabasesRequiresUsername(t *testing.T) {
	_, _, err := IndexDatabases([]DatabaseCredentials{{
		Key:      "dev",
		Hostname: "localhost",
	}})
	if err == nil || !strings.Contains(err.Error(), "missing required username") {
		t.Fatalf("expected missing username error, got %v", err)
	}
}

func TestIndexDatabasesRequiresODBCEndpoint(t *testing.T) {
	_, _, err := IndexDatabases([]DatabaseCredentials{{
		Key:      "dev",
		Username: "user",
	}})
	if err == nil || !strings.Contains(err.Error(), "missing required ODBC endpoint") {
		t.Fatalf("expected missing endpoint error, got %v", err)
	}
}

func TestIndexDatabasesRejectsNewlineInCredentials(t *testing.T) {
	_, _, err := IndexDatabases([]DatabaseCredentials{{
		Key:      "dev",
		Username: "user",
		Hostname: "local\nhost",
	}})
	if err == nil || !strings.Contains(err.Error(), "unsupported newline") {
		t.Fatalf("expected newline validation error, got %v", err)
	}
}

func TestIndexDatabasesRejectsDuplicateKey(t *testing.T) {
	db := DatabaseCredentials{Key: "dev", Username: "user", Hostname: "localhost"}
	_, _, err := IndexDatabases([]DatabaseCredentials{db, db})
	if err == nil || !strings.Contains(err.Error(), "duplicated") {
		t.Fatalf("expected duplicate key error, got %v", err)
	}
}

func TestIndexDatabasesRejectsInvalidPort(t *testing.T) {
	_, _, err := IndexDatabases([]DatabaseCredentials{{
		Key:      "dev",
		Username: "user",
		Hostname: "localhost",
		Port:     "not-a-port",
	}})
	if err == nil || !strings.Contains(err.Error(), "invalid port") {
		t.Fatalf("expected invalid port error, got %v", err)
	}
}

func TestIndexDatabasesRejectsOutOfRangePort(t *testing.T) {
	_, _, err := IndexDatabases([]DatabaseCredentials{{
		Key:      "dev",
		Username: "user",
		Hostname: "localhost",
		Port:     "65536",
	}})
	if err == nil || !strings.Contains(err.Error(), "invalid port") {
		t.Fatalf("expected invalid port error, got %v", err)
	}
}

func TestIndexDatabasesReturnsMapAndKeys(t *testing.T) {
	databaseMap, databaseKeys, err := IndexDatabases([]DatabaseCredentials{{
		Key:       "dev",
		ShortName: "Development",
		Username:  "user",
		Hostname:  "localhost",
		Password:  "secret",
		Port:      "3307",
	}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := databaseMap["dev"].Port; got != "3307" {
		t.Fatalf("expected port to be indexed, got %q", got)
	}
	if len(databaseKeys) != 1 || databaseKeys[0] != "dev" {
		t.Fatalf("unexpected keys: %#v", databaseKeys)
	}
}
