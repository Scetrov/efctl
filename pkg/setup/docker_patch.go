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
		overrideYaml += postgresServiceYaml()
		overrideYaml += suiDevGraphqlOverridesYaml()
	}

	if withFrontend {
		overrideYaml += frontendServiceYaml()
	}

	overrideYaml += overrideVolumesYaml(withGraphql, withFrontend)

	return overrideYaml
}

func suiDevServiceYaml() string {
	yaml := "  sui-dev:\n"
	return yaml
}

func postgresServiceYaml() string {
	return `  postgres:
    image: docker.io/library/postgres:16
    environment:
      POSTGRES_USER: sui
      POSTGRES_PASSWORD: sui
      POSTGRES_DB: sui_indexer
    volumes:
      - sui-pgdata:/var/lib/postgresql/data
    healthcheck:
      test: "pg_isready -U sui -d sui_indexer"
      interval: 2s
      timeout: 3s
      retries: 30

`
}

func suiDevGraphqlOverridesYaml() string {
	return `  sui-dev:
    environment:
      SUI_INDEXER_DB_URL: postgres://sui:sui@postgres:5432/sui_indexer
      SUI_GRAPHQL_ENABLED: "true"
    depends_on:
      postgres:
        condition: service_healthy
    ports:
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
	dirty := false

	if strings.Contains(content, "- ./world-contracts:/workspace/world-contracts") {
		content = strings.Replace(content, "- ./world-contracts:/workspace/world-contracts", "- ../../world-contracts:/workspace/world-contracts", 1)
		dirty = true
	}

	// Bind-mount the patched entrypoint.sh into the container so it always
	// overrides whatever is baked into the image.  This is the primary defence
	// against Podman (and Docker) build-cache issues that silently preserve an
	// unpatched entrypoint in the image layer.
	const entrypointMount = "./scripts/entrypoint.sh:/workspace/scripts/entrypoint.sh"
	if !strings.Contains(content, entrypointMount) {
		// Insert after the builder-scaffold bind mount line.
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if strings.Contains(line, "../:/workspace/builder-scaffold") {
				indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
				newLine := indent + "- " + entrypointMount
				// Insert after this line
				lines = append(lines[:i+1], append([]string{newLine}, lines[i+1:]...)...)
				dirty = true
				break
			}
		}
		content = strings.Join(lines, "\n")
	}

	// Add :z SELinux labels to bind mounts so rootless Podman can relabel them.
	// The :z suffix is a no-op on non-SELinux systems (e.g. Docker on Ubuntu).
	content, changed := patchComposeBindMountLabels(content)
	if changed {
		dirty = true
	}

	if dirty {
		if err := os.WriteFile(composePath, []byte(content), 0600); err != nil { // #nosec G703 -- path validated by safePath
			log.Printf("patch: failed to write compose.yml: %v", err)
		}
	}
}

// patchComposeBindMountLabels appends :z to bind-mount volumes that reference
// relative host paths (../ or ./) mapped into /workspace. Named volumes are
// left untouched. The function is idempotent — mounts that already carry a
// suffix (:z, :Z, :ro, :rw, etc.) are skipped.
func patchComposeBindMountLabels(content string) (string, bool) {
	// Match lines like "      - ../:/workspace/builder-scaffold"
	// Captures: (indent + "- " + host-relative-path ":" + container-path)(newline)
	// Skips lines where the container-path is already followed by ":something".
	re := regexp.MustCompile(`(?m)(- \.\.?/[^:\s]*:/workspace/[^:\s]+)\n`)
	changed := false
	content = re.ReplaceAllStringFunc(content, func(match string) string {
		trimmed := strings.TrimRight(match, "\n")
		// Already has a mount option suffix (e.g. :z, :ro)
		parts := strings.Split(trimmed, ":")
		if len(parts) > 2 {
			return match
		}
		changed = true
		return trimmed + ":z\n"
	})
	return content, changed
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
	// Safety net: inject a sed command into the Dockerfile that replaces the
	// bind-mount .env.sui path with the internal config-dir path at build time.
	// This uses a literal target path (/root/.sui) rather than ${SUI_CFG} to
	// avoid shell-escaping issues in the RUN layer. It is idempotent — the sed
	// is a no-op when patchEntrypoint has already rewritten the host file.
	const sedSafetyNet = `RUN sed -i 's|ENV_FILE="/workspace/builder-scaffold/docker/.env.sui"|ENV_FILE="/root/.sui/.env.sui"|' /workspace/scripts/entrypoint.sh`
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
		content = strings.ReplaceAll(content, bindMountEnvPath, "/root/.sui/.env.sui")
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
