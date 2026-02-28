package setup

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"efctl/pkg/container"
	"efctl/pkg/env"
	"efctl/pkg/ui"
)

// StartEnvironment builds and starts the docker-compose environment
func StartEnvironment(c container.ContainerClient, workspace string, withGraphql bool, withFrontend bool) error {
	ui.Info.Println("Starting container environment...")

	if err := checkRequiredPorts(withGraphql, withFrontend); err != nil {
		return err
	}

	dockerDir := filepath.Join(workspace, "builder-scaffold", "docker")

	if err := prepareDockerEnvironment(dockerDir, withGraphql, withFrontend); err != nil {
		return err
	}

	if err := c.ComposeBuild(dockerDir); err != nil {
		return err
	}

	if err := c.ComposeRun(dockerDir); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := c.WaitForLogs(ctx, container.ContainerSuiPlayground, container.ContainerLogReadyCtx); err != nil {
		return err
	}

	if withFrontend {
		if err := startFrontend(c, dockerDir); err != nil {
			return err
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

func startFrontend(c container.ContainerClient, dockerDir string) error {
	ui.Info.Println("Starting frontend dApp...")
	if err := c.ComposeUp(dockerDir, "frontend"); err != nil {
		return err
	}

	// Give the container a moment to start (or crash)
	time.Sleep(3 * time.Second)

	// Check both possible container names (docker-frontend-1 / docker_frontend_1)
	feRunning := c.ContainerRunning(container.ContainerFrontend) || c.ContainerRunning(container.ContainerFrontendOld)
	if !feRunning {
		logsOut := c.ContainerLogs(container.ContainerFrontend, 30)
		if logsOut == "" || strings.Contains(logsOut, "could not retrieve") {
			logsOut = c.ContainerLogs(container.ContainerFrontendOld, 30)
		}
		ui.Warn.Println("Frontend container exited immediately. Logs:")
		fmt.Println(logsOut)
		return fmt.Errorf("frontend container is not running — check the logs above for details")
	}
	return nil
}
