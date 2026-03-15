package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/pterm/pterm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtensionListCommand(t *testing.T) {
	workspace := t.TempDir()

	// Set up a mock extension
	contractDir := filepath.Join(workspace, "builder-scaffold", "move-contracts", "test_ext")
	require.NoError(t, os.MkdirAll(contractDir, 0750))
	require.NoError(t, os.WriteFile(filepath.Join(contractDir, "Move.toml"), []byte("[package]\nname = \"test_ext\"\n\n[dependencies]\nworld = { local = \"../../../world-contracts/contracts/world\" }\n"), 0600))

	// Backup and set workspacePath
	oldWS := workspacePath
	workspacePath = workspace
	defer func() { workspacePath = oldWS }()

	// Capture stdout and pterm
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w
	pterm.SetDefaultOutput(w)
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
		pterm.SetDefaultOutput(oldStdout)
	}()

	rootCmd.SetArgs([]string{"env", "extension", "list"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "CONTAINER PATH")
	assert.Contains(t, output, "LOCAL PATH")
	assert.Contains(t, output, "builder-scaffold/move-contracts/test_ext")
}

func TestExtensionListCommand_Empty(t *testing.T) {
	workspace := t.TempDir()

	// Backup and set workspacePath
	oldWS := workspacePath
	workspacePath = workspace
	defer func() { workspacePath = oldWS }()

	// Capture stdout and pterm
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w
	pterm.SetDefaultOutput(w)
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
		pterm.SetDefaultOutput(oldStdout)
	}()

	rootCmd.SetArgs([]string{"env", "extension", "list"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "No extensions found in the current workspace.")
}
