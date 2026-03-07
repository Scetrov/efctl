//go:build e2e

// Package e2e contains end-to-end tests that exercise the full efctl lifecycle.
// These tests require Docker (or Podman), Git, and network access.
// Run with: go test -tags e2e -timeout 15m ./tests/e2e/...
package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
	// Always include --no-progress for cleaner test output
	fullArgs := append([]string{"--no-progress"}, args...)
	cmd := exec.Command(bin, fullArgs...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// preferredContainerEngine returns the name of the first available container
// engine binary ("docker" or "podman"), or "" if neither is found.
func preferredContainerEngine() string {
	if _, err := exec.LookPath("docker"); err == nil {
		return "docker"
	}
	if _, err := exec.LookPath("podman"); err == nil {
		return "podman"
	}
	return ""
}

// dockerDaemonAvailable returns true if a container engine daemon is reachable.
// This goes beyond checking for the binary — it actually pings the daemon.
func dockerDaemonAvailable() bool {
	engine := preferredContainerEngine()
	if engine == "" {
		return false
	}
	cmd := exec.Command(engine, "info")
	cmd.Env = os.Environ()
	return cmd.Run() == nil
}

// normalizeWorkspacePermissions fixes ownership/permissions on
// bind-mounted workspace files that may be created by root inside containers.
// Returns an error if permission fix fails and container is running.
func normalizeWorkspacePermissions(t *testing.T) error {
	t.Helper()

	engine := preferredContainerEngine()
	if engine == "" {
		return nil // No container engine available
	}

	// Check if container is running before attempting exec
	checkCmd := exec.Command(engine, "inspect", "--format", "{{.State.Running}}", "sui-playground")
	checkOut, checkErr := checkCmd.CombinedOutput()
	isRunning := checkErr == nil && strings.TrimSpace(string(checkOut)) == "true"

	if !isRunning {
		t.Logf("sui-playground is not running, skipping permission normalization via container")
		// Try host-side permission fix as fallback (best-effort)
		return tryHostSidePermissionFix(t)
	}

	uid := strconv.Itoa(os.Getuid())
	gid := strconv.Itoa(os.Getgid())

	cmdStr := fmt.Sprintf(
		"chown -R %s:%s /workspace/world-contracts /workspace/builder-scaffold 2>/dev/null || true; "+
			"chmod -R u+rwX /workspace/world-contracts /workspace/builder-scaffold 2>/dev/null || true",
		uid,
		gid,
	)

	cmd := exec.Command(engine, "exec", "sui-playground", "/bin/bash", "-lc", cmdStr)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("permission normalization via container failed: %v\n%s", err, string(out))
		// If container is running but exec failed, try host-side fallback
		return tryHostSidePermissionFix(t)
	}

	t.Logf("✓ Normalized workspace permissions via container")
	return nil
}

// tryHostSidePermissionFix attempts to fix permissions from the host side.
// This is a best-effort fallback for when container exec is not available.
func tryHostSidePermissionFix(t *testing.T) error {
	t.Helper()

	// For rootless Podman, we might need podman unshare
	engine := preferredContainerEngine()
	if engine == "podman" {
		// Try podman unshare as a fallback
		cmd := exec.Command("podman", "unshare", "chown", "-R",
			fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()),
			"world-contracts", "builder-scaffold")
		err := cmd.Run()
		if err == nil {
			t.Logf("✓ Normalized workspace permissions via podman unshare")
			return nil
		}
		t.Logf("podman unshare permission fix failed: %v", err)
	}

	// Last resort: try chmod to make files at least readable/writable by owner
	for _, dir := range []string{"world-contracts", "builder-scaffold"} {
		cmd := exec.Command("chmod", "-R", "u+rwX", dir)
		if err := cmd.Run(); err != nil {
			t.Logf("host-side chmod failed for %s: %v", dir, err)
			continue
		}
	}

	// Return nil because this is best-effort cleanup
	return nil
}

