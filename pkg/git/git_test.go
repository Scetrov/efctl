package git

import (
	"os"
	"path/filepath"
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

	// Should return nil because directory already exists
	err = CloneRepository("https://github.com/evefrontier/world-contracts.git", dest)
	if err != nil {
		t.Errorf("Expected nil error when directory already exists, got: %v", err)
	}
}
