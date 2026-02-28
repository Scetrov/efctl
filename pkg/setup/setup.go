package setup

import (
	"path/filepath"

	"efctl/pkg/config"
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

	cfg := config.Loaded
	worldContractsUrl := cfg.GetWorldContractsURL()
	worldContractsBranch := cfg.GetWorldContractsBranch()
	worldContractsPath := filepath.Join(workspace, "world-contracts")
	if err := git.CloneRepository(worldContractsUrl, worldContractsPath); err != nil {
		return err
	}
	if err := git.CheckoutBranch(worldContractsPath, worldContractsBranch); err != nil {
		return err
	}

	builderScaffoldUrl := cfg.GetBuilderScaffoldURL()
	builderScaffoldBranch := cfg.GetBuilderScaffoldBranch()
	builderScaffoldPath := filepath.Join(workspace, "builder-scaffold")
	if err := git.CloneRepository(builderScaffoldUrl, builderScaffoldPath); err != nil {
		return err
	}
	if err := git.CheckoutBranch(builderScaffoldPath, builderScaffoldBranch); err != nil {
		return err
	}

	return nil
}
