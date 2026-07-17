SHELL := /bin/sh

BINARY := efctl
OUTPUT_DIR := output
BINARY_PATH := $(OUTPUT_DIR)/$(BINARY)
GO ?= go
GOFLAGS ?=
LDFLAGS ?= -s -w

.DEFAULT_GOAL := help

.PHONY: help build build-windows test test-unit test-integration test-e2e test-all \
	test-coverage fmt fmt-check vet lint security sec-all gosec govulncheck \
	docs tidy check pre-commit install-sec-tools env-down clean destroy-reset

## help: Show the available development targets.
help:
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_-]+:.*## / {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## build: Build the efctl binary in output/.
build:
	@mkdir -p $(OUTPUT_DIR)
	$(GO) build $(GOFLAGS) -trimpath -ldflags="$(LDFLAGS)" -o $(BINARY_PATH) main.go

## build-windows: Cross-compile an amd64 Windows binary.
build-windows:
	@mkdir -p $(OUTPUT_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -trimpath -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/$(BINARY).exe main.go

## test: Run the unit test suite.
test:
	$(GO) test $(GOFLAGS) -count=1 ./...

## test-unit: Alias for test.
test-unit: test

## test-integration: Run integration tests.
test-integration:
	$(GO) test $(GOFLAGS) -count=1 -tags integration -v ./tests/integration/...

## test-e2e: Build efctl and run Docker-backed end-to-end tests.
test-e2e: build
	EFCTL_BINARY=$(CURDIR)/$(BINARY_PATH) $(GO) test $(GOFLAGS) -count=1 -tags e2e -timeout 15m -v ./tests/e2e/...

## test-all: Run unit, integration, and end-to-end tests.
test-all: test test-integration test-e2e

## test-coverage: Run unit tests and generate text and HTML coverage reports.
test-coverage:
	$(GO) test $(GOFLAGS) -count=1 -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
	@printf 'HTML report: coverage.html\n'

## fmt: Format all Go source files.
fmt:
	$(GO) fmt ./...

## fmt-check: Fail if any Go source file needs formatting.
fmt-check:
	@test -z "$$(gofmt -l .)" || { echo "Go files need formatting; run 'make fmt'."; gofmt -l .; exit 1; }

## vet: Run the Go static analyzer.
vet:
	$(GO) vet ./...

## lint: Run formatting and static-analysis checks.
lint: fmt-check vet

## gosec: Scan Go code for security issues.
gosec:
	gosec -severity medium -confidence medium -exclude-generated ./...

## govulncheck: Check dependencies for known vulnerabilities.
govulncheck:
	govulncheck ./...

## security: Run security, vet, and unit-test checks.
security: gosec govulncheck vet test

## sec-all: Alias for security.
sec-all: security

## docs: Regenerate CLI reference documentation.
docs:
	$(GO) run ./tools/gen-docs/main.go

## tidy: Update and verify Go module metadata.
tidy:
	$(GO) mod tidy
	$(GO) mod verify

## check: Run the standard local build and quality gates.
check: lint build test

## pre-commit: Run every configured pre-commit hook.
pre-commit:
	pre-commit run --all-files

## install-sec-tools: Install local Go security tools.
install-sec-tools:
	$(GO) install github.com/securego/gosec/v2/cmd/gosec@latest
	$(GO) install golang.org/x/vuln/cmd/govulncheck@latest
	$(GO) install github.com/fzipp/gocyclo/cmd/gocyclo@latest

## env-down: Tear down the local efctl environment.
env-down: build
	$(BINARY_PATH) --no-progress env down

## clean: Remove local build and coverage artifacts.
clean:
	rm -rf $(OUTPUT_DIR)
	rm -f coverage.out coverage.html gosec-results.json gosec-ci-results.json

## destroy-reset: Destructively remove all local containers, images, and volumes.
destroy-reset:
	podman stop $$(podman ps -qa || true) > /dev/null 2>&1 || docker stop $$(docker ps -qa || true) > /dev/null 2>&1 || true
	podman rm $$(podman ps -qa || true) > /dev/null 2>&1 || docker rm $$(docker ps -qa || true) > /dev/null 2>&1 || true
	podman system prune --all --volumes --force || docker system prune --all --volumes --force || true
