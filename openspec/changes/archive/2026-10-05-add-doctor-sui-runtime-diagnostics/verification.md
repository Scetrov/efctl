# Verification evidence

## Verified locally

- Linux amd64 / WSL-visible CPU view; Go 1.27.1, Bubblewrap 0.12.0, rootless Podman 6.1.2.
- Entire Go suite in a disposable, offline Bubblewrap source snapshot. The sandbox hides host homes, checkout Git metadata, runtime sockets and container storage; all writable files are backed by repository `./tmp`. Host HEAD and remote remained unchanged.
- Actual local runtime checks with native Podman and a Docker-compatible wrapper around the same Podman engine, using a separate VFS store under `./tmp`. `podman unshare` establishes the user namespace before Bubblewrap; direct rootless Podman inside Bubblewrap fails UID mapping here.
- Running managed fixture, stopped fixture using its immutable image after the tag changes, absent-container/image-only fallback, hung version probe with bounded cleanup, and declared-volume image rejection.
- No pulls, changed managed fixture sentinel, surviving disposable probes, or anonymous volume leaks in those integration scenarios. Fixture commands and container/volume checks are retained in `tmp/doctor-runtime-integration.log` and the dedicated runtime scratch store while debugging.
- Test base was an already-local image ID `c93bce62b190a2cc5ed3c7f87fdf9832fd37f982f2c941bf6b5b327f65374fbb`. It was exported/imported offline; no image dependency was added to production. The integration Sui executable is a small version-only shell fixture, not an upstream Sui binary.
- Fixtures cover platform aliases, ARM/x86 Linux parsing, conservative heterogeneous CPU intersections, incomplete sources, macOS/Windows query responses, missing executables, cancellation, timeout, signal observation, output limits, unsafe controls and credential-bearing metadata.
- Report review: successful/partial rendering fixtures retain existing sections, distinguish host/managed/fallback observations, identify recorded container exit separately from Sui probe failure, retain explicit unavailability reasons, and do not warn on architecture aliases. Mismatch wording is possible emulation, not a crash diagnosis. Complete-report security and unconfigured-host tests pass.

## Differences found by integration testing

- Podman image inspect templates need `.ID` inside the `json` function; the `.Id` compatibility alias does not apply there. Native Podman uses that template, with a bounded fallback for Docker-compatible wrappers.
- Podman force-removal may wait a stop grace period exceeding the three-second cleanup allowance. After verifying ownership, doctor sends KILL to that probe before force-removal, using the same allowance. Stopped probes can reject KILL; removal is still attempted.

## Untested combinations and limitations

- No genuine Docker daemon, Docker Desktop VM, remote Docker/Podman service, macOS host, Windows host or ARM host was exercised. Their supported metadata/parsing paths have fixtures, not live verification.
- No real upstream Sui installation/image was exercised, and the reported user's SIGILL root cause remains unknown. Version-only success is not proof of startup compatibility.
- Container cancellation and observed SIGILL have injected/unit fixtures; live integration explicitly verifies timeout cleanup. A fixture cannot establish every runtime's behavior after abrupt efctl termination.
- Legacy gatherers are not covered by the new 20-second probe budget.
- Before safe isolation was established, repository-local temporary Git fixtures discovered the parent checkout and altered its remote/HEAD. Recovery restored efctl `eca8c61` and the canonical `https://github.com/Scetrov/efctl.git` URL, preserving implementation changes. Backups are in `tmp/doctor-diagnostics-recovery/`. The exact original remote transport URL was not recorded. The Git test now uses a local bare remote, negative fixtures use Git ceilings, and further full-suite runs use Bubblewrap snapshots.

## Go 1.27.1 baseline verification

