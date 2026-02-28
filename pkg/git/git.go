package git

import (
	"fmt"
	"os"
	"os/exec"

	"efctl/pkg/ui"
)

// GitClient defines the interface for git operations.
// Consumers should accept this interface to enable testing with mocks.
type GitClient interface {
	CloneRepository(url string, dest string) error
	CheckoutBranch(repoPath string, branch string) error
	SetupWorkDir(path string) error
}

// DefaultClient is the real git implementation.
type DefaultClient struct{}

// Compile-time check that DefaultClient implements GitClient.
var _ GitClient = (*DefaultClient)(nil)

// NewClient returns a new default git client.
func NewClient() *DefaultClient {
	return &DefaultClient{}
}

// CloneRepository clones a git repository to a specific path
func (g *DefaultClient) CloneRepository(url string, dest string) error {
	return CloneRepository(url, dest)
}

// CheckoutBranch checks out the specified branch in the given repository path.
func (g *DefaultClient) CheckoutBranch(repoPath string, branch string) error {
	return CheckoutBranch(repoPath, branch)
}

// SetupWorkDir creates the workspace directory if it doesn't exist
func (g *DefaultClient) SetupWorkDir(path string) error {
	return SetupWorkDir(path)
}

// CloneRepository clones a git repository to a specific path
func CloneRepository(url string, dest string) error {
	// Check if directory already exists
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		return nil
	}

	spinner, _ := ui.Spin(fmt.Sprintf("%s Cloning %s...", ui.GitEmoji, url))

	cmd := exec.Command("git", "clone", url, dest) // #nosec G204
	output, err := cmd.CombinedOutput()
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to clone %s", url))
		return fmt.Errorf("git clone error: %v\n%s", err, string(output))
	}

	spinner.Success(fmt.Sprintf("Cloned %s", dest))
	return nil
}

// CheckoutBranch checks out the specified branch in the given repository path.
func CheckoutBranch(repoPath string, branch string) error {
	spinner, _ := ui.Spin(fmt.Sprintf("%s Checking out branch '%s' in %s...", ui.GitEmoji, branch, repoPath))

	cmd := exec.Command("git", "-C", repoPath, "checkout", branch) // #nosec G204
	output, err := cmd.CombinedOutput()
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to checkout branch '%s'", branch))
		return fmt.Errorf("git checkout error: %v\n%s", err, string(output))
	}

	spinner.Success(fmt.Sprintf("Checked out branch '%s' in %s", branch, repoPath))
	return nil
}

// SetupWorkDir creates the workspace directory if it doesn't exist
func SetupWorkDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, 0750)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %w", path, err)
		}
	}
	return nil
}
