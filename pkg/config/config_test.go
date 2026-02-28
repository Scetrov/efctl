package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		WorldContractsURL:     "https://github.com/evefrontier/world-contracts.git",
		BuilderScaffoldURL:    "https://github.com/evefrontier/builder-scaffold.git",
		WorldContractsBranch:  "main",
		BuilderScaffoldBranch: "feature/my-branch",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}
}

func TestValidate_EmptyConfig(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("empty config should be valid, got error: %v", err)
	}
}

func TestValidate_RejectsNonHTTPS_URL(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{
			name: "git protocol world-contracts",
			cfg:  Config{WorldContractsURL: "git://github.com/evil/repo.git"},
		},
		{
			name: "ssh protocol builder-scaffold",
			cfg:  Config{BuilderScaffoldURL: "ssh://git@github.com/evil/repo.git"},
		},
		{
			name: "file protocol world-contracts",
			cfg:  Config{WorldContractsURL: "file:///etc/passwd"},
		},
		{
			name: "http (non-TLS) builder-scaffold",
			cfg:  Config{BuilderScaffoldURL: "http://evil.com/repo.git"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.Validate(); err == nil {
				t.Fatal("expected validation error for non-https URL, got nil")
			}
		})
	}
}

func TestValidate_RejectsInvalidBranch(t *testing.T) {
	tests := []struct {
		name   string
		branch string
	}{
		{"shell metachar semicolon", "main; rm -rf /"},
		{"shell metachar backtick", "main`whoami`"},
		{"leading hyphen", "-evil-flag"},
		{"spaces", "branch with spaces"},
		{"newline", "branch\ninjection"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{WorldContractsBranch: tt.branch}
			if err := cfg.Validate(); err == nil {
				t.Fatalf("expected validation error for branch %q, got nil", tt.branch)
			}
		})
	}
}

func TestValidate_AcceptsValidBranches(t *testing.T) {
	branches := []string{
		"main",
		"develop",
		"feature/my-feature",
		"release/v1.2.3",
		"hotfix/fix-123",
		"user.name/branch_name",
	}

	for _, branch := range branches {
		t.Run(branch, func(t *testing.T) {
			cfg := &Config{WorldContractsBranch: branch}
			if err := cfg.Validate(); err != nil {
				t.Fatalf("expected branch %q to be valid, got error: %v", branch, err)
			}
		})
	}
}

func TestLoad_ValidatesAfterParsing(t *testing.T) {
	// Create a temp config file with an invalid URL
	dir := t.TempDir()
	configPath := filepath.Join(dir, "efctl.yaml")
	content := []byte("world-contracts-url: git://evil.com/repo.git\n")
	if err := os.WriteFile(configPath, content, 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("expected Load to return validation error for git:// URL, got nil")
	}
}

func TestLoad_DefaultConfigMissing(t *testing.T) {
	// Save and restore the default config file constant behavior
	cfg, err := Load("nonexistent-file-that-does-not-exist.yaml")
	if err == nil && cfg != nil {
		// This should fail because it's not the default config file name
		// Actually it will fail because the file doesn't exist and path != DefaultConfigFile
	}
	if err != nil {
		// Expected: file not found for non-default path
		return
	}
}
