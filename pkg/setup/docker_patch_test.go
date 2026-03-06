package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── prepareDockerEnvironment (integration) ─────────────────────────

func TestPrepareDockerEnvironment(t *testing.T) {
	// Create temp dir
	tmpDir := t.TempDir()
	dockerDir := filepath.Join(tmpDir, "docker")
	scriptsDir := filepath.Join(dockerDir, "scripts")
	os.MkdirAll(scriptsDir, 0755)

	// Create dummy Dockerfile
	dockerfilePath := filepath.Join(dockerDir, "Dockerfile")
	dockerfileContent := `FROM ubuntu:24.04
RUN apt-get update && apt-get install -y --no-install-recommends \
    dos2unix \
    && apt-get clean
COPY scripts/ /workspace/scripts/
RUN dos2unix /workspace/scripts/*.sh && chmod +x /workspace/scripts/*.sh
ENV SUI_CONFIG_DIR=/root/.sui
ENTRYPOINT ["/workspace/scripts/entrypoint.sh"]
`
	os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644)

	// Create dummy entrypoint.sh matching upstream format
	entrypointPath := filepath.Join(scriptsDir, "entrypoint.sh")
	entrypointContent := `#!/usr/bin/env bash
set -e
SUI_CFG="${SUI_CONFIG_DIR:-/workspace/.sui}"
ENV_FILE="/workspace/builder-scaffold/docker/.env.sui"

# ---------- start local node ----------
sui start --with-faucet --force-regenesis &

for i in $(seq 1 30); do
  if [ "$i" -eq 30 ]; then
    echo "Fail"
  fi
done
sleep 2
echo "[sui-dev] RPC ready."
`
	os.WriteFile(entrypointPath, []byte(entrypointContent), 0755)

	// 1. Run patch
	err := prepareDockerEnvironment(dockerDir, "docker", true, false)
	require.NoError(t, err)

	// 2. Assert docker-compose.override.yml is cleaned up (not created)
	overridePath := filepath.Join(dockerDir, "docker-compose.override.yml")
	_, err = os.Stat(overridePath)
	assert.True(t, os.IsNotExist(err), "docker-compose.override.yml should not exist (compose removed)")

	// 3. Assert Dockerfile patches
	dockerfileBody, _ := os.ReadFile(dockerfilePath)
	bodyStr := string(dockerfileBody)
	assert.Contains(t, bodyStr, "postgresql-client \\", "Dockerfile should contain postgresql-client")
	assert.Contains(t, bodyStr, `ENV SUI_CONFIG_DIR=/workspace/.sui`, "Dockerfile should contain patched SUI_CONFIG_DIR")
	assert.Contains(t, bodyStr, `sed -i`, "Dockerfile should contain sed safety-net")

	// 4. Assert entrypoint.sh patches
	entrypointBody, _ := os.ReadFile(entrypointPath)
	epStr := string(entrypointBody)
	assert.Contains(t, epStr, "wait for postgres")
	assert.Contains(t, epStr, "--with-graphql=0.0.0.0:9125")
	assert.Contains(t, epStr, "for i in $(seq 1 60); do")
	assert.Contains(t, epStr, `ENV_FILE="${SUI_CFG}/.env.sui"`)
	assert.NotContains(t, epStr, `/workspace/builder-scaffold/docker/.env.sui`,
		"bind-mount .env.sui path must be fully eliminated")

	// 5. Test idempotency
	err = prepareDockerEnvironment(dockerDir, "docker", true, false)
	require.NoError(t, err)
	entrypointBody2, _ := os.ReadFile(entrypointPath)
	assert.Equal(t, string(entrypointBody), string(entrypointBody2), "entrypoint patch must be idempotent")
}

// ── Stale override cleanup ─────────────────────────────────────────

