package env

import (
	"os"
	"testing"
)

func TestEngineDetection(t *testing.T) {
	// Test Podman preference
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

func TestEnginePreference(t *testing.T) {
	os.Setenv("EFCTL_ENGINE", "docker")
	defer os.Unsetenv("EFCTL_ENGINE")

	res := &CheckResult{
		HasDocker: true,
		HasPodman: true,
	}

	engine, err := res.Engine()
	if err != nil {
		t.Fatalf("Engine() failed: %v", err)
	}
	if engine != "docker" {
		t.Errorf("Expected engine docker due to preference, got %s", engine)
	}

	// Preference for podman
	os.Setenv("EFCTL_ENGINE", "podman")
	enginePod, _ := res.Engine()
	if enginePod != "podman" {
		t.Errorf("Expected engine podman due to preference, got %s", enginePod)
	}
}

func TestEngineError(t *testing.T) {
	res := &CheckResult{
		HasDocker: false,
		HasPodman: false,
	}
	_, err := res.Engine()
	if err == nil {
		t.Error("Expected error when no engine found, got nil")
	}
}

func TestIsPortAvailable(t *testing.T) {
	// Port 0 should usually be available for listening (os picks one)
	// but we check a high port that is likely free.
	if !IsPortAvailable(65000) {
		t.Log("Port 65000 busy, skipping test")
	}
}

func TestCheckPrerequisites(t *testing.T) {
	res := CheckPrerequisites()
	if res == nil {
		t.Fatal("CheckPrerequisites() returned nil")
	}
	// We can't easily assert presence of tools in all environments,
	// but we can ensure it doesn't crash and returns a result.
}
