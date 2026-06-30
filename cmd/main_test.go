package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/jpxcz/sqlterm/databases"
)

func TestRunReturnsLoaderError(t *testing.T) {
	want := errors.New("load failed")
	err := run(nil, strings.NewReader(""), &bytes.Buffer{}, func() ([]databases.DatabaseCredentials, map[string]databases.DatabaseCredentials, []string, error) {
		return nil, nil, nil, want
	}, testSelector(0), testWorkspace(nil, nil))
	if !errors.Is(err, want) {
		t.Fatalf("expected loader error, got %v", err)
	}
}

func TestRunRejectsEmptyDatabases(t *testing.T) {
	err := run(nil, strings.NewReader(""), &bytes.Buffer{}, func() ([]databases.DatabaseCredentials, map[string]databases.DatabaseCredentials, []string, error) {
		return nil, nil, nil, nil
	}, testSelector(0), testWorkspace(nil, nil))
	if err == nil || !strings.Contains(err.Error(), "contained no data") {
		t.Fatalf("expected empty database error, got %v", err)
	}
}

func TestRunRejectsUnknownFlag(t *testing.T) {
	err := run([]string{"-missing"}, strings.NewReader(""), &bytes.Buffer{}, testLoader, testSelector(0), testWorkspace(nil, nil))
	if err == nil || !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Fatalf("expected flag parse error, got %v", err)
	}
}

func TestRunRejectsUnknownEnv(t *testing.T) {
	err := run([]string{"-env", "missing"}, strings.NewReader(""), &bytes.Buffer{}, testLoader, testSelector(0), testWorkspace(nil, nil))
	if err == nil || !strings.Contains(err.Error(), "unknown database environment") {
		t.Fatalf("expected unknown env error, got %v", err)
	}
}

func TestRunHonorsTableNo(t *testing.T) {
	var got workspaceCapture
	err := run([]string{"-env", "dev", "-table=NO"}, strings.NewReader(""), &bytes.Buffer{}, testLoader, testSelector(0), testWorkspace(&got, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.options.FormatAsTable {
		t.Fatal("expected -table=NO to disable table formatting")
	}
}

func TestRunHonorsTableYesAndPort(t *testing.T) {
	var got workspaceCapture
	err := run([]string{"-env", "dev", "-table=YES"}, strings.NewReader(""), &bytes.Buffer{}, testLoader, testSelector(0), testWorkspace(&got, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.options.FormatAsTable {
		t.Fatal("expected -table=YES to enable table formatting")
	}
	if got.database.Port != "3307" {
		t.Fatalf("expected port 3307, got %q", got.database.Port)
	}
}

func TestRunHonorsBareTableFlag(t *testing.T) {
	var got workspaceCapture
	err := run([]string{"-env", "dev", "-table"}, strings.NewReader(""), &bytes.Buffer{}, testLoader, testSelector(0), testWorkspace(&got, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.options.FormatAsTable {
		t.Fatal("expected bare -table to enable table formatting")
	}
}

func TestRunReturnsWorkspaceError(t *testing.T) {
	want := errors.New("workspace failed")
	err := run([]string{"-env", "dev"}, strings.NewReader(""), &bytes.Buffer{}, testLoader, testSelector(0), testWorkspace(nil, want))
	if !errors.Is(err, want) {
		t.Fatalf("expected workspace error, got %v", err)
	}
}

func TestRunRejectsInvalidTableValue(t *testing.T) {
	err := run([]string{"-env", "dev", "-table=maybe"}, strings.NewReader(""), &bytes.Buffer{}, testLoader, testSelector(0), testWorkspace(nil, nil))
	if err == nil || !strings.Contains(err.Error(), "invalid table value") {
		t.Fatalf("expected invalid table error, got %v", err)
	}
}

func TestRunReturnsSelectorError(t *testing.T) {
	want := errors.New("selector failed")
	err := run(nil, strings.NewReader(""), &bytes.Buffer{}, testLoader, func(io.Reader, io.Writer, []databases.DatabaseCredentials) (databases.DatabaseCredentials, error) {
		return databases.DatabaseCredentials{}, want
	}, testWorkspace(nil, nil))
	if !errors.Is(err, want) {
		t.Fatalf("expected selector error, got %v", err)
	}
}

func TestRunExecutesSelectedDatabase(t *testing.T) {
	var got workspaceCapture
	err := run(nil, strings.NewReader("0\n"), &bytes.Buffer{}, testLoader, testSelector(0), testWorkspace(&got, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.database.Username != "user" || got.database.Hostname != "localhost" || got.database.Password != "secret" || got.database.Port != "3307" {
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

func testSelector(index int) databaseSelector {
	return func(_ io.Reader, _ io.Writer, dbs []databases.DatabaseCredentials) (databases.DatabaseCredentials, error) {
		if index < 0 || index >= len(dbs) {
			return databases.DatabaseCredentials{}, fmt.Errorf("test selector index [%d] is out of range", index)
		}

		return dbs[index], nil
	}
}

type workspaceCapture struct {
	database databases.DatabaseCredentials
	options  workspaceOptions
}

func testWorkspace(capture *workspaceCapture, err error) sqlWorkspace {
	return func(_ io.Reader, _ io.Writer, db databases.DatabaseCredentials, options workspaceOptions) error {
		if capture != nil {
			capture.database = db
			capture.options = options
		}

		return err
	}
}
