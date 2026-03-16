package status

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStatsOutput(t *testing.T) {
	sui := ContainerStat{Name: "sui-playground", Status: "Stopped", CPU: "-", Mem: "-"}
	pg := ContainerStat{Name: "efctl-postgres", Status: "Stopped", CPU: "-", Mem: "-"}
	fe := ContainerStat{Name: "efctl-frontend", Status: "Stopped", CPU: "-", Mem: "-"}

	out := "sui-playground\t25.3%\t500MiB / 2GiB\n" +
		"efctl-postgres\t3.2%\t120MiB / 2GiB\n" +
		"efctl-frontend\t7.1%\t200MiB / 2GiB\n"

	sui, pg, fe = parseStatsOutput(out, sui, pg, fe)

	assert.Equal(t, "Running", sui.Status)
	assert.Equal(t, "25.3%", sui.CPU)
	assert.Equal(t, "Running", pg.Status)
	assert.Equal(t, "3.2%", pg.CPU)
	assert.Equal(t, "Running", fe.Status)
	assert.Equal(t, "7.1%", fe.CPU)
}

func TestGatherWorldInfo(t *testing.T) {
	workspace := t.TempDir()

	worldDir := filepath.Join(workspace, "world-contracts")
	deployDir := filepath.Join(worldDir, "deployments", "localnet")
	require.NoError(t, os.MkdirAll(deployDir, 0750))

	envContent := "ADMIN_ADDRESS=0xabc\nPLAYER_A_ADDRESS=0xdef\nUNRELATED_VAR=value\n"
	require.NoError(t, os.WriteFile(filepath.Join(worldDir, ".env"), []byte(envContent), 0600))

	jsonContent := `{"world":{"packageId":"0x111","governorCap":"0x222","adminAcl":"0x333"}}`
	require.NoError(t, os.WriteFile(filepath.Join(deployDir, "extracted-object-ids.json"), []byte(jsonContent), 0600))

	info := GatherWorldInfo(workspace, "http://localhost:9000")

	assert.Equal(t, "0x111", info.PackageID)
	assert.Equal(t, "0x222", info.Objects["governorCap"])
	assert.Equal(t, "0x333", info.Objects["adminAcl"])
	assert.Equal(t, "0xabc", info.Addresses["Admin"])
	assert.Equal(t, "0xdef", info.Addresses["Player A"])
	_, hasNonAddress := info.Addresses["UNRELATED_VAR"]
	assert.False(t, hasNonAddress)
}

func TestExtractEnvVarsFallback(t *testing.T) {
	workspace := t.TempDir()

	// Place .env in test-env/world-contracts
	worldDir := filepath.Join(workspace, "test-env", "world-contracts")
	require.NoError(t, os.MkdirAll(worldDir, 0750))

	envContent := "ADMIN_ADDRESS=0x999\n"
	require.NoError(t, os.WriteFile(filepath.Join(worldDir, ".env"), []byte(envContent), 0600))

	vars := extractEnvVars(workspace)
	assert.Equal(t, "0x999", vars["ADMIN_ADDRESS"])
}
