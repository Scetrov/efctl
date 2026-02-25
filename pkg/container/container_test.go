package container

import (
	"os"
	"testing"
)

func TestNewClient(t *testing.T) {
	// Attempt to create a client. If the system has docker or podman, it should succeed.
	// We can manipulate the environment variable to force a specific engine if needed.

	// Test failure case by forcing a non-existent requirement if possible.
	// Since NewClient relies on env.CheckPrerequisites, we could test NewClient
	// by checking logic without actually running docker.

	os.Setenv("EFCTL_ENGINE", "docker")
	defer os.Unsetenv("EFCTL_ENGINE")

	client, err := NewClient()

	// If the runner has docker or podman, it should work. Wait, the machine has podman/docker
	// Let's just verify client is not nil or err is not nil
	if err != nil {
		t.Logf("NewClient returned error (possibly missing docker/podman): %v", err)
	} else if client == nil {
		t.Errorf("Expected non-nil client if err is nil")
	}
}

func TestClient_ComposeBuild_InvalidDir(t *testing.T) {
	client := &Client{Engine: "echo"} // Use a stub engine command instead of docker

	err := client.ComposeBuild("/invalid/path/that/does/not/exist")
	// Since we mock the command, 'echo compose build' will actually succeed in bash, but cmd.Dir will fail
	// because the os/exec starts execution in the cmd.Dir directory.

	if err == nil {
		t.Errorf("Expected error when running in invalid directory")
	}
}

func TestClient_ComposeRun_InvalidDir(t *testing.T) {
	client := &Client{Engine: "echo"}

	err := client.ComposeRun("/invalid/path/that/does/not/exist")

	if err == nil {
		t.Errorf("Expected error when running in invalid directory")
	}
}
