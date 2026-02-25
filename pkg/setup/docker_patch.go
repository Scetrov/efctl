package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func prepareDockerEnvironment(dockerDir string, withGraphql bool) error {
	overridePath := filepath.Join(dockerDir, "docker-compose.override.yml")

	// 1. Manage docker-compose.override.yml
	if withGraphql {
		overrideYaml := "services:\n"
		overrideYaml += `  postgres:
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

`

		overrideYaml += "  sui-dev:\n    environment:\n"
		overrideYaml += `      SUI_INDEXER_DB_URL: postgres://sui:sui@postgres:5432/sui_indexer
`
		overrideYaml += `      SUI_GRAPHQL_ENABLED: "true"
`

		overrideYaml += `    depends_on:
      postgres:
        condition: service_healthy
`

		overrideYaml += `    ports:
      - "9125:9125"
`

		overrideYaml += "\nvolumes:\n  sui-pgdata:\n"

		if err := os.WriteFile(overridePath, []byte(overrideYaml), 0600); err != nil {
			return fmt.Errorf("failed to write override yaml: %v", err)
		}
	} else {
		// Clean up any existing override file
		_ = os.Remove(overridePath)
	}

	// 1.5 Patch compose.yml
	composePath := filepath.Join(dockerDir, "compose.yml")
	compose, err := os.ReadFile(composePath) // #nosec G304
	if err == nil {
		content := string(compose)
		if strings.Contains(content, "- ./world-contracts:/workspace/world-contracts") {
			content = strings.Replace(content, "- ./world-contracts:/workspace/world-contracts", "- ../../world-contracts:/workspace/world-contracts", 1)
			_ = os.WriteFile(composePath, []byte(content), 0600)
		}
	}

	// 2. Patch Dockerfile
	dockerfilePath := filepath.Join(dockerDir, "Dockerfile")
	dockerfile, err := os.ReadFile(dockerfilePath) // #nosec G304
	if err == nil {
		content := string(dockerfile)
		if !strings.Contains(content, "postgresql-client") {
			content = strings.Replace(content, "dos2unix \\", "dos2unix \\\n    postgresql-client \\", 1)
			_ = os.WriteFile(dockerfilePath, []byte(content), 0600)
		}
	}

	// 3. Patch entrypoint.sh
	entrypointPath := filepath.Join(dockerDir, "scripts", "entrypoint.sh")
	entrypoint, err := os.ReadFile(entrypointPath) // #nosec G304
	if err == nil {
		content := string(entrypoint)

		// Add postgres wait right before starting sui node
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

		// Patch sui start command
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

		// Patch loop wait 30 -> 60 and sleep 2 -> sleep 5
		content = strings.ReplaceAll(content, "for i in $(seq 1 30); do", "for i in $(seq 1 60); do")
		content = strings.ReplaceAll(content, "if [ \"$i\" -eq 30 ]; then", "if [ \"$i\" -eq 60 ]; then")

		if !strings.Contains(content, "RPC responding, waiting for full initialization") {
			rpcWaitScript := `echo "[sui-dev] RPC responding, waiting for full initialization..."
sleep 5
echo "[sui-dev] Node ready."`
			content = strings.ReplaceAll(content, "sleep 2\necho \"[sui-dev] RPC ready.\"", rpcWaitScript)
		}

		_ = os.WriteFile(entrypointPath, []byte(content), 0700) // #nosec G306
	}

	return nil
}
