package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// ── Additional: Load ───────────────────────────────────────────────

func TestLoad_ValidFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "efctl.yaml")
	content := `world-contracts-url: https://github.com/test/wc.git
builder-scaffold-url: https://github.com/test/bs.git
world-contracts-branch: develop
builder-scaffold-branch: feature/x
with-frontend: true
with-graphql: false
`
	require.NoError(t, os.WriteFile(p, []byte(content), 0600))

	cfg, err := Load(p)
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/test/wc.git", cfg.WorldContractsURL)
	assert.Equal(t, "https://github.com/test/bs.git", cfg.BuilderScaffoldURL)
	assert.Equal(t, "develop", cfg.WorldContractsBranch)
	assert.Equal(t, "feature/x", cfg.BuilderScaffoldBranch)
	assert.True(t, *cfg.WithFrontend)
	assert.False(t, *cfg.WithGraphql)
	// Loaded global should be set
	assert.Equal(t, cfg, Loaded)
}

func TestLoad_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.yaml")
	require.NoError(t, os.WriteFile(p, []byte(":\t\t\nbad: [yaml: {"), 0600))

	_, err := Load(p)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config file")
}

func TestLoad_DefaultFileMissing_ReturnsEmpty(t *testing.T) {
	// When loading the default file and it doesn't exist, should return empty config
	old := Loaded
	defer func() { Loaded = old }()

	// Change to temp dir so DefaultConfigFile won't be found
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(t.TempDir())

	cfg, err := Load(DefaultConfigFile)
	require.NoError(t, err)
	assert.NotNil(t, cfg)
}

// ── Getter defaults ────────────────────────────────────────────────

func TestGetWorldContractsURL_Default(t *testing.T) {
	cfg := &Config{}
	assert.Equal(t, DefaultWorldContractsURL, cfg.GetWorldContractsURL())
}

func TestGetWorldContractsURL_Custom(t *testing.T) {
	cfg := &Config{WorldContractsURL: "https://example.com/wc.git"}
	assert.Equal(t, "https://example.com/wc.git", cfg.GetWorldContractsURL())
}

func TestGetBuilderScaffoldURL_Default(t *testing.T) {
	cfg := &Config{}
	assert.Equal(t, DefaultBuilderScaffoldURL, cfg.GetBuilderScaffoldURL())
}

func TestGetBuilderScaffoldURL_Custom(t *testing.T) {
	cfg := &Config{BuilderScaffoldURL: "https://example.com/bs.git"}
	assert.Equal(t, "https://example.com/bs.git", cfg.GetBuilderScaffoldURL())
}

func TestGetWorldContractsBranch_Default(t *testing.T) {
	cfg := &Config{}
	assert.Equal(t, DefaultBranch, cfg.GetWorldContractsBranch())
}

func TestGetWorldContractsBranch_Custom(t *testing.T) {
	cfg := &Config{WorldContractsBranch: "develop"}
	assert.Equal(t, "develop", cfg.GetWorldContractsBranch())
}

func TestGetBuilderScaffoldBranch_Default(t *testing.T) {
	cfg := &Config{}
	assert.Equal(t, DefaultBranch, cfg.GetBuilderScaffoldBranch())
}

func TestGetBuilderScaffoldBranch_Custom(t *testing.T) {
	cfg := &Config{BuilderScaffoldBranch: "release/v2"}
	assert.Equal(t, "release/v2", cfg.GetBuilderScaffoldBranch())
}

func TestGetters_NilConfig(t *testing.T) {
	var cfg *Config
	assert.Equal(t, DefaultWorldContractsURL, cfg.GetWorldContractsURL())
	assert.Equal(t, DefaultBuilderScaffoldURL, cfg.GetBuilderScaffoldURL())
	assert.Equal(t, DefaultBranch, cfg.GetWorldContractsBranch())
	assert.Equal(t, DefaultBranch, cfg.GetBuilderScaffoldBranch())
}
