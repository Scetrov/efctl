package setup

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// safePath constructs and validates a file path under the given base directory,
// returning an error if the resolved path would escape the base via directory traversal.
func safePath(base string, elem ...string) (string, error) {
	parts := append([]string{base}, elem...)
	p := filepath.Join(parts...)

	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("resolve base: %w", err)
	}
	absP, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}

	// Ensure the path is within the base directory.
	if absP != absBase && !strings.HasPrefix(absP, absBase+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes base directory %q", p, base)
	}

	return absP, nil
}

func prepareDockerEnvironment(dockerDir string, withGraphql bool, withFrontend bool) error {
	overridePath := filepath.Join(dockerDir, "docker-compose.override.yml")

	// 1. Manage docker-compose.override.yml
	if withGraphql || withFrontend {
		overrideYaml := buildOverrideYaml(withGraphql, withFrontend)
		if err := os.WriteFile(overridePath, []byte(overrideYaml), 0600); err != nil {
			return fmt.Errorf("failed to write override yaml: %v", err)
		}
	} else {
		if err := os.Remove(overridePath); err != nil && !os.IsNotExist(err) {
			log.Printf("cleanup: failed to remove override file: %v", err)
		}
	}

	// 1.5 Patch compose.yml
	patchComposeYml(dockerDir)

	// 2. Patch Dockerfile
	patchDockerfile(dockerDir)

	// 3. Patch entrypoint.sh
	patchEntrypoint(dockerDir)

	return nil
}

func buildOverrideYaml(withGraphql bool, withFrontend bool) string {
	overrideYaml := "services:\n"

	if withGraphql {
		overrideYaml += graphqlServicesYaml()
	}

	if withFrontend {
		overrideYaml += frontendServiceYaml()
	}

	overrideYaml += overrideVolumesYaml(withGraphql, withFrontend)

	return overrideYaml
}

func graphqlServicesYaml() string {
	return `  postgres:
    image: docker.io/library/postgres:16
    environment:
      POSTGRES_USER: sui
      POSTGRES_PASSWORD: sui
      POSTGRES_DB: sui_indexer
    volumes:
      - sui-pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U sui -d sui_indexer"]
      interval: 2s
      timeout: 3s
      retries: 30

` + "  sui-dev:\n    environment:\n" +
		`      SUI_INDEXER_DB_URL: postgres://sui:sui@postgres:5432/sui_indexer
` + `      SUI_GRAPHQL_ENABLED: "true"
` + `    depends_on:
      postgres:
        condition: service_healthy
` + `    ports:
      - "9125:9125"
`
}

func frontendServiceYaml() string {
	return `
  frontend:
    image: docker.io/library/node:24-slim
    ports:
      - "5173:5173"
    volumes:
      - ../../:/workspace
      - frontend-node-modules:/workspace/builder-scaffold/dapps/node_modules
    working_dir: /workspace/builder-scaffold/dapps
    command:
      - sh
      - -c
      - |
        set -e
        npm install -g pnpm
        pnpm install
        exec pnpm dev --host 0.0.0.0
`
}

func overrideVolumesYaml(withGraphql bool, withFrontend bool) string {
	if !withGraphql && !withFrontend {
		return ""
	}
	result := "\nvolumes:\n"
	if withGraphql {
		result += "  sui-pgdata:\n"
	} else {
		// volumes: header already added above
	}
	if withFrontend {
		result += "  frontend-node-modules:\n"
	}
	return result
}

func patchComposeYml(dockerDir string) {
	composePath, err := safePath(dockerDir, "compose.yml")
	if err != nil {
		log.Printf("patch: invalid compose path: %v", err)
		return
	}
	compose, err := os.ReadFile(composePath) // #nosec G304 -- path validated by safePath
	if err != nil {
		return
	}
	content := string(compose)
	if strings.Contains(content, "- ./world-contracts:/workspace/world-contracts") {
		content = strings.Replace(content, "- ./world-contracts:/workspace/world-contracts", "- ../../world-contracts:/workspace/world-contracts", 1)
		if err := os.WriteFile(composePath, []byte(content), 0600); err != nil { // #nosec G703 -- path validated by safePath
			log.Printf("patch: failed to write compose.yml: %v", err)
		}
	}
}

