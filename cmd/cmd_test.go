package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── version output ─────────────────────────────────────────────────

func TestVersionCommand(t *testing.T) {
	// Set version vars
	Version = "1.2.3"
	CommitSHA = "abc1234"
	BuildDate = "2024-01-01"

	// The version command uses fmt.Printf (stdout), not cmd.OutOrStdout()
	// so we capture via os.Pipe
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs([]string{"version"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "1.2.3")
	assert.Contains(t, output, "abc1234")
	assert.Contains(t, output, "2024-01-01")
	assert.Contains(t, output, runtime.GOOS)
	assert.Contains(t, output, runtime.GOARCH)
}

// ── GetRootCmd ─────────────────────────────────────────────────────

func TestGetRootCmd(t *testing.T) {
	cmd := GetRootCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "efctl", cmd.Use)
}

// ── safeScriptNameRe ───────────────────────────────────────────────

func TestSafeScriptNameRe_Valid(t *testing.T) {
	valid := []string{
		"deploy",
		"my-script",
		"my_script",
		"path/to/script",
		"version.1.0",
		"pnpm",
	}
	for _, s := range valid {
		assert.True(t, safeScriptNameRe.MatchString(s), "expected %q to be valid", s)
	}
}

func TestSafeScriptNameRe_Invalid(t *testing.T) {
	invalid := []string{
		"script; rm -rf /",
		"script`whoami`",
		"script $(cmd)",
		"script | cat",
		"script && evil",
		"script name",
		"",
	}
	for _, s := range invalid {
		assert.False(t, safeScriptNameRe.MatchString(s), "expected %q to be invalid", s)
	}
}

// ── fetchExpectedChecksum ──────────────────────────────────────────

func TestFetchExpectedChecksum_Found(t *testing.T) {
	hash := strings.Repeat("ab", 32) // exactly 64 hex chars
	body := hash + "  efctl-linux-amd64\n" +
		strings.Repeat("cd", 32) + "  efctl-darwin-arm64\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	got, err := fetchExpectedChecksum(srv.URL, "efctl-linux-amd64")
	require.NoError(t, err)
	assert.Equal(t, hash, got)
}

func TestFetchExpectedChecksum_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, strings.Repeat("ab", 32)+"  efctl-linux-amd64\n")
	}))
	defer srv.Close()

	_, err := fetchExpectedChecksum(srv.URL, "efctl-windows-amd64")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no checksum found")
}

func TestFetchExpectedChecksum_InvalidHash(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "shorthash  efctl-linux-amd64\n")
	}))
	defer srv.Close()

	_, err := fetchExpectedChecksum(srv.URL, "efctl-linux-amd64")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid checksum length")
}

func TestFetchExpectedChecksum_HTTP404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := fetchExpectedChecksum(srv.URL, "efctl-linux-amd64")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 404")
}

// ── extractAdmin ───────────────────────────────────────────────────

func TestExtractAdmin_Found(t *testing.T) {
	ws := t.TempDir()
	envDir := filepath.Join(ws, "world-contracts")
	require.NoError(t, os.MkdirAll(envDir, 0750))
	envFile := filepath.Join(envDir, ".env")
	require.NoError(t, os.WriteFile(envFile, []byte("ADMIN_ADDRESS=0xaabbccdd1122\nOTHER=val\n"), 0600))

	assert.Equal(t, "0xaabbccdd1122", extractAdmin(ws))
}

func TestExtractAdmin_NotFound(t *testing.T) {
	ws := t.TempDir()
	envDir := filepath.Join(ws, "world-contracts")
	require.NoError(t, os.MkdirAll(envDir, 0750))
	require.NoError(t, os.WriteFile(filepath.Join(envDir, ".env"), []byte("FOO=bar\n"), 0600))

	assert.Equal(t, "Not Found", extractAdmin(ws))
}

func TestExtractAdmin_MissingFile(t *testing.T) {
	assert.Equal(t, "Unknown", extractAdmin(t.TempDir()))
}

// ── extractEnvVars ─────────────────────────────────────────────────

