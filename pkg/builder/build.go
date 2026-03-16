package builder

import (
	"context"
	"fmt"
	"strings"

	"efctl/pkg/container"
	"efctl/pkg/ui"
)

// BuildExtension compiles the Move contract inside the container without publishing.
func BuildExtension(c container.ContainerClient, workspace string, network string, candidate PublishCandidate) error {
	if err := PrepareExtensionEnv(c, workspace, network); err != nil {
		return err
	}

	ui.Info.Printf("Building extension contract from %s...\n", candidate.HostPath)
	ui.Info.Printf("Executing build inside container at %s...\n", candidate.ContainerPath)

	buildCmd := fmt.Sprintf("cd %s && sui move build --build-env testnet", candidate.ContainerPath)

	ui.Warn.Println("Build logging will be piped below:")

	output, err := c.ExecCapture(context.Background(), container.ContainerSuiPlayground, []string{"/bin/bash", "-c", buildCmd})
	if output != "" {
		fmt.Print(output)
	}
	if err != nil {
		if strings.Contains(output, "Build error") || strings.Contains(output, "Compilation error") {
			return fmt.Errorf("build failed due to compilation errors")
		}
		return fmt.Errorf("build command failed: %w", err)
	}

	ui.Success.Println("Extension contract built successfully.")
	return nil
}