func TestPrepareDockerEnvironment_CleansUpStaleOverride(t *testing.T) {
	tmpDir := t.TempDir()
	dockerDir := filepath.Join(tmpDir, "docker")
	scriptsDir := filepath.Join(dockerDir, "scripts")
	os.MkdirAll(scriptsDir, 0755)

	// Create a stale override file from a previous compose-based version
	overridePath := filepath.Join(dockerDir, "docker-compose.override.yml")
	os.WriteFile(overridePath, []byte("services:\n  postgres:\n"), 0644)

	// Minimal files for prepareDockerEnvironment to not fail
	os.WriteFile(filepath.Join(dockerDir, "Dockerfile"), []byte("FROM ubuntu:24.04\n"), 0644)
	os.WriteFile(filepath.Join(scriptsDir, "entrypoint.sh"), []byte("#!/bin/bash\n"), 0755)

	err := prepareDockerEnvironment(dockerDir, "docker", false, false)
	require.NoError(t, err)

	_, err = os.Stat(overridePath)
	assert.True(t, os.IsNotExist(err), "stale override file should be removed")
}

// ── patchEntrypointEnvPath ─────────────────────────────────────────

func TestPatchEntrypointEnvPath_DoubleQuoted(t *testing.T) {
	content := `SUI_CFG="${SUI_CONFIG_DIR:-/workspace/.sui}"
ENV_FILE="/workspace/builder-scaffold/docker/.env.sui"
`
	result := patchEntrypointEnvPath(content)
	assert.Contains(t, result, `ENV_FILE="${SUI_CFG}/.env.sui"`)
	assert.NotContains(t, result, `/workspace/builder-scaffold/docker/.env.sui`)
}

func TestPatchEntrypointEnvPath_SingleQuoted(t *testing.T) {
	content := `SUI_CFG="${SUI_CONFIG_DIR:-/workspace/.sui}"
ENV_FILE='/workspace/builder-scaffold/docker/.env.sui'
`
	result := patchEntrypointEnvPath(content)
	assert.Contains(t, result, `ENV_FILE="${SUI_CFG}/.env.sui"`)
}

func TestPatchEntrypointEnvPath_Unquoted(t *testing.T) {
	content := `SUI_CFG="${SUI_CONFIG_DIR:-/workspace/.sui}"
ENV_FILE=/workspace/builder-scaffold/docker/.env.sui
`
	result := patchEntrypointEnvPath(content)
	assert.Contains(t, result, `ENV_FILE="${SUI_CFG}/.env.sui"`)
}

func TestPatchEntrypointEnvPath_ExportPrefix(t *testing.T) {
	content := `SUI_CFG="${SUI_CONFIG_DIR:-/workspace/.sui}"
export ENV_FILE="/workspace/builder-scaffold/docker/.env.sui"
`
	result := patchEntrypointEnvPath(content)
	assert.Contains(t, result, `export ENV_FILE="${SUI_CFG}/.env.sui"`)
	assert.NotContains(t, result, `/workspace/builder-scaffold/docker/.env.sui`)
}

func TestPatchEntrypointEnvPath_Idempotent(t *testing.T) {
	content := `ENV_FILE="/workspace/builder-scaffold/docker/.env.sui"`
	first := patchEntrypointEnvPath(content)
	second := patchEntrypointEnvPath(first)
	assert.Equal(t, first, second)
}

func TestPatchEntrypointEnvPath_AlreadyPatched(t *testing.T) {
	content := `ENV_FILE="${SUI_CFG}/.env.sui"`
	result := patchEntrypointEnvPath(content)
	assert.Equal(t, content, result, "already-patched content must not change")
}

// ── patchEntrypoint post-validation ────────────────────────────────