func TestExtractEnvVars(t *testing.T) {
	ws := t.TempDir()
	envDir := filepath.Join(ws, "world-contracts")
	require.NoError(t, os.MkdirAll(envDir, 0750))
	content := "# comment\nFOO=bar\nBAZ=qux\nEMPTY=\n"
	require.NoError(t, os.WriteFile(filepath.Join(envDir, ".env"), []byte(content), 0600))

	vars := extractEnvVars(ws)
	assert.Equal(t, "bar", vars["FOO"])
	assert.Equal(t, "qux", vars["BAZ"])
	// EMPTY= has an empty value, so it should not be in the map (parts[1] != "")
	_, hasEmpty := vars["EMPTY"]
	assert.False(t, hasEmpty)
	_, hasComment := vars["# comment"]
	assert.False(t, hasComment)
}

func TestExtractEnvVars_MissingFile(t *testing.T) {
	vars := extractEnvVars(t.TempDir())
	assert.Empty(t, vars)
}

// ── extractWorldObjects ────────────────────────────────────────────

func TestExtractWorldObjects(t *testing.T) {
	ws := t.TempDir()
	dir := filepath.Join(ws, "world-contracts", "deployments", "localnet")
	require.NoError(t, os.MkdirAll(dir, 0750))
	data := map[string]interface{}{
		"world": map[string]interface{}{
			"packageId":   "0xPKG",
			"governorCap": "0xGOV",
			"adminAcl":    "0xACL",
		},
	}
	b, _ := json.Marshal(data)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "extracted-object-ids.json"), b, 0600))

	objs, pkgID := extractWorldObjects(ws)
	assert.Equal(t, "0xPKG", pkgID)
	assert.Equal(t, "0xGOV", objs["governorCap"])
	assert.Equal(t, "0xACL", objs["adminAcl"])
	_, hasPkg := objs["packageId"]
	assert.False(t, hasPkg, "packageId should not be in objs map")
}

func TestExtractWorldObjects_MissingFile(t *testing.T) {
	objs, pkgID := extractWorldObjects(t.TempDir())
	assert.Empty(t, objs)
	assert.Empty(t, pkgID)
}

// ── model helpers ──────────────────────────────────────────────────

func TestMaxLogScroll(t *testing.T) {
	m := model{height: 40, logs: make([]string, 100)}
	scroll := m.maxLogScroll()
	assert.GreaterOrEqual(t, scroll, 0)
	assert.LessOrEqual(t, scroll, len(m.logs))
}

func TestMaxLogScroll_FewLogs(t *testing.T) {
	m := model{height: 40, logs: []string{"line1"}}
	assert.Equal(t, 0, m.maxLogScroll())
}

func TestPanelWidths(t *testing.T) {
	m := model{width: 100}
	left, right, logW := m.panelWidths()
	assert.Greater(t, left, 0)
	assert.Greater(t, right, 0)
	assert.Greater(t, logW, 0)
	// left + right should roughly total width minus borders
	assert.InDelta(t, 100-3, left+right, 2)
}

func TestHexShortener(t *testing.T) {
	m := model{width: 80}
	shorten := m.hexShortener()

	// Short hex should not be shortened
	assert.Equal(t, "0x1234", shorten("0x1234"))

	// Long hex should be abbreviated
	long := "0x" + strings.Repeat("a", 64)
	shortened := shorten(long)
	assert.True(t, strings.HasPrefix(shortened, "0x"))
	assert.Contains(t, shortened, "…")
	assert.Less(t, len(shortened), len(long))
}

func TestHexShortener_NarrowTerminal(t *testing.T) {
	m := model{width: 20}
	shorten := m.hexShortener()

	long := "0x" + strings.Repeat("b", 64)
	shortened := shorten(long)
	assert.Contains(t, shortened, "…")
}

func TestRenderContainerContent_AlwaysPadsWithBlankLines(t *testing.T) {
	m := model{
		suiStat:    containerStat{Status: "Running", CPU: "1%", Mem: "10MB / 1GB"},
		pgStat:     containerStat{Status: "Stopped", CPU: "-", Mem: "-"},
		feStat:     containerStat{Status: "Stopped", CPU: "-", Mem: "-"},
		graphqlOn:  false,
		frontendOn: false,
	}

	out := m.renderContainerContent()
	assert.True(t, strings.HasPrefix(out, "\n"), "services content should start with a blank line")
	assert.True(t, strings.HasSuffix(out, "\n\n"), "services content should end with a blank line")
	assert.Contains(t, out, "sui-playground")
	assert.NotContains(t, out, "database")
	assert.NotContains(t, out, "frontend")
}

