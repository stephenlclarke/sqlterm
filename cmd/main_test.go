package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/jpxcz/sqlterm/databases"
	mysqlclient "github.com/jpxcz/sqlterm/mysql_client"
)

func TestRunReturnsLoaderError(t *testing.T) {
	want := errors.New("load failed")
	err := run(nil, strings.NewReader(""), &bytes.Buffer{}, func() ([]databases.DatabaseCredentials, map[string]databases.DatabaseCredentials, []string, error) {
		return nil, nil, nil, want
	}, func(mysqlclient.Config) error {
		t.Fatal("mysql should not be executed")
		return nil
	})
	if !errors.Is(err, want) {
		t.Fatalf("expected loader error, got %v", err)
	}
}

func TestRunRejectsUnknownFlag(t *testing.T) {
	err := run([]string{"-missing"}, strings.NewReader(""), &bytes.Buffer{}, testLoader, func(mysqlclient.Config) error {
		t.Fatal("mysql should not be executed")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Fatalf("expected flag parse error, got %v", err)
	}
}

func TestRunRejectsUnknownEnv(t *testing.T) {
	err := run([]string{"-env", "missing"}, strings.NewReader(""), &bytes.Buffer{}, testLoader, func(mysqlclient.Config) error {
		t.Fatal("mysql should not be executed")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "unknown database environment") {
		t.Fatalf("expected unknown env error, got %v", err)
	}
}

func TestRunHonorsTableNo(t *testing.T) {
	var got mysqlclient.Config
	err := run([]string{"-env", "dev", "-table=NO"}, strings.NewReader(""), &bytes.Buffer{}, testLoader, func(config mysqlclient.Config) error {
		got = config
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.FormatAsTable {
		t.Fatal("expected -table=NO to disable table formatting")
	}
}

func TestRunHonorsTableYesAndPort(t *testing.T) {
	var got mysqlclient.Config
	err := run([]string{"-env", "dev", "-table=YES"}, strings.NewReader(""), &bytes.Buffer{}, testLoader, func(config mysqlclient.Config) error {
		got = config
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.FormatAsTable {
		t.Fatal("expected -table=YES to enable table formatting")
	}
	if got.Port != "3307" {
		t.Fatalf("expected port 3307, got %q", got.Port)
	}
}

func TestRunHonorsBareTableFlag(t *testing.T) {
	var got mysqlclient.Config
	err := run([]string{"-env", "dev", "-table"}, strings.NewReader(""), &bytes.Buffer{}, testLoader, func(config mysqlclient.Config) error {
		got = config
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.FormatAsTable {
		t.Fatal("expected bare -table to enable table formatting")
	}
}

func TestRunReturnsExecError(t *testing.T) {
	want := errors.New("mysql failed")
	err := run([]string{"-env", "dev"}, strings.NewReader(""), &bytes.Buffer{}, testLoader, func(mysqlclient.Config) error {
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("expected mysql error, got %v", err)
	}
}

func TestRunRejectsInvalidTableValue(t *testing.T) {
	err := run([]string{"-env", "dev", "-table=maybe"}, strings.NewReader(""), &bytes.Buffer{}, testLoader, func(mysqlclient.Config) error {
		t.Fatal("mysql should not be executed")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "invalid table value") {
		t.Fatalf("expected invalid table error, got %v", err)
	}
}

func TestRunRejectsUnreadableSelection(t *testing.T) {
	err := run(nil, strings.NewReader("not-a-number\n"), &bytes.Buffer{}, testLoader, func(mysqlclient.Config) error {
		t.Fatal("mysql should not be executed")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "could not read database selection") {
		t.Fatalf("expected selection read error, got %v", err)
	}
}

func TestRunRejectsNegativeSelection(t *testing.T) {
	err := run(nil, strings.NewReader("-1\n"), &bytes.Buffer{}, testLoader, func(mysqlclient.Config) error {
		t.Fatal("mysql should not be executed")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "not in range") {
		t.Fatalf("expected selection range error, got %v", err)
	}
}

func TestRunRejectsOutOfRangeSelection(t *testing.T) {
	err := run(nil, strings.NewReader("1\n"), &bytes.Buffer{}, testLoader, func(mysqlclient.Config) error {
		t.Fatal("mysql should not be executed")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "not in range") {
		t.Fatalf("expected selection range error, got %v", err)
	}
}

func TestRunExecutesSelectedDatabase(t *testing.T) {
	var got mysqlclient.Config
	err := run(nil, strings.NewReader("0\n"), &bytes.Buffer{}, testLoader, func(config mysqlclient.Config) error {
		got = config
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Username != "user" || got.Hostname != "localhost" || got.Password != "secret" || got.Port != "3307" {
		t.Fatalf("unexpected config: %#v", got)
	}
}

func TestTableFlagString(t *testing.T) {
	var nilFlag *tableFlag
	if got := nilFlag.String(); got != "NO" {
		t.Fatalf("expected nil flag to stringify as NO, got %q", got)
	}

	var flag tableFlag
	if got := flag.String(); got != "NO" {
		t.Fatalf("expected false flag to stringify as NO, got %q", got)
	}
	if err := flag.Set("true"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := flag.String(); got != "YES" {
		t.Fatalf("expected true flag to stringify as YES, got %q", got)
	}
}

func testLoader() ([]databases.DatabaseCredentials, map[string]databases.DatabaseCredentials, []string, error) {
	dbs := []databases.DatabaseCredentials{{
		Key:       "dev",
		ShortName: "Development",
		Username:  "user",
		Hostname:  "localhost",
		Password:  "secret",
		Port:      "3307",
	}}
	dbMap, keys, err := databases.IndexDatabases(dbs)
	return dbs, dbMap, keys, err
}