func TestPatchEntrypoint_ForcedFallbackOnUnrecognisedFormat(t *testing.T) {
	// Simulate an entrypoint where ENV_FILE has a format the regex doesn't
	// catch but still contains the literal bind-mount path.
	tmpDir := t.TempDir()
	scriptsDir := filepath.Join(tmpDir, "scripts")
	os.MkdirAll(scriptsDir, 0755)
	entrypointPath := filepath.Join(scriptsDir, "entrypoint.sh")

	content := `#!/usr/bin/env bash
SOME_OTHER_PATH="/workspace/builder-scaffold/docker/.env.sui"
echo "writing to /workspace/builder-scaffold/docker/.env.sui"
# ---------- start local node ----------
sui start --with-faucet --force-regenesis &
`
	os.WriteFile(entrypointPath, []byte(content), 0755)

	patchEntrypoint(tmpDir)

	body, _ := os.ReadFile(entrypointPath)
	assert.NotContains(t, string(body), `/workspace/builder-scaffold/docker/.env.sui`,
		"post-patch validation must eliminate all occurrences of the bind-mount path")
	assert.Contains(t, string(body), `/workspace/.sui/.env.sui`)
}

// ── patchDockerfile safety-net ──────────────────────────────────────

func TestPatchDockerfile_InjectsSedSafetyNet(t *testing.T) {
	tmpDir := t.TempDir()
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	content := `FROM ubuntu:24.04
COPY scripts/ /workspace/scripts/
RUN dos2unix /workspace/scripts/*.sh && chmod +x /workspace/scripts/*.sh
ENTRYPOINT ["/workspace/scripts/entrypoint.sh"]
`
	os.WriteFile(dockerfilePath, []byte(content), 0644)

	patchDockerfile(tmpDir)

	body, _ := os.ReadFile(dockerfilePath)
	bodyStr := string(body)
	assert.Contains(t, bodyStr, `RUN sed -i`)
	assert.Contains(t, bodyStr, `/workspace/.sui/.env.sui`)
	// The sed must use global replacement (trailing |g') to catch all occurrences
	assert.Contains(t, bodyStr, `|g'`, "sed must use global replacement flag")
	// sed line must come after the COPY + dos2unix line
	sedIdx := strings.Index(bodyStr, "RUN sed -i")
	dosIdx := strings.Index(bodyStr, "dos2unix /workspace/scripts/*.sh")
	assert.Greater(t, sedIdx, dosIdx, "sed safety-net must come after dos2unix line")
}

func TestPatchDockerfile_ReplacesOldNarrowSed(t *testing.T) {
	tmpDir := t.TempDir()
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	// Simulate a Dockerfile that already has the OLD narrow sed from a previous efctl version
	content := `FROM ubuntu:24.04
COPY scripts/ /workspace/scripts/
RUN dos2unix /workspace/scripts/*.sh && chmod +x /workspace/scripts/*.sh
RUN sed -i 's|ENV_FILE="/workspace/builder-scaffold/docker/.env.sui"|ENV_FILE="/workspace/.sui/.env.sui"|' /workspace/scripts/entrypoint.sh
ENTRYPOINT ["/workspace/scripts/entrypoint.sh"]
`
	os.WriteFile(dockerfilePath, []byte(content), 0644)

	patchDockerfile(tmpDir)

	body, _ := os.ReadFile(dockerfilePath)
	bodyStr := string(body)
	// Old narrow sed must be removed
	assert.NotContains(t, bodyStr, `ENV_FILE="/workspace/builder-scaffold`,
		"old narrow sed pattern must be removed")
	// New global sed must be injected
	assert.Contains(t, bodyStr, `|g'`, "new global sed must be present")
	// Only one RUN sed line should exist
	assert.Equal(t, 1, strings.Count(bodyStr, "RUN sed -i"),
		"exactly one sed safety-net line should exist")
}

func TestPatchDockerfile_SedIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	content := `FROM ubuntu:24.04
COPY scripts/ /workspace/scripts/
RUN dos2unix /workspace/scripts/*.sh && chmod +x /workspace/scripts/*.sh
ENTRYPOINT ["/workspace/scripts/entrypoint.sh"]
`
	os.WriteFile(dockerfilePath, []byte(content), 0644)

	patchDockerfile(tmpDir)
	first, _ := os.ReadFile(dockerfilePath)

	patchDockerfile(tmpDir)
	second, _ := os.ReadFile(dockerfilePath)

	assert.Equal(t, string(first), string(second), "Dockerfile sed patch must be idempotent")
}
