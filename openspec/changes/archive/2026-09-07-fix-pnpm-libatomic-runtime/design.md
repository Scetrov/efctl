## Context

`efctl env up` clones configured `builder-scaffold` and `world-contracts` revisions, patches the scaffold Docker assets, builds `localhost/efctl-sui-dev`, and starts that image as `sui-playground`. `DeployWorld` then runs `pnpm` inside `sui-playground`. The optional `efctl-frontend` container separately uses `node:24-slim` and `npx pnpm`; it is not the world-deployment execution boundary.

Diagnostics on the Linux Mint Docker host `scetrov@172.29.117.128` established the failing chain:

```text
sui-playground (Ubuntu 24.04, x86_64)
  /usr/bin/pnpm (globally installed launcher, currently 12.3.4)
    -> reads world-contracts packageManager: pnpm@11.9.0
      -> /root/.local/share/pnpm/package-manager-store/v11/links/@pnpm/exe/11.9.0/.../pnpm
        -> libatomic.so.1 => not found
```

Direct `ldd` and execution of that cached binary confirmed the missing dependency. In the frontend container, the cached `@pnpm/exe@10.17.0` executable starts and does not list `libatomic`, demonstrating that the observed regression is tied to the selected standalone distribution/version rather than every pnpm execution. Neither affected container has `libatomic1` installed.

The scaffold Dockerfile is external cloned input, but `efctl` already maintains an established, tested patching layer in `pkg/setup/docker_patch.go`. The resulting image is source-derived and rebuilt locally. This change must retain direct Docker/Podman SDK orchestration and must not reintroduce Compose.

## Goals / Non-Goals

**Goals:**

- Make a fresh `sui-playground` image capable of executing the `@pnpm/exe@11.9.0` binary selected by the supported `world-contracts` project.
- Encode the fix in the existing image-preparation path using only the minimal runtime package.
- Validate the real pnpm dispatch path, clean environment creation, and recovery from partial initialization.
- Improve diagnostics enough to distinguish the frontend Node image from the Sui development image and expose the package-manager selection context.
- Validate iteratively on the available Linux Mint/Docker host until the failing deployment path succeeds or a critical external blocker is demonstrated.

**Non-Goals:**

- Installing libraries on the Linux Mint host.
- Mutating running containers as the supported fix.
- Adding compiler toolchains, `build-essential`, or unrelated packages.
- Switching container runtimes or reintroducing Docker Compose.
- Upgrading Node, pnpm, or unrelated dependencies.
- Replacing pnpm's distribution mechanism solely to remove its bundled Node runtime.

## Decisions

### Patch the Sui development image with `libatomic1`

Extend the existing Dockerfile patch to insert `libatomic1` into the established `apt-get install -y --no-install-recommends` package list. The existing Dockerfile already removes apt metadata, so the resulting layer remains minimal. The patch must be idempotent and must use the standard unmatched-target warning behavior.

This is preferred over changing `efctl-frontend`: world deployment executes in `sui-playground`, and remote evidence shows the frontend's `@pnpm/exe@10.17.0` currently starts successfully.

### Retain the existing pnpm dispatch mechanism

Keep the current global pnpm launcher and project `packageManager` selection. This preserves the upstream project's declared toolchain and makes the smallest reliable change. Corepack is not the established mechanism in the Sui image, and changing to Corepack would not by itself prove that the standalone runtime dependency disappears. Pinning or changing pnpm is also rejected as the primary remediation because it would alter toolchain behavior and could merely hide the missing runtime dependency.

The unpinned `npm install -g pnpm` remains a reproducibility concern, but changing it is separate follow-up work unless implementation testing proves it is necessary for this regression.

### Test behavior in addition to patch output

Unit coverage will verify insertion, idempotency, and warning behavior in the Dockerfile patch. A container smoke test will build the actual patched scaffold image and invoke pnpm from a working directory whose `packageManager` selects the affected version. Passing means that the selected executable starts and reports its version; checking only package text or a distro-specific library path is insufficient as the primary regression test.

Where useful, diagnostics may additionally run `ldd` against the cached standalone executable and verify that `libatomic.so.1` resolves. The behavioral `pnpm --version` result remains authoritative.

### Make the execution boundary visible in diagnostics

Deployment diagnostics will identify `sui-playground`, the base OS/architecture when available, the launcher resolved by `command -v pnpm`, and the project's `packageManager` declaration before invoking pnpm. The command must preserve pnpm stderr, because pnpm's dispatcher reports the selected standalone executable path when dynamic loading fails. Diagnostics must not leak environment values or block deployment.

### Use the remote host as an iterative acceptance environment

Implementation validation may build a candidate `efctl` locally, transfer only the candidate executable or necessary test artifact to a repository-scoped/safe temporary location on `scetrov@172.29.117.128`, and exercise Docker there. The loop continues through clean teardown, recreation, and world deployment until the failure is resolved. Running `apt-get install` in an existing container is permitted only as diagnostic confirmation, never as acceptance evidence.

## Risks / Trade-offs

- **[Upstream Dockerfile shape changes and the text patch no longer matches]** → Reuse standard patch warnings and tests for both insertion and already-applied states; fail behavioral smoke validation if the image lacks the runtime.
- **[A future pnpm executable adds other shared-library dependencies]** → Exercise the project-selected binary behaviorally and retain stderr/selection diagnostics rather than asserting only `libatomic1` text.
- **[Architecture behavior differs]** → Treat x86_64 as directly proven; run arm64 smoke coverage where CI/build infrastructure supports it and document unverified architectures truthfully.
- **[The global pnpm release drifts because installation is unpinned]** → Preserve scope for this fix, record the risk, and propose a separate pinning change if reproducibility testing warrants it.
- **[Remote validation disrupts an existing environment]** → Inspect state first, use normal `efctl env down` semantics, transfer candidates to a safe temporary path, and avoid host package or source modifications.
- **[Package installation modestly increases image size]** → Install only `libatomic1` with `--no-install-recommends` and retain apt-list cleanup.

## Migration Plan

1. Add test-first coverage for `libatomic1` patch insertion, idempotency, and diagnostics.
2. Update the existing Dockerfile patch; do not alter cloned upstream repositories directly.
3. Build a fresh Sui development image without reusing the previously broken image as acceptance evidence.
4. Validate the selected pnpm executable and full deployment locally where possible, then on the provided Linux Mint/Docker host.
5. Tear down the partially initialized remote environment using normal `efctl env down`, install/upload the candidate `efctl`, recreate from scratch, and confirm world deployment completes.
6. Roll back by reverting the `efctl` patch and rebuilding the local image; no host migration or persistent data format change is involved.

## Open Questions

- Does the current `@pnpm/exe@11.9.0` arm64 artifact also link against `libatomic.so.1`? This should be measured when arm64 execution is available, but `libatomic1` is an architecture-appropriate runtime package on supported Debian/Ubuntu platforms.
- Should the separate unpinned global pnpm bootstrap be addressed in a follow-up reproducibility proposal after this focused fix is verified?
