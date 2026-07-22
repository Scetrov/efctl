package setup

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pterm/pterm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"efctl/pkg/ui"
)

// ── Test helpers ────────────────────────────────────────────────────

// captureWarnings redirects ui.Warn output to a buffer and disables pterm
// styling so assertions can match plain text. It returns a pointer to the
// buffer; the original writer and styling mode are restored automatically.
func captureWarnings(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	w := io.Writer(&buf)
	ui.Warn.Writer = w
	pterm.DisableStyling()
	t.Cleanup(func() {
		ui.Warn.Writer = nil
		pterm.EnableStyling()
	})
	return &buf
}

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

// ── patchEntrypointFaucetWait ────────────────────────────────────────

func TestPatchEntrypointFaucetWait_ReplacesFixedSleep(t *testing.T) {
	content := `echo "[sui-dev] RPC responding, waiting for full initialization..."
sleep 5
echo "[sui-dev] Node ready."`

	result := patchEntrypointFaucetWait(content)
	assert.Contains(t, result, "Waiting for faucet on port 9123")
	assert.Contains(t, result, "curl --max-time 10 -s -o /dev/null http://127.0.0.1:9123")
	assert.Contains(t, result, `echo "[sui-dev] Node ready."`)
	// The unconditional fixed sleep must be gone in favour of a bounded poll.
	assert.NotContains(t, result, "sleep 5\necho \"[sui-dev] Node ready.\"")
}

func TestPatchEntrypointFaucetWait_Idempotent(t *testing.T) {
	content := `echo "[sui-dev] RPC responding, waiting for full initialization..."
sleep 5
echo "[sui-dev] Node ready."`

	first := patchEntrypointFaucetWait(content)
	second := patchEntrypointFaucetWait(first)
	assert.Equal(t, first, second)
}

func TestPatchEntrypointFaucetWait_NoOpWhenMarkerAbsent(t *testing.T) {
	// If upstream has already changed this section beyond recognition, the
	// patch must not corrupt the file — it should leave it untouched.
	content := `echo "[sui-dev] Something else entirely"`
	result := patchEntrypointFaucetWait(content)
	assert.Equal(t, content, result)
}

// ── patchEntrypointFaucetResilience ──────────────────────────────────

func TestPatchEntrypointFaucetResilience_ReplacesExitWithWarning(t *testing.T) {
	content := `for alias in ADMIN PLAYER_A PLAYER_B; do
  sui client switch --address "$alias"
  for attempt in 1 2 3; do
    sui client faucet 2>&1 && break
    [ "$attempt" -eq 3 ] && {
      echo "[sui-dev] Faucet failed for $alias" >&2
      exit 1
    }
    sleep 2
  done
done`

	result := patchEntrypointFaucetResilience(content)
	assert.Contains(t, result, "will not be funded automatically")
	assert.Contains(t, result, "efctl env faucet --address")
	// A single funding failure must no longer terminate the whole script.
	assert.NotContains(t, result, "exit 1")
}

func TestPatchEntrypointFaucetResilience_Idempotent(t *testing.T) {
	content := `    [ "$attempt" -eq 3 ] && {
      echo "[sui-dev] Faucet failed for $alias" >&2
      exit 1
    }`

	first := patchEntrypointFaucetResilience(content)
	second := patchEntrypointFaucetResilience(first)
	assert.Equal(t, first, second)
}

func TestPatchEntrypointFaucetResilience_NoOpWhenMarkerAbsent(t *testing.T) {
	content := `echo "[sui-dev] Something else entirely"`
	result := patchEntrypointFaucetResilience(content)
	assert.Equal(t, content, result)
}

// ── patchEntrypoint end-to-end: faucet patches applied together ─────

