//go:build e2e

// Package e2e contains end-to-end tests that exercise the full efctl lifecycle.
// These tests require Docker (or Podman), Git, and network access.
// Run with: go test -tags e2e -timeout 15m ./tests/e2e/...
package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// efctlBin returns the absolute path to the compiled efctl binary.
// Set EFCTL_BINARY env var to override. Otherwise it builds from source.
func efctlBin(t *testing.T) string {
	t.Helper()
	if bin := os.Getenv("EFCTL_BINARY"); bin != "" {
		return bin
	}

	// Build the binary
	binPath := filepath.Join(t.TempDir(), "efctl")
	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = projectRoot(t)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to build efctl: %s", string(out))
	return binPath
}

// projectRoot returns the root of the efctl project.
func projectRoot(t *testing.T) string {
	t.Helper()
	// Walk up from this test file to find go.mod
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (go.mod)")
		}
		dir = parent
	}
}

// runEfctl runs efctl with the given args in the given workspace directory.
// Returns stdout+stderr combined, and any error.
func runEfctl(t *testing.T, bin, workDir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// TestE2E_FullLifecycle runs the complete efctl smoke test:
// build → version → env up → extension init → extension publish → env run → env down
//
// Requirements:
//   - Docker or Podman available
//   - Git available
//   - Network access (clones repos, pulls images)
//   - ~10 minutes for full test (container build + world deploy)
func TestE2E_FullLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	bin := efctlBin(t)

	// Create an isolated workspace
	workspace := filepath.Join(t.TempDir(), "e2e-workspace")
	require.NoError(t, os.MkdirAll(workspace, 0750))

	// ── Step 1: version ────────────────────────────────────────
	t.Run("version", func(t *testing.T) {
		out, err := runEfctl(t, bin, workspace, "version")
		require.NoError(t, err, "efctl version failed: %s", out)
		assert.Contains(t, out, "efctl")
	})

	// ── Step 2: env up ─────────────────────────────────────────
	t.Run("env_up", func(t *testing.T) {
		start := time.Now()
		out, err := runEfctl(t, bin, workspace, "env", "up")
		elapsed := time.Since(start)
		t.Logf("env up took %s", elapsed)

		require.NoError(t, err, "efctl env up failed:\n%s", out)
		assert.Contains(t, out, "Environment is up")

		// Verify world-contracts and builder-scaffold were cloned
		assert.DirExists(t, filepath.Join(workspace, "world-contracts"))
		assert.DirExists(t, filepath.Join(workspace, "builder-scaffold"))
	})

	// ── Step 3: extension init ─────────────────────────────────
	t.Run("extension_init", func(t *testing.T) {
		out, err := runEfctl(t, bin, workspace, "env", "extension", "init")
		require.NoError(t, err, "efctl env extension init failed:\n%s", out)
		assert.Contains(t, out, "builder-scaffold successfully initialized")

		// Verify .env was created
		envPath := filepath.Join(workspace, "builder-scaffold", ".env")
		assert.FileExists(t, envPath)

		envData, _ := os.ReadFile(envPath)
		envStr := string(envData)
		assert.Contains(t, envStr, "WORLD_PACKAGE_ID=0x")
		assert.Contains(t, envStr, "ADMIN_ADDRESS=0x")
	})

	// ── Step 4: extension publish ──────────────────────────────
	t.Run("extension_publish", func(t *testing.T) {
		out, err := runEfctl(t, bin, workspace, "env", "extension", "publish", "smart_gate")
		require.NoError(t, err, "efctl env extension publish failed:\n%s", out)
		assert.Contains(t, out, "Extension contract published successfully")

		// Verify .env was updated with published IDs
		envData, _ := os.ReadFile(filepath.Join(workspace, "builder-scaffold", ".env"))
		envStr := string(envData)
		assert.Contains(t, envStr, "BUILDER_PACKAGE_ID=0x")
	})

	// ── Step 5: extension publish idempotency ──────────────────
	t.Run("extension_publish_idempotent", func(t *testing.T) {
		out, err := runEfctl(t, bin, workspace, "env", "extension", "publish", "smart_gate")
		require.NoError(t, err, "second publish should succeed:\n%s", out)
		assert.Contains(t, out, "Extension contract published successfully")
	})

	// ── Step 6: env run ────────────────────────────────────────
	t.Run("env_run", func(t *testing.T) {
		out, err := runEfctl(t, bin, workspace, "env", "run", "sui", "client", "active-address")
		require.NoError(t, err, "efctl env run failed:\n%s", out)
		// Should output a Sui address (0x...)
		assert.Contains(t, out, "0x")
	})

	// ── Step 7: env down ───────────────────────────────────────
	t.Run("env_down", func(t *testing.T) {
		out, err := runEfctl(t, bin, workspace, "env", "down")
		require.NoError(t, err, "efctl env down failed:\n%s", out)

		// Verify container is no longer running
		checkCmd := exec.Command("docker", "ps", "--filter", "name=sui-playground", "--format", "{{.Names}}")
		checkOut, _ := checkCmd.Output()
		assert.NotContains(t, strings.TrimSpace(string(checkOut)), "sui-playground")
	})
}

// TestE2E_VersionOnly is a minimal E2E test that validates the binary builds and runs.
// This can run without Docker/Podman.
func TestE2E_VersionOnly(t *testing.T) {
	bin := efctlBin(t)
	out, err := runEfctl(t, bin, t.TempDir(), "version")
	require.NoError(t, err)
	assert.Contains(t, out, "efctl")
}

// TestE2E_InvalidCommand validates that unknown commands produce a helpful error.
func TestE2E_InvalidCommand(t *testing.T) {
	bin := efctlBin(t)
	out, err := runEfctl(t, bin, t.TempDir(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, out, "unknown command")
}

// TestE2E_EnvUpWithoutPrereqs validates error handling when Docker is not available.
// This test only runs if Docker/Podman is genuinely unavailable.
func TestE2E_EnvUpWithoutPrereqs(t *testing.T) {
	// Only run if Docker is NOT available
	if _, err := exec.LookPath("docker"); err == nil {
		t.Skip("Docker is available — skipping prerequisite failure test")
	}
	if _, err := exec.LookPath("podman"); err == nil {
		t.Skip("Podman is available — skipping prerequisite failure test")
	}

	bin := efctlBin(t)
	ws := filepath.Join(t.TempDir(), "no-docker")
	os.MkdirAll(ws, 0750)

	_, err := runEfctl(t, bin, ws, "env", "up")
	assert.Error(t, err, "env up should fail without Docker/Podman")
}
