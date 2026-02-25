package setup

import (
	"efctl/pkg/container"
	"efctl/pkg/ui"
)

// CleanEnvironment stops containers, removes them, cleans up images, and volumes
func CleanEnvironment() error {
	ui.Info.Println("Cleaning up environment...")

	c, err := container.NewClient()
	if err != nil {
		return err
	}

	if err := c.Cleanup(); err != nil {
		return err
	}

	return nil
}
