package env

import (
	"fmt"
	"net"
	"os"
	"os/exec"
)

// CheckResult holds the status of prerequisite checks
type CheckResult struct {
	HasGit    bool
	HasDocker bool
	HasPodman bool
}

// Engine returns the preferred container engine (docker or podman). Returns an error if neither is available.
func (c *CheckResult) Engine() (string, error) {
	// First check if a preference is set via environment variable
	if pref := os.Getenv("EFCTL_ENGINE"); pref != "" {
		if pref == "podman" && c.HasPodman {
			return "podman", nil
		}
		if pref == "docker" && c.HasDocker {
			return "docker", nil
		}
	}

	// Default precedence: Podman, then Docker
	if c.HasPodman {
		return "podman", nil
	}
	if c.HasDocker {
		return "docker", nil
	}
	return "", fmt.Errorf("no container engine found")
}

// CheckPrerequisites verifies if required tools are installed
func CheckPrerequisites() *CheckResult {
	res := &CheckResult{}

	if _, err := exec.LookPath("git"); err == nil {
		res.HasGit = true
	}
	if _, err := exec.LookPath("docker"); err == nil {
		res.HasDocker = true
	}
	if _, err := exec.LookPath("podman"); err == nil {
		res.HasPodman = true
	}

	return res
}

// IsPortAvailable checks if a TCP port is available on the local machine
func IsPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}
