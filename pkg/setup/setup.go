package setup

import (
	"path/filepath"
	"strings"

	"efctl/pkg/config"
	"efctl/pkg/git"
	"efctl/pkg/ui"

	"github.com/pterm/pterm"
)

func extractRepoName(url string) string {
	repo := strings.TrimPrefix(url, "https://")
	repo = strings.TrimPrefix(repo, "git@")
	repo = strings.TrimSuffix(repo, ".git")
	if idx := strings.Index(repo, "github.com/"); idx != -1 {
		repo = repo[idx+len("github.com/"):]
	}
	return repo
}

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
	ui.Info.Printfln("Setting up %s using branch %s", pterm.Bold.Sprint(extractRepoName(worldContractsUrl)), pterm.Bold.Sprint(worldContractsBranch))
	if err := g.CloneRepository(worldContractsUrl, worldContractsPath); err != nil {
		return err
	}
	if err := g.CheckoutBranch(worldContractsPath, worldContractsBranch); err != nil {
		return err
	}

	builderScaffoldUrl := cfg.GetBuilderScaffoldURL()
	builderScaffoldBranch := cfg.GetBuilderScaffoldBranch()
	builderScaffoldPath := filepath.Join(workspace, "builder-scaffold")
	ui.Info.Printfln("Setting up %s using branch %s", pterm.Bold.Sprint(extractRepoName(builderScaffoldUrl)), pterm.Bold.Sprint(builderScaffoldBranch))
	if err := g.CloneRepository(builderScaffoldUrl, builderScaffoldPath); err != nil {
		return err
	}
	if err := g.CheckoutBranch(builderScaffoldPath, builderScaffoldBranch); err != nil {
		return err
	}

	return nil
}
