# SQLTerm

[![CI](https://github.com/stephenlclarke/sqlterm/actions/workflows/ci.yml/badge.svg)](https://github.com/stephenlclarke/sqlterm/actions/workflows/ci.yml)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=stephenlclarke_sqlterm&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=stephenlclarke_sqlterm)
[![Bugs](https://sonarcloud.io/api/project_badges/measure?project=stephenlclarke_sqlterm&metric=bugs)](https://sonarcloud.io/summary/new_code?id=stephenlclarke_sqlterm)
[![Code Smells](https://sonarcloud.io/api/project_badges/measure?project=stephenlclarke_sqlterm&metric=code_smells)](https://sonarcloud.io/summary/new_code?id=stephenlclarke_sqlterm)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=stephenlclarke_sqlterm&metric=coverage)](https://sonarcloud.io/summary/new_code?id=stephenlclarke_sqlterm)
[![Duplicated Lines (%)](https://sonarcloud.io/api/project_badges/measure?project=stephenlclarke_sqlterm&metric=duplicated_lines_density)](https://sonarcloud.io/summary/new_code?id=stephenlclarke_sqlterm)
[![Lines of Code](https://sonarcloud.io/api/project_badges/measure?project=stephenlclarke_sqlterm&metric=ncloc)](https://sonarcloud.io/summary/new_code?id=stephenlclarke_sqlterm)
[![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=stephenlclarke_sqlterm&metric=reliability_rating)](https://sonarcloud.io/summary/new_code?id=stephenlclarke_sqlterm)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=stephenlclarke_sqlterm&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=stephenlclarke_sqlterm)
[![Technical Debt](https://sonarcloud.io/api/project_badges/measure?project=stephenlclarke_sqlterm&metric=sqale_index)](https://sonarcloud.io/summary/new_code?id=stephenlclarke_sqlterm)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=stephenlclarke_sqlterm&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=stephenlclarke_sqlterm)
[![Vulnerabilities](https://sonarcloud.io/api/project_badges/measure?project=stephenlclarke_sqlterm&metric=vulnerabilities)](https://sonarcloud.io/summary/new_code?id=stephenlclarke_sqlterm)

Connect in an easy way to the database you need without having to memorize the credentials.

## Installation

### Credentials storage

```sh
mkdir ~/.config/sqlterm
cd ~/.config/sqlterm
touch databases.json
```

#### Credentials File Format

```json
{
  "databases": [
    {
      "key": "dev",
      "shortname": "<short name to be displayed>",
      "username": "<database username>",
      "dsn": "<ODBC DSN name>",
      "password": "<database password>",
      "database": "<optional database name>"
    }
  ]
}
```

SQLTerm accepts `dsn`, `connection_string`, or `driver` plus `hostname` as the
ODBC endpoint. Credentials remain in the config file and are passed directly to
the Go ODBC driver; no external `isql` client is required.

## Quick Start

Install from the Homebrew tap:

```sh
brew tap stephenlclarke/tap
brew install --HEAD stephenlclarke/tap/sqlterm
```

Build from source:

```sh
git clone https://github.com/stephenlclarke/sqlterm
cd sqlterm
brew install unixodbc # macOS
go build -o sqlterm ./cmd
./sqlterm
```

Use `-env` to select a configured key and `-table=YES` to render query results
as tables:

```sh
sqlterm
sqlterm -env dev -table=YES
```

Running `sqlterm` without `-env` opens an interactive Bubble Tea database
picker. Use up/down or `j`/`k` to move, `enter` to connect, or `q` to quit.
SQLTerm then opens its own query workspace using credentials from
`~/.config/sqlterm/databases.json`.

After a database is selected, SQLTerm opens an interactive SQL workspace with a
database explorer panel on the left and the SQL editor on the right. The
explorer loads database metadata over ODBC and renders a tree of databases,
tables, columns, views, indexes, functions, and observed data types. Use `Tab`
to switch between the explorer and query editor. In the explorer, use
up/down or `j`/`k` to move, `enter` or space to expand and collapse nodes, and
`Ctrl+E` to refresh metadata.

Enter SQL in the editor, use `Ctrl+F` to validate and uppercase SQL keywords
while preserving string values and identifiers, then use `Ctrl+R` to send the
query over ODBC. Results are displayed in the terminal with execution metrics
including duration, returned rows, affected rows, column count, and execution
timestamp.

## Development

```sh
make build
make test
make coverage-check
make ci
```

Tests write coverage to `reports/coverage.out`; `make coverage-check` and CI
require at least 90.0% coverage. SonarCloud analysis runs in CI when
`SONAR_TOKEN` is configured.
There is no local deploy target; tag builds produce release artefacts.

## Limitations

Currently supports SQL execution through configured ODBC endpoints.

## TODOs

- [ ] Allow to send a sql files to the connection
  - [ ] Allow to send sql files to multiple databases
- [ ] Command to do dump of one or multiple databases
- [ ] Allow support to multiple databases types
