package container

import (
	"fmt"
	"path/filepath"
	"time"
)

// AdditionalBindMount represents a resolved host directory that should be mounted
// into the container under /workspace/mounts/{identifier}.
type AdditionalBindMount struct {
	Source     string
	Identifier string
}

// SuiDevConfig returns the ContainerConfig for the main Sui development node.
func SuiDevConfig(workspace, networkName, engine string, withGraphql bool, pgUser, pgPass, pgDB string, additionalMounts []AdditionalBindMount) ContainerConfig {
	builderScaffold := filepath.Join(workspace, "builder-scaffold")
	worldContracts := filepath.Join(workspace, "world-contracts")

	ports := map[int]int{9000: 9000, 9123: 9123}
	if withGraphql {
		ports[9125] = 9125
	}

	envVars := []string{}
	if withGraphql {
		envVars = append(envVars,
			fmt.Sprintf("SUI_INDEXER_DB_URL=postgres://%s:%s@%s:5432/%s", pgUser, pgPass, ContainerPostgres, pgDB),
			"SUI_GRAPHQL_ENABLED=true",
		)
	}

	usernsMode := ""
	if engine == "podman" {
		usernsMode = "keep-id"
	}

	mounts := []MountDef{
		{Type: "volume", Source: VolumeSuiConfig, Target: "/workspace/.sui"},
		{Type: "bind", Source: builderScaffold, Target: "/workspace/builder-scaffold", SELinux: true},
		{Type: "bind", Source: worldContracts, Target: "/workspace/world-contracts", SELinux: true},
	}
	mounts = append(mounts, additionalBindMountDefs(additionalMounts)...)

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
		UsernsMode:  usernsMode,
	}
}

func additionalBindMountDefs(additionalMounts []AdditionalBindMount) []MountDef {
	if len(additionalMounts) == 0 {
		return nil
	}

	mounts := make([]MountDef, 0, len(additionalMounts))
	for _, mount := range additionalMounts {
		mounts = append(mounts, MountDef{
			Type:    "bind",
			Source:  mount.Source,
			Target:  filepath.ToSlash(filepath.Join("/workspace/mounts", mount.Identifier)),
			SELinux: true,
		})
	}

	return mounts
}

// PostgresConfig returns the ContainerConfig for the PostgreSQL indexer database.
func PostgresConfig(networkName, user, password, dbName string) ContainerConfig {
	return ContainerConfig{
		Name:        ContainerPostgres,
		Image:       ImagePostgres,
		Ports:       map[int]int{5432: 5432},
		Env:         []string{fmt.Sprintf("POSTGRES_USER=%s", user), fmt.Sprintf("POSTGRES_PASSWORD=%s", password), fmt.Sprintf("POSTGRES_DB=%s", dbName)},
		Mounts:      []MountDef{{Type: "volume", Source: VolumePgData, Target: "/var/lib/postgresql/data"}},
		NetworkName: networkName,
		Aliases:     []string{"postgres"},
		Healthcheck: &HealthcheckDef{
			Test:        []string{"CMD-SHELL", fmt.Sprintf("pg_isready -U %s -d %s", user, dbName)},
			Interval:    2 * time.Second,
			Timeout:     3 * time.Second,
			Retries:     30,
			StartPeriod: 10 * time.Second,
		},
	}
}

func FrontendConfig(workspace, networkName, engine string) ContainerConfig {
	usernsMode := ""
	if engine == "podman" {
		usernsMode = "keep-id"
	}

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
		Cmd:         []string{"sh", "-c", "set -e\nnpx pnpm install\nexec npx pnpm dev --host 0.0.0.0"},
		UsernsMode:  usernsMode,
	}
}
