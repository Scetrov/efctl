package setup

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmdDeployWorld_IncludesDiagnosticsAndFallback(t *testing.T) {
	assert.Contains(t, CmdDeployWorld, "sui-playground")
	assert.Contains(t, CmdDeployWorld, "/etc/os-release")
	assert.Contains(t, CmdDeployWorld, "uname -m")
	assert.Contains(t, CmdDeployWorld, "command -v pnpm")
	assert.Contains(t, CmdDeployWorld, "packageManager")
	assert.Contains(t, CmdDeployWorld, "pnpm --version")
	assert.Contains(t, CmdDeployWorld, "pnpm-workspace.yaml")
	assert.Contains(t, CmdDeployWorld, "pnpm approve-builds esbuild")
	assert.Contains(t, CmdDeployWorld, "pnpm install --prefer-offline")
	assert.Contains(t, CmdDeployWorld, "pnpm deploy-world")
}

func TestCmdDeployWorld_KeepsPnpmVersionFailureVisibleAndBlocking(t *testing.T) {
	assert.NotContains(t, CmdDeployWorld, "pnpm --version 2>/dev/null")
	assert.NotContains(t, CmdDeployWorld, "pnpm --version || true")
	assert.Less(t, strings.Index(CmdDeployWorld, "pnpm --version"), strings.Index(CmdDeployWorld, "pnpm install --prefer-offline"))
}

func TestCmdDeployWorld_ValidShellSyntax(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}

	cmd := exec.Command("bash", "-n", "-c", CmdDeployWorld)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "CmdDeployWorld is not valid shell syntax: %s", strings.TrimSpace(string(out)))
}
