package builder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── extractPublishIDs ──────────────────────────────────────────────

func TestExtractPublishIDs_ValidJSON(t *testing.T) {
	output := `some build log
{
  "objectChanges": [
    {"type":"published","packageId":"0xPKG123"},
    {"type":"created","objectType":"0x::some::ExtensionConfig","objectId":"0xCFG456"}
  ]
}`

	pkgID, cfgID, err := extractPublishIDs(output)
	require.NoError(t, err)
	assert.Equal(t, "0xPKG123", pkgID)
	assert.Equal(t, "0xCFG456", cfgID)
}

func TestExtractPublishIDs_NoJSON(t *testing.T) {
	_, _, err := extractPublishIDs("pure text, no json at all")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no JSON object found")
}

func TestExtractPublishIDs_InvalidJSON(t *testing.T) {
	_, _, err := extractPublishIDs("{bad json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal")
}

func TestExtractPublishIDs_NoChanges(t *testing.T) {
	output := `{"objectChanges":[]}`
	pkgID, cfgID, err := extractPublishIDs(output)
	require.NoError(t, err)
	assert.Empty(t, pkgID)
	assert.Empty(t, cfgID)
}

func TestExtractPublishIDs_OnlyPackage(t *testing.T) {
	output := `{"objectChanges":[{"type":"published","packageId":"0xPKG"}]}`
	pkgID, cfgID, err := extractPublishIDs(output)
	require.NoError(t, err)
	assert.Equal(t, "0xPKG", pkgID)
	assert.Empty(t, cfgID)
}

func TestExtractPublishIDs_CaseInsensitiveConfig(t *testing.T) {
	output := `{"objectChanges":[{"type":"created","objectType":"0x::module::extensionconfig","objectId":"0xLOWER"}]}`
	_, cfgID, err := extractPublishIDs(output)
	require.NoError(t, err)
	assert.Equal(t, "0xLOWER", cfgID)
}

// ── buildPublishCmd ────────────────────────────────────────────────

func TestBuildPublishCmd_Localnet(t *testing.T) {
	tmp := t.TempDir()
	// Create the pub file so removal works
	pubDir := filepath.Join(tmp, "builder-scaffold", "deployments", "localnet")
	require.NoError(t, os.MkdirAll(pubDir, 0750))
	pubFile := filepath.Join(pubDir, "Pub.extension.toml")
	require.NoError(t, os.WriteFile(pubFile, []byte("old"), 0600))

	cmd, err := buildPublishCmd(tmp, "localnet", "/workspace/contracts/my_ext")
	require.NoError(t, err)
	assert.Contains(t, cmd, "test-publish")
	assert.Contains(t, cmd, "/workspace/contracts/my_ext")
	// The stale file should have been removed
	_, statErr := os.Stat(pubFile)
	assert.True(t, os.IsNotExist(statErr))
}

func TestBuildPublishCmd_Testnet(t *testing.T) {
	cmd, err := buildPublishCmd("/ws", "testnet", "/workspace/contracts/ext")
	require.NoError(t, err)
	assert.Contains(t, cmd, "sui client publish")
	assert.Contains(t, cmd, "--json")
}

func TestBuildPublishCmd_UnsupportedNetwork(t *testing.T) {
	_, err := buildPublishCmd("/ws", "mainnet", "/dir")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported network")
}

// ── parseDotEnv ────────────────────────────────────────────────────

func TestParseDotEnv(t *testing.T) {
	content := `# comment
FOO=bar
BAZ = qux

# another comment
EMPTY=
`
	f := filepath.Join(t.TempDir(), ".env")
	require.NoError(t, os.WriteFile(f, []byte(content), 0600))

	m, err := parseDotEnv(f)
	require.NoError(t, err)
	assert.Equal(t, "bar", m["FOO"])
	assert.Equal(t, "qux", m["BAZ"])
	assert.Equal(t, "", m["EMPTY"])
	_, hasComment := m["# comment"]
	assert.False(t, hasComment)
}

func TestParseDotEnv_FileNotFound(t *testing.T) {
	_, err := parseDotEnv("/nonexistent/.env")
	assert.Error(t, err)
}