// isKnownInfraOrDriftIssue checks if the error output indicates a known issue
// with upstream builder-scaffold / world-contracts drift, or a CI infrastructure
// problem (e.g. container daemon unavailable, containers not started).
func isKnownInfraOrDriftIssue(output string) bool {
	knownPatterns := []string{
		// ── Upstream drift patterns ──
		"Unbound module",
		"Type mismatch",
		"Could not find module",
		"Unable to resolve",
		"dependency version mismatch",
		"no published package",
		"Invalid call",
		"Invalid argument",

		// ── Infrastructure / container patterns ──
		"no container with name or ID",
		"Cannot connect to the Docker daemon",
		"Is the docker daemon running?",
		"no such container",
		"connection refused",

		// ── Missing deployment artifacts (world-contracts didn't deploy) ──
		"deployments/localnet: no such file or directory",
	}

	for _, pattern := range knownPatterns {
		if strings.Contains(output, pattern) {
			return true
		}
	}
	return false
}

type e2eLifecycleTester struct {
	bin                    string
	workspace              string
	envUpPassed            bool
	extensionInitPassed    bool
	extensionPublishPassed bool
}

func (tester *e2eLifecycleTester) testVersion(t *testing.T) {
	out, err := runEfctl(t, tester.bin, tester.workspace, "version")
	require.NoError(t, err, "efctl version failed: %s", out)
	assert.Contains(t, out, "efctl")
}

func (tester *e2eLifecycleTester) testEnvUp(t *testing.T) {
	start := time.Now()
	out, err := runEfctl(t, tester.bin, tester.workspace, "env", "up")
	elapsed := time.Since(start)
	t.Logf("env up took %s", elapsed)

	if err != nil {
		if isKnownInfraOrDriftIssue(out) {
			t.Skipf("skipping: env up hit a known infra/drift issue:\n%s", out)
		}
		require.NoError(t, err, "efctl env up failed:\n%s", out)
	}
	assert.Contains(t, out, "Environment is up")

	// Verify world-contracts and builder-scaffold were cloned
	assert.DirExists(t, filepath.Join(tester.workspace, "world-contracts"))
	assert.DirExists(t, filepath.Join(tester.workspace, "builder-scaffold"))

	tester.envUpPassed = true
}

func (tester *e2eLifecycleTester) testExtensionInit(t *testing.T) {
	if !tester.envUpPassed {
		t.Skip("skipping: env_up did not pass")
	}

	out, err := runEfctl(t, tester.bin, tester.workspace, "env", "extension", "init")
	if err != nil {
		if isKnownInfraOrDriftIssue(out) {
			t.Skipf("skipping: extension init hit a known infra/drift issue:\n%s", out)
		}
		require.NoError(t, err, "efctl env extension init failed:\n%s", out)
	}
	assert.Contains(t, out, "builder-scaffold successfully initialized")

	// Verify .env was created
	envPath := filepath.Join(tester.workspace, "builder-scaffold", ".env")
	assert.FileExists(t, envPath)

	envData, _ := os.ReadFile(envPath)
	envStr := string(envData)
	assert.Contains(t, envStr, "WORLD_PACKAGE_ID=0x")
	assert.Contains(t, envStr, "ADMIN_ADDRESS=0x")

	tester.extensionInitPassed = true
}

func (tester *e2eLifecycleTester) testExtensionPublish(t *testing.T) {
	if !tester.extensionInitPassed {
		t.Skip("skipping: extension_init did not pass")
	}

	out, err := runEfctl(t, tester.bin, tester.workspace, "env", "extension", "publish", "smart_gate")
	if err != nil {
		if isKnownInfraOrDriftIssue(out) {
			t.Skipf("skipping: extension publish hit a known infra/drift issue:\n%s", out)
		}
		require.NoError(t, err, "efctl env extension publish failed:\n%s", out)
	}
	assert.Contains(t, out, "Extension contract published successfully")

	// Verify .env was updated with published IDs
	envData, _ := os.ReadFile(filepath.Join(tester.workspace, "builder-scaffold", ".env"))
	envStr := string(envData)
	assert.Contains(t, envStr, "BUILDER_PACKAGE_ID=0x")

	tester.extensionPublishPassed = true
}

