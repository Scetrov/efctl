package git

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"efctl/pkg/config"
	"efctl/pkg/ui"
)

func ensureGitRepository(path string) error {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--is-inside-work-tree") // #nosec G204 -- "git" is a hardcoded binary; path is a -C directory argument, not a shell command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("path %s is not a git repository: %v\n%s", path, err, string(output))
	}

	if string(output) == "" {
		return fmt.Errorf("path %s is not a git repository", path)
	}

	return nil
}

// GitClient defines the interface for git operations.
// Consumers should accept this interface to enable testing with mocks.
type GitClient interface {
	CloneRepository(url string, dest string) error
	CheckoutRef(repoPath string, ref string) error
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

// CheckoutRef checks out the specified ref (branch, tag, or commit) in the given repository path.
func (g *DefaultClient) CheckoutRef(repoPath string, ref string) error {
	return CheckoutRef(repoPath, ref)
}

// SetupWorkDir creates the workspace directory if it doesn't exist
func (g *DefaultClient) SetupWorkDir(path string) error {
	return SetupWorkDir(path)
}

// CloneRepository clones a git repository to a specific path
func CloneRepository(url string, dest string) error {
	// Check if directory already exists
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		return updateExistingRepository(url, dest)
	}

	return cloneNewRepository(url, dest)
}

func updateExistingRepository(url string, dest string) error {
	if err := ensureGitRepository(dest); err != nil {
		return err
	}

	spinner, _ := ui.Spin(fmt.Sprintf("%s Updating remote for %s...", ui.GitEmoji, dest))

	// Try setting the remote URL
	if err := setOrAddRemote(dest, url); err != nil {
		spinner.Fail(fmt.Sprintf("Failed to update remote for %s", dest))
		return err
	}

	// Fetch from the updated remote with retry logic
	if err := fetchWithRetry(dest, url); err != nil {
		spinner.Fail(fmt.Sprintf("Failed to fetch from %s", url))
		return err
	}

	spinner.Success(fmt.Sprintf("Updated remote and fetched %s", dest))
	ensureAutocrlf(dest)
	return nil
}

func cloneNewRepository(url string, dest string) error {
	spinner, _ := ui.Spin(fmt.Sprintf("%s Cloning %s...", ui.GitEmoji, url))

	autocrlf := "false"
	if config.Loaded.GetGitAutoCRLF() {
		autocrlf = "true"
	}

	var lastErr error
	var output []byte
	for attempt := 1; attempt <= 3; attempt++ {
		cmd := exec.Command("git", "clone", "-c", "core.autocrlf="+autocrlf, url, dest) // #nosec G204 -- "git" is a hardcoded binary; url/dest come from validated config, autocrlf is "true" or "false"
		output, lastErr = cmd.CombinedOutput()
		if lastErr == nil {
			spinner.Success(fmt.Sprintf("Cloned %s", dest))
			return nil
		}

		if !isRetriableGitError(string(output), lastErr) || attempt == 3 {
			break
		}

		delay := time.Duration(1<<uint(attempt)) * time.Second
		spinner.UpdateText(fmt.Sprintf("Clone attempt %d failed, retrying in %v...", attempt, delay))
		time.Sleep(delay)
		spinner.UpdateText(fmt.Sprintf("%s Cloning %s (attempt %d/3)...", ui.GitEmoji, url, attempt+1))
	}

	spinner.Fail(fmt.Sprintf("Failed to clone %s after 3 attempts", url))
	return fmt.Errorf("git clone error after 3 attempts: %v\n%s", lastErr, string(output))
}

func setOrAddRemote(dest, url string) error {
	cmd := exec.Command("git", "-C", dest, "remote", "set-url", "origin", url) // #nosec G204 -- "git" is a hardcoded binary; dest/url come from validated config
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("git", "-C", dest, "remote", "add", "origin", url) // #nosec G204 -- "git" is a hardcoded binary; dest/url come from validated config
		if err := cmd.Run(); err != nil {
			ui.Debug.Printf("failed to set or add remote origin %s: %v", url, err)
			return fmt.Errorf("failed to configure remote origin for %s: %w", dest, err)
		}
	}
	return nil
}

