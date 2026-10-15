//go:build integration

package setup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestCustomSuiContainerSelection builds the real patched instructions offline,
// using harmless shell fixtures instead of downloading or trusting Sui artifacts.
func TestCustomSuiContainerSelection(t *testing.T) {
	image := os.Getenv("EFCTL_CUSTOM_SUI_SMOKE_IMAGE")
	if image == "" {
		t.Skip("set EFCTL_CUSTOM_SUI_SMOKE_IMAGE to an immutable cached image ID")
	}
	require.Regexp(t, regexp.MustCompile(`^(sha256:)?[a-f0-9]{64}$`), image)
	engine := os.Getenv("EFCTL_CUSTOM_SUI_SMOKE_ENGINE")
	if engine == "" {
		engine = "podman"
	}
	require.Contains(t, []string{"podman", "docker"}, engine)
	for _, custom := range []bool{false, true} {
		t.Run(fmt.Sprintf("custom=%v", custom), func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(dir, "fixtures"), 0700))
			fixtures := map[string]string{
				"release-sui": "#!/bin/sh\nprintf 'RELEASE-FIXTURE\\n'\n",
				"suiup":       "#!/bin/sh\ncp /opt/fixtures/release-sui /root/.suiup/bin/sui\nchmod +x /root/.suiup/bin/sui\n",
				"curl":        "#!/bin/sh\ncat <<'INSTALL'\nmkdir -p /root/.local/bin /root/.suiup/bin\ncp /opt/fixtures/suiup /root/.local/bin/suiup\nchmod +x /root/.local/bin/suiup\nINSTALL\n",
			}
			for name, content := range fixtures {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "fixtures", name), []byte(content), 0700))
			}
			if custom {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "sui"), []byte("#!/bin/sh\nprintf 'CUSTOM-FIXTURE\\n'\n"), 0700))
			}
			content := "FROM " + image + "\nUSER 0\nCOPY fixtures/ /opt/fixtures/\nCOPY fixtures/curl /usr/local/bin/curl\nRUN chmod +x /usr/local/bin/curl\n" + strings.Replace(testGlobalSuiBlock(), "# Sui CLI via suiup pinned to testnet-1.81.1 (last release with fullnode JSON-RPC)", "# Installer comment intentionally changed", 1) + "\nWORKDIR /workspace\n"
			require.NoError(t, os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(content), 0600))
			require.NoError(t, patchDockerfile(dir))
			tag := fmt.Sprintf("localhost/efctl-sui-selection-%d", time.Now().UnixNano())
			t.Cleanup(func() { _ = exec.Command(engine, "image", "rm", "--force", tag).Run() })
			output, err := exec.Command(engine, "build", "--pull=never", "--network=none", "--tag", tag, dir).CombinedOutput()
			require.NoError(t, err, "offline build: %s", output)
			output, err = exec.Command(engine, "run", "--rm", "--pull=never", "--network=none", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--entrypoint", "sui", tag, "--version").CombinedOutput()
			require.NoError(t, err, "isolated execution: %s", output)
			expected := "RELEASE-FIXTURE"
			if custom {
				expected = "CUSTOM-FIXTURE"
			}
			require.Equal(t, expected, strings.TrimSpace(string(output)))
		})
	}
}
