package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupWorkDir(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "efctl-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // clean up

	workDir := filepath.Join(tempDir, "workspace")

	// Test 1: Directory does not exist, should create it
	err = SetupWorkDir(workDir)
	if err != nil {
		t.Errorf("Expected nil error, got: %v", err)
	}

	// Verify directory was created
	info, err := os.Stat(workDir)
	if err != nil {
		t.Errorf("Expected directory to exist, but got error: %v", err)
	} else if !info.IsDir() {
		t.Errorf("Expected path to be a directory")
	}

	// Test 2: Directory already exists, should not error
	err = SetupWorkDir(workDir)
	if err != nil {
		t.Errorf("Expected nil error when directory already exists, got: %v", err)
	}
}

func TestCloneRepository_InvalidURL(t *testing.T) {
	// We pass an invalid URL, we expect an error from git clone
	tempDir, err := os.MkdirTemp("", "efctl-git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dest := filepath.Join(tempDir, "invalid-repo")

	err = CloneRepository("https://invalid.url.that.does.not.exist/repo.git", dest)
	if err == nil {
		t.Errorf("Expected an error when cloning an invalid URL, got nil")
	}
}

func TestCloneRepository_DirectoryExists(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "efctl-git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dest := filepath.Join(tempDir, "existing-repo")
	os.Mkdir(dest, 0755)

	// Initialize as a git repo so remote commands work
	cmd := exec.Command("git", "-C", dest, "init")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Should return nil because directory already exists and remote was added/fetched successfully
	err = CloneRepository("https://github.com/evefrontier/world-contracts.git", dest)
	if err != nil {
		t.Errorf("Expected nil error when directory already exists and remote updated, got: %v", err)
	}
}

func TestCloneRepository_DirectoryExistsNotGitRepoFails(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "efctl-git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dest := filepath.Join(tempDir, "not-a-repo")
	if err := os.Mkdir(dest, 0750); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	err = CloneRepository("https://github.com/evefrontier/world-contracts.git", dest)
	if err == nil {
		t.Fatal("Expected an error when destination exists but is not a git repository")
	}

	if !strings.Contains(err.Error(), "not a git repository") {
		t.Fatalf("Expected not-a-git-repository error, got: %v", err)
	}
}

func TestCheckoutBranch_NonGitRepoFails(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "efctl-git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	err = CheckoutRef(tempDir, "main")
	if err == nil {
		t.Fatal("Expected checkout to fail for non-git directory")
	}

	if !strings.Contains(err.Error(), "not a git repository") {
		t.Fatalf("Expected not-a-git-repository error, got: %v", err)
	}
}
