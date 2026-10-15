package setup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// One-shot, manual validation for the testnet-1.81.1 pin (task 4 of the
// pin-verified-compatible-sui change). Run with:
//
//	EFCTL_PIN_VALIDATE=1 go test ./pkg/setup/ -run TestManualPinValidation -count=1 -v
//
// It prepares the tmp/pin-validation scaffold with the real production code
// path, builds the resulting image, and executes the installed Sui client
// inside it, checking the exact pinned version. Requires Podman, network access,
// and a clean scaffold staged at tmp/pin-validation/builder-scaffold.
// This is a build/version check only; JSON-RPC compatibility requires separate
// live-node checks and the full E2E lifecycle described in docs/sui-release-pin.md.
func TestManualPinValidation(t *testing.T) {
	if os.Getenv("EFCTL_PIN_VALIDATE") == "" {
		t.Skip("set EFCTL_PIN_VALIDATE=1 to run the one-shot pin validation")
	}
	if runtime.GOARCH != "amd64" {
		t.Skipf("validation is amd64-only by agreed scope, running on %s", runtime.GOARCH)
	}

	t.Chdir("../../") // go test cwd is pkg/setup
	dockerDir := filepath.Join("tmp", "pin-validation", "builder-scaffold", "docker")
	if _, err := os.Stat(filepath.Join(dockerDir, "Dockerfile")); err != nil {
		t.Fatalf("tmp/pin-validation scaffold not staged: %v", err)
	}

	if err := prepareDockerEnvironment(dockerDir, "podman", false, false); err != nil {
		t.Fatalf("prepareDockerEnvironment: %v", err)
	}
	dockerfile, err := os.ReadFile(filepath.Join(dockerDir, "Dockerfile"))
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("=== PREPARED DOCKERFILE ===\n%s\n", dockerfile)

	tag := "localhost/efctl-pin-validation:1.81.1"
	t.Cleanup(func() {
		_ = exec.Command("podman", "image", "rm", "--force", tag).Run()
	})
	runStep(t, "BUILD", "podman", "build", "--tag", tag, dockerDir)
	output := runStep(t, "SUI-VERSION", "podman", "run", "--rm", "--pull=never", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--entrypoint", "sui", tag, "--version")
	const expectedVersion = "sui 1.81.1-e04cf9aa915c"
	if got := strings.TrimSpace(output); got != expectedVersion {
		t.Fatalf("unexpected Sui version: got %q, want %q", got, expectedVersion)
	}
}

func runStep(t *testing.T, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s failed: %v\n%s", name, err, output)
	}
	fmt.Printf("=== %s ===\n%s\n", name, output)
	return string(output)
}