- The official [Go release endpoint](https://go.dev/dl/?mode=json) identified Go 1.27.1 as the current stable release. The project directive is now `go 1.27.1`; README source-build guidance agrees. CI/release/CodeQL keep their existing `go-version-file: go.mod` selection. Regression tests first failed on the old directive, then passed after the update.
- The complete Go suite passes with Go 1.27.1 in the isolated hook run. A separate complete `go test -race ./... -count=1` also passes. Race-instrumented subprocess fixtures required disabling only the detector's one-second exit delay (`atexit_sleep_ms=0`); race detection and production deadlines were not disabled or relaxed.
- `go mod tidy -diff` reports no change, `go mod verify` passes, and Linux amd64/arm64, macOS amd64/arm64 and Windows amd64/arm64 builds succeed with `CGO_ENABLED=0`. These are cross-builds, not live testing on those platforms.
- Dependency requirements are unchanged apart from the Go directive, and `go.sum` is byte-for-byte unchanged from the original efctl HEAD. Module auditing used a fresh writable scratch cache with official proxy/checksum authentication, since the read-only offline cache lacked two existing test-only dependencies. Scratch module directories are made writable only for their own cleanup; host caches remain read-only.
- Evidence logs: `tmp/go127-race.log`, `tmp/go127-module-crossbuild.log`, and `tmp/pre-commit-go127.log`. Host Git and `go.sum` remained unchanged during sandbox execution.

### Released scanner provenance

The official [module proxy](https://proxy.golang.org/golang.org/x/vuln/@latest) identified `golang.org/x/vuln v1.8.0` (released 2026-09-08). Installed only under `tmp/go127-tools`, using Go 1.27.1, `GOTOOLCHAIN=local`, the official proxy and `GOSUMDB=sum.golang.org`. No application scanner dependency or global-tool overwrite was introduced.

| Module | Version | Verified Go module checksum |
| --- | --- | --- |
| `golang.org/x/vuln` | `v1.8.0` | `h1:clG4qBU6zH5VKjti8n5j8BBuYzoSha392xXMkXS351U=` |
| `golang.org/x/tools` | `v0.50.0` | `h1:c2ifzfcuY7L90lZ2aKd8S4K2NpASF08SZx9ZuJkHmSU=` |
| `golang.org/x/mod` | `v0.41.0` | `h1:qJmnOUb4YB+FsEuM3HcWucdZASCPGhsX6uljO6pog0c=` |
| `golang.org/x/sync` | `v0.23.0` | `h1:KameEIfc1IkluZyXWLn39Wd4tURc6GbCiISGiZm2bQk=` |
| `golang.org/x/telemetry` | `v0.0.0-20260908163034-4bcc4b2ee518` | `h1:F5BWKvW126NXR74uxkxuc1jQHhm/rwm/J3rSiFyuRs4=` |

Local scanner binary SHA256: `42d0768bede55942485b281bbf9c03d87a918a1e45de6d209f488182db0468e8`. See `docs/go-toolchain.md` for reproduction and read-only tool override instructions.

## Quality gates

With `EFCTL_ISOLATED_TOOLBIN` selecting the repository-local scanner, all configured hooks were rerun in a disposable Bubblewrap checkout after the complexity refactor. Documentation generation, formatting, vet, build, the complete Go suite, gosec, **govulncheck**, gocyclo, and gitleaks passed. The threshold was not weakened or ignored. The local scanner SHA256 remained `42d0768bede55942485b281bbf9c03d87a918a1e45de6d209f488182db0468e8`. Host Git remained unchanged.

The seven previous complexity violations were split into smaller production and test helpers without changing diagnostic behavior or security assertions. `gocyclo -over 15 .` is clean.

Post-refactor evidence:

- `tmp/pre-commit-go127-refactor.log`: all configured hooks passed.
- `tmp/go127-race-refactor.log`: `go test -race ./... -count=1` passed.
- `tmp/doctor-runtime-integration-refactor.log`: Podman and Docker-compatible wrapper scenarios passed again after the refactor, using the same already-local image ID and no pulls.
- Isolated runtime scratch cleanup now uses `podman unshare rm` for the dedicated VFS store. Host `rm` cannot remove root-owned image layers created inside the user namespace; the failed host cleanup was not a probe leak. The rerun exited 0 and removed its own scratch store.

`openspec validate add-doctor-sui-runtime-diagnostics --strict` passed before archival. The Git `pre-commit` hook is still not installed; run `pre-commit install` so future commits invoke the same hooks automatically.

## Pull request preparation recheck

- All configured pre-commit hooks passed again in the isolated workspace (`tmp/pr-pre-commit.log`).
- The complete race suite passed (`tmp/pr-race.log`).
- Isolated Podman and Docker-compatible wrapper integration scenarios passed again (`tmp/pr-runtime-integration.log`). This does not constitute genuine Docker daemon verification.
- Both new archived canonical specs pass strict validation. Repository-wide strict validation reports seven pre-existing failures in unchanged tracked specs, all missing required Purpose/Requirements sections (`tmp/pr-openspec-validation.json`); these unrelated specifications were not modified.
- `git diff --check` passed; the remote main branch still matches the implementation base `eca8c61`.
