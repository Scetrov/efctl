package container

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"efctl/pkg/env"
	"efctl/pkg/ui"
)

// ContainerClient defines the interface for container operations.
// All consumers should accept this interface to enable testing with mocks.
type ContainerClient interface {
	ComposeBuild(dir string) error
	ComposeRun(dir string) error
	ComposeUp(dir string, services ...string) error
	ContainerRunning(name string) bool
	ContainerLogs(name string, tail int) string
	ContainerExitCode(name string) (int, error)
	WaitForLogs(ctx context.Context, containerName string, searchString string) error
	InteractiveShell(containerName string) error
	Exec(containerName string, command []string) error
	ExecCapture(containerName string, command []string) (string, error)
	Cleanup() error
}

// Client wraps the container engine execution and implements ContainerClient.
type Client struct {
	Engine string
}

// Compile-time check that Client implements ContainerClient.
var _ ContainerClient = (*Client)(nil)

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

// ComposeUp starts one or more compose services in detached mode
func (c *Client) ComposeUp(dir string, services ...string) error {
	args := []string{"compose", "up", "-d"}
	args = append(args, services...)

	label := strings.Join(services, ", ")
	spinner, _ := ui.Spin(fmt.Sprintf("Starting %s...", label))

	cmd := exec.Command(c.Engine, args...) // #nosec G204
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to start %s", label))
		return fmt.Errorf("compose up error: %v\n%s", err, string(output))
	}

	spinner.Success(fmt.Sprintf("%s started", label))
	return nil
}

// ContainerRunning checks if a container is currently running.
func (c *Client) ContainerRunning(name string) bool {
	out, err := exec.Command(c.Engine, "inspect", "--format", "{{.State.Running}}", name).Output() // #nosec G204
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

// ContainerLogs returns the last N lines of a container's logs.
func (c *Client) ContainerLogs(name string, tail int) string {
	out, err := exec.Command(c.Engine, "logs", "--tail", fmt.Sprintf("%d", tail), name).CombinedOutput() // #nosec G204
	if err != nil {
		return fmt.Sprintf("(could not retrieve logs: %v)", err)
	}
	return strings.TrimSpace(string(out))
}

// ContainerExitCode returns the exit code of a stopped container.
func (c *Client) ContainerExitCode(name string) (int, error) {
	out, err := exec.Command(c.Engine, "inspect", "--format", "{{.State.ExitCode}}", name).Output() // #nosec G204
	if err != nil {
		return -1, fmt.Errorf("failed to inspect container %s: %w", name, err)
	}
	var code int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &code); err != nil {
		return -1, fmt.Errorf("failed to parse exit code for %s: %w", name, err)
	}
	return code, nil
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
		lastLogs := c.ContainerLogs(containerName, 20)
		return fmt.Errorf("timeout waiting for %q in %s logs.\n\nLast 20 lines of container logs:\n%s", searchString, containerName, lastLogs)
	case found := <-done:
		if !found {
			// Container exited before the ready string appeared — provide diagnostics
			exitCode, exitErr := c.ContainerExitCode(containerName)
			lastLogs := c.ContainerLogs(containerName, 30)
			diag := fmt.Sprintf("Container %s exited before becoming ready.", containerName)
			if exitErr == nil {
				diag += fmt.Sprintf(" Exit code: %d.", exitCode)
			}
			diag += fmt.Sprintf("\n\nLast 30 lines of container logs:\n%s", lastLogs)
			spinner.Fail(fmt.Sprintf("%s exited unexpectedly (search string %q not found)", containerName, searchString))
			return fmt.Errorf("%s", diag)
		}
	}

	if err := cmd.Wait(); err != nil {
		// Non-critical: log follow process exit after search string found
		_ = err
	}

	spinner.Success(fmt.Sprintf("%s is ready", containerName))
	return nil
}

// InteractiveShell opens an interactive shell in the container
func (c *Client) InteractiveShell(containerName string) error {
	cmd := exec.Command(c.Engine, "exec", "-it", containerName, "/bin/bash") // #nosec G204
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("interactive shell error: %w", err)
	}

	return nil
}

