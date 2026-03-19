package setup

import (
	"context"
	"fmt"
	"os/exec"

	"efctl/pkg/container"
	"efctl/pkg/ui"
)

// DeployWorld deploys the world contracts, configures the state, and spawns the Smart Gate infrastructure
func DeployWorld(c container.ContainerClient, workspace string) error {
	ui.Info.Println("Deploying world contracts...")

	if !c.ContainerRunning(container.ContainerSuiPlayground) {
		lastLogs := c.ContainerLogs(container.ContainerSuiPlayground, 50)
		exitCode, exitErr := c.ContainerExitCode(container.ContainerSuiPlayground)

		// Log all containers for debugging if this fails
		ui.Warn.Println("Container not running, listing all containers for diagnostics:")
		debugCmd := exec.Command(c.GetEngine(), "ps", "-a") // #nosec G204
		debugOut, _ := debugCmd.CombinedOutput()
		fmt.Println(string(debugOut))

		return fmt.Errorf("sui-playground container is not running (ExitCode: %d, ExitErr: %v). Last 50 lines of logs:\n%s", exitCode, exitErr, lastLogs)
	}

	// 0. Ensure all scripts in the container have LF line endings.
	// This protects against Windows host-side drift (CRLF).
	if err := NormalizeContainerScripts(c, container.ContainerSuiPlayground); err != nil {
		ui.Warn.Println(fmt.Sprintf("Script normalization failed (continuing): %v", err))
	}

	// 0. Remove stale Move.lock files so the Sui CLI resolves framework
	//    dependencies from the installed binary instead of pinned git revisions
	//    that may no longer exist upstream.
	CleanStaleMoveLocks(workspace)

	// 1. Generate environment
	if err := c.Exec(context.Background(), container.ContainerSuiPlayground, []string{"/bin/bash", ScriptGenerateWorldEnv}); err != nil {
		// Log all containers for debugging if this fails
		ui.Warn.Println("Command failed, listing all containers for diagnostics:")
		debugCmd := exec.Command(c.GetEngine(), "ps", "-a") // #nosec G204
		debugOut, _ := debugCmd.CombinedOutput()
		fmt.Println(string(debugOut))

		return fmt.Errorf("failed to generate world env: %w", err)
	}
	ensureWorldSponsorAddresses(c, container.ContainerSuiPlayground)

	// 2. Install dependencies & deploy
	if err := c.Exec(context.Background(), container.ContainerSuiPlayground, []string{"/bin/bash", "-c", CmdDeployWorld}); err != nil {
		// Log all containers for debugging if this fails
		ui.Warn.Println("Command failed, listing all containers for diagnostics:")
		debugCmd := exec.Command(c.GetEngine(), "ps", "-a") // #nosec G204
		debugOut, _ := debugCmd.CombinedOutput()
		fmt.Println(string(debugOut))

		return fmt.Errorf("failed to deploy world: %w", err)
	}

	// 3. Fix dependency resolution
	// No longer renaming Pub.localnet.toml to Pub.testnet.toml as it misaligns with documentation.
	// We handle both names during publication detection instead.

	// 4. Configure World State
	if err := c.Exec(context.Background(), container.ContainerSuiPlayground, []string{"/bin/bash", "-c", CmdConfigureWorld}); err != nil {
		// Log all containers for debugging if this fails
		ui.Warn.Println("Command failed, listing all containers for diagnostics:")
		debugCmd := exec.Command(c.GetEngine(), "ps", "-a") // #nosec G204
		debugOut, _ := debugCmd.CombinedOutput()
		fmt.Println(string(debugOut))

		return fmt.Errorf("failed to configure world: %w", err)
	}

	// 5. Spawn Structures
	ui.Info.Println("Spawning game structures (Gates)...")
	if err := c.Exec(context.Background(), container.ContainerSuiPlayground, []string{"/bin/bash", "-c", CmdCreateTestResources}); err != nil {
		// Log all containers for debugging if this fails
		ui.Warn.Println("Command failed, listing all containers for diagnostics:")
		debugCmd := exec.Command(c.GetEngine(), "ps", "-a") // #nosec G204
		debugOut, _ := debugCmd.CombinedOutput()
		fmt.Println(string(debugOut))

		return fmt.Errorf("failed to create test resources: %w", err)
	}

	return nil
}

// NormalizeContainerScripts ensures all .sh files in the /workspace directory
// inside the container have LF line endings. This is a critical safety net
// for Windows users where bind-mounted scripts might drift to CRLF.
func NormalizeContainerScripts(c container.ContainerClient, containerName string) error {
	ui.Debug.Println(fmt.Sprintf("Normalizing script line endings in container %s...", containerName))

	// Find all .sh files and use dos2unix to convert CRLF→LF.
	// dos2unix is already installed in the container image and is a
	// no-op when files already have Unix line endings.
	cmd := []string{
		"/bin/bash", "-c",
		"find /workspace -type f \\( -name '*.sh' -o -name '.env*' \\) -exec dos2unix {} + 2>/dev/null || true",
	}

	return c.Exec(context.Background(), containerName, cmd)
}
