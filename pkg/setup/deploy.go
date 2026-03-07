package setup

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"efctl/pkg/container"
	"efctl/pkg/ui"
)

// DeployWorld deploys the world contracts, configures the state, and spawns the Smart Gate infrastructure
func DeployWorld(c container.ContainerClient, workspace string) error {
	ui.Info.Println("Deploying world contracts...")

	if !c.ContainerRunning(container.ContainerSuiPlayground) {
		lastLogs := c.ContainerLogs(container.ContainerSuiPlayground, 50)
		exitCode, exitErr := c.ContainerExitCode(container.ContainerSuiPlayground)
		return fmt.Errorf("sui-playground container is not running (ExitCode: %d, ExitErr: %v). Last 50 lines of logs:\n%s", exitCode, exitErr, lastLogs)
	}

	// 0. Remove stale Move.lock files so the Sui CLI resolves framework
	//    dependencies from the installed binary instead of pinned git revisions
	//    that may no longer exist upstream.
	CleanStaleMoveLocks(workspace)

	// 1. Generate environment
	if err := c.Exec(context.Background(), container.ContainerSuiPlayground, []string{"/bin/bash", ScriptGenerateWorldEnv}); err != nil {
		// Log all containers for debugging if this fails
		ui.Debug.Println("Command failed, listing all containers for diagnostics:")
		debugCmd := exec.Command(c.GetEngine(), "ps", "-a") // #nosec G204
		debugOut, _ := debugCmd.CombinedOutput()
		ui.Debug.Println(string(debugOut))

		return fmt.Errorf("failed to generate world env: %w", err)
	}
	ensureWorldSponsorAddresses(c, container.ContainerSuiPlayground)

	// 2. Install dependencies & deploy
	if err := c.Exec(context.Background(), container.ContainerSuiPlayground, []string{"/bin/bash", "-c", CmdDeployWorld}); err != nil {
		return fmt.Errorf("failed to deploy world: %w", err)
	}

	// 3. Fix dependency resolution
	pubLocalnet := filepath.Join(workspace, "world-contracts", "contracts", "world", FilePubLocalnetToml)
	pubTestnet := filepath.Join(workspace, "world-contracts", "contracts", "world", FilePubTestnetToml)
	if err := os.Rename(pubLocalnet, pubTestnet); err != nil {
		ui.Warn.Println(fmt.Sprintf("Could not rename %s (might already be renamed or missing): %v", FilePubLocalnetToml, err))
	}

	// 4. Configure World State
	if err := c.Exec(context.Background(), container.ContainerSuiPlayground, []string{"/bin/bash", "-c", CmdConfigureWorld}); err != nil {
		return fmt.Errorf("failed to configure world: %w", err)
	}

	// 5. Spawn Structures
	ui.Info.Println("Spawning game structures (Gates)...")
	if err := c.Exec(context.Background(), container.ContainerSuiPlayground, []string{"/bin/bash", "-c", CmdCreateTestResources}); err != nil {
		return fmt.Errorf("failed to create test resources: %w", err)
	}

	return nil
}
