package setup

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"efctl/pkg/container"
	"efctl/pkg/env"
	"efctl/pkg/ui"
)

// StartEnvironment builds and starts the docker-compose environment
func StartEnvironment(workspace string, withGraphql bool) error {
	ui.Info.Println("Starting container environment...")

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

	c, err := container.NewClient()
	if err != nil {
		return err
	}

	dockerDir := filepath.Join(workspace, "builder-scaffold", "docker")

	if err := prepareDockerEnvironment(dockerDir, withGraphql); err != nil {
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

	return nil
}
