package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"efctl/pkg/config"
	"efctl/pkg/git"
	"efctl/pkg/ui"

	"github.com/pterm/pterm"
)

func resolveWorkspacePath(workspace string) (string, error) {
	workspaceAbs, err := filepath.Abs(filepath.Clean(workspace))
	if err != nil {
		return "", fmt.Errorf("failed to resolve workspace path %s: %w", workspace, err)
	}

	if resolved, err := filepath.EvalSymlinks(workspaceAbs); err == nil {
		workspaceAbs = resolved
	}

	return workspaceAbs, nil
}

func resolveRepoPath(workspace string, repoName string) (string, error) {
	if repoName == "" || repoName != filepath.Base(repoName) {
		return "", fmt.Errorf("invalid repository directory name %q", repoName)
	}

	workspaceAbs, err := resolveWorkspacePath(workspace)
	if err != nil {
		return "", err
	}

	repoPath := filepath.Join(workspaceAbs, repoName)
	repoAbs, err := filepath.Abs(filepath.Clean(repoPath))
	if err != nil {
		return "", fmt.Errorf("failed to resolve repository path %s: %w", repoPath, err)
	}

	if resolved, err := filepath.EvalSymlinks(repoAbs); err == nil {
		repoAbs = resolved
	}

	relPath, err := filepath.Rel(workspaceAbs, repoAbs)
	if err != nil {
		return "", fmt.Errorf("failed to validate repository path %s: %w", repoAbs, err)
	}

	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("repository path %s escapes workspace %s", repoAbs, workspaceAbs)
	}

	return repoAbs, nil
}

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
	workspacePath, err := resolveWorkspacePath(workspace)
	if err != nil {
		return err
	}

	ui.Info.Println("Setting up workspace in " + workspacePath)

	err = g.SetupWorkDir(workspacePath)
	if err != nil {
		return err
	}

	cfg := config.Loaded
	worldContractsUrl := cfg.GetWorldContractsURL()
	worldContractsBranch := cfg.GetWorldContractsBranch()
	worldContractsPath, err := resolveRepoPath(workspacePath, "world-contracts")
	if err != nil {
		return err
	}
	ui.Info.Printfln("Setting up %s using branch %s", pterm.Bold.Sprint(extractRepoName(worldContractsUrl)), pterm.Bold.Sprint(worldContractsBranch))
	if err := g.CloneRepository(worldContractsUrl, worldContractsPath); err != nil {
		return err
	}
	if err := g.CheckoutBranch(worldContractsPath, worldContractsBranch); err != nil {
		return err
	}

	builderScaffoldUrl := cfg.GetBuilderScaffoldURL()
	builderScaffoldBranch := cfg.GetBuilderScaffoldBranch()
	builderScaffoldPath, err := resolveRepoPath(workspacePath, "builder-scaffold")
	if err != nil {
		return err
	}
	ui.Info.Printfln("Setting up %s using branch %s", pterm.Bold.Sprint(extractRepoName(builderScaffoldUrl)), pterm.Bold.Sprint(builderScaffoldBranch))
	if err := g.CloneRepository(builderScaffoldUrl, builderScaffoldPath); err != nil {
		return err
	}
	if err := g.CheckoutBranch(builderScaffoldPath, builderScaffoldBranch); err != nil {
		return err
	}

	return nil
}
