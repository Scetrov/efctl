//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"efctl/pkg/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigLoadAndGetters validates the full config loading pipeline:
// write YAML → Load → Validate → Getters return correct values.
func TestConfigLoadAndGetters(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "efctl.yaml")
	yaml := `world-contracts-url: https://github.com/test/wc.git
builder-scaffold-url: https://github.com/test/bs.git
world-contracts-branch: develop
builder-scaffold-branch: release/v2
with-frontend: true
with-graphql: true
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(yaml), 0600))

	cfg, err := config.Load(cfgPath)
	require.NoError(t, err)

	assert.Equal(t, "https://github.com/test/wc.git", cfg.GetWorldContractsURL())
	assert.Equal(t, "https://github.com/test/bs.git", cfg.GetBuilderScaffoldURL())
	assert.Equal(t, "develop", cfg.GetWorldContractsBranch())
	assert.Equal(t, "release/v2", cfg.GetBuilderScaffoldBranch())
	assert.True(t, *cfg.WithFrontend)
	assert.True(t, *cfg.WithGraphql)
}

// TestConfigDefaults validates that an empty config falls back to defaults.
func TestConfigDefaults(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "efctl.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(""), 0600))

	cfg, err := config.Load(cfgPath)
	require.NoError(t, err)

	assert.Equal(t, config.DefaultWorldContractsURL, cfg.GetWorldContractsURL())
	assert.Equal(t, config.DefaultBuilderScaffoldURL, cfg.GetBuilderScaffoldURL())
	assert.Equal(t, config.DefaultBranch, cfg.GetWorldContractsBranch())
	assert.Equal(t, config.DefaultBranch, cfg.GetBuilderScaffoldBranch())
}

// TestConfigValidation_RejectsInsecure validates that non-HTTPS URLs are rejected.
func TestConfigValidation_RejectsInsecure(t *testing.T) {
	dir := t.TempDir()
	for _, url := range []string{
		"git://github.com/evil/repo.git",
		"ssh://git@github.com/evil/repo.git",
		"file:///etc/passwd",
		"http://evil.com/repo.git",
	} {
		cfgPath := filepath.Join(dir, "efctl.yaml")
		yaml := "world-contracts-url: " + url + "\n"
		require.NoError(t, os.WriteFile(cfgPath, []byte(yaml), 0600))

		_, err := config.Load(cfgPath)
		assert.Error(t, err, "URL %s should be rejected", url)
	}
}
