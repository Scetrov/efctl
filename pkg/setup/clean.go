package setup

import (
	"efctl/pkg/container"
	"efctl/pkg/ui"
)

// CleanEnvironment stops containers, removes them, cleans up images, and volumes
func CleanEnvironment(c container.ContainerClient) error {
	ui.Info.Println("Cleaning up environment...")

	if err := c.Cleanup(); err != nil {
		return err
	}

	return nil
}
