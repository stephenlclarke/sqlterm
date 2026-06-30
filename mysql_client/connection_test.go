package mysqlclient

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildMySQLArgsKeepsPasswordOutOfArgv(t *testing.T) {
	args := buildMySQLArgs("/tmp/sqlterm.cnf", true)
	joined := strings.Join(args, " ")

	if !strings.Contains(joined, "--defaults-extra-file=/tmp/sqlterm.cnf") {
		t.Fatalf("expected defaults file arg, got %#v", args)
	}
	if !strings.Contains(joined, "-t") {
		t.Fatalf("expected table arg, got %#v", args)
	}
	if strings.Contains(joined, "secret") || strings.Contains(joined, "-p") {
		t.Fatalf("password leaked into argv: %#v", args)
	}
}

func TestWriteDefaultsFileUsesPrivateTempFile(t *testing.T) {
	path, cleanup, err := writeDefaultsFile(Config{
		Username: "user",
		Hostname: "localhost",
		Password: "secret",
		Port:     "3307",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cleanup()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("expected defaults file: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("expected 0600 defaults file, got %o", got)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"[client]", `user="user"`, `password="secret"`, `host="localhost"`, `port="3307"`} {
		if !strings.Contains(string(content), want) {
			t.Fatalf("expected %q in defaults file:\n%s", want, content)
		}
	}
}

func TestWriteDefaultsFileRejectsNewline(t *testing.T) {
	_, cleanup, err := writeDefaultsFile(Config{
		Username: "user",
		Hostname: "local\nhost",
	})
	if cleanup != nil {
		cleanup()
	}
	if err == nil || !strings.Contains(err.Error(), "unsupported newline") {
		t.Fatalf("expected newline validation error, got %v", err)
	}
}

func TestExecMySqlClientUsesDefaultsFile(t *testing.T) {
	tempDir := t.TempDir()
	capturePath := filepath.Join(tempDir, "args.txt")
	mysqlPath := filepath.Join(tempDir, "mysql")
	script := `#!/bin/sh
printf '%s\n' "$@" > "$CAPTURE_PATH"
case "$*" in
  *secret*) exit 2 ;;
esac
case "$1" in
  --defaults-extra-file=*) defaults_file="${1#--defaults-extra-file=}" ;;
  *) exit 3 ;;
esac
test -f "$defaults_file" || exit 4
grep -q 'password="secret"' "$defaults_file" || exit 5
grep -q 'port="3307"' "$defaults_file" || exit 6
`
	if err := os.WriteFile(mysqlPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CAPTURE_PATH", capturePath)
	originalMySQLExecutable := mysqlExecutable
	mysqlExecutable = func() (string, error) {
		return mysqlPath, nil
	}
	t.Cleanup(func() {
		mysqlExecutable = originalMySQLExecutable
	})

	err := ExecMySqlClient(Config{
		Username:      "user",
		Hostname:      "localhost",
		Password:      "secret",
		Port:          "3307",
		FormatAsTable: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(capturePath)
	if err != nil {
		t.Fatal(err)
	}
	args := strings.Fields(string(content))
	if len(args) != 2 || !strings.HasPrefix(args[0], "--defaults-extra-file=") || args[1] != "-t" {
		t.Fatalf("unexpected mysql args: %#v", args)
	}
	defaultsFile := strings.TrimPrefix(args[0], "--defaults-extra-file=")
	if _, err := os.Stat(defaultsFile); !os.IsNotExist(err) {
		t.Fatalf("expected defaults file to be cleaned up, stat err: %v", err)
	}
}

func TestDefaultMySQLExecutableFindsCandidate(t *testing.T) {
	tempDir := t.TempDir()
	mysqlPath := filepath.Join(tempDir, "mysql")
	if err := os.WriteFile(mysqlPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	originalCandidates := mysqlExecutableCandidates
	mysqlExecutableCandidates = []string{
		filepath.Join(tempDir, "missing"),
		mysqlPath,
	}
	t.Cleanup(func() {
		mysqlExecutableCandidates = originalCandidates
	})

	got, err := defaultMySQLExecutable()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != mysqlPath {
		t.Fatalf("expected %q, got %q", mysqlPath, got)
	}
}

func TestDefaultMySQLExecutableRejectsMissingCandidates(t *testing.T) {
	tempDir := t.TempDir()

	originalCandidates := mysqlExecutableCandidates
	mysqlExecutableCandidates = []string{
		filepath.Join(tempDir, "missing"),
		filepath.Join(tempDir, "also-missing"),
	}
	t.Cleanup(func() {
		mysqlExecutableCandidates = originalCandidates
	})

	if _, err := defaultMySQLExecutable(); err == nil || !strings.Contains(err.Error(), "mysql client not found") {
		t.Fatalf("expected missing mysql error, got %v", err)
	}
}

func TestIsExecutableFile(t *testing.T) {
	tempDir := t.TempDir()
	executable := filepath.Join(tempDir, "mysql")
	plainFile := filepath.Join(tempDir, "plain")

	if err := os.WriteFile(executable, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(plainFile, []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !isExecutableFile(executable) {
		t.Fatal("expected executable file")
	}
	if isExecutableFile(plainFile) {
		t.Fatal("expected plain file to be rejected")
	}
	if isExecutableFile(tempDir) {
		t.Fatal("expected directory to be rejected")
	}
}
