# Tasks: Bump Go baseline to 1.27.2

## 1. Baseline

- [x] 1.1 Raise the `go.mod` `go` directive to 1.27.2.
- [x] 1.2 Update the toolchain guard (`toolchain_test.go`) and `openspec/specs/go-toolchain-baseline/spec.md` to 1.27.2.
- [x] 1.3 Update source-build documentation (`README.md`, `docs/go-toolchain.md`) to 1.27.2.

## 2. Toolchain compatibility

- [x] 2.1 Rebuild `gosec v2.29.0` with an `x/tools v0.52.0` replace so it decodes V5 export data; verify a project scan reports 0 issues and exit 0.
- [x] 2.2 Install `actionlint v1.7.12` as a local binary and switch the pre-commit hook from `go run pkg@version` to the binary.
- [x] 2.3 Record the updated scanner set and checksum-authenticated module hashes in `docs/go-toolchain.md`.

## 3. Verification

- [x] 3.1 Run the full isolated test suite on 1.27.2 (`tools/test-isolated.sh`); all packages pass.
- [x] 3.2 Run the full pre-commit gate in the hermetic sandbox (`EFCTL_ISOLATED_TOOLBIN=tmp/go127-tools/bin ./tools/test-isolated.sh --pre-commit`); all hooks pass, exit 0.
