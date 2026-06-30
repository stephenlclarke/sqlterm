SHELL := /usr/bin/env bash
.SHELLFLAGS := -eo pipefail -c

GO ?= go
BINARY_NAME := sqlterm
BUILD_DIR := bin
REPORTS_DIR := reports
PKG := ./...
GO_JUNIT_REPORT := $(shell $(GO) env GOPATH)/bin/go-junit-report
STATICCHECK := $(shell $(GO) env GOPATH)/bin/staticcheck
GITLEAKS := $(shell $(GO) env GOPATH)/bin/gitleaks
GO_JUNIT_REPORT_VERSION ?= v2.1.0
STATICCHECK_VERSION ?= v0.6.1
GITLEAKS_VERSION ?= v8.30.1
COVERAGE_MIN ?= 90.0
RELEASE_TARGETS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64

.PHONY: all build build-release clean ci coverage-check help run scan security-scan staticcheck test vet

all: ci

build:
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd

build-release:
	@mkdir -p $(BUILD_DIR)
	@for target in $(RELEASE_TARGETS); do \
		target_os="$${target%/*}"; \
		target_arch="$${target#*/}"; \
		extension=""; \
		if [ "$${target_os}" = "windows" ]; then extension=".exe"; fi; \
		printf 'Building %s/%s\n' "$${target_os}" "$${target_arch}"; \
		CGO_ENABLED=0 GOOS="$${target_os}" GOARCH="$${target_arch}" \
			$(GO) build -trimpath -o "$(BUILD_DIR)/$(BINARY_NAME)_$${target_os}_$${target_arch}$${extension}" ./cmd; \
	done

clean:
	rm -rf $(BUILD_DIR) $(REPORTS_DIR) coverage.out .scannerwork

ci: build coverage-check vet staticcheck security-scan

coverage-check: test
	@coverage="$$($(GO) tool cover -func=$(REPORTS_DIR)/coverage.out | awk '/^total:/ {gsub(/%/, "", $$3); print $$3}')"; \
		awk -v coverage="$${coverage}" -v minimum="$(COVERAGE_MIN)" 'BEGIN { \
			if (coverage + 0 < minimum + 0) { \
				printf "coverage %.1f%% is below required %.1f%%\n", coverage, minimum; \
				exit 1; \
			} \
			printf "coverage %.1f%% meets required %.1f%%\n", coverage, minimum; \
		}'

run:
	$(GO) run ./cmd

test:
	@mkdir -p $(REPORTS_DIR)
	@if ! command -v go-junit-report >/dev/null 2>&1 && [ ! -x "$(GO_JUNIT_REPORT)" ]; then \
		$(GO) install github.com/jstemmer/go-junit-report/v2@$(GO_JUNIT_REPORT_VERSION); \
	fi
	$(GO) test -v -covermode=atomic -coverprofile=$(REPORTS_DIR)/coverage.out $(PKG)
	@reporter="$$(command -v go-junit-report || printf '%s' '$(GO_JUNIT_REPORT)')"; \
		$(GO) test -json $(PKG) | "$${reporter}" > $(REPORTS_DIR)/unit-test-report.xml

vet:
	$(GO) vet $(PKG)

staticcheck:
	@if ! command -v staticcheck >/dev/null 2>&1 && [ ! -x "$(STATICCHECK)" ]; then \
		$(GO) install honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION); \
	fi
	@checker="$$(command -v staticcheck || printf '%s' '$(STATICCHECK)')"; \
		"$${checker}" $(PKG)

security-scan:
	@if ! command -v gitleaks >/dev/null 2>&1 && [ ! -x "$(GITLEAKS)" ]; then \
		$(GO) install github.com/zricethezav/gitleaks/v8@$(GITLEAKS_VERSION); \
	fi
	@scanner="$$(command -v gitleaks || printf '%s' '$(GITLEAKS)')"; \
		"$${scanner}" detect --source . --no-banner --redact --verbose

scan:
	@if [ -z "$${SONAR_TOKEN:-}" ]; then \
		printf 'SONAR_TOKEN is not configured; skipping SonarCloud analysis.\n'; \
	elif command -v sonar-scanner >/dev/null 2>&1; then \
		sonar-scanner -Dsonar.token="$${SONAR_TOKEN}" $(SCAN_ARGS); \
	else \
		docker run --rm \
			-e SONAR_TOKEN="$${SONAR_TOKEN}" \
			-e SONAR_SCANNER_SKIP_JRE_PROVISIONING=true \
			-v "$$(pwd):/usr/src" \
			sonarsource/sonar-scanner-cli \
			-Dsonar.token="$${SONAR_TOKEN}" \
			$(SCAN_ARGS); \
	fi

help:
	@printf 'Available targets:\n'
	@printf '  build          Build %s into %s/\n' "$(BINARY_NAME)" "$(BUILD_DIR)"
	@printf '  build-release  Build release binaries for common platforms\n'
	@printf '  clean          Remove local generated outputs\n'
	@printf '  ci             Run build, coverage gate, vet, staticcheck, and secret scan\n'
	@printf '  coverage-check Run tests and require at least %.1f%% coverage\n' "$(COVERAGE_MIN)"
	@printf '  run            Run the CLI from source\n'
	@printf '  scan           Run SonarCloud when SONAR_TOKEN is configured\n'
	@printf '  test           Run Go tests with coverage and JUnit reports\n'
	@printf '  vet            Run go vet\n'
