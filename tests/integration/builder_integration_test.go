//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"efctl/pkg/builder"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInitExtensionEnv_FullWorkflow validates the complete init-extension
// flow using a realistic temp workspace layout.
func TestInitExtensionEnv_FullWorkflow(t *testing.T) {
	ws := t.TempDir()
	network := "localnet"

	// Setup world-contracts directory with required files
	worldDir := filepath.Join(ws, "world-contracts")
	deployDir := filepath.Join(worldDir, "deployments", network)
	contractsDir := filepath.Join(worldDir, "contracts", "world")
	require.NoError(t, os.MkdirAll(deployDir, 0750))
	require.NoError(t, os.MkdirAll(contractsDir, 0750))

	// Create .env with test keys
	envContent := `ADMIN_ADDRESS=0xadmin123
ADMIN_PRIVATE_KEY=suiprivkeyadmin
PLAYER_A_ADDRESS=0xplayera123
PLAYER_A_PRIVATE_KEY=suiprivkeyplayera
PLAYER_B_ADDRESS=0xplayerb123
PLAYER_B_PRIVATE_KEY=suiprivkeyplayerb
`
	require.NoError(t, os.WriteFile(filepath.Join(worldDir, ".env"), []byte(envContent), 0600))

	// Create extracted-object-ids.json
	ids := map[string]interface{}{
		"world": map[string]interface{}{
			"packageId": "0xworld123",
		},
	}
	idsBytes, _ := json.Marshal(ids)
	require.NoError(t, os.WriteFile(filepath.Join(deployDir, "extracted-object-ids.json"), idsBytes, 0600))

	// Create test-resources.json
	require.NoError(t, os.WriteFile(filepath.Join(worldDir, "test-resources.json"), []byte(`{"test": true}`), 0600))

	// Create Pub.localnet.toml
	require.NoError(t, os.WriteFile(filepath.Join(contractsDir, "Pub.localnet.toml"), []byte("pub-data"), 0600))

	// Create a deployment file to be copied
	require.NoError(t, os.WriteFile(filepath.Join(deployDir, "deploy.log"), []byte("deploy log"), 0600))

	// Setup builder-scaffold directory with .env.example
	scaffoldDir := filepath.Join(ws, "builder-scaffold")
	require.NoError(t, os.MkdirAll(scaffoldDir, 0750))
	envExample := `SUI_NETWORK=
WORLD_PACKAGE_ID=
ADMIN_ADDRESS=
ADMIN_PRIVATE_KEY=
PLAYER_A_ADDRESS=
PLAYER_A_PRIVATE_KEY=
`
	require.NoError(t, os.WriteFile(filepath.Join(scaffoldDir, ".env.example"), []byte(envExample), 0600))

	// Run the init
	err := builder.InitExtensionEnv(ws, network)
	require.NoError(t, err)

	// Verify builder-scaffold/.env was created and populated
	envData, err := os.ReadFile(filepath.Join(scaffoldDir, ".env"))
	require.NoError(t, err)
	envStr := string(envData)

	assert.Contains(t, envStr, "SUI_NETWORK=localnet")
	assert.Contains(t, envStr, "WORLD_PACKAGE_ID=0xworld123")
	assert.Contains(t, envStr, "ADMIN_ADDRESS=0xadmin123")

	// Verify deployment files were copied
	assert.FileExists(t, filepath.Join(scaffoldDir, "test-resources.json"))
	assert.FileExists(t, filepath.Join(scaffoldDir, "deployments", network, "deploy.log"))
}