// Exec runs a command inside a container
func (c *Client) Exec(containerName string, command []string) error {
	spinner, _ := ui.Spin(fmt.Sprintf("Executing in %s...", containerName))

	// We do not use -it because we don't have a tty
	args := make([]string, 0, 2+len(command))
	args = append(args, "exec", containerName)
	args = append(args, command...)
	cmd := exec.Command(c.Engine, args...) // #nosec G204

	output, err := cmd.CombinedOutput()

	// Print output if any, regardless of success/fail
	if len(output) > 0 {
		fmt.Printf("\n%s", string(output))
	}

	if err != nil {
		spinner.Fail("Execution failed")
		return fmt.Errorf("exec error: %w\n%s", err, string(output))
	}

	spinner.Success("Execution complete")
	return nil
}

// ExecCapture runs a command inside a container and returns the combined output.
func (c *Client) ExecCapture(containerName string, command []string) (string, error) {
	args := make([]string, 0, 2+len(command))
	args = append(args, "exec", containerName)
	args = append(args, command...)
	cmd := exec.Command(c.Engine, args...) // #nosec G204

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("exec error: %w\n%s", err, string(output))
	}

	return string(output), nil
}

// Cleanup stops/removes the container, removes images, volumes
func (c *Client) Cleanup() error {
	spinner, _ := ui.Spin("Stopping and removing sui-playground container...")
	c.stopAndRemoveContainers([]string{ContainerSuiPlayground})
	spinner.Success(fmt.Sprintf("Container %s removal attempted", ContainerSuiPlayground))

	spinnerPg, _ := ui.Spin("Stopping and removing postgres container...")
	c.stopAndRemoveContainers([]string{ContainerPostgres, ContainerPostgresOld})
	spinnerPg.Success("Postgres container removal attempted")

	spinnerFe, _ := ui.Spin("Stopping and removing frontend container...")
	c.stopAndRemoveContainers([]string{ContainerFrontend, ContainerFrontendOld})
	spinnerFe.Success("Frontend container removal attempted")

	spinner2, _ := ui.Spin("Removing sui-dev images...")
	c.removeImages([]string{ImageDockerSuiDev, ImageDockerSuiDevOld})
	spinner2.Success("Images removal attempted")

	spinner3, _ := ui.Spin("Removing config and data volumes...")
	c.removeVolumes([]string{VolumeDockerSuiConfig, VolumeDockerSuiConfigOld, VolumeDockerPgData, VolumeDockerPgDataOld, VolumeDockerFeModules, VolumeDockerFeModulesOld})
	spinner3.Success("Volumes removal attempted")

	return nil
}

func (c *Client) stopAndRemoveContainers(names []string) {
	for _, name := range names {
		if c.containerExists(name) {
			if err := exec.Command(c.Engine, "stop", name).Run(); err != nil { // #nosec G204
				ui.Warn.Println(fmt.Sprintf("Failed to stop %s: %v", name, err))
			}
			if err := exec.Command(c.Engine, "rm", name).Run(); err != nil { // #nosec G204
				ui.Warn.Println(fmt.Sprintf("Failed to remove %s: %v", name, err))
			}
		}
	}
}

func (c *Client) removeImages(names []string) {
	for _, name := range names {
		if c.imageExists(name) {
			if err := exec.Command(c.Engine, "rmi", name).Run(); err != nil { // #nosec G204
				ui.Warn.Println(fmt.Sprintf("Failed to remove %s image: %v", name, err))
			}
		}
	}
}

func (c *Client) removeVolumes(names []string) {
	for _, vol := range names {
		if c.volumeExists(vol) {
			if err := exec.Command(c.Engine, "volume", "rm", vol).Run(); err != nil { // #nosec G204
				ui.Warn.Println(fmt.Sprintf("Failed to remove %s volume: %v", vol, err))
			}
		}
	}
}

func (c *Client) containerExists(name string) bool {
	return exec.Command(c.Engine, "container", "inspect", name).Run() == nil // #nosec G204
}

func (c *Client) imageExists(name string) bool {
	return exec.Command(c.Engine, "image", "inspect", name).Run() == nil // #nosec G204
}

func (c *Client) volumeExists(name string) bool {
	return exec.Command(c.Engine, "volume", "inspect", name).Run() == nil // #nosec G204
}
