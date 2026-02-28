.PHONY: build test test-unit test-integration test-e2e test-all test-coverage vet fmt lint security sec-all gosec govulncheck clean

# ── Build & Test ──────────────────────────────────────────────

build:
	go build -o output/efctl main.go

test:
	go test -count=1 ./...

test-unit: test

test-integration:
	go test -count=1 -tags integration -v ./tests/integration/...

test-e2e: build
	EFCTL_BINARY=$(PWD)/output/efctl go test -count=1 -tags e2e -timeout 15m -v ./tests/e2e/...

test-all: test test-integration test-e2e

test-coverage:
	go test -count=1 -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -1
	@echo ""
	@echo "HTML report: go tool cover -html=coverage.out -o coverage.html"

vet:
	go vet ./...

fmt:
	go fmt ./...

# ── Security ──────────────────────────────────────────────────

# Run all local security checks (same as CI)
security: gosec govulncheck vet test
	@echo ""
	@echo "✅ All security checks passed."

# Alias for 'security'
sec-all: security

# Static analysis for Go security issues
gosec:
	@echo "▶ Running gosec..."
	gosec -severity medium -confidence medium -exclude-generated ./...

# Check dependencies for known vulnerabilities
govulncheck:
	@echo "▶ Running govulncheck..."
	govulncheck ./...

# ── Utilities ─────────────────────────────────────────────────

# Install security tools locally
install-sec-tools:
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	@echo "✅ Security tools installed."

# Full pre-commit equivalent (without needing pre-commit framework)
pre-commit: fmt vet build test gosec govulncheck
	@echo ""
	@echo "✅ All pre-commit checks passed."

clean:
	rm -f output/efctl output/efctl-*
	rm -f gosec-results.json gosec-ci-results.json
