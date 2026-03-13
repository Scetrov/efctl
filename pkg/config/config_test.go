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
		WorldContractsURL:  "https://github.com/evefrontier/world-contracts.git",
		BuilderScaffoldURL: "https://github.com/evefrontier/builder-scaffold.git",
		WorldContractsRef:  "main",
		BuilderScaffoldRef: "feature/my-branch",
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
			cfg := &Config{WorldContractsRef: tt.branch}
			if err := cfg.Validate(); err == nil {
				t.Fatalf("expected validation error for ref %q, got nil", tt.branch)
			}
		})
	}
}

func TestValidate_AcceptsValidBranches(t *testing.T) {
	refs := []string{
		"main",
		"develop",
		"feature/my-feature",
		"release/v1.2.3",
		"hotfix/fix-123",
		"user.name/branch_name",
		"v1.0.0",
		"a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2", // 40-char hex (commit)
	}

	for _, ref := range refs {
		t.Run(ref, func(t *testing.T) {
			cfg := &Config{WorldContractsRef: ref}
			if err := cfg.Validate(); err != nil {
				t.Fatalf("expected ref %q to be valid, got error: %v", ref, err)
			}
		})
	}
}

func TestValidate_AcceptsAdditionalBindMounts(t *testing.T) {
	cfg := &Config{
		AdditionalBindMounts: []AdditionalBindMount{{
			HostPath:   "./contracts",
			Identifier: "contracts_mount",
		}},
	}

	require.NoError(t, cfg.Validate())
}

func TestValidate_RejectsDuplicateAdditionalBindMountIdentifiers(t *testing.T) {
	cfg := &Config{
		AdditionalBindMounts: []AdditionalBindMount{
			{HostPath: "./contracts-one", Identifier: "duplicate_mount"},
			{HostPath: "./contracts-two", Identifier: "duplicate_mount"},
		},
	}

	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicates")
}

func TestValidate_RejectsInvalidAdditionalBindMountIdentifier(t *testing.T) {
	cfg := &Config{
		AdditionalBindMounts: []AdditionalBindMount{{
			HostPath:   "./contracts",
			Identifier: "contracts/mount",
		}},
	}

	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "identifier contains invalid characters")
}

func TestResolveAdditionalBindMounts_UsesConfigDirectory(t *testing.T) {
	configDir := t.TempDir()
	mountDir := filepath.Join(configDir, "contracts")
	require.NoError(t, os.MkdirAll(mountDir, 0750))

	cfgPath := filepath.Join(configDir, DefaultConfigFile)
	require.NoError(t, os.WriteFile(cfgPath, []byte("additional-bind-mounts:\n  - hostPath: ./contracts\n    identifier: contracts_mount\n"), 0600))

	cfg, err := Load(cfgPath)
	require.NoError(t, err)

	resolved, err := cfg.ResolveAdditionalBindMounts("")
	require.NoError(t, err)
	require.Len(t, resolved, 1)
	assert.Equal(t, mountDir, resolved[0].HostPath)
	assert.Equal(t, "contracts_mount", resolved[0].Identifier)
}

func TestResolveAdditionalBindMounts_RejectsMissingDirectory(t *testing.T) {
	cfg := &Config{
		AdditionalBindMounts: []AdditionalBindMount{{
			HostPath:   "./missing-contracts",
			Identifier: "contracts_mount",
		}},
	}

	_, err := cfg.ResolveAdditionalBindMounts(t.TempDir())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
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
	assert.Equal(t, "develop", cfg.GetWorldContractsRef())
	assert.Equal(t, "feature/x", cfg.GetBuilderScaffoldRef())
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

func TestGetWorldContractsRef_Default(t *testing.T) {
	cfg := &Config{}
	assert.Equal(t, DefaultBranch, cfg.GetWorldContractsRef())
}

func TestGetWorldContractsRef_Custom(t *testing.T) {
	cfg := &Config{WorldContractsRef: "develop"}
	assert.Equal(t, "develop", cfg.GetWorldContractsRef())
}

func TestGetWorldContractsRef_BackwardCompatibility(t *testing.T) {
	cfg := &Config{WorldContractsBranch: "legacy-branch"}
	assert.Equal(t, "legacy-branch", cfg.GetWorldContractsRef())
}

func TestGetWorldContractsRef_Priority(t *testing.T) {
	cfg := &Config{WorldContractsRef: "new-ref", WorldContractsBranch: "old-branch"}
	assert.Equal(t, "new-ref", cfg.GetWorldContractsRef())
}

func TestGetBuilderScaffoldRef_Default(t *testing.T) {
	cfg := &Config{}
	assert.Equal(t, DefaultBranch, cfg.GetBuilderScaffoldRef())
}

func TestGetBuilderScaffoldRef_Custom(t *testing.T) {
	cfg := &Config{BuilderScaffoldRef: "release/v2"}
	assert.Equal(t, "release/v2", cfg.GetBuilderScaffoldRef())
}

func TestGetBuilderScaffoldRef_BackwardCompatibility(t *testing.T) {
	cfg := &Config{BuilderScaffoldBranch: "legacy-branch"}
	assert.Equal(t, "legacy-branch", cfg.GetBuilderScaffoldRef())
}

func TestGetBuilderScaffoldRef_Priority(t *testing.T) {
	cfg := &Config{BuilderScaffoldRef: "new-ref", BuilderScaffoldBranch: "old-branch"}
	assert.Equal(t, "new-ref", cfg.GetBuilderScaffoldRef())
}

func TestGetters_NilConfig(t *testing.T) {
	var cfg *Config
	assert.Equal(t, DefaultWorldContractsURL, cfg.GetWorldContractsURL())
	assert.Equal(t, DefaultBuilderScaffoldURL, cfg.GetBuilderScaffoldURL())
	assert.Equal(t, DefaultBranch, cfg.GetWorldContractsRef())
	assert.Equal(t, DefaultBranch, cfg.GetBuilderScaffoldRef())
}

func TestFindDefaultConfigPath_FindsInCurrentDirectory(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, DefaultConfigFile), []byte("with-frontend: true\n"), 0600))

	path, found, err := FindDefaultConfigPath(dir)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, filepath.Join(dir, DefaultConfigFile), path)
}

func TestFindDefaultConfigPath_FindsInParentDirectory(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b", "c")
	require.NoError(t, os.MkdirAll(nested, 0750))
	require.NoError(t, os.WriteFile(filepath.Join(root, AlternateDefaultConfigFile), []byte("with-graphql: true\n"), 0600))

	path, found, err := FindDefaultConfigPath(nested)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, filepath.Join(root, AlternateDefaultConfigFile), path)
}

func TestFindDefaultConfigPath_NotFound(t *testing.T) {
	dir := t.TempDir()
	path, found, err := FindDefaultConfigPath(dir)
	require.NoError(t, err)
	assert.False(t, found)
	assert.Equal(t, "", path)
}

func TestFindDefaultConfigPath_PrefersYAMLOverYMLInSameDirectory(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, AlternateDefaultConfigFile), []byte("with-graphql: true\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, DefaultConfigFile), []byte("with-frontend: true\n"), 0600))

	path, found, err := FindDefaultConfigPath(dir)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, filepath.Join(dir, DefaultConfigFile), path)
}

func TestDefaultConfigYAML_MatchesRepositorySample(t *testing.T) {
	samplePath := filepath.Join("..", "..", DefaultConfigFile)
	data, err := os.ReadFile(samplePath)
	require.NoError(t, err)
	assert.Equal(t, DefaultConfigYAML(), string(data))
}
