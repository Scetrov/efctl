package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareDockerEnvironment(t *testing.T) {
	// Create temp dir
	tmpDir := t.TempDir()
	dockerDir := filepath.Join(tmpDir, "docker")
	scriptsDir := filepath.Join(dockerDir, "scripts")
	os.MkdirAll(scriptsDir, 0755)

	// Create dummy Dockerfile
	dockerfilePath := filepath.Join(dockerDir, "Dockerfile")
	dockerfileContent := `dos2unix \`
	os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644)

	// Create dummy entrypoint.sh
	entrypointPath := filepath.Join(scriptsDir, "entrypoint.sh")
	entrypointContent := `
# ---------- start local node ----------
sui start --with-faucet --force-regenesis &

for i in $(seq 1 30); do
  if [ "$i" -eq 30 ]; then
    echo "Fail"
  fi
done
sleep 2
echo "[sui-dev] RPC ready."
`
	os.WriteFile(entrypointPath, []byte(entrypointContent), 0755)

	// 1. Run patch
	err := prepareDockerEnvironment(dockerDir, true)
	if err != nil {
		t.Fatalf("prepareDockerEnvironment failed: %v", err)
	}

	// 2. Assert docker-compose.override.yml
	overridePath := filepath.Join(dockerDir, "docker-compose.override.yml")
	if _, err := os.Stat(overridePath); os.IsNotExist(err) {
		t.Errorf("docker-compose.override.yml should have been created")
	}

	// Test case 2: withGraphql = false (no sql indexer or graphql)
	err = prepareDockerEnvironment(dockerDir, false)
	if err != nil {
		t.Fatalf("prepareDockerEnvironment failed: %v", err)
	}

	// override file should be deleted
	if _, err := os.Stat(overridePath); !os.IsNotExist(err) {
		t.Errorf("docker-compose.override.yml should have been deleted")
	}

	// 3. Assert Dockerfile
	dockerfileBody, _ := os.ReadFile(dockerfilePath)
	if !strings.Contains(string(dockerfileBody), "postgresql-client \\") {
		t.Errorf("Dockerfile should contain postgresql-client")
	}

	// 4. Assert entrypoint.sh
	entrypointBody, _ := os.ReadFile(entrypointPath)
	bodyStr := string(entrypointBody)
	if !strings.Contains(bodyStr, "wait for postgres") {
		t.Errorf("entrypoint.sh should contain wait for postgres")
	}
	if !strings.Contains(bodyStr, "--with-graphql=0.0.0.0:9125") {
		t.Errorf("entrypoint.sh should contain with-graphql flag")
	}
	if !strings.Contains(bodyStr, "for i in $(seq 1 60); do") {
		t.Errorf("entrypoint.sh should be patched for 60 second wait")
	}
	if strings.Contains(bodyStr, "sleep 2\necho \"[sui-dev] RPC ready.\"") {
		t.Errorf("entrypoint.sh should have replaced sleep 2 block")
	}

	// 5. Test idempotency
	err = prepareDockerEnvironment(dockerDir, true)
	if err != nil {
		t.Fatalf("prepareDockerEnvironment second pass failed: %v", err)
	}
	entrypointBody2, _ := os.ReadFile(entrypointPath)
	if string(entrypointBody) != string(entrypointBody2) {
		t.Errorf("entrypoint patch is not idempotent")
	}
}
