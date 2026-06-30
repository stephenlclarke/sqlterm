package databases

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadDatabasesJsonFromPathReturnsMalformedJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "databases.json")
	if err := os.WriteFile(path, []byte("{not-json"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := ReadDatabasesJsonFromPath(path)
	if err == nil || !strings.Contains(err.Error(), "invalid character") {
		t.Fatalf("expected JSON parse error, got %v", err)
	}
}

func TestReadDatabasesJsonFromPathReadsDatabases(t *testing.T) {
	path := filepath.Join(t.TempDir(), "databases.json")
	config := []byte(`{"databases":[{"key":"dev","shortname":"Development","username":"user","hostname":"localhost","password":"secret","port":"3307"}]}`)
	if err := os.WriteFile(path, config, 0o600); err != nil {
		t.Fatal(err)
	}

	databases, err := ReadDatabasesJsonFromPath(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(databases) != 1 || databases[0].Key != "dev" {
		t.Fatalf("unexpected databases: %#v", databases)
	}
}

func TestDatabasesFilePathUsesDocumentedConfigLocation(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	path, err := DatabasesFilePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(homeDir, ".config", "sqlterm", "databases.json")
	if path != want {
		t.Fatalf("expected %q, got %q", want, path)
	}
}

func TestLoadDatabasesUsesDefaultPath(t *testing.T) {
	homeDir := t.TempDir()
	configDir := filepath.Join(homeDir, ".config", "sqlterm")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", homeDir)

	config := []byte(`{"databases":[{"key":"dev","shortname":"Development","username":"user","hostname":"localhost","password":"secret","port":"3307"}]}`)
	if err := os.WriteFile(filepath.Join(configDir, "databases.json"), config, 0o600); err != nil {
		t.Fatal(err)
	}

	databases, databaseMap, databaseKeys, err := LoadDatabases()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(databases) != 1 || len(databaseMap) != 1 || len(databaseKeys) != 1 {
		t.Fatalf("unexpected database index: %#v %#v %#v", databases, databaseMap, databaseKeys)
	}
}
