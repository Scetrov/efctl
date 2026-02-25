package setup

import (
	"path/filepath"

	"efctl/pkg/git"
	"efctl/pkg/ui"
)

// CloneRepositories prepares the workspace and clones required repositories
func CloneRepositories(workspace string) error {
	ui.Info.Println("Setting up workspace in " + workspace)

	err := git.SetupWorkDir(workspace)
	if err != nil {
		return err
	}

	worldContractsUrl := "https://github.com/evefrontier/world-contracts.git"
	worldContractsPath := filepath.Join(workspace, "world-contracts")
	if err := git.CloneRepository(worldContractsUrl, worldContractsPath); err != nil {
		return err
	}

	builderScaffoldUrl := "https://github.com/evefrontier/builder-scaffold.git"
	builderScaffoldPath := filepath.Join(workspace, "builder-scaffold")
	if err := git.CloneRepository(builderScaffoldUrl, builderScaffoldPath); err != nil {
		return err
	}

	return nil
}
