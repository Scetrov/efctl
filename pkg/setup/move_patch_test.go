package setup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanStaleMoveLocks_RemovesLockFiles(t *testing.T) {
	workspace := t.TempDir()
	contractsDir := filepath.Join(workspace, "world-contracts", "contracts")

	// Create two contract directories with Move.lock files
	for _, pkg := range []string{"world", "assets"} {
		pkgDir := filepath.Join(contractsDir, pkg)
		require.NoError(t, os.MkdirAll(pkgDir, 0755))
		require.NoError(t, os.WriteFile(
			filepath.Join(pkgDir, "Move.lock"),
			[]byte("[move]\nversion = 4\n"),
			0644,
		))
		// Also create a Move.toml that should NOT be removed
		require.NoError(t, os.WriteFile(
			filepath.Join(pkgDir, "Move.toml"),
			[]byte("[package]\nname = \"test\"\n"),
			0644,
		))
	}

	cleanStaleMoveLocks(workspace)

	for _, pkg := range []string{"world", "assets"} {
		lockPath := filepath.Join(contractsDir, pkg, "Move.lock")
		_, err := os.Stat(lockPath)
		assert.True(t, os.IsNotExist(err), "Move.lock should be removed for %s", pkg)

		tomlPath := filepath.Join(contractsDir, pkg, "Move.toml")
		_, err = os.Stat(tomlPath)
		assert.NoError(t, err, "Move.toml should still exist for %s", pkg)
	}
}

func TestCleanStaleMoveLocks_NoopOnMissingDir(t *testing.T) {
	workspace := t.TempDir()
	// contracts dir does not exist — should not panic
	cleanStaleMoveLocks(workspace)
}

func TestCleanStaleMoveLocks_NoopOnMissingLock(t *testing.T) {
	workspace := t.TempDir()
	contractsDir := filepath.Join(workspace, "world-contracts", "contracts")
	pkgDir := filepath.Join(contractsDir, "world")
	require.NoError(t, os.MkdirAll(pkgDir, 0755))
	// No Move.lock file — should not error
	cleanStaleMoveLocks(workspace)
}
