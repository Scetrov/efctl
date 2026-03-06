package env

import (
	"testing"
)

func TestEngineDetection(t *testing.T) {
	// Test Podman detected via alias
	res := &CheckResult{
		HasDocker: true,
		HasPodman: true,
	}

	engine, err := res.Engine()
	if err != nil {
		t.Fatalf("Engine() failed: %v", err)
	}
	if engine != "podman" {
		t.Errorf("Expected engine podman, got %s", engine)
	}

	// Test pure Docker
	resPure := &CheckResult{
		HasDocker: true,
		HasPodman: false,
	}
	enginePure, _ := resPure.Engine()
	if enginePure != "docker" {
		t.Errorf("Expected engine docker, got %s", enginePure)
	}

	// Test pure Podman
	resPodman := &CheckResult{
		HasDocker: false,
		HasPodman: true,
	}
	enginePodman, _ := resPodman.Engine()
	if enginePodman != "podman" {
		t.Errorf("Expected engine podman, got %s", enginePodman)
	}
}
