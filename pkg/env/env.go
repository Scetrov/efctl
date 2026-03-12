package env

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"efctl/pkg/config"
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
	// 0. Check if a preference is set in efctl.yaml
	pref := config.Loaded.GetContainerEngine()
	if pref != "" && pref != "auto-detect" {
		if pref == "podman" && c.HasPodman {
			return "podman", nil
		}
		if pref == "docker" && c.HasDocker {
			return "docker", nil
		}
	}

	// 1. First check if a preference is set via environment variable
	if envPref := os.Getenv("EFCTL_ENGINE"); envPref != "" {
		if envPref == "podman" && c.HasPodman {
			return "podman", nil
		}
		if envPref == "docker" && c.HasDocker {
			return "docker", nil
		}
	}

	// Default precedence: Podman (if it's aliased as docker or native), then Docker.
	// This ensures keep-id and other Podman-specific logic is applied when possible.
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
	if dockerPath, err := exec.LookPath("docker"); err == nil {
		res.HasDocker = true
		// Check if docker is an alias/symlink to podman
		if isPodmanAlias(dockerPath) {
			res.HasPodman = true
		} else if out, err := exec.Command(dockerPath, "--version").Output(); err == nil { // #nosec G204
			if strings.Contains(strings.ToLower(string(out)), "podman") {
				res.HasPodman = true
			}
		}
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

func isPodmanAlias(path string) bool {
	evalPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(evalPath), "podman")
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