func TestPatchEntrypoint_FaucetPatchesAppliedTogether(t *testing.T) {
	tmpDir := t.TempDir()
	scriptsDir := filepath.Join(tmpDir, "scripts")
	os.MkdirAll(scriptsDir, 0755)
	entrypointPath := filepath.Join(scriptsDir, "entrypoint.sh")

	// Mirrors the real upstream builder-scaffold entrypoint.sh shape for the
	// node-start-through-funding section.
	content := `#!/usr/bin/env bash
set -e
SUI_CFG="${SUI_CONFIG_DIR:-/root/.sui}"
ENV_FILE="/workspace/builder-scaffold/docker/.env.sui"

# ---------- start local node ----------
sui start --with-faucet --force-regenesis &
NODE_PID=$!

echo "[sui-dev] Waiting for RPC on port 9000..."
for i in $(seq 1 30); do
  curl -s -o /dev/null http://127.0.0.1:9000 2>/dev/null && break
  if [ "$i" -eq 30 ]; then
    echo "[sui-dev] ERROR: RPC did not become ready" >&2
    exit 1
  fi
  sleep 1
done
echo "[sui-dev] RPC responding, waiting for full initialization..."
sleep 5
echo "[sui-dev] Node ready."

# ---------- fund accounts ----------
echo "[sui-dev] Funding accounts from faucet..."
for alias in ADMIN PLAYER_A PLAYER_B; do
  sui client switch --address "$alias"
  for attempt in 1 2 3; do
    sui client faucet 2>&1 && break
    [ "$attempt" -eq 3 ] && {
      echo "[sui-dev] Faucet failed for $alias" >&2
      exit 1
    }
    sleep 2
  done
done
`
	os.WriteFile(entrypointPath, []byte(content), 0755)

	patchEntrypoint(tmpDir)

	body, _ := os.ReadFile(entrypointPath)
	bodyStr := string(body)
	assert.Contains(t, bodyStr, "Waiting for faucet on port 9123", "faucet-wait patch should be applied")
	assert.Contains(t, bodyStr, "will not be funded automatically", "faucet-resilience patch should be applied")
	assert.NotContains(t, bodyStr, "exit 1\n    }", "faucet funding failure must no longer be fatal")
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
RUN sed -i 's|ENV_FILE="/workspace/builder-scaffold/docker/.env.sui"|ENV_FILE="/root/.sui/.env.sui"|' /workspace/scripts/entrypoint.sh
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

// ── Patch diagnostic tests (tasks 1.2 + 1.3) ────────────────────────

// fullDockerfile returns Dockerfile content whose source texts are all
// matched by the required patches.
func fullDockerfile() string {
	return `FROM ubuntu:24.04
RUN apt-get update && apt-get install -y --no-install-recommends \
    dos2unix \
    && apt-get clean
COPY scripts/ /workspace/scripts/
RUN dos2unix /workspace/scripts/*.sh && chmod +x /workspace/scripts/*.sh
ENV SUI_CONFIG_DIR=/root/.sui
&& sui --version
ENTRYPOINT ["/workspace/scripts/entrypoint.sh"]
`
}

// fullEntrypoint returns entrypoint content whose source texts are all
// matched by the required patches.
func fullEntrypoint() string {
	return `#!/usr/bin/env bash
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
    [ "$attempt" -eq 3 ] && {
      echo "[sui-dev] Faucet failed for $alias" >&2
      exit 1
    }
`
}

// --- 1.2 Unmatched patches emit warnings --------------------------------------

func TestPatchDockerfile_WarnsOnUnmatchedPostgresqlClient(t *testing.T) {
	tmpDir := t.TempDir()
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	// No "dos2unix \" backslash → postgresql-client source absent.
	content := `FROM ubuntu:24.04
RUN apt-get update
ENV SUI_CONFIG_DIR=/root/.sui
&& sui --version
RUN dos2unix /workspace/scripts/*.sh && chmod +x /workspace/scripts/*.sh
`
	os.WriteFile(dockerfilePath, []byte(content), 0644)

	buf := captureWarnings(t)
	patchDockerfile(tmpDir)

	assert.Contains(t, buf.String(), "postgresql-client")
	assert.Contains(t, buf.String(), "Dockerfile")
}

func TestPatchDockerfile_WarnsOnUnmatchedSuiConfigDir(t *testing.T) {
	tmpDir := t.TempDir()
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	content := `FROM ubuntu:24.04
RUN apt-get install -y dos2unix \
    && apt-get clean
RUN dos2unix /workspace/scripts/*.sh && chmod +x /workspace/scripts/*.sh
&& sui --version
`
	os.WriteFile(dockerfilePath, []byte(content), 0644)

	buf := captureWarnings(t)
	patchDockerfile(tmpDir)

	assert.Contains(t, buf.String(), "sui-config-dir")
	assert.Contains(t, buf.String(), "Dockerfile")
}

func TestPatchDockerfile_WarnsOnUnmatchedGlobalSui(t *testing.T) {
	tmpDir := t.TempDir()
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	// No "&& sui --version" source → globalSui cannot be injected.
	content := `FROM ubuntu:24.04
RUN apt-get install -y dos2unix \
    && apt-get clean
RUN dos2unix /workspace/scripts/*.sh && chmod +x /workspace/scripts/*.sh
ENV SUI_CONFIG_DIR=/root/.sui
`
	os.WriteFile(dockerfilePath, []byte(content), 0644)

	buf := captureWarnings(t)
	patchDockerfile(tmpDir)

	assert.Contains(t, buf.String(), "global-sui")
	assert.Contains(t, buf.String(), "Dockerfile")
}

func TestPatchDockerfile_WarnsOnUnmatchedSedSafetyNet(t *testing.T) {
	tmpDir := t.TempDir()
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	// No "RUN dos2unix …" source line → sed safety-net cannot be injected.
	content := `FROM ubuntu:24.04
COPY scripts/ /workspace/scripts/
RUN apt-get install -y dos2unix \
    && apt-get clean
ENV SUI_CONFIG_DIR=/root/.sui
&& sui --version
`
	os.WriteFile(dockerfilePath, []byte(content), 0644)

	buf := captureWarnings(t)
	patchDockerfile(tmpDir)

	assert.Contains(t, buf.String(), "sed-safety-net")
	assert.Contains(t, buf.String(), "Dockerfile")
}

func TestPatchDockerfile_NoWarningForLegacySedCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	// Full content but the old narrow sed line is absent — that should NOT
	// produce a warning because legacy cleanup is excluded from diagnostics.
	os.WriteFile(dockerfilePath, []byte(fullDockerfile()), 0644)

	buf := captureWarnings(t)
	patchDockerfile(tmpDir)

	// No warning about any legacy sed operation
	assert.NotContains(t, buf.String(), "legacy-sed")
	assert.NotContains(t, buf.String(), "old-narrow-sed")
}

