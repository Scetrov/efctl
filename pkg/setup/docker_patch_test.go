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

	// Create dummy compose.yml with bind mounts
	composePath := filepath.Join(dockerDir, "compose.yml")
	composeContent := `services:
  sui-dev:
    build: .
    volumes:
      - sui-config:/root/.sui
      - ../:/workspace/builder-scaffold
      - ../../world-contracts:/workspace/world-contracts
volumes:
  sui-config:
`
	os.WriteFile(composePath, []byte(composeContent), 0644)

	// Create dummy Dockerfile
	dockerfilePath := filepath.Join(dockerDir, "Dockerfile")
	dockerfileContent := `FROM ubuntu:24.04
RUN apt-get update && apt-get install -y --no-install-recommends \
    dos2unix \
    && apt-get clean
COPY scripts/ /workspace/scripts/
RUN dos2unix /workspace/scripts/*.sh && chmod +x /workspace/scripts/*.sh
ENTRYPOINT ["/workspace/scripts/entrypoint.sh"]
`
	os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644)

	// Create dummy entrypoint.sh matching upstream format
	entrypointPath := filepath.Join(scriptsDir, "entrypoint.sh")
	entrypointContent := `#!/usr/bin/env bash
set -e
SUI_CFG="${SUI_CONFIG_DIR:-/root/.sui}"
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
	err := prepareDockerEnvironment(dockerDir, true, false)
	require.NoError(t, err)

	// 2. Assert docker-compose.override.yml
	overridePath := filepath.Join(dockerDir, "docker-compose.override.yml")
	_, err = os.Stat(overridePath)
	assert.NoError(t, err, "docker-compose.override.yml should have been created")

	// Test case 2: withGraphql = false (no sql indexer or graphql)
	err = prepareDockerEnvironment(dockerDir, false, false)
	require.NoError(t, err)
	_, err = os.Stat(overridePath)
	assert.True(t, os.IsNotExist(err), "docker-compose.override.yml should have been deleted")

	// 3. Assert Dockerfile patches
	dockerfileBody, _ := os.ReadFile(dockerfilePath)
	bodyStr := string(dockerfileBody)
	assert.Contains(t, bodyStr, "postgresql-client \\", "Dockerfile should contain postgresql-client")
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

	// 5. Assert compose.yml bind mount labels and entrypoint mount
	composeBody, _ := os.ReadFile(composePath)
	cStr := string(composeBody)
	assert.Contains(t, cStr, "../:/workspace/builder-scaffold:z")
	assert.Contains(t, cStr, "../../world-contracts:/workspace/world-contracts:z")
	assert.Contains(t, cStr, "sui-config:/root/.sui", "named volumes must NOT get :z")
	assert.NotContains(t, cStr, "sui-config:/root/.sui:z")

	// Entrypoint bind mount must be present to override cached image entrypoint
	assert.Contains(t, cStr, "./scripts/entrypoint.sh:/workspace/scripts/entrypoint.sh",
		"compose.yml must bind-mount the patched entrypoint into the container")

	// 6. Test idempotency
	err = prepareDockerEnvironment(dockerDir, true, false)
	require.NoError(t, err)
	entrypointBody2, _ := os.ReadFile(entrypointPath)
	assert.Equal(t, string(entrypointBody), string(entrypointBody2), "entrypoint patch must be idempotent")
	composeBody2, _ := os.ReadFile(composePath)
	assert.Equal(t, string(composeBody), string(composeBody2), "compose patch must be idempotent")
}

// ── patchEntrypointEnvPath ─────────────────────────────────────────

func TestPatchEntrypointEnvPath_DoubleQuoted(t *testing.T) {
	content := `SUI_CFG="${SUI_CONFIG_DIR:-/root/.sui}"
ENV_FILE="/workspace/builder-scaffold/docker/.env.sui"
`
	result := patchEntrypointEnvPath(content)
	assert.Contains(t, result, `ENV_FILE="${SUI_CFG}/.env.sui"`)
	assert.NotContains(t, result, `/workspace/builder-scaffold/docker/.env.sui`)
}

func TestPatchEntrypointEnvPath_SingleQuoted(t *testing.T) {
	content := `SUI_CFG="${SUI_CONFIG_DIR:-/root/.sui}"
