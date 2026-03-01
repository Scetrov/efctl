package env

import (
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEngineSelectionWithEnvVar(t *testing.T) {
	tests := []struct {
		name        string
		envVar      string
		hasDocker   bool
		hasPodman   bool
		expected    string
		expectError bool
	}{
		{
			name:        "Prefers podman when env says podman and both installed",
			envVar:      "podman",
			hasDocker:   true,
			hasPodman:   true,
			expected:    "podman",
			expectError: false,
		},
		{
			name:        "Prefers docker when env says docker and both installed",
			envVar:      "docker",
			hasDocker:   true,
			hasPodman:   true,
			expected:    "docker",
			expectError: false,
		},
		{
			name:        "Falls back to default (docker) if env says podman but podman isn't installed",
			envVar:      "podman",
			hasDocker:   true,
			hasPodman:   false,
			expected:    "docker", // Env pref ignored, fallback to docker
			expectError: false,
		},
		{
			name:        "Default precedence prefers docker if both installed and no env var set",
			envVar:      "",
			hasDocker:   true,
			hasPodman:   true,
			expected:    "docker",
			expectError: false,
		},
		{
			name:        "Returns docker if only docker installed",
			envVar:      "",
			hasDocker:   true,
			hasPodman:   false,
			expected:    "docker",
			expectError: false,
		},
		{
			name:        "Returns error if neither installed",
			envVar:      "",
			hasDocker:   false,
			hasPodman:   false,
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("EFCTL_ENGINE", tt.envVar)
			defer os.Unsetenv("EFCTL_ENGINE")

			res := &CheckResult{
				HasDocker: tt.hasDocker,
				HasPodman: tt.hasPodman,
			}

			engine, err := res.Engine()

			if tt.expectError && err == nil {
				t.Errorf("Expected an error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Did not expect an error, got: %v", err)
			}
			if engine != tt.expected {
				t.Errorf("Expected engine %s, got %s", tt.expected, engine)
			}
		})
	}
}

// ── IsPortAvailable ────────────────────────────────────────────────

func TestIsPortAvailable_FreePort(t *testing.T) {
	// Port 0 picks a random free port; close it immediately to test availability
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Skip("cannot allocate a random port")
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	assert.True(t, IsPortAvailable(port), "port should be available after closing listener")
}

func TestIsPortAvailable_OccupiedPort(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Skip("cannot allocate a random port")
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	assert.False(t, IsPortAvailable(port), "port should not be available while listener is active")
}

// ── CheckPrerequisites ─────────────────────────────────────────────

func TestCheckPrerequisites_Runs(t *testing.T) {
	// Basic smoke test: should return a non-nil result without panicking
	res := CheckPrerequisites()
	assert.NotNil(t, res)
	// At a minimum, git should be available in most CI environments
	// (but we don't assert it — the function should just not panic)
}
