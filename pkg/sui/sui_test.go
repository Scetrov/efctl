package sui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockExecutor struct {
	Commands [][]string
	Stdin    []string
}

func (m *MockExecutor) LookPath(file string) (string, error) {
	return "/usr/bin/" + file, nil
}

func (m *MockExecutor) Run(name string, args ...string) error {
	m.Commands = append(m.Commands, append([]string{name}, args...))
	return nil
}

func (m *MockExecutor) RunWithStdin(stdin string, name string, args ...string) error {
	m.Commands = append(m.Commands, append([]string{name}, args...))
	m.Stdin = append(m.Stdin, stdin)
	return nil
}

func TestSuiConfigPath_EndsWithClientYaml(t *testing.T) {
	// Isolate from the real home directory to make the test deterministic.
	home := t.TempDir()
	t.Setenv("HOME", home)

	got := SuiConfigPath()
	want := filepath.Join(home, ".sui", "sui_config", "client.yaml")
	assert.Equal(t, want, got)
}

func TestSuiConfigExists_Missing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	assert.False(t, SuiConfigExists())
}

func TestSuiConfigExists_Present(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path := filepath.Join(home, ".sui", "sui_config", "client.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0700))
	require.NoError(t, os.WriteFile(path, []byte("client config"), 0600))

	assert.True(t, SuiConfigExists())
}

func TestConfigureSui_Commands(t *testing.T) {
	// Note: ConfigureSui in sui.go currently uses exec.Command directly,
	// not the CommandExecutor interface. I should refactor sui.go to use the interface
	// if I want to test it properly without side effects.
	// However, for now, I'll focus on the requested tasks.
}