// ── updateEnvFile ──────────────────────────────────────────────────

func TestUpdateEnvFile_UpdatesExistingKeys(t *testing.T) {
	initial := "FOO=old\nBAR=keep\n"
	f := filepath.Join(t.TempDir(), ".env")
	require.NoError(t, os.WriteFile(f, []byte(initial), 0600))

	err := updateEnvFile(f, map[string]string{"FOO": "new"})
	require.NoError(t, err)

	content, _ := os.ReadFile(f)
	assert.Contains(t, string(content), "FOO=new")
	assert.Contains(t, string(content), "BAR=keep")
}

func TestUpdateEnvFile_AppendsNewKeys(t *testing.T) {
	initial := "FOO=val\n"
	f := filepath.Join(t.TempDir(), ".env")
	require.NoError(t, os.WriteFile(f, []byte(initial), 0600))

	err := updateEnvFile(f, map[string]string{"NEW_KEY": "new_val"})
	require.NoError(t, err)

	content, _ := os.ReadFile(f)
	assert.Contains(t, string(content), "NEW_KEY=new_val")
	assert.Contains(t, string(content), "FOO=val")
}

func TestUpdateEnvFile_PreservesComments(t *testing.T) {
	initial := "# This is a comment\nFOO=old\n"
	f := filepath.Join(t.TempDir(), ".env")
	require.NoError(t, os.WriteFile(f, []byte(initial), 0600))

	err := updateEnvFile(f, map[string]string{"FOO": "new"})
	require.NoError(t, err)

	content, _ := os.ReadFile(f)
	assert.Contains(t, string(content), "# This is a comment")
}

// ── extractWorldPackageId ──────────────────────────────────────────

func TestExtractWorldPackageId(t *testing.T) {
	data := map[string]any{
		"world": map[string]any{
			"packageId": "0xWORLD123",
		},
	}
	b, _ := json.Marshal(data)
	f := filepath.Join(t.TempDir(), "ids.json")
	require.NoError(t, os.WriteFile(f, b, 0600))

	id, err := extractWorldPackageId(f)
	require.NoError(t, err)
	assert.Equal(t, "0xWORLD123", id)
}

func TestExtractWorldPackageId_MissingFile(t *testing.T) {
	_, err := extractWorldPackageId("/no/such/file.json")
	assert.Error(t, err)
}

func TestExtractWorldPackageId_InvalidJSON(t *testing.T) {
	f := filepath.Join(t.TempDir(), "bad.json")
	require.NoError(t, os.WriteFile(f, []byte("not json"), 0600))
	_, err := extractWorldPackageId(f)
	assert.Error(t, err)
}

// ── copyFile / copyDir ─────────────────────────────────────────────

func TestCopyFile(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src.txt")
	dst := filepath.Join(t.TempDir(), "dst.txt")
	require.NoError(t, os.WriteFile(src, []byte("hello"), 0600))

	err := copyFile(src, dst)
	require.NoError(t, err)

	content, _ := os.ReadFile(dst)
	assert.Equal(t, "hello", string(content))
}

func TestCopyFile_MissingSrc(t *testing.T) {
	err := copyFile("/nonexistent", filepath.Join(t.TempDir(), "dst"))
	assert.Error(t, err)
}

func TestCopyDir(t *testing.T) {
	src := filepath.Join(t.TempDir(), "srcdir")
	dst := filepath.Join(t.TempDir(), "dstdir")
	require.NoError(t, os.MkdirAll(filepath.Join(src, "sub"), 0750))
	require.NoError(t, os.WriteFile(filepath.Join(src, "a.txt"), []byte("aaa"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(src, "sub", "b.txt"), []byte("bbb"), 0600))

	require.NoError(t, os.MkdirAll(dst, 0750))
	err := copyDir(src, dst)
	require.NoError(t, err)

	content, _ := os.ReadFile(filepath.Join(dst, "a.txt"))
	assert.Equal(t, "aaa", string(content))
	content, _ = os.ReadFile(filepath.Join(dst, "sub", "b.txt"))
	assert.Equal(t, "bbb", string(content))
}