func TestRenderContainerContent_ShowsFrontendWhenContainerRunning(t *testing.T) {
	m := model{
		suiStat:    containerStat{Status: "Running", CPU: "1%", Mem: "10MB / 1GB"},
		pgStat:     containerStat{Status: "Running", CPU: "1%", Mem: "10MB / 1GB"},
		feStat:     containerStat{Status: "Running", CPU: "1%", Mem: "10MB / 1GB"},
		graphqlOn:  true,
		frontendOn: false,
	}

	out := m.renderContainerContent()
	assert.Contains(t, out, "frontend")
}

func TestRenderContainerContent_HidesRestartShortcutsByDefault(t *testing.T) {
	m := model{
		suiStat:    containerStat{Status: "Running", CPU: "1%", Mem: "10MB / 1GB"},
		pgStat:     containerStat{Status: "Running", CPU: "1%", Mem: "10MB / 1GB"},
		feStat:     containerStat{Status: "Stopped", CPU: "-", Mem: "-"},
		graphqlOn:  true,
		frontendOn: false,
		restarting: false,
	}

	out := m.renderContainerContent()
	assert.Contains(t, out, "database")
	assert.NotContains(t, out, "[b]")
	assert.NotContains(t, out, "[f]")
}

func TestRenderContainerContent_ShowsRestartShortcutsWhenRestarting(t *testing.T) {
	m := model{
		suiStat:    containerStat{Status: "Running", CPU: "1%", Mem: "10MB / 1GB"},
		pgStat:     containerStat{Status: "Running", CPU: "1%", Mem: "10MB / 1GB"},
		feStat:     containerStat{Status: "Running", CPU: "1%", Mem: "10MB / 1GB"},
		graphqlOn:  true,
		frontendOn: true,
		restarting: true,
	}

	out := m.renderContainerContent()
	assert.GreaterOrEqual(t, strings.Count(out, "[b]"), 2)
	assert.Contains(t, out, "[f]")
}

func TestServiceRowCount_ThreeServicesWhenRunning(t *testing.T) {
	m := model{
		suiStat:    containerStat{Status: "Running"},
		pgStat:     containerStat{Status: "Running"},
		feStat:     containerStat{Status: "Running"},
		graphqlOn:  false,
		frontendOn: false,
	}

	assert.Equal(t, 3, m.serviceRowCount())
}

func TestServiceRowCount_BaseServiceOnly(t *testing.T) {
	m := model{
		suiStat:    containerStat{Status: "Running"},
		pgStat:     containerStat{Status: "Stopped"},
		feStat:     containerStat{Status: "Stopped"},
		graphqlOn:  false,
		frontendOn: false,
	}

	assert.Equal(t, 1, m.serviceRowCount())
}

func TestFitEnvLines_NoOverflow(t *testing.T) {
	m := model{}
	lines := []string{"line1", "line2"}
	rendered := padLines(lines, 2, 20)

	got, overflow := m.fitEnvLines(rendered, 4, 20)
	assert.Equal(t, 0, overflow)
	assert.Len(t, got, 4)
	assert.Contains(t, got[0], "line1")
	assert.Contains(t, got[1], "line2")
}

func TestFitEnvLines_OverflowAddsWarning(t *testing.T) {
	m := model{}
	rendered := []string{"l1", "l2", "l3", "l4", "l5"}

	got, overflow := m.fitEnvLines(rendered, 3, 24)
	assert.Equal(t, 2, overflow)
	assert.Len(t, got, 3)
	assert.Contains(t, got[2], "Overflow: +2 lines")
}

// ── command tree structure ─────────────────────────────────────────

func TestCommandTree(t *testing.T) {
	root := GetRootCmd()

	// Verify key subcommands exist
	names := make(map[string]bool)
	for _, c := range root.Commands() {
		names[c.Name()] = true
	}

	assert.True(t, names["version"], "version command should exist")
	assert.True(t, names["update"], "update command should exist")
	assert.True(t, names["env"], "env command should exist")
	assert.True(t, names["graphql"], "graphql command should exist")
}

func TestEnvUpFlagDefaultsEnabled(t *testing.T) {
	graphqlFlag := envUpCmd.Flags().Lookup("with-graphql")
	frontendFlag := envUpCmd.Flags().Lookup("with-frontend")
	require.NotNil(t, graphqlFlag)
	require.NotNil(t, frontendFlag)
	assert.Equal(t, "true", graphqlFlag.DefValue)
	assert.Equal(t, "true", frontendFlag.DefValue)
}