func patchDockerfile(dockerDir string) {
	dockerfilePath, err := safePath(dockerDir, "Dockerfile")
	if err != nil {
		log.Printf("patch: invalid Dockerfile path: %v", err)
		return
	}
	dockerfile, err := os.ReadFile(dockerfilePath) // #nosec G304 -- path validated by safePath
	if err != nil {
		return
	}
	content := string(dockerfile)
	if !strings.Contains(content, "postgresql-client") {
		content = strings.Replace(content, "dos2unix \\", "dos2unix \\\n    postgresql-client \\", 1)
		if err := os.WriteFile(dockerfilePath, []byte(content), 0600); err != nil { // #nosec G703 -- path validated by safePath
			log.Printf("patch: failed to write Dockerfile: %v", err)
		}
	}
}

func patchEntrypoint(dockerDir string) {
	entrypointPath, err := safePath(dockerDir, "scripts", "entrypoint.sh")
	if err != nil {
		log.Printf("patch: invalid entrypoint path: %v", err)
		return
	}
	entrypoint, err := os.ReadFile(entrypointPath) // #nosec G304 -- path validated by safePath
	if err != nil {
		return
	}
	content := string(entrypoint)

	content = patchEntrypointPostgresWait(content)
	content = patchEntrypointSuiStart(content)
	content = patchEntrypointLoopTimings(content)

	if err := os.WriteFile(entrypointPath, []byte(content), 0700); err != nil { // #nosec G302 G306 G703 -- entrypoint.sh must be executable; path validated by safePath
		log.Printf("patch: failed to write entrypoint.sh: %v", err)
	}
}

func patchEntrypointPostgresWait(content string) string {
	postgresWaitScript := `
# ---------- wait for postgres ----------
if [ -n "${SUI_INDEXER_DB_URL:-}" ]; then
  echo "[sui-dev] Waiting for Postgres to be ready..."
  POSTGRES_READY=0
  for i in {1..60}; do
    if pg_isready -d "$SUI_INDEXER_DB_URL" >/dev/null 2>&1; then
      echo "[sui-dev] Postgres is ready."
      POSTGRES_READY=1
      break
    fi
    sleep 1
  done

  if [ "$POSTGRES_READY" -ne 1 ]; then
    echo "[sui-dev] ERROR: Postgres did not become ready" >&2
    exit 1
  fi

  # Reset database to match --force-regenesis behavior
  echo "[sui-dev] Resetting indexer database to match fresh blockchain state..."
  DB_NAME=$(echo "$SUI_INDEXER_DB_URL" | sed -n 's|.*/\([^/?]*\).*|\1|p')
  DB_BASE_URL=$(echo "$SUI_INDEXER_DB_URL" | sed 's|/[^/]*$|/postgres|')

  psql "$DB_BASE_URL" -c "DROP DATABASE IF EXISTS $DB_NAME;" 2>/dev/null || true
  psql "$DB_BASE_URL" -c "CREATE DATABASE $DB_NAME;" 2>/dev/null
  echo "[sui-dev] Indexer database reset complete."
fi

# ---------- start local node ----------`

	if !strings.Contains(content, "wait for postgres") {
		content = strings.Replace(content, "# ---------- start local node ----------", postgresWaitScript, 1)
	}
	return content
}

func patchEntrypointSuiStart(content string) string {
	suiStartScript := `SUI_START_ARGS="--with-faucet --force-regenesis"
if [ -n "${SUI_INDEXER_DB_URL:-}" ]; then
  SUI_START_ARGS="$SUI_START_ARGS --with-indexer=$SUI_INDEXER_DB_URL"
fi
if [ "${SUI_GRAPHQL_ENABLED:-}" = "true" ]; then
  SUI_START_ARGS="$SUI_START_ARGS --with-graphql=0.0.0.0:9125"
fi
sui start $SUI_START_ARGS &`
	if !strings.Contains(content, "SUI_START_ARGS") {
		content = strings.Replace(content, "sui start --with-faucet --force-regenesis &", suiStartScript, 1)
	}
	return content
}

func patchEntrypointLoopTimings(content string) string {
	content = strings.ReplaceAll(content, "for i in $(seq 1 30); do", "for i in $(seq 1 60); do")
	content = strings.ReplaceAll(content, "if [ \"$i\" -eq 30 ]; then", "if [ \"$i\" -eq 60 ]; then")

	if !strings.Contains(content, "RPC responding, waiting for full initialization") {
		rpcWaitScript := `echo "[sui-dev] RPC responding, waiting for full initialization..."
sleep 5
echo "[sui-dev] Node ready."`
		content = strings.ReplaceAll(content, "sleep 2\necho \"[sui-dev] RPC ready.\"", rpcWaitScript)
	}
	return content
}