func fetchWithRetry(dest, url string) error {
	var fetchErr error
	var fetchOutput []byte
	for attempt := 1; attempt <= 3; attempt++ {
		cmd := exec.Command("git", "-C", dest, "fetch", "origin") // #nosec G204 -- "git" is a hardcoded binary; dest comes from validated config
		fetchOutput, fetchErr = cmd.CombinedOutput()
		if fetchErr == nil {
			return nil
		}

		if !isRetriableGitError(string(fetchOutput), fetchErr) || attempt == 3 {
			break
		}

		delay := time.Duration(1<<uint(attempt)) * time.Second
		ui.Debug.Println(fmt.Sprintf("Git fetch attempt %d failed, retrying in %v...", attempt, delay))
		time.Sleep(delay)
	}
	ui.Debug.Printf("git fetch error: %v\n%s", fetchErr, string(fetchOutput))
	return fmt.Errorf("failed to fetch remote for %s: %v\n%s", dest, fetchErr, string(fetchOutput))
}

func ensureAutocrlf(dest string) {
	autocrlf := "false"
	if config.Loaded.GetGitAutoCRLF() {
		autocrlf = "true"
	}
	_ = exec.Command("git", "-C", dest, "config", "core.autocrlf", autocrlf).Run() // #nosec G204 -- "git" is a hardcoded binary; autocrlf is "true" or "false" only
}

// isRetriableGitError checks if a git error is worth retrying (transient network issues)
func isRetriableGitError(output string, err error) bool {
	if err == nil {
		return false
	}

	// Network-related error patterns that warrant retry
	retriablePatterns := []string{
		"Could not resolve host",
		"Failed to connect",
		"Connection timed out",
		"Connection refused",
		"Connection reset",
		"Network is unreachable",
		"Temporary failure",
		"502 Bad Gateway",
		"503 Service Unavailable",
		"504 Gateway Timeout",
	}

	outputLower := strings.ToLower(output)
	for _, pattern := range retriablePatterns {
		if strings.Contains(outputLower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// CheckoutRef checks out the specified ref (branch, tag, or commit) in the given repository path.
func CheckoutRef(repoPath string, ref string) error {
	if err := ensureGitRepository(repoPath); err != nil {
		return err
	}

	spinner, _ := ui.Spin(fmt.Sprintf("%s Checking out ref '%s' in %s...", ui.GitEmoji, ref, repoPath))

	// Ensure core.autocrlf matches configuration before checkout
	autocrlf := "false"
	if config.Loaded.GetGitAutoCRLF() {
		autocrlf = "true"
	}
	cmdConfig := exec.Command("git", "-C", repoPath, "config", "core.autocrlf", autocrlf) // #nosec G204 -- "git" is a hardcoded binary; autocrlf is "true" or "false" only
	cmdConfig.Run()

	cmd := exec.Command("git", "-C", repoPath, "checkout", ref) // #nosec G204 -- "git" is a hardcoded binary; ref comes from validated config
	output, err := cmd.CombinedOutput()
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to checkout ref '%s'", ref))
		return fmt.Errorf("git checkout error: %v\n%s", err, string(output))
	}

	// Try to pull latest changes only if it looks like a branch (not a full 40-char commit hash and not a tag-like ref)
	// This is a heuristic: if it's not a 40-char hex string, we'll try to pull.
	// Tags will fail the pull but we ignore errors anyway.
	isCommit, _ := regexp.MatchString(`^[0-9a-fA-F]{40}$`, ref)
	if !isCommit {
		cmd = exec.Command("git", "-C", repoPath, "pull", "origin", ref) // #nosec G204 -- "git" is a hardcoded binary; ref comes from validated config
		// We ignore pull errors since the ref might be local-only or already up-to-date
		cmd.Run()
	}

	spinner.Success(fmt.Sprintf("Checked out ref '%s' in %s", ref, repoPath))
	return nil
}

// NormalizeLineEndings ensures a file has LF line endings (mimicks dos2unix).
func NormalizeLineEndings(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path) // #nosec G304 -- path is validated to be an existing file before this call (Stat succeeds on line 239)
	if err != nil {
		return err
	}

	// Robust line ending normalization: strip all \r characters and
	// then ensure the file uses LF (\n) correctly.
	normalized := strings.ReplaceAll(string(data), "\r", "")

	if normalized == string(data) {
		return nil // No change needed
	}

	return os.WriteFile(path, []byte(normalized), info.Mode()) // #nosec G304 G703 -- path is the same file successfully Stat'd and read above; error is returned to caller
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
