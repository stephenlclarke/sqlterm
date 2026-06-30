package mysqlclient

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Config struct {
	Username      string
	Hostname      string
	Password      string
	Port          string
	FormatAsTable bool
}

var mysqlExecutable = defaultMySQLExecutable

func ExecMySqlClient(config Config) error {
	fmt.Printf("connecting to %s host\n", config.Hostname)

	defaultsFile, cleanup, err := writeDefaultsFile(config)
	if err != nil {
		return err
	}
	defer cleanup()

	args := buildMySQLArgs(defaultsFile, config.FormatAsTable)
	mysqlPath, err := mysqlExecutable()
	if err != nil {
		return err
	}

	cmd := exec.Command(mysqlPath, args...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error connecting to mysql: %w", err)
	}

	return nil
}

func defaultMySQLExecutable() (string, error) {
	for _, path := range mysqlExecutableCandidates {
		if isExecutableFile(path) {
			return path, nil
		}
	}

	return "", errors.New("mysql client not found in supported install locations")
}

var mysqlExecutableCandidates = []string{
	"/opt/homebrew/opt/mysql-client/bin/mysql",
	"/usr/local/opt/mysql-client/bin/mysql",
	"/opt/homebrew/bin/mysql",
	"/usr/local/bin/mysql",
	"/usr/local/mysql/bin/mysql",
	"/opt/local/bin/mysql",
	"/usr/bin/mysql",
}

func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}

	return info.Mode().Perm()&0o111 != 0
}

func buildMySQLArgs(defaultsFile string, formatAsTable bool) []string {
	args := []string{"--defaults-extra-file=" + defaultsFile}
	if formatAsTable {
		args = append(args, "-t")
	}

	return args
}

func writeDefaultsFile(config Config) (string, func(), error) {
	defaultsFile, err := os.CreateTemp("", "sqlterm-mysql-*.cnf")
	if err != nil {
		return "", nil, err
	}

	cleanup := func() {
		_ = os.Remove(defaultsFile.Name())
	}

	if err := writeDefaults(defaultsFile, config); err != nil {
		_ = defaultsFile.Close()
		cleanup()
		return "", nil, err
	}
	if err := defaultsFile.Close(); err != nil {
		cleanup()
		return "", nil, err
	}

	return defaultsFile.Name(), cleanup, nil
}

func writeDefaults(w io.Writer, config Config) error {
	writer := bufio.NewWriter(w)
	if _, err := fmt.Fprintln(writer, "[client]"); err != nil {
		return err
	}

	values := map[string]string{
		"user":     config.Username,
		"password": config.Password,
		"host":     config.Hostname,
		"port":     config.Port,
	}
	for key, value := range values {
		if value == "" {
			continue
		}
		if strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("mysql %s value contains an unsupported newline", key)
		}
		if _, err := fmt.Fprintf(writer, "%s=%q\n", key, value); err != nil {
			return err
		}
	}

	return writer.Flush()
}
