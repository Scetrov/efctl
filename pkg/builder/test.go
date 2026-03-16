package builder

import (
	"context"
	"fmt"
	"strings"

	"efctl/pkg/container"
	"efctl/pkg/ui"
)

// TestExtension runs sui move test for the Move contract inside the container.
func TestExtension(c container.ContainerClient, workspace string, network string, candidate PublishCandidate) error {
	if err := PrepareExtensionEnv(c, workspace, network); err != nil {
		return err
	}

	ui.Info.Printf("Testing extension contract from %s...\n", candidate.HostPath)
	ui.Info.Printf("Executing test inside container at %s...\n", candidate.ContainerPath)

	testCmd := fmt.Sprintf("cd %s && sui move test --build-env testnet", candidate.ContainerPath)

	ui.Warn.Println("Test logging will be piped below:")

	output, err := c.ExecCapture(context.Background(), container.ContainerSuiPlayground, []string{"/bin/bash", "-c", testCmd})
	if output != "" {
		fmt.Print(output)
	}
	if err != nil {
		if strings.Contains(output, "Test failures") {
			return fmt.Errorf("tests failed")
		}
		return fmt.Errorf("test command failed: %w", err)
	}

	ui.Success.Println("Extension contract tests passed.")
	return nil
}