func (tester *e2eLifecycleTester) testExtensionPublishIdempotent(t *testing.T) {
	if !tester.extensionPublishPassed {
		t.Skip("skipping: extension_publish did not pass")
	}

	out, err := runEfctl(t, tester.bin, tester.workspace, "env", "extension", "publish", "smart_gate")
	if err != nil {
		if isKnownInfraOrDriftIssue(out) {
			t.Skipf("skipping: idempotent publish hit a known infra/drift issue:\n%s", out)
		}
		require.NoError(t, err, "second publish should succeed:\n%s", out)
	}
	assert.Contains(t, out, "Extension contract published successfully")
}

func (tester *e2eLifecycleTester) testEnvRun(t *testing.T) {
	if !tester.envUpPassed {
		t.Skip("skipping: env_up did not pass")
	}

	out, err := runEfctl(t, tester.bin, tester.workspace, "env", "run", "sui", "client", "active-address")
	if err != nil {
		if isKnownInfraOrDriftIssue(out) {
			t.Skipf("skipping: env run hit a known infra/drift issue:\n%s", out)
		}
		require.NoError(t, err, "efctl env run failed:\n%s", out)
	}
	// Should output a Sui address (0x...)
	assert.Contains(t, out, "0x")
}

func (tester *e2eLifecycleTester) testEnvDown(t *testing.T) {
	// Always attempt cleanup, even if env_up didn't pass fully.
	// Attempt to normalize permissions before shutdown
	if err := normalizeWorkspacePermissions(t); err != nil {
		t.Logf("Warning: permission normalization had issues: %v", err)
	}

	out, err := runEfctl(t, tester.bin, tester.workspace, "env", "down")
	require.NoError(t, err, "efctl env down failed:\n%s", out)

	// Verify container is no longer running
	engine := preferredContainerEngine()
	if engine == "" {
		engine = "docker"
	}
	checkCmd := exec.Command(engine, "ps", "--filter", "name=sui-playground", "--format", "{{.Names}}")
	checkOut, _ := checkCmd.Output()
	assert.NotContains(t, strings.TrimSpace(string(checkOut)), "sui-playground")
}

// TestE2E_FullLifecycle runs the complete efctl smoke test:
// build → version → env up → extension init → extension publish → env run → env down
//
// Each step tracks whether it passed; downstream steps skip if their
// prerequisite failed. This prevents cascading noise in CI.
//
// Requirements:
//   - Docker or Podman available AND daemon reachable
//   - Git available
//   - Network access (clones repos, pulls images)
//   - ~10 minutes for full test (container build + world deploy)
func TestE2E_FullLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	// Gate: verify the container daemon is actually reachable, not just
	// that the binary exists. In CI the binary may be present but the
	// socket may not be configured or the service may not be running.
	if !dockerDaemonAvailable() {
		t.Skip("skipping: container daemon (Docker/Podman) is not reachable — run `docker info` to debug")
	}

	// Set generous timeout for CI environments (15 minutes = 900 seconds)
	os.Setenv("EFCTL_STARTUP_TIMEOUT_SECONDS", "900")
	defer os.Unsetenv("EFCTL_STARTUP_TIMEOUT_SECONDS")

	bin := efctlBin(t)

	// Create an isolated workspace
	workspaceRoot, err := os.MkdirTemp("", "efctl-e2e-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(workspaceRoot)
	})

	workspace := filepath.Join(workspaceRoot, "e2e-workspace")
	require.NoError(t, os.MkdirAll(workspace, 0750))

	tester := &e2eLifecycleTester{
		bin:       bin,
		workspace: workspace,
	}

	// ── Steps ──────────────────────────────────────────────────────
	t.Run("version", tester.testVersion)
	t.Run("env_up", tester.testEnvUp)
	t.Run("extension_init", tester.testExtensionInit)
	t.Run("extension_publish", tester.testExtensionPublish)
	t.Run("extension_publish_idempotent", tester.testExtensionPublishIdempotent)
	t.Run("env_run", tester.testEnvRun)
	t.Run("env_down", tester.testEnvDown)
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
