package setup

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
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

func prepareDockerEnvironment(dockerDir string, engine string, withGraphql bool, withFrontend bool) error {
	// Clean up any stale compose override files from older efctl versions.
	overridePath := filepath.Join(dockerDir, "docker-compose.override.yml")
	if err := os.Remove(overridePath); err != nil && !os.IsNotExist(err) {
		log.Printf("cleanup: failed to remove legacy override file: %v", err)
	}

	// Patch Dockerfile (add postgresql-client, sed safety net)
	patchDockerfile(dockerDir)

	// Patch entrypoint.sh (env path, postgres wait, sui start args, loop timings)
	patchEntrypoint(dockerDir)

	return nil
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
	}
	if strings.Contains(content, "ENV SUI_CONFIG_DIR=/root/.sui") {
		content = strings.Replace(content, "ENV SUI_CONFIG_DIR=/root/.sui", "ENV SUI_CONFIG_DIR=/workspace/.sui", 1)
	}
	// Safety net: inject a sed command into the Dockerfile that globally
	// replaces the bind-mount .env.sui path with the internal config-dir path
	// at build time.  This uses a broad global replacement (not just the
	// ENV_FILE assignment) so it survives even when podman-compose reuses a
	// cached COPY layer that still contains the unpatched upstream file.
	// The command is idempotent — a no-op when the path is already correct.
	const sedSafetyNet = `RUN sed -i 's|/workspace/builder-scaffold/docker/\.env\.sui|/workspace/.sui/.env.sui|g' /workspace/scripts/entrypoint.sh`

	// Remove a narrower sed variant injected by earlier versions of efctl,
	// so we don't accumulate duplicate (and less effective) RUN layers.
	const oldSed = `RUN sed -i 's|ENV_FILE="/workspace/builder-scaffold/docker/.env.sui"|ENV_FILE="/root/.sui/.env.sui"|' /workspace/scripts/entrypoint.sh`
	if strings.Contains(content, oldSed) {
		content = strings.Replace(content, oldSed+"\n", "", 1)
		content = strings.Replace(content, oldSed, "", 1)
	}

	// Move sui and suiup to /usr/local/bin so they are globally accessible
	// for non-root users (critical for Podman keep-id).
	const globalSui = `RUN SUI_PATH=$(command -v sui) && SUIUP_PATH=$(command -v suiup) && \
    mv "$SUI_PATH" /usr/local/bin/sui && \
    mv "$SUIUP_PATH" /usr/local/bin/suiup && \
    chmod +x /usr/local/bin/sui /usr/local/bin/suiup`

	if !strings.Contains(content, globalSui) {
		content = strings.Replace(content,
			`&& sui --version`,
			`&& sui --version`+"\n"+globalSui,
			1)
	}

	if !strings.Contains(content, sedSafetyNet) {
		content = strings.Replace(content,
			`RUN dos2unix /workspace/scripts/*.sh && chmod +x /workspace/scripts/*.sh`,
			`RUN dos2unix /workspace/scripts/*.sh && chmod +x /workspace/scripts/*.sh`+"\n"+sedSafetyNet,
			1)
	}
	if err := os.WriteFile(dockerfilePath, []byte(content), 0600); err != nil { // #nosec G703 -- path validated by safePath
		log.Printf("patch: failed to write Dockerfile: %v", err)
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

	content = patchEntrypointEnvPath(content)
	content = patchEntrypointPostgresWait(content)
	content = patchEntrypointSuiStart(content)
	content = patchEntrypointLoopTimings(content)

	// Post-patch validation: ensure the bind-mount .env.sui path was fully
	// eliminated. If any patch above silently failed (e.g. upstream changed
	// quoting), force-replace the literal path as a last resort.
	const bindMountEnvPath = "/workspace/builder-scaffold/docker/.env.sui"
	if strings.Contains(content, bindMountEnvPath) {
		log.Printf("patch: entrypoint.sh still references bind-mount path for .env.sui — applying forced replacement")
		content = strings.ReplaceAll(content, bindMountEnvPath, "/workspace/.sui/.env.sui")
	}

	if err := os.WriteFile(entrypointPath, []byte(content), 0700); err != nil { // #nosec G302 G306 G703 -- entrypoint.sh must be executable; path validated by safePath
		log.Printf("patch: failed to write entrypoint.sh: %v", err)
	}
}

// envFileBindMountRe matches ENV_FILE assignments pointing at the bind-mount
// path, regardless of quoting style (double-quoted, single-quoted, unquoted)
// and optional leading "export ".
var envFileBindMountRe = regexp.MustCompile(
	`(?m)^(\s*(?:export\s+)?)ENV_FILE=['"]?/workspace/builder-scaffold/docker/\.env\.sui['"]?`,
)

func patchEntrypointEnvPath(content string) string {
	replacement := `ENV_FILE="${SUI_CFG}/.env.sui"`

	// Already patched — nothing to do.
	if strings.Contains(content, replacement) {
		return content
	}

	// Try regex-based replacement (handles quoting/export variants).
	// We use ReplaceAllStringFunc to avoid $ in the replacement being
	// interpreted as a group backreference by the regexp engine.
	if envFileBindMountRe.MatchString(content) {
		content = envFileBindMountRe.ReplaceAllStringFunc(content, func(match string) string {
			prefix := envFileBindMountRe.FindStringSubmatch(match)[1]
			return prefix + replacement
		})
	}

	return content
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
      cat > "$CLIENT_YAML" << EOF
---
keystore:
  File: $KEYSTORE
envs:
  - alias: localnet
    rpc: "http://127.0.0.1:9000"
  - alias: ef-localhost
    rpc: "http://127.0.0.1:9000"
    faucet: "http://127.0.0.1:9123"
  - alias: testnet
    rpc: "https://fullnode.testnet.sui.io"
active_env: ef-localhost
active_address: ~
EOF

      printf 'y\n' | sui client switch --env ef-localhost 2>/dev/null || true
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
