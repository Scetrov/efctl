package setup

import (
	"fmt"
	"os"
	"path/filepath"

	"efctl/pkg/container"
	"efctl/pkg/ui"
)

// DeployWorld deploys the world contracts, configures the state, and spawns the Smart Gate infrastructure
func DeployWorld(c container.ContainerClient, workspace string) error {
	ui.Info.Println("Deploying world contracts...")

	// 1. Generate environment
	if err := c.Exec(container.ContainerSuiPlayground, []string{"/bin/bash", ScriptGenerateWorldEnv}); err != nil {
		return fmt.Errorf("failed to generate world env: %w", err)
	}

	// 2. Install dependencies & deploy
	if err := c.Exec(container.ContainerSuiPlayground, []string{"/bin/bash", "-c", CmdDeployWorld}); err != nil {
		return fmt.Errorf("failed to deploy world: %w", err)
	}

	// 3. Fix dependency resolution
	pubLocalnet := filepath.Join(workspace, "world-contracts", "contracts", "world", FilePubLocalnetToml)
	pubTestnet := filepath.Join(workspace, "world-contracts", "contracts", "world", FilePubTestnetToml)
	if err := os.Rename(pubLocalnet, pubTestnet); err != nil {
		ui.Warn.Println(fmt.Sprintf("Could not rename %s (might already be renamed or missing): %v", FilePubLocalnetToml, err))
	}

	// 4. Configure World State
	if err := c.Exec(container.ContainerSuiPlayground, []string{"/bin/bash", "-c", CmdConfigureWorld}); err != nil {
		return fmt.Errorf("failed to configure world: %w", err)
	}

	// 5. Spawn Structures
	ui.Info.Println("Spawning game structures (Gates)...")
	if err := c.Exec(container.ContainerSuiPlayground, []string{"/bin/bash", "-c", CmdCreateTestResources}); err != nil {
		return fmt.Errorf("failed to create test resources: %w", err)
	}

	return nil
}
