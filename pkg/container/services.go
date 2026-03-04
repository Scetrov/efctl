package container

import (
	"path/filepath"
	"time"
)

// SuiDevConfig returns the ContainerConfig for the main Sui development node.
func SuiDevConfig(workspace, networkName, engine string, withGraphql bool) ContainerConfig {
	builderScaffold := filepath.Join(workspace, "builder-scaffold")
	worldContracts := filepath.Join(workspace, "world-contracts")

	ports := map[int]int{9000: 9000}
	if withGraphql {
		ports[9125] = 9125
	}

	envVars := []string{}
	if withGraphql {
		envVars = append(envVars,
			"SUI_INDEXER_DB_URL=postgres://sui:sui@"+ContainerPostgres+":5432/sui_indexer",
			"SUI_GRAPHQL_ENABLED=true",
		)
	}

	mounts := []MountDef{
		{Type: "volume", Source: VolumeSuiConfig, Target: "/root/.sui"},
		{Type: "bind", Source: builderScaffold, Target: "/workspace/builder-scaffold", SELinux: true},
		{Type: "bind", Source: worldContracts, Target: "/workspace/world-contracts", SELinux: true},
	}

	// The sui-dev image runs as root (Dockerfile installs sui into /root,
	// sets SUI_CONFIG_DIR=/root/.sui, etc.).  Podman rootless without
	// keep-id maps container UID 0 → host UID, so /root is accessible AND
	// bind-mount writes appear as the host user on the host filesystem.
	// keep-id would map the host UID into the container as a non-root user,
	// breaking access to /root.

	return ContainerConfig{
		Name:        ContainerSuiPlayground,
		Image:       ImageSuiDev,
		Ports:       ports,
		Mounts:      mounts,
		Env:         envVars,
		NetworkName: networkName,
		Aliases:     []string{"sui-dev", ContainerSuiPlayground},
		Tty:         true,
		OpenStdin:   true,
	}
}

// PostgresConfig returns the ContainerConfig for the PostgreSQL indexer database.
func PostgresConfig(networkName string) ContainerConfig {
	return ContainerConfig{
		Name:        ContainerPostgres,
		Image:       ImagePostgres,
		Ports:       map[int]int{5432: 5432},
		Env:         []string{"POSTGRES_USER=sui", "POSTGRES_PASSWORD=sui", "POSTGRES_DB=sui_indexer"},
		Mounts:      []MountDef{{Type: "volume", Source: VolumePgData, Target: "/var/lib/postgresql/data"}},
		NetworkName: networkName,
		Aliases:     []string{"postgres"},
		Healthcheck: &HealthcheckDef{
			Test:        []string{"CMD-SHELL", "pg_isready -U sui -d sui_indexer"},
			Interval:    2 * time.Second,
			Timeout:     3 * time.Second,
			Retries:     30,
			StartPeriod: 10 * time.Second,
		},
	}
}

// FrontendConfig returns the ContainerConfig for the Vite dev-server frontend.
func FrontendConfig(workspace, networkName, engine string) ContainerConfig {
	// The frontend container runs `npm install -g pnpm` which requires root
	// access to /usr/local/lib/node_modules.  Podman rootless without
	// keep-id maps container UID 0 → host UID, so global npm installs work
	// and bind-mount writes still appear as the host user.

	return ContainerConfig{
		Name:  ContainerFrontend,
		Image: ImageNode,
		Ports: map[int]int{5173: 5173},
		Mounts: []MountDef{
			{Type: "bind", Source: workspace, Target: "/workspace", SELinux: true},
			{Type: "volume", Source: VolumeFrontendMods, Target: "/workspace/builder-scaffold/dapps/node_modules"},
		},
		NetworkName: networkName,
		Aliases:     []string{"frontend"},
		WorkingDir:  "/workspace/builder-scaffold/dapps",
		Cmd:         []string{"sh", "-c", "set -e\nnpm install -g pnpm\npnpm install\nexec pnpm dev --host 0.0.0.0"},
	}
}
