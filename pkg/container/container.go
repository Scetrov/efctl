package container

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"efctl/pkg/env"
	"efctl/pkg/ui"
)

// Client wraps the container engine execution
type Client struct {
	Engine string
}

// NewClient returns a new container client
func NewClient() (*Client, error) {
	res := env.CheckPrerequisites()
	engine, err := res.Engine()
	if err != nil {
		return nil, err
	}
	return &Client{Engine: engine}, nil
}

// ComposeBuild runs docker/podman compose build
func (c *Client) ComposeBuild(dir string) error {
	spinner, _ := ui.Spin("Building containers...")

	cmd := exec.Command(c.Engine, "compose", "build") // #nosec G204
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		spinner.Fail("Failed to build containers")
		return fmt.Errorf("compose build error: %v\n%s", err, string(output))
	}

	spinner.Success("Containers built successfully")
	return nil
}

// ComposeRun runs the default sui-playground
func (c *Client) ComposeRun(dir string) error {
	spinner, _ := ui.Spin("Starting sui-playground...")

	cmd := exec.Command(c.Engine, "compose", "run", "-d", "--name", ContainerSuiPlayground, "--service-ports", "sui-dev") // #nosec G204
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		spinner.Fail("Failed to start sui-playground")
		return fmt.Errorf("compose run error: %v\n%s", err, string(output))
	}

	spinner.Success("sui-playground started in detached mode")
	return nil
}

// WaitForLogs waits for a specific string in the container logs
func (c *Client) WaitForLogs(ctx context.Context, containerName string, searchString string) error {
	spinner, _ := ui.Spin(fmt.Sprintf("Waiting for %s to initialize...", containerName))

	cmd := exec.CommandContext(ctx, c.Engine, "logs", "-f", containerName) // #nosec G204
	// We need both stdout and stderr since logs can go to either
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		spinner.Fail("Failed to get logs pipe")
		return err
	}

	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		spinner.Fail("Failed to start logs command")
		return err
	}

	// Channel to signal when search string is found
	done := make(chan bool)

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, searchString) {
				done <- true
				return
			}
		}
		done <- false
	}()

	select {
	case <-ctx.Done():
		spinner.Fail("Timed out waiting for logs")
		return fmt.Errorf("timeout waiting for %q in %s logs", searchString, containerName)
	case found := <-done:
		if !found {
			spinner.Fail("Log search string not found before EOF")
			return fmt.Errorf("search string %q not found in logs", searchString)
		}
	}

	_ = cmd.Wait() // Reclaim process

	spinner.Success(fmt.Sprintf("%s is ready", containerName))
	return nil
}

// Exec runs a command inside a container
func (c *Client) Exec(containerName string, command []string) error {
	spinner, _ := ui.Spin(fmt.Sprintf("Executing in %s...", containerName))

	// We do not use -it because we don't have a tty
	args := append([]string{"exec", containerName}, command...)
	cmd := exec.Command(c.Engine, args...) // #nosec G204

	output, err := cmd.CombinedOutput()
	if err != nil {
		spinner.Fail("Execution failed")
		return fmt.Errorf("exec error: %w\n%s", err, string(output))
	}

	spinner.Success("Execution complete")
	return nil
}

// Cleanup stops/removes the container, removes images, volumes
func (c *Client) Cleanup() error {
	spinner, _ := ui.Spin("Stopping and removing sui-playground container...")
	if err := exec.Command(c.Engine, "stop", ContainerSuiPlayground).Run(); err != nil { // #nosec G204
		ui.Warn.Println(fmt.Sprintf("\nFailed to stop %s (might not be running): %v", ContainerSuiPlayground, err))
	}
	if err := exec.Command(c.Engine, "rm", ContainerSuiPlayground).Run(); err != nil { // #nosec G204
		ui.Warn.Println(fmt.Sprintf("\nFailed to remove %s: %v", ContainerSuiPlayground, err))
	}
	spinner.Success(fmt.Sprintf("Container %s removal attempted", ContainerSuiPlayground))

	spinner2, _ := ui.Spin("Removing sui-dev images...")
	if err := exec.Command(c.Engine, "rmi", ImageDockerSuiDev).Run(); err != nil { // #nosec G204
		ui.Warn.Println(fmt.Sprintf("\nFailed to remove %s image: %v", ImageDockerSuiDev, err))
	}
	if err := exec.Command(c.Engine, "rmi", ImageDockerSuiDevOld).Run(); err != nil { // #nosec G204
		ui.Warn.Println(fmt.Sprintf("\nFailed to remove %s image: %v", ImageDockerSuiDevOld, err))
	}
	spinner2.Success("Images removal attempted")

	spinner3, _ := ui.Spin("Removing config volume...")
	if err := exec.Command(c.Engine, "volume", "rm", VolumeDockerSuiConfig).Run(); err != nil { // #nosec G204
		ui.Warn.Println(fmt.Sprintf("\nFailed to remove %s volume: %v", VolumeDockerSuiConfig, err))
	}
	spinner3.Success("Volume removal attempted")

	return nil
}
