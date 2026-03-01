package env

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
)

// CheckResult holds the status of prerequisite checks
type CheckResult struct {
	HasGit    bool
	HasDocker bool
	HasPodman bool
	HasNode   bool
	NodeVer   string
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

	// Default precedence: Docker, then Podman.
	// This aligns with CI and common local setups where docker compose behavior is expected.
	if c.HasDocker {
		return "docker", nil
	}
	if c.HasPodman {
		return "podman", nil
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
	if out, err := exec.Command("node", "--version").Output(); err == nil {
		res.HasNode = true
		res.NodeVer = strings.TrimSpace(string(out))
	}

	return res
}

// IsPortAvailable checks if a TCP port is available on the local machine
func IsPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	if err := ln.Close(); err != nil {
		// Non-critical: listener close failure during availability check
		_ = err
	}
	return true
}