func TestPatchEntrypointEnvPath_WarnsOnUnmatched(t *testing.T) {
	content := `SUI_CFG="/workspace/.sui"
ENV_FILE="/some/other/path"
`
	buf := captureWarnings(t)
	result := patchEntrypointEnvPath(content)

	// Content must remain unchanged.
	assert.Equal(t, content, result)
	assert.Contains(t, buf.String(), "env-file-path")
	assert.Contains(t, buf.String(), "scripts/entrypoint.sh")
}

func TestPatchEntrypointPostgresWait_WarnsOnUnmatched(t *testing.T) {
	// Content without "# ---------- start local node ----------" source
	// or "wait for postgres" marker.
	content := `#!/bin/bash
echo "completely different script"
`
	buf := captureWarnings(t)
	result := patchEntrypointPostgresWait(content)

	assert.Equal(t, content, result)
	assert.Contains(t, buf.String(), "postgres-wait")
	assert.Contains(t, buf.String(), "scripts/entrypoint.sh")
}

func TestPatchEntrypointSuiStart_WarnsOnUnmatched(t *testing.T) {
	// Content without "sui start --with-faucet --force-regenesis &" source
	// or "SUI_START_ARGS" marker.
	content := `#!/bin/bash
some_other_command &
`
	buf := captureWarnings(t)
	result := patchEntrypointSuiStart(content)

	assert.Equal(t, content, result)
	assert.Contains(t, buf.String(), "sui-start")
	assert.Contains(t, buf.String(), "scripts/entrypoint.sh")
}

func TestPatchEntrypointLoopTimings_WarnsOnUnmatchedOnce(t *testing.T) {
	// Content without any recognizable loop or RPC-ready patterns.
	content := `#!/bin/bash
for i in $(seq 1 10); do
  echo "different loop"
done
echo "done"
`
	buf := captureWarnings(t)
	result := patchEntrypointLoopTimings(content)

	assert.Equal(t, content, result)
	w := buf.String()
	assert.Contains(t, w, "loop-timings", "warning must identify the patch operation")
	// At most one warning per semantic patch attempt.
	assert.Equal(t, 1, strings.Count(w, "loop-timings"))
}

func TestPatchEntrypointFaucetWait_WarnsOnUnmatched(t *testing.T) {
	// Content without the sleep-5 + "RPC ready" source pattern or marker.
	content := `echo "[sui-dev] Something else entirely"`
	buf := captureWarnings(t)
	result := patchEntrypointFaucetWait(content)

	assert.Equal(t, content, result)
	assert.Contains(t, buf.String(), "faucet-wait")
	assert.Contains(t, buf.String(), "scripts/entrypoint.sh")
}

func TestPatchEntrypointFaucetResilience_WarnsOnUnmatched(t *testing.T) {
	content := `echo "[sui-dev] Something else entirely"`
	buf := captureWarnings(t)
	result := patchEntrypointFaucetResilience(content)

	assert.Equal(t, content, result)
	assert.Contains(t, buf.String(), "faucet-resilience")
	assert.Contains(t, buf.String(), "scripts/entrypoint.sh")
}

// --- 1.3 Successful and already-applied patches emit no warning -------------------

func TestPatchDockerfile_NoWarningOnSuccessfulPatch(t *testing.T) {
	tmpDir := t.TempDir()
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	os.WriteFile(dockerfilePath, []byte(fullDockerfile()), 0644)

	buf := captureWarnings(t)
	patchDockerfile(tmpDir)

	assert.Empty(t, buf.String(), "successful patches must not produce warnings")
}

