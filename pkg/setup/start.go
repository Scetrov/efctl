package setup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"efctl/pkg/container"
	"efctl/pkg/env"
	"efctl/pkg/ui"
)

const defaultStartupTimeout = 10 * time.Minute

// startupTimeoutFromEnv returns the startup timeout, defaulting to 10 minutes.
// Override with EFCTL_STARTUP_TIMEOUT_SECONDS for CI or slow environments.
func startupTimeoutFromEnv() time.Duration {
	if v := os.Getenv("EFCTL_STARTUP_TIMEOUT_SECONDS"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return defaultStartupTimeout
}

// StartEnvironment builds images and starts containers directly (no compose).
func StartEnvironment(c container.ContainerClient, workspace string, withGraphql bool, withFrontend bool) error {
	ui.Debug.Println(fmt.Sprintf("StartEnvironment: workspace=%s engine=%s graphql=%v frontend=%v", workspace, c.GetEngine(), withGraphql, withFrontend))
	ui.Info.Println("Starting container environment...")

	if err := checkRequiredPorts(withGraphql, withFrontend); err != nil {
		return err
	}

	dockerDir := filepath.Join(workspace, "builder-scaffold", "docker")
	ctx := context.Background()

	// Patch package.json files to allow esbuild scripts.
	patchPnpmDependencies(workspace)

	// Patch Dockerfile and entrypoint.sh from the upstream clone.
	if err := prepareDockerEnvironment(dockerDir, c.GetEngine(), withGraphql, withFrontend); err != nil {
		return err
	}

	// Remove stale images so Podman (and Docker) are forced to rebuild from
	// the patched Dockerfile and entrypoint.
	c.RemoveImages([]string{container.ImageSuiDev, container.ImageSuiDevOld, container.ImageSuiDevOld2})

	// ── Create network & build image ────────────────────────────────
	if err := c.CreateNetwork(ctx, c.NetworkName()); err != nil {
		return fmt.Errorf("failed to create network: %w", err)
	}
	if err := c.BuildImage(ctx, dockerDir, "Dockerfile", container.ImageSuiDev); err != nil {
		return err
	}
	if err := c.CreateVolume(ctx, container.VolumeSuiConfig); err != nil {
		return fmt.Errorf("failed to create sui-config volume: %w", err)
	}

	pgUser := "sui"
	pgDB := "sui_indexer"
	pgPass := os.Getenv("EFCTL_PG_PASSWORD")
	if pgPass == "" {
		var err error
		pgPass, err = env.GenerateRandomPassword(16)
		if err != nil {
			return fmt.Errorf("failed to generate postgres password: %w", err)
		}
	}

	// ── PostgreSQL (if graphql) ─────────────────────────────────────
	if withGraphql {
		if err := startPostgres(c, ctx, pgUser, pgPass, pgDB); err != nil {
			return err
		}
	}

	// ── Sui dev container ───────────────────────────────────────────
	if err := startSuiDev(c, ctx, workspace, dockerDir, withGraphql, pgUser, pgPass, pgDB); err != nil {
		return err
	}

	// ── Frontend (if requested) ─────────────────────────────────────
	if withFrontend {
		if err := startFrontend(c, ctx, workspace); err != nil {
			return err
		}
	}

	return nil
}

func startPostgres(c container.ContainerClient, ctx context.Context, user, pass, db string) error {
	networkName := c.NetworkName()

	if err := c.CreateVolume(ctx, container.VolumePgData); err != nil {
		return fmt.Errorf("failed to create pgdata volume: %w", err)
	}

	pgCfg := container.PostgresConfig(networkName, user, pass, db)
	if err := c.CreateContainer(ctx, pgCfg); err != nil {
		return fmt.Errorf("failed to create postgres container: %w", err)
	}
	if err := c.StartContainer(ctx, container.ContainerPostgres); err != nil {
		return fmt.Errorf("failed to start postgres container: %w", err)
	}
	if err := c.WaitHealthy(ctx, container.ContainerPostgres, 60*time.Second); err != nil {
		return fmt.Errorf("postgres did not become healthy: %w", err)
	}
	return nil
}

func startSuiDev(c container.ContainerClient, ctx context.Context, workspace, dockerDir string, withGraphql bool, pgUser, pgPass, pgDB string) error {
	networkName := c.NetworkName()

	suiCfg := container.SuiDevConfig(workspace, networkName, c.GetEngine(), withGraphql, pgUser, pgPass, pgDB)
	if err := c.CreateContainer(ctx, suiCfg); err != nil {
		return fmt.Errorf("failed to create sui-playground container: %w", err)
	}
	if err := c.StartContainer(ctx, container.ContainerSuiPlayground); err != nil {
		return fmt.Errorf("failed to start sui-playground container: %w", err)
	}

	// Give the container a moment to start, then verify it is still running
	// before entering the (potentially long) log-wait loop.
	time.Sleep(3 * time.Second)
	if !c.ContainerRunning(container.ContainerSuiPlayground) {
		exitCode, _ := c.ContainerExitCode(container.ContainerSuiPlayground)
		lastLogs := c.ContainerLogs(container.ContainerSuiPlayground, 30)
		return fmt.Errorf("%s exited immediately after launch (exit code %d).\n\nLast 30 lines of container logs:\n%s",
			container.ContainerSuiPlayground, exitCode, lastLogs)
	}

	startupTimeout := startupTimeoutFromEnv()
	logCtx, cancel := context.WithTimeout(ctx, startupTimeout)
	defer cancel()

	if err := c.WaitForLogs(logCtx, container.ContainerSuiPlayground, container.ContainerLogReadyCtx); err != nil {
		// On timeout or failure, capture container logs for diagnostics
		lastLogs := c.ContainerLogs(container.ContainerSuiPlayground, 50)
		return fmt.Errorf("%w\n\nLast 50 lines of container logs:\n%s", err, lastLogs)
	}

	// The container generates its internal .env.sui. We must extract it
	// manually so host scripts can use it, avoiding rootless bind mount
	// permission issues.
	var output string
	var err error
	for i := 0; i < 15; i++ {
		output, err = c.ExecCapture(container.ContainerSuiPlayground, []string{"cat", "/workspace/.sui/.env.sui"})
		if err == nil && len(strings.TrimSpace(output)) > 0 {
			break
		}
		time.Sleep(1 * time.Second)
	}

	if err != nil || len(strings.TrimSpace(output)) == 0 {
		ui.Warn.Println("Failed to extract .env.sui from container. Tests may fail to run locally.")
	} else {
		envPath := filepath.Join(dockerDir, ".env.sui")
		if writeErr := os.WriteFile(envPath, []byte(output), 0600); writeErr != nil {
			ui.Warn.Println(fmt.Sprintf("Failed to write extracted .env.sui to host: %v", writeErr))
		}
	}
	return nil
}

func checkRequiredPorts(withGraphql bool, withFrontend bool) error {
	if !env.IsPortAvailable(9000) {
		return fmt.Errorf("port 9000 is already in use")
	}
	if withGraphql {
		if !env.IsPortAvailable(8000) {
			return fmt.Errorf("port 8000 (GraphQL) is already in use")
		}
		if !env.IsPortAvailable(5432) {
			return fmt.Errorf("port 5432 (PostgreSQL) is already in use")
		}
	}
	if withFrontend {
		if !env.IsPortAvailable(5173) {
			return fmt.Errorf("port 5173 (Frontend) is already in use")
		}
	}
	return nil
}

func startFrontend(c container.ContainerClient, ctx context.Context, workspace string) error {
	ui.Info.Println("Starting frontend dApp...")

	networkName := c.NetworkName()

	if err := c.CreateVolume(ctx, container.VolumeFrontendMods); err != nil {
		return fmt.Errorf("failed to create frontend modules volume: %w", err)
	}

	feCfg := container.FrontendConfig(workspace, networkName, c.GetEngine())
	if err := c.CreateContainer(ctx, feCfg); err != nil {
		return fmt.Errorf("failed to create frontend container: %w", err)
	}
	if err := c.StartContainer(ctx, container.ContainerFrontend); err != nil {
		return fmt.Errorf("failed to start frontend container: %w", err)
	}

	// Give the container a moment to start (or crash)
	time.Sleep(3 * time.Second)

	if !c.ContainerRunning(container.ContainerFrontend) {
		logsOut := c.ContainerLogs(container.ContainerFrontend, 30)
		if logsOut == "" || strings.Contains(logsOut, "could not retrieve") {
			logsOut = "(no logs available)"
		}
		ui.Warn.Println("Frontend container exited immediately. Logs:")
		fmt.Println(logsOut)
		return fmt.Errorf("frontend container is not running — check the logs above for details")
	}
	return nil
}
