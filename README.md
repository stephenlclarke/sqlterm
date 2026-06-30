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

Connect in a easy way to the database you need without having to memorize the credentials.

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
      "hostname": "<database hostname or ip>",
      "password": "<database password>",
      "port": "<database port>"
    }
  ]
}
```

## Quick Start

Install from the Homebrew tap:

```sh
brew tap stephenlclarke/tap
brew install --HEAD stephenlclarke/tap/sqlterm
```

Build from source:

```sh
git clone https://github.com/jpxcz/sqlterm
# in the future we will need to get the dependencies `go get .`
cd sqlterm
go build -o sqlterm ./cmd/main.go
./sqlterm
```

Use `-env` to select a configured key and `-table=YES` to ask the MySQL client
to format results as tables:

```sh
sqlterm -env dev -table=YES
```

## Development

```sh
make build
make test
make coverage-check
make ci
```

Tests write coverage to `reports/coverage.out`; `make coverage-check` and CI
require at least 80.0% coverage. SonarCloud analysis runs in CI when
`SONAR_TOKEN` is configured.
There is no local deploy target; tag builds produce release artefacts.

## Limitations

Currently only supported MySQL.

## TODOs

- [ ] Add BubbleTea for a better UX interface
- [ ] Allow to send a sql files to the connection
  - [ ] Allow to send sql files to multiple databases
- [ ] Command to do dump of one or multiple databases
- [ ] Allow support to multiple databases types
