package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeSuiFixture(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "scripts"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(content), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "scripts", "entrypoint.sh"), []byte(fullEntrypoint()), 0700))
	return dir
}

func TestPrepareDockerEnvironment_PreservesFilesOnInvalidSuiSetup(t *testing.T) {
	for _, tc := range []struct{ name, content, input string }{
		{"missing WORKDIR", "FROM ubuntu:24.04\n" + testGlobalSuiBlock() + "\n" + testRelocationBlock() + "\n", ""},
		{"LFS pointer", fullDockerfile(), "version https://git-lfs.github.com/spec/v1\noid sha256:1234\nsize 999\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeSuiFixture(t, tc.content)
			if tc.input != "" {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "sui"), []byte(tc.input), 0600))
			}
			err := prepareDockerEnvironment(dir, "podman", false, false)
			require.Error(t, err)
			if tc.input != "" {
				assert.Contains(t, err.Error(), "git lfs pull")
			}
			dockerfile, readErr := os.ReadFile(filepath.Join(dir, "Dockerfile"))
			require.NoError(t, readErr)
			entrypoint, readErr := os.ReadFile(filepath.Join(dir, "scripts", "entrypoint.sh"))
			require.NoError(t, readErr)
			assert.Equal(t, tc.content, string(dockerfile))
			assert.Equal(t, fullEntrypoint(), string(entrypoint))
		})
	}
}

func TestCustomSui_InstructionVariations(t *testing.T) {
	for _, tc := range []struct{ name, block string }{
		{"changed comment", strings.Replace(testGlobalSuiBlock(), "# Sui CLI via suiup pinned to testnet-1.81.1 (last release with fullnode JSON-RPC)", "# Install Sui", 1)},
		{"CRLF", strings.ReplaceAll(testGlobalSuiBlock()+"\n"+testRelocationBlock(), "\n", "\r\n")},
		{"indented", strings.ReplaceAll(testGlobalSuiBlock()+"\n"+testRelocationBlock(), "\nRUN ", "\n  RUN ")},
		{"existing custom", "# My custom build\nCOPY sui /usr/local/bin/sui\nRUN chmod +x /usr/local/bin/sui \\\n    && sui --version"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeSuiFixture(t, "FROM ubuntu:24.04\n"+tc.block+"\nWORKDIR /workspace\n")
			require.NoError(t, os.WriteFile(filepath.Join(dir, "sui"), []byte("review fixture"), 0600))
			require.NoError(t, prepareDockerEnvironment(dir, "podman", false, false))
			first, err := os.ReadFile(filepath.Join(dir, "Dockerfile"))
			require.NoError(t, err)
			assert.NotContains(t, string(first), "suiup install sui")
			assert.NotContains(t, string(first), `ENV PATH="/root/.local/bin:/root/.suiup/bin:${PATH}"`)
			assert.Equal(t, 1, strings.Count(string(first), "COPY sui /usr/local/bin/sui"))
			assert.Contains(t, string(first), "&& /usr/local/bin/sui --version")
			require.NoError(t, prepareDockerEnvironment(dir, "podman", false, false))
			second, err := os.ReadFile(filepath.Join(dir, "Dockerfile"))
			require.NoError(t, err)
			assert.Equal(t, string(first), string(second))
		})
	}
}
