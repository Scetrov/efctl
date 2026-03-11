package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"efctl/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitCommand_CreatesDefaultConfigFile(t *testing.T) {
	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
		configFile = config.DefaultConfigFile
		initForce = false
	})
	require.NoError(t, os.Chdir(tmp))

	configFile = config.DefaultConfigFile
	initForce = false
	rootCmd.SetArgs([]string{"init"})

	require.NoError(t, rootCmd.Execute())

	data, err := os.ReadFile(filepath.Join(tmp, config.DefaultConfigFile))
	require.NoError(t, err)
	assert.Equal(t, config.DefaultConfigYAML(), string(data))
	info, err := os.Stat(filepath.Join(tmp, config.DefaultConfigFile))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestInitCommand_DoesNotOverwriteExistingFileWithoutForce(t *testing.T) {
	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
		configFile = config.DefaultConfigFile
		initForce = false
	})
	require.NoError(t, os.Chdir(tmp))

	original := []byte("with-frontend: false\n")
	require.NoError(t, os.WriteFile(filepath.Join(tmp, config.DefaultConfigFile), original, 0600))

	configFile = config.DefaultConfigFile
	initForce = false
	rootCmd.SetArgs([]string{"init"})

	err = rootCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	data, readErr := os.ReadFile(filepath.Join(tmp, config.DefaultConfigFile))
	require.NoError(t, readErr)
	assert.Equal(t, string(original), string(data))
}

func TestInitCommand_OverwritesExistingFileWithForce(t *testing.T) {
	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
		configFile = config.DefaultConfigFile
		initForce = false
	})
	require.NoError(t, os.Chdir(tmp))

	require.NoError(t, os.WriteFile(filepath.Join(tmp, config.DefaultConfigFile), []byte("with-frontend: false\n"), 0600))

	configFile = config.DefaultConfigFile
	initForce = false
	rootCmd.SetArgs([]string{"init", "--force"})

	require.NoError(t, rootCmd.Execute())

	data, err := os.ReadFile(filepath.Join(tmp, config.DefaultConfigFile))
	require.NoError(t, err)
	assert.Equal(t, config.DefaultConfigYAML(), string(data))
}

func TestInitCommand_UsesExplicitConfigPath(t *testing.T) {
	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
		configFile = config.DefaultConfigFile
		initForce = false
	})
	require.NoError(t, os.Chdir(tmp))

	targetPath := filepath.Join("configs", "project.yaml")
	configFile = config.DefaultConfigFile
	initForce = false
	rootCmd.SetArgs([]string{"init", "--config-file", targetPath})

	require.NoError(t, rootCmd.Execute())

	data, err := os.ReadFile(filepath.Join(tmp, targetPath))
	require.NoError(t, err)
	assert.Equal(t, config.DefaultConfigYAML(), string(data))
}
