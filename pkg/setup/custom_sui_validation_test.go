package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestInvalidCustomSuiInputs(t *testing.T) {
	for _, name := range []string{"empty", "directory", "symlink"} {
		t.Run(name, func(t *testing.T) {
			dir := writeSuiFixture(t, fullDockerfile())
			path := filepath.Join(dir, "sui")
			switch name {
			case "empty":
				require.NoError(t, os.WriteFile(path, nil, 0600))
			case "directory":
				require.NoError(t, os.Mkdir(path, 0700))
			case "symlink":
				target := filepath.Join(t.TempDir(), "binary")
				require.NoError(t, os.WriteFile(target, []byte("fixture"), 0600))
				if err := os.Symlink(target, path); err != nil {
					t.Skipf("symlink unavailable: %v", err)
				}
			}
			original, err := os.ReadFile(filepath.Join(dir, "Dockerfile"))
			require.NoError(t, err)
			require.Error(t, patchDockerfile(dir))
			after, err := os.ReadFile(filepath.Join(dir, "Dockerfile"))
			require.NoError(t, err)
			require.Equal(t, string(original), string(after))
		})
	}
}

func TestUnsupportedSuiLayoutsArePreserved(t *testing.T) {
	for _, content := range []string{
		"FROM ubuntu:24.04\nWORKDIR /workspace-other\n",
		"FROM ubuntu:24.04\nWORKDIR /workspace\nFROM ubuntu:24.04\n",
		"FROM ubuntu:24.04\nRUN suiup install sui --version custom\nWORKDIR /workspace\n",
	} {
		dir := writeSuiFixture(t, content)
		require.Error(t, patchDockerfile(dir))
		after, err := os.ReadFile(filepath.Join(dir, "Dockerfile"))
		require.NoError(t, err)
		require.Equal(t, content, string(after))
	}
}

func TestCustomSuiRemovalRestoresFallback(t *testing.T) {
	dir := writeSuiFixture(t, fullDockerfile())
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sui"), []byte("fixture"), 0600))
	require.NoError(t, patchDockerfile(dir))
	require.NoError(t, os.Remove(filepath.Join(dir, "sui")))
	require.NoError(t, patchDockerfile(dir))
	first, err := os.ReadFile(filepath.Join(dir, "Dockerfile"))
	require.NoError(t, err)
	require.NotContains(t, string(first), "COPY sui")
	require.Equal(t, 1, strings.Count(string(first), "suiup install sui"))
	require.NoError(t, patchDockerfile(dir))
	second, err := os.ReadFile(filepath.Join(dir, "Dockerfile"))
	require.NoError(t, err)
	require.Equal(t, string(first), string(second))
}

func TestStartEnvironmentDoesNotBuildInvalidSuiSetup(t *testing.T) {
	if err := checkRequiredPorts(false, false); err != nil {
		t.Skipf("required ports unavailable: %v", err)
	}
	workspace := t.TempDir()
	dockerDir := filepath.Join(workspace, "builder-scaffold", "docker")
	require.NoError(t, os.MkdirAll(dockerDir, 0700))
	require.NoError(t, os.Mkdir(filepath.Join(workspace, "world-contracts"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(dockerDir, "Dockerfile"), []byte("FROM ubuntu:24.04\n"), 0600))
	c := new(mockContainerClient)
	c.On("GetEngine").Return("podman")
	err := StartEnvironment(c, workspace, false, false)
	require.ErrorContains(t, err, "WORKDIR /workspace")
	c.AssertNotCalled(t, "BuildImage", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	c.AssertNotCalled(t, "RemoveImages", mock.Anything)
	c.AssertNotCalled(t, "CreateNetwork", mock.Anything, mock.Anything)
}