func TestPatchDockerfile_NoWarningWhenAlreadyPatched(t *testing.T) {
	tmpDir := t.TempDir()
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")

	// First pass: apply patches normally.
	os.WriteFile(dockerfilePath, []byte(fullDockerfile()), 0644)
	patchDockerfile(tmpDir)

	// Second pass: everything already applied — no warnings.
	buf := captureWarnings(t)
	patchDockerfile(tmpDir)

	assert.Empty(t, buf.String(), "idempotent re-run must emit no warnings")
}

func TestPatchEntrypointEnvPath_NoWarningOnSuccess(t *testing.T) {
	content := `ENV_FILE="/workspace/builder-scaffold/docker/.env.sui"`
	buf := captureWarnings(t)
	result := patchEntrypointEnvPath(content)

	assert.Contains(t, result, `ENV_FILE="${SUI_CFG}/.env.sui"`)
	assert.Empty(t, buf.String(), "successful patch must not warn")
}

func TestPatchEntrypointEnvPath_NoWarningWhenAlreadyPatched(t *testing.T) {
	content := `ENV_FILE="${SUI_CFG}/.env.sui"`
	buf := captureWarnings(t)
	result := patchEntrypointEnvPath(content)

	assert.Equal(t, content, result)
	assert.Empty(t, buf.String(), "already-applied patch must not warn")
}

func TestPatchEntrypointPostgresWait_NoWarningWhenAlreadyPatched(t *testing.T) {
	content := fullEntrypoint()
	// Apply once
	result := patchEntrypointPostgresWait(content)
	assert.Contains(t, result, "wait for postgres")

	// Apply again — already patched
	buf := captureWarnings(t)
	result2 := patchEntrypointPostgresWait(result)
	assert.Equal(t, result, result2)
	assert.Empty(t, buf.String())
}

func TestPatchEntrypointSuiStart_NoWarningOnSuccess(t *testing.T) {
	content := fullEntrypoint()
	buf := captureWarnings(t)
	result := patchEntrypointSuiStart(content)

	assert.Contains(t, result, "SUI_START_ARGS")
	assert.Empty(t, buf.String())
}

func TestPatchEntrypointLoopTimings_NoWarningOnSuccess(t *testing.T) {
	content := fullEntrypoint()
	buf := captureWarnings(t)
	result := patchEntrypointLoopTimings(content)

	assert.Contains(t, result, "seq 1 60")
	assert.Empty(t, buf.String())
}

func TestPatchEntrypointFaucetWait_NoWarningOnSuccess(t *testing.T) {
	content := `echo "[sui-dev] RPC responding, waiting for full initialization..."
sleep 5
echo "[sui-dev] Node ready."`
	buf := captureWarnings(t)
	result := patchEntrypointFaucetWait(content)

	assert.Contains(t, result, "Waiting for faucet on port 9123")
	assert.Empty(t, buf.String())
}

func TestPatchEntrypointFaucetResilience_NoWarningOnSuccess(t *testing.T) {
	content := fullEntrypoint()
	buf := captureWarnings(t)
	result := patchEntrypointFaucetResilience(content)

	assert.Contains(t, result, "will not be funded automatically")
	assert.Empty(t, buf.String())
}

func TestPatchEntrypoint_FullRunWithNoWarnings(t *testing.T) {
	tmpDir := t.TempDir()
	dockerDir := tmpDir
	scriptsDir := filepath.Join(dockerDir, "scripts")
	os.MkdirAll(scriptsDir, 0755)

	dockerfilePath := filepath.Join(dockerDir, "Dockerfile")
	os.WriteFile(dockerfilePath, []byte(fullDockerfile()), 0644)

	entrypointPath := filepath.Join(scriptsDir, "entrypoint.sh")
	os.WriteFile(entrypointPath, []byte(fullEntrypoint()), 0755)

	buf := captureWarnings(t)
	patchDockerfile(dockerDir)
	patchEntrypoint(dockerDir)

	assert.Empty(t, buf.String(), "full successful patch run must produce no warnings")
}

func TestPatchEntrypoint_IdempotentRunWithNoWarnings(t *testing.T) {
	tmpDir := t.TempDir()
	dockerDir := tmpDir
	scriptsDir := filepath.Join(dockerDir, "scripts")
	os.MkdirAll(scriptsDir, 0755)

	dockerfilePath := filepath.Join(dockerDir, "Dockerfile")
	os.WriteFile(dockerfilePath, []byte(fullDockerfile()), 0644)

	entrypointPath := filepath.Join(scriptsDir, "entrypoint.sh")
	os.WriteFile(entrypointPath, []byte(fullEntrypoint()), 0755)

	// First run: apply all patches.
	patchDockerfile(dockerDir)
	patchEntrypoint(dockerDir)

	// Second run: everything already applied — no warnings.
	buf := captureWarnings(t)
	patchDockerfile(dockerDir)
	patchEntrypoint(dockerDir)

	assert.Empty(t, buf.String(), "idempotent re-run must produce no warnings")
}
