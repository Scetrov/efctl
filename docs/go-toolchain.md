# Go build and security-tool baseline

The application requires **Go 1.27.2 or newer**, declared by the `go` directive in `go.mod`. CI, release and CodeQL workflows use `go-version-file: go.mod` so the minimum build version has one source of truth. A separate `toolchain` directive is unnecessary. Application modules and their `go.sum` checksums are not changed by this toolchain upgrade.

The Go version used to build efctl is unrelated to the Go/Sui versions baked into upstream development images. This upgrade does not replace Sui or diagnose an illegal-instruction crash.

## Source-analysis tool compatibility

Security tools are themselves compiled Go programs. An older `govulncheck` can fail to load Go 1.27's standard library because it predates language features such as generic methods. That is a tooling failure, not a reported application vulnerability. Use a compatible released scanner built with the current Go compiler; merely running an old binary with a newer `go` on PATH does not rebuild its parser.

Go 1.27.2 emits version 5 (V5) export data. A scanner whose bundled `golang.org/x/tools` predates `v0.50.0` cannot decode it and skips SSA analysis silently or fails. `github.com/securego/gosec/v2 v2.29.0` (the latest release) still requires `golang.org/x/tools v0.49.0` by default, so a plain `go install gosec@v2.29.0` builds a binary that cannot parse Go 1.27.2 packages; it must be built with a `replace` to a `v0.50.0`+ release.

Separately, Go 1.27 makes the `go run pkg@version` form perform a module deprecation lookup that requires a proxy version-list query. In the hermetic pre-commit sandbox (`GOPROXY=off`, read-only module cache) that lookup fails, so `actionlint` — like `gosec` and `govulncheck` — runs as an installed local binary rather than via `go run`. CI (`.github/workflows/ci.yml`) still uses `go run pkg@version` because it has network access.

The current released scanner set is `golang.org/x/vuln v1.8.0` (built with Go 1.27.1 and `golang.org/x/tools v0.50.0`), `github.com/securego/gosec/v2 v2.29.0` (built with Go 1.27.2 using an `x/tools v0.52.0` replace), and `github.com/rhysd/actionlint v1.7.12`. The checksum-authenticated module hashes are recorded in the change's verification notes. Check released versions again when updating tools; do not assume these remain the latest indefinitely.

To reproduce the repository-local installation without overwriting global tools:

```bash
mkdir -p tmp/go127-tools/{bin,gopath,mod,cache} tmp/go-build
export GOBIN="$PWD/tmp/go127-tools/bin" \
  GOPATH="$PWD/tmp/go127-tools/gopath" \
  GOMODCACHE="$PWD/tmp/go127-tools/mod" \
  GOCACHE="$PWD/tmp/go127-tools/cache" \
  TMPDIR="$PWD/tmp/go-build" \
  GOTOOLCHAIN=local GOTELEMETRY=off \
  GOSUMDB=sum.golang.org GOPROXY=https://proxy.golang.org \
  GOPRIVATE= GONOSUMDB=

# govulncheck and actionlint install directly.
go install golang.org/x/vuln/cmd/govulncheck@v1.8.0
go install github.com/rhysd/actionlint/cmd/actionlint@v1.7.12

# gosec needs the x/tools replace (>= v0.50.0) to decode Go 1.27.2's
# version 5 export data; a plain install resolves x/tools v0.49.0.
mkdir -p tmp/gosec-build && cd tmp/gosec-build
go mod init efctl-tools/gosec-build
go mod edit -require=github.com/securego/gosec/v2@v2.29.0 \
  -replace=golang.org/x/tools=golang.org/x/tools@v0.52.0
go get github.com/securego/gosec/v2/cmd/gosec@v2.29.0
go build -o ../go127-tools/bin/gosec github.com/securego/gosec/v2/cmd/gosec
cd ../..

go version -m ./tmp/go127-tools/bin/govulncheck
go version -m ./tmp/go127-tools/bin/gosec | grep x/tools
./tmp/go127-tools/bin/actionlint --version
```

Verify the reported scanner module hashes: govulncheck `h1:clG4qBU6zH5VKjti8n5j8BBuYzoSha392xXMkXS351U=`, gosec `h1:pF2HSLcnY5voqpxQumEe0O5YAAbvF4Syu4qeIewpLyc=`. Installation uses the official Go checksum database; a version string alone is not the integrity check.

Then run all configured hooks in the disposable Bubblewrap checkout with the updated binaries mounted read-only ahead of other tooling:

```bash
EFCTL_ISOLATED_TOOLBIN="$PWD/tmp/go127-tools/bin" \
  tools/test-isolated.sh --pre-commit
```

For module integrity/tidy auditing, use `tools/test-isolated.sh --modules`. That mode permits authenticated downloads from the official module proxy into a fresh writable scratch cache; it never modifies the host module cache. Default test mode keeps the pre-existing host module cache read-only, so auditing uncached transitive test dependencies requires the explicit module mode. Successful cross-builds do not constitute live tests on the target platforms.

The override must resolve under repository `./tmp`. The source snapshot, scanner caches and temporary files are writable only in scratch space; the original tool directory is read-only in the sandbox. Hook mode permits networking for scanners but exposes no host credentials, runtime sockets or checkout Git metadata. Default `tools/test-isolated.sh` test mode is offline. Failed hook runs are not considered successful vulnerability scans or completed implementation verification.

Install pre-commit hooks with `pre-commit install` if they are not already installed. See [doctor test isolation](doctor-support.md#developer-test-isolation-linux) for the runtime harness and platform limits.
