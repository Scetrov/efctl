## Why

A clean Linux/Docker environment can fail during world deployment because the project-selected `@pnpm/exe` standalone pnpm binary requires `libatomic.so.1`, while the `sui-playground` image does not install the corresponding runtime package. The dependency must be supplied by the image rather than by the Linux Mint host or by mutating an already-running container.

## What Changes

- Add the minimal `libatomic1` runtime dependency to the source-controlled Docker image preparation path used for `sui-playground`.
- Preserve the existing pnpm distribution mechanism and project-selected pnpm version unless implementation evidence shows that mechanism is accidental or defective beyond the missing runtime library.
- Add behavioral regression coverage proving that the project-selected pnpm executable starts successfully in a freshly built image.
- Validate teardown and recreation after a partially initialized environment.
- Document the container boundary, pnpm dispatch path, runtime dependency, and supported recovery workflow.
- Validate candidate builds on the available Linux Mint/Docker host over passwordless SSH without requiring host-level package installation.

## Capabilities

### New Capabilities
- `pnpm-container-runtime`: Defines runtime and validation requirements for executing the project-selected pnpm distribution in the Sui development container.

### Modified Capabilities
- `pnpm-diagnostics`: Extend deployment diagnostics to make the selected pnpm runtime and missing shared-library failures identifiable without assuming that `command -v pnpm` is the final executable.

## Impact

- `pkg/setup/docker_patch.go` and its tests, which patch the cloned `builder-scaffold/docker/Dockerfile` before image construction.
- The locally built `localhost/efctl-sui-dev` image and `sui-playground` container.
- World deployment commands in `pkg/setup/constants.go` and related diagnostic tests if richer runtime diagnostics are required.
- Docker smoke/e2e coverage and toolchain documentation.
- Remote validation on `scetrov@172.29.117.128`; no host package changes and no supported workflow based on `docker exec apt-get install`.
