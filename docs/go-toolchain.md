# Go build and security-tool baseline

The application requires **Go 1.27.1 or newer**, declared by the `go` directive in `go.mod`. CI, release and CodeQL workflows use `go-version-file: go.mod` so the minimum build version has one source of truth. A separate `toolchain` directive is unnecessary. Application modules and their `go.sum` checksums are not changed by this toolchain upgrade.

The Go version used to build efctl is unrelated to the Go/Sui versions baked into upstream development images. This upgrade does not replace Sui or diagnose an illegal-instruction crash.

## Source-analysis tool compatibility

Security tools are themselves compiled Go programs. An older `govulncheck` can fail to load Go 1.27's standard library because it predates language features such as generic methods. That is a tooling failure, not a reported application vulnerability. Use a compatible released scanner built with the current Go compiler; merely running an old binary with a newer `go` on PATH does not rebuild its parser.

For this change, the official module proxy identified `golang.org/x/vuln v1.8.0` as the current release. It was built with Go 1.27.1 and `golang.org/x/tools v0.50.0`. The checksum-authenticated module hashes are recorded in the change's verification notes. Check released versions again when updating tools; do not assume these remain the latest indefinitely.

To reproduce the repository-local installation without overwriting global tools:

```bash
mkdir -p tmp/go127-tools/{bin,gopath,mod,cache} tmp/go-build
GOBIN="$PWD/tmp/go127-tools/bin" \
GOPATH="$PWD/tmp/go127-tools/gopath" \
GOMODCACHE="$PWD/tmp/go127-tools/mod" \
GOCACHE="$PWD/tmp/go127-tools/cache" \
TMPDIR="$PWD/tmp/go-build" \
GOTOOLCHAIN=local GOTELEMETRY=off \
GOSUMDB=sum.golang.org GOPROXY=https://proxy.golang.org \
GOPRIVATE= GONOSUMDB= \
go install golang.org/x/vuln/cmd/govulncheck@v1.8.0

go version -m ./tmp/go127-tools/bin/govulncheck
```

Verify the reported scanner module hash is `h1:clG4qBU6zH5VKjti8n5j8BBuYzoSha392xXMkXS351U=`. Installation uses the official Go checksum database; a version string alone is not the integrity check.

Then run all configured hooks in the disposable Bubblewrap checkout with the updated binaries mounted read-only ahead of other tooling:

```bash
EFCTL_ISOLATED_TOOLBIN="$PWD/tmp/go127-tools/bin" \
  tools/test-isolated.sh --pre-commit
```

For module integrity/tidy auditing, use `tools/test-isolated.sh --modules`. That mode permits authenticated downloads from the official module proxy into a fresh writable scratch cache; it never modifies the host module cache. Default test mode keeps the pre-existing host module cache read-only, so auditing uncached transitive test dependencies requires the explicit module mode. Successful cross-builds do not constitute live tests on the target platforms.

The override must resolve under repository `./tmp`. The source snapshot, scanner caches and temporary files are writable only in scratch space; the original tool directory is read-only in the sandbox. Hook mode permits networking for scanners but exposes no host credentials, runtime sockets or checkout Git metadata. Default `tools/test-isolated.sh` test mode is offline. Failed hook runs are not considered successful vulnerability scans or completed implementation verification.

Install pre-commit hooks with `pre-commit install` if they are not already installed. See [doctor test isolation](doctor-support.md#developer-test-isolation-linux) for the runtime harness and platform limits.
