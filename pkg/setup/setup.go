package setup

import (
	"path/filepath"

	"efctl/pkg/config"
	"efctl/pkg/git"
	"efctl/pkg/ui"
)

// CloneRepositories prepares the workspace and clones required repositories
func CloneRepositories(g git.GitClient, workspace string) error {
	ui.Info.Println("Setting up workspace in " + workspace)

	err := g.SetupWorkDir(workspace)
	if err != nil {
		return err
	}

	cfg := config.Loaded
	worldContractsUrl := cfg.GetWorldContractsURL()
	worldContractsBranch := cfg.GetWorldContractsBranch()
	worldContractsPath := filepath.Join(workspace, "world-contracts")
	if err := g.CloneRepository(worldContractsUrl, worldContractsPath); err != nil {
		return err
	}
	if err := g.CheckoutBranch(worldContractsPath, worldContractsBranch); err != nil {
		return err
	}

	builderScaffoldUrl := cfg.GetBuilderScaffoldURL()
	builderScaffoldBranch := cfg.GetBuilderScaffoldBranch()
	builderScaffoldPath := filepath.Join(workspace, "builder-scaffold")
	if err := g.CloneRepository(builderScaffoldUrl, builderScaffoldPath); err != nil {
		return err
	}
	if err := g.CheckoutBranch(builderScaffoldPath, builderScaffoldBranch); err != nil {
		return err
	}

	return nil
}
