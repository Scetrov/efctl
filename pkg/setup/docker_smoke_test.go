//go:build integration

package setup

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestDockerSmoke_ProjectSelectedPnpmRequiresImageRuntime exercises the same
// project packageManager dispatch path used by world deployment. It is opt-in
// because it downloads and builds the upstream Sui image.
func TestDockerSmoke_ProjectSelectedPnpmRequiresImageRuntime(t *testing.T) {
	if os.Getenv("EFCTL_DOCKER_SMOKE") != "1" {
		t.Skip("set EFCTL_DOCKER_SMOKE=1 to run the Docker smoke test")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker is not available")
	}

	repoRoot := filepath.Join("..", "..")
	workspace := t.TempDir()
	copyScaffold := exec.Command("cp", "-a", filepath.Join(repoRoot, "builder-scaffold"), workspace)
	if out, err := copyScaffold.CombinedOutput(); err != nil {
		t.Fatalf("copy builder scaffold: %v: %s", err, out)
	}

	dockerDir := filepath.Join(workspace, "builder-scaffold", "docker")
	patchDockerfile(dockerDir)

	image := "efctl-pnpm-runtime-smoke-" + strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-"))
	t.Cleanup(func() {
		_ = exec.Command("docker", "image", "rm", "--force", image).Run()
	})

	build := exec.Command("docker", "build", "--tag", image, dockerDir)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build patched Sui image: %v: %s", err, out)
	}

	command := `mkdir -p /tmp/pnpm-runtime-smoke && cd /tmp/pnpm-runtime-smoke && printf '%s\n' '{"name":"pnpm-runtime-smoke","private":true,"packageManager":"pnpm@11.9.0"}' > package.json && pnpm --version`
	run := exec.Command("docker", "run", "--rm", "--entrypoint", "/bin/bash", image, "-lc", command)
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run project-selected pnpm: %v: %s", err, out)
	}
	if strings.Contains(string(out), "libatomic.so.1") {
		t.Fatalf("project-selected pnpm still has a libatomic loader failure: %s", out)
	}
	if !strings.Contains(string(out), "11.9.0") {
		t.Fatalf("project-selected pnpm did not report version 11.9.0: %s", out)
	}
}