ENV_FILE='/workspace/builder-scaffold/docker/.env.sui'
`
	result := patchEntrypointEnvPath(content)
	assert.Contains(t, result, `ENV_FILE="${SUI_CFG}/.env.sui"`)
}

func TestPatchEntrypointEnvPath_Unquoted(t *testing.T) {
	content := `SUI_CFG="${SUI_CONFIG_DIR:-/root/.sui}"
ENV_FILE=/workspace/builder-scaffold/docker/.env.sui
`
	result := patchEntrypointEnvPath(content)
	assert.Contains(t, result, `ENV_FILE="${SUI_CFG}/.env.sui"`)
}

func TestPatchEntrypointEnvPath_ExportPrefix(t *testing.T) {
	content := `SUI_CFG="${SUI_CONFIG_DIR:-/root/.sui}"
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
	assert.Contains(t, string(body), `/root/.sui/.env.sui`)
}

// ── patchComposeYml entrypoint bind mount ──────────────────────────

func TestPatchComposeYml_InjectsEntrypointBindMount(t *testing.T) {
	tmpDir := t.TempDir()
	composePath := filepath.Join(tmpDir, "compose.yml")
	content := `services:
  sui-dev:
    build: .
    volumes:
      - sui-config:/root/.sui
      - ../:/workspace/builder-scaffold
      - ../../world-contracts:/workspace/world-contracts
volumes:
  sui-config:
`
	os.WriteFile(composePath, []byte(content), 0644)

	patchComposeYml(tmpDir)

	body, _ := os.ReadFile(composePath)
	bodyStr := string(body)
	assert.Contains(t, bodyStr, "./scripts/entrypoint.sh:/workspace/scripts/entrypoint.sh",
		"entrypoint bind mount must be injected into compose.yml")
	// Must come after the builder-scaffold mount
	bsIdx := strings.Index(bodyStr, "../:/workspace/builder-scaffold")
	epIdx := strings.Index(bodyStr, "./scripts/entrypoint.sh:/workspace/scripts/entrypoint.sh")
	assert.Greater(t, epIdx, bsIdx, "entrypoint mount must appear after builder-scaffold mount")
}

func TestPatchComposeYml_EntrypointMountIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	composePath := filepath.Join(tmpDir, "compose.yml")
	content := `services:
  sui-dev:
    build: .
    volumes:
      - sui-config:/root/.sui
      - ../:/workspace/builder-scaffold
      - ../../world-contracts:/workspace/world-contracts
volumes:
  sui-config:
`
	os.WriteFile(composePath, []byte(content), 0644)

	patchComposeYml(tmpDir)
	first, _ := os.ReadFile(composePath)

	patchComposeYml(tmpDir)
	second, _ := os.ReadFile(composePath)

	assert.Equal(t, string(first), string(second), "entrypoint mount patch must be idempotent")
}

// ── patchComposeBindMountLabels ────────────────────────────────────

func TestPatchComposeBindMountLabels_AddsZ(t *testing.T) {
	content := `    volumes:
      - sui-config:/root/.sui
      - ../:/workspace/builder-scaffold
      - ../../world-contracts:/workspace/world-contracts
`
	result, changed := patchComposeBindMountLabels(content)
	assert.True(t, changed)
	assert.Contains(t, result, "../:/workspace/builder-scaffold:z")
	assert.Contains(t, result, "../../world-contracts:/workspace/world-contracts:z")
	// Named volume must not be touched
	assert.Contains(t, result, "sui-config:/root/.sui")
	assert.NotContains(t, result, "sui-config:/root/.sui:z")
}

func TestPatchComposeBindMountLabels_Idempotent(t *testing.T) {
	content := `      - ../:/workspace/builder-scaffold:z
      - ../../world-contracts:/workspace/world-contracts:z
`
	result, changed := patchComposeBindMountLabels(content)
	assert.False(t, changed)
	assert.Equal(t, content, result)
}

func TestPatchComposeBindMountLabels_SkipsExistingSuffix(t *testing.T) {
	content := `      - ../:/workspace/builder-scaffold:ro
`
	result, changed := patchComposeBindMountLabels(content)
	assert.False(t, changed)
	assert.Equal(t, content, result)
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
	assert.Contains(t, bodyStr, `/root/.sui/.env.sui`)
	// sed line must come after the COPY + dos2unix line
	sedIdx := strings.Index(bodyStr, "RUN sed -i")
	dosIdx := strings.Index(bodyStr, "dos2unix /workspace/scripts/*.sh")
	assert.Greater(t, sedIdx, dosIdx, "sed safety-net must come after dos2unix line")
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
