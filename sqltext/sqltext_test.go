package sqltext

import (
	"strings"
	"testing"
)

func TestPrepareUppercasesKeywordsAndPreservesData(t *testing.T) {
	statement, err := Prepare("select name from customers where note = 'select from here' and code = \"where\"")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "SELECT name FROM customers WHERE note = 'select from here' AND code = \"where\""
	if statement != want {
		t.Fatalf("expected %q, got %q", want, statement)
	}
}

func TestPrepareReturnsNormalisedInvalidSQL(t *testing.T) {
	statement, err := Prepare("select from")
	if err == nil {
		t.Fatal("expected validation error")
	}
	if statement != "SELECT FROM" {
		t.Fatalf("expected invalid SQL to still be normalised, got %q", statement)
	}
}

func TestUppercaseKeywordsPreservesCommentsAndQuotedIdentifiers(t *testing.T) {
	statement := UppercaseKeywords("select [from], `where` from t -- select\nwhere id = 1 /* from */")
	want := "SELECT [from], `where` FROM t -- select\nWHERE id = 1 /* from */"
	if statement != want {
		t.Fatalf("expected %q, got %q", want, statement)
	}
}

func TestUppercaseKeywordsHandlesEscapedAndUnterminatedQuotedText(t *testing.T) {
	cases := map[string]string{
		"select 'it''s a select' from t": "SELECT 'it''s a select' FROM t",
		"select \"unterminated from":     "SELECT \"unterminated from",
		"select [unterminated from":      "SELECT [unterminated from",
	}

	for input, want := range cases {
		if got := UppercaseKeywords(input); got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	}
}

func TestUppercaseKeywordsHandlesCommentsAndSymbols(t *testing.T) {
	cases := map[string]string{
		"select 1-- from":           "SELECT 1-- from",
		"select 1/* from":           "SELECT 1/* from",
		"select 10/2-1 from values": "SELECT 10/2-1 FROM VALUES",
	}

	for input, want := range cases {
		if got := UppercaseKeywords(input); got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	}
}

func TestValidateRejectsInvalidSQL(t *testing.T) {
	if err := Validate("select from"); err == nil {
		t.Fatal("expected syntax error")
	}
}

func TestValidateRejectsEmptySQL(t *testing.T) {
	if err := Validate(" \n "); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected empty statement error, got %v", err)
	}
}

func TestUppercaseKeywordsPreservesEscapedData(t *testing.T) {
	cases := map[string]string{
		"select e\u0301select from t":            "SELECT e\u0301select FROM t",
		`select 'it\'s select from here' from t`: `SELECT 'it\'s select from here' FROM t`,
		`select "a\" select from here" from t`:   `SELECT "a\" select from here" FROM t`,
		`select 'a\\' from t`:                    `SELECT 'a\\' FROM t`,
		"select [a]]select] from t":              "SELECT [a]]select] FROM t",
		"select \u00e9select, \u4e2dfrom from t": "SELECT \u00e9select, \u4e2dfrom FROM t",
		"select 1 # select from\nfrom t":         "SELECT 1 # select from\nFROM t",
	}
	for input, want := range cases {
		t.Run(input, func(t *testing.T) {
			if got := UppercaseKeywords(input); got != want {
				t.Fatalf("want %q, got %q", want, got)
			}
		})
	}
}
